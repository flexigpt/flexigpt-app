package artifactstore

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
)

// API provides the transport-independent Artifact Store API.
//
// The caller owns the lifecycle of the supplied Components.
type API struct {
	components *system.Components

	mutationMu        sync.RWMutex
	mutationObservers map[uint64]func(basespec.RootID)
	nextObserverID    uint64
	ownedObserverID   uint64
}

func New(components *system.Components) (*API, error) {
	if components == nil {
		return nil, errors.New("artifact store components are required")
	}

	return &API{
		components:        components,
		mutationObservers: map[uint64]func(basespec.RootID){},
	}, nil
}

func (a *API) CreateArtifactRoot(
	ctx context.Context,
	request *CreateArtifactRootRequest,
) (*CreateArtifactRootResponse, error) {
	value, err := a.components.Roots.Create(ctx, *request.Body)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(value.ID)

	return &CreateArtifactRootResponse{
		Body: &value,
	}, nil
}

func (a *API) GetArtifactRoot(
	ctx context.Context,
	request *GetArtifactRootRequest,
) (*GetArtifactRootResponse, error) {
	value, err := a.components.Roots.Get(ctx, request.RootID)
	if err != nil {
		return nil, err
	}

	return &GetArtifactRootResponse{
		Body: &value,
	}, nil
}

func (a *API) ListArtifactRoots(
	ctx context.Context,
	_ *ListArtifactRootsRequest,
) (*ListArtifactRootsResponse, error) {
	values, err := a.components.Roots.List(ctx)
	if err != nil {
		return nil, err
	}

	return &ListArtifactRootsResponse{
		Body: &ListArtifactRootsResponseBody{
			Roots: values,
		},
	}, nil
}

func (a *API) UpdateArtifactRoot(
	ctx context.Context,
	request *UpdateArtifactRootRequest,
) (*UpdateArtifactRootResponse, error) {
	value, err := a.components.Roots.Update(
		ctx,
		request.RootID,
		*request.Body,
	)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(request.RootID)

	return &UpdateArtifactRootResponse{
		Body: &value,
	}, nil
}

func (a *API) RetireArtifactRoot(
	ctx context.Context,
	request *RetireArtifactRootRequest,
) (*RetireArtifactRootResponse, error) {
	value, err := a.components.Roots.Retire(
		ctx,
		request.RootID,
		request.ExpectedRevision,
	)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(request.RootID)

	return &RetireArtifactRootResponse{
		Body: &value,
	}, nil
}

func (a *API) PurgeArtifactRoot(
	ctx context.Context,
	request *PurgeArtifactRootRequest,
) (*PurgeArtifactRootResponse, error) {
	err := a.components.Roots.Purge(
		ctx,
		request.RootID,
		request.ExpectedRevision,
	)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(request.RootID)

	return &PurgeArtifactRootResponse{
		RootID: request.RootID,
	}, nil
}

func (a *API) CreateArtifactSource(
	ctx context.Context,
	request *CreateArtifactSourceRequest,
) (*CreateArtifactSourceResponse, error) {
	value, err := a.components.Sources.Create(
		ctx,
		request.RootID,
		source.Draft{
			Kind:        request.Body.Kind,
			DisplayName: request.Body.DisplayName,
			Enabled:     request.Body.Enabled,
			Config: append(
				json.RawMessage(nil),
				request.Body.Config...,
			),
		},
	)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(request.RootID)

	return &CreateArtifactSourceResponse{
		Body: &value,
	}, nil
}

func (a *API) GetArtifactSource(
	ctx context.Context,
	request *GetArtifactSourceRequest,
) (*GetArtifactSourceResponse, error) {
	value, err := a.components.Sources.Get(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return nil, err
	}

	return &GetArtifactSourceResponse{
		Body: &value,
	}, nil
}

func (a *API) ListArtifactSources(
	ctx context.Context,
	request *ListArtifactSourcesRequest,
) (*ListArtifactSourcesResponse, error) {
	values, err := a.components.Sources.List(
		ctx,
		request.RootID,
	)
	if err != nil {
		return nil, err
	}

	return &ListArtifactSourcesResponse{
		Body: &ListArtifactSourcesResponseBody{
			Sources: values,
		},
	}, nil
}

func (a *API) UpdateArtifactSource(
	ctx context.Context,
	request *UpdateArtifactSourceRequest,
) (*UpdateArtifactSourceResponse, error) {
	value, err := a.components.Sources.Update(
		ctx,
		request.RootID,
		request.SourceID,
		source.Update{
			ExpectedRevision: request.Body.ExpectedRevision,
			DisplayName:      request.Body.DisplayName,
			Enabled:          request.Body.Enabled,
			Config: append(
				json.RawMessage(nil),
				request.Body.Config...,
			),
		},
	)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(request.RootID)

	return &UpdateArtifactSourceResponse{
		Body: &value,
	}, nil
}

func (a *API) RetireArtifactSource(
	ctx context.Context,
	request *RetireArtifactSourceRequest,
) (*RetireArtifactSourceResponse, error) {
	value, err := a.components.Sources.Retire(
		ctx,
		request.RootID,
		request.SourceID,
		request.ExpectedRevision,
	)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(request.RootID)

	return &RetireArtifactSourceResponse{
		Body: &value,
	}, nil
}

