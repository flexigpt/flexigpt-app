package artifactstore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
)

// API provides the transport-independent Artifact Store API.
//
// The caller owns the lifecycle of the supplied Components.
type API struct {
	components *system.Components
}

func New(components *system.Components) (*API, error) {
	if components == nil {
		return nil, errors.New("artifact store components are required")
	}

	return &API{
		components: components,
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

	return &RemoveManagedSourcePackageResponse{
		Body: &RemoveManagedSourcePackageResponseBody{
			Generation: result.Generation,
			Source:     result.Source,
		},
	}, nil
}

// Close exists for transport lifecycle symmetry. Components remain owned and
// closed by the application composition root.
func (*API) Close() {}
