package ssh

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sharedvolume/volume-syncer/internal/models"
	"github.com/sharedvolume/volume-syncer/internal/utils"
	"golang.org/x/crypto/ssh"
)

const (
	errSSHConnTestFailedFmt = "SSH connection test failed: %w"
	logSSHConnTestFailed    = "[SSH SYNC] ERROR: SSH connection test failed: %v"
	logSSHConnTestSuccess   = "[SSH SYNC] SSH connection test successful"
)

// maskSSHCredentials masks passwords and sensitive information in SSH commands
func maskSSHCredentials(args []string) []string {
	maskedArgs := make([]string, len(args))
	for i, arg := range args {
		// Mask SSH URLs that might contain passwords: user:password@host
		credentialRegex := regexp.MustCompile(`([^:@]+):([^@]+)@`)
		maskedArgs[i] = credentialRegex.ReplaceAllString(arg, "${1}:***@")

		// Mask sshpass password arguments: sshpass -p 'password'
		sshpassRegex := regexp.MustCompile(`sshpass -p '([^']+)'`)
		maskedArgs[i] = sshpassRegex.ReplaceAllString(maskedArgs[i], "sshpass -p '***'")

		// Also mask any arguments that look like passwords
		if strings.Contains(strings.ToLower(arg), "password") && len(arg) > 8 {
			maskedArgs[i] = "***"
		}
	}
	return maskedArgs
} // SSHSyncer handles SSH-based synchronization
type SSHSyncer struct {
	sshDetails *models.SSHDetails
	targetPath string
	timeout    time.Duration
}

// NewSSHSyncer creates a new SSH syncer
func NewSSHSyncer(sshDetails *models.SSHDetails, targetPath string, timeout time.Duration) *SSHSyncer {
	return &SSHSyncer{
		sshDetails: sshDetails,
		targetPath: targetPath,
		timeout:    timeout,
	}
}

