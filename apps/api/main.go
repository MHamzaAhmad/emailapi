package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/emailapi/api/gen/v1/emailapiv1connect"
	"github.com/emailapi/api/internal/config"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/external/svix"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	"github.com/emailapi/api/internal/repository/postgres"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/service"
	connecttransport "github.com/emailapi/api/internal/transport/connect"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
	"github.com/emailapi/api/internal/webhook"
	worker "github.com/emailapi/api/internal/worker"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func main() {
	// Setup logger with pretty console output
	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}).With().Timestamp().Caller().Logger()

	// Set global logger
	log.Logger = logger

	logger.Info().Msg("🚀 Starting Email API server (Connect RPC)")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
	}

	logger.Info().
		Str("port", cfg.Port).
		Str("env", cfg.Env).
		Msg("Configuration loaded")

	// Initialize repository layer
	store, err := postgres.NewStore(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer store.Close()
	logger.Info().Msg("✓ Connected to database")

	// Initialize SES client
	sesClient, err := ses.NewClient(context.Background(), cfg.AWSRegion)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create SES client")
	}
	logger.Info().Msg("✓ Initialized SES client")

	// Initialize S3 factory with bucket mappings
	s3Factory, err := s3.NewFactory(context.Background(), cfg.AWSRegion, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create S3 factory")
	}
	s3Factory.RegisterBucket(s3.BucketAttachments, cfg.S3Bucket)
	s3Factory.RegisterBucket(s3.BucketInbound, cfg.S3InboundBucket)
	logger.Info().Msg("✓ Initialized S3 factory")

	// Initialize ClickHouse
	chConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.ClickHouseHost},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouseDatabase,
			Username: cfg.ClickHouseUsername,
			Password: cfg.ClickHousePassword,
		},
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name, Version string
			}{
				{Name: "email-api", Version: "0.1.0"},
			},
		},
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to ClickHouse")
	}
	if err := chConn.Ping(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("Failed to ping ClickHouse")
	}
	logger.Info().Msg("✓ Connected to ClickHouse")
	chRepo := chrepo.NewEmailRepository(chConn)
	chActivityRepo := chrepo.NewActivityRepository(chConn)
	logger.Info().Msg("✓ Initialized ClickHouse repositories")

	// Initialize Redis client
	redisClient, err := redisrepo.NewClient(cfg.RedisURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	if err := redisClient.Ping(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("Failed to ping Redis")
	}
	defer redisClient.Close()
	logger.Info().Msg("✓ Connected to Redis")

	// Initialize suppression repository (hybrid Redis + PostgreSQL)
	suppressionRepo := suppression.NewRepository(redisClient, store.Queries())

	// Initialize cache repositories
	const cacheTTL = 5 * time.Minute
	domainCache := redisrepo.NewDomainCache(redisClient, cacheTTL)
	apiKeyCache := redisrepo.NewAPIKeyCache(redisClient, cacheTTL)
	userCache := redisrepo.NewUserCache(redisClient, cacheTTL)
	logger.Info().Msg("✓ Initialized cache repositories")

	// Sync suppression list from PostgreSQL to Redis on startup
	go func() {
		if err := suppressionRepo.SyncFromPostgres(context.Background()); err != nil {
			logger.Warn().Err(err).Msg("Failed to sync suppression list from PostgreSQL")
		}
	}()
	logger.Info().Msg("✓ Initialized suppression repository")

	// Initialize River
	riverPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create River connection pool")
	}
	defer riverPool.Close()

	// Initialize Svix client first (needed for workers)
	svixClient, err := svix.NewClient(cfg.SvixAPIKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create Svix client")
	}

	// Register event types in Svix (idempotent)
	if err := svixClient.EnsureEventTypes(context.Background()); err != nil {
		logger.Warn().Err(err).Msg("Failed to register Svix event types")
	}
	logger.Info().Msg("✓ Initialized Svix client")

	// Create webhook sender from Svix client
	webhookSender := webhook.NewSender(svixClient)

	// Create and register workers
	workers := river.NewWorkers()
	emailWorker := worker.NewEmailWorker(sesClient, s3Factory, chRepo, webhookSender)
	river.AddWorker(workers, emailWorker)
	logger.Info().Msg("✓ Registered River workers")

	// Create River client with workers
	riverClient, err := river.NewClient(riverpgxv5.New(riverPool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: workers,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create River client")
	}

	// Register attachment worker (needs riverClient reference)
	attachmentWorker := worker.NewAttachmentWorker(s3Factory, riverClient)
	river.AddWorker(workers, attachmentWorker)

	// Start River client
	if err := riverClient.Start(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start River client")
	}
	defer riverClient.Stop(context.Background())
	logger.Info().Msg("✓ Started River client")

	// Initialize service layer
	svc := service.NewWithDeps(service.ServiceDeps{
		Store:               store,
		SESClient:           sesClient,
		S3Factory:           s3Factory,
		Region:              cfg.AWSRegion,
		SESConfigurationSet: cfg.SESConfigurationSet,
		RiverClient:         riverClient,
		CHEmailRepo:         chRepo,
		CHActivityRepo:      chActivityRepo,
		SvixClient:          svixClient,
		WebhookSender:       webhookSender,
		SuppressionRepo:     suppressionRepo,
		DomainCache:         domainCache,
		APIKeyCache:         apiKeyCache,
		UserCache:           userCache,
		ClerkWebhookSecret:  cfg.ClerkWebhookSecret,
	})

	// Initialize Clerk SDK with secret key
	clerk.SetKey(cfg.ClerkSecretKey)

	// Create Connect interceptors
	interceptors := connect.WithInterceptors(
		interceptor.NewLoggingInterceptor(logger),
		interceptor.NewSNSInterceptor(),
		interceptor.NewWebhookInterceptor(interceptor.WebhookConfig{
			InternalWebhookSecret: cfg.InternalWebhookSecret,
		}),
		interceptor.NewAuthInterceptor(interceptor.AuthConfig{
			APIKeyService:  svc.APIKey,
			UserLookup:     svc.User,
			ClerkSecretKey: cfg.ClerkSecretKey,
		}),
	)

	// Create HTTP mux and register Connect handlers
	mux := http.NewServeMux()

	// Register all service handlers
	path, handler := emailapiv1connect.NewUserServiceHandler(
		connecttransport.NewUserHandler(svc.User),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = emailapiv1connect.NewApiKeyServiceHandler(
		connecttransport.NewApiKeyHandler(svc.APIKey),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = emailapiv1connect.NewDomainServiceHandler(
		connecttransport.NewDomainHandler(svc.Domain),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = emailapiv1connect.NewEmailServiceHandler(
		connecttransport.NewEmailHandler(svc.Email),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = emailapiv1connect.NewInternalServiceHandler(
		connecttransport.NewInternalHandler(svc.Internal),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = emailapiv1connect.NewWebhookServiceHandler(
		connecttransport.NewWebhookHandler(svc.Webhook),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = emailapiv1connect.NewSnsServiceHandler(
		connecttransport.NewSnsHandler(svc.SNSNotification, svc.InboundEmail),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = emailapiv1connect.NewActivityServiceHandler(
		connecttransport.NewActivityHandler(svc.Activity),
		interceptors,
	)
	mux.Handle(path, handler)

	// CORS middleware
	corsHandler := corsMiddleware(mux)

	// Create HTTP server with h2c (HTTP/2 Cleartext) for gRPC support
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h2c.NewHandler(corsHandler, &http2.Server{}),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info().Str("port", cfg.Port).Msg("🚀 Connect server listening (HTTP + gRPC + gRPC-Web)")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server error")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Server stopped")
}

// corsMiddleware adds CORS headers to allow cross-origin requests from the frontend.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // In production, specify exact origins
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Webhook-Secret, Connect-Protocol-Version, Connect-Timeout-Ms, Grpc-Timeout, X-Grpc-Web, X-User-Agent")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Grpc-Status, Grpc-Message")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
