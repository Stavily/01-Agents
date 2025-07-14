// Package agent implements the core Stavily Action Agent functionality
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/Stavily/01-Agents/shared/pkg/agent"
	"github.com/Stavily/01-Agents/shared/pkg/api"
	"github.com/Stavily/01-Agents/shared/pkg/config"
	"github.com/Stavily/01-Agents/shared/pkg/types"
)

// ActionAgent represents the main action agent instance
type ActionAgent struct {
	cfg              *config.Config
	logger           *zap.Logger
	orchestratorFlow *agent.OrchestratorWorkflow
	pluginMgr        *PluginManager
	executor         *ActionExecutor
	metrics          *MetricsCollector
	healthCheck      *HealthMonitor
	poller           *TaskPoller

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

	// Create plugin manager
	pluginMgr, err := NewPluginManager(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// Create action executor
	executor, err := NewActionExecutor(cfg, pluginMgr, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create action executor: %w", err)
	}

	// Create metrics collector
	metrics, err := NewMetricsCollector(&cfg.Metrics, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics collector: %w", err)
	}

	// Create health monitor
	healthCheck, err := NewHealthMonitor(&cfg.Health, pluginMgr, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create health checker: %w", err)
	}

	actionAgent := &ActionAgent{
		cfg:         cfg,
		logger:      logger,
		pluginMgr:   pluginMgr,
		executor:    executor,
		metrics:     metrics,
		healthCheck: healthCheck,
		stopChan:    make(chan struct{}),
		doneChan:    make(chan struct{}),
	}

	// Create orchestrator workflow with action-specific plugin executor
	orchestratorFlow, err := agent.NewOrchestratorWorkflow(cfg, logger, actionAgent.executeActionPlugin)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator workflow: %w", err)
	}
	actionAgent.orchestratorFlow = orchestratorFlow

	return actionAgent, nil
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

	// Start orchestrator workflow
	if err := a.orchestratorFlow.Start(ctx); err != nil {
		return fmt.Errorf("failed to start orchestrator workflow: %w", err)
	}

	// Start metrics collector
	if err := a.metrics.Start(ctx); err != nil {
		return fmt.Errorf("failed to start metrics collector: %w", err)
	}

	// Start health checker
	if err := a.healthCheck.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health checker: %w", err)
	}

	// Start action executor
	if err := a.executor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start action executor: %w", err)
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

	// Stop orchestrator workflow
	if err := a.orchestratorFlow.Stop(ctx); err != nil {
		a.logger.Error("Error stopping orchestrator workflow", zap.Error(err))
	}

	// Stop components in reverse order
	if err := a.executor.Stop(ctx); err != nil {
		a.logger.Error("Error stopping action executor", zap.Error(err))
	}

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

// executeActionPlugin executes action-specific plugins for instructions
func (a *ActionAgent) executeActionPlugin(ctx context.Context, instruction *api.Instruction) (map[string]interface{}, error) {
	a.logger.Info("Executing action plugin",
		zap.String("plugin_id", instruction.PluginID),
		zap.Any("input_data", instruction.InputData))

	// Convert api.Instruction to types.Instruction for enhanced plugin manager
	typesInstruction := a.convertAPIInstructionToTypes(instruction)
	
	// Create a poll response with the instruction
	pollResponse := &types.PollResponse{
		Instruction:      typesInstruction,
		Status:           "instruction_delivered",
		NextPollInterval: 5,
	}

	// Process the instruction using the enhanced plugin manager
	result, err := a.pluginMgr.ProcessInstruction(ctx, pollResponse)
	if err != nil {
		a.logger.Error("Failed to process instruction",
			zap.String("instruction_id", instruction.ID),
			zap.Error(err))
		return nil, err
	}

	if result == nil {
		a.logger.Info("No result from instruction processing")
		return map[string]interface{}{
			"status": "no_result",
		}, nil
	}

	// Convert the result back to the expected format
	resultMap := map[string]interface{}{
		"instruction_id": result.InstructionID,
		"success":        result.Success,
		"duration":       result.Duration,
		"start_time":     result.StartTime,
		"end_time":       result.EndTime,
		"logs":           result.ProcessingLogs,
	}

	if result.InstallResult != nil {
		resultMap["install_result"] = map[string]interface{}{
			"plugin_id":      result.InstallResult.PluginID,
			"success":        result.InstallResult.Success,
			"installed_path": result.InstallResult.InstalledPath,
			"version":        result.InstallResult.Version,
			"logs":           result.InstallResult.Logs,
			"duration":       result.InstallResult.Duration,
		}
	}

	if result.ExecutionResult != nil {
		resultMap["execution_result"] = map[string]interface{}{
			"plugin_id":    result.ExecutionResult.PluginID,
			"success":      result.ExecutionResult.Success,
			"output_data":  result.ExecutionResult.OutputData,
			"logs":         result.ExecutionResult.Logs,
			"duration":     result.ExecutionResult.Duration,
			"exit_code":    result.ExecutionResult.ExitCode,
		}
	}

	if result.Error != "" {
		resultMap["error"] = result.Error
	}

	a.logger.Info("Action plugin executed successfully",
		zap.String("instruction_id", instruction.ID),
		zap.Bool("success", result.Success))

	return resultMap, nil
}

