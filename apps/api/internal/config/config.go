package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the application.
type Config struct {
	// Server configuration
	Port     string `envconfig:"PORT" default:"8080"`
	GRPCPort string `envconfig:"GRPC_PORT" default:"9090"`
	Env      string `envconfig:"ENV" default:"development"`

	// Database configuration
	DatabaseURL   string `envconfig:"DATABASE_URL" required:"true"`
	ClickHouseURL string `envconfig:"CLICKHOUSE_URL" required:"true"`

	// Redis configuration
	RedisURL string `envconfig:"REDIS_URL" required:"true"`

	// Rate limiting
	RateLimitPerMinute int `envconfig:"RATE_LIMIT_PER_MINUTE" default:"100"`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	// Try to load .env file from root directory (../../.env relative to apps/api)
	// If it doesn't exist, that's ok - we might be using actual env vars
	if wd, err := os.Getwd(); err == nil {
		// Try multiple possible paths
		envPaths := []string{
			filepath.Join(wd, ".env"),                   // Current directory
			filepath.Join(wd, "..", "..", ".env"),       // Root from apps/api
			filepath.Join(wd, "..", "..", "..", ".env"), // Root from apps/api/cmd or nested
		}

		for _, envPath := range envPaths {
			if err := godotenv.Load(envPath); err == nil {
				log.Printf("Loaded .env from: %s", envPath)
				break
			}
		}
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
