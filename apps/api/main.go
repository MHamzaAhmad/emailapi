package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/config"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/repository/postgres"
	"github.com/emailapi/api/internal/service"
	grpctransport "github.com/emailapi/api/internal/transport/grpc"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize repository layer
	store, err := postgres.NewStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer store.Close()

	// Initialize SES client
	sesClient, err := ses.NewClient(context.Background(), "us-east-1") // TODO: Make region configurable
	if err != nil {
		log.Fatalf("failed to create SES client: %v", err)
	}

	// Initialize service layer
	svc := service.NewWithSES(store, sesClient, "us-east-1")

	// Start gRPC server
	go func() {
		if err := runGRPCServer(cfg, svc); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Start HTTP gateway server
	go func() {
		if err := runHTTPServer(cfg); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")
}

// runGRPCServer starts the gRPC server on port 9090.
func runGRPCServer(cfg *config.Config, svc *service.Service) error {
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()

	// Register services
	emailapiv1.RegisterUserServiceServer(grpcServer, grpctransport.NewUserServer(svc.User))
	emailapiv1.RegisterApiKeyServiceServer(grpcServer, grpctransport.NewApiKeyServer(svc.APIKey))
	emailapiv1.RegisterDomainServiceServer(grpcServer, grpctransport.NewDomainServer(svc.Domain))

	// Enable reflection for grpcurl
	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on :%s", cfg.GRPCPort)
	return grpcServer.Serve(lis)
}

// runHTTPServer starts the gRPC-Gateway HTTP server on port 8080.
func runHTTPServer(cfg *config.Config) error {
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

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("HTTP server listening on :%s (gRPC-Gateway)", cfg.Port)
	return srv.ListenAndServe()
}