func (a *API) PurgeArtifactSource(
	ctx context.Context,
	request *PurgeArtifactSourceRequest,
) (*PurgeArtifactSourceResponse, error) {
	err := a.components.Sources.Purge(
		ctx,
		request.RootID,
		request.SourceID,
		request.ExpectedRevision,
	)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(request.RootID)

	return &PurgeArtifactSourceResponse{
		RootID:   request.RootID,
		SourceID: request.SourceID,
	}, nil
}

func (a *API) ListArtifactSourceKinds(
	_ context.Context,
	_ *ListArtifactSourceKindsRequest,
) (*ListArtifactSourceKindsResponse, error) {
	return &ListArtifactSourceKindsResponse{
		Body: &ListArtifactSourceKindsResponseBody{
			Kinds: a.components.Sources.Kinds(),
		},
	}, nil
}

func (a *API) PurgeArtifact(
	ctx context.Context,
	request *PurgeArtifactRequest,
) (*PurgeArtifactResponse, error) {
	err := a.components.Artifacts.Purge(
		ctx,
		request.Artifact,
		request.ExpectedRevision,
	)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(request.Artifact.RootID)

	return &PurgeArtifactResponse{
		Artifact: request.Artifact,
	}, nil
}

func (a *API) GetManagedSourceState(
	ctx context.Context,
	request *GetManagedSourceStateRequest,
) (*GetManagedSourceStateResponse, error) {
	result, err := a.components.GetManagedSourceState(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return nil, err
	}

	return &GetManagedSourceStateResponse{
		Body: &GetManagedSourceStateResponseBody{
			Generation: result.Generation,
			Source:     result.Source,
		},
	}, nil
}

func (a *API) PublishManagedSourcePackage(
	ctx context.Context,
	request *PublishManagedSourcePackageRequest,
) (*PublishManagedSourcePackageResponse, error) {
	result, err := a.components.PublishManagedPackage(
		ctx,
		request.RootID,
		request.SourceID,
		request.Body.ExpectedSourceRevision,
		source.ManagedPackagePublication{
			Directory:          request.Body.Directory,
			ExpectedGeneration: request.Body.ExpectedGeneration,
			Files:              request.Body.Files,
		},
	)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(request.RootID)

	return &PublishManagedSourcePackageResponse{
		Body: &PublishManagedSourcePackageResponseBody{
			Generation: result.Generation,
			Source:     result.Source,
		},
	}, nil
}

func (a *API) RemoveManagedSourcePackage(
	ctx context.Context,
	request *RemoveManagedSourcePackageRequest,
) (*RemoveManagedSourcePackageResponse, error) {
	result, err := a.components.RemoveManagedPackage(
		ctx,
		request.RootID,
		request.SourceID,
		request.ExpectedSourceRevision,
		request.Directory,
		request.ExpectedGeneration,
	)
	if err != nil {
		return nil, err
	}

	a.notifyRootMutation(request.RootID)

	return &RemoveManagedSourcePackageResponse{
		Body: &RemoveManagedSourcePackageResponseBody{
			Generation: result.Generation,
			Source:     result.Source,
		},
	}, nil
}

// SubscribeRootMutation registers an independent application-level listener.
// The returned function is idempotent and may be called during shutdown.
func (a *API) SubscribeRootMutation(
	observer func(basespec.RootID),
) func() {
	if a == nil || observer == nil {
		return func() {}
	}

	a.mutationMu.Lock()
	a.nextObserverID++
	id := a.nextObserverID
	a.mutationObservers[id] = observer
	a.mutationMu.Unlock()

	return func() {
		a.mutationMu.Lock()
		delete(a.mutationObservers, id)
		a.mutationMu.Unlock()
	}
}

// SetRootMutationObserver remains only for callers that require one owned
// observer slot. New feature composition must use SubscribeRootMutation.
func (a *API) SetRootMutationObserver(
	observer func(basespec.RootID),
) {
	a.mutationMu.Lock()
	if a.ownedObserverID != 0 {
		delete(a.mutationObservers, a.ownedObserverID)
		a.ownedObserverID = 0
	}
	if observer != nil {
		a.nextObserverID++
		a.ownedObserverID = a.nextObserverID
		a.mutationObservers[a.ownedObserverID] = observer
	}
	a.mutationMu.Unlock()
}

// notifyRootMutation delivers a post-commit invalidation. Delivery is
// asynchronous: observer latency or failure must not alter a completed durable
// Artifact Store mutation. Observers must re-read current state and tolerate
// coalesced or reordered notifications for the same Root.
func (a *API) notifyRootMutation(rootID basespec.RootID) {
	a.mutationMu.RLock()
	observers := make(
		[]func(basespec.RootID),
		0,
		len(a.mutationObservers),
	)
	for _, observer := range a.mutationObservers {
		observers = append(observers, observer)
	}
	a.mutationMu.RUnlock()

	for _, observer := range observers {
		go a.invokeRootMutationObserver(observer, rootID)
	}
}

func (a *API) invokeRootMutationObserver(
	observer func(basespec.RootID),
	rootID basespec.RootID,
) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error(
				"artifact store root mutation observer panicked",
				"rootID", rootID,
				"panic", recovered,
			)
		}
	}()
	observer(rootID)
}
