// Package instruction provides instruction handling functionality
package instruction

import (
	"context"
	"fmt"
	"time"

	"github.com/Stavily/01-Agents/shared/pkg/plugin"
	"github.com/Stavily/01-Agents/shared/pkg/types"
	"go.uber.org/zap"
)

// Handler handles different types of instructions from polling responses
type Handler struct {
	logger     *zap.Logger
	factory    *plugin.Factory
	downloader *plugin.PluginDownloader
	executor   *plugin.PluginExecutor
}

// HandlerConfig contains configuration for the instruction handler
type HandlerConfig struct {
	PluginBaseDir string
	GitTimeout    time.Duration
	ExecTimeout   time.Duration
}

// NewHandler creates a new instruction handler
func NewHandler(logger *zap.Logger, config *HandlerConfig) *Handler {
	// Create plugin factory
	factoryConfig := &plugin.FactoryConfig{
		BaseDir:     config.PluginBaseDir,
		GitTimeout:  config.GitTimeout,
		ExecTimeout: config.ExecTimeout,
	}
	factory := plugin.NewFactory(logger, factoryConfig)

	// Create components using factory
	downloader := factory.CreateDownloader()
	executor := factory.CreateExecutor()

	return &Handler{
		logger:     logger,
		factory:    factory,
		downloader: downloader,
		executor:   executor,
	}
}

// ProcessPollResponse processes a poll response and handles any instructions
func (h *Handler) ProcessPollResponse(ctx context.Context, response *types.PollResponse) (*types.InstructionResult, error) {
	if response.Instruction == nil {
		h.logger.Debug("No instruction in poll response")
		return nil, nil
	}

	instruction := response.Instruction
	h.logger.Info("Processing instruction",
		zap.String("instruction_id", instruction.ID),
		zap.String("instruction_type", string(instruction.Type)),
		zap.String("plugin_id", instruction.PluginID))

	startTime := time.Now()
	
	switch instruction.Type {
	case types.InstructionTypePluginInstall:
		return h.handlePluginInstall(ctx, instruction, startTime)
	case types.InstructionTypePluginUpdate:
		return h.handlePluginUpdate(ctx, instruction, startTime)
	case types.InstructionTypeExecute:
		return h.handlePluginExecute(ctx, instruction, startTime)
	default:
		return h.createErrorResult(instruction, startTime, fmt.Sprintf("unsupported instruction type: %s", instruction.Type))
	}
}

// handlePluginInstall handles plugin installation instructions
func (h *Handler) handlePluginInstall(ctx context.Context, inst *types.Instruction, startTime time.Time) (*types.InstructionResult, error) {
	h.logger.Info("Handling plugin installation",
		zap.String("instruction_id", inst.ID),
		zap.String("plugin_id", inst.PluginID))

	// Check if plugin is already installed
	if h.downloader.IsPluginInstalled(inst.PluginID) {
		h.logger.Warn("Plugin already installed, skipping",
			zap.String("plugin_id", inst.PluginID))
		
		return &types.InstructionResult{
			InstructionID: inst.ID,
			Type:          inst.Type,
			Success:       true,
			InstallResult: &types.InstallationResult{
				PluginID:      inst.PluginID,
				Success:       true,
				InstalledPath: h.downloader.GetInstalledPluginPath(inst.PluginID),
				Timestamp:     time.Now(),
			},
			ProcessingLogs: []string{"Plugin already installed"},
			StartTime:      startTime,
			EndTime:        time.Now(),
			Duration:       time.Since(startTime).Seconds(),
		}, nil
	}

	// Download and install the plugin
	installResult, err := h.downloader.DownloadPlugin(ctx, inst)
	if err != nil {
		h.logger.Error("Plugin installation failed",
			zap.String("instruction_id", inst.ID),
			zap.String("plugin_id", inst.PluginID),
			zap.Error(err))

		// Cleanup failed installation
		if cleanupErr := h.downloader.CleanupFailedInstallation(inst.PluginID); cleanupErr != nil {
			h.logger.Error("Failed to cleanup failed installation",
				zap.String("plugin_id", inst.PluginID),
				zap.Error(cleanupErr))
		}

		return h.createErrorResult(inst, startTime, fmt.Sprintf("plugin installation failed: %v", err))
	}

	h.logger.Info("Plugin installation completed successfully",
		zap.String("instruction_id", inst.ID),
		zap.String("plugin_id", inst.PluginID),
		zap.String("installation_path", installResult.InstalledPath))

	// Create processing logs from installation logs
	processingLogs := []string{"Starting plugin installation"}
	processingLogs = append(processingLogs, installResult.Logs...)
	processingLogs = append(processingLogs, "Plugin installation completed successfully")

	return &types.InstructionResult{
		InstructionID:   inst.ID,
		Type:            inst.Type,
		Success:         true,
		InstallResult:   installResult,
		ProcessingLogs:  processingLogs,
		StartTime:       startTime,
		EndTime:         time.Now(),
		Duration:        time.Since(startTime).Seconds(),
	}, nil
}

