package syncer

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHSyncer handles SSH-based synchronization
type SSHSyncer struct {
	sshDetails *SSHDetails
	targetPath string
	timeout    time.Duration
}

// NewSSHSyncer creates a new SSH syncer
func NewSSHSyncer(sshDetails *SSHDetails, targetPath string, timeout time.Duration) *SSHSyncer {
	return &SSHSyncer{
		sshDetails: sshDetails,
		targetPath: targetPath,
		timeout:    timeout,
	}
}

// Sync performs the synchronization using rsync over SSH
func (s *SSHSyncer) Sync() error {
	log.Printf("Starting SSH sync from %s@%s:%s to %s", s.sshDetails.Username, s.sshDetails.Host, s.sshDetails.Path, s.targetPath)

	// Decode private key
	privateKeyBytes, err := base64.StdEncoding.DecodeString(s.sshDetails.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to decode private key: %w", err)
	}

	// Create temporary file for private key
	tmpKeyFile, err := s.createTempKeyFile(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to create temporary key file: %w", err)
	}
	defer os.Remove(tmpKeyFile)

	// Test SSH connection first
	if err := s.testSSHConnection(privateKeyBytes); err != nil {
		return fmt.Errorf("SSH connection test failed: %w", err)
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(s.targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Build rsync command
	rsyncCmd := s.buildRsyncCommand(tmpKeyFile)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Execute rsync command
	cmd := exec.CommandContext(ctx, "rsync", rsyncCmd...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Executing rsync command: %v", cmd.Args)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("sync operation timed out after %v", s.timeout)
		}
		return fmt.Errorf("rsync failed: %w", err)
	}

	log.Printf("SSH sync completed successfully")
	return nil
}

// testSSHConnection tests the SSH connection
func (s *SSHSyncer) testSSHConnection(privateKeyBytes []byte) error {
	// Parse private key
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User: s.sshDetails.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
		Timeout:         10 * time.Second,
	}

	// Connect to SSH server
	addr := fmt.Sprintf("%s:%d", s.sshDetails.Host, s.sshDetails.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %w", err)
	}
	defer client.Close()

	// Create session to test connection
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Run a simple command to verify connection
	if err := session.Run("echo 'connection test'"); err != nil {
		return fmt.Errorf("SSH connection test command failed: %w", err)
	}

	return nil
}

// createTempKeyFile creates a temporary file with the private key
func (s *SSHSyncer) createTempKeyFile(privateKeyBytes []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "ssh_key_*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Set restrictive permissions
	if err := tmpFile.Chmod(0600); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	// Write private key
	if _, err := tmpFile.Write(privateKeyBytes); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// buildRsyncCommand builds the rsync command arguments
func (s *SSHSyncer) buildRsyncCommand(keyFile string) []string {
	// Log detailed source information
	log.Printf("=== SSH SOURCE DETAILS ===")
	log.Printf("Host: %s", s.sshDetails.Host)
	log.Printf("Port: %d", s.sshDetails.Port)
	log.Printf("Username: %s", s.sshDetails.Username)
	log.Printf("Original Path: %s", s.sshDetails.Path)

	// Build SSH command for rsync
	sshCmd := fmt.Sprintf("ssh -i %s -p %d -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
		keyFile, s.sshDetails.Port)
	log.Printf("SSH Command: %s", sshCmd)

	// Use the source path from SSH details
	sourcePath := s.sshDetails.Path
	if sourcePath == "" {
		log.Printf("ERROR: Source path is empty!")
		return nil // or return an error - path is required
	}

	// Ensure the path is absolute and properly formatted
	if sourcePath[0] != '/' {
		sourcePath = "/" + sourcePath
		log.Printf("Converted relative to absolute path: %s", sourcePath)
	}

	// Add trailing slash to sync contents of the directory, not the directory itself
	if sourcePath[len(sourcePath)-1] != '/' {
		sourcePath += "/"
		log.Printf("Added trailing slash to source path: %s", sourcePath)
	}

	// Log target information
	log.Printf("=== TARGET DETAILS ===")
	log.Printf("Target Path: %s", s.targetPath)
	log.Printf("Target Path (with slash): %s", s.targetPath+"/")

	// Build the full source string
	fullSource := fmt.Sprintf("%s@%s:%s", s.sshDetails.Username, s.sshDetails.Host, sourcePath)
	log.Printf("=== RSYNC COMMAND DETAILS ===")
	log.Printf("Full Source: %s", fullSource)
	log.Printf("Full Target: %s", s.targetPath+"/")

	// Build rsync arguments
	args := []string{
		"-avz",       // archive, verbose, compress
		"--delete",   // delete files that don't exist on source
		"--progress", // show progress
		"-e", sshCmd, // specify SSH command
		fullSource,         // source with path
		s.targetPath + "/", // target (ensure trailing slash)
	}

	log.Printf("Complete rsync arguments: %v", args)
	log.Printf("=============================")

	return args
}
