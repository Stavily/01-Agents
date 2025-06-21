// Package agent implements the core sensor agent functionality
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

// SensorAgent represents the main sensor agent
type SensorAgent struct {
	config        *config.Config
	logger        *zap.Logger
	apiClient     *api.Client
	pluginManager plugin.PluginManager

	// Agent lifecycle
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	mu      sync.RWMutex

	// Heartbeat and registration
	heartbeatTicker *time.Ticker
	registrationID  string

	// Trigger detection
	triggerPlugins []plugin.TriggerPlugin
	eventChannel   chan *plugin.TriggerEvent

	// Metrics and monitoring
	metrics *Metrics
}

// NewSensorAgent creates a new sensor agent instance
func NewSensorAgent(cfg *config.Config, logger *zap.Logger) (*SensorAgent, error) {
	// Create API client
	apiClient, err := api.NewClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
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

	agent := &SensorAgent{
		config:        cfg,
		logger:        logger,
		apiClient:     apiClient,
		pluginManager: pluginManager,
		metrics:       metrics,
		eventChannel:  make(chan *plugin.TriggerEvent, 100), // Buffered channel
	}

	return agent, nil
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

	// Register with orchestrator
	if err := s.registerWithOrchestrator(); err != nil {
		return fmt.Errorf("failed to register with orchestrator: %w", err)
	}

	// Load and start trigger plugins
	if err := s.loadTriggerPlugins(); err != nil {
		return fmt.Errorf("failed to load trigger plugins: %w", err)
	}

	// Start core services
	s.wg.Add(4)
	go s.heartbeatLoop()
	go s.eventProcessingLoop()
	go s.pluginMonitoringLoop()
	go s.metricsCollectionLoop()

	// Start metrics server if enabled
	if s.config.Metrics.Enabled {
		if err := s.metrics.Start(); err != nil {
			s.logger.Warn("Failed to start metrics server", zap.Error(err))
		}
	}

	s.started = true
	s.logger.Info("Sensor agent started successfully",
		zap.String("agent_id", s.config.Agent.ID),
		zap.String("registration_id", s.registrationID))

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

	// Stop heartbeat ticker
	if s.heartbeatTicker != nil {
		s.heartbeatTicker.Stop()
	}

	// Stop trigger plugins
	s.stopTriggerPlugins()

	// Stop metrics server
	if s.metrics != nil {
		if err := s.metrics.Stop(); err != nil {
			s.logger.Error("Failed to stop metrics", zap.Error(err))
		}
	}

	// Deregister from orchestrator
	if err := s.deregisterFromOrchestrator(); err != nil {
		s.logger.Warn("Failed to deregister from orchestrator", zap.Error(err))
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

	// Close resources
	if s.apiClient != nil {
		s.apiClient.Close()
	}

	close(s.eventChannel)
	s.started = false

	s.logger.Info("Sensor agent stopped")
	return nil
}

// registerWithOrchestrator registers the agent with the orchestrator
func (s *SensorAgent) registerWithOrchestrator() error {
	s.logger.Info("Registering with orchestrator")

	registrationData := map[string]interface{}{
		"agent_id":     s.config.Agent.ID,
		"agent_type":   s.config.Agent.Type,
		"tenant_id":    s.config.Agent.TenantID,
		"name":         s.config.Agent.Name,
		"version":      s.config.Agent.Version,
		"environment":  s.config.Agent.Environment,
		"region":       s.config.Agent.Region,
		"tags":         s.config.Agent.Tags,
		"capabilities": s.getAgentCapabilities(),
		"timestamp":    time.Now().UTC(),
	}

	resp, err := s.apiClient.Post(s.ctx, s.config.API.AgentsEndpoint+"/register", registrationData)
	if err != nil {
		return fmt.Errorf("registration request failed: %w", err)
	}

	if resp.StatusCode != 201 {
		return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Parse registration response to get registration ID
	// This is a simplified implementation - in practice, you'd parse JSON response
	s.registrationID = s.config.GetFullAgentID()

	s.logger.Info("Successfully registered with orchestrator",
		zap.String("registration_id", s.registrationID))

	return nil
}

// deregisterFromOrchestrator deregisters the agent from the orchestrator
func (s *SensorAgent) deregisterFromOrchestrator() error {
	if s.registrationID == "" {
		return nil
	}

	s.logger.Info("Deregistering from orchestrator")

	endpoint := fmt.Sprintf("%s/%s/deregister", s.config.API.AgentsEndpoint, s.registrationID)
	resp, err := s.apiClient.Delete(s.ctx, endpoint)
	if err != nil {
		return fmt.Errorf("deregistration request failed: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		return fmt.Errorf("deregistration failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	s.logger.Info("Successfully deregistered from orchestrator")
	return nil
}

// heartbeatLoop sends periodic heartbeats to the orchestrator
func (s *SensorAgent) heartbeatLoop() {
	defer s.wg.Done()

	s.heartbeatTicker = time.NewTicker(s.config.Agent.Heartbeat)
	defer s.heartbeatTicker.Stop()

	s.logger.Debug("Starting heartbeat loop",
		zap.Duration("interval", s.config.Agent.Heartbeat))

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Debug("Heartbeat loop stopping")
			return
		case <-s.heartbeatTicker.C:
			if err := s.sendHeartbeat(); err != nil {
				s.logger.Error("Failed to send heartbeat", zap.Error(err))
				s.metrics.IncrementHeartbeatErrors()
			} else {
				s.metrics.IncrementHeartbeats()
			}
		}
	}
}

// sendHeartbeat sends a heartbeat to the orchestrator
func (s *SensorAgent) sendHeartbeat() error {
	heartbeatData := map[string]interface{}{
		"agent_id":     s.registrationID,
		"timestamp":    time.Now().UTC(),
		"status":       "healthy",
		"uptime":       time.Since(time.Now()).Seconds(), // This would be calculated properly
		"plugin_count": len(s.triggerPlugins),
		"metrics":      s.metrics.GetCurrentMetrics(),
	}

	endpoint := fmt.Sprintf("%s/%s/heartbeat", s.config.API.AgentsEndpoint, s.registrationID)
	resp, err := s.apiClient.Post(s.ctx, endpoint, heartbeatData)
	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("heartbeat failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	s.logger.Debug("Heartbeat sent successfully")
	return nil
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
		"agent_id":     s.registrationID,
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

	// Send to orchestrator
	resp, err := s.apiClient.Post(s.ctx, "/api/v1/triggers", eventData)
	if err != nil {
		return fmt.Errorf("failed to send trigger event: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("trigger event rejected with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	s.logger.Debug("Trigger event sent successfully",
		zap.String("event_id", event.ID),
		zap.Int("status_code", resp.StatusCode))

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
func (s *SensorAgent) getAgentCapabilities() []string {
	capabilities := []string{
		"trigger_detection",
		"plugin_management",
		"metrics_export",
		"health_monitoring",
	}

	if s.config.Security.TLS.Enabled {
		capabilities = append(capabilities, "tls_communication")
	}

	if s.config.Security.Audit.Enabled {
		capabilities = append(capabilities, "audit_logging")
	}

	return capabilities
}

// GetStatus returns the current status of the sensor agent
func (s *SensorAgent) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"started":          s.started,
		"registration_id":  s.registrationID,
		"plugin_count":     len(s.triggerPlugins),
		"event_queue_size": len(s.eventChannel),
		"metrics":          s.metrics.GetCurrentMetrics(),
	}
}