// handlePluginUpdate handles plugin update instructions
func (h *Handler) handlePluginUpdate(ctx context.Context, inst *types.Instruction, startTime time.Time) (*types.InstructionResult, error) {
	h.logger.Info("Handling plugin update",
		zap.String("instruction_id", inst.ID),
		zap.String("plugin_id", inst.PluginID))

	var processingLogs []string
	processingLogs = append(processingLogs, "Starting plugin update")

	// Check if plugin is currently installed
	if h.downloader.IsPluginInstalled(inst.PluginID) {
		h.logger.Info("Plugin is installed, removing for update",
			zap.String("plugin_id", inst.PluginID))
		
		// Remove the existing plugin
		if err := h.downloader.CleanupFailedInstallation(inst.PluginID); err != nil {
			h.logger.Error("Failed to remove existing plugin for update",
				zap.String("plugin_id", inst.PluginID),
				zap.Error(err))
			processingLogs = append(processingLogs, fmt.Sprintf("Failed to remove existing plugin: %v", err))
			return h.createErrorResult(inst, startTime, fmt.Sprintf("failed to remove existing plugin for update: %v", err))
		}
		
		processingLogs = append(processingLogs, "Existing plugin removed successfully")
	} else {
		h.logger.Info("Plugin not currently installed, proceeding with fresh installation",
			zap.String("plugin_id", inst.PluginID))
		processingLogs = append(processingLogs, "Plugin not currently installed, proceeding with fresh installation")
	}

	// Download and install the updated plugin
	installResult, err := h.downloader.DownloadPlugin(ctx, inst)
	if err != nil {
		h.logger.Error("Plugin update failed",
			zap.String("instruction_id", inst.ID),
			zap.String("plugin_id", inst.PluginID),
			zap.Error(err))

		// Cleanup failed installation
		if cleanupErr := h.downloader.CleanupFailedInstallation(inst.PluginID); cleanupErr != nil {
			h.logger.Error("Failed to cleanup failed update",
				zap.String("plugin_id", inst.PluginID),
				zap.Error(cleanupErr))
		}

		processingLogs = append(processingLogs, fmt.Sprintf("Plugin update failed: %v", err))
		return h.createErrorResult(inst, startTime, fmt.Sprintf("plugin update failed: %v", err))
	}

	h.logger.Info("Plugin update completed successfully",
		zap.String("instruction_id", inst.ID),
		zap.String("plugin_id", inst.PluginID),
		zap.String("installation_path", installResult.InstalledPath))

	// Create processing logs from installation logs
	processingLogs = append(processingLogs, installResult.Logs...)
	processingLogs = append(processingLogs, "Plugin update completed successfully")

	return &types.InstructionResult{
		InstructionID:   inst.ID,
		Type:            inst.Type,
		Success:         true,
		InstallResult:   installResult,
		ProcessingLogs:  processingLogs,
		StartTime:       startTime,
		EndTime:         time.Now(),
		Duration:        time.Since(startTime).Seconds(),
	}, nil
}

