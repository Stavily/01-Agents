// Package agent implements the core Stavily Action Agent functionality
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/stavily/agents/shared/pkg/api"
	"github.com/stavily/agents/shared/pkg/config"
	"github.com/stavily/agents/shared/pkg/plugin"
)

// ActionAgent represents the main action agent instance
type ActionAgent struct {
	cfg         *config.Config
	logger      *zap.Logger
	apiClient   *api.Client
	pluginMgr   plugin.PluginManager
	executor    *ActionExecutor
	poller      *TaskPoller
	metrics     *MetricsCollector
	healthCheck *HealthMonitor

	// Runtime state
	mu        sync.RWMutex
	running   bool
	startTime time.Time

	// Channels for coordination
	stopChan chan struct{}
	doneChan chan struct{}
}

// NewActionAgent creates a new action agent instance
func NewActionAgent(cfg *config.Config, logger *zap.Logger) (*ActionAgent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Create API client for orchestrator communication
	apiClient, err := api.NewClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Create plugin manager
	pluginMgr, err := NewPluginManager(&cfg.Plugins, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// Create action executor
	executor, err := NewActionExecutor(cfg, pluginMgr, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create action executor: %w", err)
	}

	// Create task poller
	poller, err := NewTaskPoller(cfg, apiClient, executor, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create task poller: %w", err)
	}

	// Create metrics collector
	metrics, err := NewMetricsCollector(&cfg.Metrics, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics collector: %w", err)
	}

	// Create health monitor
	healthCheck, err := NewHealthMonitor(&cfg.Health, pluginMgr.(*PluginManager), logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create health checker: %w", err)
	}

	return &ActionAgent{
		cfg:         cfg,
		logger:      logger,
		apiClient:   apiClient,
		pluginMgr:   pluginMgr,
		executor:    executor,
		poller:      poller,
		metrics:     metrics,
		healthCheck: healthCheck,
		stopChan:    make(chan struct{}),
		doneChan:    make(chan struct{}),
	}, nil
}

// Start starts the action agent
func (a *ActionAgent) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("action agent is already running")
	}

	a.logger.Info("Starting action agent",
		zap.String("agent_id", a.cfg.Agent.ID),
		zap.String("tenant_id", a.cfg.Agent.TenantID))

	// Start metrics collector
	if err := a.metrics.Start(ctx); err != nil {
		return fmt.Errorf("failed to start metrics collector: %w", err)
	}

	// Start health checker
	if err := a.healthCheck.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health checker: %w", err)
	}

	// Plugin manager is ready to use (no explicit initialization needed)

	// Start action executor
	if err := a.executor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start action executor: %w", err)
	}

	// Start task poller
	if err := a.poller.Start(ctx); err != nil {
		return fmt.Errorf("failed to start task poller: %w", err)
	}

	a.running = true
	a.startTime = time.Now()

	// Start the main run loop
	go a.run(ctx)

	a.logger.Info("Action agent started successfully")
	return nil
}

// Stop stops the action agent gracefully
func (a *ActionAgent) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.logger.Info("Stopping action agent")

	// Signal shutdown
	close(a.stopChan)

	// Wait for main loop to finish or timeout
	select {
	case <-a.doneChan:
		a.logger.Info("Action agent main loop stopped")
	case <-ctx.Done():
		a.logger.Warn("Action agent shutdown timed out")
		return ctx.Err()
	}

	// Stop components in reverse order
	if err := a.poller.Stop(ctx); err != nil {
		a.logger.Error("Error stopping task poller", zap.Error(err))
	}

	if err := a.executor.Stop(ctx); err != nil {
		a.logger.Error("Error stopping action executor", zap.Error(err))
	}

	// Plugin manager cleanup is handled automatically

	if err := a.healthCheck.Stop(ctx); err != nil {
		a.logger.Error("Error stopping health checker", zap.Error(err))
	}

	if err := a.metrics.Stop(ctx); err != nil {
		a.logger.Error("Error stopping metrics collector", zap.Error(err))
	}

	a.running = false
	a.logger.Info("Action agent stopped successfully")
	return nil
}

// IsRunning returns whether the agent is currently running
func (a *ActionAgent) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// GetStatus returns the current agent status
func (a *ActionAgent) GetStatus() *AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := &AgentStatus{
		AgentID:     a.cfg.Agent.ID,
		TenantID:    a.cfg.Agent.TenantID,
		Type:        "action",
		Version:     "dev", // TODO: Get from build info
		Running:     a.running,
		StartTime:   a.startTime,
		Uptime:      time.Since(a.startTime),
		Environment: a.cfg.Agent.Environment,
	}

	if a.running {
		status.PluginStatus = a.pluginMgr.(*PluginManager).GetPluginStatuses()
		status.ExecutorStatus = a.executor.GetStatus()
		status.PollerStatus = a.poller.GetStatus()
		status.HealthStatus = a.healthCheck.GetStatus()
		status.MetricsStatus = a.metrics.GetStatus()
	}

	return status
}

