package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/secret"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
)

func (a *API) SaveProviderSettings(
	ctx context.Context,
	request SaveProviderSettingsRequest,
) (ProviderView, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderView{}, err
	}
	if err := request.Provider.Validate(); err != nil {
		return ProviderView{}, err
	}
	if err := validateExpectedArtifactRevision(
		request.ExpectedProviderRevision,
	); err != nil {
		return ProviderView{}, err
	}

	record, err := a.requireKind(
		ctx,
		request.Provider,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return ProviderView{}, err
	}
	if record.State != artifact.StateAvailable {
		return ProviderView{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Revision != request.ExpectedProviderRevision {
		return ProviderView{}, basespec.ErrConflict
	}

	current, found, err := a.overlays.GetProviderOverlay(
		ctx,
		request.Provider,
	)
	if err != nil {
		return ProviderView{}, err
	}
	if found {
		if current.Revision != request.ExpectedSettingsRevision {
			return ProviderView{}, basespec.ErrConflict
		}
	} else if request.ExpectedSettingsRevision != 0 {
		return ProviderView{}, basespec.ErrConflict
	}

	if request.ExpectedSettingsRevision == ^uint64(0) {
		return ProviderView{}, fmt.Errorf(
			"%w: Model Provider settings revision is exhausted",
			basespec.ErrInvalid,
		)
	}

	value := modelOverlay.ProviderOverlay{
		SchemaVersion:     modelOverlay.OverlaySchemaVersion,
		Revision:          request.ExpectedSettingsRevision + 1,
		Connection:        cloneRaw(request.Connection),
		Defaults:          cloneRaw(request.Defaults),
		Capabilities:      cloneRaw(request.Capabilities),
		DefaultModel:      cloneOptionalReference(request.DefaultModel),
		AdapterParameters: cloneRaw(request.AdapterParameters),
	}
	if err := a.overlays.PutProviderOverlay(
		ctx,
		request.Provider,
		request.ExpectedProviderRevision,
		request.ExpectedSettingsRevision,
		value,
	); err != nil {
		return ProviderView{}, err
	}

	return a.GetProvider(ctx, request.Provider)
}

func (a *API) ResetProviderSettings(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedProviderRevision uint64,
	expectedSettingsRevision uint64,
) (ProviderView, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderView{}, err
	}
	if err := ref.Validate(); err != nil {
		return ProviderView{}, err
	}
	if expectedProviderRevision == 0 || expectedSettingsRevision == 0 {
		return ProviderView{}, fmt.Errorf(
			"%w: expected Provider and Provider settings revisions are required",
			basespec.ErrInvalid,
		)
	}

	record, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return ProviderView{}, err
	}
	if record.Revision != expectedProviderRevision {
		return ProviderView{}, basespec.ErrConflict
	}

	if err := a.overlays.DeleteProviderOverlay(
		ctx,
		ref,
		expectedProviderRevision,
		expectedSettingsRevision,
	); err != nil {
		return ProviderView{}, err
	}

	return a.GetProvider(ctx, ref)
}

