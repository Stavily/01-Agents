package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators registers custom validation rules
func RegisterCustomValidators(v *validator.Validate) error {
	validators := map[string]validator.Func{
		"file_exists":  validateFileExists,
		"dir_exists":   validateDirExists,
		"agent_id":     validateAgentID,
		"tenant_id":    validateTenantID,
		"duration_min": validateDurationMin,
		"duration_max": validateDurationMax,
		"memory_size":  validateMemorySize,
		"file_size":    validateFileSize,
		"port_range":   validatePortRange,
		"url_scheme":   validateURLScheme,
	}

	for tag, fn := range validators {
		if err := v.RegisterValidation(tag, fn); err != nil {
			return fmt.Errorf("failed to register validator %s: %w", tag, err)
		}
	}

	return nil
}

// validateFileExists checks if a file exists
func validateFileExists(fl validator.FieldLevel) bool {
	filename := fl.Field().String()
	if filename == "" {
		return true // Allow empty files, other validators handle required
	}

	_, err := os.Stat(filename)
	return err == nil
}

// validateDirExists checks if a directory exists
func validateDirExists(fl validator.FieldLevel) bool {
	dirname := fl.Field().String()
	if dirname == "" {
		return true // Allow empty directories, other validators handle required
	}

	info, err := os.Stat(dirname)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// validateAgentID validates agent ID format
func validateAgentID(fl validator.FieldLevel) bool {
	agentID := fl.Field().String()
	if agentID == "" {
		return false
	}

	// Agent ID must be alphanumeric with hyphens and underscores, 3-64 characters
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9_-]{2,63}$`, agentID)
	return matched
}

// validateTenantID validates tenant ID format
func validateTenantID(fl validator.FieldLevel) bool {
	tenantID := fl.Field().String()
	if tenantID == "" {
		return false
	}

	// Tenant ID must be alphanumeric with hyphens, 3-64 characters
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9-]{2,63}$`, tenantID)
	return matched
}

// validateDurationMin validates minimum duration
func validateDurationMin(fl validator.FieldLevel) bool {
	duration := fl.Field().Interface().(time.Duration)
	param := fl.Param()

	minDuration, err := time.ParseDuration(param)
	if err != nil {
		return false
	}

	return duration >= minDuration
}

// validateDurationMax validates maximum duration
func validateDurationMax(fl validator.FieldLevel) bool {
	duration := fl.Field().Interface().(time.Duration)
	param := fl.Param()

	maxDuration, err := time.ParseDuration(param)
	if err != nil {
		return false
	}

	return duration <= maxDuration
}

// validateMemorySize validates memory size in bytes
func validateMemorySize(fl validator.FieldLevel) bool {
	size := fl.Field().Int()

	// Minimum 1MB, maximum 16GB
	const minSize = 1024 * 1024             // 1MB
	const maxSize = 16 * 1024 * 1024 * 1024 // 16GB

	return size >= minSize && size <= maxSize
}

// validateFileSize validates file size in bytes
func validateFileSize(fl validator.FieldLevel) bool {
	size := fl.Field().Int()

	// Minimum 1KB, maximum 1GB
	const minSize = 1024               // 1KB
	const maxSize = 1024 * 1024 * 1024 // 1GB

	return size >= minSize && size <= maxSize
}

// validatePortRange validates port number range
func validatePortRange(fl validator.FieldLevel) bool {
	port := fl.Field().Int()

	// Valid port range: 1024-65535 (avoid privileged ports)
	return port >= 1024 && port <= 65535
}

