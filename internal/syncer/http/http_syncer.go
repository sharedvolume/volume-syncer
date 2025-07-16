package http

import (
	"context"
	"fmt"
	"io"
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
	// Ensure the target directory exists
	if err := utils.EnsureDir(h.targetPath); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", h.details.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed: %s", resp.Status)
	}

	// Extract filename from URL
	urlPath := req.URL.Path
	filename := path.Base(urlPath)
	if filename == "." || filename == "/" || filename == "" {
		filename = "downloaded_file"
	}

	// If Content-Disposition header is present, prefer that filename
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if idx := strings.Index(cd, "filename="); idx != -1 {
			fn := cd[idx+len("filename="):]
			fn = strings.Trim(fn, "\"'")
			if fn != "" {
				filename = fn
			}
		}
	}

	outPath := path.Join(h.targetPath, filename)
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
