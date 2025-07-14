package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/Stavily/01-Agents/shared/pkg/config"
	"go.uber.org/zap"
)

// AuthManager handles authentication for API requests
type AuthManager struct {
	config config.AuthConfig
	logger *zap.Logger

	// API Key management
	apiKey string
	mu     sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config config.AuthConfig, logger *zap.Logger) (*AuthManager, error) {
	manager := &AuthManager{
		config: config,
		logger: logger,
	}

	switch config.Method {
	case "api_key":
		if err := manager.initAPIKey(); err != nil {
			return nil, fmt.Errorf("failed to initialize API key auth: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", config.Method)
	}

	return manager, nil
}

// initAPIKey initializes API key authentication
func (a *AuthManager) initAPIKey() error {
	// First, check if we have an API key directly in config
	if a.config.APIKey != "" {
		a.mu.Lock()
		a.apiKey = a.config.APIKey
		a.mu.Unlock()
		a.logger.Debug("Using API key from configuration")
		return nil
	}

	// If no direct API key, check if we have a token file to read from
	if a.config.TokenFile != "" {
		a.logger.Debug("Loading API key from file", zap.String("file", a.config.TokenFile))
		token, err := os.ReadFile(a.config.TokenFile)
		if err != nil {
			return fmt.Errorf("failed to read API key file: %w", err)
		}

		// Trim whitespace and set as API key
		apiKey := strings.TrimSpace(string(token))
		if apiKey == "" {
			return fmt.Errorf("API key file is empty: %s", a.config.TokenFile)
		}

		a.mu.Lock()
		a.apiKey = apiKey
		a.mu.Unlock()

		a.logger.Debug("Successfully loaded API key from file")
		return nil
	}

	return fmt.Errorf("no API key provided: either set api_key in config or provide token_file")
}

// AddAuth adds authentication to an HTTP request
func (a *AuthManager) AddAuth(req *http.Request) error {
	switch a.config.Method {
	case "api_key":
		return a.addAPIKeyAuth(req)
	default:
		return fmt.Errorf("unsupported authentication method: %s", a.config.Method)
	}
}

// addAPIKeyAuth adds API key authentication to the request
func (a *AuthManager) addAPIKeyAuth(req *http.Request) error {
	a.mu.RLock()
	apiKey := a.apiKey
	a.mu.RUnlock()

	if apiKey == "" {
		return fmt.Errorf("no API key available")
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	return nil
}

// GetAPIKey returns the current API key (for debugging/monitoring)
func (a *AuthManager) GetAPIKey() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	// Return masked version for security
	if len(a.apiKey) > 8 {
		return a.apiKey[:4] + "..." + a.apiKey[len(a.apiKey)-4:]
	}
	return "***"
}

// UpdateAPIKey updates the API key (useful for key rotation)
func (a *AuthManager) UpdateAPIKey(newAPIKey string) error {
	if newAPIKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	a.mu.Lock()
	a.apiKey = newAPIKey
	a.mu.Unlock()

	a.logger.Debug("API key updated")
	return nil
}

// Close cleans up the authentication manager
func (a *AuthManager) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Clear sensitive data
	a.apiKey = ""

	return nil
}
