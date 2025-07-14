// Package plugin provides plugin execution functionality
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Stavily/01-Agents/shared/pkg/types"
	"go.uber.org/zap"
)

// PluginExecutor handles executing installed plugins
type PluginExecutor struct {
	logger         *zap.Logger
	baseDir        string
	defaultTimeout time.Duration
}

// ExecutionConfig contains configuration for plugin execution
type ExecutionConfig struct {
	Entrypoint        string                 `json:"entrypoint"`
	WorkingDirectory  string                 `json:"working_directory"`
	Environment       map[string]string      `json:"environment"`
	Arguments         []string               `json:"arguments"`
	Timeout           time.Duration          `json:"timeout"`
	InputData         map[string]interface{} `json:"input_data"`
	Context           map[string]interface{} `json:"context"`
	Variables         map[string]interface{} `json:"variables"`
}

// Runtime represents different plugin runtime environments
type Runtime string

const (
	RuntimePython     Runtime = "python"
	RuntimeNode       Runtime = "node"
	RuntimeGo         Runtime = "go"
	RuntimeBash       Runtime = "bash"
	RuntimeDocker     Runtime = "docker"
	RuntimeExecutable Runtime = "executable"
)

// NewPluginExecutor creates a new plugin executor
func NewPluginExecutor(logger *zap.Logger, baseDir string) *PluginExecutor {
	return &PluginExecutor{
		logger:         logger,
		baseDir:        baseDir,
		defaultTimeout: 5 * time.Minute,
	}
}

// SetDefaultTimeout sets the default timeout for plugin execution
func (pe *PluginExecutor) SetDefaultTimeout(timeout time.Duration) {
	pe.defaultTimeout = timeout
}

// ExecutePlugin executes a plugin based on the instruction
func (pe *PluginExecutor) ExecutePlugin(ctx context.Context, inst *types.Instruction) (*types.ExecutionResult, error) {
	startTime := time.Now()
	
	pe.logger.Info("Starting plugin execution",
		zap.String("instruction_id", inst.ID),
		zap.String("plugin_id", inst.PluginID))

	// Get plugin installation path
	pluginDir := filepath.Join(pe.baseDir, inst.PluginID)
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return &types.ExecutionResult{
			Success:   false,
			PluginID:  inst.PluginID,
			Error:     fmt.Sprintf("plugin not installed: %s", inst.PluginID),
			Duration:  time.Since(startTime).Seconds(),
			Timestamp: time.Now(),
		}, fmt.Errorf("plugin not installed: %s", inst.PluginID)
	}

	// Extract execution configuration
	config, err := pe.extractExecutionConfig(inst, pluginDir)
	if err != nil {
		return &types.ExecutionResult{
			Success:   false,
			PluginID:  inst.PluginID,
			Error:     fmt.Sprintf("failed to extract execution config: %v", err),
			Duration:  time.Since(startTime).Seconds(),
			Timestamp: time.Now(),
		}, err
	}

	// Create execution context with timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = time.Duration(inst.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = pe.defaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute the plugin
	result, err := pe.executeWithRuntime(execCtx, config, pluginDir)
	if err != nil {
		pe.logger.Error("Plugin execution failed",
			zap.String("instruction_id", inst.ID),
			zap.String("plugin_id", inst.PluginID),
			zap.Error(err))
		
		return &types.ExecutionResult{
			Success:   false,
			PluginID:  inst.PluginID,
			Error:     err.Error(),
			Logs:      result.Logs,
			Duration:  time.Since(startTime).Seconds(),
			ExitCode:  result.ExitCode,
			Timestamp: time.Now(),
		}, err
	}

	result.PluginID = inst.PluginID
	result.Duration = time.Since(startTime).Seconds()
	result.Timestamp = time.Now()

	pe.logger.Info("Plugin execution completed",
		zap.String("instruction_id", inst.ID),
		zap.String("plugin_id", inst.PluginID),
		zap.Bool("success", result.Success),
		zap.Float64("duration_seconds", result.Duration))

	return result, nil
}

