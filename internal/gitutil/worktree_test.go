package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorktreeManager(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping worktree test in short mode")
	}

	// Create a temporary directory for the test repository
	repoDir := t.TempDir()

	// Create a temporary directory for worktrees
	worktreeDir := t.TempDir()

	// Initialize a git repository
	initTestRepo(t, repoDir)

	// Create a worktree manager
	wm, err := NewWorktreeManager(repoDir, worktreeDir)
	require.NoError(t, err, "Failed to create worktree manager")

	// Test creating a worktree
	worktreePath, err := wm.CreateWorktree("test-agent", "")
	require.NoError(t, err, "Failed to create worktree")

	// Verify worktree was created
	verifyDirectory(t, worktreePath)

	// Make a change in the worktree
	makeTestChange(t, worktreePath)

	// Get the diff
	diff, err := wm.GetDiff(worktreePath)
	require.NoError(t, err, "Failed to get diff")
	assert.Contains(t, diff, "test-file.txt", "Diff should contain the changed file")

	// Remove the worktree
	err = wm.RemoveWorktree(worktreePath)
	require.NoError(t, err, "Failed to remove worktree")

	// Verify worktree was removed
	_, err = os.Stat(worktreePath)
	require.Error(t, err, "Worktree should be removed")

	// Test cleanup
	worktreePath1, err := wm.CreateWorktree("agent1", "")
	require.NoError(t, err, "Failed to create worktree 1")

	worktreePath2, err := wm.CreateWorktree("agent2", "")
	require.NoError(t, err, "Failed to create worktree 2")

	// Verify worktrees exist
	verifyDirectory(t, worktreePath1)
	verifyDirectory(t, worktreePath2)

	// Clean up
	err = wm.Cleanup()
	require.NoError(t, err, "Failed to clean up worktrees")

	// Verify all worktrees were removed
	_, err = os.Stat(worktreePath1)
	require.Error(t, err, "Worktree 1 should be removed")

	_, err = os.Stat(worktreePath2)
	require.Error(t, err, "Worktree 2 should be removed")
}

func TestWorktreeManagerErrors(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping worktree test in short mode")
	}

	// Test with non-existent directory
	_, err := NewWorktreeManager("/non/existent/path", t.TempDir())
	require.Error(t, err, "Should error with non-existent repo path")

	// Test with non-git directory
	nonGitDir := t.TempDir()
	_, err = NewWorktreeManager(nonGitDir, t.TempDir())
	require.Error(t, err, "Should error with non-git directory")

	// Create a valid git repo
	repoDir := t.TempDir()
	initTestRepo(t, repoDir)
	worktreeDir := t.TempDir()

	wm, err := NewWorktreeManager(repoDir, worktreeDir)
	require.NoError(t, err, "Failed to create worktree manager")

	// Test invalid ref
	_, err = wm.CreateWorktree("test-agent", "non-existent-branch")
	require.Error(t, err, "Should error with non-existent ref")

	// Test getting diff from invalid worktree path
	_, err = wm.GetDiff("/invalid/path")
	require.Error(t, err, "Should error with invalid worktree path")

	// Test removing invalid worktree path
	err = wm.RemoveWorktree("/invalid/path")
	require.Error(t, err, "Should error with invalid worktree path")
}

// Helper functions

// initTestRepo initializes a git repository with a test file
func initTestRepo(t *testing.T, repoDir string) {
	// Initialize git repository
	cmd := exec.Command("git", "init", repoDir)
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run(), "Failed to initialize git repository")

	// Configure git user
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run(), "Failed to configure git user name")

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run(), "Failed to configure git user email")

	// Create a test file
	testFilePath := filepath.Join(repoDir, "test-file.txt")
	err := os.WriteFile(testFilePath, []byte("Initial content\n"), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Add and commit the file
	cmd = exec.Command("git", "add", "test-file.txt")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run(), "Failed to add test file")

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run(), "Failed to commit test file")
}

// makeTestChange makes a change to the test file in the worktree
func makeTestChange(t *testing.T, worktreePath string) {
	// Update the test file
	testFilePath := filepath.Join(worktreePath, "test-file.txt")
	err := os.WriteFile(testFilePath, []byte("Updated content\n"), 0644)
	require.NoError(t, err, "Failed to update test file")
}

// verifyDirectory checks if a directory exists
func verifyDirectory(t *testing.T, path string) {
	stat, err := os.Stat(path)
	require.NoError(t, err, "Directory should exist: %s", path)
	require.True(t, stat.IsDir(), "Path should be a directory: %s", path)
}