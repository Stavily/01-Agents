// Package agent implements action execution functionality for the action agent
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
	sharedagent "github.com/stavily/agents/shared/pkg/agent"
)

// ActionExecutor executes action tasks using plugins
type ActionExecutor struct {
	cfg       *config.Config
	pluginMgr plugin.PluginManager
	apiClient *api.Client
	logger    *zap.Logger

	// Runtime state
	mu            sync.RWMutex
	running       bool
	activeTasks   map[string]*TaskExecution
	taskQueue     chan *api.Task
	stats         *ExecutorStats
	maxConcurrent int

	// Channels for coordination
	stopChan chan struct{}
	doneChan chan struct{}
}

// TaskExecution represents a running task execution
type TaskExecution struct {
	Task      *api.Task
	StartTime time.Time
	Context   context.Context
	Cancel    context.CancelFunc
	Plugin    plugin.ActionPlugin
	Status    TaskStatus
}

// TaskStatus represents the status of a task execution
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusTimeout   TaskStatus = "timeout"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// ExecutorStats tracks executor statistics
type ExecutorStats struct {
	TasksExecuted   int
	TasksCompleted  int
	TasksFailed     int
	TasksTimeout    int
	AverageExecTime time.Duration
	LastExecTime    time.Time
}

// NewActionExecutor creates a new action executor
func NewActionExecutor(cfg *config.Config, pluginMgr plugin.PluginManager, logger *zap.Logger) (*ActionExecutor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if pluginMgr == nil {
		return nil, fmt.Errorf("plugin manager is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	maxConcurrent := cfg.Agent.MaxConcurrentTasks
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // Default to 10 concurrent tasks
	}

	return &ActionExecutor{
		cfg:           cfg,
		pluginMgr:     pluginMgr,
		logger:        logger,
		activeTasks:   make(map[string]*TaskExecution),
		taskQueue:     make(chan *api.Task, maxConcurrent*2), // Buffer for queued tasks
		stats:         &ExecutorStats{},
		maxConcurrent: maxConcurrent,
		stopChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
	}, nil
}

// Start starts the action executor
func (e *ActionExecutor) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("action executor is already running")
	}

	e.logger.Info("Starting action executor",
		zap.Int("max_concurrent_tasks", e.maxConcurrent))

	e.running = true

	// Start worker goroutines
	for i := 0; i < e.maxConcurrent; i++ {
		go e.worker(ctx, i)
	}

	// Start the main coordination loop
	go e.run(ctx)

	e.logger.Info("Action executor started successfully")
	return nil
}

// Stop stops the action executor gracefully
func (e *ActionExecutor) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	e.logger.Info("Stopping action executor")

	// Signal shutdown
	close(e.stopChan)

	// Wait for main loop to finish or timeout
	select {
	case <-e.doneChan:
		e.logger.Info("Action executor stopped")
	case <-ctx.Done():
		e.logger.Warn("Action executor shutdown timed out")
		return ctx.Err()
	}

	// Cancel all active tasks
	e.mu.Lock()
	for taskID, execution := range e.activeTasks {
		e.logger.Info("Cancelling active task", zap.String("task_id", taskID))
		execution.Cancel()
	}
	e.mu.Unlock()

	// Wait for all tasks to complete or timeout
	deadline := time.Now().Add(30 * time.Second)
	for {
		e.mu.RLock()
		activeCount := len(e.activeTasks)
		e.mu.RUnlock()

		if activeCount == 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	e.running = false
	return nil
}

// SubmitTask submits a task for execution
func (e *ActionExecutor) SubmitTask(ctx context.Context, task *api.Task) error {
	e.mu.RLock()
	running := e.running
	e.mu.RUnlock()

	if !running {
		return fmt.Errorf("action executor is not running")
	}

	select {
	case e.taskQueue <- task:
		e.logger.Debug("Task submitted for execution", zap.String("task_id", task.ID))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("task queue is full")
	}
}

