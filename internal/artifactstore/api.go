package artifactstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
)

var (
	ErrAPIUnavailable = errors.New("artifact store API is not initialized")
	ErrInvalidRequest = errors.New("invalid artifact store API request")
)

// API is the transport-independent Artifact Store boundary.
//
// It owns the provided system components. Callers must not close the same
// components separately after passing them to New.
type API struct {
	lifecycleMu sync.RWMutex
	components  *system.Components
}

func New(components *system.Components) (*API, error) {
	if components == nil {
		return nil, fmt.Errorf("%w: components are nil", ErrAPIUnavailable)
	}
	return &API{
		components: components,
	}, nil
}

// Components exposes the shared Artifact Store component graph only for
// application composition, such as constructing the Workspace API.
//
// Normal Artifact Store callers should use API methods instead.
func (a *API) Components() *system.Components {
	if a == nil {
		return nil
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()

	return a.components
}

func (a *API) Ready() error {
	if a == nil {
		return ErrAPIUnavailable
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()

	if a.components == nil {
		return ErrAPIUnavailable
	}
	return nil
}

func (a *API) Close() error {
	if a == nil {
		return nil
	}

	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()

	components := a.components
	if components == nil {
		return nil
	}

	a.components = nil
	return components.Close()
}

func withComponents[T any](
	api *API,
	ctx context.Context,
	operation func(*system.Components) (T, error),
) (T, error) {
	var zero T

	if ctx == nil {
		return zero, invalidAPIRequest("context is required")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if api == nil {
		return zero, ErrAPIUnavailable
	}

	api.lifecycleMu.RLock()
	defer api.lifecycleMu.RUnlock()

	if api.components == nil {
		return zero, ErrAPIUnavailable
	}

	return operation(api.components)
}

func invalidAPIRequest(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidRequest, message)
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}

func (a *API) CreateArtifactRoot(
	ctx context.Context,
	request *CreateArtifactRootRequest,
) (*CreateArtifactRootResponse, error) {
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("artifact root body is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*CreateArtifactRootResponse, error) {
			value, err := components.Roots.Create(ctx, *request.Body)
			if err != nil {
				return nil, err
			}
			return &CreateArtifactRootResponse{Body: &value}, nil
		},
	)
}

func (a *API) GetArtifactRoot(
	ctx context.Context,
	request *GetArtifactRootRequest,
) (*GetArtifactRootResponse, error) {
	if request == nil {
		return nil, invalidAPIRequest("artifact root request is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*GetArtifactRootResponse, error) {
			value, err := components.Roots.Get(ctx, request.RootID)
			if err != nil {
				return nil, err
			}
			return &GetArtifactRootResponse{Body: &value}, nil
		},
	)
}

func (a *API) ListArtifactRoots(
	ctx context.Context,
	_ *ListArtifactRootsRequest,
) (*ListArtifactRootsResponse, error) {
	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*ListArtifactRootsResponse, error) {
			values, err := components.Roots.List(ctx)
			if err != nil {
				return nil, err
			}
			return &ListArtifactRootsResponse{
				Body: &ListArtifactRootsResponseBody{
					Roots: values,
				},
			}, nil
		},
	)
}

func (a *API) UpdateArtifactRoot(
	ctx context.Context,
	request *UpdateArtifactRootRequest,
) (*UpdateArtifactRootResponse, error) {
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("artifact root update body is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*UpdateArtifactRootResponse, error) {
			value, err := components.Roots.Update(
				ctx,
				request.RootID,
				*request.Body,
			)
			if err != nil {
				return nil, err
			}
			return &UpdateArtifactRootResponse{Body: &value}, nil
		},
	)
}

func (a *API) RetireArtifactRoot(
	ctx context.Context,
	request *RetireArtifactRootRequest,
) (*RetireArtifactRootResponse, error) {
	if request == nil {
		return nil, invalidAPIRequest("artifact root retirement request is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*RetireArtifactRootResponse, error) {
			value, err := components.Roots.Retire(
				ctx,
				request.RootID,
				request.ExpectedRevision,
			)
			if err != nil {
				return nil, err
			}
			return &RetireArtifactRootResponse{Body: &value}, nil
		},
	)
}

func (a *API) PurgeArtifactRoot(
	ctx context.Context,
	request *PurgeArtifactRootRequest,
) (*PurgeArtifactRootResponse, error) {
	if request == nil {
		return nil, invalidAPIRequest("artifact root purge request is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*PurgeArtifactRootResponse, error) {
			if err := components.Roots.Purge(
				ctx,
				request.RootID,
				request.ExpectedRevision,
			); err != nil {
				return nil, err
			}
			return &PurgeArtifactRootResponse{
				RootID: request.RootID,
			}, nil
		},
	)
}

func (a *API) CreateArtifactSource(
	ctx context.Context,
	request *CreateArtifactSourceRequest,
) (*CreateArtifactSourceResponse, error) {
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("artifact source body is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*CreateArtifactSourceResponse, error) {
			value, err := components.Sources.Create(
				ctx,
				request.RootID,
				source.Draft{
					Kind:        request.Body.Kind,
					DisplayName: request.Body.DisplayName,
					Enabled:     request.Body.Enabled,
					Config:      cloneRawMessage(request.Body.Config),
				},
			)
			if err != nil {
				return nil, err
			}
			return &CreateArtifactSourceResponse{Body: &value}, nil
		},
	)
}

