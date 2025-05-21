package gitutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Sample diffs for testing
const (
	sampleDiff1 = `diff --git a/file.txt b/file.txt
--- a/file.txt	2023-05-01 12:34:56.000000000 +0000
+++ b/file.txt	2023-05-01 12:45:12.000000000 +0000
index abcdef1234..fedcba4321 100644
@@ -1,3 +1,4 @@
 line 1
-line 2
+line 2 modified
+new line
 line 3
`

	sampleDiff2 = `diff --git a/file.txt b/file.txt
--- a/file.txt	2023-05-02 10:11:12.000000000 +0000
+++ b/file.txt	2023-05-02 10:22:33.000000000 +0000
index aaaabbbb..ccccdddd 100644
@@ -1,3 +1,4 @@
 line 1
-line 2
+line 2 modified
+new line
 line 3
`

	sampleDiff3 = `diff --git a/file.txt b/file.txt
--- a/file.txt	2023-05-03 15:16:17.000000000 +0000
+++ b/file.txt	2023-05-03 15:18:19.000000000 +0000
index ffffeeeee..0000fffff 100644
@@ -1,3 +1,5 @@
 line 1
-line 2
+line 2 modified
+new line
+another new line
 line 3
`

	sampleConflictDiff = `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,7 @@
 line 1
<<<<<<< HEAD
-line 2
=======
+line 2 modified
>>>>>>> feature-branch
 line 3
`
)

func TestNormalizeDiff(t *testing.T) {
	// Test that two functionally equivalent diffs normalize to the same output
	norm1 := NormalizeDiff(sampleDiff1)
	norm2 := NormalizeDiff(sampleDiff2)

	// Timestamps and index hashes should be removed
	assert.NotContains(t, norm1, "2023-05-01")
	assert.NotContains(t, norm1, "abcdef1234")

	// Both normalized diffs should be functionally equivalent
	assert.Equal(t, norm1, norm2, "Normalized diffs should be identical")

	// Different diffs should still be different after normalization
	norm3 := NormalizeDiff(sampleDiff3)
	assert.NotEqual(t, norm1, norm3, "Different diffs should remain different after normalization")
}

func TestGetDiffStats(t *testing.T) {
	// Test regular diff
	stats1 := GetDiffStats(sampleDiff1)
	assert.Equal(t, 1, stats1.FilesChanged)
	assert.Equal(t, 2, stats1.LinesAdded)
	assert.Equal(t, 1, stats1.LinesRemoved)
	assert.False(t, stats1.HasConflicts)

	// Test diff with more changes
	stats3 := GetDiffStats(sampleDiff3)
	assert.Equal(t, 1, stats3.FilesChanged)
	assert.Equal(t, 3, stats3.LinesAdded)
	assert.Equal(t, 1, stats3.LinesRemoved)
	assert.False(t, stats3.HasConflicts)

	// Test diff with conflicts
	statsConflict := GetDiffStats(sampleConflictDiff)
	assert.Equal(t, 1, statsConflict.FilesChanged)
	assert.Equal(t, 1, statsConflict.LinesAdded)
	assert.Equal(t, 1, statsConflict.LinesRemoved)
	assert.True(t, statsConflict.HasConflicts)

	// Test empty diff
	statsEmpty := GetDiffStats("")
	assert.Equal(t, 0, statsEmpty.FilesChanged)
	assert.Equal(t, 0, statsEmpty.LinesAdded)
	assert.Equal(t, 0, statsEmpty.LinesRemoved)
	assert.False(t, statsEmpty.HasConflicts)
}

func TestRemoveContextLines(t *testing.T) {
	// Remove context lines from a diff
	reduced := RemoveContextLines(sampleDiff1)

	// Should contain headers
	assert.Contains(t, reduced, "diff --git")
	assert.Contains(t, reduced, "@@ -1,3 +1,4 @@")

	// Should contain changed lines
	assert.Contains(t, reduced, "-line 2")
	assert.Contains(t, reduced, "+line 2 modified")
	assert.Contains(t, reduced, "+new line")

	// Should not contain context lines
	assert.NotContains(t, reduced, "line 1")
	assert.NotContains(t, reduced, "line 3")
}

func TestCompareDiffs(t *testing.T) {
	// Compare identical diffs
	assert.True(t, CompareDiffs(sampleDiff1, sampleDiff2), "Functionally equivalent diffs should compare as equal")

	// Compare different diffs
	assert.False(t, CompareDiffs(sampleDiff1, sampleDiff3), "Different diffs should compare as unequal")

	// Compare with an empty diff
	assert.False(t, CompareDiffs(sampleDiff1, ""), "Comparing with empty diff should be false")
	assert.False(t, CompareDiffs("", sampleDiff1), "Comparing with empty diff should be false")
	assert.True(t, CompareDiffs("", ""), "Comparing empty diffs should be true")
}

func TestFindLargestDiff(t *testing.T) {
	// Find the largest diff among several
	diffs := []string{sampleDiff1, sampleDiff3, sampleDiff2}
	largest := FindLargestDiff(diffs)

	// sampleDiff3 has the most changes (3 added, 1 removed)
	assert.Equal(t, sampleDiff3, largest, "Should select the diff with the most changes")

	// Test with conflict diffs
	diffsWithConflict := []string{sampleDiff1, sampleConflictDiff}
	largestWithoutConflict := FindLargestDiff(diffsWithConflict)

	// Should skip the conflict diff and select the clean one
	assert.Equal(t, sampleDiff1, largestWithoutConflict, "Should skip diffs with conflicts")

	// Test with empty list
	assert.Equal(t, "", FindLargestDiff([]string{}), "Should handle empty list")
}

func TestMergeDiffs(t *testing.T) {
	// Test merging with empty overlay diff
	merged, success := MergeDiffs(sampleDiff1, []string{})
	assert.True(t, success, "Merge with empty overlay should succeed")
	assert.Equal(t, sampleDiff1, merged, "Merging with empty overlay should return base diff")

	// Test merging with non-empty overlay diff

	// Sample diffs with changes to different parts
	sampleDiffA := `diff --git a/file.txt b/file.txt
@@ -1,5 +1,5 @@
 line 1
-bad line
+good line
 line 3
 line 4
 line 5
`

	sampleDiffB := `diff --git a/file.txt b/file.txt
@@ -3,7 +3,7 @@
 line 3
 line 4
 line 5
-wrong line
+correct line
 line 7
 line 8
`

	merged, success = MergeDiffs(sampleDiffA, []string{sampleDiffB})
	assert.True(t, success, "Merge should succeed")

	// Merged diff should contain changes from both
	assert.Contains(t, merged, "+good line")
	assert.Contains(t, merged, "+correct line")
}