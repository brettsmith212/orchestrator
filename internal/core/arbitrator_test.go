package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brettsmith212/orchestrator/internal/gitutil"
	"github.com/brettsmith212/orchestrator/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArbitrator(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping arbitrator test in short mode")
	}

	// Create a test repository with a failing test
	baseRepoDir := t.TempDir()
	createTestProjectWithFailingTest(t, baseRepoDir)

	// Create a test runner
	testRunner := NewTestRunner("go test ./...", 30*time.Second)

	// Create an arbitrator
	arbitrator := NewArbitrator(testRunner, baseRepoDir)

	// Set baseline test results
	ctx := context.Background()
	err := arbitrator.SetBaselineTestResults(ctx)
	require.NoError(t, err)

	// Create patches with different fixes
	patches, err := createTestPatches(t, baseRepoDir)
	require.NoError(t, err)

	// Select the best patch
	bestPatch, err := arbitrator.SelectBestPatch(ctx, patches)
	require.NoError(t, err)

	// Verify the best patch is the one that fixes all tests with minimal changes
	assert.Equal(t, "good-agent", bestPatch.AgentID)
	assert.True(t, bestPatch.TestResults.Success)
	assert.Greater(t, bestPatch.Score, 0)
}

func TestCalculateScore(t *testing.T) {
	// Define test cases
	tests := []struct {
		name       string
		improved   bool
		diffStats  gitutil.DiffStats
		testResult TestResult
		expected   int
	}{
		{
			name:     "perfect fix",
			improved: true,
			diffStats: gitutil.DiffStats{
				FilesChanged: 1,
				LinesAdded:   3,
				LinesRemoved: 2,
			},
			testResult: TestResult{
				Success:     true,
				TotalTests:  10,
				PassedTests: 10,
				FailedTests: 0,
			},
			expected: 100 + 50 + 10*5 + 5 + 10, // Improved + All pass + 10 passing tests + Small change + minimal fix
		},
		{
			name:     "partial fix",
			improved: true,
			diffStats: gitutil.DiffStats{
				FilesChanged: 1,
				LinesAdded:   5,
				LinesRemoved: 2,
			},
			testResult: TestResult{
				Success:     false,
				TotalTests:  10,
				PassedTests: 8,
				FailedTests: 2,
			},
			expected: 100 + 0 + 8*5 - 2*10 + 5, // Improved + 8 passing - 2 failing + Small change
		},
		{
			name:     "no improvement",
			improved: false,
			diffStats: gitutil.DiffStats{
				FilesChanged: 2,
				LinesAdded:   10,
				LinesRemoved: 5,
			},
			testResult: TestResult{
				Success:     false,
				TotalTests:  10,
				PassedTests: 5,
				FailedTests: 5,
			},
			expected: 0 + 0 + 5*5 - 5*10 + 0, // No improvement + 5 passing - 5 failing
		},
		{
			name:     "huge change",
			improved: true,
			diffStats: gitutil.DiffStats{
				FilesChanged: 10,
				LinesAdded:   200,
				LinesRemoved: 100,
			},
			testResult: TestResult{
				Success:     true,
				TotalTests:  10,
				PassedTests: 10,
				FailedTests: 0,
			},
			expected: 100 + 50 + 10*5 - 5, // Improved + All pass + 10 passing tests - Large change penalty
		},
	}

	// Run the test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score := calculateScore(tc.improved, tc.diffStats, &tc.testResult)
			assert.Equal(t, tc.expected, score)
		})
	}
}

func TestFormatPatchResult(t *testing.T) {
	// Create a sample patch result
	result := &PatchResult{
		AgentID: "test-agent",
		Score:   150,
		Reason:  "Tests now passing",
		DiffStats: gitutil.DiffStats{
			FilesChanged: 2,
			LinesAdded:   10,
			LinesRemoved: 5,
		},
		TestResults: &TestResult{
			Success:     true,
			TotalTests:  10,
			PassedTests: 10,
			FailedTests: 0,
		},
	}

	// Format the result
	output := FormatPatchResult(result)

	// Verify the output
	assert.Contains(t, output, "Agent: test-agent")
	assert.Contains(t, output, "Score: 150")
	assert.Contains(t, output, "Tests now passing")
	assert.Contains(t, output, "2 files modified")
	assert.Contains(t, output, "10 lines added")
	assert.Contains(t, output, "5 lines removed")
	assert.Contains(t, output, "10 total")
	assert.Contains(t, output, "10 passed")
}

// Helper functions to set up test patches

func createTestProjectWithFailingTest(t *testing.T, dir string) {
	// Create a go.mod file
	goMod := filepath.Join(dir, "go.mod")
	err := os.WriteFile(goMod, []byte("module testproject\n\ngo 1.19\n"), 0644)
	require.NoError(t, err)

	// Create a simple package
	pkgDir := filepath.Join(dir, "pkg")
	err = os.MkdirAll(pkgDir, 0755)
	require.NoError(t, err)

	// Create a buggy Go file
	goFile := filepath.Join(pkgDir, "buggy.go")
	goContent := `package pkg

func Divide(a, b int) (int, error) {
	// Bug: Missing check for division by zero
	return a / b, nil
}
`
	err = os.WriteFile(goFile, []byte(goContent), 0644)
	require.NoError(t, err)

	// Create a test file that will fail
	testFile := filepath.Join(pkgDir, "buggy_test.go")
	testContent := `package pkg

import "testing"

func TestDivide(t *testing.T) {
	// This will pass
	result, err := Divide(10, 2)
	if err != nil || result != 5 {
		t.Errorf("Expected 5, got %d with error %v", result, err)
	}

	// This will fail due to division by zero
	_, err = Divide(10, 0)
	if err == nil {
		t.Error("Expected error for division by zero, got none")
	}
}
`
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Initialize git repository
	_, err = runGitCommand(dir, "init")
	require.NoError(t, err)

	// Configure git user
	_, err = runGitCommand(dir, "config", "user.name", "Test User")
	require.NoError(t, err)
	_, err = runGitCommand(dir, "config", "user.email", "test@example.com")
	require.NoError(t, err)

	// Add and commit files
	_, err = runGitCommand(dir, "add", ".")
	require.NoError(t, err)
	_, err = runGitCommand(dir, "commit", "-m", "Initial commit with buggy code")
	require.NoError(t, err)
}

