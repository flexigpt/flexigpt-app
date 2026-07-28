package workspace

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

func TestDependenciesAndNewRejectIncompleteComposition(t *testing.T) {
	t.Parallel()

	if err := (Dependencies{}).Validate(); !errors.Is(err, engine.ErrInvalidWorkspace) {
		t.Fatalf("empty Dependencies.Validate error=%v", err)
	}
	api, err := New(Dependencies{}, DefaultConfig())
	if api != nil || !errors.Is(err, engine.ErrInvalidWorkspace) {
		t.Fatalf("New with incomplete dependencies api=%#v err=%v", api, err)
	}
}

func TestConfigDefaultsProfilesAndValidation(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	if len(config.Supports) != 2 || config.DiscoveryPolicyRevision != defaultDiscoveryPolicyRevision ||
		len(config.SkillRoots) != 1 || config.SkillRoots[0] != skilladapter.DefaultWorkspaceSkillRoot {
		t.Fatalf("DefaultConfig=%#v", config)
	}
	if err := config.ContextComposition.Validate(); err != nil {
		t.Fatalf("default ContextComposition: %v", err)
	}
	if config.runtimePolicy() == nil {
		t.Fatal("default runtime policy is nil")
	}

	supports := BuiltinArtifactSupports()
	supports[0].Kind = "changed.kind"
	if fresh := BuiltinArtifactSupports(); fresh[0].Kind == "changed.kind" {
		t.Fatalf("BuiltinArtifactSupports leaked mutable storage: %#v", fresh)
	}
	decoders := BuiltinDecoders()
	if len(decoders) != 2 || decoders[0].ID() == decoders[1].ID() {
		t.Fatalf("BuiltinDecoders=%#v", decoders)
	}
	profiles := BuiltinDiscoveryProfiles()
	if len(profiles.Primary.ExplicitLocators) == 0 || len(profiles.Primary.DirectoryRoots) == 0 ||
		len(profiles.Attached.DirectoryRoots) == 0 {
		t.Fatalf("BuiltinDiscoveryProfiles=%#v", profiles)
	}

	normalized := (Config{}).normalized()
	if len(normalized.Supports) != len(BuiltinArtifactSupports()) ||
		normalized.DiscoveryPolicyRevision != defaultDiscoveryPolicyRevision {
		t.Fatalf("normalized zero Config=%#v", normalized)
	}
	if _, err := (Config{}).normalizedSupports(); !errors.Is(err, engine.ErrInvalidWorkspace) {
		t.Fatalf("empty normalizedSupports error=%v", err)
	}
	duplicate := Config{Supports: []engine.ArtifactSupport{
		{
			Kind:      "test.kind",
			SchemaID:  "test.schema",
			DecoderID: "test.decoder",
			Validator: func(definition.Definition) error { return nil },
		},
		{
			Kind:      "test.kind",
			SchemaID:  "other.schema",
			DecoderID: "other.decoder",
			Validator: func(definition.Definition) error { return nil },
		},
	}}
	if _, err := duplicate.normalizedSupports(); !errors.Is(err, engine.ErrInvalidWorkspace) {
		t.Fatalf("duplicate support error=%v", err)
	}
	if _, err := (Config{DiscoveryPolicyRevision: " "}).discoveryPolicyRevision(); !errors.Is(
		err,
		engine.ErrInvalidWorkspace,
	) {
		t.Fatalf("invalid discovery policy revision error=%v", err)
	}
}

