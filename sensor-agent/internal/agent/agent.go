// Package agent implements the core sensor agent functionality
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
	"github.com/Stavily/01-Agents/shared/pkg/plugin"
	"github.com/Stavily/01-Agents/shared/pkg/types"
)

// SensorAgent represents the main sensor agent
type SensorAgent struct {
	config            *config.Config
	logger            *zap.Logger
	orchestratorFlow  *agent.OrchestratorWorkflow
	pluginManager     *PluginManager

	// Agent lifecycle
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	mu      sync.RWMutex

	// Trigger detection
	triggerPlugins []plugin.TriggerPlugin
	eventChannel   chan *plugin.TriggerEvent

	// Metrics and monitoring
	metrics *Metrics
}

// NewSensorAgent creates a new sensor agent instance
func NewSensorAgent(cfg *config.Config, logger *zap.Logger) (*SensorAgent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Create plugin manager
	pluginManager, err := NewPluginManager(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin manager: %w", err)
	}

	// Initialize metrics
	metrics, err := NewMetrics(cfg.Metrics, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	sensorAgent := &SensorAgent{
		config:        cfg,
		logger:        logger,
		pluginManager: pluginManager,
		metrics:       metrics,
		eventChannel:  make(chan *plugin.TriggerEvent, 100), // Buffered channel
	}

	// Create orchestrator workflow with sensor-specific plugin executor
	orchestratorFlow, err := agent.NewOrchestratorWorkflow(cfg, logger, sensorAgent.executeSensorPlugin)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator workflow: %w", err)
	}
	sensorAgent.orchestratorFlow = orchestratorFlow

	return sensorAgent, nil
}

// Start starts the sensor agent
func (s *SensorAgent) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("sensor agent is already started")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.logger.Info("Starting sensor agent")

	// Load and start trigger plugins
	if err := s.loadTriggerPlugins(); err != nil {
		return fmt.Errorf("failed to load trigger plugins: %w", err)
	}

	// Start orchestrator workflow
	if err := s.orchestratorFlow.Start(ctx); err != nil {
		return fmt.Errorf("failed to start orchestrator workflow: %w", err)
	}

	// Start core services
	s.wg.Add(3)
	go s.eventProcessingLoop()
	go s.pluginMonitoringLoop()
	go s.metricsCollectionLoop()

	// Start metrics server if enabled
	if s.config.Metrics.Enabled {
		if err := s.metrics.Start(ctx); err != nil {
			s.logger.Warn("Failed to start metrics server", zap.Error(err))
		}
	}

	s.started = true
	s.logger.Info("Sensor agent started successfully",
		zap.String("agent_id", s.config.Agent.ID))

	return nil
}

// Stop stops the sensor agent gracefully
func (s *SensorAgent) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	s.logger.Info("Stopping sensor agent")

	// Cancel context to signal shutdown
	s.cancel()

	// Stop orchestrator workflow
	if err := s.orchestratorFlow.Stop(ctx); err != nil {
		s.logger.Error("Failed to stop orchestrator workflow", zap.Error(err))
	}

	// Stop trigger plugins
	s.stopTriggerPlugins()

	// Stop metrics server
	if s.metrics != nil {
		if err := s.metrics.Stop(ctx); err != nil {
			s.logger.Error("Failed to stop metrics", zap.Error(err))
		}
	}

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("All goroutines stopped")
	case <-ctx.Done():
		s.logger.Warn("Shutdown timeout, some goroutines may not have stopped cleanly")
	}

	close(s.eventChannel)
	s.started = false

	s.logger.Info("Sensor agent stopped")
	return nil
}