// GetStatus returns the current executor status
func (e *ActionExecutor) GetStatus() *ExecutorStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return &ExecutorStatus{
		ActiveTasks:    len(e.activeTasks),
		QueuedTasks:    len(e.taskQueue),
		CompletedTasks: e.stats.TasksCompleted,
		FailedTasks:    e.stats.TasksFailed,
	}
}

// GetHealth returns the executor health information
func (e *ActionExecutor) GetHealth() *ComponentHealth {
	e.mu.RLock()
	defer e.mu.RUnlock()

	health := &ComponentHealth{
		Status:     sharedagent.HealthStatusHealthy,
		LastCheck:  time.Now(),
		ErrorCount: e.stats.TasksFailed,
	}

	if !e.running {
		health.Status = sharedagent.HealthStatusUnhealthy
		health.Message = "Action executor is not running"
		return health
	}

	// Check if task queue is backing up
	queueUtilization := float64(len(e.taskQueue)) / float64(cap(e.taskQueue))
	if queueUtilization > 0.8 {
		health.Status = sharedagent.HealthStatusDegraded
		health.Message = "Task queue is near capacity"
	}

	// Check failure rate
	if e.stats.TasksExecuted > 0 {
		failureRate := float64(e.stats.TasksFailed) / float64(e.stats.TasksExecuted)
		if failureRate > 0.1 { // More than 10% failure rate
			health.Status = sharedagent.HealthStatusDegraded
			health.Message = "High task failure rate"
		}
	}

	return health
}

// run is the main executor coordination loop
func (e *ActionExecutor) run(ctx context.Context) {
	defer close(e.doneChan)

	e.logger.Info("Action executor main loop started")

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Action executor context cancelled")
			return
		case <-e.stopChan:
			e.logger.Info("Action executor stop signal received")
			return
		}
	}
}

// worker is a worker goroutine that processes tasks from the queue
func (e *ActionExecutor) worker(ctx context.Context, workerID int) {
	logger := e.logger.With(zap.Int("worker_id", workerID))
	logger.Info("Action executor worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker context cancelled")
			return
		case <-e.stopChan:
			logger.Info("Worker stop signal received")
			return
		case task := <-e.taskQueue:
			e.executeTask(ctx, task, logger)
		}
	}
}

// executeTask executes a single task
func (e *ActionExecutor) executeTask(ctx context.Context, task *api.Task, logger *zap.Logger) {
	startTime := time.Now()

	logger.Info("Starting task execution",
		zap.String("task_id", task.ID),
		zap.String("task_type", task.Type))

	// Create task execution context with timeout
	taskCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()

	// Create task execution record
	execution := &TaskExecution{
		Task:      task,
		StartTime: startTime,
		Context:   taskCtx,
		Cancel:    cancel,
		Status:    TaskStatusPending,
	}

	// Register active task
	e.mu.Lock()
	e.activeTasks[task.ID] = execution
	e.stats.TasksExecuted++
	e.mu.Unlock()

	// Defer cleanup
	defer func() {
		e.mu.Lock()
		delete(e.activeTasks, task.ID)
		e.stats.LastExecTime = time.Now()
		e.mu.Unlock()
	}()

	// Find appropriate plugin for task type
	actionPlugin, err := e.findActionPlugin(task.Type)
	if err != nil {
		e.handleTaskFailure(task, err, logger)
		return
	}

	execution.Plugin = actionPlugin
	execution.Status = TaskStatusRunning

	// Create action request
	actionReq := &plugin.ActionRequest{
		ID:          task.ID,
		Type:        task.Type,
		Parameters:  task.Parameters,
		Context:     task.Context,
		Timeout:     task.Timeout,
		Metadata:    task.Metadata,
		RequestedAt: task.CreatedAt,
	}

	// Execute action
	result, err := actionPlugin.ExecuteAction(taskCtx, actionReq)
	if err != nil {
		if taskCtx.Err() == context.DeadlineExceeded {
			execution.Status = TaskStatusTimeout
			e.mu.Lock()
			e.stats.TasksTimeout++
			e.mu.Unlock()
			e.handleTaskTimeout(task, logger)
		} else {
			execution.Status = TaskStatusFailed
			e.handleTaskFailure(task, err, logger)
		}
		return
	}

	execution.Status = TaskStatusCompleted
	e.mu.Lock()
	e.stats.TasksCompleted++
	e.mu.Unlock()

	e.handleTaskSuccess(task, result, logger)

	duration := time.Since(startTime)
	logger.Info("Task execution completed",
		zap.String("task_id", task.ID),
		zap.Duration("duration", duration))
}

