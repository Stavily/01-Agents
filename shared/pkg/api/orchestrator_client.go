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
	"time"

	"github.com/stavily/agents/shared/pkg/config"
	"go.uber.org/zap"
)

// OrchestratorClient handles communication with the Stavily Orchestrator API
// following the AGENT_USE.md specification
type OrchestratorClient struct {
	baseURL    string
	agentID    string
	apiKey     string
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

	// Load API key from file if specified
	apiKey := ""
	if cfg.Security.Auth.TokenFile != "" {
		tokenBytes, err := os.ReadFile(cfg.Security.Auth.TokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read API key from file: %w", err)
		}
		apiKey = string(bytes.TrimSpace(tokenBytes))
	}

	httpClient := &http.Client{
		Timeout: cfg.API.Timeout,
	}

	return &OrchestratorClient{
		baseURL:    cfg.API.BaseURL,
		agentID:    cfg.Agent.ID,
		apiKey:     apiKey,
		httpClient: httpClient,
		logger:     logger,
	}, nil
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

// SendHeartbeat sends a heartbeat to the orchestrator
func (c *OrchestratorClient) SendHeartbeat(ctx context.Context) error {
	url := fmt.Sprintf("%s/agents/v1/%s/heartbeat", c.baseURL, c.agentID)
	
	heartbeatData := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"status":    "healthy",
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("User-Agent", "Stavily-Agent/1.0.0")
}

// Close closes the orchestrator client
func (c *OrchestratorClient) Close() error {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	return nil
} 