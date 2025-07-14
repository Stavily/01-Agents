// Package plugin provides a factory for creating plugin components
package plugin

import (
	"time"

	"go.uber.org/zap"
)

// Factory creates plugin components with consistent configuration
type Factory struct {
	logger    *zap.Logger
	baseDir   string
	gitTimeout time.Duration
	execTimeout time.Duration
}

// FactoryConfig contains configuration for the plugin factory
type FactoryConfig struct {
	BaseDir     string
	GitTimeout  time.Duration
	ExecTimeout time.Duration
}

// NewFactory creates a new plugin factory
func NewFactory(logger *zap.Logger, config *FactoryConfig) *Factory {
	if config == nil {
		config = &FactoryConfig{}
	}

	// Set defaults
	if config.BaseDir == "" {
		config.BaseDir = "./plugins"
	}
	if config.GitTimeout == 0 {
		config.GitTimeout = 5 * time.Minute
	}
	if config.ExecTimeout == 0 {
		config.ExecTimeout = 10 * time.Minute
	}

	return &Factory{
		logger:      logger,
		baseDir:     config.BaseDir,
		gitTimeout:  config.GitTimeout,
		execTimeout: config.ExecTimeout,
	}
}

// CreateDownloader creates a new plugin downloader with factory configuration
func (f *Factory) CreateDownloader() *PluginDownloader {
	downloader := NewPluginDownloader(f.logger, f.baseDir)
	downloader.SetGitTimeout(f.gitTimeout)
	return downloader
}

// CreateExecutor creates a new plugin executor with factory configuration
func (f *Factory) CreateExecutor() *PluginExecutor {
	executor := NewPluginExecutor(f.logger, f.baseDir)
	executor.SetDefaultTimeout(f.execTimeout)
	return executor
}

// GetBaseDir returns the base directory for plugins
func (f *Factory) GetBaseDir() string {
	return f.baseDir
}

// GetGitTimeout returns the git timeout
func (f *Factory) GetGitTimeout() time.Duration {
	return f.gitTimeout
}

// GetExecTimeout returns the execution timeout
func (f *Factory) GetExecTimeout() time.Duration {
	return f.execTimeout
} 