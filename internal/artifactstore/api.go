package artifactstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
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
	if components == nil ||
		components.Roots == nil ||
		components.Sources == nil {
		return nil, errors.New("artifact store components are required")
	}
	if components.Shareables == nil {
		return nil, errors.New("artifact store shareable documents are required")
	}

	return &API{
		components: components,
	}, nil
}

func requireRequest[T any](value *T, subject string) error {
	if value != nil {
		return nil
	}
	return fmt.Errorf("%w: %s is required", basespec.ErrInvalid, subject)
}

func requireBody[T any](value *T, subject string) error {
	if value != nil {
		return nil
	}
	return fmt.Errorf("%w: %s is required", basespec.ErrInvalid, subject)
}

func (a *API) CreateArtifactRoot(
	ctx context.Context,
	request *CreateArtifactRootRequest,
) (*CreateArtifactRootResponse, error) {
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "create artifact root request"); err != nil {
		return nil, err
	}
	if err := requireBody(request.Body, "artifact root body"); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "get artifact root request"); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "update artifact root request"); err != nil {
		return nil, err
	}
	if err := requireBody(request.Body, "artifact root update body"); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "retire artifact root request"); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "purge artifact root request"); err != nil {
		return nil, err
	}
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

func (a *API) GetShareableCollectionDocument(
	ctx context.Context,
	request *GetShareableCollectionDocumentRequest,
) (*GetShareableCollectionDocumentResponse, error) {
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "get shareable collection document request"); err != nil {
		return nil, err
	}
	value, err := a.components.Shareables.GetCollection(
		ctx,
		collection.CollectionRef{
			RootID:       request.RootID,
			CollectionID: request.CollectionID,
		},
	)
	if err != nil {
		return nil, err
	}
	return &GetShareableCollectionDocumentResponse{Body: &value}, nil
}

func (a *API) CreateArtifactSource(
	ctx context.Context,
	request *CreateArtifactSourceRequest,
) (*CreateArtifactSourceResponse, error) {
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "create artifact source request"); err != nil {
		return nil, err
	}
	if err := requireBody(request.Body, "artifact source body"); err != nil {
		return nil, err
	}
	value, err := a.components.Sources.Create(
		ctx,
		request.RootID,
		source.Draft{
			ID:          request.Body.ID,
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "get artifact source request"); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "list artifact sources request"); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "update artifact source request"); err != nil {
		return nil, err
	}
	if err := requireBody(request.Body, "artifact source update body"); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "retire artifact source request"); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "purge artifact source request"); err != nil {
		return nil, err
	}
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
	ctx context.Context,
	_ *ListArtifactSourceKindsRequest,
) (*ListArtifactSourceKindsResponse, error) {
	if err := a.check(ctx); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "get managed Source state request"); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "publish managed Source package request"); err != nil {
		return nil, err
	}
	if err := requireBody(request.Body, "managed Source package body"); err != nil {
		return nil, err
	}
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
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	if err := requireRequest(request, "remove managed Source package request"); err != nil {
		return nil, err
	}
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

func (a *API) check(ctx context.Context) error {
	if a == nil ||
		a.components == nil ||
		a.components.Roots == nil ||
		a.components.Sources == nil ||
		a.components.Shareables == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: artifact store API context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}
