package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/attachmentdata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/discovery"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

func TestQueryServiceCatalogResolveAndLoadPlan(t *testing.T) {
	t.Parallel()

	workspace := plannerTestWorkspace(t)
	definitionValue := queryTestDefinition(t)
	record := queryTestArtifact(t, workspace, definitionValue.Digest)
	snapshot := queryTestSnapshot(workspace, definitionValue.Digest, record.Binding)
	catalogErr := error(nil)

	store := &engineTestCollectionStore{
		getFn: func(_ context.Context, ref collection.CollectionRef) (collectionValue collection.Collection, err error) {
			if ref != workspace.Collection.Ref() {
				return collection.Collection{}, basespec.ErrCollectionNotFound
			}
			return workspace.Collection, nil
		},
		listAttachmentsFn: func(_ context.Context, ref collection.CollectionRef) ([]collection.Attachment, error) {
			if ref != workspace.Collection.Ref() {
				return nil, basespec.ErrCollectionNotFound
			}
			return append([]collection.Attachment(nil), workspace.Attachments...), nil
		},
	}
	sources := engineTestSources{
		getFn: func(_ context.Context, rootID basespec.RootID, sourceID basespec.SourceID) (source.Summary, error) {
			if rootID != workspace.Collection.RootID {
				return source.Summary{}, basespec.ErrSourceNotFound
			}
			for _, item := range workspace.Sources {
				if item.ID == sourceID {
					return item, nil
				}
			}
			return source.Summary{}, basespec.ErrSourceNotFound
		},
	}
	workspaces, err := NewService(store, sources, "policy.v1")
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	catalogs := engineTestCatalogs{getFn: func(context.Context, collection.CollectionRef) (catalog.Snapshot, error) {
		return snapshot, catalogErr
	}}
	artifacts := engineTestArtifacts{
		getFn: func(_ context.Context, ref artifact.ArtifactRef) (artifact.Artifact, error) {
			if ref == record.Ref() {
				return record, nil
			}
			return artifact.Artifact{}, basespec.ErrArtifactNotFound
		},
		listFn: func(_ context.Context, ref collection.CollectionRef) ([]artifact.Artifact, error) {
			if ref != workspace.Collection.Ref() {
				return nil, basespec.ErrCollectionNotFound
			}
			return []artifact.Artifact{record}, nil
		},
	}
	definitions := engineTestDefinitions{
		getFn: func(_ context.Context, rootID basespec.RootID, digest cryptoutil.Digest) (definition.Definition, error) {
			if rootID != workspace.Collection.RootID || digest != definitionValue.Digest {
				return definition.Definition{}, basespec.ErrDefinitionNotFound
			}
			return definitionValue, nil
		},
	}
	query, err := NewQueryService(
		workspaces,
		catalogs,
		artifacts,
		definitions,
		func() (cryptoutil.Digest, error) { return snapshot.DecoderFingerprint, nil },
		"policy.v1",
		spec.ArtifactSupport{
			Kind:      "test.kind",
			SchemaID:  "test.schema",
			DecoderID: "test.decoder",
			Validator: func(definition.Definition) error { return nil },
		},
	)
	if err != nil {
		t.Fatalf("NewQueryService: %v", err)
	}

	view, err := query.Catalog(t.Context(), workspace.Collection.Ref())
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if !view.CatalogCurrent || len(view.Resources) != 1 || !view.Resources[0].ProjectionValid ||
		view.Resources[0].Definition.Digest != definitionValue.Digest || len(view.Groups) != 1 {
		t.Fatalf("catalog view=%#v", view)
	}

	resolved, err := query.Resolve(
		t.Context(),
		workspace.Collection.Ref(),
		spec.Reference{Artifact: new(record.Ref())},
	)
	if err != nil || resolved.Artifact.ID != record.ID {
		t.Fatalf("Resolve artifact=%#v err=%v", resolved, err)
	}
	resolved, err = query.Resolve(
		t.Context(),
		workspace.Collection.Ref(),
		spec.Reference{
			Selector: &definition.Selector{Kind: "test.kind", LogicalName: "first", VersionConstraint: "=v1"},
		},
	)
	if err != nil || resolved.Artifact.ID != record.ID {
		t.Fatalf("Resolve selector=%#v err=%v", resolved, err)
	}
	if _, err := query.Resolve(
		t.Context(),
		workspace.Collection.Ref(),
		spec.Reference{},
	); !errors.Is(
		err,
		spec.ErrReferenceUnresolved,
	) {
		t.Fatalf("empty reference error=%v", err)
	}
	if _, err := query.Resolve(
		t.Context(),
		workspace.Collection.Ref(),
		spec.Reference{Selector: &definition.Selector{Kind: "test.kind", VersionConstraint: ">=v1"}},
	); !errors.Is(
		err,
		spec.ErrReferenceUnresolved,
	) {
		t.Fatalf("unsupported selector error=%v", err)
	}
	wrongRoot := record.Ref()
	wrongRoot.RootID = "019d3150-7209-7a6b-a34e-d9032342bc31"
	if _, err := query.Resolve(
		t.Context(),
		workspace.Collection.Ref(),
		spec.Reference{Artifact: &wrongRoot},
	); !errors.Is(
		err,
		spec.ErrReferenceUnresolved,
	) {
		t.Fatalf("cross-root reference error=%v", err)
	}

	plan, err := query.ComposeLoadPlan(
		t.Context(),
		workspace.Collection.Ref(),
		[]artifact.ArtifactRef{record.Ref()},
	)
	if err != nil || len(plan.Items) != 1 || plan.Items[0].Definition.Digest != definitionValue.Digest ||
		plan.Items[0].SourceGeneration != snapshot.SourceGenerations[record.Binding.SourceID] {
		t.Fatalf("ComposeLoadPlan=%#v err=%v", plan, err)
	}
	if _, err := query.ComposeLoadPlan(
		t.Context(),
		workspace.Collection.Ref(),
		[]artifact.ArtifactRef{record.Ref(), record.Ref()},
	); !errors.Is(
		err,
		spec.ErrInvalidWorkspace,
	) {
		t.Fatalf("duplicate load-plan artifact error=%v", err)
	}

	catalogErr = basespec.ErrCatalogStale
	view, err = query.Catalog(t.Context(), workspace.Collection.Ref())
	if err != nil || view.CatalogCurrent || len(view.FreshnessDiagnostics) == 0 ||
		view.FreshnessDiagnostics[0].Code != DiagnosticCodeCatalogStale {
		t.Fatalf("stale Catalog view=%#v err=%v", view, err)
	}
	if _, err := query.Resolve(
		t.Context(),
		workspace.Collection.Ref(),
		spec.Reference{Artifact: new(record.Ref())},
	); !errors.Is(
		err,
		basespec.ErrCatalogStale,
	) {
		t.Fatalf("stale Resolve error=%v, want ErrCatalogStale", err)
	}
}

