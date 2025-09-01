package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/semantrix/semaroute/internal/server"
	"github.com/spf13/viper"
)

// Version information - this would be set during build
var (
	version   = "dev"
	commitSHA = "unknown"
	buildTime = "unknown"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("semaroute version %s\n", version)
		fmt.Printf("Commit: %s\n", commitSHA)
		fmt.Printf("Built: %s\n", buildTime)
		os.Exit(0)
	}

	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create server instance
	srv, err := server.NewServer(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Start server
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	srv.WaitForShutdown()
}

// loadConfig loads configuration from file and environment variables.
func loadConfig(configFile string) (*server.Config, error) {
	// Set up Viper
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	
	// Read environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("SEMAROUTE")

	// Set defaults
	setDefaults()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults
		fmt.Println("Config file not found, using defaults")
	}

	// Create config struct
	var config server.Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets sensible default values for configuration.
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 30*time.Second)
	viper.SetDefault("server.write_timeout", 30*time.Second)
	viper.SetDefault("server.idle_timeout", 60*time.Second)
	viper.SetDefault("server.shutdown_timeout", 10*time.Second)

	// Health check defaults
	viper.SetDefault("health_check.interval", 30*time.Second)
	viper.SetDefault("health_check.timeout", 10*time.Second)

	// Routing policy defaults
	viper.SetDefault("routing_policy.type", "cost_based")

	// Cache defaults
	viper.SetDefault("cache.type", "memory")
	viper.SetDefault("cache.ttl", 1*time.Hour)
	viper.SetDefault("cache.max_size", 1000)
	viper.SetDefault("cache.cleanup_interval", 10*time.Minute)

	// Observability defaults
	viper.SetDefault("observability.logging.level", "info")
	viper.SetDefault("observability.logging.format", "json")
	viper.SetDefault("observability.logging.output_path", "logs/app.log")
	viper.SetDefault("observability.logging.error_path", "logs/error.log")
	viper.SetDefault("observability.logging.development", false)

	viper.SetDefault("observability.metrics.enabled", true)
	viper.SetDefault("observability.metrics.port", 9090)
	viper.SetDefault("observability.metrics.path", "/metrics")
	viper.SetDefault("observability.metrics.collect_interval", 15*time.Second)

	viper.SetDefault("observability.tracing.enabled", false)
	viper.SetDefault("observability.tracing.service_name", "semaroute")
	viper.SetDefault("observability.tracing.environment", "development")

	// Provider defaults
	viper.SetDefault("providers.openai.enabled", false)
	viper.SetDefault("providers.openai.timeout", 30*time.Second)
	viper.SetDefault("providers.openai.max_retries", 3)
	viper.SetDefault("providers.openai.retry_delay", 1*time.Second)
	viper.SetDefault("providers.openai.health_check_interval", 30*time.Second)

	viper.SetDefault("providers.anthropic.enabled", false)
	viper.SetDefault("providers.anthropic.timeout", 30*time.Second)
	viper.SetDefault("providers.anthropic.max_retries", 3)
	viper.SetDefault("providers.anthropic.retry_delay", 1*time.Second)
	viper.SetDefault("providers.anthropic.health_check_interval", 30*time.Second)
}
