// Package api provides HTTP client functionality for communicating with the Stavily orchestrator
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stavily/agents/shared/pkg/config"
	"go.uber.org/zap"
)

// OrchestratorClient handles communication with the Stavily Orchestrator API
// following the AGENT_USE.md specification
type OrchestratorClient struct {
	baseURL    string
	agentID    string
	authToken  string
	authMethod string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewOrchestratorClient creates a new orchestrator client
func NewOrchestratorClient(cfg *config.Config, logger *zap.Logger) (*OrchestratorClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	logger.Debug("Creating new orchestrator client",
		zap.String("base_url", cfg.API.BaseURL),
		zap.String("agent_id", cfg.Agent.ID),
		zap.String("auth_method", cfg.Security.Auth.Method))

	// Validate auth method
	if cfg.Security.Auth.Method != "api_key" && cfg.Security.Auth.Method != "jwt" {
		return nil, fmt.Errorf("unsupported authentication method: %s (must be 'api_key' or 'jwt')", cfg.Security.Auth.Method)
	}

	// Initialize auth token based on configuration
	var authToken string

	// If token file is specified, it takes precedence
	if cfg.Security.Auth.TokenFile != "" {
		logger.Debug("Reading token from file", zap.String("token_file", cfg.Security.Auth.TokenFile))
		tokenBytes, err := os.ReadFile(cfg.Security.Auth.TokenFile)
		if err != nil {
			logger.Error("Failed to read token file",
				zap.String("token_file", cfg.Security.Auth.TokenFile),
				zap.Error(err))
			return nil, fmt.Errorf("failed to read token from file: %w", err)
		}
		authToken = string(bytes.TrimSpace(tokenBytes))
		logger.Debug("Token read from file",
			zap.String("token_file", cfg.Security.Auth.TokenFile),
			zap.Int("token_length", len(authToken)))
	} else {
		// Otherwise use the API key/token from config
		logger.Debug("Using token from config")
		authToken = cfg.Security.Auth.APIKey
	}

	// Validate that we have a token
	if authToken == "" {
		logger.Error("No authentication token provided")
		return nil, fmt.Errorf("no authentication token provided: set api_key/token in config or provide valid token_file")
	}

	// Clean the token - remove any "Bearer " prefix if present
	authToken = strings.TrimPrefix(strings.TrimSpace(authToken), "Bearer ")
	
	logger.Debug("Token validation",
		zap.Int("token_length", len(authToken)),
		zap.String("token_prefix", authToken[:min(len(authToken), 10)]+"..."))

	httpClient := &http.Client{
		Timeout: cfg.API.Timeout,
	}

	return &OrchestratorClient{
		baseURL:    cfg.API.BaseURL,
		agentID:    cfg.Agent.ID,
		authToken:  authToken,
		authMethod: cfg.Security.Auth.Method,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// InstructionResponse represents the response from polling for instructions
type InstructionResponse struct {
	Instruction      *Instruction `json:"instruction"`
	Status           string       `json:"status"`
	NextPollInterval int          `json:"next_poll_interval"`
}

// Instruction represents an instruction from the orchestrator
type Instruction struct {
	ID                   string                 `json:"id"`
	PluginID             string                 `json:"plugin_id"`
	PluginConfiguration  map[string]interface{} `json:"plugin_configuration"`
	InputData            map[string]interface{} `json:"input_data"`
	TimeoutSeconds       int                    `json:"timeout_seconds"`
	MaxRetries           int                    `json:"max_retries"`
	CorrelationID        string                 `json:"correlation_id,omitempty"`
}

// InstructionUpdateRequest represents a request to update an instruction
type InstructionUpdateRequest struct {
	Status       string   `json:"status,omitempty"`
	MaxRetries   int      `json:"max_retries,omitempty"`
	ExecutionLog []string `json:"execution_log,omitempty"`
}

// InstructionUpdateResponse represents the response from updating an instruction
type InstructionUpdateResponse struct {
	Success        bool     `json:"success"`
	InstructionID  string   `json:"instruction_id"`
	UpdatedFields  []string `json:"updated_fields"`
}

// InstructionResultRequest represents a request to submit instruction results
type InstructionResultRequest struct {
	Status       string                 `json:"status"`
	Result       map[string]interface{} `json:"result,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	ErrorDetails map[string]interface{} `json:"error_details,omitempty"`
	ExecutionLog []string               `json:"execution_log,omitempty"`
}

// InstructionResultResponse represents the response from submitting results
type InstructionResultResponse struct {
	Acknowledged    bool         `json:"acknowledged"`
	NextInstruction *Instruction `json:"next_instruction"`
}

// PollInstructions polls for the next pending instruction
func (c *OrchestratorClient) PollInstructions(ctx context.Context) (*InstructionResponse, error) {
	url := fmt.Sprintf("%s/agents/v1/%s/instructions", c.baseURL, c.agentID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var instructionResp InstructionResponse
	if err := json.NewDecoder(resp.Body).Decode(&instructionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &instructionResp, nil
}

// UpdateInstruction updates an instruction during execution
func (c *OrchestratorClient) UpdateInstruction(ctx context.Context, instructionID string, update *InstructionUpdateRequest) (*InstructionUpdateResponse, error) {
	url := fmt.Sprintf("%s/agents/v1/%s/instructions/%s", c.baseURL, c.agentID, instructionID)
	
	bodyBytes, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var updateResp InstructionUpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updateResp, nil
}

// SubmitInstructionResult submits the final execution result
func (c *OrchestratorClient) SubmitInstructionResult(ctx context.Context, instructionID string, result *InstructionResultRequest) (*InstructionResultResponse, error) {
	url := fmt.Sprintf("%s/agents/v1/%s/instructions/%s/result", c.baseURL, c.agentID, instructionID)
	
	bodyBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var resultResp InstructionResultResponse
	if err := json.NewDecoder(resp.Body).Decode(&resultResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resultResp, nil
}

// SendHeartbeat sends a heartbeat to the orchestrator. The status parameter
// allows the caller to specify the agent's state (e.g. "online", "offline").
// If an empty string is provided, the status defaults to "online".
func (c *OrchestratorClient) SendHeartbeat(ctx context.Context, status string) error {
	if status == "" {
		status = "online"
	}

	url := fmt.Sprintf("%s/agents/v1/%s/heartbeat", c.baseURL, c.agentID)
	heartbeatData := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"status":    status,
	}

	bodyBytes, err := json.Marshal(heartbeatData)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// setHeaders sets the required headers for API requests
func (c *OrchestratorClient) setHeaders(req *http.Request) {
	c.logger.Debug("Setting request headers",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.String("auth_method", c.authMethod))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	// Set auth header - always use Bearer for both JWT and API key
	authHeader := fmt.Sprintf("Bearer %s", c.authToken)
	req.Header.Set("Authorization", authHeader)
	
	c.logger.Debug("Authorization header set",
		zap.Int("auth_header_length", len(authHeader)),
		zap.String("auth_header_prefix", authHeader[:min(len(authHeader), 15)]+"..."))
	
	req.Header.Set("User-Agent", "Stavily-Agent/1.0.0")
}

// Close closes the orchestrator client
func (c *OrchestratorClient) Close() error {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	return nil
} 