func TestQueryServiceConstructorAndSelectorBoundaries(t *testing.T) {
	t.Parallel()

	if _, err := NewQueryService(nil, nil, nil, nil, nil, "policy.v1"); !errors.Is(err, spec.ErrInvalidWorkspace) {
		t.Fatalf("NewQueryService nil dependencies error=%v", err)
	}
	if err := validateWorkspaceSelector(
		definition.Selector{Kind: "test.kind", VersionConstraint: "="},
	); !errors.Is(
		err,
		spec.ErrReferenceUnresolved,
	) {
		t.Fatalf("empty exact selector error=%v", err)
	}
	if err := validateWorkspaceSelector(definition.Selector{Kind: "test.kind", VersionConstraint: "v1"}); err != nil {
		t.Fatalf("exact plain selector error=%v", err)
	}
	if matchesSelector(
		definition.Definition{Kind: "test.kind", LogicalVersion: "v1", Labels: map[string]string{"scope": "one"}},
		definition.Selector{Kind: "test.kind", VersionConstraint: "v2"},
	) {
		t.Fatal("matchesSelector accepted a different version")
	}
}

func queryTestDefinition(t *testing.T) definition.Definition {
	t.Helper()
	value, err := definition.Canonicalize(definition.Definition{
		Kind:           "test.kind",
		SchemaID:       "test.schema",
		SchemaVersion:  "v1",
		LogicalName:    "first",
		LogicalVersion: "v1",
		DisplayName:    "First",
		Labels:         map[string]string{"scope": "one"},
		Body:           []byte(`{"value":"one"}`),
	})
	if err != nil {
		t.Fatalf("Canonicalize test definition: %v", err)
	}
	return value
}