func TestViewsProjectOnlyOwnedAPISafeData(t *testing.T) {
	t.Parallel()

	discovery := engine.DiscoveryPreferences{
		AdditionalLocators: []artifactstore.Locator{"docs/guide.md"},
		AdditionalRoots:    []engine.DiscoveryRoot{{Root: "docs", Recursive: true, IncludePatterns: []string{"*.md"}}},
		IncludeReadme:      true,
	}
	viewDiscovery := workspaceDiscoveryOf(discovery)
	viewDiscovery.AdditionalLocators[0] = "changed.md"
	viewDiscovery.AdditionalRoots[0].IncludePatterns[0] = "*.txt"
	if discovery.AdditionalLocators[0] != "docs/guide.md" || discovery.AdditionalRoots[0].IncludePatterns[0] != "*.md" {
		t.Fatalf("workspaceDiscoveryOf leaked storage: %#v", discovery)
	}
	preferences := discoveryPreferencesOf(viewDiscovery)
	preferences.AdditionalRoots[0].IncludePatterns[0] = "*.json"
	if viewDiscovery.AdditionalRoots[0].IncludePatterns[0] != "*.txt" {
		t.Fatalf("discoveryPreferencesOf leaked storage: %#v", viewDiscovery)
	}

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	attachmentRaw := json.RawMessage(`{"recursive":true,"authoritative":false}`)
	workspaceValue := engine.Workspace{
		Collection: collection.Collection{
			ID:          "019d3150-7301-7a6b-a34e-d9032342bc31",
			RootID:      "019d3150-7302-7a6b-a34e-d9032342bc31",
			Kind:        engine.CollectionKind,
			DisplayName: "Workspace",
			Description: "Description",
			Enabled:     true,
			Revision:    3,
			CreatedAt:   now,
			ModifiedAt:  now,
		},
		Data:            engine.CollectionData{DiscoveryPolicyRevision: "policy.v1", Discovery: discovery},
		Mode:            engine.ModeFilesystem,
		PrimarySourceID: "019d3150-7303-7a6b-a34e-d9032342bc31",
		Attachments: []collection.Attachment{
			{
				SourceID: "019d3150-7303-7a6b-a34e-d9032342bc31",
				Role:     engine.RolePrimary,
				Enabled:  true,
				Data:     attachmentRaw,
				Revision: 2,
			},
		},
	}
	workspaceView, err := workspaceViewOf(workspaceValue)
	if err != nil || workspaceView.PrimarySourceID != workspaceValue.PrimarySourceID ||
		len(workspaceView.Attachments) != 1 ||
		workspaceView.Attachments[0].Settings.Recursive == nil ||
		!*workspaceView.Attachments[0].Settings.Recursive ||
		workspaceView.Attachments[0].Settings.Authoritative == nil ||
		*workspaceView.Attachments[0].Settings.Authoritative {
		t.Fatalf("workspaceView=%#v err=%v", workspaceView, err)
	}
	if _, err := workspaceAttachmentSettingsOf(json.RawMessage(`[]`)); !errors.Is(err, engine.ErrInvalidWorkspace) {
		t.Fatalf("invalid attachment settings error=%v", err)
	}
	settings := attachmentDataOf(workspaceView.Attachments[0].Settings)
	*settings.Recursive = false
	if !*workspaceView.Attachments[0].Settings.Recursive {
		t.Fatal("attachmentDataOf reused bool pointer")
	}

	definitionValue := definition.Definition{
		Digest:        artifactstore.DigestBytes([]byte("definition")),
		Kind:          "test.kind",
		SchemaID:      "test.schema",
		SchemaVersion: "v1",
		LogicalName:   "test",
		Labels:        map[string]string{"scope": "one"},
		Body:          json.RawMessage(`{"value":"one"}`),
		Dependencies:  []definition.Selector{{Kind: "other.kind", Labels: map[string]string{"scope": "two"}}},
	}
	definitionView := workspaceDefinitionViewOf(definitionValue)
	definitionView.Labels["scope"] = "changed"
	definitionView.Body[2] = 'X'
	definitionView.Dependencies[0].Labels["scope"] = "changed"
	if definitionValue.Labels["scope"] != "one" || string(definitionValue.Body) != `{"value":"one"}` ||
		definitionValue.Dependencies[0].Labels["scope"] != "two" {
		t.Fatalf("workspaceDefinitionViewOf leaked storage: %#v", definitionValue)
	}

	digest := artifactstore.DigestBytes([]byte("resolved"))
	artifactValue := artifact.Artifact{
		ID:           "019d3150-7304-7a6b-a34e-d9032342bc31",
		RootID:       workspaceValue.Collection.RootID,
		CollectionID: workspaceValue.Collection.ID,
		Binding: artifactstore.SourceBinding{
			SourceID:     workspaceValue.PrimarySourceID,
			Locator:      "AGENTS.md",
			ExpectedKind: "test.kind",
		},
		Kind:               "test.kind",
		Name:               "Artifact",
		Enabled:            true,
		Adoption:           artifact.AdoptionObserved,
		ResolvedDefinition: &digest,
		Data:               json.RawMessage(`[]`),
		State:              artifact.StateAvailable,
		Revision:           1,
		CreatedAt:          now,
		ModifiedAt:         now,
	}
	artifactView := workspaceArtifactViewOf(artifactValue)
	if artifactView.ResolvedDefinition == nil || len(artifactView.Diagnostics) != 1 ||
		artifactView.Diagnostics[0].Code != engine.DiagnosticCodeProjectionInvalid {
		t.Fatalf("workspaceArtifactViewOf=%#v", artifactView)
	}
	*artifactView.ResolvedDefinition = artifactstore.DigestBytes([]byte("changed"))
	if *artifactValue.ResolvedDefinition != digest {
		t.Fatal("workspaceArtifactViewOf reused digest pointer")
	}

	occurrenceDigest := artifactstore.DigestBytes([]byte("occurrence"))
	occurrence := catalog.Occurrence{
		Key: catalog.OccurrenceKey{
			CollectionID: workspaceValue.Collection.ID,
			SourceID:     workspaceValue.PrimarySourceID,
			Locator:      "AGENTS.md",
		},
		Kind:             "test.kind",
		LogicalName:      "agents",
		DefinitionDigest: &occurrenceDigest,
		State:            catalog.OccurrenceValid,
	}
	occurrenceView := workspaceOccurrenceViewOf(occurrence, map[string]artifact.Artifact{
		occurrenceViewKey(occurrence.Key.SourceID, occurrence.Key.Locator, "", occurrence.Kind): artifactValue,
	})
	if !occurrenceView.Recorded || occurrenceView.Artifact == nil ||
		occurrenceView.Artifact.ArtifactID != artifactValue.ID {
		t.Fatalf("workspaceOccurrenceViewOf=%#v", occurrenceView)
	}
}

