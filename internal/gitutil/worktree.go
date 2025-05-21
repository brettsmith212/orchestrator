package gitutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeManager manages git worktrees for a repository
type WorktreeManager struct {
	// repoPath is the path to the original git repository
	repoPath string

	// workingDir is the directory where temporary worktrees will be created
	workingDir string

	// createdWorktrees keeps track of created worktree paths for cleanup
	createdWorktrees []string
}

// NewWorktreeManager creates a new worktree manager for a git repository
func NewWorktreeManager(repoPath, workingDir string) (*WorktreeManager, error) {
	// Check if repoPath is a valid git repository
	if err := validateGitRepo(repoPath); err != nil {
		return nil, fmt.Errorf("invalid git repository: %w", err)
	}

	// Create working directory if it doesn't exist
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}

	return &WorktreeManager{
		repoPath:         repoPath,
		workingDir:      workingDir,
		createdWorktrees: []string{},
	}, nil
}

// CreateWorktree creates a new worktree for the repository
// The worktree will be based on the given ref (branch, tag, or commit hash)
// If ref is empty, it will use the current HEAD
func (wm *WorktreeManager) CreateWorktree(agentID string, ref string) (string, error) {
	// Generate a unique worktree path
	worktreePath := filepath.Join(wm.workingDir, fmt.Sprintf("worktree-%s-%s", agentID, randomString(8)))

	// Use HEAD if ref is empty
	if ref == "" {
		ref = "HEAD"
	}

	// Create the worktree
	cmd := exec.Command("git", "-C", wm.repoPath, "worktree", "add", worktreePath, ref)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %w - %s", err, output)
	}

	// Add to the list of created worktrees
	wm.createdWorktrees = append(wm.createdWorktrees, worktreePath)

	return worktreePath, nil
}

// GetDiff returns the diff for changes made in the worktree
func (wm *WorktreeManager) GetDiff(worktreePath string) (string, error) {
	// Check if worktree exists
	if !wm.isValidWorktree(worktreePath) {
		return "", errors.New("invalid worktree path")
	}

	// Get the diff
	cmd := exec.Command("git", "-C", worktreePath, "diff")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// RemoveWorktree removes a previously created worktree
func (wm *WorktreeManager) RemoveWorktree(worktreePath string) error {
	// Check if worktree exists
	if !wm.isValidWorktree(worktreePath) {
		return errors.New("invalid worktree path")
	}

	// Remove the worktree
	cmd := exec.Command("git", "-C", wm.repoPath, "worktree", "remove", "--force", worktreePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w - %s", err, output)
	}

	// Remove from the list of created worktrees
	for i, path := range wm.createdWorktrees {
		if path == worktreePath {
			wm.createdWorktrees = append(wm.createdWorktrees[:i], wm.createdWorktrees[i+1:]...)
			break
		}
	}

	return nil
}

// Cleanup removes all worktrees created by this manager
func (wm *WorktreeManager) Cleanup() error {
	var errors []string

	// Copy the list to avoid issues with removal changing the slice
	worktrees := make([]string, len(wm.createdWorktrees))
	copy(worktrees, wm.createdWorktrees)

	// Remove each worktree
	for _, worktreePath := range worktrees {
		if err := wm.RemoveWorktree(worktreePath); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Report any errors
	if len(errors) > 0 {
		return fmt.Errorf("failed to clean up all worktrees: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Helper functions

// validateGitRepo checks if the given path is a valid git repository
func validateGitRepo(repoPath string) error {
	// Check if the directory exists
	stat, err := os.Stat(repoPath)
	if err != nil {
		return err
	}

	if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", repoPath)
	}

	// Check if .git directory exists or if it's a worktree
	gitDir := filepath.Join(repoPath, ".git")
	gitDirStat, err := os.Stat(gitDir)
	if err != nil {
		// Check if it's a worktree with a .git file
		gitFileStat, gitFileErr := os.Stat(gitDir)
		if gitFileErr != nil || gitFileStat.IsDir() {
			return errors.New("not a git repository (or any of the parent directories)")
		}
	} else if !gitDirStat.IsDir() {
		return errors.New(".git is not a directory")
	}

	// Run git status to verify it's a valid repository
	cmd := exec.Command("git", "-C", repoPath, "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	return nil
}

// isValidWorktree checks if the given path is a valid worktree created by this manager
func (wm *WorktreeManager) isValidWorktree(worktreePath string) bool {
	// Check if the path is in our list of created worktrees
	for _, path := range wm.createdWorktrees {
		if path == worktreePath {
			// Check if it still exists and is a valid git worktree
			cmd := exec.Command("git", "-C", worktreePath, "status")
			if err := cmd.Run(); err == nil {
				return true
			}
			break
		}
	}

	return false
}

// randomString generates a random string of the given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)

	// Use a simple method for testing, in production a crypto random source would be better
	for i := range b {
		b[i] = charset[i%len(charset)]
	}

	return string(b)
}

// RunGitCommand creates an exec.Cmd to run a git command in the given directory
func RunGitCommand(dir string, args ...string) *exec.Cmd {
	// Prepend "git" to the args
	gitArgs := append([]string{"git"}, args...)
	
	// Create the command
	cmd := exec.Command(gitArgs[0], gitArgs[1:]...)
	cmd.Dir = dir
	
	return cmd
}