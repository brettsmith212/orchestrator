package core

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/brettsmith212/orchestrator/internal/gitutil"
	"github.com/brettsmith212/orchestrator/internal/protocol"
)

// PatchResult represents an agent's patch and its evaluation
type PatchResult struct {
	// AgentID identifies which agent generated this patch
	AgentID string

	// Diff is the git diff of the patch
	Diff string

	// DiffStats provides statistics about the diff
	DiffStats gitutil.DiffStats

	// TestResults contains the results of running tests on this patch
	TestResults *TestResult

	// Events contains all events emitted by the agent
	Events []*protocol.Event

	// Score is a numeric evaluation of the patch quality (higher is better)
	Score int

	// Reason is a human-readable explanation for the score
	Reason string
}

// Arbitrator evaluates and selects the best patch from multiple agents
type Arbitrator struct {
	// TestRunner runs tests on patched code
	testRunner *TestRunner

	// BaseTestResults are test results before applying any patches
	baseTestResults *TestResult

	// BaseRepoPath is the path to the original repository
	baseRepoPath string
}

// NewArbitrator creates a new arbitrator for patch selection
func NewArbitrator(testRunner *TestRunner, baseRepoPath string) *Arbitrator {
	return &Arbitrator{
		testRunner:  testRunner,
		baseRepoPath: baseRepoPath,
	}
}

// SetBaselineTestResults runs tests on the original code to establish a baseline
func (a *Arbitrator) SetBaselineTestResults(ctx context.Context) error {
	var err error
	a.baseTestResults, err = a.testRunner.Run(ctx, a.baseRepoPath)
	return err
}

// EvaluatePatch evaluates a single patch
func (a *Arbitrator) EvaluatePatch(ctx context.Context, agentID, worktreePath, diff string, events []*protocol.Event) (*PatchResult, error) {
	// Skip empty diffs
	if strings.TrimSpace(diff) == "" {
		return &PatchResult{
			AgentID: agentID,
			Diff:    "",
			Score:   0,
			Reason:  "No changes made",
			Events:  events,
		}, nil
	}

	// Analyze the diff
	diffStats := gitutil.GetDiffStats(diff)

	// Skip diffs with conflicts
	if diffStats.HasConflicts {
		return &PatchResult{
			AgentID:   agentID,
			Diff:      diff,
			DiffStats: diffStats,
			Score:     -10,
			Reason:    "Patch contains merge conflicts",
			Events:    events,
		}, nil
	}

	// Run tests on the patched code
	testResults, err := a.testRunner.Run(ctx, worktreePath)
	if err != nil {
		return nil, fmt.Errorf("failed to run tests on patched code: %w", err)
	}

	// Compare with baseline tests
	improved, reason := CompareResults(a.baseTestResults, testResults)

	// Calculate score
	score := calculateScore(improved, diffStats, testResults)

	return &PatchResult{
		AgentID:     agentID,
		Diff:        diff,
		DiffStats:   diffStats,
		TestResults: testResults,
		Events:      events,
		Score:       score,
		Reason:      reason,
	}, nil
}

// SelectBestPatch evaluates all patches and selects the best one
func (a *Arbitrator) SelectBestPatch(ctx context.Context, patches map[string]*PatchDetails) (*PatchResult, error) {
	if len(patches) == 0 {
		return nil, fmt.Errorf("no patches to evaluate")
	}

	// Evaluate each patch
	results := make([]*PatchResult, 0, len(patches))
	for agentID, patch := range patches {
		result, err := a.EvaluatePatch(ctx, agentID, patch.WorktreePath, patch.Diff, patch.Events)
		if err != nil {
			// Skip this patch but continue evaluating others
			fmt.Printf("Error evaluating patch from %s: %v\n", agentID, err)
			continue
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("all patches failed evaluation")
	}

	// Sort patches by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return the highest scoring patch
	return results[0], nil
}

// PatchDetails contains information about a patch from an agent
type PatchDetails struct {
	// WorktreePath is the path to the worktree with the patch applied
	WorktreePath string

	// Diff is the git diff of the patch
	Diff string

	// Events is the list of events from the agent
	Events []*protocol.Event
}

// calculateScore computes a numeric score for a patch
func calculateScore(improved bool, diffStats gitutil.DiffStats, testResults *TestResult) int {
	var score int

	// Base points for test improvement
	if improved {
		score += 100
	}

	// Additional points for passing all tests
	if testResults.Success {
		score += 50
	}

	// Points for each passing test
	score += testResults.PassedTests * 5

	// Penalties for failing tests
	score -= testResults.FailedTests * 10

	// Slight preference for smaller diffs when all else is equal
	totalChanges := diffStats.LinesAdded + diffStats.LinesRemoved
	if totalChanges > 0 && totalChanges <= 10 {
		score += 5 // Small changes are good
	} else if totalChanges > 50 {
		score -= 5 // Penalize very large changes
	}

	// Bonus for fixing things with minimal changes
	if testResults.Success && totalChanges < 20 {
		score += 10 // Clean, minimal fixes are ideal
	}

	return score
}

// FormatPatchResult returns a human-readable summary of a patch result
func FormatPatchResult(result *PatchResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Agent: %s\n", result.AgentID))
	sb.WriteString(fmt.Sprintf("Score: %d (%s)\n", result.Score, result.Reason))
	
	if result.DiffStats.FilesChanged > 0 {
		sb.WriteString(fmt.Sprintf("Changes: %d files modified, %d lines added, %d lines removed\n", 
			result.DiffStats.FilesChanged, result.DiffStats.LinesAdded, result.DiffStats.LinesRemoved))
	}

	if result.TestResults != nil {
		sb.WriteString(fmt.Sprintf("Tests: %d total, %d passed, %d failed\n", 
			result.TestResults.TotalTests, result.TestResults.PassedTests, result.TestResults.FailedTests))
	}

	return sb.String()
}