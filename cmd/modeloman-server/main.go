package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bcrosbie/modeloman/internal/config"
	"github.com/bcrosbie/modeloman/internal/service"
	"github.com/bcrosbie/modeloman/internal/store"
	grpcx "github.com/bcrosbie/modeloman/internal/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()

	fileStore := store.NewFileStore(cfg.DataFile)
	if err := fileStore.Load(); err != nil {
		log.Fatalf("store initialization failed: %v", err)
	}

	hubService := service.NewHubService(fileStore, cfg.DataFile)
	handler := grpcx.NewHubHandler(hubService)

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcx.RecoveryUnaryInterceptor(),
			grpcx.AuthUnaryInterceptor(cfg.AuthToken),
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
		if cfg.AuthToken == "" {
			log.Printf("AUTH_TOKEN is not set; write methods are currently unauthenticated.")
		}
		if err := server.Serve(listener); err != nil {
			log.Fatalf("grpc serve failed: %v", err)
		}
	}()

	waitForShutdown(server)
}

func waitForShutdown(server *grpc.Server) {
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

}
