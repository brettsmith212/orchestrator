package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// TestResult contains the results of running tests
type TestResult struct {
	// Success is true if all tests passed
	Success bool `json:"success"`

	// TotalTests is the total number of tests run
	TotalTests int `json:"total_tests"`

	// PassedTests is the number of tests that passed
	PassedTests int `json:"passed_tests"`

	// FailedTests is the number of tests that failed
	FailedTests int `json:"failed_tests"`

	// SkippedTests is the number of tests that were skipped
	SkippedTests int `json:"skipped_tests"`

	// Duration is how long the tests took to run
	Duration time.Duration `json:"duration"`

	// Output is the captured test output
	Output string `json:"output"`

	// Error is set if there was an error running the tests
	Error string `json:"error,omitempty"`
}

// TestRunner runs tests for a repository
type TestRunner struct {
	// TestCommand is the command to run tests, e.g. "go test ./..."
	TestCommand string

	// Timeout is the maximum time to wait for tests to complete
	Timeout time.Duration
}

// NewTestRunner creates a new test runner
func NewTestRunner(testCommand string, timeout time.Duration) *TestRunner {
	if timeout == 0 {
		timeout = 5 * time.Minute // Default timeout
	}

	return &TestRunner{
		TestCommand: testCommand,
		Timeout:     timeout,
	}
}

// Run executes tests in the given worktree path
func (tr *TestRunner) Run(ctx context.Context, worktreePath string) (*TestResult, error) {
	// Create a timeout context if one wasn't provided, or add timeout to existing context
	var cancel context.CancelFunc
	if ctx == nil {
		ctx, cancel = context.WithTimeout(context.Background(), tr.Timeout)
		defer cancel()
	} else {
		// Apply our timeout to the provided context
		ctx, cancel = context.WithTimeout(ctx, tr.Timeout)
		defer cancel()
	}

	// Prepare the test command
	cmdParts := strings.Fields(tr.TestCommand)
	if len(cmdParts) == 0 {
		return nil, errors.New("empty test command")
	}

	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	cmd.Dir = worktreePath

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Track start time
	startTime := time.Now()

	// Run the tests
	err := cmd.Run()

	// Calculate duration
	duration := time.Since(startTime)

	// Combine stdout and stderr
	output := stdout.String() + stderr.String()

	// Check for context timeout specifically
	if ctx.Err() == context.DeadlineExceeded {
		err = ctx.Err() // Make sure the error is set to the context error
	}

	// Parse results
	result := parseTestResults(output, duration, err)

	return result, nil
}

// parseTestResults analyzes test output to determine how many tests passed/failed
func parseTestResults(output string, duration time.Duration, runErr error) *TestResult {
	result := &TestResult{
		Success:  runErr == nil,
		Duration: duration,
		Output:   output,
	}

	// If there was an error running the tests, it might be a build failure
	if runErr != nil {
		result.Error = runErr.Error()
	}

	// Try to parse "go test" output format
	if lines := strings.Split(output, "\n"); len(lines) > 0 {
		for _, line := range lines {
			// Look for the test summary line, e.g. "ok  \tpackage/path\t0.015s"
			if strings.HasPrefix(line, "ok\t") || strings.HasPrefix(line, "FAIL\t") {
				result.TotalTests++
				if strings.HasPrefix(line, "ok\t") {
					result.PassedTests++
				} else {
					result.FailedTests++
				}
			}

			// Look for test2json format (go test -json)
			if strings.Contains(line, "\"Test\":") {
				var testEvent map[string]interface{}
				if err := json.Unmarshal([]byte(line), &testEvent); err == nil {
					if action, ok := testEvent["Action"].(string); ok {
						switch action {
						case "pass":
							result.TotalTests++
							result.PassedTests++
						case "fail":
							result.TotalTests++
							result.FailedTests++
						case "skip":
							result.TotalTests++
							result.SkippedTests++
						}
					}
				}
			}
		}
	}

	// If we couldn't find any test details but there was no error, assume at least one test passed
	if result.TotalTests == 0 && result.Success {
		result.TotalTests = 1
		result.PassedTests = 1
	}

	// Make sure the success flag accounts for failed tests
	result.Success = result.Success && result.FailedTests == 0

	return result
}

// FormatResults returns a human-readable summary of test results
func FormatResults(result *TestResult) string {
	var status string
	if result.Success {
		status = "PASSED"
	} else {
		status = "FAILED"
	}

	return fmt.Sprintf(
		"Tests %s (%d total, %d passed, %d failed, %d skipped) in %s",
		status,
		result.TotalTests,
		result.PassedTests,
		result.FailedTests,
		result.SkippedTests,
		result.Duration.Round(time.Millisecond),
	)
}

// CompareResults compares two test results to see if the patch improved the test outcome
func CompareResults(before, after *TestResult) (bool, string) {
	// If tests were failing and now passing, that's an improvement
	if !before.Success && after.Success {
		return true, "Tests now passing"
	}

	// If both failed, but fewer failures now, that's an improvement
	if !before.Success && !after.Success && after.FailedTests < before.FailedTests {
		return true, fmt.Sprintf("Reduced failing tests from %d to %d", before.FailedTests, after.FailedTests)
	}

	// If same number of failures but more tests passing, that's an improvement
	if after.PassedTests > before.PassedTests {
		return true, fmt.Sprintf("Increased passing tests from %d to %d", before.PassedTests, after.PassedTests)
	}

	// If tests were passing and are still passing, no change
	if before.Success && after.Success {
		return false, "No change in test results, all tests still passing"
	}

	// If tests were failing and still are with same stats, no change
	if !before.Success && !after.Success &&
		before.FailedTests == after.FailedTests &&
		before.PassedTests == after.PassedTests {
		return false, "No change in test results, same failures"
	}

	// If tests were passing but now failing, that's a regression
	if before.Success && !after.Success {
		return false, "Tests now failing, patch introduces regression"
	}

	// Default case
	return false, "No significant change in test results"
}