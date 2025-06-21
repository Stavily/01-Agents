// Package main implements the Stavily Action Agent
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/stavily/agents/action-agent/internal/agent"
	"github.com/stavily/agents/shared/pkg/config"
)

var (
	version   = "dev"
	buildTime = "unknown"
	cfgFile   string
	logLevel  string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "action-agent",
	Short: "Stavily Action Agent - Execute automation tasks based on workflow definitions",
	Long: `The Stavily Action Agent executes automation tasks based on workflow definitions,
polling the orchestrator for action requests via secure API.

The action agent is designed for reliable task execution with sandboxed plugin 
environment, running on customer infrastructure to provide automation capabilities.`,
	Version: fmt.Sprintf("%s (built %s)", version, buildTime),
	RunE:    runActionAgent,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/stavily/action-agent.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level (debug, info, warn, error, fatal)")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(pluginCmd)
}

// initConfig reads in config file and ENV variables
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in various locations
		viper.SetConfigName("action-agent")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc/stavily/")
		viper.AddConfigPath("$HOME/.stavily/")
		viper.AddConfigPath("./configs/")
		viper.AddConfigPath(".")
	}

	// Environment variables
	viper.SetEnvPrefix("STAVILY_ACTION")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if cfgFile != "" {
			fmt.Fprintf(os.Stderr, "Error reading config file %s: %v\n", cfgFile, err)
			os.Exit(1)
		}
		// If no config file specified and default not found, that's okay
		fmt.Fprintf(os.Stderr, "Warning: No config file found, using defaults\n")
	}

	// Override log level if specified
	if logLevel != "" {
		viper.Set("logging.level", logLevel)
	}
}

// runActionAgent is the main execution function
func runActionAgent(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(viper.ConfigFileUsed())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate that this is an action agent configuration
	if !cfg.IsActionAgent() {
		return fmt.Errorf("configuration is not for an action agent (type: %s)", cfg.Agent.Type)
	}

	// Initialize logger
	logger, err := initLogger(cfg.Logging)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			// Ignore sync errors on stderr/stdout - this is common and expected
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	logger.Info("Starting Stavily Action Agent",
		zap.String("version", version),
		zap.String("build_time", buildTime),
		zap.String("agent_id", cfg.Agent.ID),
		zap.String("tenant_id", cfg.Agent.TenantID),
		zap.String("environment", cfg.Agent.Environment))

	// Validate configuration paths and permissions
	if err := config.ValidateConfigPaths(cfg); err != nil {
		logger.Error("Configuration validation failed", zap.Error(err))
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if err := config.ValidateAgentConfig(cfg); err != nil {
		logger.Error("Agent configuration validation failed", zap.Error(err))
		return fmt.Errorf("agent configuration validation failed: %w", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create and initialize the action agent
	actionAgent, err := agent.NewActionAgent(cfg, logger)
	if err != nil {
		logger.Error("Failed to create action agent", zap.Error(err))
		return fmt.Errorf("failed to create action agent: %w", err)
	}

	// Start the agent
	if err := actionAgent.Start(ctx); err != nil {
		logger.Error("Failed to start action agent", zap.Error(err))
		return fmt.Errorf("failed to start action agent: %w", err)
	}

	logger.Info("Action agent started successfully")

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down")
	}

	// Graceful shutdown
	logger.Info("Initiating graceful shutdown...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := actionAgent.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
		return fmt.Errorf("error during shutdown: %w", err)
	}

	logger.Info("Action agent stopped successfully")
	return nil
}

// initLogger initializes the structured logger
func initLogger(cfg config.LoggingConfig) (*zap.Logger, error) {
	var zapConfig zap.Config

	// Configure log level
	var level zap.AtomicLevel
	switch cfg.Level {
	case "debug":
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "fatal":
		level = zap.NewAtomicLevelAt(zap.FatalLevel)
	default:
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Configure output format
	if cfg.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	zapConfig.Level = level

	// Handle output paths
	if cfg.Output == "file" && cfg.File != "" {
		zapConfig.OutputPaths = []string{cfg.File}
		zapConfig.ErrorOutputPaths = []string{cfg.File}
	} else if cfg.Output == "stderr" {
		zapConfig.OutputPaths = []string{"stderr"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	} else {
		zapConfig.OutputPaths = []string{"stdout"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	}

	return zapConfig.Build()
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Stavily Action Agent %s (built %s)\n", version, buildTime)
	},
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
}

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig(viper.ConfigFileUsed())
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if !cfg.IsActionAgent() {
			return fmt.Errorf("configuration is not for an action agent (type: %s)", cfg.Agent.Type)
		}

		if err := config.ValidateConfigPaths(cfg); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		if err := config.ValidateAgentConfig(cfg); err != nil {
			return fmt.Errorf("agent configuration validation failed: %w", err)
		}

		fmt.Println("Configuration is valid")
		return nil
	},
}

// healthCmd represents the health command
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check agent health status",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement health check logic
		fmt.Println("Health check not yet implemented")
		return nil
	},
}

// pluginCmd represents the plugin command
var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Plugin management commands",
}

func init() {
	// Plugin subcommands
	pluginCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement plugin listing
			fmt.Println("Plugin listing not yet implemented")
			return nil
		},
	})

	pluginCmd.AddCommand(&cobra.Command{
		Use:   "install [plugin-path]",
		Short: "Install a plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement plugin installation
			fmt.Printf("Plugin installation not yet implemented: %s\n", args[0])
			return nil
		},
	})

	pluginCmd.AddCommand(&cobra.Command{
		Use:   "remove [plugin-id]",
		Short: "Remove a plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement plugin removal
			fmt.Printf("Plugin removal not yet implemented: %s\n", args[0])
			return nil
		},
	})
}