// convertAPIInstructionToTypes converts an api.Instruction to types.Instruction
func (a *ActionAgent) convertAPIInstructionToTypes(apiInst *api.Instruction) *types.Instruction {
	// Use the instruction type from the API, with fallback logic
	instructionType := types.InstructionTypeExecute // Default to execute
	
	// First, check if instruction_type is explicitly provided
	if apiInst.InstructionType != "" {
		instructionType = types.InstructionType(apiInst.InstructionType)
	} else {
		// Fallback: determine instruction type based on plugin configuration
		if pluginURL, hasPluginURL := apiInst.PluginConfiguration["plugin_url"]; hasPluginURL && pluginURL != "" {
			instructionType = types.InstructionTypePluginInstall
		} else if repoURL, hasRepoURL := apiInst.PluginConfiguration["repository_url"]; hasRepoURL && repoURL != "" {
			instructionType = types.InstructionTypePluginInstall
		}
	}
	
	return &types.Instruction{
		ID:                  apiInst.ID,
		AgentID:             a.cfg.Agent.ID, // Use the agent's ID
		PluginID:            apiInst.PluginID,
		Status:              types.InstructionStatusPending, // Default status
		Priority:            types.PriorityNormal,           // Default priority
		Type:                instructionType,
		Source:              types.InstructionSourceWebUI,       // Default source
		PluginConfiguration: apiInst.PluginConfiguration,
		InputData:           apiInst.InputData,
		Context:             make(map[string]interface{}), // Empty context
		Variables:           make(map[string]interface{}), // Empty variables
		TimeoutSeconds:      apiInst.TimeoutSeconds,
		MaxRetries:          apiInst.MaxRetries,
		RetryCount:          0,                      // Default retry count
		Metadata:            make(map[string]interface{}), // Empty metadata
	}
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
		// Convert shared types to local types for JSON serialization
		pluginStatuses := a.pluginMgr.GetPluginStatuses()
		localPluginStatus := make(map[string]*PluginStatus)
		for k, v := range pluginStatuses {
			localPluginStatus[k] = &PluginStatus{
				Status:    "running", // Simplified status mapping
				Message:   fmt.Sprintf("Loaded: %d, Running: %d, Errors: %d", v.Loaded, v.Running, v.Errors),
				Timestamp: time.Now(),
			}
		}
		status.PluginStatus = localPluginStatus
		
		status.ExecutorStatus = a.executor.GetStatus()
		
		healthStatus := a.healthCheck.GetStatus()
		status.HealthStatus = &HealthCheckStatus{
			Status:    "healthy", // Simplified status mapping
			Message:   fmt.Sprintf("Checks passed: %d, failed: %d", healthStatus.ChecksPassed, healthStatus.ChecksFailed),
			Timestamp: healthStatus.LastCheck,
		}
		
		metricsStatus := a.metrics.GetStatus()
		status.MetricsStatus = &MetricsStatus{
			Status:    "active", // Simplified status mapping
			Message:   fmt.Sprintf("Exported: %d, errors: %d", metricsStatus.MetricsExported, metricsStatus.ExportErrors),
			Timestamp: metricsStatus.LastExport,
		}
		
		// Add orchestrator workflow status
		if a.orchestratorFlow != nil {
			status.OrchestratorStatus = a.orchestratorFlow.GetStatus()
		}
	}

	return status
}

