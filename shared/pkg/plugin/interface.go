// Package plugin provides the plugin interface and management for Stavily agents
package plugin

import (
	"context"
	"time"
)

// Plugin represents the base interface that all plugins must implement
type Plugin interface {
	// GetInfo returns plugin metadata
	GetInfo() *Info

	// Initialize initializes the plugin with configuration
	Initialize(ctx context.Context, config map[string]interface{}) error

	// Start starts the plugin execution
	Start(ctx context.Context) error

	// Stop stops the plugin execution
	Stop(ctx context.Context) error

	// GetStatus returns the current plugin status
	GetStatus() Status

	// GetHealth returns the plugin health information
	GetHealth() *Health
}

// TriggerPlugin represents a plugin that detects triggers (for sensor agents)
type TriggerPlugin interface {
	Plugin

	// DetectTriggers detects and returns trigger events
	DetectTriggers(ctx context.Context) (<-chan *TriggerEvent, error)

	// GetTriggerConfig returns the trigger configuration schema
	GetTriggerConfig() *TriggerConfig
}

// ActionPlugin represents a plugin that executes actions (for action agents)
type ActionPlugin interface {
	Plugin

	// ExecuteAction executes an action with the given parameters
	ExecuteAction(ctx context.Context, action *ActionRequest) (*ActionResult, error)

	// GetActionConfig returns the action configuration schema
	GetActionConfig() *ActionConfig
}

// OutputPlugin represents a plugin that handles outputs (for action agents)
type OutputPlugin interface {
	Plugin

	// SendOutput sends output data to the configured destination
	SendOutput(ctx context.Context, output *OutputData) error

	// GetOutputConfig returns the output configuration schema
	GetOutputConfig() *OutputConfig
}

// Info contains plugin metadata
type Info struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Author      string            `json:"author"`
	License     string            `json:"license"`
	Homepage    string            `json:"homepage"`
	Repository  string            `json:"repository"`
	Tags        []string          `json:"tags"`
	Categories  []string          `json:"categories"`
	Type        PluginType        `json:"type"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PluginType represents the type of plugin
type PluginType string

const (
	PluginTypeTrigger PluginType = "trigger"
	PluginTypeAction  PluginType = "action"
	PluginTypeOutput  PluginType = "output"
)

// Status represents the plugin execution status
type Status string

const (
	StatusStopped  Status = "stopped"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusStopping Status = "stopping"
	StatusError    Status = "error"
)

// Health contains plugin health information
type Health struct {
	Status     HealthStatus           `json:"status"`
	Message    string                 `json:"message"`
	LastCheck  time.Time              `json:"last_check"`
	Uptime     time.Duration          `json:"uptime"`
	ErrorCount int                    `json:"error_count"`
	LastError  string                 `json:"last_error,omitempty"`
	Metrics    map[string]interface{} `json:"metrics,omitempty"`
}

// HealthStatus represents the health status of a plugin
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// TriggerEvent represents a detected trigger event
type TriggerEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
	Tags      []string               `json:"tags"`
	Severity  Severity               `json:"severity"`
}

// TriggerConfig defines the configuration schema for trigger plugins
type TriggerConfig struct {
	Schema      map[string]*ConfigField  `json:"schema"`
	Required    []string                 `json:"required"`
	Examples    []map[string]interface{} `json:"examples"`
	Description string                   `json:"description"`
}

// ActionRequest represents an action execution request
type ActionRequest struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Context     map[string]interface{} `json:"context"`
	Timeout     time.Duration          `json:"timeout"`
	Metadata    map[string]interface{} `json:"metadata"`
	RequestedAt time.Time              `json:"requested_at"`
}

// ActionResult represents the result of an action execution
type ActionResult struct {
	ID          string                 `json:"id"`
	Status      ActionStatus           `json:"status"`
	Data        map[string]interface{} `json:"data"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at"`
	Duration    time.Duration          `json:"duration"`
}

// ActionConfig defines the configuration schema for action plugins
type ActionConfig struct {
	Schema      map[string]*ConfigField  `json:"schema"`
	Required    []string                 `json:"required"`
	Examples    []map[string]interface{} `json:"examples"`
	Description string                   `json:"description"`
	Timeout     time.Duration            `json:"timeout"`
}

