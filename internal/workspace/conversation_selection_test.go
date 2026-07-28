package workspace

import (
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
)

func TestConversationUsageStatusAndMappings(t *testing.T) {
	for _, test := range []struct {
		input contextadapter.CompositionStatus
		want  ConversationContextUsageStatus
	}{
		{contextadapter.CompositionIncluded, ConversationContextUsageIncluded},
		{contextadapter.CompositionTruncated, ConversationContextUsageTruncated},
		{contextadapter.CompositionExcluded, ConversationContextUsageExcluded},
		{contextadapter.CompositionDenied, ConversationContextUsageDenied},
		{contextadapter.CompositionUnavailable, ConversationContextUsageUnavailable},
		{"other", ConversationContextUsageUnavailable},
	} {
		if got := conversationContextUsageStatusOf(test.input); got != test.want {
			t.Errorf("conversationContextUsageStatusOf(%q)=%q, want %q", test.input, got, test.want)
		}
	}

	cases := []struct {
		name  string
		usage ConversationUsage
		want  ConversationSelectionStatus
	}{
		{name: "empty", want: ConversationSelectionReady},
		{
			name: "all usable",
			usage: ConversationUsage{
				Contexts: []ConversationContextUsage{{Status: ConversationContextUsageIncluded}},
				Skills:   []ConversationSkillUsage{{Status: ConversationSkillUsageAvailable}},
			},
			want: ConversationSelectionReady,
		},
		{
			name: "partial",
			usage: ConversationUsage{
				Contexts: []ConversationContextUsage{
					{Status: ConversationContextUsageTruncated},
					{Status: ConversationContextUsageDenied},
				},
			},
			want: ConversationSelectionPartial,
		},
		{
			name: "none",
			usage: ConversationUsage{
				Contexts: []ConversationContextUsage{{Status: ConversationContextUsageExcluded}},
				Skills:   []ConversationSkillUsage{{Status: ConversationSkillUsageUnavailable}},
			},
			want: ConversationSelectionUnavailable,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ResolveConversationUsageStatus(&test.usage)
			if test.usage.Status != test.want {
				t.Fatalf("status=%q, want %q", test.usage.Status, test.want)
			}
		})
	}
	ResolveConversationUsageStatus(nil)
}

func TestConversationSelectionHelpersTrackChangesAndUnavailableReferences(t *testing.T) {
	t.Parallel()

	if conversationResourceChanged("", "", 0, 0, "", "") {
		t.Fatal("empty selection was reported changed")
	}
	for _, test := range []struct {
		selectedDigest artifactstore.Digest
		usedDigest     artifactstore.Digest
		selectedRev    uint64
		usedRev        uint64
		selectedLoc    artifactstore.Locator
		usedLoc        artifactstore.Locator
	}{
		{selectedDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", usedDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{selectedRev: 1, usedRev: 2},
		{selectedLoc: "old.md", usedLoc: "new.md"},
	} {
		if !conversationResourceChanged(
			test.selectedDigest,
			test.usedDigest,
			test.selectedRev,
			test.usedRev,
			test.selectedLoc,
			test.usedLoc,
		) {
			t.Errorf("change was not detected for %#v", test)
		}
	}

	selection := ConversationSelection{
		Workspace: WorkspaceRef{
			RootID:       "019d3150-7401-7a6b-a34e-d9032342bc31",
			CollectionID: "019d3150-7402-7a6b-a34e-d9032342bc31",
		},
		DisplayName:       "Selected workspace",
		WorkspaceRevision: 3,
		CatalogRevision:   4,
		ContextRefs: []ConversationResourceSelectionRef{
			{
				Artifact: artifactstore.ArtifactRef{
					RootID:     "019d3150-7401-7a6b-a34e-d9032342bc31",
					ArtifactID: "019d3150-7403-7a6b-a34e-d9032342bc31",
				},
				Name: "Context",
			},
		},
		SkillRefs: []ConversationSkillSelectionRef{
			{
				ConversationResourceSelectionRef: ConversationResourceSelectionRef{
					Artifact: artifactstore.ArtifactRef{
						RootID:     "019d3150-7401-7a6b-a34e-d9032342bc31",
						ArtifactID: "019d3150-7404-7a6b-a34e-d9032342bc31",
					},
				},
				DisplayName: "Skill",
			},
		},
	}
	usage := unresolvedConversationUsage(selection, nil)
	if usage.Status != ConversationSelectionUnavailable || usage.DisplayName != selection.DisplayName ||
		len(usage.Contexts) != 1 || usage.Contexts[0].Status != ConversationContextUsageUnavailable ||
		len(usage.Skills) != 1 || usage.Skills[0].Status != ConversationSkillUsageUnavailable ||
		len(usage.Diagnostics) != 1 || usage.Diagnostics[0].Code != "workspace.conversation.unavailable" {
		t.Fatalf("unresolved usage=%#v", usage)
	}
	message := strings.Repeat("x", artifactstore.MaxDiagnosticMessageBytes+100)
	diagnostic := conversationSelectionDiagnostic("workspace.conversation.test", message)
	if len(diagnostic.Message) > artifactstore.MaxDiagnosticMessageBytes ||
		diagnostic.Code != "workspace.conversation.test" {
		t.Fatalf("bounded diagnostic=%#v", diagnostic)
	}
}