// GetHealth returns the agent health information
func (a *ActionAgent) GetHealth() *AgentHealth {
	a.mu.RLock()
	defer a.mu.RUnlock()

	health := &AgentHealth{
		AgentID:    a.cfg.Agent.ID,
		Status:     "healthy",
		Timestamp:  time.Now(),
		Uptime:     time.Since(a.startTime),
		Components: make(map[string]*ComponentHealth),
	}

	if !a.running {
		health.Status = "unhealthy"
		health.Message = "Agent is not running"
		return health
	}

	// Check component health - convert shared types to local types
	pluginMgrHealth := a.pluginMgr.GetHealth()
	health.Components["plugin_manager"] = &ComponentHealth{
		Status:    string(pluginMgrHealth.Status),
		Message:   pluginMgrHealth.Message,
		Timestamp: pluginMgrHealth.LastCheck,
	}
	
	executorHealth := a.executor.GetHealth()
	health.Components["executor"] = &ComponentHealth{
		Status:    string(executorHealth.Status),
		Message:   executorHealth.Message,
		Timestamp: executorHealth.LastCheck,
	}
	
	if a.poller != nil {
		pollerHealth := a.poller.GetHealth()
		health.Components["poller"] = &ComponentHealth{
			Status:    string(pollerHealth.Status),
			Message:   pollerHealth.Message,
			Timestamp: pollerHealth.LastCheck,
		}
	}
	
	metricsHealth := a.metrics.GetHealth()
	health.Components["metrics"] = &ComponentHealth{
		Status:    string(metricsHealth.Status),
		Message:   metricsHealth.Message,
		Timestamp: metricsHealth.LastCheck,
	}
	
	healthCheckHealth := a.healthCheck.GetHealth()
	health.Components["health_check"] = &ComponentHealth{
		Status:    string(healthCheckHealth.Status),
		Message:   healthCheckHealth.Message,
		Timestamp: healthCheckHealth.LastCheck,
	}

	overallHealthy := true
	for _, componentHealth := range health.Components {
		if componentHealth.Status != "healthy" {
			overallHealthy = false
		}
	}

	if !overallHealthy {
		health.Status = "degraded"
		health.Message = "One or more components are unhealthy"
	}

	return health
}

// run is the main agent loop
func (a *ActionAgent) run(ctx context.Context) {
	defer close(a.doneChan)

	ticker := time.NewTicker(30 * time.Second) // Health check interval
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
			// Periodic health checks and maintenance
			a.performHealthCheck()
		}
	}
}

// performHealthCheck performs periodic health checks
func (a *ActionAgent) performHealthCheck() {
	health := a.GetHealth()
	
	if health.Status != "healthy" {
		a.logger.Warn("Agent health check failed",
			zap.String("status", string(health.Status)),
			zap.String("message", health.Message))
	} else {
		a.logger.Debug("Agent health check passed")
	}
}

// AgentStatus represents the current status of the action agent
type AgentStatus struct {
	AgentID             string                     `json:"agent_id"`
	TenantID            string                     `json:"tenant_id"`
	Type                string                     `json:"type"`
	Version             string                     `json:"version"`
	Running             bool                       `json:"running"`
	StartTime           time.Time                  `json:"start_time"`
	Uptime              time.Duration              `json:"uptime"`
	Environment         string                     `json:"environment"`
	PluginStatus        map[string]*PluginStatus   `json:"plugin_status,omitempty"`
	ExecutorStatus      *ExecutorStatus            `json:"executor_status,omitempty"`
	OrchestratorStatus  map[string]interface{}     `json:"orchestrator_status,omitempty"`
	HealthStatus        *HealthCheckStatus         `json:"health_status,omitempty"`
	MetricsStatus       *MetricsStatus             `json:"metrics_status,omitempty"`
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

// Define basic types locally
type HealthStatus string
type ComponentHealth struct {
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
type PluginStatus struct {
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthChecker interface for components that can report health
type HealthChecker interface {
	GetHealth() *ComponentHealth
}

// ExecutorStatus represents the status of the action executor
type ExecutorStatus struct {
	ActiveTasks    int `json:"active_tasks"`
	QueuedTasks    int `json:"queued_tasks"`
	CompletedTasks int `json:"completed_tasks"`
	FailedTasks    int `json:"failed_tasks"`
}

type HealthCheckStatus struct {
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type MetricsStatus struct {
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}


