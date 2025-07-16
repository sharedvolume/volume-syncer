package models

import "time"

// SyncRequest represents the sync request payload
type SyncRequest struct {
	Source Source `json:"source" binding:"required"`
	Target Target `json:"target" binding:"required"`
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
	Port       int    `json:"port"`
	User       string `json:"user" binding:"required"`
	Password   string `json:"password,omitempty"`
	KeyPath    string `json:"key_path,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`    // Base64 encoded private key
	Path       string `json:"path" binding:"required"` // Remote path to sync
}

// GitCloneDetails represents Git clone details
type GitCloneDetails struct {
	URL        string `json:"url" binding:"required"`
	Branch     string `json:"branch"`
	Depth      int    `json:"depth"`
	User       string `json:"user,omitempty"`       // For HTTP(S) authentication
	Password   string `json:"password,omitempty"`   // For HTTP(S) authentication
	PrivateKey string `json:"privateKey,omitempty"` // Base64 encoded private key for SSH
}

// HTTPDownloadDetails represents HTTP download details
type HTTPDownloadDetails struct {
	URL string `json:"url" binding:"required"`
}

// S3Details represents S3 synchronization details
type S3Details struct {
	EndpointURL string `json:"endpointUrl" binding:"required"`
	BucketName  string `json:"bucketName" binding:"required"`
	Path        string `json:"path" binding:"required"`
	AccessKey   string `json:"accessKey" binding:"required"`
	SecretKey   string `json:"secretKey" binding:"required"`
	Region      string `json:"region" binding:"required"`
}

// SyncResponse represents the response for sync operations
type SyncResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Error     string    `json:"error,omitempty"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}
