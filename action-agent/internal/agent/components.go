// Package agent implements supporting components for the action agent
package agent

import (
	"go.uber.org/zap"

	"github.com/stavily/agents/shared/pkg/config"
	sharedagent "github.com/stavily/agents/shared/pkg/agent"
)

// PluginManager is an alias to the shared plugin manager
type PluginManager = sharedagent.PluginManager

// NewPluginManager creates a new plugin manager using the shared implementation
func NewPluginManager(cfg *config.PluginConfig, logger *zap.Logger) (*PluginManager, error) {
	return sharedagent.NewPluginManager(cfg, logger)
}



// MetricsCollector is an alias to the shared metrics collector
type MetricsCollector = sharedagent.MetricsCollector

// NewMetricsCollector creates a new metrics collector using the shared implementation
func NewMetricsCollector(cfg *config.MetricsConfig, logger *zap.Logger) (*MetricsCollector, error) {
	return sharedagent.NewMetricsCollector(cfg, logger)
}

// HealthMonitor is an alias to the shared health checker
type HealthMonitor = sharedagent.HealthChecker

// NewHealthMonitor creates a new health monitor using the shared implementation
func NewHealthMonitor(cfg *config.HealthConfig, pluginMgr *PluginManager, logger *zap.Logger) (*HealthMonitor, error) {
	hc, err := sharedagent.NewHealthChecker(cfg, logger)
	if err != nil {
		return nil, err
	}
	
	// Register plugin manager for health checking
	hc.RegisterComponent("plugin_manager", pluginMgr.GetHealth)
	
	return hc, nil
}


