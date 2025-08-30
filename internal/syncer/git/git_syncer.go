package git

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sharedvolume/volume-syncer/internal/models"
	"github.com/sharedvolume/volume-syncer/internal/utils"
)

// GitSyncer handles git-based synchronization
type GitSyncer struct {
	details   *models.GitCloneDetails
	targetDir string
	timeout   time.Duration
}

// maskCredentials masks passwords and sensitive information in URLs and commands
func maskCredentials(text string) string {
	// Regex to match URLs with credentials: protocol://user:password@host/path
	credentialURLRegex := regexp.MustCompile(`(https?://)([^:]+):([^@]+)(@[^/\s]+)`)
	masked := credentialURLRegex.ReplaceAllString(text, "${1}${2}:***${4}")

	// Also mask any standalone passwords that might appear
	// This is a more generic approach for any password-like strings
	passwordRegex := regexp.MustCompile(`(\bpassword[=:\s]+)([^\s&]+)`)
	masked = passwordRegex.ReplaceAllString(masked, "${1}***")

	return masked
}

// maskGitCommand masks credentials in git command arguments
func maskGitCommand(args []string) []string {
	maskedArgs := make([]string, len(args))
	for i, arg := range args {
		maskedArgs[i] = maskCredentials(arg)
	}
	return maskedArgs
}

// NewGitSyncer creates a new Git syncer
func NewGitSyncer(details *models.GitCloneDetails, targetDir string, timeout time.Duration) *GitSyncer {
	return &GitSyncer{
		details:   details,
		targetDir: targetDir,
		timeout:   timeout,
	}
}