// executeSensorPlugin executes sensor-specific plugins for instructions
func (s *SensorAgent) executeSensorPlugin(ctx context.Context, instruction *api.Instruction) (map[string]interface{}, error) {
	s.logger.Info("Executing sensor plugin",
		zap.String("plugin_id", instruction.PluginID),
		zap.Any("input_data", instruction.InputData))

	// Convert api.Instruction to types.Instruction for enhanced plugin manager
	typesInstruction := s.convertAPIInstructionToTypes(instruction)
	
	// Create a poll response with the instruction
	pollResponse := &types.PollResponse{
		Instruction:      typesInstruction,
		Status:           "instruction_delivered",
		NextPollInterval: 5,
	}

	// Process the instruction using the enhanced plugin manager
	result, err := s.pluginManager.ProcessInstruction(ctx, pollResponse)
	if err != nil {
		s.logger.Error("Failed to process instruction",
			zap.String("instruction_id", instruction.ID),
			zap.Error(err))
		return nil, err
	}

	if result == nil {
		s.logger.Info("No result from instruction processing")
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

	s.logger.Info("Sensor plugin executed successfully",
		zap.String("instruction_id", instruction.ID),
		zap.Bool("success", result.Success))

	return resultMap, nil
}

// convertAPIInstructionToTypes converts an api.Instruction to types.Instruction
func (s *SensorAgent) convertAPIInstructionToTypes(apiInst *api.Instruction) *types.Instruction {
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
		AgentID:             s.config.Agent.ID, // Use the agent's ID
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





// loadTriggerPlugins loads and starts all trigger plugins
func (s *SensorAgent) loadTriggerPlugins() error {
	s.logger.Info("Loading trigger plugins")

	// Get all trigger plugins from plugin manager
	plugins := s.pluginManager.ListPluginsByType(plugin.PluginTypeTrigger)

	for _, p := range plugins {
		triggerPlugin, ok := p.(plugin.TriggerPlugin)
		if !ok {
			s.logger.Warn("Plugin is not a trigger plugin", zap.String("plugin_id", p.GetInfo().ID))
			continue
		}

		// Start the plugin
		if err := triggerPlugin.Start(s.ctx); err != nil {
			s.logger.Error("Failed to start trigger plugin",
				zap.String("plugin_id", p.GetInfo().ID),
				zap.Error(err))
			continue
		}

		s.triggerPlugins = append(s.triggerPlugins, triggerPlugin)

		// Start monitoring this plugin's triggers
		s.wg.Add(1)
		go s.monitorTriggerPlugin(triggerPlugin)

		s.logger.Info("Started trigger plugin",
			zap.String("plugin_id", p.GetInfo().ID),
			zap.String("plugin_name", p.GetInfo().Name))
	}

	s.logger.Info("Loaded trigger plugins", zap.Int("count", len(s.triggerPlugins)))
	return nil
}

// stopTriggerPlugins stops all trigger plugins
func (s *SensorAgent) stopTriggerPlugins() {
	s.logger.Info("Stopping trigger plugins")

	for _, triggerPlugin := range s.triggerPlugins {
		if err := triggerPlugin.Stop(s.ctx); err != nil {
			s.logger.Error("Failed to stop trigger plugin",
				zap.String("plugin_id", triggerPlugin.GetInfo().ID),
				zap.Error(err))
		}
	}

	s.triggerPlugins = nil
}

// monitorTriggerPlugin monitors a trigger plugin for events
func (s *SensorAgent) monitorTriggerPlugin(triggerPlugin plugin.TriggerPlugin) {
	defer s.wg.Done()

	pluginID := triggerPlugin.GetInfo().ID
	s.logger.Debug("Starting trigger monitoring", zap.String("plugin_id", pluginID))

	// Get the trigger event channel from the plugin
	eventChan, err := triggerPlugin.DetectTriggers(s.ctx)
	if err != nil {
		s.logger.Error("Failed to get trigger event channel",
			zap.String("plugin_id", pluginID),
			zap.Error(err))
		return
	}

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("Trigger monitoring stopping", zap.String("plugin_id", pluginID))
			return
		case event, ok := <-eventChan:
			if !ok {
				s.logger.Info("Trigger event channel closed", zap.String("plugin_id", pluginID))
				return
			}

			// Forward event to main event channel
			select {
			case s.eventChannel <- event:
				s.metrics.IncrementTriggersDetected()
			case <-s.ctx.Done():
				return
			default:
				s.logger.Warn("Event channel full, dropping event",
					zap.String("plugin_id", pluginID),
					zap.String("event_id", event.ID))
				s.metrics.IncrementEventsDropped()
			}
		}
	}
}

// eventProcessingLoop processes trigger events and sends them to orchestrator
func (s *SensorAgent) eventProcessingLoop() {
	defer s.wg.Done()

	s.logger.Debug("Starting event processing loop")

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("Event processing loop stopping")
			return
		case event := <-s.eventChannel:
			if err := s.processTriggerEvent(event); err != nil {
				s.logger.Error("Failed to process trigger event",
					zap.String("event_id", event.ID),
					zap.Error(err))
				s.metrics.IncrementEventProcessingErrors()
			} else {
				s.metrics.IncrementEventsProcessed()
			}
		}
	}
}