// extractExecutionConfig extracts execution configuration from instruction
func (pe *PluginExecutor) extractExecutionConfig(inst *types.Instruction, pluginDir string) (*ExecutionConfig, error) {
	config := &ExecutionConfig{
		WorkingDirectory: pluginDir,
		Environment:      make(map[string]string),
		InputData:        inst.InputData,
		Context:          inst.Context,
		Variables:        inst.Variables,
	}

	// Extract entrypoint from plugin configuration
	if entrypoint, ok := inst.PluginConfiguration["entrypoint"].(string); ok {
		config.Entrypoint = entrypoint
	} else {
		return nil, fmt.Errorf("entrypoint not specified in plugin configuration")
	}

	// Extract additional configuration
	if args, ok := inst.PluginConfiguration["arguments"].([]interface{}); ok {
		for _, arg := range args {
			if argStr, ok := arg.(string); ok {
				config.Arguments = append(config.Arguments, argStr)
			}
		}
	}

	if env, ok := inst.PluginConfiguration["environment"].(map[string]interface{}); ok {
		for k, v := range env {
			if vStr, ok := v.(string); ok {
				config.Environment[k] = vStr
			}
		}
	}

	if timeoutSec, ok := inst.PluginConfiguration["timeout_seconds"].(float64); ok {
		config.Timeout = time.Duration(timeoutSec) * time.Second
	}

	return config, nil
}

// executeWithRuntime executes the plugin based on detected runtime
func (pe *PluginExecutor) executeWithRuntime(ctx context.Context, config *ExecutionConfig, pluginDir string) (*types.ExecutionResult, error) {
	runtime := pe.detectRuntime(config.Entrypoint, pluginDir)
	
	pe.logger.Debug("Detected plugin runtime",
		zap.String("runtime", string(runtime)),
		zap.String("entrypoint", config.Entrypoint))

	switch runtime {
	case RuntimePython:
		return pe.executePython(ctx, config, pluginDir)
	case RuntimeNode:
		return pe.executeNode(ctx, config, pluginDir)
	case RuntimeBash:
		return pe.executeBash(ctx, config, pluginDir)
	case RuntimeDocker:
		return pe.executeDocker(ctx, config, pluginDir)
	case RuntimeExecutable:
		return pe.executeExecutable(ctx, config, pluginDir)
	default:
		return pe.executeGeneric(ctx, config, pluginDir)
	}
}

// detectRuntime detects the runtime based on entrypoint and available files
func (pe *PluginExecutor) detectRuntime(entrypoint, pluginDir string) Runtime {
	entrypointPath := filepath.Join(pluginDir, entrypoint)
	
	// Check for specific file extensions
	ext := strings.ToLower(filepath.Ext(entrypoint))
	switch ext {
	case ".py":
		return RuntimePython
	case ".js", ".mjs":
		return RuntimeNode
	case ".sh":
		return RuntimeBash
	}

	// Check for Docker
	if entrypoint == "Dockerfile" || entrypoint == "docker" {
		return RuntimeDocker
	}

	// Check if file is executable
	if info, err := os.Stat(entrypointPath); err == nil {
		if info.Mode()&0111 != 0 {
			return RuntimeExecutable
		}
	}

	// Check for runtime-specific files in directory
	if pe.fileExists(filepath.Join(pluginDir, "requirements.txt")) || 
	   pe.fileExists(filepath.Join(pluginDir, "setup.py")) ||
	   pe.fileExists(filepath.Join(pluginDir, "pyproject.toml")) {
		return RuntimePython
	}

	if pe.fileExists(filepath.Join(pluginDir, "package.json")) {
		return RuntimeNode
	}

	if pe.fileExists(filepath.Join(pluginDir, "go.mod")) {
		return RuntimeGo
	}

	if pe.fileExists(filepath.Join(pluginDir, "Dockerfile")) {
		return RuntimeDocker
	}

	return RuntimeExecutable
}