// Sync clones the repository to the target directory
func (g *GitSyncer) Sync() error {
	log.Printf("[GIT SYNC] Starting git sync: repo=%s targetDir=%s timeout=%v", g.details.URL, g.targetDir, g.timeout)
	log.Printf("[GIT SYNC] Git details - Branch: %s, Depth: %d", g.details.Branch, g.details.Depth)

	log.Printf("[GIT SYNC] Validating git configuration...")
	if err := g.validate(); err != nil {
		log.Printf("[GIT SYNC] ERROR: Validation failed: %v", err)
		return err
	}
	log.Printf("[GIT SYNC] Git configuration validation passed")

	// Ensure target directory exists
	log.Printf("[GIT SYNC] Creating target directory: %s", g.targetDir)
	if err := utils.EnsureDir(g.targetDir); err != nil {
		log.Printf("[GIT SYNC] ERROR: Failed to create target directory: %v", err)
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	log.Printf("[GIT SYNC] Target directory created successfully")

	branch := g.details.Branch
	if branch == "" {
		log.Printf("[GIT SYNC] No branch specified, will use repository's default branch")
	} else {
		log.Printf("[GIT SYNC] Using specified branch: %s", branch)
	}

	// Check if target directory exists
	gitDir := g.targetDir + "/.git"
	log.Printf("[GIT SYNC] Checking if target directory is an existing git repository...")
	if stat, err := os.Stat(g.targetDir); err == nil && stat.IsDir() {
		if _, err := os.Stat(gitDir); err == nil {
			log.Printf("[GIT SYNC] Found existing git repository, performing sync...")
			return g.syncExistingRepo(branch)
		}

		// Directory exists but is not a git repository
		log.Printf("[GIT SYNC] Target directory exists but is not a git repository")

		// Check if directory is empty
		entries, err := os.ReadDir(g.targetDir)
		if err != nil {
			log.Printf("[GIT SYNC] ERROR: Failed to read target directory: %v", err)
			return fmt.Errorf("failed to read target directory %s: %w", g.targetDir, err)
		}

		if len(entries) > 0 {
			log.Printf("[GIT SYNC] Target directory is not empty (%d entries)", len(entries))
			log.Printf("[GIT SYNC] SAFETY: Will attempt clone to temporary location first to verify operation before modifying target")
			return g.safeCloneWithReplace(branch)
		} else {
			log.Printf("[GIT SYNC] Target directory is empty, proceeding with clone")
		}
	} else {
		log.Printf("[GIT SYNC] Target directory does not exist or is not a directory")
	}

	// Do a shallow clone
	log.Printf("[GIT SYNC] Performing fresh clone...")
	return g.cloneRepo(branch)
}

// safeCloneWithReplace safely clones to a temporary location first, then replaces target
func (g *GitSyncer) safeCloneWithReplace(branch string) error {
	log.Printf("[GIT SYNC] Starting safe clone with replace for non-empty target directory")

	// Create temporary directory in the same filesystem as target
	targetParent := filepath.Dir(g.targetDir)
	tmpDir, err := os.MkdirTemp(targetParent, "volume-syncer-git-*")
	if err != nil {
		log.Printf("[GIT SYNC] ERROR: Failed to create temporary directory in %s: %v", targetParent, err)
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() {
		log.Printf("[GIT SYNC] Cleaning up temporary directory: %s", tmpDir)
		os.RemoveAll(tmpDir)
	}()

	log.Printf("[GIT SYNC] Created temporary directory for safe clone: %s", tmpDir)

	// Create a temporary syncer to clone to temp location
	tempSyncer := &GitSyncer{
		details:   g.details,
		targetDir: tmpDir,
		timeout:   g.timeout,
	}

	// Attempt clone to temporary location
	log.Printf("[GIT SYNC] Attempting clone to temporary location to verify operation before modifying target...")
	if err := tempSyncer.cloneRepo(branch); err != nil {
		log.Printf("[GIT SYNC] ERROR: Clone to temporary location failed: %v", err)
		log.Printf("[GIT SYNC] SAFETY: Target directory preserved due to clone failure")
		return fmt.Errorf("clone failed, target directory preserved: %w", err)
	}

	log.Printf("[GIT SYNC] Clone to temporary location successful, operation verified")

	// Create backup name for current target
	backupDir := g.targetDir + ".backup-" + fmt.Sprintf("%d", time.Now().Unix())

	// Rename current target to backup (this is atomic and reversible)
	log.Printf("[GIT SYNC] Backing up current target directory to: %s", backupDir)
	if err := os.Rename(g.targetDir, backupDir); err != nil {
		log.Printf("[GIT SYNC] ERROR: Failed to backup current target directory: %v", err)
		log.Printf("[GIT SYNC] SAFETY: Target directory preserved due to backup failure")
		return fmt.Errorf("failed to backup target directory, target preserved: %w", err)
	}

	// Now move temp to target location (atomic operation on same filesystem)
	log.Printf("[GIT SYNC] Moving temporary clone to target location")
	if err := os.Rename(tmpDir, g.targetDir); err != nil {
		log.Printf("[GIT SYNC] ERROR: Failed to move temporary clone to target: %v", err)
		log.Printf("[GIT SYNC] SAFETY: Restoring original target directory from backup")

		// Restore from backup
		if restoreErr := os.Rename(backupDir, g.targetDir); restoreErr != nil {
			log.Printf("[GIT SYNC] CRITICAL ERROR: Failed to restore backup, manual intervention required: %v", restoreErr)
			return fmt.Errorf("failed to move temp and failed to restore backup - target at %s, backup at %s: %w", g.targetDir, backupDir, err)
		}

		log.Printf("[GIT SYNC] Target directory successfully restored from backup")
		return fmt.Errorf("failed to move temporary clone to target, target restored: %w", err)
	}

	// Success! Remove the backup
	log.Printf("[GIT SYNC] Operation successful, removing backup directory: %s", backupDir)
	if err := os.RemoveAll(backupDir); err != nil {
		log.Printf("[GIT SYNC] WARNING: Failed to remove backup directory %s: %v", backupDir, err)
		// Don't return error here since the main operation succeeded
	}

	log.Printf("[GIT SYNC] Safe clone with replace completed successfully")
	return nil
}

// syncExistingRepo syncs an existing git repository
func (g *GitSyncer) syncExistingRepo(branch string) error {
	log.Printf("[GIT SYNC] Syncing existing repository at %s", g.targetDir)

	// Setup authentication
	cleanup, err := g.setupSSHKey()
	if err != nil {
		return err
	}
	defer cleanup()

	// Prepare authenticated URL if using username/password
	repoURL, err := g.prepareAuthenticatedURL()
	if err != nil {
		return err
	}

	// Check if the remote URL matches (compare base URL without credentials)
	log.Printf("[GIT SYNC] Checking remote URL...")
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	remoteURLBytes, err := exec.CommandContext(ctx, "git", "-C", g.targetDir, "config", "--get", "remote.origin.url").Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[GIT SYNC] ERROR: Git config command timed out after %v", g.timeout)
			return fmt.Errorf("git config command timed out after %v", g.timeout)
		}
		log.Printf("[GIT SYNC] ERROR: Failed to get remote URL: %v", err)
		return fmt.Errorf("failed to get remote URL: %w", err)
	}

	remoteURL := strings.TrimSpace(string(remoteURLBytes))
	log.Printf("[GIT SYNC] Current remote URL: %s", maskCredentials(remoteURL))
	log.Printf("[GIT SYNC] Expected base URL: %s", g.details.URL)

	// Compare base URLs (without credentials)
	if !g.urlsMatch(remoteURL, g.details.URL) {
		log.Printf("[GIT SYNC] Remote URL mismatch, need to replace with different repository")
		log.Printf("[GIT SYNC] SAFETY: Will attempt clone to temporary location first to verify operation")
		return g.safeCloneWithReplace(branch)
	}

	// Update remote URL if authentication is needed
	if g.details.User != "" && g.details.Password != "" {
		log.Printf("[GIT SYNC] Updating remote URL with username/password authentication")
		if err := g.runGitInTarget([]string{"remote", "set-url", "origin", repoURL}); err != nil {
			log.Printf("[GIT SYNC] ERROR: Failed to update remote URL: %v", err)
			return fmt.Errorf("failed to update remote URL: %w", err)
		}
	} else if g.details.PrivateKey != "" {
		log.Printf("[GIT SYNC] Using SSH authentication with private key (no URL update needed)")
	}

	log.Printf("[GIT SYNC] Remote URL matches, proceeding with sync")

	// git fetch
	log.Printf("[GIT SYNC] Fetching latest changes...")
	if err := g.runGitInTarget([]string{"fetch", "--all"}); err != nil {
		log.Printf("[GIT SYNC] ERROR: Git fetch failed: %v", err)
		return fmt.Errorf("git fetch failed: %w", err)
	}
	log.Printf("[GIT SYNC] Fetch completed successfully")

	// Force local branch to match remote
	if branch == "" {
		// If no branch specified, get the default branch
		defaultBranch, err := g.getDefaultBranch()
		if err != nil {
			log.Printf("[GIT SYNC] ERROR: Failed to get default branch: %v", err)
			return fmt.Errorf("failed to get default branch: %w", err)
		}
		branch = defaultBranch
		log.Printf("[GIT SYNC] Using default branch: %s", branch)
	}

	log.Printf("[GIT SYNC] Checking out branch %s...", branch)
	const originPrefix = "origin/"
	if err := g.runGitInTarget([]string{"checkout", "-B", branch, originPrefix + branch}); err != nil {
		// Try fallback to master if main fails
		if branch == "main" {
			log.Printf("[GIT SYNC] Branch 'main' not found, falling back to 'master'")
			branch = "master"
			if err := g.runGitInTarget([]string{"checkout", "-B", branch, originPrefix + branch}); err != nil {
				log.Printf("[GIT SYNC] ERROR: Git checkout -B master failed: %v", err)
				return fmt.Errorf("git checkout -B master failed: %w", err)
			}
		} else {
			log.Printf("[GIT SYNC] ERROR: Git checkout -B %s %s%s failed: %v", branch, originPrefix, branch, err)
			return fmt.Errorf("git checkout -B %s %s%s failed: %w", branch, originPrefix, branch, err)
		}
	}
	log.Printf("[GIT SYNC] Branch checkout completed successfully")

	// git reset --hard origin/<branch>
	log.Printf("[GIT SYNC] Resetting to origin/%s...", branch)
	if err := g.runGitInTarget([]string{"reset", "--hard", originPrefix + branch}); err != nil {
		log.Printf("[GIT SYNC] ERROR: Git reset failed: %v", err)
		return fmt.Errorf("git reset failed: %w", err)
	}
	log.Printf("[GIT SYNC] Reset completed successfully")

	// git clean -fdx (always run clean)
	log.Printf("[GIT SYNC] Cleaning untracked files...")
	if err := g.runGitInTarget([]string{"clean", "-fdx"}); err != nil {
		log.Printf("[GIT SYNC] ERROR: Git clean failed: %v", err)
		return fmt.Errorf("git clean failed: %w", err)
	}
	log.Printf("[GIT SYNC] Clean completed successfully")

	log.Printf("[GIT SYNC] Git repo synced to origin/%s", branch)
	return nil
}

