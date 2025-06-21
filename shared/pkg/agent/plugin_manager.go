package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/stavily/agents/shared/pkg/config"
	"github.com/stavily/agents/shared/pkg/plugin"
	"go.uber.org/zap"
)

// PluginManager manages plugins for agents
type PluginManager struct {
	cfg     *config.PluginConfig
	logger  *zap.Logger
	plugins map[string]plugin.Plugin
	mu      sync.RWMutex
}

// PluginStatus represents the status of plugins
type PluginStatus struct {
	Loaded  int `json:"loaded"`
	Running int `json:"running"`
	Errors  int `json:"errors"`
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(cfg *config.PluginConfig, logger *zap.Logger) (*PluginManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("plugin config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	
	return &PluginManager{
		cfg:     cfg,
		logger:  logger,
		plugins: make(map[string]plugin.Plugin),
	}, nil
}

// Initialize initializes the plugin manager
func (pm *PluginManager) Initialize(ctx context.Context) error {
	pm.logger.Info("Initializing plugin manager")
	// TODO: Implement plugin discovery and loading
	return nil
}

// Shutdown shuts down the plugin manager
func (pm *PluginManager) Shutdown(ctx context.Context) error {
	pm.logger.Info("Shutting down plugin manager")
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Stop all plugins
	for id, p := range pm.plugins {
		if err := p.Stop(ctx); err != nil {
			pm.logger.Error("Failed to stop plugin during shutdown",
				zap.String("plugin_id", id),
				zap.Error(err))
		}
	}
	
	return nil
}

// GetPluginStatuses returns the status of all plugins
func (pm *PluginManager) GetPluginStatuses() map[string]*PluginStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	statuses := make(map[string]*PluginStatus)
	
	for id, p := range pm.plugins {
		status := &PluginStatus{
			Loaded: 1,
		}
		
		if p.GetStatus() == plugin.StatusRunning {
			status.Running = 1
		}
		
		statuses[id] = status
	}
	
	return statuses
}

// ListPluginsByType returns plugins of a specific type
func (pm *PluginManager) ListPluginsByType(pluginType plugin.PluginType) []plugin.Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	var plugins []plugin.Plugin
	for _, p := range pm.plugins {
		if p.GetInfo().Type == pluginType {
			plugins = append(plugins, p)
		}
	}
	
	return plugins
}

// GetHealth returns the plugin manager health
func (pm *PluginManager) GetHealth() *ComponentHealth {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	status := HealthStatusHealthy
	errorCount := 0
	
	// Check plugin health
	for _, p := range pm.plugins {
		if health := p.GetHealth(); health != nil && health.Status != plugin.HealthStatusHealthy {
			errorCount++
			if status == HealthStatusHealthy {
				status = HealthStatusDegraded
			}
		}
	}
	
	if errorCount > len(pm.plugins)/2 {
		status = HealthStatusUnhealthy
	}
	
	return &ComponentHealth{
		Status:     status,
		LastCheck:  time.Now(),
		ErrorCount: errorCount,
	}
}

// RegisterPlugin registers a plugin
func (pm *PluginManager) RegisterPlugin(p plugin.Plugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	info := p.GetInfo()
	if info == nil {
		return fmt.Errorf("plugin info is nil")
	}
	
	if _, exists := pm.plugins[info.ID]; exists {
		return fmt.Errorf("plugin with ID %s already registered", info.ID)
	}
	
	pm.plugins[info.ID] = p
	pm.logger.Info("Plugin registered", zap.String("plugin_id", info.ID))
	
	return nil
}

// UnregisterPlugin unregisters a plugin
func (pm *PluginManager) UnregisterPlugin(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	p, exists := pm.plugins[id]
	if !exists {
		return fmt.Errorf("plugin with ID %s not found", id)
	}
	
	// Stop the plugin if it's running
	if p.GetStatus() == plugin.StatusRunning {
		if err := p.Stop(context.Background()); err != nil {
			pm.logger.Warn("Failed to stop plugin during unregistration",
				zap.String("plugin_id", id),
				zap.Error(err))
		}
	}
	
	delete(pm.plugins, id)
	pm.logger.Info("Plugin unregistered", zap.String("plugin_id", id))
	
	return nil
}

