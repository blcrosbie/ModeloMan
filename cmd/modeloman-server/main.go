package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bcrosbie/modeloman/internal/config"
	"github.com/bcrosbie/modeloman/internal/service"
	"github.com/bcrosbie/modeloman/internal/store"
	grpcx "github.com/bcrosbie/modeloman/internal/transport/grpc"
	httpx "github.com/bcrosbie/modeloman/internal/transport/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()

	hubStore, dataSource, err := buildStore(cfg)
	if err != nil {
		log.Fatalf("store setup failed: %v", err)
	}
	defer func() {
		if err := hubStore.Close(); err != nil {
			log.Printf("store close warning: %v", err)
		}
	}()

	if err := hubStore.Load(); err != nil {
		log.Fatalf("store initialization failed: %v", err)
	}

	keyAuth, _ := hubStore.(store.AgentKeyAuthenticator)
	if keyAuth != nil && strings.TrimSpace(cfg.BootstrapAgentKey) != "" {
		keyID, created, err := keyAuth.EnsureAgentKey(cfg.BootstrapAgentID, cfg.BootstrapAgentKey)
		if err != nil {
			log.Fatalf("failed to seed bootstrap agent key: %v", err)
		}
		if created {
			log.Printf("Bootstrapped agent key agent_id=%s key_id=%s", cfg.BootstrapAgentID, keyID)
		}
	}

	hubService := service.NewHubService(hubStore, dataSource)
	handler := grpcx.NewHubHandler(hubService)
	httpServer := httpx.NewServer(cfg.HTTPAddr, hubService)

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcx.RecoveryUnaryInterceptor(),
			grpcx.AuthUnaryInterceptor(cfg.AuthToken, keyAuth),
			grpcx.LoggingUnaryInterceptor(),
			grpcx.ErrorUnaryInterceptor(),
		),
	)
	grpcx.RegisterHubServer(server, handler)

	healthService := health.NewServer()
	healthService.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthService)

	reflection.Register(server)

	go func() {
		log.Printf("ModeloMan gRPC server listening on %s", cfg.GRPCAddr)
		log.Printf("Store driver=%s source=%s", cfg.StoreDriver, dataSource)
		if keyAuth == nil && cfg.AuthToken == "" {
			log.Printf("AUTH_TOKEN and agent key auth are not configured; write methods are currently unauthenticated.")
		}
		if keyAuth != nil {
			log.Printf("Per-agent API key auth is enabled for write methods.")
		}
		if err := server.Serve(listener); err != nil {
			log.Fatalf("grpc serve failed: %v", err)
		}
	}()

	go func() {
		if strings.TrimSpace(cfg.HTTPAddr) == "" {
			return
		}
		log.Printf("ModeloMan HTTP dashboard listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http serve failed: %v", err)
		}
	}()

	waitForShutdown(server, httpServer)
}

func waitForShutdown(server *grpc.Server, httpServer *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("shutdown signal received; draining gRPC server")
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("gRPC server stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Println("graceful timeout reached; forcing stop")
		server.Stop()
	}
	if httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("http shutdown warning: %v", err)
		}
	}

}

func buildStore(cfg config.Config) (store.HubStore, string, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.StoreDriver)) {
	case "postgres":
		pgStore, err := store.NewPostgresStore(cfg.DatabaseURL)
		if err != nil {
			return nil, "", err
		}
		return pgStore, "postgres", nil
	case "", "file":
		return store.NewFileStore(cfg.DataFile), cfg.DataFile, nil
	default:
		return nil, "", fmt.Errorf("unsupported STORE_DRIVER %q; expected file|postgres", cfg.StoreDriver)
	}
}
