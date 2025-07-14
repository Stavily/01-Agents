// Package types provides shared types for the plugin system
package types

import (
	"time"
)

// InstructionType represents the type of instruction
type InstructionType string

const (
	InstructionTypeManual       InstructionType = "manual"
	InstructionTypeWorkflow     InstructionType = "workflow"
	InstructionTypeScheduled    InstructionType = "scheduled"
	InstructionTypeAPI          InstructionType = "api"
	InstructionTypePluginInstall InstructionType = "plugin_install"
	InstructionTypePluginUpdate InstructionType = "plugin_update"
	InstructionTypeExecute      InstructionType = "execute"
)

// InstructionStatus represents the status of an instruction
type InstructionStatus string

const (
	InstructionStatusPending   InstructionStatus = "pending"
	InstructionStatusDelivered InstructionStatus = "delivered"
	InstructionStatusExecuting InstructionStatus = "executing"
	InstructionStatusCompleted InstructionStatus = "completed"
	InstructionStatusFailed    InstructionStatus = "failed"
	InstructionStatusTimeout   InstructionStatus = "timeout"
	InstructionStatusCancelled InstructionStatus = "cancelled"
)

// InstructionSource represents the source of an instruction
type InstructionSource string

const (
	InstructionSourceWebUI         InstructionSource = "web-ui"
	InstructionSourceAPI           InstructionSource = "api"
	InstructionSourceWorkflowEngine InstructionSource = "workflow-engine"
	InstructionSourceCLI           InstructionSource = "cli"
)

// Priority represents instruction priority
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// Instruction represents a complete instruction from the database
type Instruction struct {
	ID                  string                 `json:"id"`
	AgentID             string                 `json:"agent_id"`
	PluginID            string                 `json:"plugin_id"`
	CreatedBy           *string                `json:"created_by"`
	Status              InstructionStatus      `json:"status"`
	Priority            Priority               `json:"priority"`
	Type                InstructionType        `json:"instruction_type"`
	Source              InstructionSource      `json:"source"`
	PluginConfiguration map[string]interface{} `json:"plugin_configuration"`
	InputData           map[string]interface{} `json:"input_data"`
	Context             map[string]interface{} `json:"context"`
	Variables           map[string]interface{} `json:"variables"`
	TimeoutSeconds      int                    `json:"timeout_seconds"`
	MaxRetries          int                    `json:"max_retries"`
	RetryCount          int                    `json:"retry_count"`
	RetryPolicy         map[string]interface{} `json:"retry_policy"`
	ScheduledAt         *time.Time             `json:"scheduled_at"`
	CompletedAt         *string                `json:"completed_at"`
	ExecutionLog        []interface{}          `json:"execution_log"`
	CorrelationID       *string                `json:"correlation_id"`
	WorkflowExecutionID *string                `json:"workflow_execution_id"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// PollResponse represents the response from polling for instructions
type PollResponse struct {
	Instruction      *Instruction `json:"instruction"`
	Status           string       `json:"status"`
	NextPollInterval int          `json:"next_poll_interval"`
}

// InstallationResult represents the result of a plugin installation
type InstallationResult struct {
	PluginID      string    `json:"plugin_id"`
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"`
	InstalledPath string    `json:"installed_path"`
	Version       string    `json:"version"`
	Logs          []string  `json:"logs"`
	Duration      float64   `json:"duration_seconds"`
	Timestamp     time.Time `json:"timestamp"`
}

// ExecutionResult represents the result of a plugin execution
type ExecutionResult struct {
	PluginID   string                 `json:"plugin_id"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	OutputData map[string]interface{} `json:"output_data"`
	Logs       []string               `json:"logs"`
	Duration   float64                `json:"duration_seconds"`
	Timestamp  time.Time              `json:"timestamp"`
	ExitCode   int                    `json:"exit_code"`
}

// InstructionResult represents the result of processing an instruction
type InstructionResult struct {
	InstructionID    string               `json:"instruction_id"`
	Type             InstructionType      `json:"instruction_type"`
	Success          bool                 `json:"success"`
	Error            string               `json:"error,omitempty"`
	InstallResult    *InstallationResult  `json:"install_result,omitempty"`
	ExecutionResult  *ExecutionResult     `json:"execution_result,omitempty"`
	ProcessingLogs   []string             `json:"processing_logs"`
	StartTime        time.Time            `json:"start_time"`
	EndTime          time.Time            `json:"end_time"`
	Duration         float64              `json:"duration_seconds"`
} 