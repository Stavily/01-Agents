// Package api provides types for API communication with the Stavily orchestrator
package api

import (
	"time"
)

// PollRequest represents a request to poll for pending tasks
type PollRequest struct {
	AgentID      string    `json:"agent_id"`
	TenantID     string    `json:"tenant_id"`
	AgentType    string    `json:"agent_type"` // "sensor" or "action"
	Environment  string    `json:"environment"`
	MaxTasks     int       `json:"max_tasks"`
	Capabilities []string  `json:"capabilities,omitempty"`
	LastPollTime time.Time `json:"last_poll_time,omitempty"`
}

// PollResponse represents the response from a poll request
type PollResponse struct {
	Tasks       []*Task            `json:"tasks"`
	NextPollIn  time.Duration      `json:"next_poll_in,omitempty"`
	ServerTime  time.Time          `json:"server_time"`
	AgentConfig *AgentConfigUpdate `json:"agent_config,omitempty"`
}

// Task represents an action task from the orchestrator
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	WorkflowID  string                 `json:"workflow_id"`
	TenantID    string                 `json:"tenant_id"`
	Parameters  map[string]interface{} `json:"parameters"`
	Context     map[string]interface{} `json:"context"`
	Timeout     time.Duration          `json:"timeout"`
	Priority    int                    `json:"priority"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	CreatedAt   time.Time              `json:"created_at"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID      string                 `json:"task_id"`
	AgentID     string                 `json:"agent_id"`
	Status      string                 `json:"status"` // "completed", "failed", "timeout"
	Data        map[string]interface{} `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AgentConfigUpdate represents configuration updates from the orchestrator
type AgentConfigUpdate struct {
	PollInterval       time.Duration   `json:"poll_interval,omitempty"`
	MaxConcurrentTasks int             `json:"max_concurrent_tasks,omitempty"`
	LogLevel           string          `json:"log_level,omitempty"`
	PluginUpdates      []*PluginUpdate `json:"plugin_updates,omitempty"`
}

// PluginUpdate represents a plugin update instruction
type PluginUpdate struct {
	PluginID string                 `json:"plugin_id"`
	Action   string                 `json:"action"` // "install", "update", "remove", "enable", "disable"
	Version  string                 `json:"version,omitempty"`
	URL      string                 `json:"url,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// TriggerEvent represents a detected trigger event (for sensor agents)
type TriggerEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
	Tags      []string               `json:"tags"`
	Severity  string                 `json:"severity"`
	AgentID   string                 `json:"agent_id"`
	TenantID  string                 `json:"tenant_id"`
}

// ReportTriggerRequest represents a request to report trigger events
type ReportTriggerRequest struct {
	AgentID string          `json:"agent_id"`
	Events  []*TriggerEvent `json:"events"`
}

// ReportTriggerResponse represents the response to a trigger report
type ReportTriggerResponse struct {
	ProcessedEvents int       `json:"processed_events"`
	FailedEvents    []string  `json:"failed_events,omitempty"`
	ServerTime      time.Time `json:"server_time"`
}

// AgentStatusReport represents an agent status report
type AgentStatusReport struct {
	AgentID     string                 `json:"agent_id"`
	TenantID    string                 `json:"tenant_id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"` // "online", "offline", "degraded"
	Version     string                 `json:"version"`
	StartTime   time.Time              `json:"start_time"`
	LastSeen    time.Time              `json:"last_seen"`
	Environment string                 `json:"environment"`
	Metrics     *AgentMetrics          `json:"metrics,omitempty"`
	Health      *AgentHealthReport     `json:"health,omitempty"`
	Plugins     []*PluginStatusReport  `json:"plugins,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AgentMetrics represents agent performance metrics
type AgentMetrics struct {
	CPUUsage     float64       `json:"cpu_usage"`
	MemoryUsage  int64         `json:"memory_usage"`
	DiskUsage    int64         `json:"disk_usage"`
	NetworkIn    int64         `json:"network_in"`
	NetworkOut   int64         `json:"network_out"`
	TasksTotal   int           `json:"tasks_total,omitempty"`
	TasksSuccess int           `json:"tasks_success,omitempty"`
	TasksFailed  int           `json:"tasks_failed,omitempty"`
	Uptime       time.Duration `json:"uptime"`
	Timestamp    time.Time     `json:"timestamp"`
}

// AgentHealthReport represents agent health information
type AgentHealthReport struct {
	Status     string                      `json:"status"` // "healthy", "degraded", "unhealthy"
	Message    string                      `json:"message,omitempty"`
	LastCheck  time.Time                   `json:"last_check"`
	Components map[string]*ComponentHealth `json:"components"`
	Checks     []*HealthCheckResult        `json:"checks,omitempty"`
}

// ComponentHealth represents the health of a component
type ComponentHealth struct {
	Status     string                 `json:"status"`
	Message    string                 `json:"message,omitempty"`
	LastCheck  time.Time              `json:"last_check"`
	ErrorCount int                    `json:"error_count"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// PluginStatusReport represents the status of a plugin
type PluginStatusReport struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Version    string                 `json:"version"`
	Type       string                 `json:"type"`
	Status     string                 `json:"status"` // "loaded", "running", "stopped", "error"
	Health     string                 `json:"health"` // "healthy", "degraded", "unhealthy"
	StartTime  time.Time              `json:"start_time,omitempty"`
	LastError  string                 `json:"last_error,omitempty"`
	ErrorCount int                    `json:"error_count"`
	Metrics    map[string]interface{} `json:"metrics,omitempty"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error     string      `json:"error"`
	Message   string      `json:"message"`
	Code      string      `json:"code,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}
