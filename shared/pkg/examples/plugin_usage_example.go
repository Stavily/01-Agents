// Package examples provides usage examples for the plugin system
package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Stavily/01-Agents/shared/pkg/agent"
	"github.com/Stavily/01-Agents/shared/pkg/config"
	"github.com/Stavily/01-Agents/shared/pkg/types"
	"go.uber.org/zap"
)

// PluginUsageExample demonstrates how to use the plugin download and execution system
func PluginUsageExample() {
	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	// Create enhanced plugin manager configuration
	cfg := &agent.EnhancedPluginConfig{
		PluginConfig: &config.PluginConfig{
			Enabled: true,
		},
		PluginBaseDir: "./plugins",
		GitTimeout:    5 * time.Minute,
		ExecTimeout:   10 * time.Minute,
	}

	// Create enhanced plugin manager
	pluginManager, err := agent.NewEnhancedPluginManager(cfg, logger)
	if err != nil {
		log.Fatal("Failed to create enhanced plugin manager:", err)
	}

	// Initialize the manager
	ctx := context.Background()
	if err := pluginManager.Initialize(ctx); err != nil {
		log.Fatal("Failed to initialize plugin manager:", err)
	}
	defer pluginManager.Shutdown(ctx)

	// Example 1: Process plugin installation instruction from polling
	fmt.Println("=== Example 1: Plugin Installation via Polling ===")
	installPollResponse := &types.PollResponse{
		Instruction: &types.Instruction{
			ID:              "0aeb8570-9b0e-4dd5-b567-76261e29f0d7",
			AgentID:         "65bb7a51-1812-4a37-90d5-3e8a859ba972",
			PluginID:        "example-python-plugin",
					Status:   types.InstructionStatusPending,
		Priority: types.PriorityHigh,
		Type:     types.InstructionTypePluginInstall,
		Source:   types.InstructionSourceWebUI,
			PluginConfiguration: map[string]interface{}{
				"plugin_url": "https://github.com/stavily/06-plugins",
				"entrypoint": "main.py",
			},
			InputData:       map[string]interface{}{},
			Context:         map[string]interface{}{},
			Variables:       map[string]interface{}{},
			TimeoutSeconds:  300,
			MaxRetries:      3,
			RetryCount:      0,
			RetryPolicy:     map[string]interface{}{},
			ScheduledAt:     timePtr(time.Now()),
			ExecutionLog:    []interface{}{},
			CorrelationID:   nil,
			WorkflowExecutionID: nil,
			Metadata:        map[string]interface{}{},
		},
		Status:           "instruction_delivered",
		NextPollInterval: 5,
	}

	installResult, err := pluginManager.ProcessInstruction(ctx, installPollResponse)
	if err != nil {
		logger.Error("Plugin installation failed", zap.Error(err))
	} else {
		logger.Info("Plugin installation completed",
			zap.String("status", string(installResult.Status)),
			zap.Duration("duration", installResult.Duration))
		
		// Print installation result
		resultJSON, _ := json.MarshalIndent(installResult, "", "  ")
		fmt.Printf("Installation Result:\n%s\n\n", string(resultJSON))
	}

	// Example 2: Process plugin execution instruction from polling
	fmt.Println("=== Example 2: Plugin Execution via Polling ===")
	execPollResponse := &types.PollResponse{
		Instruction: &types.Instruction{
			ID:              "26a71c76-69d9-4923-a86b-783d9796fb17",
			AgentID:         "65bb7a51-1812-4a37-90d5-3e8a859ba972",
			PluginID:        "example-python-plugin",
					Status:   types.InstructionStatusPending,
		Priority: types.PriorityHigh,
		Type:     types.InstructionTypeExecute,
		Source:   types.InstructionSourceWebUI,
			PluginConfiguration: map[string]interface{}{
				"entrypoint": "main.py",
				"arguments":  []string{"--verbose"},
				"environment": map[string]interface{}{
					"PLUGIN_ENV": "production",
				},
			},
			InputData: map[string]interface{}{
				"message": "Hello from Stavily!",
				"count":   42,
			},
			Context: map[string]interface{}{
				"execution_id": "exec-123",
				"workflow_id":  "wf-456",
			},
			Variables: map[string]interface{}{
				"api_key": "secret-key-123",
			},
			TimeoutSeconds:  300,
			MaxRetries:      3,
			RetryCount:      0,
			RetryPolicy:     map[string]interface{}{},
			ScheduledAt:     timePtr(time.Now()),
			CompletedAt:     nil,
			ExecutionLog:    []interface{}{},
			CorrelationID:   nil,
			WorkflowExecutionID: nil,
			Metadata:        map[string]interface{}{},
		},
		Status:           "instruction_delivered",
		NextPollInterval: 5,
	}

	execResult, err := pluginManager.ProcessInstruction(ctx, execPollResponse)
	if err != nil {
		logger.Error("Plugin execution failed", zap.Error(err))
	} else {
		logger.Info("Plugin execution completed",
			zap.String("status", string(execResult.Status)),
			zap.Duration("duration", execResult.Duration))
		
		// Print execution result
		resultJSON, _ := json.MarshalIndent(execResult, "", "  ")
		fmt.Printf("Execution Result:\n%s\n\n", string(resultJSON))
	}

	// Example 3: Direct plugin installation (programmatic)
	fmt.Println("=== Example 3: Direct Plugin Installation ===")
	directInstallResult, err := pluginManager.InstallPlugin(
		ctx,
		"direct-install-plugin",
		"https://github.com/stavily/06-plugins",
		"main",
	)
	if err != nil {
		logger.Error("Direct plugin installation failed", zap.Error(err))
	} else {
		logger.Info("Direct plugin installation completed",
			zap.Bool("success", directInstallResult.Success),
			zap.String("path", directInstallResult.InstallationPath))
	}

	// Example 4: Direct plugin execution (programmatic)
	fmt.Println("=== Example 4: Direct Plugin Execution ===")
	if pluginManager.IsPluginInstalled("example-python-plugin") {
		directExecResult, err := pluginManager.ExecutePlugin(
			ctx,
			"example-python-plugin",
			"main.py",
			map[string]interface{}{
				"task":   "process_data",
				"data":   []int{1, 2, 3, 4, 5},
				"output": "json",
			},
		)
		if err != nil {
			logger.Error("Direct plugin execution failed", zap.Error(err))
		} else {
			logger.Info("Direct plugin execution completed",
				zap.Bool("success", directExecResult.Success),
				zap.Int("exit_code", directExecResult.ExitCode))
		}
	}

	// Example 5: Check plugin manager status
	fmt.Println("=== Example 5: Plugin Manager Status ===")
	status := pluginManager.GetEnhancedStatus()
	statusJSON, _ := json.MarshalIndent(status, "", "  ")
	fmt.Printf("Enhanced Plugin Manager Status:\n%s\n\n", string(statusJSON))

	// Example 6: Demonstration of polling loop
	fmt.Println("=== Example 6: Simulated Polling Loop ===")
	demonstratePollingLoop(pluginManager, logger)
}

