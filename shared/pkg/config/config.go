// Package config provides configuration management for Stavily agents
package config

import (
	"fmt"
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

	// Action agent specific fields
	PollInterval       time.Duration `mapstructure:"poll_interval" validate:"min=5s,max=300s"`
	MaxConcurrentTasks int           `mapstructure:"max_concurrent_tasks" validate:"min=1,max=100"`
	TaskTimeout        time.Duration `mapstructure:"task_timeout" validate:"min=10s,max=3600s"`
}

// APIConfig contains orchestrator API configuration
type APIConfig struct {
	BaseURL         string            `mapstructure:"base_url" validate:"required,url"`
	AgentsEndpoint  string            `mapstructure:"agents_endpoint" validate:"required"`
	Timeout         time.Duration     `mapstructure:"timeout" validate:"min=5s,max=300s"`
	RetryAttempts   int               `mapstructure:"retry_attempts" validate:"min=1,max=10"`
	RetryDelay      time.Duration     `mapstructure:"retry_delay" validate:"min=1s,max=60s"`
	RateLimitRPS    int               `mapstructure:"rate_limit_rps" validate:"min=1,max=1000"`
	Headers         map[string]string `mapstructure:"headers"`
	UserAgent       string            `mapstructure:"user_agent"`
	MaxIdleConns    int               `mapstructure:"max_idle_conns" validate:"min=1,max=100"`
	IdleConnTimeout time.Duration     `mapstructure:"idle_conn_timeout" validate:"min=30s,max=300s"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	TLS     TLSConfig     `mapstructure:"tls" validate:"required"`
	Auth    AuthConfig    `mapstructure:"auth" validate:"required"`
	Sandbox SandboxConfig `mapstructure:"sandbox"`
	Audit   AuditConfig   `mapstructure:"audit"`
}

// TLSConfig contains TLS configuration
type TLSConfig struct {
	Enabled            bool     `mapstructure:"enabled"`
	CertFile           string   `mapstructure:"cert_file" validate:"required_if=Enabled true"`
	KeyFile            string   `mapstructure:"key_file" validate:"required_if=Enabled true"`
	CAFile             string   `mapstructure:"ca_file" validate:"required_if=Enabled true"`
	ServerName         string   `mapstructure:"server_name"`
	InsecureSkipVerify bool     `mapstructure:"insecure_skip_verify"`
	MinVersion         string   `mapstructure:"min_version" validate:"oneof=1.2 1.3"`
	CipherSuites       []string `mapstructure:"cipher_suites"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Method    string        `mapstructure:"method" validate:"required,oneof=jwt certificate"`
	JWT       JWTConfig     `mapstructure:"jwt"`
	TokenFile string        `mapstructure:"token_file"`
	TokenTTL  time.Duration `mapstructure:"token_ttl" validate:"min=1m,max=24h"`
}

// JWTConfig contains JWT-specific configuration
type JWTConfig struct {
	SecretFile   string        `mapstructure:"secret_file"`
	Algorithm    string        `mapstructure:"algorithm" validate:"oneof=HS256 HS384 HS512 RS256 RS384 RS512"`
	Issuer       string        `mapstructure:"issuer"`
	Audience     string        `mapstructure:"audience"`
	RefreshToken bool          `mapstructure:"refresh_token"`
	RefreshTTL   time.Duration `mapstructure:"refresh_ttl"`
}

// SandboxConfig contains plugin sandbox configuration
type SandboxConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	MaxMemory      int64         `mapstructure:"max_memory" validate:"min=1048576"`  // 1MB minimum
	MaxCPU         float64       `mapstructure:"max_cpu" validate:"min=0.1,max=8.0"` // CPU cores
	MaxExecTime    time.Duration `mapstructure:"max_exec_time" validate:"min=1s,max=300s"`
	MaxFileSize    int64         `mapstructure:"max_file_size" validate:"min=1024"` // 1KB minimum
	AllowedPaths   []string      `mapstructure:"allowed_paths"`
	ForbiddenPaths []string      `mapstructure:"forbidden_paths"`
	NetworkAccess  bool          `mapstructure:"network_access"`
}

