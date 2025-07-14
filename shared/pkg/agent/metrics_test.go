package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/Stavily/01-Agents/shared/pkg/config"
)

func TestNewMetricsCollector(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.MetricsConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.MetricsConfig{
				Enabled: true,
				Port:    9090,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "disabled metrics",
			cfg: &config.MetricsConfig{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			collector, err := NewMetricsCollector(tt.cfg, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, collector)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, collector)
			}
		})
	}
}

func TestMetricsCollector_StartStop(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: true,
		Port:    9091, // Use different port to avoid conflicts
	}

	logger := zaptest.NewLogger(t)
	collector, err := NewMetricsCollector(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, collector)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test start
	err = collector.Start(ctx)
	assert.NoError(t, err)

	// Test stop
	err = collector.Stop(ctx)
	assert.NoError(t, err)
}

func TestMetricsCollector_GetStatus(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: true,
		Port:    9092,
	}

	logger := zaptest.NewLogger(t)
	collector, err := NewMetricsCollector(cfg, logger)
	require.NoError(t, err)

	status := collector.GetStatus()
	assert.NotNil(t, status)
	assert.Equal(t, 0, status.MetricsExported)
	assert.Equal(t, 0, status.ExportErrors)
}

func TestMetricsCollector_GetHealth(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: true,
		Port:    9093,
	}

	logger := zaptest.NewLogger(t)
	collector, err := NewMetricsCollector(cfg, logger)
	require.NoError(t, err)

	health := collector.GetHealth()
	assert.NotNil(t, health)
	assert.Equal(t, HealthStatusHealthy, health.Status)
}

func TestMetricsCollector_DisabledMetrics(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: false,
	}

	logger := zaptest.NewLogger(t)
	collector, err := NewMetricsCollector(cfg, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Start should succeed even when disabled
	err = collector.Start(ctx)
	assert.NoError(t, err)

	// Status should reflect initial state
	status := collector.GetStatus()
	assert.Equal(t, 0, status.MetricsExported)

	// Health should still be healthy
	health := collector.GetHealth()
	assert.Equal(t, HealthStatusHealthy, health.Status)

	// Stop should succeed
	err = collector.Stop(ctx)
	assert.NoError(t, err)
} 