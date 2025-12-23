package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/config"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/external/svix"
	middleware "github.com/emailapi/api/internal/middleware"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	"github.com/emailapi/api/internal/repository/postgres"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/service"
	grpctransport "github.com/emailapi/api/internal/transport/grpc"
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

	logger.Info().Msg("🚀 Starting Email API server")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
	}

	logger.Info().
		Str("port", cfg.Port).
		Str("grpc_port", cfg.GRPCPort).
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
	// Use native interface for better performance (async insert, etc.)
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
	// Create a new pgxpool for River (recommended to separate from application pool)
	riverPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create River connection pool")
	}
	defer riverPool.Close()

	workers := river.NewWorkers()
	// Register EmailWorker (no PG dependency - email data in job payload)
	emailWorker := worker.NewEmailWorker(sesClient, s3Factory, chRepo)
	river.AddWorker(workers, emailWorker)

	riverClient, err := river.NewClient(riverpgxv5.New(riverPool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: workers,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create River client")
	}

	// Start River client
	if err := riverClient.Start(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start River client")
	}
	defer riverClient.Stop(context.Background())
	logger.Info().Msg("✓ Started River client")

	// Initialize Svix client
	svixClient, err := svix.NewClient(cfg.SvixAPIKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create Svix client")
	}

	// Register event types in Svix (idempotent)
	if err := svixClient.EnsureEventTypes(context.Background()); err != nil {
		logger.Warn().Err(err).Msg("Failed to register Svix event types")
	}
	logger.Info().Msg("✓ Initialized Svix client")

	// Initialize service layer (no PG email repo - stateless architecture)
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
		SuppressionRepo:     suppressionRepo,
		DomainCache:         domainCache,
		APIKeyCache:         apiKeyCache,
		UserCache:           userCache,
	})

	// Start gRPC server
	go func() {
		if err := runGRPCServer(cfg, svc, logger); err != nil {
			logger.Fatal().Err(err).Msg("gRPC server error")
		}
	}()

	// Start HTTP gateway server
	go func() {
		if err := runHTTPServer(cfg, logger); err != nil {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down servers...")
}

// runGRPCServer starts the gRPC server on port 9090.
func runGRPCServer(cfg *config.Config, svc *service.Service, logger zerolog.Logger) error {
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return err
	}

	// Create interceptors
	loggingInterceptor := middleware.NewLoggingInterceptor(logger)
	snsInterceptor := middleware.NewSNSInterceptor()
	authInterceptor := middleware.NewAuthInterceptor(svc.APIKey)
	webhookInterceptor := middleware.NewWebhookInterceptor(cfg.InternalWebhookSecret)

	// Chain interceptors: logging -> SNS verification -> webhook verification -> auth
	// Note: SNS verification runs for SNS endpoints (public, verified by signature)
	//       webhook verification runs for internal endpoints
	//       auth runs for API endpoints
	chainedInterceptor := func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		return loggingInterceptor.Unary()(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
			return snsInterceptor.Unary()(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
				return webhookInterceptor.Unary()(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
					return authInterceptor.Unary()(ctx, req, info, handler)
				})
			})
		})
	}

	// Create gRPC server with chained interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(chainedInterceptor),
	)

	// Register services
	emailapiv1.RegisterUserServiceServer(grpcServer, grpctransport.NewUserServer(svc.User))
	emailapiv1.RegisterApiKeyServiceServer(grpcServer, grpctransport.NewApiKeyServer(svc.APIKey))
	emailapiv1.RegisterDomainServiceServer(grpcServer, grpctransport.NewDomainServer(svc.Domain))
	emailapiv1.RegisterEmailServiceServer(grpcServer, grpctransport.NewEmailServer(svc.Email))
	emailapiv1.RegisterInternalServiceServer(grpcServer, grpctransport.NewInternalServer(svc.Internal))
	emailapiv1.RegisterWebhookServiceServer(grpcServer, grpctransport.NewWebhookServer(svc.Webhook))
	emailapiv1.RegisterSnsServiceServer(grpcServer, grpctransport.NewSnsServer(svc.SNSNotification, svc.InboundEmail))

	// Enable reflection for grpcurl
	reflection.Register(grpcServer)

	logger.Info().Str("port", cfg.GRPCPort).Msg("📡 gRPC server listening")
	return grpcServer.Serve(lis)
}

// runHTTPServer starts the gRPC-Gateway HTTP server on port 8080.
func runHTTPServer(cfg *config.Config, logger zerolog.Logger) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// Register HTTP handlers that proxy to gRPC
	grpcEndpoint := "localhost:" + cfg.GRPCPort

	if err := emailapiv1.RegisterUserServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		return err
	}
	if err := emailapiv1.RegisterApiKeyServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		return err
	}
	if err := emailapiv1.RegisterDomainServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		return err
	}
	if err := emailapiv1.RegisterEmailServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		return err
	}
	if err := emailapiv1.RegisterInternalServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		return err
	}
	if err := emailapiv1.RegisterWebhookServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		return err
	}
	if err := emailapiv1.RegisterSnsServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts); err != nil {
		return err
	}

	// Chain middlewares: CORS -> Logging -> gRPC-Gateway
	handler := corsMiddleware(httpLoggingMiddleware(mux, logger))

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	logger.Info().Str("port", cfg.Port).Msg("🌐 HTTP server listening (gRPC-Gateway)")
	return srv.ListenAndServe()
}

// httpLoggingMiddleware logs HTTP requests.
func httpLoggingMiddleware(next http.Handler, logger zerolog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(ww, r)

		// Log the request
		duration := time.Since(start)
		logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", ww.statusCode).
			Dur("duration", duration).
			Str("remote_addr", r.RemoteAddr).
			Msg("HTTP request")
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// corsMiddleware adds CORS headers to allow cross-origin requests from the frontend.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // In production, specify exact origins
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Webhook-Secret")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
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
