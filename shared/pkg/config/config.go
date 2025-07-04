// Package config provides configuration management for Stavily agents
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config represents the base configuration for all agents
type Config struct {
	// Agent identification
	Agent AgentConfig `mapstructure:"agent" validate:"required"`

	// API configuration for communicating with orchestrator
	API APIConfig `mapstructure:"api" validate:"required"`

	// Security configuration
	Security SecurityConfig `mapstructure:"security" validate:"required"`

	// Logging configuration
	Logging LoggingConfig `mapstructure:"logging"`

	// Metrics configuration
	Metrics MetricsConfig `mapstructure:"metrics"`

	// Plugin configuration
	Plugins PluginConfig `mapstructure:"plugins"`

	// Health check configuration
	Health HealthConfig `mapstructure:"health"`
}

// AgentConfig contains agent-specific configuration
type AgentConfig struct {
	ID          string        `mapstructure:"id" validate:"required,min=1"`
	Name        string        `mapstructure:"name" validate:"required,min=1"`
	Type        string        `mapstructure:"type" validate:"required,oneof=sensor action"`
	TenantID    string        `mapstructure:"tenant_id" validate:"required,min=1"`
	Environment string        `mapstructure:"environment" validate:"required,oneof=dev staging prod"`
	Version     string        `mapstructure:"version"`
	Region      string        `mapstructure:"region"`
	Tags        []string      `mapstructure:"tags"`
	Heartbeat   time.Duration `mapstructure:"heartbeat" validate:"min=10s,max=300s"`
	
	// Base folder for agent data (logs, plugins, etc.)
	BaseFolder string `mapstructure:"base_folder" validate:"required,min=1"`

	// Action agent specific fields
	PollInterval       time.Duration `mapstructure:"poll_interval" validate:"min=5s,max=300s"`
	MaxConcurrentTasks int           `mapstructure:"max_concurrent_tasks" validate:"min=1,max=100"`
	TaskTimeout        time.Duration `mapstructure:"task_timeout" validate:"min=10s,max=3600s"`
}