// findActionPlugin finds an appropriate action plugin for the given task type
func (e *ActionExecutor) findActionPlugin(taskType string) (plugin.ActionPlugin, error) {
	plugins := e.pluginMgr.ListPluginsByType(plugin.PluginTypeAction)

	for _, p := range plugins {
		if actionPlugin, ok := p.(plugin.ActionPlugin); ok {
			config := actionPlugin.GetActionConfig()
			// Check if plugin supports this task type
			if e.pluginSupportsTaskType(config, taskType) {
				return actionPlugin, nil
			}
		}
	}

	return nil, fmt.Errorf("no action plugin found for task type: %s", taskType)
}

// pluginSupportsTaskType checks if a plugin supports the given task type
func (e *ActionExecutor) pluginSupportsTaskType(config *plugin.ActionConfig, taskType string) bool {
	// This is a simplified check - in practice, you might have more sophisticated
	// plugin discovery and matching logic
	return config != nil && config.Description != ""
}

// handleTaskSuccess handles successful task completion
func (e *ActionExecutor) handleTaskSuccess(task *api.Task, result *plugin.ActionResult, logger *zap.Logger) {
	// Report success to orchestrator
	taskResult := &api.TaskResult{
		TaskID:      task.ID,
		AgentID:     e.cfg.Agent.ID,
		Status:      "completed",
		Data:        result.Data,
		StartedAt:   result.StartedAt,
		CompletedAt: result.CompletedAt,
		Duration:    result.Duration,
		Metadata:    result.Metadata,
	}

	if err := e.apiClient.ReportTaskResult(context.Background(), taskResult); err != nil {
		logger.Error("Failed to report task success",
			zap.String("task_id", task.ID),
			zap.Error(err))
	}
}

// handleTaskFailure handles task execution failure
func (e *ActionExecutor) handleTaskFailure(task *api.Task, err error, logger *zap.Logger) {
	e.mu.Lock()
	e.stats.TasksFailed++
	e.mu.Unlock()

	logger.Error("Task execution failed",
		zap.String("task_id", task.ID),
		zap.Error(err))

	// Report failure to orchestrator
	taskResult := &api.TaskResult{
		TaskID:      task.ID,
		AgentID:     e.cfg.Agent.ID,
		Status:      "failed",
		Error:       err.Error(),
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}

	if reportErr := e.apiClient.ReportTaskResult(context.Background(), taskResult); reportErr != nil {
		logger.Error("Failed to report task failure",
			zap.String("task_id", task.ID),
			zap.Error(reportErr))
	}
}

// handleTaskTimeout handles task execution timeout
func (e *ActionExecutor) handleTaskTimeout(task *api.Task, logger *zap.Logger) {
	logger.Warn("Task execution timed out",
		zap.String("task_id", task.ID),
		zap.Duration("timeout", task.Timeout))

	// Report timeout to orchestrator
	taskResult := &api.TaskResult{
		TaskID:      task.ID,
		AgentID:     e.cfg.Agent.ID,
		Status:      "timeout",
		Error:       "Task execution timed out",
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}

	if err := e.apiClient.ReportTaskResult(context.Background(), taskResult); err != nil {
		logger.Error("Failed to report task timeout",
			zap.String("task_id", task.ID),
			zap.Error(err))
	}
}
