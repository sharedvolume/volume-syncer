package git

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
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

	if err := g.validate(); err != nil {
		log.Printf("[GIT SYNC] Validation error: %v", err)
		return err
	}

	// Ensure target directory exists
	if err := utils.EnsureDir(g.targetDir); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	branch := g.details.Branch
	if branch == "" {
		branch = "main" // default branch
	}

	// If targetDir exists and is a git repo, do fetch/reset/clean
	gitDir := g.targetDir + "/.git"
	if stat, err := os.Stat(g.targetDir); err == nil && stat.IsDir() {
		if _, err := os.Stat(gitDir); err == nil {
			return g.syncExistingRepo(branch)
		}
	}

	// Otherwise, do a shallow clone
	return g.cloneRepo(branch)
}

// syncExistingRepo syncs an existing git repository
func (g *GitSyncer) syncExistingRepo(branch string) error {
	// Check if the remote URL matches
	remoteURLBytes, err := exec.Command("git", "-C", g.targetDir, "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return fmt.Errorf("failed to get remote URL: %w", err)
	}

	remoteURL := string(remoteURLBytes)
	if remoteURL != g.details.URL+"\n" {
		log.Printf("[GIT SYNC] Remote URL mismatch (expected: %s, got: %s), removing directory and recloning", g.details.URL, remoteURL)
		if err := os.RemoveAll(g.targetDir); err != nil {
			return fmt.Errorf("failed to remove old repo at %s: %w", g.targetDir, err)
		}
		return g.cloneRepo(branch)
	}

	log.Printf("[GIT SYNC] Target exists and is a git repo, syncing...")

	// git fetch
	if err := g.runGitInTarget([]string{"fetch", "--all"}); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	// Force local branch to match remote
	if err := g.runGitInTarget([]string{"checkout", "-B", branch, "origin/" + branch}); err != nil {
		// Try fallback to master if main fails
		if branch == "main" {
			log.Printf("[GIT SYNC] Falling back to master branch")
			branch = "master"
			if err := g.runGitInTarget([]string{"checkout", "-B", branch, "origin/" + branch}); err != nil {
				return fmt.Errorf("git checkout -B master failed: %w", err)
			}
		} else {
			return fmt.Errorf("git checkout -B %s origin/%s failed: %w", branch, branch, err)
		}
	}

	// git reset --hard origin/<branch>
	if err := g.runGitInTarget([]string{"reset", "--hard", "origin/" + branch}); err != nil {
		return fmt.Errorf("git reset failed: %w", err)
	}

	// git clean -fdx (always run clean)
	if err := g.runGitInTarget([]string{"clean", "-fdx"}); err != nil {
		return fmt.Errorf("git clean failed: %w", err)
	}

	log.Printf("[GIT SYNC] Git repo synced to origin/%s", branch)
	return nil
}

// cloneRepo clones a new repository
func (g *GitSyncer) cloneRepo(branch string) error {
	gitCmd := []string{"clone", "--depth", "1"}

	if branch != "" {
		gitCmd = append(gitCmd, "--branch", branch)
	}

	gitCmd = append(gitCmd, g.details.URL, g.targetDir)

	log.Printf("[GIT SYNC] Running command: git %v", gitCmd)

	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", gitCmd...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git clone timed out after %v", g.timeout)
		}
		return fmt.Errorf("git clone failed: %w", err)
	}

	log.Printf("[GIT SYNC] Git clone completed successfully: repo=%s targetDir=%s", g.details.URL, g.targetDir)
	return nil
}

// runGitInTarget runs a git command in the target directory
func (g *GitSyncer) runGitInTarget(args []string) error {
	log.Printf("[GIT SYNC] Running in %s: git %v", g.targetDir, args)

	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// validate validates the git details
func (g *GitSyncer) validate() error {
	if g.details == nil {
		return fmt.Errorf("GitCloneDetails is required")
	}

	if g.details.URL == "" {
		return fmt.Errorf("repository URL is required")
	}

	return nil
}
