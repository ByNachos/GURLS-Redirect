package main

import (
	"GURLS-Redirect/internal/config"
	"GURLS-Redirect/internal/grpc/client"
	handler "GURLS-Redirect/internal/handler/http"
	"context"
	"errors"
	lg "log"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

const regexesPath = "assets/regexes.yaml"

func main() {
	cfg := config.MustLoad()
	
	// Initialize logger
	var log *zap.Logger
	var err error
	if cfg.Env == "production" {
		log, err = zap.NewProduction()
	} else {
		log, err = zap.NewDevelopment()
	}
	if err != nil {
		lg.Fatalf("failed to create logger: %v", err)
	}
	defer func() {
		if err := log.Sync(); err != nil {
			lg.Printf("ERROR: failed to sync zap logger: %v\n", err)
		}
	}()

	log.Info("starting GURLS-Redirect HTTP server", zap.String("env", cfg.Env))

	// Initialize gRPC client to backend
	backendClient, err := client.NewBackendClient(
		cfg.GRPCClient.BackendAddress,
		cfg.GRPCClient.Timeout,
		log,
	)
	if err != nil {
		log.Fatal("failed to connect to backend", zap.Error(err))
	}
	defer backendClient.Close()

	// Create HTTP server
	httpServer, err := handler.NewServer(
		cfg.HTTPServer.Address,
		log,
		backendClient,
		regexesPath,
		cfg.HTTPServer.Timeout,
		cfg.HTTPServer.Timeout,
		cfg.HTTPServer.IdleTimeout,
	)
	if err != nil {
		log.Fatal("failed to create http server", zap.Error(err))
	}

	// Start HTTP server
	go func() {
		log.Info("http server listening", zap.String("address", cfg.HTTPServer.Address))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			log.Error("http server failed to start", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down GURLS-Redirect...")

	if err := httpServer.Shutdown(context.Background()); err != nil {
		log.Error("http server shutdown error", zap.Error(err))
	}
	log.Info("http server stopped")
}