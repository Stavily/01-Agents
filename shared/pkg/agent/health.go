// Package agent provides shared agent components and utilities
package agent

import (
	"context"
	"sync"
	"time"

	"github.com/stavily/agents/shared/pkg/config"
	"github.com/stavily/agents/shared/pkg/plugin"
	"go.uber.org/zap"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ComponentHealth represents the health of a component
type ComponentHealth struct {
	Status     HealthStatus `json:"status"`
	LastCheck  time.Time    `json:"last_check"`
	ErrorCount int          `json:"error_count"`
	Message    string       `json:"message,omitempty"`
}

// HealthStats tracks health check statistics
type HealthStats struct {
	ChecksPassed int       `json:"checks_passed"`
	ChecksFailed int       `json:"checks_failed"`
	LastCheck    time.Time `json:"last_check"`
}

// HealthCheckStatus represents the status of health checks
type HealthCheckStatus struct {
	LastCheck     time.Time     `json:"last_check"`
	CheckInterval time.Duration `json:"check_interval"`
	ChecksPassed  int           `json:"checks_passed"`
	ChecksFailed  int           `json:"checks_failed"`
}

// HealthChecker performs health checks on agent components
type HealthChecker struct {
	cfg       *config.HealthConfig
	logger    *zap.Logger
	stats     *HealthStats
	mu        sync.RWMutex
	
	// Component-specific health checkers
	checkers map[string]func() *ComponentHealth
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(cfg *config.HealthConfig, logger *zap.Logger) (*HealthChecker, error) {
	return &HealthChecker{
		cfg:      cfg,
		logger:   logger,
		stats:    &HealthStats{},
		checkers: make(map[string]func() *ComponentHealth),
	}, nil
}

// RegisterComponent registers a component for health checking
func (hc *HealthChecker) RegisterComponent(name string, checker func() *ComponentHealth) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checkers[name] = checker
}

// Start starts the health checker
func (hc *HealthChecker) Start(ctx context.Context) error {
	hc.logger.Info("Starting health checker")
	
	// Start periodic health checks
	go hc.healthCheckLoop(ctx)
	
	return nil
}

// Stop stops the health checker
func (hc *HealthChecker) Stop(ctx context.Context) error {
	hc.logger.Info("Stopping health checker")
	return nil
}

// GetStatus returns the health checker status
func (hc *HealthChecker) GetStatus() *HealthCheckStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	return &HealthCheckStatus{
		LastCheck:     hc.stats.LastCheck,
		CheckInterval: 30 * time.Second, // Default interval, should come from config
		ChecksPassed:  hc.stats.ChecksPassed,
		ChecksFailed:  hc.stats.ChecksFailed,
	}
}

// GetHealth returns the health checker health
func (hc *HealthChecker) GetHealth() *ComponentHealth {
	return &ComponentHealth{
		Status:     HealthStatusHealthy,
		LastCheck:  time.Now(),
		ErrorCount: 0,
	}
}

// CheckAllComponents performs health checks on all registered components
func (hc *HealthChecker) CheckAllComponents() map[string]*ComponentHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	results := make(map[string]*ComponentHealth)
	
	for name, checker := range hc.checkers {
		results[name] = checker()
	}
	
	return results
}

// healthCheckLoop runs periodic health checks
func (hc *HealthChecker) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Should come from config
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.performHealthCheck()
		}
	}
}

// performHealthCheck performs a single health check cycle
func (hc *HealthChecker) performHealthCheck() {
	hc.mu.Lock()
	hc.stats.LastCheck = time.Now()
	hc.mu.Unlock()
	
	results := hc.CheckAllComponents()
	
	allHealthy := true
	for name, health := range results {
		if health.Status != HealthStatusHealthy {
			allHealthy = false
			hc.logger.Warn("Component health check failed",
				zap.String("component", name),
				zap.String("status", string(health.Status)),
				zap.String("message", health.Message))
		}
	}
	
	hc.mu.Lock()
	if allHealthy {
		hc.stats.ChecksPassed++
	} else {
		hc.stats.ChecksFailed++
	}
	hc.mu.Unlock()
} 