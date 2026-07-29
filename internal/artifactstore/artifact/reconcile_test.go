package artifact

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	artifactTestRootID       basespec.RootID       = "019d3150-6a25-7a6b-a34e-d9032342bc31"
	artifactTestCollectionID basespec.CollectionID = "019d3150-6a26-7a6b-a34e-d9032342bc31"
	artifactTestSourceID     basespec.SourceID     = "019d3150-6a27-7a6b-a34e-d9032342bc31"
	artifactTestExistingID   basespec.ArtifactID   = "019d3150-6a28-7a6b-a34e-d9032342bc31"
	artifactTestCreatedID    string                = "019d3150-6a29-7a6b-a34e-d9032342bc31"
)

type artifactTestClock struct{ now time.Time }

func (c artifactTestClock) Now() time.Time { return c.now }

type artifactTestIDs struct{ values []string }

func (g *artifactTestIDs) NewID(context.Context) (string, error) {
	if len(g.values) == 0 {
		return "", errors.New("no test ID")
	}
	value := g.values[0]
	g.values = g.values[1:]
	return value, nil
}

type artifactTestDefinitions struct {
	values map[cryptoutil.Digest]definition.Definition
}

func (r artifactTestDefinitions) Get(
	ctx context.Context,
	rootID basespec.RootID,
	digest cryptoutil.Digest,
) (definition.Definition, error) {
	value, found := r.values[digest]
	if !found {
		return definition.Definition{}, basespec.ErrDefinitionNotFound
	}
	return value, nil
}

type artifactTestPolicy struct {
	draft       Draft
	create      bool
	diagnostics []diagnostic.Diagnostic
}

func (p artifactTestPolicy) Derive(
	context.Context,
	collection.Collection,
	catalog.Occurrence,
	definition.Definition,
) (Draft, bool, []diagnostic.Diagnostic) {
	return p.draft, p.create, p.diagnostics
}

func artifactTestDefinition(t *testing.T) definition.Definition {
	t.Helper()
	value, err := definition.Canonicalize(definition.Definition{
		Kind:          "test.artifact",
		SchemaID:      "test.schema",
		SchemaVersion: "v1",
		LogicalName:   "example",
		Body:          []byte(`{"value":true}`),
	})
	if err != nil {
		t.Fatalf("canonical test definition: %v", err)
	}
	return value
}

