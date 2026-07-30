package artifactstore

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
)

type rootMutationObserver struct {
	observer func(basespec.RootID)

	pending map[basespec.RootID]struct{}
	wake    chan struct{}
	done    chan struct{}
	stopped bool
}

// API provides the transport-independent Artifact Store API.
//
// The caller owns the lifecycle of the supplied Components.
type API struct {
	components *system.Components

	mutationMu        sync.Mutex
	mutationObservers map[uint64]*rootMutationObserver
	nextObserverID    uint64
	ownedObserverID   uint64
	closed            bool
}

func New(components *system.Components) (*API, error) {
	if components == nil {
		return nil, errors.New("artifact store components are required")
	}

	return &API{
		components:        components,
		mutationObservers: map[uint64]*rootMutationObserver{},
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

// Close stops future mutation-observer delivery. It does not close Components,
// whose lifecycle remains owned by the caller.
func (a *API) Close() {
	if a == nil {
		return
	}

	a.mutationMu.Lock()
	defer a.mutationMu.Unlock()
	if a.closed {
		return
	}
	a.closed = true
	for id := range a.mutationObservers {
		a.stopRootMutationObserverLocked(id)
	}
	a.ownedObserverID = 0
}

// SubscribeRootMutation registers an independent application-level listener.
// Delivery is asynchronous and coalesced per observer and Root. The returned
// function is idempotent and may be called during shutdown.
func (a *API) SubscribeRootMutation(
	observer func(basespec.RootID),
) func() {
	if a == nil || observer == nil {
		return func() {}
	}

	registration := &rootMutationObserver{
		observer: observer,
		pending:  map[basespec.RootID]struct{}{},
		wake:     make(chan struct{}, 1),
		done:     make(chan struct{}),
	}

	a.mutationMu.Lock()
	if a.closed {
		a.mutationMu.Unlock()
		return func() {}
	}
	a.nextObserverID++
	id := a.nextObserverID
	a.mutationObservers[id] = registration
	a.mutationMu.Unlock()

	go a.runRootMutationObserver(registration)

	return func() {
		a.unsubscribeRootMutation(id)
	}
}

// SetRootMutationObserver remains only for callers that require one owned
// observer slot. New feature composition must use SubscribeRootMutation.
func (a *API) SetRootMutationObserver(
	observer func(basespec.RootID),
) {
	if a == nil {
		return
	}

	var registration *rootMutationObserver
	a.mutationMu.Lock()
	if a.closed {
		a.mutationMu.Unlock()
		return
	}
	if a.ownedObserverID != 0 {
		a.stopRootMutationObserverLocked(a.ownedObserverID)
		a.ownedObserverID = 0
	}
	if observer != nil {
		a.nextObserverID++
		a.ownedObserverID = a.nextObserverID
		registration = &rootMutationObserver{
			observer: observer,
			pending:  map[basespec.RootID]struct{}{},
			wake:     make(chan struct{}, 1),
			done:     make(chan struct{}),
		}
		a.mutationObservers[a.ownedObserverID] = registration
	}
	a.mutationMu.Unlock()

	if registration != nil {
		go a.runRootMutationObserver(registration)
	}
}

// notifyRootMutation delivers a post-commit invalidation. Delivery is
// asynchronous and bounded: observer latency or failure must not alter a
// completed durable Artifact Store mutation. Observers must re-read current
// state and tolerate coalesced notifications for the same Root.
func (a *API) notifyRootMutation(rootID basespec.RootID) {
	if a == nil {
		return
	}

	a.mutationMu.Lock()
	defer a.mutationMu.Unlock()
	if a.closed {
		return
	}
	for _, registration := range a.mutationObservers {
		if registration.stopped {
			continue
		}
		if registration.pending == nil {
			registration.pending = map[basespec.RootID]struct{}{}
		}
		registration.pending[rootID] = struct{}{}
		select {
		case registration.wake <- struct{}{}:
		default:
		}
	}
}

func (a *API) unsubscribeRootMutation(id uint64) {
	if a == nil {
		return
	}
	a.mutationMu.Lock()
	a.stopRootMutationObserverLocked(id)
	a.mutationMu.Unlock()
}

func (a *API) stopRootMutationObserverLocked(id uint64) {
	registration, exists := a.mutationObservers[id]
	if !exists {
		return
	}
	delete(a.mutationObservers, id)
	registration.stopped = true
	registration.pending = nil
	close(registration.done)
}

func (a *API) runRootMutationObserver(
	registration *rootMutationObserver,
) {
	for {
		select {
		case <-registration.done:
			return
		case <-registration.wake:
		}

		for {
			rootIDs, active := a.takePendingRootMutations(registration)
			if !active {
				return
			}
			if len(rootIDs) == 0 {
				break
			}
			for _, rootID := range rootIDs {
				if !a.rootMutationObserverActive(registration) {
					return
				}
				a.invokeRootMutationObserver(
					registration.observer,
					rootID,
				)
			}
		}
	}
}

func (a *API) takePendingRootMutations(
	registration *rootMutationObserver,
) ([]basespec.RootID, bool) {
	a.mutationMu.Lock()
	defer a.mutationMu.Unlock()
	if a.closed || registration.stopped {
		return nil, false
	}

	rootIDs := make([]basespec.RootID, 0, len(registration.pending))
	for rootID := range registration.pending {
		rootIDs = append(rootIDs, rootID)
	}
	registration.pending = map[basespec.RootID]struct{}{}
	slices.Sort(rootIDs)
	return rootIDs, true
}

func (a *API) rootMutationObserverActive(
	registration *rootMutationObserver,
) bool {
	a.mutationMu.Lock()
	active := !a.closed && !registration.stopped
	a.mutationMu.Unlock()
	return active
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