// executePython executes a Python plugin
func (pe *PluginExecutor) executePython(ctx context.Context, config *ExecutionConfig, pluginDir string) (*types.ExecutionResult, error) {
	var logs []string
	
	// Prepare input data as JSON file if needed
	inputFile, err := pe.prepareInputFile(config, pluginDir)
	if err != nil {
		return &types.ExecutionResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to prepare input file: %v", err),
			Logs:      logs,
			Timestamp: time.Now(),
		}, err
	}
	defer pe.cleanupInputFile(inputFile)

	// Build Python command
	args := []string{config.Entrypoint}
	args = append(args, config.Arguments...)
	if inputFile != "" {
		args = append(args, "--input", inputFile)
	}

	cmd := exec.CommandContext(ctx, "python3", args...)
	cmd.Dir = config.WorkingDirectory
	cmd.Env = pe.buildEnvironment(config.Environment)

	pe.logger.Debug("Executing Python plugin",
		zap.Strings("args", args),
		zap.String("working_dir", cmd.Dir))

	output, err := cmd.CombinedOutput()
	logs = append(logs, string(output))

	result := &types.ExecutionResult{
		Success:   err == nil,
		Logs:      logs,
		ExitCode:  cmd.ProcessState.ExitCode(),
		Timestamp: time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// Try to parse output as JSON for structured results
	if output := strings.TrimSpace(string(output)); output != "" {
		var outputData map[string]interface{}
		if json.Unmarshal([]byte(output), &outputData) == nil {
			result.OutputData = outputData
		} else {
			result.OutputData = map[string]interface{}{"raw_output": output}
		}
	}

	return result, nil
}

