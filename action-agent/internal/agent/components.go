// Package agent implements supporting components for the action agent
package agent

import (
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/Stavily/01-Agents/shared/pkg/config"
	sharedagent "github.com/Stavily/01-Agents/shared/pkg/agent"
)

// PluginManager is an alias to the shared enhanced plugin manager
type PluginManager = sharedagent.EnhancedPluginManager

// NewPluginManager creates a new enhanced plugin manager using the shared implementation
func NewPluginManager(cfg *config.Config, logger *zap.Logger) (*PluginManager, error) {
	// Create the plugin directory path based on agent base folder
	pluginDir := filepath.Join(cfg.Agent.BaseFolder, "config", "plugins")
	
	// Create enhanced plugin manager configuration
	enhancedCfg := &sharedagent.EnhancedPluginConfig{
		PluginConfig:  &cfg.Plugins,
		PluginBaseDir: pluginDir,
		GitTimeout:    5 * time.Minute,
		ExecTimeout:   10 * time.Minute,
	}
	
	return sharedagent.NewEnhancedPluginManager(enhancedCfg, logger)
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


