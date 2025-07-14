// Package main implements the Stavily Sensor Agent
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

	"github.com/Stavily/01-Agents/sensor-agent/internal/agent"
	"github.com/Stavily/01-Agents/shared/pkg/config"
)

var (
	version   = "dev"
	buildTime = "unknown"
	cfgFile   string
	logLevel  string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sensor-agent",
	Short: "Stavily Sensor Agent - Monitor systems and detect trigger conditions",
	Long: `The Stavily Sensor Agent monitors systems and detects trigger conditions,
reporting them to the orchestrator via secure API.

The sensor agent is designed for minimal resource consumption and high reliability,
running on customer infrastructure to provide real-time monitoring capabilities.`,
	Version: fmt.Sprintf("%s (built %s)", version, buildTime),
	RunE:    runSensorAgent,
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/stavily/sensor-agent.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level (debug, info, warn, error, fatal)")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(healthCmd)
}

// initConfig reads in config file and ENV variables
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in various locations
		viper.SetConfigName("sensor-agent")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc/stavily/")
		viper.AddConfigPath("$HOME/.stavily/")
		viper.AddConfigPath("./configs/")
		viper.AddConfigPath(".")
	}

	// Environment variables
	viper.SetEnvPrefix("STAVILY_SENSOR")
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

// runSensorAgent is the main execution function
func runSensorAgent(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(viper.ConfigFileUsed())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate that this is a sensor agent configuration
	if !cfg.IsSensorAgent() {
		return fmt.Errorf("configuration is not for a sensor agent (type: %s)", cfg.Agent.Type)
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

	logger.Info("Starting Stavily Sensor Agent",
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

	// Create and initialize the sensor agent
	sensorAgent, err := agent.NewSensorAgent(cfg, logger)
	if err != nil {
		logger.Error("Failed to create sensor agent", zap.Error(err))
		return fmt.Errorf("failed to create sensor agent: %w", err)
	}

	// Start the agent
	if err := sensorAgent.Start(ctx); err != nil {
		logger.Error("Failed to start sensor agent", zap.Error(err))
		return fmt.Errorf("failed to start sensor agent: %w", err)
	}

	logger.Info("Sensor agent started successfully")

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

	if err := sensorAgent.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
		return fmt.Errorf("error during shutdown: %w", err)
	}

	logger.Info("Sensor agent stopped successfully")
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

	// Configure output destination
	switch cfg.Output {
	case "stderr":
		zapConfig.OutputPaths = []string{"stderr"}
	case "file":
		if cfg.File != "" {
			zapConfig.OutputPaths = []string{cfg.File}
		} else {
			zapConfig.OutputPaths = []string{"stdout"}
		}
	default:
		zapConfig.OutputPaths = []string{"stdout"}
	}

	return zapConfig.Build()
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Stavily Sensor Agent\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Go Version: %s\n", "go1.21")
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

		if err := config.ValidateConfigPaths(cfg); err != nil {
			return fmt.Errorf("configuration path validation failed: %w", err)
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
	Short: "Check agent health",
	RunE: func(cmd *cobra.Command, args []string) error {
		// This would implement a health check against a running agent
		fmt.Println("Health check not implemented for standalone execution")
		return nil
	},
}