func artifactTestCollection(now time.Time) collection.Collection {
	return collection.Collection{
		ID:          artifactTestCollectionID,
		RootID:      artifactTestRootID,
		Kind:        "test.collection",
		DisplayName: "Test collection",
		Enabled:     true,
		Data:        []byte(`{}`),
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
}

func artifactTestOccurrence(
	locator basespec.Locator,
	kind basespec.ArtifactKind,
	digest cryptoutil.Digest,
	now time.Time,
) catalog.Occurrence {
	contentDigest := cryptoutil.DigestBytes([]byte("content:" + string(locator)))
	return catalog.Occurrence{
		RootID:       artifactTestRootID,
		CollectionID: artifactTestCollectionID,
		Key: catalog.OccurrenceKey{
			CollectionID: artifactTestCollectionID,
			SourceID:     artifactTestSourceID,
			Locator:      locator,
		},
		Kind:                kind,
		LogicalName:         "example",
		DefinitionDigest:    &digest,
		SourceContentDigest: &contentDigest,
		DecoderID:           "test.decoder",
		State:               catalog.OccurrenceValid,
		ObservedAt:          now,
	}
}

func TestReconcilerUpdatesKindChangesAndCreatesNewObservedArtifacts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	definitionValue := artifactTestDefinition(t)
	reconciler, err := NewReconciler(
		&artifactTestIDs{values: []string{artifactTestCreatedID}},
		artifactTestClock{now: now},
	)
	if err != nil {
		t.Fatalf("NewReconciler: %v", err)
	}
	collectionValue := artifactTestCollection(now)
	binding := SourceBinding{
		SourceID:     artifactTestSourceID,
		Locator:      "changed.json",
		ExpectedKind: "test.artifact",
	}
	resolved := definitionValue.Digest
	existing := Artifact{
		ID:                 artifactTestExistingID,
		RootID:             artifactTestRootID,
		CollectionID:       artifactTestCollectionID,
		Binding:            binding,
		Kind:               "test.artifact",
		Name:               "Existing artifact",
		Enabled:            true,
		Adoption:           AdoptionObserved,
		ResolvedDefinition: &resolved,
		Data:               []byte(`{}`),
		State:              StateAvailable,
		Revision:           1,
		CreatedAt:          now,
		ModifiedAt:         now,
	}
	changed := artifactTestOccurrence("changed.json", "test.changed", definitionValue.Digest, now)
	newValue := artifactTestOccurrence("new.json", "test.artifact", definitionValue.Digest, now)
	result, err := reconciler.Reconcile(
		t.Context(),
		collectionValue,
		[]catalog.Occurrence{newValue, changed},
		[]Artifact{existing},
		nil,
		artifactTestDefinitions{
			values: map[cryptoutil.Digest]definition.Definition{definitionValue.Digest: definitionValue},
		},
		artifactTestPolicy{
			draft:  Draft{Name: "New artifact", Enabled: true, Data: []byte(`{"z":2,"a":1}`)},
			create: true,
		},
	)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(result.Updates) != 1 || result.Updates[0].State != StateIncompatible || result.Updates[0].Revision != 2 {
		t.Fatalf("updates=%#v", result.Updates)
	}
	if len(result.Updates[0].Diagnostics) != 1 ||
		result.Updates[0].Diagnostics[0].Code != "artifact.kind-incompatible" {
		t.Fatalf("incompatible diagnostics=%#v", result.Updates[0].Diagnostics)
	}
	if !result.Updates[0].ModifiedAt.After(existing.ModifiedAt) {
		t.Fatalf("update timestamp=%v did not advance %v", result.Updates[0].ModifiedAt, existing.ModifiedAt)
	}
	if len(result.Creates) != 1 {
		t.Fatalf("creates=%#v", result.Creates)
	}
	created := result.Creates[0]
	if created.ID != basespec.ArtifactID(artifactTestCreatedID) || created.Adoption != AdoptionObserved ||
		created.State != StateAvailable ||
		string(created.Data) != `{"a":1,"z":2}` {
		t.Fatalf("created artifact=%#v", created)
	}
}

func TestReconcilerHonorsSuppressionsAndRejectsDuplicatePhysicalOccurrences(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	definitionValue := artifactTestDefinition(t)
	reconciler, err := NewReconciler(
		&artifactTestIDs{values: []string{artifactTestCreatedID}},
		artifactTestClock{now: now},
	)
	if err != nil {
		t.Fatalf("NewReconciler: %v", err)
	}
	occurrence := artifactTestOccurrence("new.json", "test.artifact", definitionValue.Digest, now)
	suppression := Suppression{
		RootID:       artifactTestRootID,
		CollectionID: artifactTestCollectionID,
		Binding: SourceBinding{
			SourceID:     artifactTestSourceID,
			Locator:      "new.json",
			ExpectedKind: "test.artifact",
		},
		Revision:   1,
		CreatedAt:  now,
		ModifiedAt: now,
	}
	result, err := reconciler.Reconcile(
		t.Context(),
		artifactTestCollection(now),
		[]catalog.Occurrence{occurrence},
		nil,
		[]Suppression{suppression},
		artifactTestDefinitions{
			values: map[cryptoutil.Digest]definition.Definition{definitionValue.Digest: definitionValue},
		},
		artifactTestPolicy{draft: Draft{Name: "ignored", Data: []byte(`{}`)}, create: true},
	)
	if err != nil {
		t.Fatalf("suppressed Reconcile: %v", err)
	}
	if len(result.Creates) != 0 {
		t.Fatalf("suppression did not prevent creation: %#v", result.Creates)
	}

	duplicate := occurrence
	duplicate.Kind = "test.other"
	if _, err := reconciler.Reconcile(
		t.Context(),
		artifactTestCollection(now),
		[]catalog.Occurrence{occurrence, duplicate},
		nil,
		nil,
		artifactTestDefinitions{
			values: map[cryptoutil.Digest]definition.Definition{definitionValue.Digest: definitionValue},
		},
		artifactTestPolicy{},
	); !errors.Is(
		err,
		basespec.ErrInvalid,
	) {
		t.Fatalf("duplicate physical occurrence error=%v, want ErrInvalid", err)
	}
}
