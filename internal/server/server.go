package server

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sharedvolume/volume-syncer/internal/config"
	"github.com/sharedvolume/volume-syncer/internal/handler"
	"github.com/sharedvolume/volume-syncer/internal/service"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	cfg        *config.Config
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config) *Server {
	log.Printf("[SERVER] Initializing HTTP server")
	log.Printf("[SERVER] Port: %s", cfg.Server.Port)
	log.Printf("[SERVER] Read timeout: %v", cfg.Server.ReadTimeout)
	log.Printf("[SERVER] Write timeout: %v", cfg.Server.WriteTimeout)
	log.Printf("[SERVER] Idle timeout: %v", cfg.Server.IdleTimeout)

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	log.Printf("[SERVER] Gin mode set to: %s", gin.Mode())

	// Create services
	log.Printf("[SERVER] Creating sync service...")
	syncService := service.NewSyncService(cfg)
	log.Printf("[SERVER] Sync service created")

	// Create handlers
	log.Printf("[SERVER] Creating sync handler...")
	syncHandler := handler.NewSyncHandler(syncService)
	log.Printf("[SERVER] Sync handler created")

	// Create router
	log.Printf("[SERVER] Creating Gin router...")
	router := gin.Default()

	// Setup routes
	log.Printf("[SERVER] Setting up routes...")
	router.GET("/health", syncHandler.HealthCheck)
	router.POST("/api/1.0/sync", syncHandler.Sync)
	log.Printf("[SERVER] Routes configured: GET /health, POST /api/1.0/sync")

	// Create HTTP server
	log.Printf("[SERVER] Creating HTTP server...")
	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	log.Printf("[SERVER] HTTP server created successfully")
	return &Server{
		httpServer: httpServer,
		cfg:        cfg,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("[SERVER] Starting HTTP server on port %s...", s.cfg.Server.Port)
	log.Printf("[SERVER] Server address: %s", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Printf("[SERVER] ERROR: Failed to start server: %v", err)
	}
	return err
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Printf("[SERVER] Initiating graceful shutdown...")
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		log.Printf("[SERVER] ERROR: Failed to shutdown gracefully: %v", err)
	} else {
		log.Printf("[SERVER] Server shutdown completed successfully")
	}
	return err
}