// GetPlugin returns a plugin by ID
func (pm *PluginManager) GetPlugin(id string) (plugin.Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	p, exists := pm.plugins[id]
	if !exists {
		return nil, fmt.Errorf("plugin with ID %s not found", id)
	}
	
	return p, nil
}

// ListPlugins returns all plugins
func (pm *PluginManager) ListPlugins() []plugin.Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	plugins := make([]plugin.Plugin, 0, len(pm.plugins))
	for _, p := range pm.plugins {
		plugins = append(plugins, p)
	}
	
	return plugins
}

// GetPluginInfo returns plugin info by ID
func (pm *PluginManager) GetPluginInfo(id string) (*plugin.Info, error) {
	p, err := pm.GetPlugin(id)
	if err != nil {
		return nil, err
	}
	
	return p.GetInfo(), nil
}

// LoadPlugin loads a plugin from path
func (pm *PluginManager) LoadPlugin(ctx context.Context, path string) (plugin.Plugin, error) {
	pm.logger.Info("Loading plugin", zap.String("path", path))
	// TODO: Implement plugin loading from path
	return nil, fmt.Errorf("plugin loading not implemented")
}

// UnloadPlugin unloads a plugin
func (pm *PluginManager) UnloadPlugin(ctx context.Context, p plugin.Plugin) error {
	return pm.UnregisterPlugin(p.GetInfo().ID)
}

// ReloadPlugin reloads a plugin
func (pm *PluginManager) ReloadPlugin(ctx context.Context, p plugin.Plugin) (plugin.Plugin, error) {
	info := p.GetInfo()
	if err := pm.UnloadPlugin(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to unload plugin: %w", err)
	}
	
	// TODO: Implement plugin reloading
	pm.logger.Info("Plugin reloaded", zap.String("plugin_id", info.ID))
	return nil, fmt.Errorf("plugin reloading not implemented")
}

// ValidatePlugin validates a plugin before loading
func (pm *PluginManager) ValidatePlugin(path string) error {
	pm.logger.Info("Validating plugin", zap.String("path", path))
	// TODO: Implement plugin validation
	return nil
}

// StartPlugin starts a plugin
func (pm *PluginManager) StartPlugin(ctx context.Context, id string) error {
	p, err := pm.GetPlugin(id)
	if err != nil {
		return err
	}
	
	return p.Start(ctx)
}

// StopPlugin stops a plugin
func (pm *PluginManager) StopPlugin(ctx context.Context, id string) error {
	p, err := pm.GetPlugin(id)
	if err != nil {
		return err
	}
	
	return p.Stop(ctx)
}

// RestartPlugin restarts a plugin
func (pm *PluginManager) RestartPlugin(ctx context.Context, id string) error {
	p, err := pm.GetPlugin(id)
	if err != nil {
		return err
	}
	
	if err := p.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop plugin: %w", err)
	}
	
	return p.Start(ctx)
}

// GetPluginStatus gets the status of a plugin
func (pm *PluginManager) GetPluginStatus(id string) (plugin.Status, error) {
	p, err := pm.GetPlugin(id)
	if err != nil {
		return "", err
	}
	
	return p.GetStatus(), nil
}

// GetPluginHealth gets the health of a plugin
func (pm *PluginManager) GetPluginHealth(id string) (*plugin.Health, error) {
	p, err := pm.GetPlugin(id)
	if err != nil {
		return nil, err
	}
	
	return p.GetHealth(), nil
}

// UpdatePlugin updates a plugin to a new version
func (pm *PluginManager) UpdatePlugin(ctx context.Context, id string, version string) error {
	pm.logger.Info("Updating plugin", zap.String("plugin_id", id), zap.String("version", version))
	// TODO: Implement plugin updates
	return fmt.Errorf("plugin updates not implemented")
}

// ConfigurePlugin configures a plugin with new settings
func (pm *PluginManager) ConfigurePlugin(ctx context.Context, id string, config map[string]interface{}) error {
	p, err := pm.GetPlugin(id)
	if err != nil {
		return err
	}
	
	return p.Initialize(ctx, config)
} 