// GetHealth returns the agent health information
func (a *ActionAgent) GetHealth() *AgentHealth {
	a.mu.RLock()
	defer a.mu.RUnlock()

	health := &AgentHealth{
		AgentID:    a.cfg.Agent.ID,
		Status:     HealthStatusHealthy,
		Timestamp:  time.Now(),
		Uptime:     time.Since(a.startTime),
		Components: make(map[string]*ComponentHealth),
	}

	if !a.running {
		health.Status = HealthStatusUnhealthy
		health.Message = "Agent is not running"
		return health
	}

	// Check component health
	components := map[string]HealthChecker{
		"plugin_manager": a.pluginMgr.(*PluginManager),
		"executor":       a.executor,
		"poller":         a.poller,
		"metrics":        a.metrics,
		"health_check":   a.healthCheck,
	}

	overallHealthy := true
	for name, component := range components {
		componentHealth := component.GetHealth()
		health.Components[name] = componentHealth

		if componentHealth.Status != HealthStatusHealthy {
			overallHealthy = false
		}
	}

	if !overallHealthy {
		health.Status = HealthStatusDegraded
		health.Message = "One or more components are unhealthy"
	}

	return health
}

// run is the main agent loop
func (a *ActionAgent) run(ctx context.Context) {
	defer close(a.doneChan)

	ticker := time.NewTicker(30 * time.Second) // Status reporting interval
	defer ticker.Stop()

	a.logger.Info("Action agent main loop started")

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Action agent context cancelled")
			return
		case <-a.stopChan:
			a.logger.Info("Action agent stop signal received")
			return
		case <-ticker.C:
			// Periodic status reporting and health checks
			a.reportStatus(ctx)
		}
	}
}

// reportStatus reports agent status to the orchestrator
func (a *ActionAgent) reportStatus(ctx context.Context) {
	status := a.GetStatus()

	if err := a.apiClient.ReportAgentStatus(ctx, status); err != nil {
		a.logger.Error("Failed to report agent status", zap.Error(err))
		return
	}

	a.logger.Debug("Agent status reported successfully")
}

// AgentStatus represents the current status of the action agent
type AgentStatus struct {
	AgentID        string                   `json:"agent_id"`
	TenantID       string                   `json:"tenant_id"`
	Type           string                   `json:"type"`
	Version        string                   `json:"version"`
	Running        bool                     `json:"running"`
	StartTime      time.Time                `json:"start_time"`
	Uptime         time.Duration            `json:"uptime"`
	Environment    string                   `json:"environment"`
	PluginStatus   map[string]*PluginStatus `json:"plugin_status,omitempty"`
	ExecutorStatus *ExecutorStatus          `json:"executor_status,omitempty"`
	PollerStatus   *PollerStatus            `json:"poller_status,omitempty"`
	HealthStatus   *HealthCheckStatus       `json:"health_status,omitempty"`
	MetricsStatus  *MetricsStatus           `json:"metrics_status,omitempty"`
}

// AgentHealth represents the health information of the action agent
type AgentHealth struct {
	AgentID    string                      `json:"agent_id"`
	Status     HealthStatus                `json:"status"`
	Message    string                      `json:"message,omitempty"`
	Timestamp  time.Time                   `json:"timestamp"`
	Uptime     time.Duration               `json:"uptime"`
	Components map[string]*ComponentHealth `json:"components"`
}

// HealthStatus represents the health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentHealth represents the health of a component
type ComponentHealth struct {
	Status     HealthStatus `json:"status"`
	Message    string       `json:"message,omitempty"`
	LastCheck  time.Time    `json:"last_check"`
	ErrorCount int          `json:"error_count"`
}

// HealthChecker interface for components that can report health
type HealthChecker interface {
	GetHealth() *ComponentHealth
}

// PluginStatus represents the status of plugins
type PluginStatus struct {
	Loaded  int `json:"loaded"`
	Running int `json:"running"`
	Errors  int `json:"errors"`
}

// ExecutorStatus represents the status of the action executor
type ExecutorStatus struct {
	ActiveTasks    int `json:"active_tasks"`
	QueuedTasks    int `json:"queued_tasks"`
	CompletedTasks int `json:"completed_tasks"`
	FailedTasks    int `json:"failed_tasks"`
}

// PollerStatus represents the status of the task poller
type PollerStatus struct {
	LastPoll      time.Time     `json:"last_poll"`
	PollInterval  time.Duration `json:"poll_interval"`
	TasksReceived int           `json:"tasks_received"`
	PollErrors    int           `json:"poll_errors"`
}

// HealthCheckStatus represents the status of the health checker
type HealthCheckStatus struct {
	LastCheck     time.Time     `json:"last_check"`
	CheckInterval time.Duration `json:"check_interval"`
	ChecksPassed  int           `json:"checks_passed"`
	ChecksFailed  int           `json:"checks_failed"`
}

// MetricsStatus represents the status of the metrics collector
type MetricsStatus struct {
	MetricsExported int       `json:"metrics_exported"`
	LastExport      time.Time `json:"last_export"`
	ExportErrors    int       `json:"export_errors"`
}
