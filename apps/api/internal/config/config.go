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
	// DatabaseURL is for pooled connections (app queries)
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`
	// DatabaseURLDirect is for direct (non-pooled) connections
	// Required by River work coordinator which uses LISTEN/NOTIFY
	DatabaseURLDirect string `envconfig:"DATABASE_URL_DIRECT" required:"true"`

	// Tinybird configuration (analytics)
	TinybirdToken   string `envconfig:"TINYBIRD_TOKEN" required:"true"`
	TinybirdBaseURL string `envconfig:"TINYBIRD_BASE_URL" default:"https://api.tinybird.co"`

	// Redis configuration
	RedisURL string `envconfig:"REDIS_URL" required:"true"`

	// Database pool configuration
	DBPoolMaxConns        int32 `envconfig:"DB_POOL_MAX_CONNS" default:"50"`
	DBPoolMinConns        int32 `envconfig:"DB_POOL_MIN_CONNS" default:"10"`
	DBPoolMaxConnLifetime int   `envconfig:"DB_POOL_MAX_CONN_LIFETIME_MIN" default:"30"` // minutes
	DBPoolMaxConnIdleTime int   `envconfig:"DB_POOL_MAX_CONN_IDLE_MIN" default:"5"`      // minutes

	// Redis pool configuration
	RedisPoolSize     int `envconfig:"REDIS_POOL_SIZE" default:"50"`
	RedisMinIdleConns int `envconfig:"REDIS_MIN_IDLE_CONNS" default:"10"`

	// Rate limiting
	RateLimitPerMinute   int  `envconfig:"RATE_LIMIT_PER_MINUTE" default:"100"`
	RateLimitEnabled     bool `envconfig:"RATE_LIMIT_ENABLED" default:"true"`
	MaxConcurrentStreams int  `envconfig:"MAX_CONCURRENT_STREAMS" default:"5"`

	// AWS/S3 Configuration
	AWSRegion          string `envconfig:"AWS_REGION" required:"true"`
	AWSAccessKeyID     string `envconfig:"AWS_ACCESS_KEY_ID" required:"true"`
	AWSSecretAccessKey string `envconfig:"AWS_SECRET_ACCESS_KEY" required:"true"`
	S3Bucket           string `envconfig:"S3_BUCKET" required:"true"`

	// Internal webhook authentication
	InternalWebhookSecret string `envconfig:"INTERNAL_WEBHOOK_SECRET" required:"true"`

	// Svix Configuration (for sending webhooks to users)
	SvixAPIKey string `envconfig:"SVIX_API_KEY" required:"true"`

	// S3 bucket where SES stores inbound emails (replies)
	S3InboundBucket string `envconfig:"S3_INBOUND_BUCKET" required:"true"`

	// SES Configuration Set for email event notifications (delivery, bounce, complaint)
	SESConfigurationSet string `envconfig:"SES_CONFIGURATION_SET" default:""`

	// Clerk Configuration (for verifying Clerk JWTs and webhook signatures)
	ClerkSecretKey     string `envconfig:"CLERK_SECRET_KEY" required:"true"`
	ClerkWebhookSecret string `envconfig:"CLERK_WEBHOOK_SECRET" required:"true"`

	// API Key Configuration
	APIKeyHMACSecret string `envconfig:"API_KEY_HMAC_SECRET" default:"dev-secret-please-change-in-production"`

	// Web Risk Configuration (for URL safety validation)
	WebRiskAPIKey    string `envconfig:"WEB_RISK_API_KEY"`
	WebRiskProjectID string `envconfig:"WEB_RISK_PROJECT_ID"`
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
