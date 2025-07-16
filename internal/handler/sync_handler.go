package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sharedvolume/volume-syncer/internal/models"
	"github.com/sharedvolume/volume-syncer/internal/service"
)

// SyncHandler handles sync-related HTTP requests
type SyncHandler struct {
	syncService *service.SyncService
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(syncService *service.SyncService) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
	}
}

// HealthCheck handles health check requests
func (h *SyncHandler) HealthCheck(c *gin.Context) {
	log.Printf("[SYNC HANDLER] Health check requested from %s", c.ClientIP())
	response := models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
	}
	log.Printf("[SYNC HANDLER] Health check response sent: %s", response.Status)
	c.JSON(http.StatusOK, response)
}

// Sync handles synchronization requests
func (h *SyncHandler) Sync(c *gin.Context) {
	log.Printf("[SYNC HANDLER] Sync request received from %s", c.ClientIP())

	// Check if sync is already in progress
	log.Printf("[SYNC HANDLER] Checking if sync is already in progress...")
	if h.syncService.IsSyncInProgress() {
		log.Printf("[SYNC HANDLER] ERROR: Sync already in progress")
		response := models.SyncResponse{
			Status:    "busy",
			Error:     "syncing in progress already",
			Timestamp: time.Now().UTC(),
		}
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}
	log.Printf("[SYNC HANDLER] No sync in progress, proceeding...")

	// Parse request
	log.Printf("[SYNC HANDLER] Parsing request body...")
	var request models.SyncRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("[SYNC HANDLER] ERROR: Invalid request format: %v", err)
		response := models.SyncResponse{
			Status:    "error",
			Error:     "invalid request format",
			Details:   err.Error(),
			Timestamp: time.Now().UTC(),
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}
	log.Printf("[SYNC HANDLER] Request parsed successfully - Type: %s, Target: %s", request.Source.Type, request.Target.Path)

	// Start sync
	log.Printf("[SYNC HANDLER] Starting sync operation...")
	if err := h.syncService.StartSync(&request); err != nil {
		log.Printf("[SYNC HANDLER] ERROR: Failed to start sync: %v", err)
		response := models.SyncResponse{
			Status:    "error",
			Error:     "invalid request",
			Details:   err.Error(),
			Timestamp: time.Now().UTC(),
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Return success response
	log.Printf("[SYNC HANDLER] Sync operation started successfully")
	response := models.SyncResponse{
		Status:    "sync started",
		Message:   "synchronization process has been initiated",
		Timestamp: time.Now().UTC(),
	}
	c.JSON(http.StatusCreated, response)
}