// demonstratePollingLoop shows how an agent would use the plugin manager in a polling loop
func demonstratePollingLoop(pluginManager *agent.EnhancedPluginManager, logger *zap.Logger) {
	ctx := context.Background()
	
	// Simulate different poll responses
	pollResponses := []*types.PollResponse{
		// No instruction
		{
			Instruction:      nil,
			Status:           "no_pending_instructions",
			NextPollInterval: 10,
		},
		// Plugin install instruction
		{
			Instruction: &instruction.Instruction{
				ID:              "install-001",
				AgentID:         "agent-001",
				PluginID:        "monitoring-plugin",
				InstructionType: instruction.InstructionTypePluginInstall,
				PluginConfiguration: map[string]interface{}{
					"plugin_url": "https://github.com/stavily/06-plugins",
					"entrypoint": "monitor.py",
				},
				TimeoutSeconds: 300,
			},
			Status:           "instruction_delivered",
			NextPollInterval: 5,
		},
		// Plugin execute instruction
		{
			Instruction: &instruction.Instruction{
				ID:              "exec-001",
				AgentID:         "agent-001",
				PluginID:        "monitoring-plugin",
				InstructionType: instruction.InstructionTypeExecute,
				PluginConfiguration: map[string]interface{}{
					"entrypoint": "monitor.py",
				},
				InputData: map[string]interface{}{
					"target": "localhost",
					"port":   8080,
				},
				TimeoutSeconds: 60,
			},
			Status:           "instruction_delivered",
			NextPollInterval: 5,
		},
	}

	for i, response := range pollResponses {
		fmt.Printf("--- Poll Iteration %d ---\n", i+1)
		
		if response.Instruction == nil {
			logger.Info("No instructions pending")
			fmt.Printf("Next poll in %d seconds\n", response.NextPollInterval)
		} else {
			logger.Info("Processing instruction",
				zap.String("instruction_id", response.Instruction.ID),
				zap.String("type", string(response.Instruction.InstructionType)))

			result, err := pluginManager.ProcessInstruction(ctx, response)
			if err != nil {
				logger.Error("Instruction processing failed", zap.Error(err))
			} else if result != nil {
				logger.Info("Instruction processed successfully",
					zap.String("instruction_id", result.InstructionID),
					zap.String("status", string(result.Status)))
			}
		}
		
		fmt.Println()
		
		// Simulate polling interval
		time.Sleep(time.Duration(response.NextPollInterval) * time.Second)
	}
}

