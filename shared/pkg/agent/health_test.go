package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stavily/agents/shared/pkg/config"
	"go.uber.org/zap/zaptest"
)

func TestNewHealthChecker(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.HealthConfig{
		Enabled:  true,
		Interval: 30 * time.Second,
	}

	hc, err := NewHealthChecker(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create health checker: %v", err)
	}

	if hc == nil {
		t.Fatal("Health checker is nil")
	}

	if hc.cfg != cfg {
		t.Error("Config not set correctly")
	}

	if hc.logger != logger {
		t.Error("Logger not set correctly")
	}

	if hc.checkers == nil {
		t.Error("Checkers map not initialized")
	}
}

func TestHealthChecker_RegisterComponent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.HealthConfig{}
	hc, _ := NewHealthChecker(cfg, logger)

	// Register a test component
	componentName := "test-component"
	checker := func() *ComponentHealth {
		return &ComponentHealth{
			Status:    HealthStatusHealthy,
			LastCheck: time.Now(),
		}
	}

	hc.RegisterComponent(componentName, checker)

	// Verify component was registered
	hc.mu.RLock()
	_, exists := hc.checkers[componentName]
	hc.mu.RUnlock()

	if !exists {
		t.Error("Component was not registered")
	}
}

func TestHealthChecker_CheckAllComponents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.HealthConfig{}
	hc, _ := NewHealthChecker(cfg, logger)

	// Register test components
	healthyChecker := func() *ComponentHealth {
		return &ComponentHealth{
			Status:    HealthStatusHealthy,
			LastCheck: time.Now(),
		}
	}

	unhealthyChecker := func() *ComponentHealth {
		return &ComponentHealth{
			Status:     HealthStatusUnhealthy,
			LastCheck:  time.Now(),
			ErrorCount: 1,
			Message:    "Test error",
		}
	}

	hc.RegisterComponent("healthy-component", healthyChecker)
	hc.RegisterComponent("unhealthy-component", unhealthyChecker)

	// Check all components
	results := hc.CheckAllComponents()

	if len(results) != 2 {
		t.Errorf("Expected 2 components, got %d", len(results))
	}

	healthyResult := results["healthy-component"]
	if healthyResult == nil || healthyResult.Status != HealthStatusHealthy {
		t.Error("Healthy component check failed")
	}

	unhealthyResult := results["unhealthy-component"]
	if unhealthyResult == nil || unhealthyResult.Status != HealthStatusUnhealthy {
		t.Error("Unhealthy component check failed")
	}
}

func TestHealthChecker_StartStop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.HealthConfig{}
	hc, _ := NewHealthChecker(cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start health checker
	err := hc.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start health checker: %v", err)
	}

	// Stop health checker
	err = hc.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop health checker: %v", err)
	}
}

func TestHealthChecker_GetStatus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.HealthConfig{}
	hc, _ := NewHealthChecker(cfg, logger)

	status := hc.GetStatus()
	if status == nil {
		t.Fatal("Status is nil")
	}

	if status.CheckInterval != 30*time.Second {
		t.Error("Default check interval not set correctly")
	}
}

func TestHealthChecker_GetHealth(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.HealthConfig{}
	hc, _ := NewHealthChecker(cfg, logger)

	health := hc.GetHealth()
	if health == nil {
		t.Fatal("Health is nil")
	}

	if health.Status != HealthStatusHealthy {
		t.Error("Health checker should be healthy by default")
	}

	if health.ErrorCount != 0 {
		t.Error("Error count should be 0 by default")
	}
}

func TestHealthStatus_String(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusHealthy, "healthy"},
		{HealthStatusDegraded, "degraded"},
		{HealthStatusUnhealthy, "unhealthy"},
		{HealthStatusUnknown, "unknown"},
	}

	for _, test := range tests {
		if string(test.status) != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, string(test.status))
		}
	}
}

func BenchmarkHealthChecker_CheckAllComponents(b *testing.B) {
	logger := zaptest.NewLogger(b)
	cfg := &config.HealthConfig{}
	hc, _ := NewHealthChecker(cfg, logger)

	// Register multiple components
	for i := 0; i < 10; i++ {
		componentName := fmt.Sprintf("component-%d", i)
		checker := func() *ComponentHealth {
			return &ComponentHealth{
				Status:    HealthStatusHealthy,
				LastCheck: time.Now(),
			}
		}
		hc.RegisterComponent(componentName, checker)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hc.CheckAllComponents()
	}
} 