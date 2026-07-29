package contextadapter

import (
	"errors"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

func TestCompositionPolicyNormalizationAndValidation(t *testing.T) {
	t.Parallel()

	defaults := DefaultCompositionPolicy()
	if defaults.MaxPromptBytes != defaultMaxContextPromptBytes ||
		defaults.MaxDocumentBytes != defaultMaxContextDocumentBytes ||
		defaults.Overflow != OverflowTruncate {
		t.Fatalf("DefaultCompositionPolicy()=%#v", defaults)
	}

	normalized := (CompositionPolicy{}).Normalized()
	if normalized != defaults {
		t.Fatalf("zero policy normalized to %#v, want %#v", normalized, defaults)
	}

	cases := []CompositionPolicy{
		{MaxPromptBytes: -1},
		{MaxPromptBytes: basespec.MaxDefinitionBodyBytes + 1},
		{MaxPromptBytes: 8, MaxDocumentBytes: 9},
		{MaxPromptBytes: 8, MaxDocumentBytes: -1},
		{MaxPromptBytes: 8, MaxDocumentBytes: 8, Overflow: "discard"},
	}
	for _, value := range cases {
		if err := value.Validate(); !errors.Is(err, engine.ErrInvalidWorkspace) {
			t.Errorf("CompositionPolicy(%#v).Validate() error=%v, want ErrInvalidWorkspace", value, err)
		}
	}
}

func TestApplyCompositionPolicyTruncatesWithoutBreakingUTF8(t *testing.T) {
	t.Parallel()

	value := testContribution("AGENTS.md", "agent-instructions", "a€bc")
	included, prompt, diagnostics, decisions := applyCompositionPolicy(
		CompositionPolicy{
			MaxPromptBytes:   256,
			MaxDocumentBytes: 2,
			Overflow:         OverflowTruncate,
		},
		[]ContextContribution{value},
		nil,
		nil,
	)

	if len(included) != 1 || included[0].Content != "a" ||
		included[0].IncludedBytes != 1 || !included[0].Truncated {
		t.Fatalf("included=%#v", included)
	}
	if !strings.Contains(prompt, "a\n<<<END_WORKSPACE_CONTEXT>>>") {
		t.Fatalf("prompt=%q", prompt)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != DiagnosticCodeContextDocumentTruncated {
		t.Fatalf("diagnostics=%#v", diagnostics)
	}
	if len(decisions) != 1 || decisions[0].Status != CompositionTruncated ||
		decisions[0].Code != DiagnosticCodeContextDocumentTruncated ||
		decisions[0].OriginalBytes != len("a€bc") || decisions[0].IncludedBytes != 1 {
		t.Fatalf("decisions=%#v", decisions)
	}

	for _, test := range []struct {
		maximum int
		want    string
	}{
		{maximum: 0, want: ""},
		{maximum: 1, want: "a"},
		{maximum: 2, want: "a"},
		{maximum: 4, want: "a€"},
		{maximum: len("a€bc"), want: "a€bc"},
	} {
		if got := truncateUTF8("a€bc", test.maximum); got != test.want {
			t.Errorf("truncateUTF8 maximum=%d got %q, want %q", test.maximum, got, test.want)
		}
	}
}

func TestApplyCompositionPolicyEnforcesAggregateBudgetAndExclusion(t *testing.T) {
	t.Parallel()

	first := testContribution("AGENTS.md", "agent-instructions", "one")
	second := testContribution("README.md", "project-readme", "abcdef")
	firstRendered := renderContextContribution(first, first.Content)
	emptySecondRendered := renderContextContribution(second, "")

	included, prompt, diagnostics, decisions := applyCompositionPolicy(
		CompositionPolicy{
			MaxPromptBytes:   len(firstRendered) + len(contextPromptSeparator) + len(emptySecondRendered) + 2,
			MaxDocumentBytes: 64,
			Overflow:         OverflowTruncate,
		},
		[]ContextContribution{first, second},
		nil,
		nil,
	)
	if len(included) != 2 || included[1].Content != "ab" || !included[1].Truncated {
		t.Fatalf("aggregate included=%#v", included)
	}
	if len(prompt) > len(firstRendered)+len(contextPromptSeparator)+len(emptySecondRendered)+2 {
		t.Fatalf("prompt exceeds aggregate budget: %d", len(prompt))
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != DiagnosticCodeContextBudgetExceeded {
		t.Fatalf("aggregate diagnostics=%#v", diagnostics)
	}
	if len(decisions) != 2 || decisions[1].Status != CompositionTruncated ||
		decisions[1].Code != DiagnosticCodeContextBudgetExceeded {
		t.Fatalf("aggregate decisions=%#v", decisions)
	}

	included, prompt, diagnostics, decisions = applyCompositionPolicy(
		CompositionPolicy{
			MaxPromptBytes:   len(firstRendered),
			MaxDocumentBytes: 64,
			Overflow:         OverflowTruncate,
		},
		[]ContextContribution{first, second},
		nil,
		nil,
	)
	if len(included) != 1 || prompt != firstRendered || len(diagnostics) != 1 ||
		diagnostics[0].Code != DiagnosticCodeContextBudgetExceeded ||
		len(decisions) != 2 || decisions[1].Status != CompositionExcluded {
		t.Fatalf(
			"exhausted budget included=%#v prompt=%q diagnostics=%#v decisions=%#v",
			included,
			prompt,
			diagnostics,
			decisions,
		)
	}

	included, prompt, diagnostics, decisions = applyCompositionPolicy(
		CompositionPolicy{
			MaxPromptBytes:   256,
			MaxDocumentBytes: 2,
			Overflow:         OverflowExclude,
		},
		[]ContextContribution{second},
		nil,
		nil,
	)
	if len(included) != 0 || prompt != "" || len(diagnostics) != 1 ||
		diagnostics[0].Code != DiagnosticCodeContextDocumentExcluded ||
		len(decisions) != 1 || decisions[0].Status != CompositionExcluded {
		t.Fatalf(
			"overflow exclude included=%#v prompt=%q diagnostics=%#v decisions=%#v",
			included,
			prompt,
			diagnostics,
			decisions,
		)
	}
}

func testContribution(locator basespec.Locator, role, content string) ContextContribution {
	return ContextContribution{
		Artifact: artifact.ArtifactRef{
			RootID:     "019d3150-6c01-7a6b-a34e-d9032342bc31",
			ArtifactID: "019d3150-6c02-7a6b-a34e-d9032342bc31",
		},
		Locator: locator,
		Name:    string(locator),
		Role:    role,
		Content: content,
	}
}
