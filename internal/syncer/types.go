package syncer

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sharedvolume/volume-syncer/internal/models"
	"github.com/sharedvolume/volume-syncer/internal/syncer/git"
	"github.com/sharedvolume/volume-syncer/internal/syncer/http"
	"github.com/sharedvolume/volume-syncer/internal/syncer/s3"
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
	log.Printf("[SYNCER FACTORY] Creating syncer for type: %s", source.Type)
	log.Printf("[SYNCER FACTORY] Target path: %s", targetPath)
	log.Printf("[SYNCER FACTORY] Timeout: %v", f.timeout)

	switch source.Type {
	case "ssh":
		log.Printf("[SYNCER FACTORY] Creating SSH syncer")
		return f.createSSHSyncer(source.Details, targetPath)
	case "git":
		log.Printf("[SYNCER FACTORY] Creating Git syncer")
		return f.createGitSyncer(source.Details, targetPath)
	case "http":
		log.Printf("[SYNCER FACTORY] Creating HTTP syncer")
		return f.createHTTPSyncer(source.Details, targetPath)
	case "s3":
		log.Printf("[SYNCER FACTORY] Creating S3 syncer")
		return f.createS3Syncer(source.Details, targetPath)
	default:
		log.Printf("[SYNCER FACTORY] ERROR: Unsupported source type: %s", source.Type)
		return nil, fmt.Errorf("unsupported source type: %s", source.Type)
	}
}

func (f *SyncerFactory) createSSHSyncer(details interface{}, targetPath string) (Syncer, error) {
	log.Printf("[SYNCER FACTORY] Parsing SSH details...")
	sshDetails, err := parseSSHDetails(details)
	if err != nil {
		log.Printf("[SYNCER FACTORY] ERROR: Failed to parse SSH details: %v", err)
		return nil, err
	}
	log.Printf("[SYNCER FACTORY] SSH details parsed successfully - Host: %s, User: %s, Port: %d",
		sshDetails.Host, sshDetails.User, sshDetails.Port)
	return ssh.NewSSHSyncer(sshDetails, targetPath, f.timeout), nil
}

func (f *SyncerFactory) createGitSyncer(details interface{}, targetPath string) (Syncer, error) {
	log.Printf("[SYNCER FACTORY] Parsing Git details...")
	gitDetails, err := parseGitDetails(details)
	if err != nil {
		log.Printf("[SYNCER FACTORY] ERROR: Failed to parse Git details: %v", err)
		return nil, err
	}
	log.Printf("[SYNCER FACTORY] Git details parsed successfully - URL: %s, Branch: %s, Depth: %d",
		gitDetails.URL, gitDetails.Branch, gitDetails.Depth)
	return git.NewGitSyncer(gitDetails, targetPath, f.timeout), nil
}

func (f *SyncerFactory) createHTTPSyncer(details interface{}, targetPath string) (Syncer, error) {
	log.Printf("[SYNCER FACTORY] Parsing HTTP details...")
	httpDetails, err := parseHTTPDetails(details)
	if err != nil {
		log.Printf("[SYNCER FACTORY] ERROR: Failed to parse HTTP details: %v", err)
		return nil, err
	}
	log.Printf("[SYNCER FACTORY] HTTP details parsed successfully - URL: %s", httpDetails.URL)
	return http.NewHTTPSyncer(httpDetails, targetPath, f.timeout), nil
}

func (f *SyncerFactory) createS3Syncer(details interface{}, targetPath string) (Syncer, error) {
	log.Printf("[SYNCER FACTORY] Parsing S3 details...")
	s3Details, err := parseS3Details(details)
	if err != nil {
		log.Printf("[SYNCER FACTORY] ERROR: Failed to parse S3 details: %v", err)
		return nil, err
	}
	log.Printf("[SYNCER FACTORY] S3 details parsed successfully - Endpoint: %s, Bucket: %s, Path: %s",
		s3Details.EndpointURL, s3Details.BucketName, s3Details.Path)
	return s3.NewS3Syncer(s3Details, targetPath, f.timeout)
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

	if privateKey, ok := detailsMap["privateKey"].(string); ok {
		sshDetails.PrivateKey = privateKey
	}

	// Parse the path field - this is required for SSH sync
	if path, ok := detailsMap["path"].(string); ok {
		sshDetails.Path = path
	}

	// Validate that password and privateKey are not both provided
	if sshDetails.Password != "" && (sshDetails.PrivateKey != "" || sshDetails.KeyPath != "") {
		return nil, errors.New("password and privateKey/key_path cannot be provided at the same time")
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

	if username, ok := detailsMap["user"].(string); ok {
		gitDetails.User = username
	}

	if password, ok := detailsMap["password"].(string); ok {
		gitDetails.Password = password
	}

	if privateKey, ok := detailsMap["privateKey"].(string); ok {
		gitDetails.PrivateKey = privateKey
	}

	// Validate that username/password and privateKey are not both provided
	if (gitDetails.User != "" || gitDetails.Password != "") && gitDetails.PrivateKey != "" {
		return nil, errors.New("username/password and privateKey cannot be provided at the same time")
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

// parseS3Details parses S3 details from interface{}
func parseS3Details(details interface{}) (*models.S3Details, error) {
	detailsMap, ok := details.(map[string]interface{})
	if !ok {
		return nil, errors.New("S3 details must be an object")
	}

	endpointURL, ok := detailsMap["endpointUrl"].(string)
	if !ok || endpointURL == "" {
		return nil, errors.New("S3 endpoint URL is required")
	}

	bucketName, ok := detailsMap["bucketName"].(string)
	if !ok || bucketName == "" {
		return nil, errors.New("S3 bucket name is required")
	}

	path, ok := detailsMap["path"].(string)
	if !ok || path == "" {
		return nil, errors.New("S3 path is required")
	}

	accessKey, ok := detailsMap["accessKey"].(string)
	if !ok || accessKey == "" {
		return nil, errors.New("S3 access key is required")
	}

	secretKey, ok := detailsMap["secretKey"].(string)
	if !ok || secretKey == "" {
		return nil, errors.New("S3 secret key is required")
	}

	region, ok := detailsMap["region"].(string)
	if !ok || region == "" {
		return nil, errors.New("S3 region is required")
	}

	return &models.S3Details{
		EndpointURL: endpointURL,
		BucketName:  bucketName,
		Path:        path,
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		Region:      region,
	}, nil
}