// cloneRepo clones a new repository
func (g *GitSyncer) cloneRepo(branch string) error {
	log.Printf("[GIT SYNC] Starting fresh clone of repository")

	// Setup authentication
	cleanup, err := g.setupSSHKey()
	if err != nil {
		return err
	}
	defer cleanup()

	// Prepare authenticated URL if using username/password
	repoURL, err := g.prepareAuthenticatedURL()
	if err != nil {
		return err
	}

	depth := g.details.Depth
	if depth == 0 {
		depth = 1 // default to shallow clone
	}

	gitCmd := []string{"clone", "--depth", fmt.Sprintf("%d", depth)}
	log.Printf("[GIT SYNC] Using clone depth: %d", depth)

	if branch != "" {
		gitCmd = append(gitCmd, "--branch", branch)
		log.Printf("[GIT SYNC] Cloning specific branch: %s", branch)
	} else {
		log.Printf("[GIT SYNC] Cloning repository's default branch")
	}

	gitCmd = append(gitCmd, repoURL, g.targetDir)

	// Log the command appropriately based on authentication type
	if g.details.PrivateKey != "" {
		if branch != "" {
			log.Printf("[GIT SYNC] Executing git command with SSH key authentication: git clone --depth %d --branch %s [SSH_URL] %s", depth, branch, g.targetDir)
		} else {
			log.Printf("[GIT SYNC] Executing git command with SSH key authentication: git clone --depth %d [SSH_URL] %s", depth, g.targetDir)
		}
	} else if g.details.User != "" && g.details.Password != "" {
		if branch != "" {
			log.Printf("[GIT SYNC] Executing git command with username/password authentication: git clone --depth %d --branch %s [URL_WITH_CREDENTIALS] %s", depth, branch, g.targetDir)
		} else {
			log.Printf("[GIT SYNC] Executing git command with username/password authentication: git clone --depth %d [URL_WITH_CREDENTIALS] %s", depth, g.targetDir)
		}
	} else {
		// Mask credentials in git command logging
		maskedGitCmd := maskGitCommand(gitCmd)
		log.Printf("[GIT SYNC] Executing git command: git %v", maskedGitCmd)
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", gitCmd...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("[GIT SYNC] Starting clone process...")
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[GIT SYNC] ERROR: Git clone timed out after %v", g.timeout)
			return fmt.Errorf("git clone timed out after %v", g.timeout)
		}
		log.Printf("[GIT SYNC] ERROR: Git clone failed: %v", err)
		return fmt.Errorf("git clone failed: %w", err)
	}

	// If no branch was specified, log the current branch after clone
	if branch == "" {
		// Get the current branch name with timeout
		branchCtx, branchCancel := context.WithTimeout(context.Background(), g.timeout)
		defer branchCancel()

		currentBranchOutput, err := exec.CommandContext(branchCtx, "git", "-C", g.targetDir, "branch", "--show-current").Output()
		if err == nil {
			currentBranch := strings.TrimSpace(string(currentBranchOutput))
			log.Printf("[GIT SYNC] Cloned to default branch: %s", currentBranch)
		} else if branchCtx.Err() == context.DeadlineExceeded {
			log.Printf("[GIT SYNC] WARNING: Git branch command timed out after %v", g.timeout)
		} else {
			log.Printf("[GIT SYNC] WARNING: Failed to get current branch name: %v", err)
		}
	}

	log.Printf("[GIT SYNC] Git clone completed successfully: repo=%s targetDir=%s", g.details.URL, g.targetDir)
	return nil
}