// OutputData represents data to be sent via an output plugin
type OutputData struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Content     interface{}            `json:"content"`
	Format      string                 `json:"format"`
	Destination string                 `json:"destination"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
}

// OutputConfig defines the configuration schema for output plugins
type OutputConfig struct {
	Schema      map[string]*ConfigField  `json:"schema"`
	Required    []string                 `json:"required"`
	Examples    []map[string]interface{} `json:"examples"`
	Description string                   `json:"description"`
	Formats     []string                 `json:"formats"`
}

// ConfigField defines a configuration field schema
type ConfigField struct {
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Default     interface{}   `json:"default,omitempty"`
	Required    bool          `json:"required"`
	Enum        []string      `json:"enum,omitempty"`
	Pattern     string        `json:"pattern,omitempty"`
	Minimum     *float64      `json:"minimum,omitempty"`
	Maximum     *float64      `json:"maximum,omitempty"`
	MinLength   *int          `json:"min_length,omitempty"`
	MaxLength   *int          `json:"max_length,omitempty"`
	Format      string        `json:"format,omitempty"`
	Examples    []interface{} `json:"examples,omitempty"`
}

// ActionStatus represents the status of an action execution
type ActionStatus string

const (
	ActionStatusPending   ActionStatus = "pending"
	ActionStatusRunning   ActionStatus = "running"
	ActionStatusCompleted ActionStatus = "completed"
	ActionStatusFailed    ActionStatus = "failed"
	ActionStatusCancelled ActionStatus = "cancelled"
	ActionStatusTimeout   ActionStatus = "timeout"
)

// Severity represents the severity level of events
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// PluginRegistry interface for plugin discovery and management
type PluginRegistry interface {
	// RegisterPlugin registers a plugin with the registry
	RegisterPlugin(plugin Plugin) error

	// UnregisterPlugin unregisters a plugin from the registry
	UnregisterPlugin(id string) error

	// GetPlugin retrieves a plugin by ID
	GetPlugin(id string) (Plugin, error)

	// ListPlugins lists all registered plugins
	ListPlugins() []Plugin

	// ListPluginsByType lists plugins of a specific type
	ListPluginsByType(pluginType PluginType) []Plugin

	// GetPluginInfo gets plugin information by ID
	GetPluginInfo(id string) (*Info, error)
}

// PluginLoader interface for loading and unloading plugins
type PluginLoader interface {
	// LoadPlugin loads a plugin from the specified path
	LoadPlugin(ctx context.Context, path string) (Plugin, error)

	// UnloadPlugin unloads a plugin
	UnloadPlugin(ctx context.Context, plugin Plugin) error

	// ReloadPlugin reloads a plugin
	ReloadPlugin(ctx context.Context, plugin Plugin) (Plugin, error)

	// ValidatePlugin validates a plugin before loading
	ValidatePlugin(path string) error
}

// PluginManager interface for comprehensive plugin management
type PluginManager interface {
	PluginRegistry
	PluginLoader

	// StartPlugin starts a plugin
	StartPlugin(ctx context.Context, id string) error

	// StopPlugin stops a plugin
	StopPlugin(ctx context.Context, id string) error

	// RestartPlugin restarts a plugin
	RestartPlugin(ctx context.Context, id string) error

	// GetPluginStatus gets the status of a plugin
	GetPluginStatus(id string) (Status, error)

	// GetPluginHealth gets the health of a plugin
	GetPluginHealth(id string) (*Health, error)

	// UpdatePlugin updates a plugin to a new version
	UpdatePlugin(ctx context.Context, id string, version string) error

	// ConfigurePlugin configures a plugin with new settings
	ConfigurePlugin(ctx context.Context, id string, config map[string]interface{}) error
}

// ExecutionContext provides context for plugin execution
type ExecutionContext struct {
	AgentID     string                 `json:"agent_id"`
	TenantID    string                 `json:"tenant_id"`
	WorkflowID  string                 `json:"workflow_id,omitempty"`
	ExecutionID string                 `json:"execution_id,omitempty"`
	Environment string                 `json:"environment"`
	Variables   map[string]interface{} `json:"variables"`
	Secrets     map[string]string      `json:"secrets"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timeout     time.Duration          `json:"timeout"`
	StartTime   time.Time              `json:"start_time"`
}

// PluginEvent represents events emitted by plugins
type PluginEvent struct {
	PluginID  string                 `json:"plugin_id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Severity  Severity               `json:"severity"`
}

// EventHandler handles plugin events
type EventHandler interface {
	HandleEvent(ctx context.Context, event *PluginEvent) error
}

// SecurityContext provides security constraints for plugin execution
type SecurityContext struct {
	MaxMemory      int64         `json:"max_memory"`
	MaxCPU         float64       `json:"max_cpu"`
	MaxExecTime    time.Duration `json:"max_exec_time"`
	MaxFileSize    int64         `json:"max_file_size"`
	AllowedPaths   []string      `json:"allowed_paths"`
	ForbiddenPaths []string      `json:"forbidden_paths"`
	NetworkAccess  bool          `json:"network_access"`
	Capabilities   []string      `json:"capabilities"`
}
