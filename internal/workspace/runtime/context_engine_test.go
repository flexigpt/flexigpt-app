package runtime

import "testing"

func TestEngineComposeTruncatesAtUTF8Boundary(t *testing.T) {
	t.Parallel()

	result, err := NewEngine().Compose(
		CompositionPolicy{
			MaxPromptBytes:   1024,
			MaxDocumentBytes: 3,
			Overflow:         OverflowTruncate,
		},
		[]ContextContribution{{
			ID:      "context-1",
			Kind:    "context",
			Name:    "README.md",
			Locator: "README.md",
			Content: "éé",
		}},
	)
	if err != nil {
		t.Fatalf("compose context: %v", err)
	}

	if len(result.Contributions) != 1 {
		t.Fatalf("contribution count = %d, want 1", len(result.Contributions))
	}
	if got := result.Contributions[0].Content; got != "é" {
		t.Fatalf("content = %q, want %q", got, "é")
	}
	if got := result.Contributions[0].OriginalBytes; got != 4 {
		t.Fatalf("original bytes = %d, want 4", got)
	}
	if got := result.Contributions[0].IncludedBytes; got != 2 {
		t.Fatalf("included bytes = %d, want 2", got)
	}
	if !result.Contributions[0].Truncated {
		t.Fatal("contribution was not marked truncated")
	}

	if len(result.Decisions) != 1 {
		t.Fatalf("decision count = %d, want 1", len(result.Decisions))
	}
	if got := result.Decisions[0].Status; got != CompositionTruncated {
		t.Fatalf("decision status = %q, want %q", got, CompositionTruncated)
	}
}
