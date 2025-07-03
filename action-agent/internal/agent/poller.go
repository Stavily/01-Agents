// Package agent implements task polling functionality for the action agent
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/stavily/agents/shared/pkg/agent"
	"github.com/stavily/agents/shared/pkg/api"
	"github.com/stavily/agents/shared/pkg/config"
)

// TaskPoller polls the orchestrator for pending action tasks
type TaskPoller struct {
	cfg       *config.Config
	apiClient *api.Client
	executor  *ActionExecutor
	logger    *zap.Logger

	// Runtime state
	mu           sync.RWMutex
	running      bool
	pollInterval time.Duration
	stats        *PollerStats

	// Channels for coordination
	stopChan chan struct{}
	doneChan chan struct{}
}

// PollerStats tracks poller statistics
type PollerStats struct {
	TasksReceived int
	PollErrors    int
	LastPollError string
	LastPollTime  time.Time
}

// PollerStatus represents the current status of the task poller
type PollerStatus struct {
	LastPoll      time.Time     `json:"last_poll"`
	PollInterval  time.Duration `json:"poll_interval"`
	TasksReceived int           `json:"tasks_received"`
	PollErrors    int           `json:"poll_errors"`
}

// NewTaskPoller creates a new task poller
func NewTaskPoller(cfg *config.Config, apiClient *api.Client, executor *ActionExecutor, logger *zap.Logger) (*TaskPoller, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if apiClient == nil {
		return nil, fmt.Errorf("API client is required")
	}
	if executor == nil {
		return nil, fmt.Errorf("action executor is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Get poll interval from config, default to 30 seconds
	pollInterval := 30 * time.Second
	if cfg.Agent.PollInterval > 0 {
		pollInterval = cfg.Agent.PollInterval
	}

	return &TaskPoller{
		cfg:          cfg,
		apiClient:    apiClient,
		executor:     executor,
		logger:       logger,
		pollInterval: pollInterval,
		stats:        &PollerStats{},
		stopChan:     make(chan struct{}),
		doneChan:     make(chan struct{}),
	}, nil
}

// Start starts the task poller
func (p *TaskPoller) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("task poller is already running")
	}

	p.logger.Info("Starting task poller",
		zap.Duration("poll_interval", p.pollInterval))

	p.running = true

	// Start the polling loop
	go p.pollLoop(ctx)

	p.logger.Info("Task poller started successfully")
	return nil
}

// Stop stops the task poller gracefully
func (p *TaskPoller) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.logger.Info("Stopping task poller")

	// Signal shutdown
	close(p.stopChan)

	// Wait for polling loop to finish or timeout
	select {
	case <-p.doneChan:
		p.logger.Info("Task poller stopped")
	case <-ctx.Done():
		p.logger.Warn("Task poller shutdown timed out")
		return ctx.Err()
	}

	p.running = false
	return nil
}

// GetStatus returns the current poller status
func (p *TaskPoller) GetStatus() *PollerStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return &PollerStatus{
		LastPoll:      p.stats.LastPollTime,
		PollInterval:  p.pollInterval,
		TasksReceived: p.stats.TasksReceived,
		PollErrors:    p.stats.PollErrors,
	}
}

// GetHealth returns the poller health information
func (p *TaskPoller) GetHealth() *agent.ComponentHealth {
	p.mu.RLock()
	defer p.mu.RUnlock()

	health := &agent.ComponentHealth{
		Status:     agent.HealthStatusHealthy,
		LastCheck:  time.Now(),
		ErrorCount: p.stats.PollErrors,
	}

	if !p.running {
		health.Status = agent.HealthStatusUnhealthy
		health.Message = "Task poller is not running"
		return health
	}

	// Check if we've had recent poll errors
	if p.stats.PollErrors > 0 && time.Since(p.stats.LastPollTime) > p.pollInterval*2 {
		health.Status = agent.HealthStatusDegraded
		health.Message = "Recent polling errors detected"
	}

	// Check if last poll was too long ago
	if time.Since(p.stats.LastPollTime) > p.pollInterval*3 {
		health.Status = agent.HealthStatusUnhealthy
		health.Message = "Polling has stalled"
	}

	return health
}

// pollLoop is the main polling loop
func (p *TaskPoller) pollLoop(ctx context.Context) {
	defer close(p.doneChan)

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	p.logger.Info("Task poller loop started")

	// Do an initial poll
	p.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Task poller context cancelled")
			return
		case <-p.stopChan:
			p.logger.Info("Task poller stop signal received")
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// poll polls the orchestrator for pending tasks
func (p *TaskPoller) poll(ctx context.Context) {
	p.logger.Debug("Polling for tasks")

	p.mu.Lock()
	p.stats.LastPollTime = time.Now()
	p.mu.Unlock()

	// Create poll request
	pollReq := &api.PollRequest{
		AgentID:     p.cfg.Agent.ID,
		TenantID:    p.cfg.Agent.TenantID,
		AgentType:   "action",
		Environment: p.cfg.Agent.Environment,
		MaxTasks:    p.cfg.Agent.MaxConcurrentTasks,
	}

	// Poll for tasks
	response, err := p.apiClient.PollForTasks(ctx, pollReq)
	if err != nil {
		p.mu.Lock()
		p.stats.PollErrors++
		p.stats.LastPollError = err.Error()
		p.mu.Unlock()

		p.logger.Error("Failed to poll for tasks", zap.Error(err))
		return
	}

	if len(response.Tasks) == 0 {
		p.logger.Debug("No tasks received from poll")
		return
	}

	p.logger.Info("Received tasks from poll",
		zap.Int("task_count", len(response.Tasks)))

	p.mu.Lock()
	p.stats.TasksReceived += len(response.Tasks)
	p.mu.Unlock()

	// Submit tasks to executor
	for _, task := range response.Tasks {
		if err := p.executor.SubmitTask(ctx, task); err != nil {
			p.logger.Error("Failed to submit task to executor",
				zap.String("task_id", task.ID),
				zap.Error(err))

			// Report task execution failure back to orchestrator
			if reportErr := p.reportTaskFailure(ctx, task.ID, err); reportErr != nil {
				p.logger.Error("Failed to report task failure",
					zap.String("task_id", task.ID),
					zap.Error(reportErr))
			}
		}
	}
}

// reportTaskFailure reports a task failure back to the orchestrator
func (p *TaskPoller) reportTaskFailure(ctx context.Context, taskID string, err error) error {
	result := &api.TaskResult{
		TaskID:      taskID,
		AgentID:     p.cfg.Agent.ID,
		Status:      "failed",
		Error:       err.Error(),
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}

	return p.apiClient.ReportTaskResult(ctx, result)
}
