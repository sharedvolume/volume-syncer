package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sharedvolume/volume-syncer/internal/syncer"
)

type SyncServer struct {
	syncInProgress bool
	mutex          sync.Mutex
	syncManager    *syncer.Manager
}

func NewSyncServer() *SyncServer {
	return &SyncServer{
		syncInProgress: false,
		syncManager:    syncer.NewManager(),
	}
}

func (s *SyncServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *SyncServer) syncEndpoint(c *gin.Context) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.syncInProgress {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":  "syncing in progress already",
			"status": "busy",
		})
		return
	}

	var request syncer.SyncRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate request
	if err := s.syncManager.ValidateRequest(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// Start sync process in background
	s.syncInProgress = true
	go func() {
		defer func() {
			s.mutex.Lock()
			s.syncInProgress = false
			s.mutex.Unlock()
		}()

		if err := s.syncManager.StartSync(&request); err != nil {
			log.Printf("Sync failed: %v", err)
		} else {
			log.Printf("Sync completed successfully")
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"status":    "sync started",
		"message":   "synchronization process has been initiated",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func main() {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	r := gin.Default()

	// Create sync server
	syncServer := NewSyncServer()

	// Routes
	r.GET("/health", syncServer.healthCheck)
	r.POST("/api/1.0/sync", syncServer.syncEndpoint)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port 8080...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give a timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
