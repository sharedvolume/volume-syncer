package http

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/sharedvolume/volume-syncer/internal/models"
	"github.com/sharedvolume/volume-syncer/internal/utils"
)

// HTTPSyncer handles HTTP download synchronization
type HTTPSyncer struct {
	details    *models.HTTPDownloadDetails
	targetPath string
	timeout    time.Duration
}

// NewHTTPSyncer creates a new HTTP syncer
func NewHTTPSyncer(details *models.HTTPDownloadDetails, targetPath string, timeout time.Duration) *HTTPSyncer {
	return &HTTPSyncer{
		details:    details,
		targetPath: targetPath,
		timeout:    timeout,
	}
}

// Sync downloads the file from the URL to the target path
func (h *HTTPSyncer) Sync() error {
	log.Printf("[HTTP SYNC] Starting HTTP download from %s to %s", h.details.URL, h.targetPath)
	log.Printf("[HTTP SYNC] Timeout configured: %v", h.timeout)

	// Ensure the target directory exists
	log.Printf("[HTTP SYNC] Creating target directory: %s", h.targetPath)
	if err := utils.EnsureDir(h.targetPath); err != nil {
		log.Printf("[HTTP SYNC] ERROR: Failed to create target directory: %v", err)
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	log.Printf("[HTTP SYNC] Target directory created successfully")

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	log.Printf("[HTTP SYNC] Creating HTTP request...")
	req, err := http.NewRequestWithContext(ctx, "GET", h.details.URL, nil)
	if err != nil {
		log.Printf("[HTTP SYNC] ERROR: Failed to create HTTP request: %v", err)
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")
	log.Printf("[HTTP SYNC] HTTP request created with User-Agent header")

	client := &http.Client{}
	log.Printf("[HTTP SYNC] Sending HTTP request...")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[HTTP SYNC] ERROR: Failed to download file: %v", err)
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[HTTP SYNC] HTTP response received - Status: %s", resp.Status)
	log.Printf("[HTTP SYNC] Response headers - Content-Type: %s, Content-Length: %s",
		resp.Header.Get("Content-Type"), resp.Header.Get("Content-Length"))

	if resp.StatusCode != http.StatusOK {
		log.Printf("[HTTP SYNC] ERROR: HTTP request failed with status: %s", resp.Status)
		return fmt.Errorf("HTTP request failed: %s", resp.Status)
	}

	// Extract filename from URL
	urlPath := req.URL.Path
	filename := path.Base(urlPath)
	if filename == "." || filename == "/" || filename == "" {
		filename = "downloaded_file"
	}
	log.Printf("[HTTP SYNC] Initial filename from URL: %s", filename)

	// If Content-Disposition header is present, prefer that filename
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		log.Printf("[HTTP SYNC] Content-Disposition header found: %s", cd)
		if idx := strings.Index(cd, "filename="); idx != -1 {
			fn := cd[idx+len("filename="):]
			fn = strings.Trim(fn, "\"'")
			if fn != "" {
				filename = fn
				log.Printf("[HTTP SYNC] Using filename from Content-Disposition: %s", filename)
			}
		}
	}

	outPath := path.Join(h.targetPath, filename)
	log.Printf("[HTTP SYNC] Creating output file: %s", outPath)
	out, err := os.Create(outPath)
	if err != nil {
		log.Printf("[HTTP SYNC] ERROR: Failed to create target file: %v", err)
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer out.Close()

	log.Printf("[HTTP SYNC] Starting file download...")
	bytesWritten, err := io.Copy(out, resp.Body)
	if err != nil {
		log.Printf("[HTTP SYNC] ERROR: Failed to write file: %v", err)
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("[HTTP SYNC] Download completed successfully")
	log.Printf("[HTTP SYNC] File saved: %s (%d bytes)", outPath, bytesWritten)
	return nil
}
