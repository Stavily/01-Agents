// Package agent provides shared orchestrator workflow functionality following AGENT_USE.md specification
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/stavily/agents/shared/pkg/api"
	"github.com/stavily/agents/shared/pkg/config"
	"go.uber.org/zap"
)

// OrchestratorWorkflow represents the shared workflow that both sensor and action agents use
type OrchestratorWorkflow struct {
	cfg                *config.Config
	logger             *zap.Logger
	orchestratorClient *api.OrchestratorClient

	// Runtime state
	mu        sync.RWMutex
	running   bool
	startTime time.Time

	// Channels for coordination
	stopChan chan struct{}
	doneChan chan struct{}

	// Current instruction being processed
	currentInstruction *api.Instruction
	executionLog       []string

	// Plugin executor function (provided by the specific agent)
	pluginExecutor PluginExecutor
}

// PluginExecutor is a function type that specific agents implement to execute plugins
type PluginExecutor func(ctx context.Context, instruction *api.Instruction) (map[string]interface{}, error)

// NewOrchestratorWorkflow creates a new shared orchestrator workflow
func NewOrchestratorWorkflow(cfg *config.Config, logger *zap.Logger, pluginExecutor PluginExecutor) (*OrchestratorWorkflow, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if pluginExecutor == nil {
		return nil, fmt.Errorf("plugin executor is required")
	}

	// Create orchestrator client
	orchestratorClient, err := api.NewOrchestratorClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator client: %w", err)
	}

	return &OrchestratorWorkflow{
		cfg:                cfg,
		logger:             logger,
		orchestratorClient: orchestratorClient,
		pluginExecutor:     pluginExecutor,
		stopChan:           make(chan struct{}),
		doneChan:           make(chan struct{}),
		executionLog:       make([]string, 0),
	}, nil
}

// Start starts the orchestrator workflow
func (w *OrchestratorWorkflow) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return fmt.Errorf("orchestrator workflow is already running")
	}

	w.logger.Info("Starting orchestrator workflow",
		zap.String("agent_id", w.cfg.Agent.ID),
		zap.String("agent_type", w.cfg.Agent.Type),
		zap.String("tenant_id", w.cfg.Agent.TenantID))

	w.running = true
	w.startTime = time.Now()

	// Start the main workflow loop
	go w.run(ctx)

	w.logger.Info("Orchestrator workflow started successfully")
	return nil
}

// Stop stops the orchestrator workflow gracefully
func (w *OrchestratorWorkflow) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	w.logger.Info("Stopping orchestrator workflow")

	// Signal shutdown
	close(w.stopChan)

	// Wait for main loop to finish or timeout
	select {
	case <-w.doneChan:
		w.logger.Info("Orchestrator workflow main loop stopped")
	case <-ctx.Done():
		w.logger.Warn("Orchestrator workflow shutdown timed out")
		return ctx.Err()
	}

	// Send a final "offline" heartbeat before closing the client
	if err := w.orchestratorClient.SendHeartbeat(ctx, "offline"); err != nil {
		w.logger.Error("Failed to send offline heartbeat", zap.Error(err))
	}

	// Close orchestrator client
	if err := w.orchestratorClient.Close(); err != nil {
		w.logger.Error("Error closing orchestrator client", zap.Error(err))
	}

	w.running = false
	w.logger.Info("Orchestrator workflow stopped successfully")
	return nil
}

