package syncer

import (
	"errors"
	"fmt"
	"time"

	"github.com/sharedvolume/volume-syncer/internal/models"
	"github.com/sharedvolume/volume-syncer/internal/syncer/git"
	"github.com/sharedvolume/volume-syncer/internal/syncer/http"
	"github.com/sharedvolume/volume-syncer/internal/syncer/ssh"
)

// Syncer interface defines the contract for all synchronization implementations
type Syncer interface {
	Sync() error
}

// SyncerFactory creates syncers based on source type
type SyncerFactory struct {
	timeout time.Duration
}

// NewSyncerFactory creates a new syncer factory
func NewSyncerFactory(timeout time.Duration) *SyncerFactory {
	return &SyncerFactory{
		timeout: timeout,
	}
}

// CreateSyncer creates a syncer based on the source type and details
func (f *SyncerFactory) CreateSyncer(source models.Source, targetPath string) (Syncer, error) {
	switch source.Type {
	case "ssh":
		return f.createSSHSyncer(source.Details, targetPath)
	case "git":
		return f.createGitSyncer(source.Details, targetPath)
	case "http":
		return f.createHTTPSyncer(source.Details, targetPath)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", source.Type)
	}
}

func (f *SyncerFactory) createSSHSyncer(details interface{}, targetPath string) (Syncer, error) {
	sshDetails, err := parseSSHDetails(details)
	if err != nil {
		return nil, err
	}
	return ssh.NewSSHSyncer(sshDetails, targetPath, f.timeout), nil
}

func (f *SyncerFactory) createGitSyncer(details interface{}, targetPath string) (Syncer, error) {
	gitDetails, err := parseGitDetails(details)
	if err != nil {
		return nil, err
	}
	return git.NewGitSyncer(gitDetails, targetPath, f.timeout), nil
}

func (f *SyncerFactory) createHTTPSyncer(details interface{}, targetPath string) (Syncer, error) {
	httpDetails, err := parseHTTPDetails(details)
	if err != nil {
		return nil, err
	}
	return http.NewHTTPSyncer(httpDetails, targetPath, f.timeout), nil
}

// parseSSHDetails parses SSH details from interface{}
func parseSSHDetails(details interface{}) (*models.SSHDetails, error) {
	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return nil, errors.New("SSH details must be an object")
	}

	host, ok := detailsMap["host"].(string)
	if !ok || host == "" {
		return nil, errors.New("SSH host is required")
	}

	user, ok := detailsMap["user"].(string)
	if !ok || user == "" {
		return nil, errors.New("SSH user is required")
	}

	sshDetails := &models.SSHDetails{
		Host: host,
		User: user,
		Port: 22, // default port
	}

	if port, ok := detailsMap["port"].(float64); ok {
		sshDetails.Port = int(port)
	}

	if password, ok := detailsMap["password"].(string); ok {
		sshDetails.Password = password
	}

	if keyPath, ok := detailsMap["key_path"].(string); ok {
		sshDetails.KeyPath = keyPath
	}

	return sshDetails, nil
}

// parseGitDetails parses Git details from interface{}
func parseGitDetails(details interface{}) (*models.GitCloneDetails, error) {
	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return nil, errors.New("Git details must be an object")
	}

	url, ok := detailsMap["url"].(string)
	if !ok || url == "" {
		return nil, errors.New("Git URL is required")
	}

	gitDetails := &models.GitCloneDetails{
		URL: url,
	}

	if branch, ok := detailsMap["branch"].(string); ok {
		gitDetails.Branch = branch
	}

	if depth, ok := detailsMap["depth"].(float64); ok {
		gitDetails.Depth = int(depth)
	}

	return gitDetails, nil
}

// parseHTTPDetails parses HTTP details from interface{}
func parseHTTPDetails(details interface{}) (*models.HTTPDownloadDetails, error) {
	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return nil, errors.New("HTTP details must be an object")
	}

	url, ok := detailsMap["url"].(string)
	if !ok || url == "" {
		return nil, errors.New("HTTP URL is required")
	}

	return &models.HTTPDownloadDetails{URL: url}, nil
}
