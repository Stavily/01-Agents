// Package plugin provides plugin download and installation functionality
package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Stavily/01-Agents/shared/pkg/types"
	"go.uber.org/zap"
)

// PluginDownloader handles downloading and installing plugins from repositories
type PluginDownloader struct {
	logger     *zap.Logger
	baseDir    string
	gitTimeout time.Duration
}

// DownloadConfig contains configuration for plugin downloads
type DownloadConfig struct {
	RepositoryURL string `json:"repository_url"`
	Version       string `json:"version"`
	Branch        string `json:"branch"`
	Tag           string `json:"tag"`
	CommitHash    string `json:"commit_hash"`
	SubDirectory  string `json:"sub_directory"`
}

// NewPluginDownloader creates a new plugin downloader
func NewPluginDownloader(logger *zap.Logger, baseDir string) *PluginDownloader {
	return &PluginDownloader{
		logger:     logger,
		baseDir:    baseDir,
		gitTimeout: 5 * time.Minute,
	}
}

// SetGitTimeout sets the timeout for git operations
func (pd *PluginDownloader) SetGitTimeout(timeout time.Duration) {
	pd.gitTimeout = timeout
}

// DownloadPlugin downloads a plugin based on the instruction
func (pd *PluginDownloader) DownloadPlugin(ctx context.Context, inst *types.Instruction) (*types.InstallationResult, error) {
	startTime := time.Now()
	
	pd.logger.Info("Starting plugin download",
		zap.String("instruction_id", inst.ID),
		zap.String("plugin_id", inst.PluginID))

	// Extract download configuration from plugin_configuration or metadata
	config, err := pd.extractDownloadConfig(inst)
	if err != nil {
		return &types.InstallationResult{
			Success:  false,
			PluginID: inst.PluginID,
			Error:    fmt.Sprintf("failed to extract download config: %v", err),
			Duration: time.Since(startTime).Seconds(),
		}, err
	}

	// Create plugin directory
	pluginDir := filepath.Join(pd.baseDir, inst.PluginID)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return &types.InstallationResult{
			Success:  false,
			PluginID: inst.PluginID,
			Error:    fmt.Sprintf("failed to create plugin directory: %v", err),
			Duration: time.Since(startTime).Seconds(),
		}, err
	}

	// Download the plugin
	logs, err := pd.gitClone(ctx, config, pluginDir)
	if err != nil {
		return &types.InstallationResult{
			Success:  false,
			PluginID: inst.PluginID,
			Error:    fmt.Sprintf("git clone failed: %v", err),
			Logs:     logs,
			Duration: time.Since(startTime).Seconds(),
		}, err
	}

	// Verify plugin structure
	if err := pd.verifyPluginStructure(pluginDir); err != nil {
		return &types.InstallationResult{
			Success:  false,
			PluginID: inst.PluginID,
			Error:    fmt.Sprintf("plugin structure verification failed: %v", err),
			Logs:     logs,
			Duration: time.Since(startTime).Seconds(),
		}, err
	}

	result := &types.InstallationResult{
		Success:       true,
		PluginID:      inst.PluginID,
		Version:       config.Version,
		InstalledPath: pluginDir,
		Logs:          logs,
		Duration:      time.Since(startTime).Seconds(),
		Timestamp:     time.Now(),
	}

	pd.logger.Info("Plugin download completed successfully",
		zap.String("instruction_id", inst.ID),
		zap.String("plugin_id", inst.PluginID),
		zap.String("installation_path", pluginDir),
		zap.Float64("duration_seconds", result.Duration))

	return result, nil
}

// extractDownloadConfig extracts download configuration from instruction
func (pd *PluginDownloader) extractDownloadConfig(inst *types.Instruction) (*DownloadConfig, error) {
	config := &DownloadConfig{}

	// Check plugin_configuration for plugin URL (new format)
	if pluginURL, ok := inst.PluginConfiguration["plugin_url"].(string); ok {
		config.RepositoryURL = pluginURL
	} else if repoURL, ok := inst.PluginConfiguration["repository_url"].(string); ok {
		// Fallback to old format for backward compatibility
		config.RepositoryURL = repoURL
	} else if repoURL, ok := inst.Metadata["repository_url"].(string); ok {
		config.RepositoryURL = repoURL
	} else {
		return nil, fmt.Errorf("plugin_url or repository_url not found in plugin configuration or metadata")
	}

	// Extract version information
	if version, ok := inst.PluginConfiguration["version"].(string); ok {
		config.Version = version
	}
	if branch, ok := inst.PluginConfiguration["branch"].(string); ok {
		config.Branch = branch
	}
	if tag, ok := inst.PluginConfiguration["tag"].(string); ok {
		config.Tag = tag
	}
	if commit, ok := inst.PluginConfiguration["commit_hash"].(string); ok {
		config.CommitHash = commit
	}
	if subDir, ok := inst.PluginConfiguration["sub_directory"].(string); ok {
		config.SubDirectory = subDir
	}

	// Support plugin_version field as branch specifier
	if pluginVersion, ok := inst.PluginConfiguration["plugin_version"].(string); ok && pluginVersion != "" {
		// If plugin_version is specified, use it as the branch (overrides branch field)
		config.Branch = pluginVersion
		config.Version = pluginVersion
	}

	// Default to main branch if no specific version info
	if config.Branch == "" && config.Tag == "" && config.CommitHash == "" {
		config.Branch = "main"
	}

	return config, nil
}

