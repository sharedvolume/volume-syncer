package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sharedvolume/volume-syncer/internal/config"
	"github.com/sharedvolume/volume-syncer/internal/server"
)

func main() {
	log.Printf("[MAIN] Starting Volume Syncer application")
	log.Printf("[MAIN] Process ID: %d", os.Getpid())

	// Load configuration
	log.Printf("[MAIN] Loading configuration...")
	cfg := config.Load()
	log.Printf("[MAIN] Configuration loaded successfully")

	// Create server
	log.Printf("[MAIN] Creating server...")
	srv := server.NewServer(cfg)
	log.Printf("[MAIN] Server created successfully")

	// Start server in a goroutine
	log.Printf("[MAIN] Starting server...")
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[MAIN] FATAL: Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	log.Printf("[MAIN] Server started successfully, waiting for shutdown signal...")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("[MAIN] Shutdown signal received, initiating graceful shutdown...")
	// Give a timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[MAIN] FATAL: Server forced to shutdown: %v", err)
	}

	log.Printf("[MAIN] Server shutdown completed successfully")
}
