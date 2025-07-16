package git

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
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

	// If targetDir exists and is a git repo, do fetch/reset/clean
	gitDir := g.targetDir + "/.git"
	log.Printf("[GIT SYNC] Checking if target directory is an existing git repository...")
	if stat, err := os.Stat(g.targetDir); err == nil && stat.IsDir() {
		if _, err := os.Stat(gitDir); err == nil {
			log.Printf("[GIT SYNC] Found existing git repository, performing sync...")
			return g.syncExistingRepo(branch)
		}
		log.Printf("[GIT SYNC] Target directory exists but is not a git repository")
	} else {
		log.Printf("[GIT SYNC] Target directory does not exist or is not a directory")
	}

	// Otherwise, do a shallow clone
	log.Printf("[GIT SYNC] Performing fresh clone...")
	return g.cloneRepo(branch)
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
	remoteURLBytes, err := exec.Command("git", "-C", g.targetDir, "config", "--get", "remote.origin.url").Output()
	if err != nil {
		log.Printf("[GIT SYNC] ERROR: Failed to get remote URL: %v", err)
		return fmt.Errorf("failed to get remote URL: %w", err)
	}

	remoteURL := strings.TrimSpace(string(remoteURLBytes))
	log.Printf("[GIT SYNC] Current remote URL: %s", remoteURL)
	log.Printf("[GIT SYNC] Expected base URL: %s", g.details.URL)

	// Compare base URLs (without credentials)
	if !g.urlsMatch(remoteURL, g.details.URL) {
		log.Printf("[GIT SYNC] Remote URL mismatch, removing directory and performing fresh clone")
		if err := os.RemoveAll(g.targetDir); err != nil {
			log.Printf("[GIT SYNC] ERROR: Failed to remove old repo: %v", err)
			return fmt.Errorf("failed to remove old repo at %s: %w", g.targetDir, err)
		}
		log.Printf("[GIT SYNC] Old repository removed successfully")
		return g.cloneRepo(branch)
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
		log.Printf("[GIT SYNC] Executing git command: git %v", gitCmd)
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
		// Get the current branch name
		currentBranchOutput, err := exec.Command("git", "-C", g.targetDir, "branch", "--show-current").Output()
		if err == nil {
			currentBranch := strings.TrimSpace(string(currentBranchOutput))
			log.Printf("[GIT SYNC] Cloned to default branch: %s", currentBranch)
		}
	}

	log.Printf("[GIT SYNC] Git clone completed successfully: repo=%s targetDir=%s", g.details.URL, g.targetDir)
	return nil
}

// runGitInTarget runs a git command in the target directory
func (g *GitSyncer) runGitInTarget(args []string) error {
	log.Printf("[GIT SYNC] Executing in %s: git %v", g.targetDir, args)

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

	// Try to get the default branch from remote HEAD
	output, err := exec.Command("git", "-C", g.targetDir, "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err != nil {
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

		// Try again after setting remote HEAD
		output, err = exec.Command("git", "-C", g.targetDir, "symbolic-ref", "refs/remotes/origin/HEAD").Output()
		if err != nil {
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