// APIConfig contains orchestrator API configuration
type APIConfig struct {
	BaseURL          string            `mapstructure:"base_url" validate:"required,url"`
	AgentsEndpoint   string            `mapstructure:"agents_endpoint"`
	Timeout          time.Duration     `mapstructure:"timeout" validate:"min=5s,max=300s"`
	RetryAttempts    int               `mapstructure:"retry_attempts" validate:"min=1,max=10"`
	RetryDelay       time.Duration     `mapstructure:"retry_delay" validate:"min=1s,max=60s"`
	RateLimitRPS     int               `mapstructure:"rate_limit_rps" validate:"min=1,max=1000"`
	MaxIdleConns     int               `mapstructure:"max_idle_conns" validate:"min=1,max=100"`
	IdleConnTimeout  time.Duration     `mapstructure:"idle_conn_timeout" validate:"min=30s,max=300s"`
	UserAgent        string            `mapstructure:"user_agent"`
	Headers          map[string]string `mapstructure:"headers"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	TLS     TLSConfig     `mapstructure:"tls"`
	Auth    AuthConfig    `mapstructure:"auth"`
	Sandbox SandboxConfig `mapstructure:"sandbox"`
	Audit   AuditConfig   `mapstructure:"audit"`
}

// TLSConfig contains TLS configuration
type TLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	CertFile           string `mapstructure:"cert_file" validate:"omitempty,file_exists"`
	KeyFile            string `mapstructure:"key_file" validate:"omitempty,file_exists"`
	CAFile             string `mapstructure:"ca_file" validate:"omitempty,file_exists"`
	ServerName         string `mapstructure:"server_name"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	MinVersion         string `mapstructure:"min_version" validate:"oneof=1.2 1.3"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Method    string        `mapstructure:"method" validate:"required,oneof=api_key"`
	TokenFile string        `mapstructure:"token_file" validate:"omitempty,file_exists"`
	APIKey    string        `mapstructure:"api_key"`
	TokenTTL  time.Duration `mapstructure:"token_ttl"`
}

// SandboxConfig contains sandbox configuration
type SandboxConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	MaxMemory     int64         `mapstructure:"max_memory" validate:"min=1048576"`      // 1MB minimum
	MaxCPU        float64       `mapstructure:"max_cpu" validate:"min=0.1,max=8"`       // 0.1 to 8 cores
	MaxExecTime   time.Duration `mapstructure:"max_exec_time" validate:"min=1s,max=3600s"`
	MaxFileSize   int64         `mapstructure:"max_file_size" validate:"min=1024"`      // 1KB minimum
	AllowedPaths  []string      `mapstructure:"allowed_paths"`
	NetworkAccess bool          `mapstructure:"network_access"`
}

// AuditConfig contains audit logging configuration
type AuditConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	LogFile    string `mapstructure:"log_file"`
	MaxSize    int    `mapstructure:"max_size" validate:"min=1,max=1000"`    // MB
	MaxBackups int    `mapstructure:"max_backups" validate:"min=1,max=100"`
	MaxAge     int    `mapstructure:"max_age" validate:"min=1,max=365"`      // days
	Compress   bool   `mapstructure:"compress"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level" validate:"oneof=debug info warn error"`
	Format     string `mapstructure:"format" validate:"oneof=json text"`
	Output     string `mapstructure:"output" validate:"oneof=stdout stderr file"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size" validate:"min=1,max=1000"`    // MB
	MaxBackups int    `mapstructure:"max_backups" validate:"min=1,max=100"`
	MaxAge     int    `mapstructure:"max_age" validate:"min=1,max=365"`      // days
	Compress   bool   `mapstructure:"compress"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Port      int    `mapstructure:"port" validate:"port_range"`
	Path      string `mapstructure:"path"`
	Namespace string `mapstructure:"namespace"`
}

// PluginConfig contains plugin configuration
type PluginConfig struct {
	Directory     string            `mapstructure:"directory" validate:"required,dir_exists"`
	AutoLoad      bool              `mapstructure:"auto_load"`
	WatchChanges  bool              `mapstructure:"watch_changes"`
	UpdateCheck   time.Duration     `mapstructure:"update_check"`
	Timeout       time.Duration     `mapstructure:"timeout" validate:"min=1s,max=300s"`
	MaxConcurrent int               `mapstructure:"max_concurrent" validate:"min=1,max=100"`
	Registry      PluginRegistryConfig `mapstructure:"registry"`
}

// PluginRegistryConfig contains plugin registry configuration
type PluginRegistryConfig struct {
	URL      string        `mapstructure:"url" validate:"omitempty,url"`
	Auth     bool          `mapstructure:"auth"`
	CacheDir string        `mapstructure:"cache_dir"`
	CacheTTL time.Duration `mapstructure:"cache_ttl"`
}

// HealthConfig contains health check configuration
type HealthConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Port     int           `mapstructure:"port" validate:"port_range"`
	Path     string        `mapstructure:"path"`
	Interval time.Duration `mapstructure:"interval" validate:"min=10s,max=300s"`
	Timeout  time.Duration `mapstructure:"timeout" validate:"min=1s,max=60s"`
}

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()
	viper.SetEnvPrefix("STAVILY")

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Expand base folder paths
	if err := cfg.expandBaseFolderPaths(); err != nil {
		return nil, fmt.Errorf("failed to expand base folder paths: %w", err)
	}

	return &cfg, nil
}

// expandBaseFolderPaths expands relative paths based on the base folder
func (c *Config) expandBaseFolderPaths() error {
	if c.Agent.BaseFolder == "" {
		return fmt.Errorf("base_folder is required")
	}

	// Create complete agent directory structure
	if err := c.createAgentDirectoryStructure(); err != nil {
		return fmt.Errorf("failed to create agent directory structure: %w", err)
	}

	// Expand logging file path
	if c.Logging.File != "" && !filepath.IsAbs(c.Logging.File) {
		c.Logging.File = filepath.Join(c.Agent.BaseFolder, "logs", c.Logging.File)
	}

	// Expand plugin directory path
	if c.Plugins.Directory != "" && !filepath.IsAbs(c.Plugins.Directory) {
		c.Plugins.Directory = filepath.Join(c.Agent.BaseFolder, "data", "plugins")
	}

	// Expand plugin cache directory
	if c.Plugins.Registry.CacheDir != "" && !filepath.IsAbs(c.Plugins.Registry.CacheDir) {
		c.Plugins.Registry.CacheDir = filepath.Join(c.Agent.BaseFolder, "data", "cache", "plugins")
	}

	// Expand audit log file path
	if c.Security.Audit.LogFile != "" && !filepath.IsAbs(c.Security.Audit.LogFile) {
		c.Security.Audit.LogFile = filepath.Join(c.Agent.BaseFolder, "logs", "audit", c.Security.Audit.LogFile)
	}

	// Expand auth token file path
	if c.Security.Auth.TokenFile != "" && !filepath.IsAbs(c.Security.Auth.TokenFile) {
		c.Security.Auth.TokenFile = filepath.Join(c.Agent.BaseFolder, "config", "certificates", c.Security.Auth.TokenFile)
	}

	return nil
}

// createAgentDirectoryStructure creates the complete directory structure for the agent
func (c *Config) createAgentDirectoryStructure() error {
	baseDir := c.Agent.BaseFolder
	
	// Define the complete directory structure as documented in README.md
	dirs := []string{
		baseDir,                                    // Base directory
		filepath.Join(baseDir, "config"),          // Configuration directory
		filepath.Join(baseDir, "config", "plugins"), // Plugin configurations
		filepath.Join(baseDir, "config", "certificates"), // TLS certificates
		filepath.Join(baseDir, "data"),            // Data directory
		filepath.Join(baseDir, "data", "plugins"), // Plugin binaries and data
		filepath.Join(baseDir, "data", "cache"),   // Temporary cache files
		filepath.Join(baseDir, "data", "state"),   // Agent state files
		filepath.Join(baseDir, "logs"),            // Logs directory
		filepath.Join(baseDir, "logs", "plugins"), // Plugin logs
		filepath.Join(baseDir, "logs", "audit"),   // Audit logs
		filepath.Join(baseDir, "tmp"),             // Temporary files
		filepath.Join(baseDir, "tmp", "workdir"),  // Work directory for actions
	}
	
	// Create all directories with appropriate permissions
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	return nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Agent defaults
	viper.SetDefault("agent.heartbeat", "30s")
	viper.SetDefault("agent.poll_interval", "30s")
	viper.SetDefault("agent.max_concurrent_tasks", 10)
	viper.SetDefault("agent.task_timeout", "300s")
	viper.SetDefault("agent.base_folder", "./agent-data")

	// API defaults
	viper.SetDefault("api.agents_endpoint", "/api/v1/agents")
	viper.SetDefault("api.timeout", "30s")
	viper.SetDefault("api.retry_attempts", 3)
	viper.SetDefault("api.retry_delay", "5s")
	viper.SetDefault("api.rate_limit_rps", 10)
	viper.SetDefault("api.max_idle_conns", 10)
	viper.SetDefault("api.idle_conn_timeout", "90s")
	viper.SetDefault("api.user_agent", "Stavily-Agent/1.0.0")

	// Security defaults
	viper.SetDefault("security.tls.enabled", false)
	viper.SetDefault("security.tls.min_version", "1.3")
	viper.SetDefault("security.auth.method", "api_key")
	viper.SetDefault("security.auth.token_ttl", "1h")
	viper.SetDefault("security.sandbox.enabled", true)
	viper.SetDefault("security.sandbox.max_memory", 134217728) // 128MB
	viper.SetDefault("security.sandbox.max_cpu", 0.5)
	viper.SetDefault("security.sandbox.max_exec_time", "30s")
	viper.SetDefault("security.sandbox.max_file_size", 10485760) // 10MB
	viper.SetDefault("security.sandbox.network_access", false)
	viper.SetDefault("security.audit.enabled", true)
	viper.SetDefault("security.audit.max_size", 100)
	viper.SetDefault("security.audit.max_backups", 10)
	viper.SetDefault("security.audit.max_age", 30)
	viper.SetDefault("security.audit.compress", true)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "file")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 5)
	viper.SetDefault("logging.max_age", 30)
	viper.SetDefault("logging.compress", true)

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 9090)
	viper.SetDefault("metrics.path", "/metrics")
	viper.SetDefault("metrics.namespace", "stavily")

	// Plugin defaults
	viper.SetDefault("plugins.directory", "data/plugins")
	viper.SetDefault("plugins.auto_load", true)
	viper.SetDefault("plugins.watch_changes", true)
	viper.SetDefault("plugins.update_check", "1h")
	viper.SetDefault("plugins.timeout", "30s")
	viper.SetDefault("plugins.max_concurrent", 10)
	viper.SetDefault("plugins.registry.cache_ttl", "1h")

	// Health defaults
	viper.SetDefault("health.enabled", true)
	viper.SetDefault("health.port", 8080)
	viper.SetDefault("health.path", "/health")
	viper.SetDefault("health.interval", "30s")
	viper.SetDefault("health.timeout", "10s")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	return validate.Struct(c)
}

// GetAgentType returns the agent type
func (c *Config) GetAgentType() string {
	return c.Agent.Type
}

// IsSensorAgent returns true if this is a sensor agent
func (c *Config) IsSensorAgent() bool {
	return c.Agent.Type == "sensor"
}

// IsActionAgent returns true if this is an action agent
func (c *Config) IsActionAgent() bool {
	return c.Agent.Type == "action"
}

// GetFullAgentID returns the full agent identifier
func (c *Config) GetFullAgentID() string {
	return fmt.Sprintf("%s-%s-%s", c.Agent.TenantID, c.Agent.Type, c.Agent.ID)
}

// GetLogDir returns the log directory path
func (c *Config) GetLogDir() string {
	return filepath.Join(c.Agent.BaseFolder, "logs")
}

// GetPluginDir returns the plugin directory path
func (c *Config) GetPluginDir() string {
	return c.Plugins.Directory
}

// GetCacheDir returns the cache directory path
func (c *Config) GetCacheDir() string {
	return filepath.Join(c.Agent.BaseFolder, "data", "cache")
}

// GetDataDir returns the data directory path
func (c *Config) GetDataDir() string {
	return filepath.Join(c.Agent.BaseFolder, "data")
}

// GetStateDir returns the state directory path  
func (c *Config) GetStateDir() string {
	return filepath.Join(c.Agent.BaseFolder, "data", "state")
}

// GetTmpDir returns the temporary directory path
func (c *Config) GetTmpDir() string {
	return filepath.Join(c.Agent.BaseFolder, "tmp")
}

// GetWorkDir returns the work directory path (for action agents)
func (c *Config) GetWorkDir() string {
	return filepath.Join(c.Agent.BaseFolder, "tmp", "workdir")
}

// GetConfigDir returns the config directory path
func (c *Config) GetConfigDir() string {
	return filepath.Join(c.Agent.BaseFolder, "config")
}
