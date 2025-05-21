package gitutil

import (
	"bufio"
	"regexp"
	"strings"
)

// DiffStats holds statistics about a git diff
type DiffStats struct {
	// FilesChanged is the number of files modified in the diff
	FilesChanged int

	// LinesAdded is the number of lines added in the diff
	LinesAdded int

	// LinesRemoved is the number of lines removed in the diff
	LinesRemoved int

	// HasConflicts indicates if the diff contains merge conflicts
	HasConflicts bool
}

// Constants for diff parsing
const (
	AddedLinePrefix   = "+"
	RemovedLinePrefix = "-"
	HunkHeaderPrefix  = "@@"
	NoNewlineMarker   = "\\ No newline at end of file"
)

// Regular expressions for diff parsing
var (
	// Match diff file header: "diff --git a/file.txt b/file.txt"
	fileHeaderRegex = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)

	// Match hunk header: "@@ -1,7 +1,9 @@"
	hunkHeaderRegex = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+\d+(?:,\d+)? @@`)

	// Match timestamp lines that might vary between otherwise identical diffs
	timestampRegex = regexp.MustCompile(`^(\+\+\+|---) .*\s+\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)

	// Match index lines that contain SHA hashes 
	indexLineRegex = regexp.MustCompile(`^index [0-9a-f]+\.\.[0-9a-f]+`)
)

// NormalizeDiff normalizes a git diff for consistent comparison
// It removes timestamps, index hashes, and other variable elements
func NormalizeDiff(diff string) string {
	var normalized strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(diff))

	for scanner.Scan() {
		line := scanner.Text()

		// Skip timestamp lines
		if timestampRegex.MatchString(line) {
			continue
		}

		// Skip index lines with SHA hashes
		if indexLineRegex.MatchString(line) {
			continue
		}

		// Keep file headers but normalize them
		if fileHeaderRegex.MatchString(line) {
			matches := fileHeaderRegex.FindStringSubmatch(line)
			if len(matches) >= 3 && matches[1] == matches[2] {
				filePath := matches[1]
				normalized.WriteString("diff --git a/" + filePath + " b/" + filePath + "\n")
				continue
			}
		}

		// Keep the line as is
		normalized.WriteString(line + "\n")
	}

	return normalized.String()
}

// GetDiffStats calculates statistics for a diff
func GetDiffStats(diff string) DiffStats {
	stats := DiffStats{}

	// Exit early if diff is empty
	if diff == "" {
		return stats
	}

	scanner := bufio.NewScanner(strings.NewReader(diff))
	inFile := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check for new file in diff
		if fileHeaderRegex.MatchString(line) {
			stats.FilesChanged++
			inFile = true
			continue
		}

		// Count added and removed lines
		if inFile {
			if strings.HasPrefix(line, AddedLinePrefix) && !strings.HasPrefix(line, "+++") {
				stats.LinesAdded++
			} else if strings.HasPrefix(line, RemovedLinePrefix) && !strings.HasPrefix(line, "---") {
				stats.LinesRemoved++
			}

			// Check for conflict markers
			if strings.HasPrefix(line, "<<<<<<<") || 
			   strings.HasPrefix(line, "=======") || 
			   strings.HasPrefix(line, ">>>>>>>") {
				stats.HasConflicts = true
			}
		}
	}

	return stats
}

// RemoveContextLines reduces a diff to just the changed lines, removing context lines
func RemoveContextLines(diff string) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(diff))

	for scanner.Scan() {
		line := scanner.Text()

		// Keep file headers and hunk headers
		if fileHeaderRegex.MatchString(line) || 
		   hunkHeaderRegex.MatchString(line) || 
		   strings.HasPrefix(line, "+++") || 
		   strings.HasPrefix(line, "---") {
			result.WriteString(line + "\n")
			continue
		}

		// Keep added and removed lines
		if strings.HasPrefix(line, AddedLinePrefix) || strings.HasPrefix(line, RemovedLinePrefix) {
			result.WriteString(line + "\n")
		}

		// Keep "No newline" markers
		if strings.Contains(line, NoNewlineMarker) {
			result.WriteString(line + "\n")
		}
	}

	return result.String()
}

// CompareDiffs compares two normalized diffs for similarity
// Returns true if diffs are functionally equivalent
func CompareDiffs(diff1, diff2 string) bool {
	// Normalize both diffs
	normalized1 := NormalizeDiff(diff1)
	normalized2 := NormalizeDiff(diff2)

	// Remove context lines to focus on just the changes
	reduced1 := RemoveContextLines(normalized1)
	reduced2 := RemoveContextLines(normalized2)

	// Compare the normalized, context-free diffs
	return reduced1 == reduced2
}

// FindLargestDiff compares multiple diffs and returns the one with the most significant changes
// Useful for selecting the most comprehensive patch when multiple options solve the same problem
func FindLargestDiff(diffs []string) string {
	if len(diffs) == 0 {
		return ""
	}

	largestDiffIndex := 0
	largestScore := -1

	for i, diff := range diffs {
		stats := GetDiffStats(diff)

		// Skip diffs with conflicts
		if stats.HasConflicts {
			continue
		}

		// Simple scoring: total number of changed lines
		score := stats.LinesAdded + stats.LinesRemoved

		if score > largestScore {
			largestScore = score
			largestDiffIndex = i
		}
	}

	return diffs[largestDiffIndex]
}

// MergeDiffs attempts to merge multiple compatible diffs into a single comprehensive diff
// This can be useful for combining partial solutions from different agents
func MergeDiffs(baseDiff string, overlayDiffs []string) (string, bool) {
	// If there's nothing to merge, return the base diff
	if len(overlayDiffs) == 0 {
		return baseDiff, true
	}

	// Parse the base diff
	baseLines := parseLines(baseDiff)
	for _, overlayDiff := range overlayDiffs {
		// Parse the overlay diff
		overlayLines := parseLines(overlayDiff)

		// Attempt to merge (simplified version)
		baseLines = simpleSetUnion(baseLines, overlayLines)
	}

	// Reconstruct merged diff
	return strings.Join(baseLines, "\n"), true
}

// Helper functions

// parseLines splits a diff into lines
func parseLines(diff string) []string {
	return strings.Split(strings.TrimSpace(diff), "\n")
}

// simpleSetUnion combines two sets of lines
func simpleSetUnion(set1, set2 []string) []string {
	lineMap := make(map[string]bool)

	// Add all lines from first set
	for _, line := range set1 {
		lineMap[line] = true
	}

	// Add all lines from second set
	for _, line := range set2 {
		lineMap[line] = true
	}

	// Convert back to slice
	result := make([]string, 0, len(lineMap))
	for line := range lineMap {
		result = append(result, line)
	}

	return result
}