// AuditConfig contains audit logging configuration
type AuditConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	LogFile    string   `mapstructure:"log_file"`
	MaxSize    int      `mapstructure:"max_size" validate:"min=1,max=1000"` // MB
	MaxBackups int      `mapstructure:"max_backups" validate:"min=1,max=100"`
	MaxAge     int      `mapstructure:"max_age" validate:"min=1,max=365"` // days
	Events     []string `mapstructure:"events"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level" validate:"oneof=debug info warn error fatal"`
	Format     string `mapstructure:"format" validate:"oneof=json text"`
	Output     string `mapstructure:"output" validate:"oneof=stdout stderr file"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size" validate:"min=1,max=1000"` // MB
	MaxBackups int    `mapstructure:"max_backups" validate:"min=1,max=100"`
	MaxAge     int    `mapstructure:"max_age" validate:"min=1,max=365"` // days
	Compress   bool   `mapstructure:"compress"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled   bool              `mapstructure:"enabled"`
	Port      int               `mapstructure:"port" validate:"min=1024,max=65535"`
	Path      string            `mapstructure:"path"`
	Namespace string            `mapstructure:"namespace"`
	Subsystem string            `mapstructure:"subsystem"`
	Labels    map[string]string `mapstructure:"labels"`
}

// PluginConfig contains plugin system configuration
type PluginConfig struct {
	Directory     string         `mapstructure:"directory" validate:"required"`
	AutoLoad      bool           `mapstructure:"auto_load"`
	WatchChanges  bool           `mapstructure:"watch_changes"`
	UpdateCheck   time.Duration  `mapstructure:"update_check" validate:"min=1m,max=24h"`
	Timeout       time.Duration  `mapstructure:"timeout" validate:"min=1s,max=300s"`
	MaxConcurrent int            `mapstructure:"max_concurrent" validate:"min=1,max=100"`
	Registry      RegistryConfig `mapstructure:"registry"`
}

// RegistryConfig contains plugin registry configuration
type RegistryConfig struct {
	URL      string            `mapstructure:"url" validate:"url"`
	Auth     bool              `mapstructure:"auth"`
	Headers  map[string]string `mapstructure:"headers"`
	CacheDir string            `mapstructure:"cache_dir"`
	CacheTTL time.Duration     `mapstructure:"cache_ttl" validate:"min=1m,max=24h"`
}

// HealthConfig contains health check configuration
type HealthConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Port     int           `mapstructure:"port" validate:"min=1024,max=65535"`
	Path     string        `mapstructure:"path"`
	Interval time.Duration `mapstructure:"interval" validate:"min=10s,max=300s"`
	Timeout  time.Duration `mapstructure:"timeout" validate:"min=1s,max=60s"`
	Checks   []HealthCheck `mapstructure:"checks"`
}

// HealthCheck represents a single health check
type HealthCheck struct {
	Name     string        `mapstructure:"name" validate:"required"`
	Type     string        `mapstructure:"type" validate:"required,oneof=tcp http command"`
	Target   string        `mapstructure:"target" validate:"required"`
	Timeout  time.Duration `mapstructure:"timeout" validate:"min=1s,max=60s"`
	Interval time.Duration `mapstructure:"interval" validate:"min=10s,max=300s"`
	Retries  int           `mapstructure:"retries" validate:"min=0,max=10"`
}

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	// Set defaults
	setDefaults()

	// Configure viper
	viper.SetConfigFile(configPath)
	viper.SetEnvPrefix("STAVILY")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validate.Struct(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Agent defaults
	viper.SetDefault("agent.heartbeat", "30s")
	viper.SetDefault("agent.environment", "dev")

	// API defaults
	viper.SetDefault("api.timeout", "30s")
	viper.SetDefault("api.retry_attempts", 3)
	viper.SetDefault("api.retry_delay", "5s")
	viper.SetDefault("api.rate_limit_rps", 10)
	viper.SetDefault("api.user_agent", "Stavily-Agent/1.0")
	viper.SetDefault("api.max_idle_conns", 10)
	viper.SetDefault("api.idle_conn_timeout", "90s")
	viper.SetDefault("api.agents_endpoint", "/api/v1/agents")

	// Security defaults
	viper.SetDefault("security.tls.enabled", true)
	viper.SetDefault("security.tls.min_version", "1.3")
	viper.SetDefault("security.auth.method", "jwt")
	viper.SetDefault("security.auth.token_ttl", "1h")
	viper.SetDefault("security.auth.jwt.algorithm", "HS256")
	viper.SetDefault("security.sandbox.enabled", true)
	viper.SetDefault("security.sandbox.max_memory", 134217728) // 128MB
	viper.SetDefault("security.sandbox.max_cpu", 1.0)
	viper.SetDefault("security.sandbox.max_exec_time", "60s")
	viper.SetDefault("security.sandbox.max_file_size", 10485760) // 10MB
	viper.SetDefault("security.sandbox.network_access", false)
	viper.SetDefault("security.audit.enabled", true)
	viper.SetDefault("security.audit.max_size", 100)
	viper.SetDefault("security.audit.max_backups", 10)
	viper.SetDefault("security.audit.max_age", 30)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 10)
	viper.SetDefault("logging.max_age", 30)
	viper.SetDefault("logging.compress", true)

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 9090)
	viper.SetDefault("metrics.path", "/metrics")
	viper.SetDefault("metrics.namespace", "stavily")

	// Plugin defaults
	viper.SetDefault("plugins.directory", "/opt/stavily/plugins")
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
