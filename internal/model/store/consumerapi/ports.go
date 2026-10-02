package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/root"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
)

// ManagementStoreFacade is the aggregate-facing Model Store persistence port.
// It deliberately excludes settings implementation details and Artifact Store
// mutation internals.
type ManagementStoreFacade struct {
	api *API
}

func NewManagementStore(
	api *API,
) (*ManagementStoreFacade, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: Model management Store requires an API",
			basespec.ErrInvalid,
		)
	}
	return &ManagementStoreFacade{api: api}, nil
}

func (s *ManagementStoreFacade) ResolveModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedModel, error) {
	if s == nil || s.api == nil {
		return ResolvedModel{}, basespec.ErrClosed
	}
	return s.api.ResolveModel(ctx, ref)
}

func (s *ManagementStoreFacade) ResolveProvider(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedProvider, error) {
	if s == nil || s.api == nil {
		return ResolvedProvider{}, basespec.ErrClosed
	}
	return s.api.ResolveProvider(ctx, ref)
}

func (s *ManagementStoreFacade) ResolveProviderDefaultModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (DefaultModelResolution, error) {
	if s == nil || s.api == nil {
		return DefaultModelResolution{}, basespec.ErrClosed
	}
	return s.api.ResolveProviderDefaultModel(ctx, ref)
}

func (s *ManagementStoreFacade) CreateProvider(
	ctx context.Context,
	request ManagedProviderCreateRequest,
) (ManagedProviderCreateResult, error) {
	if s == nil || s.api == nil {
		return ManagedProviderCreateResult{}, basespec.ErrClosed
	}
	return s.api.CreateProvider(ctx, request)
}

func (s *ManagementStoreFacade) ReplaceProvider(
	ctx context.Context,
	request ManagedProviderReplaceRequest,
) (ManagedProviderReplaceResult, error) {
	if s == nil || s.api == nil {
		return ManagedProviderReplaceResult{}, basespec.ErrClosed
	}
	return s.api.ReplaceProvider(ctx, request)
}

func (s *ManagementStoreFacade) DeleteProvider(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.DeleteProvider(ctx, ref, expectedArtifactRevision)
}

func (s *ManagementStoreFacade) CreateModel(
	ctx context.Context,
	request ManagedModelCreateRequest,
) (ManagedModelCreateResult, error) {
	if s == nil || s.api == nil {
		return ManagedModelCreateResult{}, basespec.ErrClosed
	}
	return s.api.CreateModel(ctx, request)
}

func (s *ManagementStoreFacade) ReplaceModel(
	ctx context.Context,
	request ManagedModelReplaceRequest,
) (ManagedModelReplaceResult, error) {
	if s == nil || s.api == nil {
		return ManagedModelReplaceResult{}, basespec.ErrClosed
	}
	return s.api.ReplaceModel(ctx, request)
}

func (s *ManagementStoreFacade) DeleteModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.DeleteModel(ctx, ref, expectedArtifactRevision)
}

func (s *ManagementStoreFacade) SetProviderEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if s == nil || s.api == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	return s.api.SetProviderEnabled(ctx, ref, expectedRevision, enabled)
}

func (s *ManagementStoreFacade) SetModelEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if s == nil || s.api == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	return s.api.SetModelEnabled(ctx, ref, expectedRevision, enabled)
}

func (s *ManagementStoreFacade) SaveProviderSettings(
	ctx context.Context,
	request SaveProviderSettingsRequest,
) (ProviderView, error) {
	if s == nil || s.api == nil {
		return ProviderView{}, basespec.ErrClosed
	}
	return s.api.SaveProviderSettings(ctx, request)
}

func (s *ManagementStoreFacade) ResetProviderSettings(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedProviderRevision uint64,
	expectedSettingsRevision uint64,
) (ProviderView, error) {
	if s == nil || s.api == nil {
		return ProviderView{}, basespec.ErrClosed
	}
	return s.api.ResetProviderSettings(
		ctx,
		ref,
		expectedProviderRevision,
		expectedSettingsRevision,
	)
}

func (s *ManagementStoreFacade) SetProviderAPIKey(
	ctx context.Context,
	request SetProviderAPIKeyRequest,
) (ProviderAPIKeyStatus, error) {
	if s == nil || s.api == nil {
		return ProviderAPIKeyStatus{}, basespec.ErrClosed
	}
	return s.api.SetProviderAPIKey(ctx, request)
}

func (s *ManagementStoreFacade) ClearProviderAPIKey(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedProviderRevision uint64,
	expectedAPIKeyRevision uint64,
) (ProviderAPIKeyStatus, error) {
	if s == nil || s.api == nil {
		return ProviderAPIKeyStatus{}, basespec.ErrClosed
	}
	return s.api.ClearProviderAPIKey(
		ctx,
		ref,
		expectedProviderRevision,
		expectedAPIKeyRevision,
	)
}

func (s *ManagementStoreFacade) GetProvider(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ProviderView, error) {
	if s == nil || s.api == nil {
		return ProviderView{}, basespec.ErrClosed
	}
	return s.api.GetProvider(ctx, ref)
}

func (s *ManagementStoreFacade) GetModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ModelView, error) {
	if s == nil || s.api == nil {
		return ModelView{}, basespec.ErrClosed
	}
	return s.api.GetModel(ctx, ref)
}

// CatalogStore is the narrow cross-Root listing port used by later management
// and frontend composition layers.
type CatalogStore struct {
	api *API
}

func NewCatalogStore(api *API) (*CatalogStore, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: Model catalog Store requires an API",
			basespec.ErrInvalid,
		)
	}
	return &CatalogStore{api: api}, nil
}

func (s *CatalogStore) ListProviders(
	ctx context.Context,
	rootID root.RootID,
) ([]ProviderListItem, error) {
	if s == nil || s.api == nil {
		return nil, basespec.ErrClosed
	}
	return s.api.ListProviders(ctx, ListProvidersRequest{
		RootID: rootID,
	})
}

func (s *CatalogStore) ListModels(
	ctx context.Context,
	rootID root.RootID,
) ([]ModelListItem, error) {
	if s == nil || s.api == nil {
		return nil, basespec.ErrClosed
	}
	return s.api.ListModels(ctx, ListModelsRequest{
		RootID: rootID,
	})
}

func (s *CatalogStore) ListModelsByProvider(
	ctx context.Context,
	rootID root.RootID,
	provider declaration.ArtifactNameReference,
) ([]ModelListItem, error) {
	if s == nil || s.api == nil {
		return nil, basespec.ErrClosed
	}
	return s.api.ListModelsByProvider(ctx, ListModelsByProviderRequest{
		RootID:   rootID,
		Provider: provider,
	})
}

func ProviderDocumentForManagedAuthoring(
	document modelDomain.ProviderDocument,
) modelDomain.ProviderDocument {
	return document
}

func ModelDocumentForManagedAuthoring(
	document modelDomain.ModelDocument,
) modelDomain.ModelDocument {
	return document
}
