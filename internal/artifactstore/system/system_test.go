package system

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
)

const (
	systemTestRootID       = "019d3150-6a30-7a6b-a34e-d9032342bc31"
	systemTestSourceID     = "019d3150-6a31-7a6b-a34e-d9032342bc31"
	systemTestCollectionID = "019d3150-6a32-7a6b-a34e-d9032342bc31"
	systemTestArtifactID   = "019d3150-6a33-7a6b-a34e-d9032342bc31"
)

type systemTestClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *systemTestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(time.Millisecond)
	return c.now
}

type systemTestIDs struct {
	mu     sync.Mutex
	values []string
}

func (g *systemTestIDs) NewID(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.values) == 0 {
		return "", errors.New("test ID generator exhausted")
	}
	value := g.values[0]
	g.values = g.values[1:]
	return value, nil
}

type systemTestDecoder struct{}

func (systemTestDecoder) ID() basespec.DecoderID { return "test.decoder" }
func (systemTestDecoder) Revision() string       { return "v1" }
func (systemTestDecoder) Recognize(context.Context, discovery.Candidate) discovery.Recognition {
	return discovery.RecognitionPreferred
}

func (systemTestDecoder) Decode(
	context.Context,
	discovery.Candidate,
) ([]discovery.Decoded, []diagnostic.Diagnostic) {
	return []discovery.Decoded{{Definition: definition.Definition{
		Kind:          "test.artifact",
		SchemaID:      "test.schema",
		SchemaVersion: "v1",
		LogicalName:   "generated",
		Body:          []byte(`{"source":"managed"}`),
	}}}, nil
}

