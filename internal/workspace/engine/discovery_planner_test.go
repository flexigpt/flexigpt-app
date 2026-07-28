package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
)

func TestPlannerBuildsIndependentRoleAwarePlans(t *testing.T) {
	t.Parallel()

	profiles := DiscoveryProfiles{
		Primary: DiscoveryProfile{
			ExplicitLocators: []artifactstore.Locator{"BASE.md"},
			ReadmeLocator:    "README.md",
			DirectoryRoots: []discovery.DirectoryRoot{{
				Root:            "workspace",
				Recursive:       true,
				IncludePatterns: []string{"*.md"},
			}},
		},
		Attached: DiscoveryProfile{DirectoryRoots: []discovery.DirectoryRoot{{
			Root:            ".",
			Recursive:       true,
			IncludePatterns: []string{"*.md"},
		}}},
	}
	planner, err := NewPlanner(profiles, "policy.v1", "decoder.z", "decoder.a", "decoder.z")
	if err != nil {
		t.Fatalf("NewPlanner: %v", err)
	}
	profiles.Primary.ExplicitLocators[0] = "mutated.md"
	profiles.Attached.DirectoryRoots[0].IncludePatterns[0] = "*.json"

	workspace := plannerTestWorkspace(t)
	expected := artifactstore.DigestBytes([]byte("member"))
	plan, err := planner.Build(workspace, DescriptorObservation{
		SourceID:   workspace.PrimarySourceID,
		Generation: "generation-1",
		Preferences: DiscoveryPreferences{
			AdditionalLocators: []artifactstore.Locator{"extra.md"},
			AdditionalRoots: []DiscoveryRoot{{
				Root:            "docs",
				Recursive:       false,
				IncludePatterns: []string{"*.md"},
			}},
			IncludeReadme: true,
		},
		ExpectedContentDigests: map[artifactstore.Locator]artifactstore.Digest{"extra.md": expected},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(plan.Sources) != 2 || plan.Revision != "policy.v1" {
		t.Fatalf("plan=%#v", plan)
	}

	var primary, attached discovery.SourcePlan
	for _, sourcePlan := range plan.Sources {
		switch sourcePlan.SourceID {
		case workspace.PrimarySourceID:
			primary = sourcePlan
		case workspace.Attachments[1].SourceID:
			attached = sourcePlan
		}
	}
	if len(primary.ExplicitLocators) != 3 || primary.ExplicitLocators[0] != "BASE.md" ||
		primary.ExplicitLocators[1] != "README.md" || primary.ExplicitLocators[2] != "extra.md" ||
		primary.ExpectedContentDigests["extra.md"] != expected ||
		len(primary.AllowedDecoderIDs) != 2 || primary.AllowedDecoderIDs[0] != "decoder.a" ||
		primary.AllowedDecoderIDs[1] != "decoder.z" {
		t.Fatalf("primary plan=%#v", primary)
	}
	if len(primary.DirectoryRoots) != 2 || primary.DirectoryRoots[0].Root != "docs" ||
		primary.DirectoryRoots[1].Root != "workspace" {
		t.Fatalf("primary roots=%#v", primary.DirectoryRoots)
	}
	if attached.Authoritative || len(attached.DirectoryRoots) != 1 || attached.DirectoryRoots[0].Recursive ||
		attached.DirectoryRoots[0].IncludePatterns[0] != "*.md" {
		t.Fatalf("attached plan=%#v", attached)
	}
	if len(attached.ExplicitLocators) != 0 || len(attached.ExpectedContentDigests) != 0 {
		t.Fatalf("attached plan unexpectedly inherited workspace preferences: %#v", attached)
	}

	primary.DirectoryRoots[0].IncludePatterns[0] = "*.changed"
	if attached.DirectoryRoots[0].IncludePatterns[0] == "*.changed" {
		t.Fatal("source plans unexpectedly share directory-root backing storage")
	}
}

func TestPlannerAndDiscoveryMergesRejectInvalidInputAndAreDeterministic(t *testing.T) {
	t.Parallel()

	if _, err := NewPlanner(DiscoveryProfiles{}, "policy.v1", "decoder.a"); !errors.Is(err, ErrInvalidWorkspace) {
		t.Fatalf("empty primary profile error=%v, want ErrInvalidWorkspace", err)
	}
	if _, err := NewPlanner(
		DiscoveryProfiles{Primary: DiscoveryProfile{ExplicitLocators: []artifactstore.Locator{"AGENTS.md"}}},
		" ",
		"decoder.a",
	); err == nil {
		t.Fatal("blank policy revision was accepted")
	}

	merged, err := mergeDiscoveryPreferences(
		DiscoveryPreferences{
			AdditionalLocators: []artifactstore.Locator{"b.md", "a.md"},
			AdditionalRoots:    []DiscoveryRoot{{Root: "docs", Recursive: false, IncludePatterns: []string{"*.md"}}},
		},
		DiscoveryPreferences{
			AdditionalLocators: []artifactstore.Locator{"a.md", "c.md"},
			AdditionalRoots: []DiscoveryRoot{
				{Root: "docs", Recursive: true, IncludePatterns: []string{"*.txt", "*.md"}},
			},
			IncludeReadme: true,
		},
	)
	if err != nil {
		t.Fatalf("mergeDiscoveryPreferences: %v", err)
	}
	if len(merged.AdditionalLocators) != 3 || merged.AdditionalLocators[0] != "a.md" ||
		merged.AdditionalLocators[2] != "c.md" || !merged.IncludeReadme ||
		len(merged.AdditionalRoots) != 1 || !merged.AdditionalRoots[0].Recursive ||
		len(merged.AdditionalRoots[0].IncludePatterns) != 2 || merged.AdditionalRoots[0].IncludePatterns[0] != "*.md" {
		t.Fatalf("merged preferences=%#v", merged)
	}
	if got := mergePatterns([]string{"*.md"}, nil); got != nil {
		t.Fatalf("mergePatterns with unrestricted side=%#v, want nil", got)
	}

	profile := MergeDiscoveryProfile(
		DiscoveryProfile{
			ExplicitLocators: []artifactstore.Locator{"AGENTS.md"},
			DirectoryRoots: []discovery.DirectoryRoot{
				{Root: "docs", Recursive: false, IncludePatterns: []string{"*.md"}},
			},
		},
		DiscoveryProfile{
			ExplicitLocators: []artifactstore.Locator{"AGENTS.md", "README.md"},
			ReadmeLocator:    "README.md",
			DirectoryRoots:   []discovery.DirectoryRoot{{Root: "docs", Recursive: true, IncludePatterns: nil}},
		},
	)
	if len(profile.ExplicitLocators) != 2 || profile.ReadmeLocator != "README.md" ||
		len(
			profile.DirectoryRoots,
		) != 1 || !profile.DirectoryRoots[0].Recursive || profile.DirectoryRoots[0].IncludePatterns != nil {
		t.Fatalf("merged profile=%#v", profile)
	}

	workspace := plannerTestWorkspace(t)
	workspace.Sources = workspace.Sources[:1]
	planner, err := NewPlanner(
		DiscoveryProfiles{Primary: DiscoveryProfile{ExplicitLocators: []artifactstore.Locator{"AGENTS.md"}}},
		"policy.v1",
		"decoder.a",
	)
	if err != nil {
		t.Fatalf("NewPlanner valid: %v", err)
	}
	if _, err := planner.Build(workspace, DescriptorObservation{}); !errors.Is(err, ErrInvalidWorkspace) {
		t.Fatalf("missing attachment source error=%v, want ErrInvalidWorkspace", err)
	}
}

func plannerTestWorkspace(t *testing.T) Workspace {
	t.Helper()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	collectionData, err := encodeCollectionData(CollectionData{
		DiscoveryPolicyRevision: "policy.v1",
		Discovery:               DiscoveryPreferences{IncludeReadme: true},
	})
	if err != nil {
		t.Fatalf("encode collection data: %v", err)
	}
	primaryData, err := encodeAttachmentData(AttachmentData{})
	if err != nil {
		t.Fatalf("encode primary data: %v", err)
	}
	falseValue := false
	attachedData, err := encodeAttachmentData(AttachmentData{Recursive: &falseValue, Authoritative: &falseValue})
	if err != nil {
		t.Fatalf("encode attached data: %v", err)
	}
	rootID := artifactstore.RootID("019d3150-7001-7a6b-a34e-d9032342bc31")
	collectionID := artifactstore.CollectionID("019d3150-7002-7a6b-a34e-d9032342bc31")
	primaryID := artifactstore.SourceID("019d3150-7003-7a6b-a34e-d9032342bc31")
	attachedID := artifactstore.SourceID("019d3150-7004-7a6b-a34e-d9032342bc31")
	return Workspace{
		Collection: collection.Collection{
			ID:          collectionID,
			RootID:      rootID,
			Kind:        CollectionKind,
			DisplayName: "Workspace",
			Enabled:     true,
			Data:        collectionData,
			Revision:    1,
			CreatedAt:   now,
			ModifiedAt:  now,
		},
		Data: CollectionData{
			DiscoveryPolicyRevision: "policy.v1",
			Discovery:               DiscoveryPreferences{IncludeReadme: true},
		},
		Mode:            ModeFilesystem,
		PrimarySourceID: primaryID,
		Attachments: []collection.Attachment{
			{
				RootID:       rootID,
				CollectionID: collectionID,
				SourceID:     primaryID,
				Role:         RolePrimary,
				Enabled:      true,
				Data:         primaryData,
				Revision:     1,
				CreatedAt:    now,
				ModifiedAt:   now,
			},
			{
				RootID:       rootID,
				CollectionID: collectionID,
				SourceID:     attachedID,
				Role:         RoleLibrary,
				Enabled:      true,
				Data:         attachedData,
				Revision:     1,
				CreatedAt:    now,
				ModifiedAt:   now,
			},
		},
		Sources: []source.Summary{
			{
				ID:          primaryID,
				RootID:      rootID,
				Kind:        fsdir.Kind,
				DisplayName: "Primary",
				Enabled:     true,
				Revision:    1,
				CreatedAt:   now,
				ModifiedAt:  now,
			},
			{
				ID:          attachedID,
				RootID:      rootID,
				Kind:        "test.source",
				DisplayName: "Library",
				Enabled:     true,
				Revision:    1,
				CreatedAt:   now,
				ModifiedAt:  now,
			},
		},
	}
}