// handlePluginExecute handles plugin execution instructions
func (h *Handler) handlePluginExecute(ctx context.Context, inst *types.Instruction, startTime time.Time) (*types.InstructionResult, error) {
	h.logger.Info("Handling plugin execution",
		zap.String("instruction_id", inst.ID),
		zap.String("plugin_id", inst.PluginID))

	// Check if plugin is installed
	if !h.downloader.IsPluginInstalled(inst.PluginID) {
		err := fmt.Errorf("plugin not installed: %s", inst.PluginID)
		h.logger.Error("Cannot execute plugin - not installed",
			zap.String("instruction_id", inst.ID),
			zap.String("plugin_id", inst.PluginID))
		
		return h.createErrorResult(inst, startTime, err.Error())
	}

	// Execute the plugin
	execResult, err := h.executor.ExecutePlugin(ctx, inst)
	if err != nil {
		h.logger.Error("Plugin execution failed",
			zap.String("instruction_id", inst.ID),
			zap.String("plugin_id", inst.PluginID),
			zap.Error(err))

		return h.createErrorResult(inst, startTime, fmt.Sprintf("plugin execution failed: %v", err))
	}

	h.logger.Info("Plugin execution completed",
		zap.String("instruction_id", inst.ID),
		zap.String("plugin_id", inst.PluginID),
		zap.Bool("success", execResult.Success))

	// Create processing logs from execution logs
	processingLogs := []string{"Starting plugin execution"}
	processingLogs = append(processingLogs, execResult.Logs...)
	
	finalMessage := "Plugin execution completed successfully"
	if !execResult.Success {
		finalMessage = "Plugin execution failed"
	}
	processingLogs = append(processingLogs, finalMessage)

	result := &types.InstructionResult{
		InstructionID:   inst.ID,
		Type:            inst.Type,
		Success:         execResult.Success,
		ExecutionResult: execResult,
		ProcessingLogs:  processingLogs,
		StartTime:       startTime,
		EndTime:         time.Now(),
		Duration:        time.Since(startTime).Seconds(),
	}

	if !execResult.Success {
		result.Error = execResult.Error
	}

	return result, nil
}

// createErrorResult creates an error result for failed instructions
func (h *Handler) createErrorResult(inst *types.Instruction, startTime time.Time, errorMsg string) (*types.InstructionResult, error) {
	return &types.InstructionResult{
		InstructionID:  inst.ID,
		Type:           inst.Type,
		Success:        false,
		Error:          errorMsg,
		ProcessingLogs: []string{errorMsg},
		StartTime:      startTime,
		EndTime:        time.Now(),
		Duration:       time.Since(startTime).Seconds(),
	}, fmt.Errorf(errorMsg)
}

// ValidateInstruction validates an instruction based on its type
func (h *Handler) ValidateInstruction(inst *types.Instruction) error {
	if inst == nil {
		return fmt.Errorf("instruction cannot be nil")
	}

	// Validate common fields
	if inst.ID == "" {
		return fmt.Errorf("instruction ID is required")
	}
	if inst.PluginID == "" {
		return fmt.Errorf("plugin ID is required")
	}
	if inst.Type == "" {
		return fmt.Errorf("instruction type is required")
	}

	// Validate based on instruction type
	switch inst.Type {
	case types.InstructionTypePluginInstall:
		return h.validatePluginInstallInstruction(inst)
	case types.InstructionTypePluginUpdate:
		return h.validatePluginUpdateInstruction(inst)
	case types.InstructionTypeExecute:
		return h.validatePluginExecuteInstruction(inst)
	default:
		return fmt.Errorf("unsupported instruction type: %s", inst.Type)
	}
}

// validatePluginInstallInstruction validates a plugin install instruction
func (h *Handler) validatePluginInstallInstruction(inst *types.Instruction) error {
	// Check for plugin URL in configuration or metadata (new format)
	if pluginURL, ok := inst.PluginConfiguration["plugin_url"].(string); ok && pluginURL != "" {
		return nil
	}
	
	// Fallback to old format for backward compatibility
	if repoURL, ok := inst.PluginConfiguration["repository_url"].(string); ok && repoURL != "" {
		return nil
	}
	
	if repoURL, ok := inst.Metadata["repository_url"].(string); ok && repoURL != "" {
		return nil
	}

	return fmt.Errorf("plugin_url or repository_url is required for plugin installation")
}

// validatePluginUpdateInstruction validates a plugin update instruction
func (h *Handler) validatePluginUpdateInstruction(inst *types.Instruction) error {
	// Plugin update has the same requirements as plugin install
	return h.validatePluginInstallInstruction(inst)
}

// validatePluginExecuteInstruction validates a plugin execute instruction
func (h *Handler) validatePluginExecuteInstruction(inst *types.Instruction) error {
	// Check for entrypoint in configuration
	if entrypoint, ok := inst.PluginConfiguration["entrypoint"].(string); !ok || entrypoint == "" {
		return fmt.Errorf("entrypoint is required for plugin execution")
	}

	return nil
}

// GetStatus returns the current status of the handler
func (h *Handler) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"handler_type": "plugin_instruction_handler",
		"base_dir":     h.factory.GetBaseDir(),
		"git_timeout":  h.factory.GetGitTimeout(),
		"exec_timeout": h.factory.GetExecTimeout(),
		"components": map[string]interface{}{
			"downloader": "ready",
			"executor":   "ready",
		},
	}
} 