// IsRunning returns whether the workflow is currently running
func (w *OrchestratorWorkflow) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// run is the main workflow loop implementing the AGENT_USE.md specification
func (w *OrchestratorWorkflow) run(ctx context.Context) {
	defer close(w.doneChan)

	// Use default values if not configured
	heartbeatInterval := w.cfg.Agent.Heartbeat
	if heartbeatInterval <= 0 {
		heartbeatInterval = 30 * time.Second
	}
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	pollInterval := w.cfg.Agent.PollInterval
	if pollInterval <= 0 {
		pollInterval = 10 * time.Second
	}
	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	w.logger.Info("Orchestrator workflow main loop started",
		zap.Duration("heartbeat_interval", heartbeatInterval),
		zap.Duration("poll_interval", pollInterval))

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Orchestrator workflow context cancelled")
			return
		case <-w.stopChan:
			w.logger.Info("Orchestrator workflow stop signal received")
			return
		case <-heartbeatTicker.C:
			w.sendHeartbeat(ctx)
		case <-pollTicker.C:
			w.pollAndProcessInstructions(ctx)
		}
	}
}

// sendHeartbeat sends a heartbeat to the orchestrator
func (w *OrchestratorWorkflow) sendHeartbeat(ctx context.Context) {
	w.logger.Debug("Sending heartbeat")

	if err := w.orchestratorClient.SendHeartbeat(ctx, "online"); err != nil {
		w.logger.Error("Failed to send heartbeat", zap.Error(err))
		return
	}

	w.logger.Debug("Heartbeat sent successfully")
}

// pollAndProcessInstructions polls for instructions and processes them
func (w *OrchestratorWorkflow) pollAndProcessInstructions(ctx context.Context) {
	w.logger.Debug("Polling for instructions")

	// Check if we're already processing an instruction
	w.mu.RLock()
	isProcessing := w.currentInstruction != nil
	w.mu.RUnlock()

	if isProcessing {
		w.logger.Debug("Already processing an instruction, skipping poll")
		return
	}

	// Poll for instructions
	response, err := w.orchestratorClient.PollInstructions(ctx)
	if err != nil {
		w.logger.Error("Failed to poll for instructions", zap.Error(err))
		return
	}

	w.logger.Debug("Poll response received",
		zap.String("status", response.Status),
		zap.Int("next_poll_interval", response.NextPollInterval))

	// Update poll interval based on server response
	if response.NextPollInterval > 0 {
		newInterval := time.Duration(response.NextPollInterval) * time.Second
		w.logger.Debug("Server suggested poll interval", zap.Duration("new_interval", newInterval))
		// Note: In a production implementation, you might want to update the ticker
	}

	// Process instruction if available
	if response.Instruction != nil {
		w.processInstruction(ctx, response.Instruction)
	}
}