func queryTestArtifact(t *testing.T, workspace spec.Workspace, digest cryptoutil.Digest) artifact.Artifact {
	t.Helper()
	raw, err := EncodeArtifactData(spec.ArtifactData{})
	if err != nil {
		t.Fatalf("EncodeArtifactData: %v", err)
	}
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	return artifact.Artifact{
		ID:           "019d3150-7205-7a6b-a34e-d9032342bc31",
		RootID:       workspace.Collection.RootID,
		CollectionID: workspace.Collection.ID,
		Binding: artifact.SourceBinding{
			SourceID:     workspace.PrimarySourceID,
			Locator:      "docs/first.md",
			ExpectedKind: "test.kind",
		},
		Kind:               "test.kind",
		Name:               "First",
		Enabled:            true,
		Adoption:           artifact.AdoptionObserved,
		ResolvedDefinition: &digest,
		Data:               raw,
		State:              artifact.StateAvailable,
		Revision:           1,
		CreatedAt:          now,
		ModifiedAt:         now,
	}
}

func queryTestSnapshot(
	workspace spec.Workspace,
	digest cryptoutil.Digest,
	binding artifact.SourceBinding,
) catalog.Snapshot {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	sourceContent := cryptoutil.DigestBytes([]byte("source"))
	return catalog.Snapshot{
		RootID:             workspace.Collection.RootID,
		CollectionID:       workspace.Collection.ID,
		Revision:           1,
		CollectionRevision: workspace.Collection.Revision,
		AttachmentRevisions: map[basespec.SourceID]uint64{
			workspace.Attachments[0].SourceID: workspace.Attachments[0].Revision,
			workspace.Attachments[1].SourceID: workspace.Attachments[1].Revision,
		},
		SourceRevisions: map[basespec.SourceID]uint64{
			workspace.Sources[0].ID: workspace.Sources[0].Revision,
			workspace.Sources[1].ID: workspace.Sources[1].Revision,
		},
		SourceGenerations: map[basespec.SourceID]string{
			workspace.Sources[0].ID: "generation-primary",
			workspace.Sources[1].ID: "generation-attached",
		},
		PlanFingerprint:    cryptoutil.DigestBytes([]byte("plan")),
		DecoderFingerprint: cryptoutil.DigestBytes([]byte("decoders")),
		PublishedAt:        now,
		Occurrences: []catalog.Occurrence{{
			RootID:       workspace.Collection.RootID,
			CollectionID: workspace.Collection.ID,
			Key: catalog.OccurrenceKey{
				CollectionID: workspace.Collection.ID,
				SourceID:     binding.SourceID,
				Locator:      binding.Locator,
			},
			Kind:                binding.ExpectedKind,
			LogicalName:         "first",
			LogicalVersion:      "v1",
			DefinitionDigest:    &digest,
			SourceContentDigest: &sourceContent,
			DecoderID:           "test.decoder",
			State:               catalog.OccurrenceValid,
			ObservedAt:          now,
		}},
	}
}

func plannerTestWorkspace(t *testing.T) spec.Workspace {
	t.Helper()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	collectionData, err := discovery.EncodeCollectionData(spec.CollectionData{
		DiscoveryPolicyRevision: "policy.v1",
		Discovery:               spec.DiscoveryPreferences{IncludeReadme: true},
	})
	if err != nil {
		t.Fatalf("encode collection data: %v", err)
	}
	primaryData, err := attachmentdata.EncodeAttachmentData(spec.AttachmentData{})
	if err != nil {
		t.Fatalf("encode primary data: %v", err)
	}
	falseValue := false
	attachedData, err := attachmentdata.EncodeAttachmentData(
		spec.AttachmentData{Recursive: &falseValue, Authoritative: &falseValue},
	)
	if err != nil {
		t.Fatalf("encode attached data: %v", err)
	}
	rootID := basespec.RootID("019d3150-7001-7a6b-a34e-d9032342bc31")
	collectionID := basespec.CollectionID("019d3150-7002-7a6b-a34e-d9032342bc31")
	primaryID := basespec.SourceID("019d3150-7003-7a6b-a34e-d9032342bc31")
	attachedID := basespec.SourceID("019d3150-7004-7a6b-a34e-d9032342bc31")
	return spec.Workspace{
		Collection: collection.Collection{
			ID:          collectionID,
			RootID:      rootID,
			Kind:        spec.CollectionKind,
			DisplayName: "Workspace",
			Enabled:     true,
			Data:        collectionData,
			Revision:    1,
			CreatedAt:   now,
			ModifiedAt:  now,
		},
		Data: spec.CollectionData{
			DiscoveryPolicyRevision: "policy.v1",
			Discovery:               spec.DiscoveryPreferences{IncludeReadme: true},
		},
		Mode:            spec.ModeFilesystem,
		PrimarySourceID: primaryID,
		Attachments: []collection.Attachment{
			{
				RootID:       rootID,
				CollectionID: collectionID,
				SourceID:     primaryID,
				Role:         spec.RolePrimary,
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
				Role:         spec.RoleLibrary,
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