// timePtr returns a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}

// ExamplePollResponseStructures shows the expected poll response structures
func ExamplePollResponseStructures() {
	fmt.Println("=== Expected Poll Response Structures ===")

	// No pending instructions
	noPending := &types.PollResponse{
		Instruction:      nil,
		Status:           "no_pending_instructions",
		NextPollInterval: 10,
	}
	
	noPendingJSON, _ := json.MarshalIndent(noPending, "", "  ")
	fmt.Printf("No Pending Instructions:\n%s\n\n", string(noPendingJSON))

	// Plugin installation instruction
	installInstruction := &types.PollResponse{
		Instruction: &instruction.Instruction{
			ID:              "0aeb8570-9b0e-4dd5-b567-76261e29f0d7",
			AgentID:         "65bb7a51-1812-4a37-90d5-3e8a859ba972",
			PluginID:        "e56434ca-9756-446e-92cd-f7545cb5b7b2",
			CreatedBy:       nil,
			Status:          instruction.InstructionStatusPending,
			Priority:        instruction.PriorityHigh,
			InstructionType: instruction.InstructionTypePluginInstall,
			Source:          instruction.InstructionSourceWebUI,
			PluginConfiguration: map[string]interface{}{
				"plugin_url": "https://github.com/stavily/06-plugins",
				"entrypoint": "main.py",
			},
			InputData:       map[string]interface{}{},
			Context:         map[string]interface{}{},
			Variables:       map[string]interface{}{},
			TimeoutSeconds:  300,
			MaxRetries:      3,
			RetryCount:      0,
			RetryPolicy:     map[string]interface{}{},
			ScheduledAt:     timePtr(time.Now()),
			CompletedAt:     nil,
			ExecutionLog:    []interface{}{},
			CorrelationID:   nil,
			WorkflowExecutionID: nil,
			Metadata:        map[string]interface{}{},
		},
		Status:           "instruction_delivered",
		NextPollInterval: 5,
	}

	installJSON, _ := json.MarshalIndent(installInstruction, "", "  ")
	fmt.Printf("Plugin Install Instruction:\n%s\n\n", string(installJSON))

	// Plugin execution instruction
	executeInstruction := &types.PollResponse{
		Instruction: &instruction.Instruction{
			ID:              "26a71c76-69d9-4923-a86b-783d9796fb17",
			AgentID:         "65bb7a51-1812-4a37-90d5-3e8a859ba972",
			PluginID:        "8024d189-0016-48cc-9f01-16d54210ac59",
			CreatedBy:       nil,
			Status:          instruction.InstructionStatusPending,
			Priority:        instruction.PriorityHigh,
			InstructionType: instruction.InstructionTypeExecute,
			Source:          instruction.InstructionSourceWebUI,
			PluginConfiguration: map[string]interface{}{
				"entrypoint": "main.py",
			},
			InputData:       map[string]interface{}{},
			Context:         map[string]interface{}{},
			Variables:       map[string]interface{}{},
			TimeoutSeconds:  300,
			MaxRetries:      3,
			RetryCount:      0,
			RetryPolicy:     map[string]interface{}{},
			ScheduledAt:     timePtr(time.Now()),
			CompletedAt:     stringPtr(""),
			ExecutionLog:    []interface{}{},
			CorrelationID:   nil,
			WorkflowExecutionID: nil,
			Metadata:        map[string]interface{}{},
		},
		Status:           "instruction_delivered",
		NextPollInterval: 5,
	}

	executeJSON, _ := json.MarshalIndent(executeInstruction, "", "  ")
	fmt.Printf("Plugin Execute Instruction:\n%s\n\n", string(executeJSON))
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
} 