func createTestPatches(t *testing.T, baseRepoDir string) (map[string]*PatchDetails, error) {
	// Create a worktree manager
	wm, err := gitutil.NewWorktreeManager(baseRepoDir, filepath.Join(t.TempDir(), "worktrees"))
	if err != nil {
		return nil, err
	}

	// Create patches with different fixes
	patches := make(map[string]*PatchDetails)

	// Create a good fix (minimal change that fixes the bug)
	goodWorktreePath, err := wm.CreateWorktree("good-agent", "")
	if err != nil {
		return nil, err
	}

	// Fix the divide function with a proper zero check
	fixedFile := filepath.Join(goodWorktreePath, "pkg", "buggy.go")
	fixedContent := `package pkg

import "errors"

func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}
`
	err = os.WriteFile(fixedFile, []byte(fixedContent), 0644)
	if err != nil {
		return nil, err
	}

	// Get the diff for the good fix
	goodDiff, err := wm.GetDiff(goodWorktreePath)
	if err != nil {
		return nil, err
	}

	// Create some sample events
	goodEvents := []*protocol.Event{
		protocol.NewEvent(protocol.EventTypeThinking, "good-agent", 1),
		protocol.NewEvent(protocol.EventTypeAction, "good-agent", 2),
		protocol.NewEvent(protocol.EventTypeComplete, "good-agent", 3),
	}

	// Add the good patch
	patches["good-agent"] = &PatchDetails{
		WorktreePath: goodWorktreePath,
		Diff:         goodDiff,
		Events:       goodEvents,
	}

	// Create an overly complex fix (works but makes too many changes)
	complexWorktreePath, err := wm.CreateWorktree("complex-agent", "")
	if err != nil {
		return nil, err
	}

	// Completely rewrite the divide function with lots of unnecessary changes
	complexFile := filepath.Join(complexWorktreePath, "pkg", "buggy.go")
	complexContent := `package pkg

import (
	"errors"
	"fmt"
)

// Lots of extra comments
// And more comments
// And even more

// Divide performs division of two integers
// It returns an error if b is zero
// Otherwise it returns a/b and nil error
func Divide(a, b int) (result int, err error) {
	// Validate input
	if b == 0 {
		// Can't divide by zero
		return 0, errors.New("cannot divide by zero: division would be undefined")
	}
	
	// Perform the division
	result = a / b
	
	// Create success message
	msg := fmt.Sprintf("Successfully divided %d by %d to get %d", a, b, result)
	
	// Debug output
	_ = msg
	
	// Return the result
	return result, nil
}
`
	err = os.WriteFile(complexFile, []byte(complexContent), 0644)
	if err != nil {
		return nil, err
	}

	// Get the diff for the complex fix
	complexDiff, err := wm.GetDiff(complexWorktreePath)
	if err != nil {
		return nil, err
	}

	// Create some sample events
	complexEvents := []*protocol.Event{
		protocol.NewEvent(protocol.EventTypeThinking, "complex-agent", 1),
		protocol.NewEvent(protocol.EventTypeAction, "complex-agent", 2),
		protocol.NewEvent(protocol.EventTypeComplete, "complex-agent", 3),
	}

	// Add the complex patch
	patches["complex-agent"] = &PatchDetails{
		WorktreePath: complexWorktreePath,
		Diff:         complexDiff,
		Events:       complexEvents,
	}

	// Create a bad fix (doesn't actually fix the issue)
	badWorktreePath, err := wm.CreateWorktree("bad-agent", "")
	if err != nil {
		return nil, err
	}

	// Make some changes but don't actually fix the bug
	badFile := filepath.Join(badWorktreePath, "pkg", "buggy.go")
	badContent := `package pkg

// Renamed function but didn't fix the bug
func DivideNumbers(a, b int) (int, error) {
	// Still no check for zero
	return a / b, nil
}

// The original function with the same bug
func Divide(a, b int) (int, error) {
	return a / b, nil
}
`
	err = os.WriteFile(badFile, []byte(badContent), 0644)
	if err != nil {
		return nil, err
	}

	// Get the diff for the bad fix
	badDiff, err := wm.GetDiff(badWorktreePath)
	if err != nil {
		return nil, err
	}

	// Create some sample events
	badEvents := []*protocol.Event{
		protocol.NewEvent(protocol.EventTypeThinking, "bad-agent", 1),
		protocol.NewEvent(protocol.EventTypeAction, "bad-agent", 2),
		protocol.NewEvent(protocol.EventTypeComplete, "bad-agent", 3),
	}

	// Add the bad patch
	patches["bad-agent"] = &PatchDetails{
		WorktreePath: badWorktreePath,
		Diff:         badDiff,
		Events:       badEvents,
	}

	return patches, nil
}

// Helper to run git commands
func runGitCommand(dir string, args ...string) (string, error) {
	cmd := gitutil.RunGitCommand(dir, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}