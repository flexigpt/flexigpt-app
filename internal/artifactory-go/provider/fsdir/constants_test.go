package fsdir

import (
	"slices"
	"testing"
)

func TestDefaultTraversalPolicyOwnsItsExclusionSlice(t *testing.T) {
	t.Parallel()

	first := DefaultTraversalPolicy()
	second := DefaultTraversalPolicy()
	if !slices.Equal(first.ExcludedDirectoryNames, second.ExcludedDirectoryNames) {
		t.Fatalf(
			"default traversal exclusions differ: %v / %v",
			first.ExcludedDirectoryNames,
			second.ExcludedDirectoryNames,
		)
	}
	first.ExcludedDirectoryNames[0] = "changed"
	if second.ExcludedDirectoryNames[0] == "changed" {
		t.Fatal("DefaultTraversalPolicy exposed provider-owned exclusion storage")
	}
}
