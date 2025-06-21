// Package agent implements supporting components for the sensor agent
package agent

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/stavily/agents/shared/pkg/config"
	"github.com/stavily/agents/shared/pkg/plugin"
	sharedagent "github.com/stavily/agents/shared/pkg/agent"
)

// PluginManager is an alias to the shared plugin manager
type PluginManager = sharedagent.PluginManager

// NewPluginManager creates a new plugin manager using the shared implementation
func NewPluginManager(cfg *config.Config, logger *zap.Logger) (*PluginManager, error) {
	return sharedagent.NewPluginManager(&cfg.Plugins, logger)
}





// Metrics wraps the shared MetricsCollector with sensor-specific functionality
type Metrics struct {
	*sharedagent.MetricsCollector
}

// NewMetrics creates a new metrics collector for the sensor agent
func NewMetrics(cfg config.MetricsConfig, logger *zap.Logger) (*Metrics, error) {
	collector, err := sharedagent.NewMetricsCollector(&cfg, logger)
	if err != nil {
		return nil, err
	}
	
	return &Metrics{
		MetricsCollector: collector,
	}, nil
}

// Sensor-specific metrics methods that extend the shared MetricsCollector

// RecordTriggerDetected records a trigger detection event
func (m *Metrics) RecordTriggerDetected() {
	m.IncrementCounter("triggers_detected")
}

// RecordEventProcessed records an event processing event
func (m *Metrics) RecordEventProcessed() {
	m.IncrementCounter("events_processed")
}

// IncrementHeartbeats increments the heartbeat counter
func (m *Metrics) IncrementHeartbeats() {
	m.IncrementCounter("heartbeats")
}

// IncrementHeartbeatErrors increments the heartbeat error counter
func (m *Metrics) IncrementHeartbeatErrors() {
	m.IncrementCounter("heartbeat_errors")
}

// IncrementTriggersDetected increments the triggers detected counter
func (m *Metrics) IncrementTriggersDetected() {
	m.IncrementCounter("triggers_detected")
}

// IncrementEventsDropped increments the events dropped counter
func (m *Metrics) IncrementEventsDropped() {
	m.IncrementCounter("events_dropped")
}

// IncrementEventProcessingErrors increments the event processing errors counter
func (m *Metrics) IncrementEventProcessingErrors() {
	m.IncrementCounter("event_processing_errors")
}

// IncrementEventsProcessed increments the events processed counter
func (m *Metrics) IncrementEventsProcessed() {
	m.IncrementCounter("events_processed")
}

// UpdatePluginHealth updates plugin health metrics
func (m *Metrics) UpdatePluginHealth(pluginID string, health *plugin.Health) {
	m.RecordMetric("plugin_health_"+pluginID, health)
}

// SetActivePlugins sets the number of active plugins
func (m *Metrics) SetActivePlugins(count int) {
	m.SetGauge("active_plugins", float64(count))
}

// SetEventChannelSize sets the event channel size metric
func (m *Metrics) SetEventChannelSize(size int) {
	m.SetGauge("event_channel_size", float64(size))
}

// HealthChecker is an alias to the shared health checker
type HealthChecker = sharedagent.HealthChecker

// NewHealthChecker creates a new health checker for the sensor agent
func NewHealthChecker(cfg *config.HealthConfig, pluginMgr *PluginManager, logger *zap.Logger) (*HealthChecker, error) {
	hc, err := sharedagent.NewHealthChecker(cfg, logger)
	if err != nil {
		return nil, err
	}
	
	// Register plugin manager for health checking
	hc.RegisterComponent("plugin_manager", pluginMgr.GetHealth)
	
	return hc, nil
}

// TriggerDetector handles trigger detection and event generation
type TriggerDetector struct {
	cfg       *config.Config
	pluginMgr *PluginManager
	logger    *zap.Logger
	stats     *DetectorStats
	triggers  map[string]plugin.TriggerPlugin
	mu        sync.RWMutex
}

// DetectorStats tracks trigger detection statistics
type DetectorStats struct {
	TriggersLoaded  int
	TriggersRunning int
	EventsGenerated int
	DetectionErrors int
	LastDetection   time.Time
}

// NewTriggerDetector creates a new trigger detector
func NewTriggerDetector(cfg *config.Config, pluginMgr *PluginManager, logger *zap.Logger) (*TriggerDetector, error) {
	return &TriggerDetector{
		cfg:       cfg,
		pluginMgr: pluginMgr,
		logger:    logger,
		stats:     &DetectorStats{},
		triggers:  make(map[string]plugin.TriggerPlugin),
	}, nil
}

// Start starts the trigger detector
func (td *TriggerDetector) Start(ctx context.Context) error {
	td.logger.Info("Starting trigger detector")
	// TODO: Implement trigger detection logic
	return nil
}

// Stop stops the trigger detector
func (td *TriggerDetector) Stop(ctx context.Context) error {
	td.logger.Info("Stopping trigger detector")
	return nil
}

// GetStatus returns the trigger detector status
func (td *TriggerDetector) GetStatus() *DetectorStatus {
	td.mu.RLock()
	defer td.mu.RUnlock()

	return &DetectorStatus{
		TriggersLoaded:  td.stats.TriggersLoaded,
		TriggersRunning: td.stats.TriggersRunning,
		EventsGenerated: td.stats.EventsGenerated,
		DetectionErrors: td.stats.DetectionErrors,
		LastDetection:   td.stats.LastDetection,
	}
}

// GetHealth returns the trigger detector health
func (td *TriggerDetector) GetHealth() *sharedagent.ComponentHealth {
	return &sharedagent.ComponentHealth{
		Status:     sharedagent.HealthStatusHealthy,
		LastCheck:  time.Now(),
		ErrorCount: 0,
	}
}

// Use shared types from the agent package
type ComponentHealth = sharedagent.ComponentHealth
type HealthStatus = sharedagent.HealthStatus
type PluginStatus = sharedagent.PluginStatus
type HealthCheckStatus = sharedagent.HealthCheckStatus

// Sensor-specific status types
type MetricsStatus struct {
	MetricsExported  int                    `json:"metrics_exported"`
	LastExport       time.Time              `json:"last_export"`
	ExportErrors     int                    `json:"export_errors"`
	TriggersDetected int                    `json:"triggers_detected"`
	EventsProcessed  int                    `json:"events_processed"`
	CustomMetrics    map[string]interface{} `json:"custom_metrics,omitempty"`
}

// DetectorStatus represents the status of trigger detection
type DetectorStatus struct {
	TriggersLoaded  int       `json:"triggers_loaded"`
	TriggersRunning int       `json:"triggers_running"`
	EventsGenerated int       `json:"events_generated"`
	DetectionErrors int       `json:"detection_errors"`
	LastDetection   time.Time `json:"last_detection"`
}
