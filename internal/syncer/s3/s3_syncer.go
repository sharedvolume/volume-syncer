package s3

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sharedvolume/volume-syncer/internal/models"
	"github.com/sharedvolume/volume-syncer/internal/utils"
)

// S3Syncer handles S3 synchronization
type S3Syncer struct {
	details    *models.S3Details
	targetPath string
	timeout    time.Duration
	session    *session.Session
	s3Client   *s3.S3
	downloader *s3manager.Downloader
}

// NewS3Syncer creates a new S3 syncer
func NewS3Syncer(details *models.S3Details, targetPath string, timeout time.Duration) (*S3Syncer, error) {
	log.Printf("[S3 SYNC] Initializing S3 syncer")
	log.Printf("[S3 SYNC] Endpoint: %s", details.EndpointURL)
	log.Printf("[S3 SYNC] Bucket: %s", details.BucketName)
	log.Printf("[S3 SYNC] Path: %s", details.Path)
	log.Printf("[S3 SYNC] Region: %s", details.Region)
	log.Printf("[S3 SYNC] Target Path: %s", targetPath)
	log.Printf("[S3 SYNC] Timeout: %v", timeout)

	// Determine if this is AWS S3 or S3-compatible service
	isAWSS3 := strings.Contains(details.EndpointURL, "amazonaws.com")

	// Auto-detect path style preference
	forcePathStyle := true // Default to path style for compatibility
	if details.ForcePathStyle != nil {
		forcePathStyle = *details.ForcePathStyle
		log.Printf("[S3 SYNC] Using explicit forcePathStyle setting: %v", forcePathStyle)
	} else if isAWSS3 {
		forcePathStyle = false // AWS S3 prefers virtual-hosted style
		log.Printf("[S3 SYNC] Detected AWS S3, using virtual-hosted style")
	} else {
		log.Printf("[S3 SYNC] Detected S3-compatible service, using path style")
	}

	// Auto-detect SSL preference
	disableSSL := false
	if details.DisableSSL != nil {
		disableSSL = *details.DisableSSL
		log.Printf("[S3 SYNC] Using explicit SSL setting - disabled: %v", disableSSL)
	} else if strings.HasPrefix(details.EndpointURL, "http://") {
		disableSSL = true
		log.Printf("[S3 SYNC] Detected HTTP endpoint, disabling SSL")
	} else {
		log.Printf("[S3 SYNC] Using SSL (HTTPS)")
	}

	// Create AWS session
	log.Printf("[S3 SYNC] Creating AWS session...")
	config := &aws.Config{
		Region:           aws.String(details.Region),
		Endpoint:         aws.String(details.EndpointURL),
		Credentials:      credentials.NewStaticCredentials(details.AccessKey, details.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(forcePathStyle),
		DisableSSL:       aws.Bool(disableSSL),
	}

	// Additional settings for better compatibility
	if !isAWSS3 {
		// For S3-compatible services, disable SSL certificate verification for self-signed certs
		// This is common in development/private cloud environments
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		log.Printf("[S3 SYNC] Configured for S3-compatible service with relaxed SSL verification")
	}

	sess, err := session.NewSession(config)
	if err != nil {
		log.Printf("[S3 SYNC] ERROR: Failed to create AWS session: %v", err)
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}
	log.Printf("[S3 SYNC] AWS session created successfully")

	s3Client := s3.New(sess)
	downloader := s3manager.NewDownloader(sess)

	// Test the connection to ensure compatibility
	syncer := &S3Syncer{
		details:    details,
		targetPath: targetPath,
		timeout:    timeout,
		session:    sess,
		s3Client:   s3Client,
		downloader: downloader,
	}

	log.Printf("[S3 SYNC] Testing S3 connection...")
	if err := syncer.testConnection(); err != nil {
		log.Printf("[S3 SYNC] WARNING: Initial connection test failed: %v", err)

		// If it's not AWS S3 and we failed, try the opposite path style
		if !isAWSS3 {
			log.Printf("[S3 SYNC] Retrying with virtual-hosted style...")
			config.S3ForcePathStyle = aws.Bool(false)

			sess, err = session.NewSession(config)
			if err != nil {
				log.Printf("[S3 SYNC] ERROR: Failed to create fallback AWS session: %v", err)
				return nil, fmt.Errorf("failed to create fallback AWS session: %w", err)
			}

			s3Client = s3.New(sess)
			downloader = s3manager.NewDownloader(sess)
			syncer.session = sess
			syncer.s3Client = s3Client
			syncer.downloader = downloader

			if err := syncer.testConnection(); err != nil {
				log.Printf("[S3 SYNC] ERROR: Both path styles failed: %v", err)
				return nil, fmt.Errorf("failed to establish S3 connection with both path styles: %w", err)
			}
			log.Printf("[S3 SYNC] Successfully connected with virtual-hosted style")
		} else {
			return nil, fmt.Errorf("failed to connect to AWS S3: %w", err)
		}
	} else {
		log.Printf("[S3 SYNC] S3 connection test successful")
	}

	log.Printf("[S3 SYNC] S3 client and downloader initialized")

	return syncer, nil
}

// testConnection tests the S3 connection by attempting to list bucket contents
func (s *S3Syncer) testConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to list just one object to test connectivity
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.details.BucketName),
		MaxKeys: aws.Int64(1),
	}

	_, err := s.s3Client.ListObjectsV2WithContext(ctx, input)
	return err
}

