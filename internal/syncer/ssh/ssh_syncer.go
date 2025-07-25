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

		// Also mask any arguments that look like passwords
		if strings.Contains(strings.ToLower(arg), "password") && len(arg) > 8 {
			maskedArgs[i] = "***"
		} else {
			maskedArgs[i] = arg
		}
	}
	return maskedArgs
}

// SSHSyncer handles SSH-based synchronization
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
		log.Printf("[SSH SYNC] Base64 private key decoded successfully (%d bytes)", len(privateKeyBytes))

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
	// Build SSH command for rsync
	sshCmd := fmt.Sprintf("ssh -i %s -p %d -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
		keyFile, s.sshDetails.Port)

	// Build the full source string using the specified path
	fullSource := fmt.Sprintf("%s@%s:%s", s.sshDetails.User, s.sshDetails.Host, s.sshDetails.Path)

	// Build rsync arguments
	args := []string{
		"-avz",       // archive, verbose, compress
		"--delete",   // delete files that don't exist on source
		"--progress", // show progress
		"-e", sshCmd, // specify SSH command
		fullSource,         // source
		s.targetPath + "/", // target (ensure trailing slash)
	}

	return args
}
