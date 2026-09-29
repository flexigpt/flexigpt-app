package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
)

func (a *API) GetProviderRuntimeOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ProviderRuntimeOverlayView, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	record, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	if record.State != artifact.StateAvailable {
		return ProviderRuntimeOverlayView{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	value, found, err := a.overlays.GetProviderOverlay(ctx, ref)
	if err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	if !found {
		return ProviderRuntimeOverlayView{}, nil
	}
	return providerOverlayView(value), nil
}

func (a *API) UpdateProviderRuntimeOverlay(
	ctx context.Context,
	request ProviderRuntimeOverlayUpdateRequest,
) (ProviderRuntimeOverlayView, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	if err := request.Provider.Validate(); err != nil {
		return ProviderRuntimeOverlayView{}, err
	}

	record, err := a.requireKind(
		ctx,
		request.Provider,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	if record.State != artifact.StateAvailable {
		return ProviderRuntimeOverlayView{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if !a.protection.IsProtectedRoot(record.RootID) &&
		request.DefaultModel != nil {
		return ProviderRuntimeOverlayView{}, fmt.Errorf(
			"%w: mutable Model Provider default belongs in Artifact.Data",
			basespec.ErrInvalid,
		)
	}

	current, found, err := a.overlays.GetProviderOverlay(
		ctx,
		request.Provider,
	)
	if err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	if found && current.Revision != request.ExpectedRevision {
		return ProviderRuntimeOverlayView{}, basespec.ErrConflict
	}
	if !found && request.ExpectedRevision != 0 {
		return ProviderRuntimeOverlayView{}, basespec.ErrConflict
	}

	nextRevision := uint64(1)
	if found {
		nextRevision = current.Revision + 1
	}
	value := modelOverlay.ProviderOverlay{
		SchemaVersion:     modelOverlay.OverlaySchemaVersion,
		Revision:          nextRevision,
		CredentialRef:     request.CredentialRef,
		Connection:        cloneRaw(request.Connection),
		Defaults:          cloneRaw(request.Defaults),
		Capabilities:      cloneRaw(request.Capabilities),
		DefaultModel:      cloneOptionalReference(request.DefaultModel),
		AdapterParameters: cloneRaw(request.AdapterParameters),
	}
	if err := a.overlays.PutProviderOverlay(
		ctx,
		request.Provider,
		request.ExpectedRevision,
		value,
	); err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	return providerOverlayView(value), nil
}

func (a *API) DeleteProviderRuntimeOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := a.ready(ctx); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Model Provider overlay revision is required",
			basespec.ErrInvalid,
		)
	}
	if _, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	); err != nil {
		return err
	}
	return a.overlays.DeleteProviderOverlay(
		ctx,
		ref,
		expectedRevision,
	)
}

func (a *API) GetModelRuntimeOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ModelRuntimeOverlayView, error) {
	if err := a.ready(ctx); err != nil {
		return ModelRuntimeOverlayView{}, err
	}
	record, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelArtifactKind,
	)
	if err != nil {
		return ModelRuntimeOverlayView{}, err
	}
	if record.State != artifact.StateAvailable {
		return ModelRuntimeOverlayView{}, fmt.Errorf(
			"%w: Model Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	value, found, err := a.overlays.GetModelOverlay(ctx, ref)
	if err != nil {
		return ModelRuntimeOverlayView{}, err
	}
	if !found {
		return ModelRuntimeOverlayView{}, nil
	}
	return modelOverlayView(value), nil
}

func (a *API) UpdateModelRuntimeOverlay(
	ctx context.Context,
	request ModelRuntimeOverlayUpdateRequest,
) (ModelRuntimeOverlayView, error) {
	if err := a.ready(ctx); err != nil {
		return ModelRuntimeOverlayView{}, err
	}
	if err := request.Model.Validate(); err != nil {
		return ModelRuntimeOverlayView{}, err
	}

	record, err := a.requireKind(
		ctx,
		request.Model,
		modelDomain.ModelArtifactKind,
	)
	if err != nil {
		return ModelRuntimeOverlayView{}, err
	}
	if record.State != artifact.StateAvailable {
		return ModelRuntimeOverlayView{}, fmt.Errorf(
			"%w: Model Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	current, found, err := a.overlays.GetModelOverlay(
		ctx,
		request.Model,
	)
	if err != nil {
		return ModelRuntimeOverlayView{}, err
	}
	if found && current.Revision != request.ExpectedRevision {
		return ModelRuntimeOverlayView{}, basespec.ErrConflict
	}
	if !found && request.ExpectedRevision != 0 {
		return ModelRuntimeOverlayView{}, basespec.ErrConflict
	}

	nextRevision := uint64(1)
	if found {
		nextRevision = current.Revision + 1
	}
	value := modelOverlay.ModelOverlay{
		SchemaVersion:     modelOverlay.OverlaySchemaVersion,
		Revision:          nextRevision,
		Defaults:          cloneRaw(request.Defaults),
		Capabilities:      cloneRaw(request.Capabilities),
		AdapterParameters: cloneRaw(request.AdapterParameters),
	}
	if err := a.overlays.PutModelOverlay(
		ctx,
		request.Model,
		request.ExpectedRevision,
		value,
	); err != nil {
		return ModelRuntimeOverlayView{}, err
	}
	return modelOverlayView(value), nil
}

func (a *API) DeleteModelRuntimeOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := a.ready(ctx); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Model overlay revision is required",
			basespec.ErrInvalid,
		)
	}
	if _, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelArtifactKind,
	); err != nil {
		return err
	}
	return a.overlays.DeleteModelOverlay(
		ctx,
		ref,
		expectedRevision,
	)
}

func (a *API) purgeProviderOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	value, found, err := a.overlays.GetProviderOverlay(ctx, ref)
	if err != nil || !found {
		return err
	}
	return a.overlays.DeleteProviderOverlay(
		ctx,
		ref,
		value.Revision,
	)
}

func (a *API) purgeModelOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	value, found, err := a.overlays.GetModelOverlay(ctx, ref)
	if err != nil || !found {
		return err
	}
	return a.overlays.DeleteModelOverlay(
		ctx,
		ref,
		value.Revision,
	)
}

func providerOverlayView(
	value modelOverlay.ProviderOverlay,
) ProviderRuntimeOverlayView {
	return ProviderRuntimeOverlayView{
		Revision:             value.Revision,
		CredentialConfigured: value.CredentialRef != "",
		Connection:           cloneRaw(value.Connection),
		Defaults:             cloneRaw(value.Defaults),
		Capabilities:         cloneRaw(value.Capabilities),
		DefaultModel:         cloneOptionalReference(value.DefaultModel),
		AdapterParameters:    cloneRaw(value.AdapterParameters),
	}
}

func modelOverlayView(
	value modelOverlay.ModelOverlay,
) ModelRuntimeOverlayView {
	return ModelRuntimeOverlayView{
		Revision:          value.Revision,
		Defaults:          cloneRaw(value.Defaults),
		Capabilities:      cloneRaw(value.Capabilities),
		AdapterParameters: cloneRaw(value.AdapterParameters),
	}
}
