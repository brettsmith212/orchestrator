package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestRunner(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping test runner test in short mode")
	}

	// Create a temporary test project
	tempDir := t.TempDir()
	createTestProject(t, tempDir)

	// Create a test runner with go test command
	testRunner := NewTestRunner("go test ./...", 30*time.Second)

	// Run the tests
	ctx := context.Background()
	result, err := testRunner.Run(ctx, tempDir)

	// Check results
	require.NoError(t, err, "Test runner should not error")
	assert.True(t, result.Success, "Tests should pass")
	assert.Greater(t, result.TotalTests, 0, "Should have run at least one test")
	assert.Equal(t, result.TotalTests, result.PassedTests, "All tests should pass")
	assert.Equal(t, 0, result.FailedTests, "No tests should fail")
	assert.NotEmpty(t, result.Output, "Should have test output")

	// Now introduce a failing test
	introduceFailingTest(t, tempDir)

	// Run the tests again
	result, err = testRunner.Run(ctx, tempDir)

	// Check results with failing test
	require.NoError(t, err, "Test runner should not error even with failing tests")
	assert.False(t, result.Success, "Tests should fail")
	assert.Greater(t, result.FailedTests, 0, "Should have failing tests")
	assert.Contains(t, result.Output, "FAIL", "Output should indicate failure")
}

func TestTestRunnerTimeout(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping test runner timeout test in short mode")
	}

	// Create a temporary test project with a slow test
	tempDir := t.TempDir()
	createSlowTestProject(t, tempDir)

	// Create a test runner with a very short timeout
	testRunner := NewTestRunner("go test ./...", 500*time.Millisecond)

	// Run the tests
	ctx := context.Background()
	result, _ := testRunner.Run(ctx, tempDir)

	// Check results
	assert.False(t, result.Success, "Tests should not succeed due to timeout")
	assert.Contains(t, result.Error, "context deadline exceeded", "Error should indicate timeout")
}

func TestFormatResults(t *testing.T) {
	// Test formatting passing results
	passing := &TestResult{
		Success:      true,
		TotalTests:   10,
		PassedTests:  10,
		FailedTests:  0,
		SkippedTests: 0,
		Duration:     3 * time.Second,
	}

	passingStr := FormatResults(passing)
	assert.Contains(t, passingStr, "PASSED")
	assert.Contains(t, passingStr, "10 total")

	// Test formatting failing results
	failing := &TestResult{
		Success:      false,
		TotalTests:   10,
		PassedTests:  8,
		FailedTests:  2,
		SkippedTests: 0,
		Duration:     3 * time.Second,
	}

	failingStr := FormatResults(failing)
	assert.Contains(t, failingStr, "FAILED")
	assert.Contains(t, failingStr, "2 failed")
}

func TestCompareResults(t *testing.T) {
	// Define test cases
	tests := []struct {
		name     string
		before   *TestResult
		after    *TestResult
		improved bool
	}{
		{
			name: "failing to passing",
			before: &TestResult{
				Success:     false,
				TotalTests:  10,
				PassedTests: 8,
				FailedTests: 2,
			},
			after: &TestResult{
				Success:     true,
				TotalTests:  10,
				PassedTests: 10,
				FailedTests: 0,
			},
			improved: true,
		},
		{
			name: "fewer failures",
			before: &TestResult{
				Success:     false,
				TotalTests:  10,
				PassedTests: 7,
				FailedTests: 3,
			},
			after: &TestResult{
				Success:     false,
				TotalTests:  10,
				PassedTests: 8,
				FailedTests: 2,
			},
			improved: true,
		},
		{
			name: "passing to failing",
			before: &TestResult{
				Success:     true,
				TotalTests:  10,
				PassedTests: 10,
				FailedTests: 0,
			},
			after: &TestResult{
				Success:     false,
				TotalTests:  10,
				PassedTests: 9,
				FailedTests: 1,
			},
			improved: false,
		},
		{
			name: "no change passing",
			before: &TestResult{
				Success:     true,
				TotalTests:  10,
				PassedTests: 10,
				FailedTests: 0,
			},
			after: &TestResult{
				Success:     true,
				TotalTests:  10,
				PassedTests: 10,
				FailedTests: 0,
			},
			improved: false,
		},
	}

	// Run the test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			improved, reason := CompareResults(tc.before, tc.after)
			assert.Equal(t, tc.improved, improved)
			assert.NotEmpty(t, reason)
		})
	}
}

// Helper functions to set up a test project
func createTestProject(t *testing.T, dir string) {
	// Create a go.mod file
	goMod := filepath.Join(dir, "go.mod")
	err := os.WriteFile(goMod, []byte("module testproject\n\ngo 1.19\n"), 0644)
	require.NoError(t, err)

	// Create a simple package
	pkgDir := filepath.Join(dir, "pkg")
	err = os.MkdirAll(pkgDir, 0755)
	require.NoError(t, err)

	// Create a simple Go file
	goFile := filepath.Join(pkgDir, "simple.go")
	goContent := `package pkg

func Add(a, b int) int {
	return a + b
}
`
	err = os.WriteFile(goFile, []byte(goContent), 0644)
	require.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(pkgDir, "simple_test.go")
	testContent := `package pkg

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Expected 5, got %d", result)
	}
}
`
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
}

func introduceFailingTest(t *testing.T, dir string) {
	// Add a failing test file
	pkgDir := filepath.Join(dir, "pkg")
	failingFile := filepath.Join(pkgDir, "failing_test.go")
	failingContent := `package pkg

import "testing"

func TestFailing(t *testing.T) {
	// This test will always fail
	t.Error("This is a failing test")
}
`
	err := os.WriteFile(failingFile, []byte(failingContent), 0644)
	require.NoError(t, err)
}

func createSlowTestProject(t *testing.T, dir string) {
	// Create a go.mod file
	goMod := filepath.Join(dir, "go.mod")
	err := os.WriteFile(goMod, []byte("module slowtestproject\n\ngo 1.19\n"), 0644)
	require.NoError(t, err)

	// Create a simple package
	pkgDir := filepath.Join(dir, "pkg")
	err = os.MkdirAll(pkgDir, 0755)
	require.NoError(t, err)

	// Create a slow test
	testFile := filepath.Join(pkgDir, "slow_test.go")
	testContent := `package pkg

import (
	"testing"
	"time"
)

func TestSlow(t *testing.T) {
	// This test will run for 30 seconds (definitely longer than our timeout)
	time.Sleep(30 * time.Second)
}
`
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
}