// validateURLScheme validates URL scheme
func validateURLScheme(fl validator.FieldLevel) bool {
	urlStr := fl.Field().String()
	if urlStr == "" {
		return true // Allow empty URLs, other validators handle required
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Only allow HTTPS for security
	return u.Scheme == "https"
}

// ValidateConfigPaths validates that all required paths exist and are accessible
func ValidateConfigPaths(config *Config) error {
	var errors []string

	// Validate TLS certificate files
	if config.Security.TLS.Enabled {
		if config.Security.TLS.CertFile != "" {
			if err := validateFilePath(config.Security.TLS.CertFile, "TLS certificate"); err != nil {
				errors = append(errors, err.Error())
			}
		}

		if config.Security.TLS.KeyFile != "" {
			if err := validateFilePath(config.Security.TLS.KeyFile, "TLS key"); err != nil {
				errors = append(errors, err.Error())
			}
		}

		if config.Security.TLS.CAFile != "" {
			if err := validateFilePath(config.Security.TLS.CAFile, "TLS CA"); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}

	// Validate auth token file
	if config.Security.Auth.TokenFile != "" {
		if err := validateFilePath(config.Security.Auth.TokenFile, "auth token"); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate JWT secret file
	if config.Security.Auth.JWT.SecretFile != "" {
		if err := validateFilePath(config.Security.Auth.JWT.SecretFile, "JWT secret"); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate plugin directory
	if config.Plugins.Directory != "" {
		if err := validateDirPath(config.Plugins.Directory, "plugin"); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate log file directory if specified
	if config.Logging.Output == "file" && config.Logging.File != "" {
		logDir := filepath.Dir(config.Logging.File)
		if err := validateDirPath(logDir, "log file"); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate audit log file directory if specified
	if config.Security.Audit.Enabled && config.Security.Audit.LogFile != "" {
		auditDir := filepath.Dir(config.Security.Audit.LogFile)
		if err := validateDirPath(auditDir, "audit log file"); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration path validation failed: %v", errors)
	}

	return nil
}

// validateFilePath validates that a file path exists and is readable
func validateFilePath(path, description string) error {
	if path == "" {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s file does not exist: %s", description, path)
		}
		return fmt.Errorf("cannot access %s file: %s (%v)", description, path, err)
	}

	if info.IsDir() {
		return fmt.Errorf("%s path is a directory, not a file: %s", description, path)
	}

	// Check if file is readable
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%s file is not readable: %s (%v)", description, path, err)
	}
	file.Close()

	return nil
}

// validateDirPath validates that a directory path exists and is accessible
func validateDirPath(path, description string) error {
	if path == "" {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s directory does not exist: %s", description, path)
		}
		return fmt.Errorf("cannot access %s directory: %s (%v)", description, path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s path is not a directory: %s", description, path)
	}

	// Check if directory is readable and writable
	testFile := filepath.Join(path, "stavily-test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("%s directory is not writable: %s (%v)", description, path, err)
	}
	file.Close()
	os.Remove(testFile)

	return nil
}

// ValidateAgentConfig validates agent-specific configuration requirements
func ValidateAgentConfig(config *Config) error {
	var errors []string

	// Validate agent type specific requirements
	switch config.Agent.Type {
	case "sensor":
		// Sensor agents should have read-only permissions
		if len(config.Security.Sandbox.AllowedPaths) == 0 {
			errors = append(errors, "sensor agents must specify allowed paths for monitoring")
		}

	case "action":
		// Action agents need execution capabilities
		if config.Security.Sandbox.Enabled && config.Security.Sandbox.MaxExecTime == 0 {
			errors = append(errors, "action agents must specify maximum execution time")
		}

	default:
		errors = append(errors, fmt.Sprintf("invalid agent type: %s", config.Agent.Type))
	}

	// Validate tenant and agent ID combination
	if config.Agent.TenantID == config.Agent.ID {
		errors = append(errors, "agent ID cannot be the same as tenant ID")
	}

	// Validate environment-specific settings
	switch config.Agent.Environment {
	case "prod":
		// Production environment must have security enabled
		if !config.Security.TLS.Enabled {
			errors = append(errors, "TLS must be enabled in production environment")
		}
		if !config.Security.Audit.Enabled {
			errors = append(errors, "audit logging must be enabled in production environment")
		}
		if config.Logging.Level == "debug" {
			errors = append(errors, "debug logging should not be used in production environment")
		}

	case "dev":
		// Development environment warnings (not errors)
		// TLS is optional in dev environment
	}

	if len(errors) > 0 {
		return fmt.Errorf("agent configuration validation failed: %v", errors)
	}

	return nil
}