// processInstruction processes a single instruction
func (w *OrchestratorWorkflow) processInstruction(ctx context.Context, instruction *api.Instruction) {
	w.mu.Lock()
	w.currentInstruction = instruction
	w.executionLog = []string{"Instruction received"}
	w.mu.Unlock()

	w.logger.Info("Processing instruction",
		zap.String("instruction_id", instruction.ID),
		zap.String("plugin_id", instruction.PluginID))

	// Create a context with timeout for the instruction
	instructionCtx := ctx
	if instruction.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		instructionCtx, cancel = context.WithTimeout(ctx, time.Duration(instruction.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	// Update instruction status to executing
	w.updateInstructionStatus(ctx, instruction.ID, "executing", []string{"Started plugin execution"})

	// Execute the instruction using the provided plugin executor
	result, err := w.pluginExecutor(instructionCtx, instruction)

	// Submit final result
	if err != nil {
		w.submitFailedResult(ctx, instruction.ID, err)
	} else {
		w.submitSuccessResult(ctx, instruction.ID, result)
	}

	// Clear current instruction
	w.mu.Lock()
	w.currentInstruction = nil
	w.executionLog = nil
	w.mu.Unlock()
}

// updateInstructionStatus updates the instruction status during execution
func (w *OrchestratorWorkflow) updateInstructionStatus(ctx context.Context, instructionID, status string, logEntries []string) {
	w.appendExecutionLog(logEntries...)

	update := &api.InstructionUpdateRequest{
		Status:       status,
		ExecutionLog: w.getExecutionLog(),
	}

	response, err := w.orchestratorClient.UpdateInstruction(ctx, instructionID, update)
	if err != nil {
		w.logger.Error("Failed to update instruction status",
			zap.String("instruction_id", instructionID),
			zap.String("status", status),
			zap.Error(err))
		return
	}

	w.logger.Debug("Instruction status updated",
		zap.String("instruction_id", response.InstructionID),
		zap.Strings("updated_fields", response.UpdatedFields))
}

// submitSuccessResult submits a successful execution result
func (w *OrchestratorWorkflow) submitSuccessResult(ctx context.Context, instructionID string, result map[string]interface{}) {
	w.appendExecutionLog("Task completed successfully")

	resultRequest := &api.InstructionResultRequest{
		Status:       "completed",
		Result:       result,
		ExecutionLog: w.getExecutionLog(),
	}

	response, err := w.orchestratorClient.SubmitInstructionResult(ctx, instructionID, resultRequest)
	if err != nil {
		w.logger.Error("Failed to submit success result",
			zap.String("instruction_id", instructionID),
			zap.Error(err))
		return
	}

	w.logger.Info("Success result submitted",
		zap.String("instruction_id", instructionID),
		zap.Bool("acknowledged", response.Acknowledged))
}

// submitFailedResult submits a failed execution result
func (w *OrchestratorWorkflow) submitFailedResult(ctx context.Context, instructionID string, execErr error) {
	w.appendExecutionLog(fmt.Sprintf("Error occurred: %s", execErr.Error()))

	resultRequest := &api.InstructionResultRequest{
		Status:       "failed",
		ErrorMessage: execErr.Error(),
		ErrorDetails: map[string]interface{}{
			"error_type": fmt.Sprintf("%T", execErr),
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		},
		ExecutionLog: w.getExecutionLog(),
	}

	response, err := w.orchestratorClient.SubmitInstructionResult(ctx, instructionID, resultRequest)
	if err != nil {
		w.logger.Error("Failed to submit failed result",
			zap.String("instruction_id", instructionID),
			zap.Error(err))
		return
	}

	w.logger.Info("Failed result submitted",
		zap.String("instruction_id", instructionID),
		zap.Bool("acknowledged", response.Acknowledged),
		zap.String("error", execErr.Error()))
}

// appendExecutionLog appends entries to the execution log
func (w *OrchestratorWorkflow) appendExecutionLog(entries ...string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, entry := range entries {
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
		logEntry := fmt.Sprintf("[%s] %s", timestamp, entry)
		w.executionLog = append(w.executionLog, logEntry)
	}
}

// getExecutionLog returns a copy of the current execution log
func (w *OrchestratorWorkflow) getExecutionLog() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	logCopy := make([]string, len(w.executionLog))
	copy(logCopy, w.executionLog)
	return logCopy
}

// GetStatus returns the current workflow status
func (w *OrchestratorWorkflow) GetStatus() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	status := map[string]interface{}{
		"agent_id":   w.cfg.Agent.ID,
		"tenant_id":  w.cfg.Agent.TenantID,
		"type":       w.cfg.Agent.Type,
		"running":    w.running,
		"start_time": w.startTime,
		"uptime":     time.Since(w.startTime),
	}

	if w.currentInstruction != nil {
		status["current_instruction"] = map[string]interface{}{
			"id":        w.currentInstruction.ID,
			"plugin_id": w.currentInstruction.PluginID,
		}
	}

	return status
}

// GetHealth returns the workflow health information
func (w *OrchestratorWorkflow) GetHealth() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	health := map[string]interface{}{
		"agent_id":  w.cfg.Agent.ID,
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(w.startTime),
	}

	if !w.running {
		health["status"] = "unhealthy"
		health["message"] = "Orchestrator workflow is not running"
	}

	return health
}

// AddExecutionLogEntry allows agents to add custom execution log entries
func (w *OrchestratorWorkflow) AddExecutionLogEntry(entry string) {
	w.appendExecutionLog(entry)
} 