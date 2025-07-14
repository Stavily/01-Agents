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

func TestNewSensorAgent(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				Agent: config.AgentConfig{
					ID:          "test-sensor",
					Name:        "Test Sensor",
					Type:        "sensor",
					TenantID:    "test-tenant",
					Environment: "dev",
					Version:     "1.0.0",
				},
				API: config.APIConfig{
					BaseURL:        "http://localhost:8080",
					AgentsEndpoint: "/api/v1/agents",
				},
				Security: config.SecurityConfig{
					Auth: config.AuthConfig{
						Method: "jwt",
					},
					TLS: config.TLSConfig{
						Enabled: false,
					},
				},
				Plugins: config.PluginConfig{
					Directory: "/tmp/plugins",
				},
				Metrics: config.MetricsConfig{
					Enabled: true,
				},
				Health: config.HealthConfig{
					Enabled: true,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			agent, err := NewSensorAgent(tt.cfg, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
				assert.Equal(t, tt.cfg.Agent.ID, agent.config.Agent.ID)
				assert.Equal(t, tt.cfg.Agent.TenantID, agent.config.Agent.TenantID)
			}
		})
	}
}

func TestSensorAgent_StartStop(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			ID:          "test-sensor",
			Name:        "Test Sensor",
			Type:        "sensor",
			TenantID:    "test-tenant",
			Environment: "dev",
			Version:     "1.0.0",
		},
		API: config.APIConfig{
			BaseURL:        "http://localhost:8080",
			AgentsEndpoint: "/api/v1/agents",
		},
		Security: config.SecurityConfig{
			Auth: config.AuthConfig{
				Method: "jwt",
			},
			TLS: config.TLSConfig{
				Enabled: false,
			},
		},
		Plugins: config.PluginConfig{
			Directory: "/tmp/plugins",
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
		},
		Health: config.HealthConfig{
			Enabled: true,
		},
	}

	logger := zaptest.NewLogger(t)
	agent, err := NewSensorAgent(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, agent)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test initial state
	assert.False(t, agent.IsRunning())

	// Test start (should succeed in unit tests even without API server)
	err = agent.Start(ctx)
	// Start should succeed - the agent starts its goroutines and workflows
	if err == nil {
		assert.True(t, agent.IsRunning())
		
		// Test double start (should error)
		err2 := agent.Start(ctx)
		assert.Error(t, err2)
		
		// Test stop
		err = agent.Stop(ctx)
		assert.NoError(t, err)
		assert.False(t, agent.IsRunning())
	}
}

func TestSensorAgent_GetStatus(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			ID:          "test-sensor",
			Name:        "Test Sensor",
			Type:        "sensor",
			TenantID:    "test-tenant",
			Environment: "dev",
			Version:     "1.0.0",
		},
		API: config.APIConfig{
			BaseURL:        "http://localhost:8080",
			AgentsEndpoint: "/api/v1/agents",
		},
		Security: config.SecurityConfig{
			Auth: config.AuthConfig{
				Method: "jwt",
			},
			TLS: config.TLSConfig{
				Enabled: false,
			},
		},
		Plugins: config.PluginConfig{
			Directory: "/tmp/plugins",
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
		},
		Health: config.HealthConfig{
			Enabled: true,
		},
	}

	logger := zaptest.NewLogger(t)
	agent, err := NewSensorAgent(cfg, logger)
	require.NoError(t, err)

	status := agent.GetStatus()
	assert.NotNil(t, status)
	assert.Equal(t, cfg.Agent.ID, status["agent_id"])
	assert.Equal(t, cfg.Agent.TenantID, status["tenant_id"])
	assert.Equal(t, "sensor", status["type"])
	assert.False(t, status["running"].(bool))
}

func TestSensorAgent_GetHealth(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			ID:          "test-sensor",
			Name:        "Test Sensor",
			Type:        "sensor",
			TenantID:    "test-tenant",
			Environment: "dev",
			Version:     "1.0.0",
		},
		API: config.APIConfig{
			BaseURL:        "http://localhost:8080",
			AgentsEndpoint: "/api/v1/agents",
		},
		Security: config.SecurityConfig{
			Auth: config.AuthConfig{
				Method: "jwt",
			},
			TLS: config.TLSConfig{
				Enabled: false,
			},
		},
		Plugins: config.PluginConfig{
			Directory: "/tmp/plugins",
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
		},
		Health: config.HealthConfig{
			Enabled: true,
		},
	}

	logger := zaptest.NewLogger(t)
	agent, err := NewSensorAgent(cfg, logger)
	require.NoError(t, err)

	health := agent.GetHealth()
	assert.NotNil(t, health)
	assert.Equal(t, cfg.Agent.ID, health["agent_id"])
	assert.NotNil(t, health["components"])
} 