// processTriggerEvent processes and sends a trigger event to the orchestrator
func (s *SensorAgent) processTriggerEvent(event *plugin.TriggerEvent) error {
	s.logger.Debug("Processing trigger event",
		zap.String("event_id", event.ID),
		zap.String("event_type", event.Type),
		zap.String("source", event.Source))

	// Prepare event data for orchestrator
	eventData := map[string]interface{}{
		"event_type":   "trigger_event",
		"agent_id":     s.config.Agent.ID,
		"tenant_id":    s.config.Agent.TenantID,
		"timestamp":    event.Timestamp,
		"trigger_type": event.Type,
		"payload": map[string]interface{}{
			"id":       event.ID,
			"source":   event.Source,
			"data":     event.Data,
			"metadata": event.Metadata,
			"tags":     event.Tags,
			"severity": event.Severity,
		},
	}

	// Send to orchestrator via workflow (this is a placeholder - in the current architecture,
	// sensor agents don't directly send events to orchestrator, they respond to instructions)
	s.logger.Info("Trigger event detected",
		zap.String("event_id", event.ID),
		zap.String("event_type", event.Type),
		zap.Any("event_data", eventData))

	// For now, we'll just log the event. In a complete implementation, this would
	// either queue the event for later processing or send it through a different channel

	s.logger.Debug("Trigger event processed successfully",
		zap.String("event_id", event.ID))

	return nil
}

// pluginMonitoringLoop monitors plugin health and status
func (s *SensorAgent) pluginMonitoringLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	s.logger.Debug("Starting plugin monitoring loop")

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("Plugin monitoring loop stopping")
			return
		case <-ticker.C:
			s.checkPluginHealth()
		}
	}
}

// checkPluginHealth checks the health of all plugins
func (s *SensorAgent) checkPluginHealth() {
	for _, triggerPlugin := range s.triggerPlugins {
		health := triggerPlugin.GetHealth()
		pluginID := triggerPlugin.GetInfo().ID

		switch health.Status {
		case plugin.HealthStatusHealthy:
			s.logger.Debug("Plugin healthy", zap.String("plugin_id", pluginID))
		case plugin.HealthStatusDegraded:
			s.logger.Warn("Plugin degraded",
				zap.String("plugin_id", pluginID),
				zap.String("message", health.Message))
		case plugin.HealthStatusUnhealthy:
			s.logger.Error("Plugin unhealthy",
				zap.String("plugin_id", pluginID),
				zap.String("message", health.Message),
				zap.String("last_error", health.LastError))
			// Could implement plugin restart logic here
		}

		s.metrics.UpdatePluginHealth(pluginID, health)
	}
}

// metricsCollectionLoop collects and updates metrics
func (s *SensorAgent) metricsCollectionLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // Collect every 10 seconds
	defer ticker.Stop()

	s.logger.Debug("Starting metrics collection loop")

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("Metrics collection loop stopping")
			return
		case <-ticker.C:
			s.collectMetrics()
		}
	}
}

// collectMetrics collects current metrics
func (s *SensorAgent) collectMetrics() {
	// Update agent-level metrics
	s.metrics.SetActivePlugins(len(s.triggerPlugins))
	s.metrics.SetEventChannelSize(len(s.eventChannel))

	// Could add more metrics collection here
}

// getAgentCapabilities returns the capabilities of this agent


// IsRunning returns whether the sensor agent is currently running
func (s *SensorAgent) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

// GetStatus returns the current status of the sensor agent
func (s *SensorAgent) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"agent_id":         s.config.Agent.ID,
		"tenant_id":        s.config.Agent.TenantID,
		"type":             "sensor",
		"running":          s.started,
		"plugin_count":     len(s.triggerPlugins),
		"event_queue_size": len(s.eventChannel),
		"metrics":          s.metrics.GetCurrentMetrics(),
	}

	// Add orchestrator workflow status
	if s.orchestratorFlow != nil {
		status["orchestrator_workflow"] = s.orchestratorFlow.GetStatus()
	}

	return status
}

// GetHealth returns the health status of the sensor agent
func (s *SensorAgent) GetHealth() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	components := make(map[string]interface{})
	
	// Add plugin health
	for _, triggerPlugin := range s.triggerPlugins {
		pluginID := triggerPlugin.GetInfo().ID
		health := triggerPlugin.GetHealth()
		components[pluginID] = map[string]interface{}{
			"status":     string(health.Status),
			"message":    health.Message,
			"last_error": health.LastError,
		}
	}

	return map[string]interface{}{
		"agent_id":   s.config.Agent.ID,
		"status":     "healthy", // Could be more sophisticated
		"components": components,
	}
}


