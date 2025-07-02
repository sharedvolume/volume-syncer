package syncer

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

// SyncRequest represents the sync request payload
type SyncRequest struct {
	Source  Source `json:"source" binding:"required"`
	Target  Target `json:"target" binding:"required"`
	Timeout string `json:"timeout,omitempty"`
}

// Source represents the source configuration
type Source struct {
	Type    string      `json:"type" binding:"required"`
	Details interface{} `json:"details" binding:"required"`
}

// Target represents the target configuration
type Target struct {
	Path string `json:"path" binding:"required"`
}

// SSHDetails represents SSH connection details
type SSHDetails struct {
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port,omitempty"`
	Username   string `json:"username,omitempty"`
	Path       string `json:"path" binding:"required"`
	PrivateKey string `json:"privateKey" binding:"required"`
}

// Manager handles sync operations
type Manager struct{}

// NewManager creates a new sync manager
func NewManager() *Manager {
	return &Manager{}
}

// ValidateRequest validates the sync request
func (m *Manager) ValidateRequest(req *SyncRequest) error {
	if req.Source.Type != "ssh" {
		return errors.New("only SSH source type is supported currently")
	}

	// Parse SSH details
	sshDetails, err := m.parseSSHDetails(req.Source.Details)
	if err != nil {
		return fmt.Errorf("invalid SSH details: %w", err)
	}

	// Validate SSH details
	if sshDetails.Host == "" {
		return errors.New("SSH host is required")
	}

	if sshDetails.Path == "" {
		return errors.New("SSH path is required")
	}

	if sshDetails.PrivateKey == "" {
		return errors.New("SSH private key is required")
	}

	// Validate private key format (base64)
	if _, err := base64.StdEncoding.DecodeString(sshDetails.PrivateKey); err != nil {
		return errors.New("private key must be base64 encoded")
	}

	// Validate target path
	if req.Target.Path == "" {
		return errors.New("target path is required")
	}

	// Validate timeout if provided
	if req.Timeout != "" {
		if _, err := time.ParseDuration(req.Timeout); err != nil {
			return fmt.Errorf("invalid timeout format: %w", err)
		}
	}

	return nil
}

// StartSync starts the synchronization process
func (m *Manager) StartSync(req *SyncRequest) error {
	sshDetails, err := m.parseSSHDetails(req.Source.Details)
	if err != nil {
		return fmt.Errorf("failed to parse SSH details: %w", err)
	}

	// Set defaults
	if sshDetails.Port == 0 {
		sshDetails.Port = 22
	}
	if sshDetails.Username == "" {
		sshDetails.Username = "root"
	}

	timeout := 30 * time.Second
	if req.Timeout != "" {
		if parsedTimeout, err := time.ParseDuration(req.Timeout); err == nil {
			timeout = parsedTimeout
		}
	}

	// Create SSH syncer
	sshSyncer := NewSSHSyncer(sshDetails, req.Target.Path, timeout)

	// Start sync
	return sshSyncer.Sync()
}

// parseSSHDetails parses SSH details from interface{}
func (m *Manager) parseSSHDetails(details interface{}) (*SSHDetails, error) {
	fmt.Printf("DEBUG: Parsing SSH details: %+v\n", details)

	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return nil, errors.New("SSH details must be an object")
	}

	fmt.Printf("DEBUG: Details map: %+v\n", detailsMap)

	sshDetails := &SSHDetails{}

	if host, ok := detailsMap["host"].(string); ok {
		sshDetails.Host = host
		fmt.Printf("DEBUG: Found host: %s\n", host)
	}

	if port, ok := detailsMap["port"].(float64); ok {
		sshDetails.Port = int(port)
		fmt.Printf("DEBUG: Found port: %d\n", int(port))
	}

	if username, ok := detailsMap["username"].(string); ok {
		sshDetails.Username = username
		fmt.Printf("DEBUG: Found username: %s\n", username)
	}

	if path, ok := detailsMap["path"].(string); ok {
		sshDetails.Path = path
		fmt.Printf("DEBUG: Found path: %s\n", path)
	} else {
		fmt.Printf("DEBUG: Path not found or not a string. Raw value: %+v (type: %T)\n", detailsMap["path"], detailsMap["path"])
	}

	if privateKey, ok := detailsMap["privateKey"].(string); ok {
		sshDetails.PrivateKey = privateKey
		fmt.Printf("DEBUG: Found privateKey (length: %d)\n", len(privateKey))
	}

	fmt.Printf("DEBUG: Final SSH details: Host=%s, Port=%d, Username=%s, Path=%s, PrivateKey length=%d\n",
		sshDetails.Host, sshDetails.Port, sshDetails.Username, sshDetails.Path, len(sshDetails.PrivateKey))

	return sshDetails, nil
}
