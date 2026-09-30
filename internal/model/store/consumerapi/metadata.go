package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
)

func (a *API) SetMutableProviderDefaultModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	defaultModel declaration.ArtifactNameReference,
) (ProviderView, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderView{}, err
	}
	if err := validateExpectedArtifactRevision(
		expectedArtifactRevision,
	); err != nil {
		return ProviderView{}, err
	}
	if err := defaultModel.Validate(); err != nil {
		return ProviderView{}, err
	}

	provider, err := a.loadProvider(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	if provider.Artifact.Revision != expectedArtifactRevision {
		return ProviderView{}, basespec.ErrConflict
	}
	if a.protection.IsProtectedRoot(provider.Artifact.RootID) {
		return ProviderView{}, fmt.Errorf(
			"%w: protected Model Provider default belongs in the protected runtime overlay API",
			basespec.ErrProtected,
		)
	}

	current, found, err := a.overlays.GetProviderOverlay(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	expectedOverlayRevision := uint64(0)
	if found {
		expectedOverlayRevision = current.Revision
	}
	if expectedOverlayRevision == ^uint64(0) {
		return ProviderView{}, fmt.Errorf(
			"%w: Model Provider overlay revision is exhausted",
			basespec.ErrInvalid,
		)
	}

	next := modelOverlay.ProviderOverlay{
		SchemaVersion:     modelOverlay.OverlaySchemaVersion,
		Revision:          expectedOverlayRevision + 1,
		Connection:        cloneRaw(current.Connection),
		Defaults:          cloneRaw(current.Defaults),
		Capabilities:      cloneRaw(current.Capabilities),
		DefaultModel:      &defaultModel,
		AdapterParameters: cloneRaw(current.AdapterParameters),
	}
	if err := a.overlays.PutProviderOverlay(
		ctx,
		ref,
		expectedArtifactRevision,
		expectedOverlayRevision,
		next,
	); err != nil {
		return ProviderView{}, err
	}

	updated, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	updatedProvider, err := modelDomain.DecodeProvider(
		updated,
		provider.Definition,
	)
	if err != nil {
		return ProviderView{}, err
	}
	return a.providerView(updatedProvider)
}

func (a *API) ClearMutableProviderDefaultModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
) (ProviderView, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderView{}, err
	}
	if err := validateExpectedArtifactRevision(
		expectedArtifactRevision,
	); err != nil {
		return ProviderView{}, err
	}

	provider, err := a.loadProvider(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	if provider.Artifact.Revision != expectedArtifactRevision {
		return ProviderView{}, basespec.ErrConflict
	}
	if a.protection.IsProtectedRoot(provider.Artifact.RootID) {
		return ProviderView{}, fmt.Errorf(
			"%w: protected Model Provider default belongs in the protected runtime overlay API",
			basespec.ErrProtected,
		)
	}

	current, found, err := a.overlays.GetProviderOverlay(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	if !found || current.DefaultModel == nil {
		return a.providerView(provider)
	}
	if current.Revision == ^uint64(0) {
		return ProviderView{}, fmt.Errorf(
			"%w: Model Provider overlay revision is exhausted",
			basespec.ErrInvalid,
		)
	}

	next := current.Clone()
	next.Revision++
	next.DefaultModel = nil
	if err := a.overlays.PutProviderOverlay(
		ctx,
		ref,
		expectedArtifactRevision,
		current.Revision,
		next,
	); err != nil {
		return ProviderView{}, err
	}

	updated, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return ProviderView{}, err
	}
	updatedProvider, err := modelDomain.DecodeProvider(
		updated,
		provider.Definition,
	)
	if err != nil {
		return ProviderView{}, err
	}
	return a.providerView(updatedProvider)
}