// gitClone performs git clone operation with proper error handling
func (pd *PluginDownloader) gitClone(ctx context.Context, config *DownloadConfig, targetDir string) ([]string, error) {
	var logs []string
	
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, pd.gitTimeout)
	defer cancel()

	// Build git clone command
	args := []string{"clone"}
	
	// Add depth for faster clone
	args = append(args, "--depth", "1")
	
	// Add branch/tag if specified
	if config.Tag != "" {
		args = append(args, "--branch", config.Tag)
	} else if config.Branch != "" {
		args = append(args, "--branch", config.Branch)
	}
	
	args = append(args, config.RepositoryURL, targetDir)

	pd.logger.Debug("Executing git clone",
		zap.Strings("args", args),
		zap.String("target_dir", targetDir))

	// Execute git clone
	cmd := exec.CommandContext(timeoutCtx, "git", args...)
	output, err := cmd.CombinedOutput()
	
	logs = append(logs, fmt.Sprintf("git %s", strings.Join(args, " ")))
	logs = append(logs, string(output))

	if err != nil {
		pd.logger.Error("Git clone failed",
			zap.Error(err),
			zap.String("output", string(output)))
		return logs, fmt.Errorf("git clone failed: %v, output: %s", err, string(output))
	}

	// If specific commit hash is required, checkout to it
	if config.CommitHash != "" {
		err := pd.gitCheckoutCommit(timeoutCtx, targetDir, config.CommitHash, &logs)
		if err != nil {
			return logs, err
		}
	}

	return logs, nil
}

// gitCheckoutCommit checks out a specific commit
func (pd *PluginDownloader) gitCheckoutCommit(ctx context.Context, repoDir, commitHash string, logs *[]string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "checkout", commitHash)
	output, err := cmd.CombinedOutput()
	
	*logs = append(*logs, fmt.Sprintf("git -C %s checkout %s", repoDir, commitHash))
	*logs = append(*logs, string(output))

	if err != nil {
		pd.logger.Error("Git checkout failed",
			zap.Error(err),
			zap.String("output", string(output)),
			zap.String("commit_hash", commitHash))
		return fmt.Errorf("git checkout failed: %v, output: %s", err, string(output))
	}

	return nil
}

// verifyPluginStructure verifies that the downloaded plugin has the required structure
func (pd *PluginDownloader) verifyPluginStructure(pluginDir string) error {
	// Check if directory exists and is not empty
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %v", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("plugin directory is empty")
	}

	// Look for common plugin files (this can be enhanced based on plugin standards)
	expectedFiles := []string{
		"plugin.json",
		"plugin.yaml",
		"plugin.yml",
		"manifest.json",
		"manifest.yaml",
		"manifest.yml",
		"Dockerfile",
		"main.py",
		"main.js",
		"main.go",
		"index.js",
		"index.py",
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(filepath.Join(pluginDir, file)); err == nil {
			pd.logger.Debug("Found expected plugin file",
				zap.String("file", file),
				zap.String("plugin_dir", pluginDir))
			return nil
		}
	}

	pd.logger.Warn("No recognized plugin manifest files found, but allowing installation",
		zap.String("plugin_dir", pluginDir))
	
	return nil
}

// CleanupFailedInstallation removes a failed plugin installation
func (pd *PluginDownloader) CleanupFailedInstallation(pluginID string) error {
	pluginDir := filepath.Join(pd.baseDir, pluginID)
	
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return nil // Nothing to clean up
	}

	pd.logger.Info("Cleaning up failed plugin installation",
		zap.String("plugin_id", pluginID),
		zap.String("plugin_dir", pluginDir))

	if err := os.RemoveAll(pluginDir); err != nil {
		pd.logger.Error("Failed to cleanup plugin directory",
			zap.Error(err),
			zap.String("plugin_dir", pluginDir))
		return fmt.Errorf("failed to cleanup plugin directory: %v", err)
	}

	return nil
}

// GetInstalledPluginPath returns the installation path for a plugin
func (pd *PluginDownloader) GetInstalledPluginPath(pluginID string) string {
	return filepath.Join(pd.baseDir, pluginID)
}

// IsPluginInstalled checks if a plugin is already installed
func (pd *PluginDownloader) IsPluginInstalled(pluginID string) bool {
	pluginDir := filepath.Join(pd.baseDir, pluginID)
	_, err := os.Stat(pluginDir)
	return err == nil
} 