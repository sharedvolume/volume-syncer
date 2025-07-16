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
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create services
	syncService := service.NewSyncService(cfg)

	// Create handlers
	syncHandler := handler.NewSyncHandler(syncService)

	// Create router
	router := gin.Default()

	// Setup routes
	router.GET("/health", syncHandler.HealthCheck)
	router.POST("/api/1.0/sync", syncHandler.Sync)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return &Server{
		httpServer: httpServer,
		cfg:        cfg,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting server on port %s...", s.cfg.Server.Port)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}
