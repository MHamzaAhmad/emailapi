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

	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/config"
	"github.com/emailapi/api/internal/eventstream"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/external/svix"
	"github.com/emailapi/api/internal/external/webrisk"
	"github.com/emailapi/api/internal/repository/postgres"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/repository/suppression"
	tbrepo "github.com/emailapi/api/internal/repository/tinybird"
	"github.com/emailapi/api/internal/service"
	connecttransport "github.com/emailapi/api/internal/transport/connect"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
	"github.com/emailapi/api/internal/webhook"
	worker "github.com/emailapi/api/internal/worker"

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
	store, err := postgres.NewStore(postgres.StoreConfig{
		DatabaseURL:     cfg.DatabaseURL,
		MaxConns:        cfg.DBPoolMaxConns,
		MinConns:        cfg.DBPoolMinConns,
		MaxConnLifetime: time.Duration(cfg.DBPoolMaxConnLifetime) * time.Minute,
		MaxConnIdleTime: time.Duration(cfg.DBPoolMaxConnIdleTime) * time.Minute,
	})
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

	// Initialize Tinybird client
	tbClient := tbrepo.NewClient(cfg.TinybirdToken, cfg.TinybirdBaseURL)
	if err := tbClient.Ping(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("Failed to ping Tinybird")
	}
	logger.Info().Msg("✓ Connected to Tinybird")
	tbEmailRepo := tbrepo.NewEmailRepository(tbClient)
	tbActivityRepo := tbrepo.NewActivityRepository(tbClient)
	logger.Info().Msg("✓ Initialized Tinybird repositories")

	// Initialize Redis client
	redisClient, err := redisrepo.NewClient(redisrepo.ClientConfig{
		URL:          cfg.RedisURL,
		PoolSize:     cfg.RedisPoolSize,
		MinIdleConns: cfg.RedisMinIdleConns,
	})
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
	mxCache := redisrepo.NewMXCache(redisClient)
	logger.Info().Msg("✓ Initialized cache repositories")

	// Initialize rate limiter
	rateLimiter := redisrepo.NewRateLimiter(redisClient)
	logger.Info().Msg("✓ Initialized rate limiter")

	// Sync suppression list from PostgreSQL to Redis on startup
	go func() {
		if err := suppressionRepo.SyncFromPostgres(context.Background()); err != nil {
			logger.Warn().Err(err).Msg("Failed to sync suppression list from PostgreSQL")
		}
	}()
	logger.Info().Msg("✓ Initialized suppression repository")

	// Initialize River with direct (non-pooled) connection
	// River work coordinator requires direct connection for LISTEN/NOTIFY
	// See: https://riverqueue.com/docs/pgbouncer
	riverPoolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURLDirect)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to parse River database URL")
	}
	riverPoolConfig.MaxConns = cfg.DBPoolMaxConns
	riverPoolConfig.MinConns = cfg.DBPoolMinConns
	riverPoolConfig.MaxConnLifetime = time.Duration(cfg.DBPoolMaxConnLifetime) * time.Minute
	riverPoolConfig.MaxConnIdleTime = time.Duration(cfg.DBPoolMaxConnIdleTime) * time.Minute

	riverPool, err := pgxpool.NewWithConfig(context.Background(), riverPoolConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create River connection pool")
	}
	defer riverPool.Close()
	logger.Info().Msg("✓ Connected River to database (direct connection)")

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

	// Initialize Web Risk client
	webRiskClient, err := webrisk.NewClient(context.Background(), cfg.WebRiskProjectID, cfg.WebRiskAPIKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create Web Risk client")
	}
	logger.Info().Msg("✓ Initialized Web Risk client")

	// Create event stream publisher and consumer
	eventPublisher := eventstream.NewPublisher(redisClient)
	eventConsumer := eventstream.NewConsumer(redisClient)
	logger.Info().Msg("✓ Initialized event stream")

	// Create webhook sender with event stream publisher
	webhookSender := webhook.NewSender(svixClient, eventPublisher)

	// Create and register workers
	workers := river.NewWorkers()
	emailWorker := worker.NewEmailWorker(sesClient, s3Factory, tbEmailRepo, webhookSender)
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
		TBEmailRepo:         tbEmailRepo,
		TBActivityRepo:      tbActivityRepo,
		SvixClient:          svixClient,
		WebhookSender:       webhookSender,
		SuppressionRepo:     suppressionRepo,
		DomainCache:         domainCache,
		APIKeyCache:         apiKeyCache,
		UserCache:           userCache,
		MXCache:             mxCache,
		EventConsumer:       eventConsumer,
		ClerkWebhookSecret:  cfg.ClerkWebhookSecret,
		APIKeyHMACSecret:    cfg.APIKeyHMACSecret,
		RedisClient:         redisClient,
		WebRiskClient:       webRiskClient,
	})

	// Initialize Clerk SDK with secret key
	clerk.SetKey(cfg.ClerkSecretKey)

	// Create Connect interceptors
	// Order matters: logging -> SNS/webhook preprocessing -> auth -> rate limit
	interceptors := connect.WithInterceptors(
		interceptor.NewLoggingInterceptor(logger),
		interceptor.NewSNSInterceptor(),
		interceptor.NewWebhookInterceptor(interceptor.WebhookConfig{
			InternalWebhookSecret: cfg.InternalWebhookSecret,
		}),
		interceptor.NewCombinedAuthInterceptor(interceptor.AuthConfig{
			APIKeyService:  svc.APIKey,
			UserLookup:     svc.User,
			ClerkSecretKey: cfg.ClerkSecretKey,
		}),
		interceptor.NewRateLimitInterceptor(interceptor.RateLimitConfig{
			RateLimiter:          rateLimiter,
			RequestsPerMinute:    cfg.RateLimitPerMinute,
			MaxConcurrentStreams: cfg.MaxConcurrentStreams,
			Enabled:              cfg.RateLimitEnabled,
		}),
	)

	// Create HTTP mux and register Connect handlers
	mux := http.NewServeMux()

	// Register all service handlers
	path, handler := v1connect.NewUserServiceHandler(
		connecttransport.NewUserHandler(svc.User),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = v1connect.NewApiKeyServiceHandler(
		connecttransport.NewApiKeyHandler(svc.APIKey),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = v1connect.NewDomainServiceHandler(
		connecttransport.NewDomainHandler(svc.Domain),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = v1connect.NewEmailServiceHandler(
		connecttransport.NewEmailHandler(svc.Email),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = v1connect.NewInternalServiceHandler(
		connecttransport.NewInternalHandler(svc.Internal),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = v1connect.NewWebhookServiceHandler(
		connecttransport.NewWebhookHandler(svc.Webhook),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = v1connect.NewSnsServiceHandler(
		connecttransport.NewSnsHandler(svc.SNSNotification, svc.InboundEmail),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = v1connect.NewActivityServiceHandler(
		connecttransport.NewActivityHandler(svc.Activity),
		interceptors,
	)
	mux.Handle(path, handler)

	// Apply raw body capture middleware (for webhook signature verification)
	rawBodyHandler := interceptor.RawBodyHTTPMiddleware(mux)

	// CORS middleware
	corsHandler := corsMiddleware(rawBodyHandler)

	// Create HTTP server with h2c (HTTP/2 Cleartext) for gRPC support
	// IdleTimeout closes truly idle connections. WriteTimeout not used - it limits total response time.
	srv := &http.Server{
		Addr:        ":" + cfg.Port,
		Handler:     h2c.NewHandler(corsHandler, &http2.Server{}),
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 120 * time.Second, // Close connections idle for 2 minutes
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
	// Define allowed origins
	allowedOrigins := map[string]bool{
		"https://simpleemailapi.dev":     true,
		"https://www.simpleemailapi.dev": true,
		"http://localhost:3000":          true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if the origin is allowed
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if origin == "" {
			// If no Origin header (e.g., same-origin or non-browser request), allow the first origin
			w.Header().Set("Access-Control-Allow-Origin", "https://simpleemailapi.dev")
		}

		// Set other CORS headers
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
