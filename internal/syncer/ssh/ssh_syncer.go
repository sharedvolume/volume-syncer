package ssh

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/sharedvolume/volume-syncer/internal/models"
	"github.com/sharedvolume/volume-syncer/internal/utils"
	"golang.org/x/crypto/ssh"
)

const errSSHConnTestFailedFmt = "SSH connection test failed: %w"

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
	log.Printf("Starting SSH sync from %s@%s to %s", s.sshDetails.User, s.sshDetails.Host, s.targetPath)

	// Ensure target directory exists
	if err := utils.EnsureDir(s.targetPath); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	var tmpKeyFile string
	var privateKeyBytes []byte
	var err error

	// If private key is provided, use key auth
	if s.sshDetails.KeyPath != "" {
		privateKeyBytes, err = os.ReadFile(s.sshDetails.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key file: %w", err)
		}
		tmpKeyFile, err = s.createTempKeyFile(privateKeyBytes)
		if err != nil {
			return fmt.Errorf("failed to create temporary key file: %w", err)
		}
		defer os.Remove(tmpKeyFile)
		// Test SSH connection with key
		if err := s.testSSHConnection(privateKeyBytes, ""); err != nil {
			return fmt.Errorf(errSSHConnTestFailedFmt, err)
		}
	} else if s.sshDetails.Password != "" {
		// Test SSH connection with password
		if err := s.testSSHConnection(nil, s.sshDetails.Password); err != nil {
			return fmt.Errorf(errSSHConnTestFailedFmt, err)
		}
	} else {
		// Test SSH connection with no auth
		if err := s.testSSHConnection(nil, ""); err != nil {
			return fmt.Errorf(errSSHConnTestFailedFmt, err)
		}
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

	// Build the full source string (assuming we want to sync the home directory)
	fullSource := fmt.Sprintf("%s@%s:~/", s.sshDetails.User, s.sshDetails.Host)

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
