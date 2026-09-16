package resolve

import (
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

func TestCapabilityPlanPreservesPartialRelationships(t *testing.T) {
	t.Parallel()

	root, err := declaration.NewEntry(declaration.Header{
		Type: declaration.TypeCollection,
		Name: "review",
	})
	if err != nil {
		t.Fatalf("create root entry: %v", err)
	}
	available, err := declaration.NewSymbolicEntry(
		declaration.TypeSkill,
		basespec.LogicalName("review-code"),
	)
	if err != nil {
		t.Fatalf("create available entry: %v", err)
	}
	missing, err := declaration.NewSymbolicEntry(
		declaration.TypeMCP,
		basespec.LogicalName("github"),
	)
	if err != nil {
		t.Fatalf("create missing entry: %v", err)
	}

	rootCopy := root.Clone()
	plan, err := CapabilityPlanForGraph(Graph{
		Root: &ResolvedEntry{
			Type:   declaration.TypeCollection,
			Inline: &rootCopy,
			MemberResults: []ResolvedRelationship{
				{
					Declared: available,
					Status:   ResolutionAvailable,
					Resolved: &ResolvedEntry{
						Type: declaration.TypeSkill,
					},
				},
				{
					Declared: missing,
					Status:   ResolutionUnavailable,
					Issue: &ResolutionIssue{
						Code:    "artifact.reference-unresolved",
						Message: "mcp/github is unavailable",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("build capability plan: %v", err)
	}
	if plan.Complete {
		t.Fatal("partial capability plan was marked complete")
	}
	if len(plan.Occurrences) != 2 {
		t.Fatalf(
			"occurrence count = %d, want 2",
			len(plan.Occurrences),
		)
	}
	if plan.Occurrences[1].Status != ResolutionUnavailable {
		t.Fatalf(
			"second status = %q, want unavailable",
			plan.Occurrences[1].Status,
		)
	}
	if err := RequireComplete(plan.Occurrences); !errors.Is(
		err,
		basespec.ErrReferenceUnresolved,
	) {
		t.Fatalf("RequireComplete error = %v", err)
	}
}