func (a *API) GetProviderAPIKeyStatus(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ProviderAPIKeyStatus, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderAPIKeyStatus{}, err
	}

	record, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return ProviderAPIKeyStatus{}, err
	}
	if record.State != artifact.StateAvailable {
		return ProviderAPIKeyStatus{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	binding, found, err := a.overlays.GetProviderCredential(ctx, ref)
	if err != nil {
		return ProviderAPIKeyStatus{}, err
	}

	output := ProviderAPIKeyStatus{
		ProviderRevision: record.Revision,
	}
	if found {
		output.APIKeyRevision = binding.Revision
		output.Configured = binding.Active()
	}
	return output, nil
}

func (a *API) SetProviderAPIKey(
	ctx context.Context,
	request SetProviderAPIKeyRequest,
) (ProviderAPIKeyStatus, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderAPIKeyStatus{}, err
	}
	if err := request.Provider.Validate(); err != nil {
		return ProviderAPIKeyStatus{}, err
	}
	if err := validateExpectedArtifactRevision(
		request.ExpectedProviderRevision,
	); err != nil {
		return ProviderAPIKeyStatus{}, err
	}
	if request.APIKey == "" {
		return ProviderAPIKeyStatus{}, fmt.Errorf(
			"%w: Provider API key is required",
			basespec.ErrInvalid,
		)
	}

	record, err := a.requireKind(
		ctx,
		request.Provider,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return ProviderAPIKeyStatus{}, err
	}
	if record.State != artifact.StateAvailable {
		return ProviderAPIKeyStatus{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Revision != request.ExpectedProviderRevision {
		return ProviderAPIKeyStatus{}, basespec.ErrConflict
	}

	if _, err := a.overlays.ReplaceProviderCredential(
		ctx,
		secret.ReplaceBindingRequest{
			Key: modelOverlay.ProviderCredentialBindingKey(
				request.Provider,
			),
			ExpectedArtifactRevision: request.ExpectedProviderRevision,
			ExpectedBindingRevision:  request.ExpectedAPIKeyRevision,
			Value:                    request.APIKey,
		},
	); err != nil {
		return ProviderAPIKeyStatus{}, err
	}

	return a.GetProviderAPIKeyStatus(ctx, request.Provider)
}

func (a *API) ClearProviderAPIKey(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedProviderRevision uint64,
	expectedAPIKeyRevision uint64,
) (ProviderAPIKeyStatus, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderAPIKeyStatus{}, err
	}
	if err := ref.Validate(); err != nil {
		return ProviderAPIKeyStatus{}, err
	}
	if err := validateExpectedArtifactRevision(
		expectedProviderRevision,
	); err != nil {
		return ProviderAPIKeyStatus{}, err
	}

	record, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return ProviderAPIKeyStatus{}, err
	}
	if record.State != artifact.StateAvailable {
		return ProviderAPIKeyStatus{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Revision != expectedProviderRevision {
		return ProviderAPIKeyStatus{}, basespec.ErrConflict
	}

	current, found, err := a.overlays.GetProviderCredential(ctx, ref)
	if err != nil {
		return ProviderAPIKeyStatus{}, err
	}
	if !found {
		if expectedAPIKeyRevision != 0 {
			return ProviderAPIKeyStatus{}, basespec.ErrConflict
		}
		return ProviderAPIKeyStatus{
			ProviderRevision: record.Revision,
		}, nil
	}
	if current.Revision != expectedAPIKeyRevision {
		return ProviderAPIKeyStatus{}, basespec.ErrConflict
	}

	if err := a.overlays.ClearProviderCredential(
		ctx,
		secret.ClearBindingRequest{
			Key:                      modelOverlay.ProviderCredentialBindingKey(ref),
			ExpectedArtifactRevision: expectedProviderRevision,
			ExpectedBindingRevision:  expectedAPIKeyRevision,
		},
	); err != nil {
		return ProviderAPIKeyStatus{}, err
	}

	return a.GetProviderAPIKeyStatus(ctx, ref)
}

func (a *API) SaveModelSettings(
	ctx context.Context,
	request SaveModelSettingsRequest,
) (ModelView, error) {
	if err := a.ready(ctx); err != nil {
		return ModelView{}, err
	}
	if err := request.Model.Validate(); err != nil {
		return ModelView{}, err
	}
	if err := validateExpectedArtifactRevision(
		request.ExpectedModelRevision,
	); err != nil {
		return ModelView{}, err
	}

	record, err := a.requireKind(
		ctx,
		request.Model,
		modelDomain.ModelArtifactKind,
	)
	if err != nil {
		return ModelView{}, err
	}
	if record.State != artifact.StateAvailable {
		return ModelView{}, fmt.Errorf(
			"%w: Model Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Revision != request.ExpectedModelRevision {
		return ModelView{}, basespec.ErrConflict
	}

	current, found, err := a.overlays.GetModelOverlay(ctx, request.Model)
	if err != nil {
		return ModelView{}, err
	}
	if found {
		if current.Revision != request.ExpectedSettingsRevision {
			return ModelView{}, basespec.ErrConflict
		}
	} else if request.ExpectedSettingsRevision != 0 {
		return ModelView{}, basespec.ErrConflict
	}

	if request.ExpectedSettingsRevision == ^uint64(0) {
		return ModelView{}, fmt.Errorf(
			"%w: Model settings revision is exhausted",
			basespec.ErrInvalid,
		)
	}

	value := modelOverlay.ModelOverlay{
		SchemaVersion:     modelOverlay.OverlaySchemaVersion,
		Revision:          request.ExpectedSettingsRevision + 1,
		Defaults:          cloneRaw(request.Defaults),
		Capabilities:      cloneRaw(request.Capabilities),
		AdapterParameters: cloneRaw(request.AdapterParameters),
	}
	if err := a.overlays.PutModelOverlay(
		ctx,
		request.Model,
		request.ExpectedModelRevision,
		request.ExpectedSettingsRevision,
		value,
	); err != nil {
		return ModelView{}, err
	}

	return a.GetModel(ctx, request.Model)
}

func (a *API) ResetModelSettings(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedModelRevision uint64,
	expectedSettingsRevision uint64,
) (ModelView, error) {
	if err := a.ready(ctx); err != nil {
		return ModelView{}, err
	}
	if err := ref.Validate(); err != nil {
		return ModelView{}, err
	}
	if expectedModelRevision == 0 || expectedSettingsRevision == 0 {
		return ModelView{}, fmt.Errorf(
			"%w: expected Model and Model settings revisions are required",
			basespec.ErrInvalid,
		)
	}

	record, err := a.requireKind(ctx, ref, modelDomain.ModelArtifactKind)
	if err != nil {
		return ModelView{}, err
	}
	if record.Revision != expectedModelRevision {
		return ModelView{}, basespec.ErrConflict
	}

	if err := a.overlays.DeleteModelOverlay(
		ctx,
		ref,
		expectedModelRevision,
		expectedSettingsRevision,
	); err != nil {
		return ModelView{}, err
	}

	return a.GetModel(ctx, ref)
}

func (a *API) providerSettings(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ProviderSettings, error) {
	value, found, err := a.overlays.GetProviderOverlay(ctx, ref)
	if err != nil {
		return ProviderSettings{}, err
	}
	if !found {
		return ProviderSettings{}, nil
	}

	return ProviderSettings{
		Revision:          value.Revision,
		Connection:        cloneRaw(value.Connection),
		Defaults:          cloneRaw(value.Defaults),
		Capabilities:      cloneRaw(value.Capabilities),
		DefaultModel:      cloneOptionalReference(value.DefaultModel),
		AdapterParameters: cloneRaw(value.AdapterParameters),
	}, nil
}

func (a *API) modelSettings(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ModelSettings, error) {
	value, found, err := a.overlays.GetModelOverlay(ctx, ref)
	if err != nil {
		return ModelSettings{}, err
	}
	if !found {
		return ModelSettings{}, nil
	}

	return ModelSettings{
		Revision:          value.Revision,
		Defaults:          cloneRaw(value.Defaults),
		Capabilities:      cloneRaw(value.Capabilities),
		AdapterParameters: cloneRaw(value.AdapterParameters),
	}, nil
}

func (a *API) purgeProviderLocalState(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	return a.overlays.PurgeProviderLocalState(ctx, ref)
}

func (a *API) purgeModelLocalState(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	return a.overlays.PurgeModelLocalState(ctx, ref)
}