// runGitInTarget runs a git command in the target directory
func (g *GitSyncer) runGitInTarget(args []string) error {
	// Mask credentials in the log output
	maskedArgs := maskGitCommand(args)
	log.Printf("[GIT SYNC] Executing in %s: git %v", g.targetDir, maskedArgs)

	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[GIT SYNC] ERROR: Git command timed out after %v", g.timeout)
			return fmt.Errorf("git command timed out after %v", g.timeout)
		}
		log.Printf("[GIT SYNC] ERROR: Git command failed: %v", err)
		return err
	}

	log.Printf("[GIT SYNC] Git command completed successfully: %v", args)
	return nil
}

// validate validates the git details
func (g *GitSyncer) validate() error {
	if g.details == nil {
		return fmt.Errorf("GitCloneDetails is required")
	}

	if g.details.URL == "" {
		return fmt.Errorf("repository URL is required")
	}

	// Check that both privateKey and username/password are not provided at the same time
	hasPrivateKey := g.details.PrivateKey != ""
	hasUsernamePassword := g.details.User != "" && g.details.Password != ""

	if hasPrivateKey && hasUsernamePassword {
		return fmt.Errorf("cannot provide both private key and username/password authentication")
	}

	// If username is provided, password must also be provided
	if g.details.User != "" && g.details.Password == "" {
		return fmt.Errorf("password is required when username is provided")
	}

	// If password is provided, username must also be provided
	if g.details.Password != "" && g.details.User == "" {
		return fmt.Errorf("username is required when password is provided")
	}

	return nil
}