// Sync synchronizes data from S3 bucket to local target path
func (s *S3Syncer) Sync() error {
	log.Printf("[S3 SYNC] Starting S3 sync from s3://%s/%s to %s", s.details.BucketName, s.details.Path, s.targetPath)
	log.Printf("[S3 SYNC] Sync timeout: %v", s.timeout)

	// Create context with timeout for all S3 operations
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Ensure target directory exists
	log.Printf("[S3 SYNC] Creating target directory: %s", s.targetPath)
	if err := utils.EnsureDir(s.targetPath); err != nil {
		log.Printf("[S3 SYNC] ERROR: Failed to create target directory: %v", err)
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	log.Printf("[S3 SYNC] Target directory created successfully")

	// List objects in the bucket with the given prefix
	log.Printf("[S3 SYNC] Listing objects in bucket with prefix: %s", s.details.Path)
	objects, err := s.listObjects(ctx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[S3 SYNC] ERROR: S3 listing operation timed out after %v", s.timeout)
			return fmt.Errorf("S3 listing operation timed out after %v", s.timeout)
		}
		log.Printf("[S3 SYNC] ERROR: Failed to list S3 objects: %v", err)
		return fmt.Errorf("failed to list S3 objects: %w", err)
	}

	if len(objects) == 0 {
		log.Printf("[S3 SYNC] No objects found in s3://%s/%s", s.details.BucketName, s.details.Path)
		return nil
	}

	log.Printf("[S3 SYNC] Found %d objects to sync", len(objects))

	// Download each object
	for i, obj := range objects {
		log.Printf("[S3 SYNC] Processing object %d/%d: %s", i+1, len(objects), *obj.Key)
		if err := s.downloadObject(ctx, obj); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("[S3 SYNC] ERROR: S3 download operation timed out after %v", s.timeout)
				return fmt.Errorf("S3 download operation timed out after %v", s.timeout)
			}
			log.Printf("[S3 SYNC] ERROR: Failed to download object %s: %v", *obj.Key, err)
			return fmt.Errorf("failed to download object %s: %w", *obj.Key, err)
		}
	}

	log.Printf("[S3 SYNC] Successfully synced %d objects", len(objects))
	return nil
}

// listObjects lists all objects in the bucket with the given prefix
func (s *S3Syncer) listObjects(ctx context.Context) ([]*s3.Object, error) {
	log.Printf("[S3 SYNC] Starting object listing operation")
	var objects []*s3.Object

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.details.BucketName),
		Prefix: aws.String(s.details.Path),
	}

	log.Printf("[S3 SYNC] Listing objects with prefix: %s", s.details.Path)
	pageNum := 0
	err := s.s3Client.ListObjectsV2PagesWithContext(ctx, input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		pageNum++
		log.Printf("[S3 SYNC] Processing page %d (last page: %v)", pageNum, lastPage)

		for _, obj := range page.Contents {
			// Skip directories (objects ending with /)
			if !strings.HasSuffix(*obj.Key, "/") {
				objects = append(objects, obj)
				log.Printf("[S3 SYNC] Added object: %s (size: %d bytes)", *obj.Key, *obj.Size)
			} else {
				log.Printf("[S3 SYNC] Skipping directory: %s", *obj.Key)
			}
		}
		return !lastPage
	})

	if err != nil {
		log.Printf("[S3 SYNC] ERROR: Failed to list objects: %v", err)
		return nil, err
	}

	log.Printf("[S3 SYNC] Object listing completed - found %d objects across %d pages", len(objects), pageNum)
	return objects, nil
}

// downloadObject downloads a single object from S3
func (s *S3Syncer) downloadObject(ctx context.Context, obj *s3.Object) error {
	log.Printf("[S3 SYNC] Starting download of object: %s", *obj.Key)

	// Calculate relative path by removing the prefix
	relativePath := strings.TrimPrefix(*obj.Key, s.details.Path)
	if relativePath == "" {
		relativePath = filepath.Base(*obj.Key)
	}
	log.Printf("[S3 SYNC] Relative path: %s", relativePath)

	// Create the full local path
	localPath := filepath.Join(s.targetPath, relativePath)
	log.Printf("[S3 SYNC] Local path: %s", localPath)

	// Ensure the directory exists for the file
	log.Printf("[S3 SYNC] Creating directory for file: %s", filepath.Dir(localPath))
	if err := utils.EnsureDir(filepath.Dir(localPath)); err != nil {
		log.Printf("[S3 SYNC] ERROR: Failed to create directory for %s: %v", localPath, err)
		return fmt.Errorf("failed to create directory for %s: %w", localPath, err)
	}

	// Create the local file
	log.Printf("[S3 SYNC] Creating local file: %s", localPath)
	file, err := os.Create(localPath)
	if err != nil {
		log.Printf("[S3 SYNC] ERROR: Failed to create local file %s: %v", localPath, err)
		return fmt.Errorf("failed to create local file %s: %w", localPath, err)
	}
	defer file.Close()

	// Download the object with context
	log.Printf("[S3 SYNC] Downloading s3://%s/%s -> %s", s.details.BucketName, *obj.Key, localPath)

	bytesWritten, err := s.downloader.DownloadWithContext(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(s.details.BucketName),
		Key:    obj.Key,
	})

	if err != nil {
		// Clean up the file if download failed
		log.Printf("[S3 SYNC] ERROR: Download failed, cleaning up file: %s", localPath)
		os.Remove(localPath)
		log.Printf("[S3 SYNC] ERROR: Failed to download object: %v", err)
		return fmt.Errorf("failed to download object: %w", err)
	}

	log.Printf("[S3 SYNC] Successfully downloaded %s (%d bytes written, %d bytes expected)", *obj.Key, bytesWritten, *obj.Size)
	return nil
}
