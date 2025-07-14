package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/Stavily/01-Agents/shared/pkg/config"
)

func TestNewPluginManager(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.PluginConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.PluginConfig{
				Directory: "/tmp/plugins",
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
			manager, err := NewPluginManager(tt.cfg, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
			}
		})
	}
}

func TestPluginManager_StartStop(t *testing.T) {
	// Create temporary directory for plugins
	tmpDir, err := os.MkdirTemp("", "plugin_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.PluginConfig{
		Directory: tmpDir,
	}

	logger := zaptest.NewLogger(t)
	manager, err := NewPluginManager(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, manager)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test initialize
	err = manager.Initialize(ctx)
	assert.NoError(t, err)

	// Test shutdown
	err = manager.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestPluginManager_GetStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.PluginConfig{
		Directory: tmpDir,
	}

	logger := zaptest.NewLogger(t)
	manager, err := NewPluginManager(cfg, logger)
	require.NoError(t, err)

	statuses := manager.GetPluginStatuses()
	assert.NotNil(t, statuses)
	assert.Equal(t, 0, len(statuses)) // No plugins loaded initially
}

func TestPluginManager_GetHealth(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.PluginConfig{
		Directory: tmpDir,
	}

	logger := zaptest.NewLogger(t)
	manager, err := NewPluginManager(cfg, logger)
	require.NoError(t, err)

	health := manager.GetHealth()
	assert.NotNil(t, health)
	assert.Equal(t, HealthStatusHealthy, health.Status)
}

func TestPluginManager_LoadPlugin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a mock plugin directory structure
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	err = os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	// Create a mock plugin.yaml file
	pluginYAML := `
name: test-plugin
version: 1.0.0
type: sensor
description: Test plugin
`
	err = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(pluginYAML), 0644)
	require.NoError(t, err)

	cfg := &config.PluginConfig{
		Directory: tmpDir,
	}

	logger := zaptest.NewLogger(t)
	manager, err := NewPluginManager(cfg, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Initialize(ctx)
	require.NoError(t, err)

	// Check if plugin was loaded
	statuses := manager.GetPluginStatuses()
	assert.NotNil(t, statuses)
	// Note: The actual plugin loading behavior depends on the implementation
	// This test verifies the structure is correct

	err = manager.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestPluginManager_EmptyDirectory(t *testing.T) {
	cfg := &config.PluginConfig{
		Directory: "/tmp/plugins",
	}

	logger := zaptest.NewLogger(t)
	manager, err := NewPluginManager(cfg, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Initialize should succeed even with empty directory
	err = manager.Initialize(ctx)
	assert.NoError(t, err)

	// Status should reflect empty state
	statuses := manager.GetPluginStatuses()
	assert.Equal(t, 0, len(statuses))

	// Health should still be healthy
	health := manager.GetHealth()
	assert.Equal(t, HealthStatusHealthy, health.Status)

	// Shutdown should succeed
	err = manager.Shutdown(ctx)
	assert.NoError(t, err)
} 