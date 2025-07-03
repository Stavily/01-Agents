// Enhanced Agent Main - Implements AGENT_USE.md specification
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/stavily/agents/shared/pkg/agent"
	"github.com/stavily/agents/shared/pkg/config"
)

var (
	version   = "dev"
	buildTime = "unknown"
	cfgFile   string
	debug     bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "enhanced-agent",
	Short: "Stavily Enhanced Agent - Implements AGENT_USE.md specification",
	Long: `Enhanced Agent is a Stavily agent implementation that follows the AGENT_USE.md specification.
It provides:
- Heartbeat functionality
- Instruction polling (GET instructions)
- Instruction updates (PUT instructions)  
- Result submission (POST instruction results)`,
	RunE: runAgent,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(validateCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Look for config in current directory
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("STAVILY")
}

func runAgent(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Setup logger
	logger, err := setupLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("Starting Enhanced Agent",
		zap.String("version", version),
		zap.String("build_time", buildTime),
		zap.String("agent_id", cfg.Agent.ID),
		zap.String("tenant_id", cfg.Agent.TenantID),
		zap.String("base_folder", cfg.Agent.BaseFolder))

	// Create enhanced agent
	enhancedAgent, err := agent.NewEnhancedAgent(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create enhanced agent: %w", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the agent
	if err := enhancedAgent.Start(ctx); err != nil {
		return fmt.Errorf("failed to start enhanced agent: %w", err)
	}

	logger.Info("Enhanced Agent started successfully")

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop the agent gracefully
	if err := enhancedAgent.Stop(shutdownCtx); err != nil {
		logger.Error("Error during agent shutdown", zap.Error(err))
		return err
	}

	logger.Info("Enhanced Agent stopped successfully")
	return nil
}

func loadConfig() (*config.Config, error) {
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg, err := config.LoadConfig(viper.ConfigFileUsed())
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

func setupLogger(cfg *config.Config) (*zap.Logger, error) {
	var zapConfig zap.Config

	if debug || cfg.Logging.Level == "debug" {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	} else {
		zapConfig = zap.NewProductionConfig()
		
		// Set log level based on config
		switch cfg.Logging.Level {
		case "info":
			zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		case "warn":
			zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
		case "error":
			zapConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
		default:
			zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		}
	}

	// Configure output
	if cfg.Logging.Output == "file" && cfg.Logging.File != "" {
		// Ensure log directory exists
		if err := os.MkdirAll(filepath.Dir(cfg.Logging.File), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		zapConfig.OutputPaths = []string{cfg.Logging.File}
		zapConfig.ErrorOutputPaths = []string{cfg.Logging.File}
	}

	// Set encoding format
	if cfg.Logging.Format == "text" {
		zapConfig.Encoding = "console"
	} else {
		zapConfig.Encoding = "json"
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return logger, nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Enhanced Agent\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Specification: AGENT_USE.md\n")
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		fmt.Printf("Configuration is valid\n")
		fmt.Printf("Agent ID: %s\n", cfg.Agent.ID)
		fmt.Printf("Agent Type: %s\n", cfg.Agent.Type)
		fmt.Printf("Tenant ID: %s\n", cfg.Agent.TenantID)
		fmt.Printf("Base Folder: %s\n", cfg.Agent.BaseFolder)
		fmt.Printf("API Base URL: %s\n", cfg.API.BaseURL)
		fmt.Printf("Heartbeat Interval: %s\n", cfg.Agent.Heartbeat)
		fmt.Printf("Poll Interval: %s\n", cfg.Agent.PollInterval)

		return nil
	},
} 