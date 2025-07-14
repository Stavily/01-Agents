// Package agent provides enhanced plugin management with instruction handling
package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Stavily/01-Agents/shared/pkg/config"
	"github.com/Stavily/01-Agents/shared/pkg/instruction"
	"github.com/Stavily/01-Agents/shared/pkg/plugin"
	"github.com/Stavily/01-Agents/shared/pkg/types"
	"go.uber.org/zap"
)

// EnhancedPluginManager extends the basic plugin manager with instruction handling capabilities
type EnhancedPluginManager struct {
	*PluginManager                   // Embed the basic plugin manager
	instructionHandler *instruction.Handler
	factory           *plugin.Factory
	pendingInstructions sync.Map // map[string]*types.Instruction
}

// EnhancedPluginConfig contains configuration for the enhanced plugin manager
type EnhancedPluginConfig struct {
	*config.PluginConfig
	PluginBaseDir string
	GitTimeout    time.Duration
	ExecTimeout   time.Duration
}

// NewEnhancedPluginManager creates a new enhanced plugin manager with instruction handling
func NewEnhancedPluginManager(cfg *EnhancedPluginConfig, logger *zap.Logger) (*EnhancedPluginManager, error) {
	if cfg == nil || cfg.PluginConfig == nil {
		return nil, fmt.Errorf("plugin config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Create base plugin manager
	basePM, err := NewPluginManager(cfg.PluginConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create base plugin manager: %w", err)
	}

	// Set default base directory if not provided
	baseDir := cfg.PluginBaseDir
	if baseDir == "" {
		baseDir = filepath.Join(".", "plugins")
	}

	// Create plugin factory
	factoryConfig := &plugin.FactoryConfig{
		BaseDir:     baseDir,
		GitTimeout:  cfg.GitTimeout,
		ExecTimeout: cfg.ExecTimeout,
	}
	factory := plugin.NewFactory(logger, factoryConfig)

	// Create instruction handler
	handlerConfig := &instruction.HandlerConfig{
		PluginBaseDir: baseDir,
		GitTimeout:    cfg.GitTimeout,
		ExecTimeout:   cfg.ExecTimeout,
	}
	instructionHandler := instruction.NewHandler(logger, handlerConfig)

	return &EnhancedPluginManager{
		PluginManager:      basePM,
		instructionHandler: instructionHandler,
		factory:           factory,
	}, nil
}

// ProcessInstruction processes an instruction from a poll response
func (epm *EnhancedPluginManager) ProcessInstruction(ctx context.Context, response *types.PollResponse) (*types.InstructionResult, error) {
	if response.Instruction == nil {
		return nil, nil
	}

	inst := response.Instruction
	epm.logger.Info("Processing instruction in enhanced plugin manager",
		zap.String("instruction_id", inst.ID),
		zap.String("instruction_type", string(inst.Type)))

	// Validate instruction
	if err := epm.instructionHandler.ValidateInstruction(inst); err != nil {
		epm.logger.Error("Instruction validation failed",
			zap.String("instruction_id", inst.ID),
			zap.Error(err))
		return nil, fmt.Errorf("instruction validation failed: %w", err)
	}

	// Store pending instruction
	epm.pendingInstructions.Store(inst.ID, inst)
	defer epm.pendingInstructions.Delete(inst.ID)

	// Process the instruction
	return epm.instructionHandler.ProcessPollResponse(ctx, response)
}

// InstallPlugin installs a plugin from a repository URL
func (epm *EnhancedPluginManager) InstallPlugin(ctx context.Context, pluginID, repositoryURL, version string) (*types.InstallationResult, error) {
	epm.logger.Info("Installing plugin",
		zap.String("plugin_id", pluginID),
		zap.String("repository_url", repositoryURL),
		zap.String("version", version))

	// Create a synthetic instruction for installation
	inst := &types.Instruction{
		ID:       fmt.Sprintf("install-%s-%d", pluginID, time.Now().Unix()),
		PluginID: pluginID,
		AgentID:  "direct-install", // Placeholder for direct installations
		Status:   types.InstructionStatusPending,
		Type:     types.InstructionTypePluginInstall,
		Source:   types.InstructionSourceAPI,
		PluginConfiguration: map[string]interface{}{
			"plugin_url": repositoryURL,
			"version":    version,
		},
		TimeoutSeconds: 300,
		MaxRetries:     3,
	}

	// Use the factory to create downloader
	downloader := epm.factory.CreateDownloader()
	return downloader.DownloadPlugin(ctx, inst)
}

// ExecutePlugin executes an installed plugin
func (epm *EnhancedPluginManager) ExecutePlugin(ctx context.Context, pluginID, entrypoint string, inputData map[string]interface{}) (*types.ExecutionResult, error) {
	epm.logger.Info("Executing plugin",
		zap.String("plugin_id", pluginID),
		zap.String("entrypoint", entrypoint))

	// Create a synthetic instruction for execution
	inst := &types.Instruction{
		ID:       fmt.Sprintf("exec-%s-%d", pluginID, time.Now().Unix()),
		PluginID: pluginID,
		AgentID:  "direct-exec", // Placeholder for direct executions
		Status:   types.InstructionStatusPending,
		Type:     types.InstructionTypeExecute,
		Source:   types.InstructionSourceAPI,
		PluginConfiguration: map[string]interface{}{
			"entrypoint": entrypoint,
		},
		InputData:      inputData,
		TimeoutSeconds: 300,
		MaxRetries:     1,
	}

	// Use the factory to create executor
	executor := epm.factory.CreateExecutor()
	return executor.ExecutePlugin(ctx, inst)
}

// IsPluginInstalled checks if a plugin is installed
func (epm *EnhancedPluginManager) IsPluginInstalled(pluginID string) bool {
	downloader := epm.factory.CreateDownloader()
	return downloader.IsPluginInstalled(pluginID)
}

// GetInstalledPluginPath returns the installation path for a plugin
func (epm *EnhancedPluginManager) GetInstalledPluginPath(pluginID string) string {
	return filepath.Join(epm.factory.GetBaseDir(), pluginID)
}

// UninstallPlugin removes an installed plugin
func (epm *EnhancedPluginManager) UninstallPlugin(pluginID string) error {
	epm.logger.Info("Uninstalling plugin", zap.String("plugin_id", pluginID))

	downloader := epm.factory.CreateDownloader()
	if err := downloader.CleanupFailedInstallation(pluginID); err != nil {
		return fmt.Errorf("failed to uninstall plugin: %w", err)
	}

	// Also remove from registered plugins if it was registered
	epm.UnregisterPlugin(pluginID)

	return nil
}

// GetPendingInstructions returns all pending instructions
func (epm *EnhancedPluginManager) GetPendingInstructions() map[string]*types.Instruction {
	pending := make(map[string]*types.Instruction)
	
	epm.pendingInstructions.Range(func(key, value interface{}) bool {
		if id, ok := key.(string); ok {
			if inst, ok := value.(*types.Instruction); ok {
				pending[id] = inst
			}
		}
		return true
	})
	
	return pending
}

// GetEnhancedStatus returns enhanced status information including instruction capabilities
func (epm *EnhancedPluginManager) GetEnhancedStatus() map[string]interface{} {
	baseStatus := epm.GetPluginStatuses()
	handlerStatus := epm.instructionHandler.GetStatus()
	pendingCount := 0
	
	epm.pendingInstructions.Range(func(key, value interface{}) bool {
		pendingCount++
		return true
	})

	return map[string]interface{}{
		"plugin_statuses":    baseStatus,
		"instruction_handler": handlerStatus,
		"pending_instructions": pendingCount,
		"base_directory":     epm.factory.GetBaseDir(),
		"capabilities": []string{
			"plugin_install",
			"plugin_execute",
			"instruction_processing",
			"git_clone",
			"multi_runtime_support",
		},
	}
}

// Initialize initializes the enhanced plugin manager
func (epm *EnhancedPluginManager) Initialize(ctx context.Context) error {
	epm.logger.Info("Initializing enhanced plugin manager",
		zap.String("base_dir", epm.factory.GetBaseDir()))

	// Initialize base plugin manager
	if err := epm.PluginManager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize base plugin manager: %w", err)
	}

	// Ensure base directory exists
	if err := ensureDirectory(epm.factory.GetBaseDir()); err != nil {
		return fmt.Errorf("failed to create plugin base directory: %w", err)
	}

	epm.logger.Info("Enhanced plugin manager initialized successfully")
	return nil
}

// Shutdown shuts down the enhanced plugin manager
func (epm *EnhancedPluginManager) Shutdown(ctx context.Context) error {
	epm.logger.Info("Shutting down enhanced plugin manager")

	// Cancel any pending instructions
	epm.pendingInstructions.Range(func(key, value interface{}) bool {
		epm.pendingInstructions.Delete(key)
		return true
	})

	// Shutdown base plugin manager
	return epm.PluginManager.Shutdown(ctx)
}

// ValidateInstructionSupport checks if the manager supports a specific instruction type
func (epm *EnhancedPluginManager) ValidateInstructionSupport(instructionType types.InstructionType) bool {
	switch instructionType {
	case types.InstructionTypePluginInstall, types.InstructionTypeExecute:
		return true
	default:
		return false
	}
}

// GetSupportedInstructionTypes returns the list of supported instruction types
func (epm *EnhancedPluginManager) GetSupportedInstructionTypes() []types.InstructionType {
	return []types.InstructionType{
		types.InstructionTypePluginInstall,
		types.InstructionTypeExecute,
	}
}

// ensureDirectory creates a directory if it doesn't exist
func ensureDirectory(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
} 