func TestComponentsManagedLifecycleRefreshAndOptimisticConcurrency(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non win test")
	}
	ctx := t.Context()
	clock := &systemTestClock{now: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)}
	ids := &systemTestIDs{values: []string{
		systemTestRootID,
		systemTestSourceID,
		systemTestCollectionID,
		systemTestArtifactID,
	}}
	components, err := Open(ctx, Config{
		BaseDirectory: t.TempDir(),
		Clock:         clock,
		IDGenerator:   ids,
		Decoders:      []discovery.Decoder{systemTestDecoder{}},
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if err := components.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	rootValue, err := components.Roots.Create(ctx, root.RootDraft{DisplayName: "Test root"})
	if err != nil {
		t.Fatalf("Create root: %v", err)
	}
	if string(rootValue.ID) != systemTestRootID || rootValue.Revision != 1 {
		t.Fatalf("root=%#v", rootValue)
	}

	sourceValue, err := components.Sources.Create(ctx, rootValue.ID, source.Draft{
		Kind:        managed.Kind,
		DisplayName: "Managed source",
		Enabled:     true,
		Config:      []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Create source: %v", err)
	}
	collectionValue, attachments, err := components.Collections.Create(ctx, rootValue.ID, collection.Draft{
		Kind:        "test.collection",
		DisplayName: "Test collection",
		Enabled:     true,
		Data:        []byte(`{}`),
	}, []collection.AttachmentDraft{{
		SourceID: sourceValue.ID,
		Role:     "primary",
		Enabled:  true,
		Data:     []byte(`{}`),
	}})
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	if len(attachments) != 1 || attachments[0].Revision != 1 {
		t.Fatalf("attachments=%#v", attachments)
	}
	collectionRef := collectionValue.Ref()

	if _, err := components.Roots.Retire(
		ctx,
		rootValue.ID,
		rootValue.Revision,
	); !errors.Is(
		err,
		basespec.ErrConflict,
	) {
		t.Fatalf("retire root with children error=%v, want ErrConflict", err)
	}

	state, err := components.GetManagedSourceState(ctx, rootValue.ID, sourceValue.ID)
	if err != nil {
		t.Fatalf("GetManagedSourceState: %v", err)
	}
	publication := source.ManagedPackagePublication{
		Directory:          "packages/one",
		ExpectedGeneration: state.Generation,
		Files: []source.ManagedPackageFile{{
			Locator: "artifact.json",
			Content: []byte(`{"artifact":true}`),
		}},
	}
	published, err := components.PublishManagedPackage(
		ctx,
		rootValue.ID,
		sourceValue.ID,
		sourceValue.Revision,
		publication,
	)
	if err != nil {
		t.Fatalf("PublishManagedPackage: %v", err)
	}
	if published.Source.Revision != sourceValue.Revision+1 || published.Generation == state.Generation {
		t.Fatalf("published managed state=%#v initial=%#v", published, state)
	}
	idempotent, err := components.PublishManagedPackage(
		ctx,
		rootValue.ID,
		sourceValue.ID,
		published.Source.Revision,
		publication,
	)
	if err != nil {
		t.Fatalf("idempotent PublishManagedPackage: %v", err)
	}
	if idempotent.Source.Revision != published.Source.Revision || idempotent.Generation != published.Generation {
		t.Fatalf("idempotent result=%#v, published=%#v", idempotent, published)
	}
	if _, err := components.PublishManagedPackage(
		ctx,
		rootValue.ID,
		sourceValue.ID,
		sourceValue.Revision,
		publication,
	); !errors.Is(
		err,
		basespec.ErrConflict,
	) {
		t.Fatalf("stale source revision publication error=%v, want ErrConflict", err)
	}

	refreshResult, err := components.Refresh.Refresh(ctx, collectionRef, discovery.Plan{
		Revision: "test-refresh-v1",
		Sources: []discovery.SourcePlan{{
			SourceID:         sourceValue.ID,
			ExplicitLocators: []basespec.Locator{"packages/one/artifact.json"},
		}},
	}, artifactPolicyAdapter{})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if refreshResult.Catalog.Revision != 1 || len(refreshResult.CreatedArtifacts) != 1 ||
		refreshResult.Candidates != 1 {
		t.Fatalf("refresh result=%#v", refreshResult)
	}
	artifacts, err := components.Artifacts.ListByCollection(ctx, collectionRef)
	if err != nil {
		t.Fatalf("ListByCollection: %v", err)
	}
	if len(artifacts) != 1 || artifacts[0].ID != basespec.ArtifactID(systemTestArtifactID) ||
		string(artifacts[0].Data) != `{"a":1,"z":2}` {
		t.Fatalf("artifacts=%#v", artifacts)
	}
	currentArtifact := artifacts[0]

	if _, err := components.Sources.Retire(
		ctx,
		rootValue.ID,
		sourceValue.ID,
		published.Source.Revision,
	); !errors.Is(
		err,
		basespec.ErrConflict,
	) {
		t.Fatalf("retire attached source error=%v, want ErrConflict", err)
	}
	currentArtifact, err = components.Artifacts.SetEnabled(ctx, currentArtifact.Ref(), currentArtifact.Revision, false)
	if err != nil {
		t.Fatalf("SetEnabled false: %v", err)
	}
	currentArtifact, err = components.Artifacts.UpdateData(
		ctx,
		currentArtifact.Ref(),
		currentArtifact.Revision,
		[]byte(`{"z":3,"a":1}`),
	)
	if err != nil {
		t.Fatalf("UpdateData: %v", err)
	}
	if string(currentArtifact.Data) != `{"a":1,"z":3}` {
		t.Fatalf("canonical artifact data=%q", currentArtifact.Data)
	}

	start := make(chan struct{})
	type updateResult struct {
		value artifact.Artifact
		err   error
	}
	results := make(chan updateResult, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Go(func() {
			<-start
			value, err := components.Artifacts.SetEnabled(ctx, currentArtifact.Ref(), currentArtifact.Revision, true)
			results <- updateResult{value: value, err: err}
		})
	}
	close(start)
	group.Wait()
	close(results)
	successes := 0
	for result := range results {
		if result.err == nil {
			successes++
			currentArtifact = result.value
			continue
		}
		if !errors.Is(result.err, basespec.ErrConflict) {
			t.Fatalf("concurrent SetEnabled error=%v", result.err)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent SetEnabled successes=%d, want 1", successes)
	}

	secondPublication := source.ManagedPackagePublication{
		Directory:          "packages/two",
		ExpectedGeneration: published.Generation,
		Files:              []source.ManagedPackageFile{{Locator: "next.json", Content: []byte(`{"next":true}`)}},
	}
	changedSource, err := components.PublishManagedPackage(
		ctx,
		rootValue.ID,
		sourceValue.ID,
		published.Source.Revision,
		secondPublication,
	)
	if err != nil {
		t.Fatalf("second PublishManagedPackage: %v", err)
	}
	if changedSource.Source.Revision != published.Source.Revision+1 {
		t.Fatalf("changed source=%#v", changedSource)
	}
	if _, err := components.Catalogs.GetCurrent(ctx, collectionRef); !errors.Is(err, basespec.ErrCatalogStale) {
		t.Fatalf("catalog after source mutation error=%v, want ErrCatalogStale", err)
	}

	updatedCollection, err := components.Collections.Detach(
		ctx,
		collectionRef,
		sourceValue.ID,
		collectionValue.Revision,
		attachments[0].Revision,
	)
	if err != nil {
		t.Fatalf("Detach: %v", err)
	}
	if err := components.Artifacts.Purge(ctx, currentArtifact.Ref(), currentArtifact.Revision); err != nil {
		t.Fatalf("Purge artifact: %v", err)
	}
	retiredSource, err := components.Sources.Retire(ctx, rootValue.ID, sourceValue.ID, changedSource.Source.Revision)
	if err != nil {
		t.Fatalf("Retire detached source: %v", err)
	}
	retiredCollection, err := components.Collections.Retire(ctx, collectionRef, updatedCollection.Revision)
	if err != nil {
		t.Fatalf("Retire collection: %v", err)
	}
	retiredRoot, err := components.Roots.Retire(ctx, rootValue.ID, rootValue.Revision)
	if err != nil {
		t.Fatalf("Retire root: %v", err)
	}
	if err := components.Collections.Purge(ctx, collectionRef, retiredCollection.Revision); err != nil {
		t.Fatalf("Purge collection: %v", err)
	}
	if err := components.Sources.Purge(ctx, rootValue.ID, sourceValue.ID, retiredSource.Revision); err != nil {
		t.Fatalf("Purge source: %v", err)
	}
	if err := components.Roots.Purge(ctx, rootValue.ID, retiredRoot.Revision); err != nil {
		t.Fatalf("Purge root: %v", err)
	}
	if _, err := components.Roots.Get(ctx, rootValue.ID); !errors.Is(err, basespec.ErrRootNotFound) {
		t.Fatalf("Get purged root error=%v", err)
	}
}

// artifactPolicyAdapter uses the exact catalog type required by artifact.Policy.
type artifactPolicyAdapter struct{}

func (artifactPolicyAdapter) Derive(
	context.Context,
	collection.Collection,
	catalog.Occurrence,
	definition.Definition,
) (artifact.Draft, bool, []diagnostic.Diagnostic) {
	return artifact.Draft{Name: "Generated artifact", Enabled: true, Data: []byte(`{"z":2,"a":1}`)}, true, nil
}