// Sync performs the synchronization using rsync over SSH
func (s *SSHSyncer) Sync() error {
	log.Printf("[SSH SYNC] Starting SSH sync from %s@%s:%d to %s", s.sshDetails.User, s.sshDetails.Host, s.sshDetails.Port, s.targetPath)
	log.Printf("[SSH SYNC] SSH Details - Host: %s, Port: %d, User: %s, Path: '%s'", s.sshDetails.Host, s.sshDetails.Port, s.sshDetails.User, s.sshDetails.Path)
	log.Printf("[SSH SYNC] Timeout configured: %v", s.timeout)

	// Ensure target directory exists
	log.Printf("[SSH SYNC] Creating target directory: %s", s.targetPath)
	if err := utils.EnsureDir(s.targetPath); err != nil {
		log.Printf("[SSH SYNC] ERROR: Failed to create target directory: %v", err)
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	log.Printf("[SSH SYNC] Target directory created successfully")

	var tmpKeyFile string
	var privateKeyBytes []byte
	var err error

	// If private key from file is provided, use key auth
	if s.sshDetails.KeyPath != "" {
		log.Printf("[SSH SYNC] Using private key authentication from file: %s", s.sshDetails.KeyPath)
		privateKeyBytes, err = os.ReadFile(s.sshDetails.KeyPath)
		if err != nil {
			log.Printf("[SSH SYNC] ERROR: Failed to read private key file: %v", err)
			return fmt.Errorf("failed to read private key file: %w", err)
		}

		// Ensure the key ends with a newline (required for SSH key files)
		keyStr := strings.TrimSpace(string(privateKeyBytes))
		if !strings.HasSuffix(keyStr, "\n") {
			keyStr += "\n"
		}
		privateKeyBytes = []byte(keyStr)

		log.Printf("[SSH SYNC] Private key loaded successfully (%d bytes)", len(privateKeyBytes))

		log.Printf("[SSH SYNC] Creating temporary key file for rsync")
		tmpKeyFile, err = s.createTempKeyFile(privateKeyBytes)
		if err != nil {
			log.Printf("[SSH SYNC] ERROR: Failed to create temporary key file: %v", err)
			return fmt.Errorf("failed to create temporary key file: %w", err)
		}
		defer func() {
			log.Printf("[SSH SYNC] Cleaning up temporary key file: %s", tmpKeyFile)
			os.Remove(tmpKeyFile)
		}()
		log.Printf("[SSH SYNC] Temporary key file created: %s", tmpKeyFile)

		// Test SSH connection with key
		log.Printf("[SSH SYNC] Testing SSH connection with private key...")
		if err := s.testSSHConnection(privateKeyBytes, ""); err != nil {
			log.Printf(logSSHConnTestFailed, err)
			return fmt.Errorf(errSSHConnTestFailedFmt, err)
		}
		log.Printf(logSSHConnTestSuccess)
	} else if s.sshDetails.PrivateKey != "" {
		log.Printf("[SSH SYNC] Using private key authentication from base64 encoded string")

		// Decode base64 private key
		privateKeyBytes, err = base64.StdEncoding.DecodeString(s.sshDetails.PrivateKey)
		if err != nil {
			log.Printf("[SSH SYNC] ERROR: Failed to decode base64 private key: %v", err)
			return fmt.Errorf("failed to decode base64 private key: %w", err)
		}

		// Trim whitespace and empty lines from the decoded key
		keyStr := strings.TrimSpace(string(privateKeyBytes))

		// Ensure the key ends with a newline (required for SSH key files)
		if !strings.HasSuffix(keyStr, "\n") {
			keyStr += "\n"
		}

		privateKeyBytes = []byte(keyStr)
		log.Printf("[SSH SYNC] Base64 private key decoded and trimmed successfully (%d bytes)", len(privateKeyBytes))

		// Debug: Check if the decoded key looks correct
		log.Printf("[SSH SYNC] Key starts with: %s", keyStr[:min(50, len(keyStr))])
		log.Printf("[SSH SYNC] Key ends with: %s", keyStr[max(0, len(keyStr)-50):])
		if !strings.Contains(keyStr, "BEGIN OPENSSH PRIVATE KEY") {
			log.Printf("[SSH SYNC] WARNING: Decoded key doesn't contain expected OpenSSH header")
		}
		if !strings.Contains(keyStr, "END OPENSSH PRIVATE KEY") {
			log.Printf("[SSH SYNC] WARNING: Decoded key doesn't contain expected OpenSSH footer")
		}

		log.Printf("[SSH SYNC] Creating temporary key file for rsync")
		tmpKeyFile, err = s.createTempKeyFile(privateKeyBytes)
		if err != nil {
			log.Printf("[SSH SYNC] ERROR: Failed to create temporary key file: %v", err)
			return fmt.Errorf("failed to create temporary key file: %w", err)
		}
		defer func() {
			log.Printf("[SSH SYNC] Cleaning up temporary key file: %s", tmpKeyFile)
			os.Remove(tmpKeyFile)
		}()
		log.Printf("[SSH SYNC] Temporary key file created: %s", tmpKeyFile)

		// Test SSH connection with key
		log.Printf("[SSH SYNC] Testing SSH connection with private key...")
		if err := s.testSSHConnection(privateKeyBytes, ""); err != nil {
			log.Printf(logSSHConnTestFailed, err)
			return fmt.Errorf(errSSHConnTestFailedFmt, err)
		}
		log.Printf(logSSHConnTestSuccess)
	} else if s.sshDetails.Password != "" {
		log.Printf("[SSH SYNC] Using password authentication")

		// Check if sshpass is available
		if _, err := exec.LookPath("sshpass"); err != nil {
			log.Printf("[SSH SYNC] ERROR: Password authentication requires 'sshpass' utility, but it's not installed")
			log.Printf("[SSH SYNC] Please install sshpass or use SSH key authentication instead")
			return fmt.Errorf("password authentication requires 'sshpass' utility, but it's not available. Please install sshpass or use SSH key authentication")
		}

		// Test SSH connection with password
		log.Printf("[SSH SYNC] Testing SSH connection with password...")
		if err := s.testSSHConnection(nil, s.sshDetails.Password); err != nil {
			log.Printf(logSSHConnTestFailed, err)
			return fmt.Errorf(errSSHConnTestFailedFmt, err)
		}
		log.Printf(logSSHConnTestSuccess)
	} else {
		log.Printf("[SSH SYNC] Using no authentication (public key from ssh-agent)")
		// Test SSH connection with no auth
		log.Printf("[SSH SYNC] Testing SSH connection...")
		if err := s.testSSHConnection(nil, ""); err != nil {
			log.Printf(logSSHConnTestFailed, err)
			return fmt.Errorf(errSSHConnTestFailedFmt, err)
		}
		log.Printf(logSSHConnTestSuccess)
	}

	// Build rsync command
	log.Printf("[SSH SYNC] Building rsync command...")

	// Check if ssh is available and log its location
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		log.Printf("[SSH SYNC] WARNING: ssh command not found in PATH: %v", err)
		log.Printf("[SSH SYNC] Checking common locations...")
		for _, path := range []string{"/usr/bin/ssh", "/bin/ssh", "/usr/local/bin/ssh"} {
			if _, err := os.Stat(path); err == nil {
				log.Printf("[SSH SYNC] Found ssh at: %s", path)
				sshPath = path
				break
			}
		}
		if sshPath == "" {
			log.Printf("[SSH SYNC] ERROR: ssh command not found in any common location")
			return fmt.Errorf("ssh command not found")
		}
	} else {
		log.Printf("[SSH SYNC] Found ssh command at: %s", sshPath)
	}

	rsyncCmd := s.buildRsyncCommand(tmpKeyFile)
	log.Printf("[SSH SYNC] Rsync command built with %d arguments", len(rsyncCmd))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Execute rsync command
	cmd := exec.CommandContext(ctx, "rsync", rsyncCmd...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Mask credentials in the command logging
	maskedArgs := maskSSHCredentials(cmd.Args)
	log.Printf("[SSH SYNC] Executing rsync command: %v", maskedArgs)
	log.Printf("[SSH SYNC] Starting data transfer...")

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[SSH SYNC] ERROR: Sync operation timed out after %v", s.timeout)
			return fmt.Errorf("sync operation timed out after %v", s.timeout)
		}
		log.Printf("[SSH SYNC] ERROR: Rsync failed: %v", err)
		return fmt.Errorf("rsync failed: %w", err)
	}

	log.Printf("[SSH SYNC] Data transfer completed successfully")
	log.Printf("[SSH SYNC] SSH sync completed successfully")
	return nil
}

