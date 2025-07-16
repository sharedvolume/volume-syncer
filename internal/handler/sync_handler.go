package handler

import (
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
	response := models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
	}
	c.JSON(http.StatusOK, response)
}

// Sync handles synchronization requests
func (h *SyncHandler) Sync(c *gin.Context) {
	// Check if sync is already in progress
	if h.syncService.IsSyncInProgress() {
		response := models.SyncResponse{
			Status:    "busy",
			Error:     "syncing in progress already",
			Timestamp: time.Now().UTC(),
		}
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	// Parse request
	var request models.SyncRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response := models.SyncResponse{
			Status:    "error",
			Error:     "invalid request format",
			Details:   err.Error(),
			Timestamp: time.Now().UTC(),
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Start sync
	if err := h.syncService.StartSync(&request); err != nil {
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
	response := models.SyncResponse{
		Status:    "sync started",
		Message:   "synchronization process has been initiated",
		Timestamp: time.Now().UTC(),
	}
	c.JSON(http.StatusCreated, response)
}
