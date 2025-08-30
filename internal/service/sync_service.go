/*
Copyright 2025 SharedVolume

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	log.Printf("[SYNC SERVICE] Starting sync operation")
	log.Printf("[SYNC SERVICE] Source type: %s", req.Source.Type)
	log.Printf("[SYNC SERVICE] Target path: %s", req.Target.Path)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.syncInProgress {
		log.Printf("[SYNC SERVICE] ERROR: Sync operation already in progress")
		return errors.NewValidationError("sync operation already in progress")
	}

	// Validate request
	log.Printf("[SYNC SERVICE] Validating sync request...")
	if err := s.validateRequest(req); err != nil {
		log.Printf("[SYNC SERVICE] ERROR: Request validation failed: %v", err)
		return err
	}
	log.Printf("[SYNC SERVICE] Request validation passed")

	// Create syncer
	log.Printf("[SYNC SERVICE] Creating syncer for type: %s", req.Source.Type)
	syncer, err := s.factory.CreateSyncer(req.Source, req.Target.Path)
	if err != nil {
		log.Printf("[SYNC SERVICE] ERROR: Failed to create syncer: %v", err)
		return fmt.Errorf("failed to create syncer: %w", err)
	}
	log.Printf("[SYNC SERVICE] Syncer created successfully")

	// Start sync process in background
	s.syncInProgress = true
	log.Printf("[SYNC SERVICE] Starting background sync process...")
	go func() {
		defer func() {
			s.mutex.Lock()
			s.syncInProgress = false
			s.mutex.Unlock()
			log.Printf("[SYNC SERVICE] Background sync process completed, status reset")
		}()

		log.Printf("[SYNC SERVICE] Executing sync operation...")
		if err := syncer.Sync(); err != nil {
			log.Printf("[SYNC SERVICE] ERROR: Sync failed: %v", err)
		} else {
			log.Printf("[SYNC SERVICE] Sync completed successfully")
		}
	}()

	log.Printf("[SYNC SERVICE] Sync operation started successfully")
	return nil
}

// validateRequest validates the sync request
func (s *SyncService) validateRequest(req *models.SyncRequest) error {
	log.Printf("[SYNC SERVICE] Validating sync request structure...")

	if req == nil {
		log.Printf("[SYNC SERVICE] ERROR: Sync request is nil")
		return errors.NewValidationError("sync request is required")
	}

	if req.Source.Type == "" {
		log.Printf("[SYNC SERVICE] ERROR: Source type is empty")
		return errors.NewValidationError("source type is required")
	}

	if req.Source.Details == nil {
		log.Printf("[SYNC SERVICE] ERROR: Source details are nil")
		return errors.NewValidationError("source details are required")
	}

	if req.Target.Path == "" {
		log.Printf("[SYNC SERVICE] ERROR: Target path is empty")
		return errors.NewValidationError("target path is required")
	}

	// Validate source type
	log.Printf("[SYNC SERVICE] Validating source type: %s", req.Source.Type)
	switch req.Source.Type {
	case "ssh", "git", "http", "s3":
		log.Printf("[SYNC SERVICE] Source type is valid")
	default:
		log.Printf("[SYNC SERVICE] ERROR: Unsupported source type: %s", req.Source.Type)
		return errors.NewValidationError(fmt.Sprintf("unsupported source type: %s", req.Source.Type))
	}

	log.Printf("[SYNC SERVICE] Request validation completed successfully")
	return nil
}