func TestContextAndSkillViewConversionsOwnSlices(t *testing.T) {
	t.Parallel()

	diagnostic := artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticWarning,
		Code:     "test.warning",
		Message:  "warning",
	}
	contextPlan := contextadapter.ContextLoadPlan{
		Workspace: WorkspaceRef{
			RootID:       "019d3150-7305-7a6b-a34e-d9032342bc31",
			CollectionID: "019d3150-7306-7a6b-a34e-d9032342bc31",
		},
		CatalogRevision: 4,
		Diagnostics:     []artifactstore.Diagnostic{diagnostic},
		Contributions: []contextadapter.ContextContribution{
			{
				Artifact: artifactstore.ArtifactRef{
					RootID:     "019d3150-7305-7a6b-a34e-d9032342bc31",
					ArtifactID: "019d3150-7307-7a6b-a34e-d9032342bc31",
				},
				Role:      "agent-instructions",
				MediaType: "text/markdown",
				Content:   "content",
			},
		},
		Decisions:   []contextadapter.CompositionDecision{{Status: contextadapter.CompositionIncluded}},
		Prompt:      "prompt",
		PromptBytes: 6,
	}
	contextView := contextLoadPlanViewOf(contextPlan)
	contextView.Diagnostics[0].Message = "changed"
	if contextPlan.Diagnostics[0].Message != "warning" || len(contextView.Contributions) != 1 ||
		len(contextView.Decisions) != 1 {
		t.Fatalf("context view=%#v input=%#v", contextView, contextPlan)
	}

	skill := skilladapter.WorkspaceSkill{
		Skill: skilladapter.SkillSummary{
			Tags:      []string{"one"},
			Arguments: []skilladapter.SkillArgument{{Name: "argument"}},
		},
		Diagnostics: []artifactstore.Diagnostic{diagnostic},
	}
	skillView := workspaceSkillViewOf(skill)
	skillView.Skill.Tags[0] = "changed"
	skillView.Diagnostics[0].Message = "changed"
	if skill.Skill.Tags[0] != "one" || skill.Diagnostics[0].Message != "warning" {
		t.Fatalf("workspaceSkillViewOf leaked storage: %#v", skill)
	}
}
