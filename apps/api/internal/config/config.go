package config

import (
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the application.
type Config struct {
	// Server configuration
	Port string `envconfig:"PORT" default:"8080"`
	Env  string `envconfig:"ENV" default:"development"`

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
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