// executeNode executes a Node.js plugin
func (pe *PluginExecutor) executeNode(ctx context.Context, config *ExecutionConfig, pluginDir string) (*types.ExecutionResult, error) {
	var logs []string
	
	inputFile, err := pe.prepareInputFile(config, pluginDir)
	if err != nil {
		return &types.ExecutionResult{
			Success:   false,
			Error:     fmt.Sprintf("failed to prepare input file: %v", err),
			Logs:      logs,
			Timestamp: time.Now(),
		}, err
	}
	defer pe.cleanupInputFile(inputFile)

	args := []string{config.Entrypoint}
	args = append(args, config.Arguments...)
	if inputFile != "" {
		args = append(args, "--input", inputFile)
	}

	cmd := exec.CommandContext(ctx, "node", args...)
	cmd.Dir = config.WorkingDirectory
	cmd.Env = pe.buildEnvironment(config.Environment)

	output, err := cmd.CombinedOutput()
	logs = append(logs, string(output))

	result := &types.ExecutionResult{
		Success:   err == nil,
		Logs:      logs,
		ExitCode:  cmd.ProcessState.ExitCode(),
		Timestamp: time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	if output := strings.TrimSpace(string(output)); output != "" {
		var outputData map[string]interface{}
		if json.Unmarshal([]byte(output), &outputData) == nil {
			result.OutputData = outputData
		} else {
			result.OutputData = map[string]interface{}{"raw_output": output}
		}
	}

	return result, nil
}

// executeBash executes a Bash script plugin
func (pe *PluginExecutor) executeBash(ctx context.Context, config *ExecutionConfig, pluginDir string) (*types.ExecutionResult, error) {
	var logs []string
	
	args := []string{config.Entrypoint}
	args = append(args, config.Arguments...)

	cmd := exec.CommandContext(ctx, "bash", args...)
	cmd.Dir = config.WorkingDirectory
	cmd.Env = pe.buildEnvironment(config.Environment)

	output, err := cmd.CombinedOutput()
	logs = append(logs, string(output))

	result := &types.ExecutionResult{
		Success:    err == nil,
		Logs:       logs,
		ExitCode:   cmd.ProcessState.ExitCode(),
		OutputData: map[string]interface{}{"raw_output": string(output)},
		Timestamp:  time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result, err
}

// executeExecutable executes a binary/executable plugin
func (pe *PluginExecutor) executeExecutable(ctx context.Context, config *ExecutionConfig, pluginDir string) (*types.ExecutionResult, error) {
	var logs []string
	
	entrypointPath := filepath.Join(pluginDir, config.Entrypoint)
	args := config.Arguments

	cmd := exec.CommandContext(ctx, entrypointPath, args...)
	cmd.Dir = config.WorkingDirectory
	cmd.Env = pe.buildEnvironment(config.Environment)

	output, err := cmd.CombinedOutput()
	logs = append(logs, string(output))

	result := &types.ExecutionResult{
		Success:    err == nil,
		Logs:       logs,
		ExitCode:   cmd.ProcessState.ExitCode(),
		OutputData: map[string]interface{}{"raw_output": string(output)},
		Timestamp:  time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result, err
}

// executeDocker executes a Docker-based plugin
func (pe *PluginExecutor) executeDocker(ctx context.Context, config *ExecutionConfig, pluginDir string) (*types.ExecutionResult, error) {
	// This is a simplified Docker execution - can be enhanced based on requirements
	var logs []string
	
	// Build Docker image
	imageName := fmt.Sprintf("stavily-plugin-%s", filepath.Base(pluginDir))
	buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", imageName, ".")
	buildCmd.Dir = pluginDir
	
	buildOutput, err := buildCmd.CombinedOutput()
	logs = append(logs, fmt.Sprintf("Docker build: %s", string(buildOutput)))
	
	if err != nil {
		return &types.ExecutionResult{
			Success:   false,
			Error:     fmt.Sprintf("docker build failed: %v", err),
			Logs:      logs,
			Timestamp: time.Now(),
		}, err
	}

	// Run Docker container
	runArgs := []string{"run", "--rm"}
	
	// Add environment variables
	for k, v := range config.Environment {
		runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	
	runArgs = append(runArgs, imageName)
	runArgs = append(runArgs, config.Arguments...)

	runCmd := exec.CommandContext(ctx, "docker", runArgs...)
	output, err := runCmd.CombinedOutput()
	logs = append(logs, string(output))

	result := &types.ExecutionResult{
		Success:    err == nil,
		Logs:       logs,
		ExitCode:   runCmd.ProcessState.ExitCode(),
		OutputData: map[string]interface{}{"raw_output": string(output)},
		Timestamp:  time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result, err
}

// executeGeneric executes a plugin using generic approach
func (pe *PluginExecutor) executeGeneric(ctx context.Context, config *ExecutionConfig, pluginDir string) (*types.ExecutionResult, error) {
	return pe.executeExecutable(ctx, config, pluginDir)
}

// prepareInputFile creates a temporary JSON file with input data
func (pe *PluginExecutor) prepareInputFile(config *ExecutionConfig, pluginDir string) (string, error) {
	if len(config.InputData) == 0 && len(config.Context) == 0 && len(config.Variables) == 0 {
		return "", nil
	}

	inputData := map[string]interface{}{
		"input_data": config.InputData,
		"context":    config.Context,
		"variables":  config.Variables,
	}

	data, err := json.Marshal(inputData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal input data: %v", err)
	}

	inputFile := filepath.Join(pluginDir, "input.json")
	if err := os.WriteFile(inputFile, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write input file: %v", err)
	}

	return inputFile, nil
}

// cleanupInputFile removes temporary input file
func (pe *PluginExecutor) cleanupInputFile(inputFile string) {
	if inputFile != "" {
		os.Remove(inputFile)
	}
}

// buildEnvironment builds environment variables for execution
func (pe *PluginExecutor) buildEnvironment(envVars map[string]string) []string {
	env := os.Environ()
	
	for k, v := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	
	return env
}

// fileExists checks if a file exists
func (pe *PluginExecutor) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
} 