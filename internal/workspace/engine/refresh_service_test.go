package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func TestRefresherBuildsPlanAndDelegatesToRunner(t *testing.T) {
	t.Parallel()

	workspace := refresherTestWorkspace(t)
	store := &engineTestCollectionStore{
		getFn: func(_ context.Context, ref collection.CollectionRef) (collection.Collection, error) {
			if ref != workspace.Collection.Ref() {
				return collection.Collection{}, basespec.ErrCollectionNotFound
			}
			return workspace.Collection, nil
		},
		listAttachmentsFn: func(context.Context, collection.CollectionRef) ([]collection.Attachment, error) {
			return nil, nil
		},
	}
	service, err := NewService(store, engineTestSources{}, "policy.v1")
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	loader, err := NewDescriptorLoader(engineTestRuntime{})
	if err != nil {
		t.Fatalf("NewDescriptorLoader: %v", err)
	}
	planner, err := NewPlanner(
		DiscoveryProfiles{Primary: DiscoveryProfile{ExplicitLocators: []basespec.Locator{"AGENTS.md"}}},
		"policy.v1",
		"test.decoder",
	)
	if err != nil {
		t.Fatalf("NewPlanner: %v", err)
	}
	policy, err := NewArtifactPolicy(
		ArtifactSupport{
			Kind:      "test.kind",
			SchemaID:  "test.schema",
			DecoderID: "test.decoder",
			Validator: func(definition.Definition) error { return nil },
		},
	)
	if err != nil {
		t.Fatalf("NewArtifactPolicy: %v", err)
	}
	var received discovery.Plan
	runner := refresherTestRunner{
		refreshFn: func(_ context.Context, ref collection.CollectionRef, plan discovery.Plan, receivedPolicy artifact.Policy) (refresh.Result, error) {
			if ref != workspace.Collection.Ref() || receivedPolicy != policy {
				t.Fatalf("runner ref=%#v policy=%#v", ref, receivedPolicy)
			}
			received = plan
			return refresh.Result{Candidates: 2}, nil
		},
	}
	refresher, err := NewRefresher(service, loader, planner, runner, policy)
	if err != nil {
		t.Fatalf("NewRefresher: %v", err)
	}
	result, err := refresher.Refresh(t.Context(), workspace.Collection.Ref())
	if err != nil || result.Candidates != 2 || received.Revision != "policy.v1" || len(received.Sources) != 0 {
		t.Fatalf("Refresh result=%#v plan=%#v err=%v", result, received, err)
	}
	if _, err := refresher.Refresh(t.Context(), collection.CollectionRef{}); err == nil {
		t.Fatal("Refresh accepted an invalid workspace reference")
	}
}

func TestRefresherConstructorRejectsMissingDependencies(t *testing.T) {
	t.Parallel()

	if _, err := NewRefresher(nil, nil, nil, nil, nil); !errors.Is(err, ErrInvalidWorkspace) {
		t.Fatalf("NewRefresher nil dependencies error=%v", err)
	}
}

type refresherTestRunner struct {
	refreshFn func(context.Context, collection.CollectionRef, discovery.Plan, artifact.Policy) (refresh.Result, error)
}

func (r refresherTestRunner) Refresh(
	ctx context.Context,
	ref collection.CollectionRef,
	plan discovery.Plan,
	policy artifact.Policy,
) (refresh.Result, error) {
	if r.refreshFn == nil {
		return refresh.Result{}, errors.New("unexpected refresh runner call")
	}
	return r.refreshFn(ctx, ref, plan, policy)
}

func refresherTestWorkspace(t *testing.T) Workspace {
	t.Helper()
	value := validationTestCollection(t)
	return Workspace{
		Collection: value,
		Data:       CollectionData{DiscoveryPolicyRevision: "policy.v1"},
		Mode:       ModeEmpty,
	}
}

var (
	_ source.Runtime = engineTestRuntime{}
	_ refresh.Runner = refresherTestRunner{}
)