// prepareAuthenticatedURL prepares the URL with authentication credentials
func (g *GitSyncer) prepareAuthenticatedURL() (string, error) {
	// If private key is provided, use SSH authentication (no URL modification needed)
	if g.details.PrivateKey != "" {
		log.Printf("[GIT SYNC] Using SSH authentication with private key")
		return g.details.URL, nil
	}

	// If username/password is provided, use HTTP authentication
	if g.details.User != "" && g.details.Password != "" {
		log.Printf("[GIT SYNC] Preparing URL with username/password authentication")

		// Parse the URL to inject credentials
		parsedURL, err := url.Parse(g.details.URL)
		if err != nil {
			return "", fmt.Errorf("failed to parse Git URL: %w", err)
		}

		// Add credentials to URL
		parsedURL.User = url.UserPassword(g.details.User, g.details.Password)
		authenticatedURL := parsedURL.String()

		// Log without showing credentials
		log.Printf("[GIT SYNC] URL prepared with credentials for user: %s", g.details.User)
		return authenticatedURL, nil
	}

	// No authentication provided
	log.Printf("[GIT SYNC] No authentication provided, using URL as-is")
	return g.details.URL, nil
}

// setupSSHKey sets up SSH key authentication if private key is provided
func (g *GitSyncer) setupSSHKey() (func(), error) {
	if g.details.PrivateKey == "" {
		// No private key provided, return empty cleanup function
		return func() { /* no cleanup needed */ }, nil
	}

	log.Printf("[GIT SYNC] Setting up SSH key authentication")

	// Decode base64 private key
	privateKeyBytes, err := base64.StdEncoding.DecodeString(g.details.PrivateKey)
	if err != nil {
		log.Printf("[GIT SYNC] ERROR: Failed to decode base64 private key: %v", err)
		return func() { /* no cleanup needed */ }, fmt.Errorf("failed to decode base64 private key: %w", err)
	}
	log.Printf("[GIT SYNC] Base64 private key decoded successfully (%d bytes)", len(privateKeyBytes))

	// Create temporary key file
	tmpKeyFile, err := g.createTempKeyFile(privateKeyBytes)
	if err != nil {
		log.Printf("[GIT SYNC] ERROR: Failed to create temporary key file: %v", err)
		return func() { /* no cleanup needed */ }, fmt.Errorf("failed to create temporary key file: %w", err)
	}
	log.Printf("[GIT SYNC] Temporary SSH key file created: %s", tmpKeyFile)

	// Setup SSH command to use the key
	sshCommand := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", tmpKeyFile)
	os.Setenv("GIT_SSH_COMMAND", sshCommand)
	log.Printf("[GIT SYNC] GIT_SSH_COMMAND set: %s", sshCommand)

	// Return cleanup function
	cleanup := func() {
		log.Printf("[GIT SYNC] Cleaning up SSH key and environment")
		os.Remove(tmpKeyFile)
		os.Unsetenv("GIT_SSH_COMMAND")
	}

	return cleanup, nil
}