func (a *API) GetArtifactSource(
	ctx context.Context,
	request *GetArtifactSourceRequest,
) (*GetArtifactSourceResponse, error) {
	if request == nil {
		return nil, invalidAPIRequest("artifact source request is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*GetArtifactSourceResponse, error) {
			value, err := components.Sources.Get(
				ctx,
				request.RootID,
				request.SourceID,
			)
			if err != nil {
				return nil, err
			}
			return &GetArtifactSourceResponse{Body: &value}, nil
		},
	)
}

func (a *API) ListArtifactSources(
	ctx context.Context,
	request *ListArtifactSourcesRequest,
) (*ListArtifactSourcesResponse, error) {
	if request == nil {
		return nil, invalidAPIRequest("artifact source list request is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*ListArtifactSourcesResponse, error) {
			values, err := components.Sources.List(ctx, request.RootID)
			if err != nil {
				return nil, err
			}
			return &ListArtifactSourcesResponse{
				Body: &ListArtifactSourcesResponseBody{
					Sources: values,
				},
			}, nil
		},
	)
}

func (a *API) UpdateArtifactSource(
	ctx context.Context,
	request *UpdateArtifactSourceRequest,
) (*UpdateArtifactSourceResponse, error) {
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("artifact source update body is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*UpdateArtifactSourceResponse, error) {
			value, err := components.Sources.Update(
				ctx,
				request.RootID,
				request.SourceID,
				source.Update{
					ExpectedRevision: request.Body.ExpectedRevision,
					DisplayName:      request.Body.DisplayName,
					Enabled:          request.Body.Enabled,
					Config:           cloneRawMessage(request.Body.Config),
				},
			)
			if err != nil {
				return nil, err
			}
			return &UpdateArtifactSourceResponse{Body: &value}, nil
		},
	)
}

func (a *API) RetireArtifactSource(
	ctx context.Context,
	request *RetireArtifactSourceRequest,
) (*RetireArtifactSourceResponse, error) {
	if request == nil {
		return nil, invalidAPIRequest("artifact source retirement request is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*RetireArtifactSourceResponse, error) {
			value, err := components.Sources.Retire(
				ctx,
				request.RootID,
				request.SourceID,
				request.ExpectedRevision,
			)
			if err != nil {
				return nil, err
			}
			return &RetireArtifactSourceResponse{Body: &value}, nil
		},
	)
}

func (a *API) PurgeArtifactSource(
	ctx context.Context,
	request *PurgeArtifactSourceRequest,
) (*PurgeArtifactSourceResponse, error) {
	if request == nil {
		return nil, invalidAPIRequest("artifact source purge request is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*PurgeArtifactSourceResponse, error) {
			if err := components.Sources.Purge(
				ctx,
				request.RootID,
				request.SourceID,
				request.ExpectedRevision,
			); err != nil {
				return nil, err
			}
			return &PurgeArtifactSourceResponse{
				RootID:   request.RootID,
				SourceID: request.SourceID,
			}, nil
		},
	)
}

func (a *API) ListArtifactSourceKinds(
	ctx context.Context,
	_ *ListArtifactSourceKindsRequest,
) (*ListArtifactSourceKindsResponse, error) {
	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*ListArtifactSourceKindsResponse, error) {
			return &ListArtifactSourceKindsResponse{
				Body: &ListArtifactSourceKindsResponseBody{
					Kinds: components.Sources.Kinds(),
				},
			}, nil
		},
	)
}

func (a *API) GetManagedSourceState(
	ctx context.Context,
	request *GetManagedSourceStateRequest,
) (*GetManagedSourceStateResponse, error) {
	if request == nil {
		return nil, invalidAPIRequest("managed source state request is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*GetManagedSourceStateResponse, error) {
			result, err := components.GetManagedSourceState(
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
		},
	)
}

func (a *API) PublishManagedSourcePackage(
	ctx context.Context,
	request *PublishManagedSourcePackageRequest,
) (*PublishManagedSourcePackageResponse, error) {
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest("managed source package body is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*PublishManagedSourcePackageResponse, error) {
			result, err := components.PublishManagedPackage(
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
			return &PublishManagedSourcePackageResponse{
				Body: &PublishManagedSourcePackageResponseBody{
					Generation: result.Generation,
					Source:     result.Source,
				},
			}, nil
		},
	)
}

func (a *API) RemoveManagedSourcePackage(
	ctx context.Context,
	request *RemoveManagedSourcePackageRequest,
) (*RemoveManagedSourcePackageResponse, error) {
	if request == nil {
		return nil, invalidAPIRequest("managed source package removal request is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*RemoveManagedSourcePackageResponse, error) {
			result, err := components.RemoveManagedPackage(
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
			return &RemoveManagedSourcePackageResponse{
				Body: &RemoveManagedSourcePackageResponseBody{
					Generation: result.Generation,
					Source:     result.Source,
				},
			}, nil
		},
	)
}

func (a *API) PurgeArtifact(
	ctx context.Context,
	request *PurgeArtifactRequest,
) (*PurgeArtifactResponse, error) {
	if request == nil {
		return nil, invalidAPIRequest("artifact purge request is required")
	}

	return withComponents(
		a,
		ctx,
		func(components *system.Components) (*PurgeArtifactResponse, error) {
			if err := components.Artifacts.Purge(
				ctx,
				request.Artifact,
				request.ExpectedRevision,
			); err != nil {
				return nil, err
			}
			return &PurgeArtifactResponse{
				Artifact: request.Artifact,
			}, nil
		},
	)
}