// testSSHConnection tests the SSH connection
func (s *SSHSyncer) testSSHConnection(privateKeyBytes []byte, password string) error {
	var authMethods []ssh.AuthMethod
	if len(privateKeyBytes) > 0 {
		signer, err := ssh.ParsePrivateKey(privateKeyBytes)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	// If no auth methods, try empty list (let SSH try agent, etc.)
	config := &ssh.ClientConfig{
		User:            s.sshDetails.User,
		Auth:            authMethods,
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
	// Detect SSH path
	sshPath := "ssh" // default fallback
	if detectedPath, err := exec.LookPath("ssh"); err == nil {
		sshPath = detectedPath
	} else {
		// Check common locations
		for _, path := range []string{"/usr/bin/ssh", "/bin/ssh", "/usr/local/bin/ssh"} {
			if _, err := os.Stat(path); err == nil {
				sshPath = path
				break
			}
		}
	}

	log.Printf("[SSH SYNC] Using SSH path: %s", sshPath)

	// Build SSH command for rsync
	var sshCmd string
	if keyFile != "" {
		// Use private key authentication with detected ssh path
		sshCmd = fmt.Sprintf("%s -i %s -p %d -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
			sshPath, keyFile, s.sshDetails.Port)
	} else if s.sshDetails.Password != "" {
		// Use password authentication with sshpass (if available)
		// Escape single quotes in password to prevent shell injection
		escapedPassword := strings.ReplaceAll(s.sshDetails.Password, "'", "'\"'\"'")

		// Detect sshpass path
		sshpassPath := "sshpass"
		if detectedPath, err := exec.LookPath("sshpass"); err == nil {
			sshpassPath = detectedPath
		}

		sshCmd = fmt.Sprintf("%s -p '%s' %s -p %d -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
			sshpassPath, escapedPassword, sshPath, s.sshDetails.Port)
	} else {
		// Use ssh-agent or default SSH authentication
		sshCmd = fmt.Sprintf("%s -p %d -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
			sshPath, s.sshDetails.Port)
	}

	// Build the full source string using the specified path
	log.Printf("[SSH SYNC] Building source path - User: %s, Host: %s, Path: '%s'", s.sshDetails.User, s.sshDetails.Host, s.sshDetails.Path)

	// Add trailing slash to source path to copy contents of directory, not the directory itself
	sourcePath := s.sshDetails.Path
	if !strings.HasSuffix(sourcePath, "/") {
		sourcePath += "/"
	}

	fullSource := fmt.Sprintf("%s@%s:%s", s.sshDetails.User, s.sshDetails.Host, sourcePath)
	log.Printf("[SSH SYNC] Full source string: %s", fullSource)

	// Build rsync arguments
	args := []string{
		"-avz",       // archive, verbose, compress
		"--delete",   // delete files that don't exist on source
		"--progress", // show progress
		"-e", sshCmd, // specify SSH command
		fullSource,         // source
		s.targetPath + "/", // target (ensure trailing slash)
	}

	// Log the command for debugging
	log.Printf("[SSH SYNC] SSH command for rsync: %s", sshCmd)

	return args
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
