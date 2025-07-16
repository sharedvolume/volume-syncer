package service

import (
	"fmt"
	"log"
	"sync"

	"github.com/sharedvolume/volume-syncer/internal/config"
	"github.com/sharedvolume/volume-syncer/internal/models"
	"github.com/sharedvolume/volume-syncer/internal/syncer"
	"github.com/sharedvolume/volume-syncer/pkg/errors"
)

// SyncService handles synchronization operations
type SyncService struct {
	factory        *syncer.SyncerFactory
	syncInProgress bool
	mutex          sync.Mutex
}

// NewSyncService creates a new sync service
func NewSyncService(cfg *config.Config) *SyncService {
	return &SyncService{
		factory:        syncer.NewSyncerFactory(cfg.Sync.DefaultTimeout),
		syncInProgress: false,
	}
}

// IsSyncInProgress returns true if a sync operation is currently in progress
func (s *SyncService) IsSyncInProgress() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.syncInProgress
}

// StartSync starts the synchronization process
func (s *SyncService) StartSync(req *models.SyncRequest) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.syncInProgress {
		return errors.NewValidationError("sync operation already in progress")
	}

	// Validate request
	if err := s.validateRequest(req); err != nil {
		return err
	}

	// Create syncer
	syncer, err := s.factory.CreateSyncer(req.Source, req.Target.Path)
	if err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}

	// Start sync process in background
	s.syncInProgress = true
	go func() {
		defer func() {
			s.mutex.Lock()
			s.syncInProgress = false
			s.mutex.Unlock()
		}()

		if err := syncer.Sync(); err != nil {
			log.Printf("Sync failed: %v", err)
		} else {
			log.Printf("Sync completed successfully")
		}
	}()

	return nil
}

// validateRequest validates the sync request
func (s *SyncService) validateRequest(req *models.SyncRequest) error {
	if req == nil {
		return errors.NewValidationError("sync request is required")
	}

	if req.Source.Type == "" {
		return errors.NewValidationError("source type is required")
	}

	if req.Source.Details == nil {
		return errors.NewValidationError("source details are required")
	}

	if req.Target.Path == "" {
		return errors.NewValidationError("target path is required")
	}

	// Validate source type
	switch req.Source.Type {
	case "ssh", "git", "http":
		// Valid types
	default:
		return errors.NewValidationError(fmt.Sprintf("unsupported source type: %s", req.Source.Type))
	}

	return nil
}
