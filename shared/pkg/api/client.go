// Package api provides HTTP client functionality for communicating with the Stavily orchestrator
package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/Stavily/01-Agents/shared/pkg/config"
	"go.uber.org/zap"
)

// Client represents the API client for communicating with the orchestrator
type Client struct {
	baseURL    string
	httpClient *http.Client
	config     *config.APIConfig
	auth       *AuthManager
	logger     *zap.Logger

	// Rate limiting
	rateLimiter *RateLimiter

	// Connection pooling
	mu sync.RWMutex
}

// NewClient creates a new API client
func NewClient(cfg *config.Config, logger *zap.Logger) (*Client, error) {
	// Create HTTP client with security configuration
	httpClient, err := createHTTPClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create authentication manager
	authManager, err := NewAuthManager(cfg.Security.Auth, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth manager: %w", err)
	}

	// Create rate limiter
	rateLimiter := NewRateLimiter(cfg.API.RateLimitRPS)

	client := &Client{
		baseURL:     cfg.API.BaseURL,
		httpClient:  httpClient,
		config:      &cfg.API,
		auth:        authManager,
		logger:      logger,
		rateLimiter: rateLimiter,
	}

	return client, nil
}

// createHTTPClient creates an HTTP client with the specified security configuration
func createHTTPClient(cfg *config.Config) (*http.Client, error) {
	transport := &http.Transport{
		MaxIdleConns:       cfg.API.MaxIdleConns,
		IdleConnTimeout:    cfg.API.IdleConnTimeout,
		DisableCompression: false,
		ForceAttemptHTTP2:  true,
	}

	// Configure TLS if enabled
	if cfg.Security.TLS.Enabled {
		tlsConfig, err := createTLSConfig(cfg.Security.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		transport.TLSClientConfig = tlsConfig
	}

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.API.Timeout,
	}, nil
}

// createTLSConfig creates a TLS configuration from the security config
func createTLSConfig(tlsConfig config.TLSConfig) (*tls.Config, error) {
	config := &tls.Config{
		ServerName:         tlsConfig.ServerName,
		InsecureSkipVerify: tlsConfig.InsecureSkipVerify,
	}

	// Set minimum TLS version
	switch tlsConfig.MinVersion {
	case "1.2":
		config.MinVersion = tls.VersionTLS12
	case "1.3":
		config.MinVersion = tls.VersionTLS13
	default:
		config.MinVersion = tls.VersionTLS13
	}

	// Load client certificates if specified
	if tlsConfig.CertFile != "" && tlsConfig.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsConfig.CertFile, tlsConfig.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if specified
	if tlsConfig.CAFile != "" {
		caCert, err := os.ReadFile(tlsConfig.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		config.RootCAs = caCertPool
	}

	return config, nil
}

// Request represents an API request
type Request struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    interface{}
	Query   map[string]string
}

// Response represents an API response
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Do executes an API request with retry logic and rate limiting
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	var lastErr error

	// Retry logic
	for attempt := 1; attempt <= c.config.RetryAttempts; attempt++ {
		resp, err := c.doRequest(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on certain errors
		if !isRetryableError(err) {
			break
		}

		// Don't retry on the last attempt
		if attempt == c.config.RetryAttempts {
			break
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(c.config.RetryDelay * time.Duration(attempt)):
			// Continue to next attempt
		}

		c.logger.Debug("Retrying API request",
			zap.Int("attempt", attempt),
			zap.String("method", req.Method),
			zap.String("path", req.Path),
			zap.Error(err))
	}

	return nil, fmt.Errorf("API request failed after %d attempts: %w", c.config.RetryAttempts, lastErr)
}

// doRequest executes a single API request
func (c *Client) doRequest(ctx context.Context, req *Request) (*Response, error) {
	// Build URL
	fullURL, err := c.buildURL(req.Path, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Prepare request body
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	c.setHeaders(httpReq, req.Headers)

	// Add authentication
	if err := c.auth.AddAuth(httpReq); err != nil {
		return nil, fmt.Errorf("failed to add authentication: %w", err)
	}

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	response := &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header,
		Body:       respBody,
	}

	// Check for HTTP errors
	if httpResp.StatusCode >= 400 {
		return response, &HTTPError{
			StatusCode: httpResp.StatusCode,
			Message:    string(respBody),
		}
	}

	return response, nil
}

// buildURL constructs the full URL for the request
func (c *Client) buildURL(path string, query map[string]string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	u.Path = path

	if len(query) > 0 {
		values := url.Values{}
		for k, v := range query {
			values.Set(k, v)
		}
		u.RawQuery = values.Encode()
	}

	return u.String(), nil
}

// setHeaders sets the request headers
func (c *Client) setHeaders(req *http.Request, headers map[string]string) {
	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	// Set configured headers
	for k, v := range c.config.Headers {
		req.Header.Set(k, v)
	}

	// Set request-specific headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

// Get executes a GET request
func (c *Client) Get(ctx context.Context, path string, query map[string]string) (*Response, error) {
	req := &Request{
		Method: http.MethodGet,
		Path:   path,
		Query:  query,
	}
	return c.Do(ctx, req)
}

// Post executes a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}) (*Response, error) {
	req := &Request{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	}
	return c.Do(ctx, req)
}

// Put executes a PUT request
func (c *Client) Put(ctx context.Context, path string, body interface{}) (*Response, error) {
	req := &Request{
		Method: http.MethodPut,
		Path:   path,
		Body:   body,
	}
	return c.Do(ctx, req)
}

// Delete executes a DELETE request
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	req := &Request{
		Method: http.MethodDelete,
		Path:   path,
	}
	return c.Do(ctx, req)
}

// Close closes the API client and cleans up resources
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}

	return nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// IsHTTPError checks if an error is an HTTP error
func IsHTTPError(err error) (*HTTPError, bool) {
	httpErr, ok := err.(*HTTPError)
	return httpErr, ok
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if httpErr, ok := IsHTTPError(err); ok {
		// Retry on server errors and some client errors
		return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429 || httpErr.StatusCode == 408
	}

	// Retry on network errors
	return true
}

// PollForTasks polls the orchestrator for pending tasks
func (c *Client) PollForTasks(ctx context.Context, req *PollRequest) (*PollResponse, error) {
	resp, err := c.Post(ctx, "/api/v1/agents/poll", req)
	if err != nil {
		return nil, fmt.Errorf("failed to poll for tasks: %w", err)
	}

	var pollResp PollResponse
	if err := json.Unmarshal(resp.Body, &pollResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal poll response: %w", err)
	}

	return &pollResp, nil
}

// ReportTaskResult reports the result of a task execution
func (c *Client) ReportTaskResult(ctx context.Context, result *TaskResult) error {
	_, err := c.Post(ctx, "/api/v1/agents/tasks/result", result)
	if err != nil {
		return fmt.Errorf("failed to report task result: %w", err)
	}

	return nil
}

// ReportAgentStatus reports the current agent status
func (c *Client) ReportAgentStatus(ctx context.Context, status interface{}) error {
	_, err := c.Post(ctx, "/api/v1/agents/status", status)
	if err != nil {
		return fmt.Errorf("failed to report agent status: %w", err)
	}

	return nil
}