// createTempKeyFile creates a temporary file for the SSH private key
func (g *GitSyncer) createTempKeyFile(privateKeyBytes []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "git_ssh_key_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary key file: %w", err)
	}

	// Write the private key to the file
	if _, err := tmpFile.Write(privateKeyBytes); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write private key to temporary file: %w", err)
	}

	tmpFile.Close()

	// Set proper permissions for the private key
	if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to set permissions on temporary key file: %w", err)
	}

	return tmpFile.Name(), nil
}

// urlsMatch compares two Git URLs to see if they refer to the same repository
func (g *GitSyncer) urlsMatch(url1, url2 string) bool {
	// Remove trailing slashes and .git suffixes for comparison
	cleanURL1 := strings.TrimSuffix(strings.TrimSuffix(url1, "/"), ".git")
	cleanURL2 := strings.TrimSuffix(strings.TrimSuffix(url2, "/"), ".git")

	// Parse URLs to compare without credentials
	parsedURL1, err1 := url.Parse(cleanURL1)
	parsedURL2, err2 := url.Parse(cleanURL2)

	if err1 != nil || err2 != nil {
		// If we can't parse URLs, do simple string comparison
		return cleanURL1 == cleanURL2
	}

	// Compare host and path without credentials
	return parsedURL1.Host == parsedURL2.Host && parsedURL1.Path == parsedURL2.Path
}

// getDefaultBranch gets the default branch from the remote repository
func (g *GitSyncer) getDefaultBranch() (string, error) {
	log.Printf("[GIT SYNC] Getting default branch from remote repository")

	// Try to get the default branch from remote HEAD with timeout
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, "git", "-C", g.targetDir, "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[GIT SYNC] ERROR: Git symbolic-ref command timed out after %v", g.timeout)
			return "", fmt.Errorf("git symbolic-ref command timed out after %v", g.timeout)
		}

		// If that fails, try to set the remote HEAD first
		log.Printf("[GIT SYNC] Failed to get remote HEAD, trying to set it")
		if err := g.runGitInTarget([]string{"remote", "set-head", "origin", "--auto"}); err != nil {
			log.Printf("[GIT SYNC] Failed to set remote HEAD, falling back to common branch names")
			// Try common branch names
			for _, branchName := range []string{"main", "master", "develop"} {
				if err := g.runGitInTarget([]string{"checkout", "-B", branchName, "origin/" + branchName}); err == nil {
					log.Printf("[GIT SYNC] Successfully checked out branch: %s", branchName)
					return branchName, nil
				}
			}
			return "", fmt.Errorf("unable to determine default branch")
		}

		// Try again after setting remote HEAD with timeout
		retryCtx, retryCancel := context.WithTimeout(context.Background(), g.timeout)
		defer retryCancel()

		output, err = exec.CommandContext(retryCtx, "git", "-C", g.targetDir, "symbolic-ref", "refs/remotes/origin/HEAD").Output()
		if err != nil {
			if retryCtx.Err() == context.DeadlineExceeded {
				log.Printf("[GIT SYNC] ERROR: Git symbolic-ref retry command timed out after %v", g.timeout)
				return "", fmt.Errorf("git symbolic-ref retry command timed out after %v", g.timeout)
			}
			return "", fmt.Errorf("failed to get default branch: %w", err)
		}
	}

	// Parse the output to get branch name
	refName := strings.TrimSpace(string(output))
	// refs/remotes/origin/main -> main
	parts := strings.Split(refName, "/")
	if len(parts) >= 4 {
		branchName := parts[len(parts)-1]
		log.Printf("[GIT SYNC] Default branch determined: %s", branchName)
		return branchName, nil
	}

	return "", fmt.Errorf("unable to parse default branch from: %s", refName)
}
