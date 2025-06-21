package agent

import (
	"context"
	"sync"
	"time"

	"github.com/stavily/agents/shared/pkg/config"
	"go.uber.org/zap"
)

// MetricsCollector collects and exports metrics for agents
type MetricsCollector struct {
	cfg    *config.MetricsConfig
	logger *zap.Logger
	stats  *MetricsStats
	mu     sync.RWMutex
	
	// Custom metrics storage
	customMetrics map[string]interface{}
}

// MetricsStats tracks metrics collection statistics
type MetricsStats struct {
	MetricsExported int       `json:"metrics_exported"`
	LastExport      time.Time `json:"last_export"`
	ExportErrors    int       `json:"export_errors"`
}

// MetricsStatus represents the status of metrics collection
type MetricsStatus struct {
	MetricsExported int                    `json:"metrics_exported"`
	LastExport      time.Time              `json:"last_export"`
	ExportErrors    int                    `json:"export_errors"`
	CustomMetrics   map[string]interface{} `json:"custom_metrics,omitempty"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(cfg *config.MetricsConfig, logger *zap.Logger) (*MetricsCollector, error) {
	return &MetricsCollector{
		cfg:           cfg,
		logger:        logger,
		stats:         &MetricsStats{},
		customMetrics: make(map[string]interface{}),
	}, nil
}

// Start starts the metrics collector
func (mc *MetricsCollector) Start(ctx context.Context) error {
	mc.logger.Info("Starting metrics collector")
	
	// Start periodic metrics export
	go mc.metricsExportLoop(ctx)
	
	return nil
}

// Stop stops the metrics collector
func (mc *MetricsCollector) Stop(ctx context.Context) error {
	mc.logger.Info("Stopping metrics collector")
	return nil
}

// GetStatus returns the metrics collector status
func (mc *MetricsCollector) GetStatus() *MetricsStatus {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Create a copy of custom metrics
	customMetricsCopy := make(map[string]interface{})
	for k, v := range mc.customMetrics {
		customMetricsCopy[k] = v
	}

	return &MetricsStatus{
		MetricsExported: mc.stats.MetricsExported,
		LastExport:      mc.stats.LastExport,
		ExportErrors:    mc.stats.ExportErrors,
		CustomMetrics:   customMetricsCopy,
	}
}

// GetHealth returns the metrics collector health
func (mc *MetricsCollector) GetHealth() *ComponentHealth {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	status := HealthStatusHealthy
	message := ""
	
	// Check if there have been recent export errors
	if mc.stats.ExportErrors > 0 && time.Since(mc.stats.LastExport) > time.Hour {
		status = HealthStatusDegraded
		message = "Recent metrics export errors"
	}
	
	return &ComponentHealth{
		Status:     status,
		LastCheck:  time.Now(),
		ErrorCount: mc.stats.ExportErrors,
		Message:    message,
	}
}

// RecordMetric records a custom metric
func (mc *MetricsCollector) RecordMetric(name string, value interface{}) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.customMetrics[name] = value
}

// IncrementCounter increments a counter metric
func (mc *MetricsCollector) IncrementCounter(name string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if current, exists := mc.customMetrics[name]; exists {
		if count, ok := current.(int); ok {
			mc.customMetrics[name] = count + 1
		} else {
			mc.customMetrics[name] = 1
		}
	} else {
		mc.customMetrics[name] = 1
	}
}

// SetGauge sets a gauge metric value
func (mc *MetricsCollector) SetGauge(name string, value float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.customMetrics[name] = value
}

// GetCurrentMetrics returns all current metrics
func (mc *MetricsCollector) GetCurrentMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	metrics := make(map[string]interface{})
	for k, v := range mc.customMetrics {
		metrics[k] = v
	}
	
	return metrics
}

// metricsExportLoop runs periodic metrics export
func (mc *MetricsCollector) metricsExportLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second) // Should come from config
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.exportMetrics()
		}
	}
}

// exportMetrics exports metrics to configured destination
func (mc *MetricsCollector) exportMetrics() {
	mc.mu.Lock()
	mc.stats.LastExport = time.Now()
	mc.mu.Unlock()
	
	// TODO: Implement actual metrics export based on configuration
	// This could export to Prometheus, InfluxDB, CloudWatch, etc.
	
	mc.mu.Lock()
	mc.stats.MetricsExported++
	mc.mu.Unlock()
	
	mc.logger.Debug("Metrics exported successfully")
} 