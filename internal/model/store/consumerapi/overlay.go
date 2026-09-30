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
	credential, credentialFound, err := a.overlays.GetProviderCredential(
		ctx,
		ref,
	)
	if err != nil {
		return ProviderRuntimeOverlayView{}, err
	}

	return providerOverlayView(
		record.Revision,
		value,
		found,
		credential,
		credentialFound,
	), nil
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
	if err := validateExpectedArtifactRevision(
		request.ExpectedArtifactRevision,
	); err != nil {
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
	if record.Revision != request.ExpectedArtifactRevision {
		return ProviderRuntimeOverlayView{}, basespec.ErrConflict
	}

	current, found, err := a.overlays.GetProviderOverlay(
		ctx,
		request.Provider,
	)
	if err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	if found {
		if current.Revision != request.ExpectedOverlayRevision {
			return ProviderRuntimeOverlayView{}, basespec.ErrConflict
		}
	} else if request.ExpectedOverlayRevision != 0 {
		return ProviderRuntimeOverlayView{}, basespec.ErrConflict
	}

	if request.ExpectedOverlayRevision == ^uint64(0) {
		return ProviderRuntimeOverlayView{}, fmt.Errorf(
			"%w: Model Provider overlay revision is exhausted",
			basespec.ErrInvalid,
		)
	}

	value := modelOverlay.ProviderOverlay{
		SchemaVersion:     modelOverlay.OverlaySchemaVersion,
		Revision:          request.ExpectedOverlayRevision + 1,
		Connection:        cloneRaw(request.Connection),
		Defaults:          cloneRaw(request.Defaults),
		Capabilities:      cloneRaw(request.Capabilities),
		DefaultModel:      cloneOptionalReference(request.DefaultModel),
		AdapterParameters: cloneRaw(request.AdapterParameters),
	}
	if err := a.overlays.PutProviderOverlay(
		ctx,
		request.Provider,
		request.ExpectedArtifactRevision,
		request.ExpectedOverlayRevision,
		value,
	); err != nil {
		return ProviderRuntimeOverlayView{}, err
	}

	return a.GetProviderRuntimeOverlay(ctx, request.Provider)
}

func (a *API) DeleteProviderRuntimeOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
) error {
	if err := a.ready(ctx); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if expectedArtifactRevision == 0 ||
		expectedOverlayRevision == 0 {
		return fmt.Errorf(
			"%w: expected Artifact and Model Provider overlay revisions are required",
			basespec.ErrInvalid,
		)
	}

	record, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelProviderArtifactKind,
	)
	if err != nil {
		return err
	}
	if record.Revision != expectedArtifactRevision {
		return basespec.ErrConflict
	}

	return a.overlays.DeleteProviderOverlay(
		ctx,
		ref,
		expectedArtifactRevision,
		expectedOverlayRevision,
	)
}

func (a *API) UpdateProviderCredential(
	ctx context.Context,
	request ProviderCredentialUpdateRequest,
) (ProviderRuntimeOverlayView, error) {
	if err := a.ready(ctx); err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	if err := request.Provider.Validate(); err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	if err := validateExpectedArtifactRevision(
		request.ExpectedArtifactRevision,
	); err != nil {
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
	if record.Revision != request.ExpectedArtifactRevision {
		return ProviderRuntimeOverlayView{}, basespec.ErrConflict
	}

	key := modelOverlay.ProviderCredentialBindingKey(
		request.Provider,
	)
	if request.Secret == "" {
		current, found, err := a.overlays.GetProviderCredential(
			ctx,
			request.Provider,
		)
		if err != nil {
			return ProviderRuntimeOverlayView{}, err
		}
		if !found {
			if request.ExpectedBindingRevision != 0 {
				return ProviderRuntimeOverlayView{}, basespec.ErrConflict
			}
			return a.GetProviderRuntimeOverlay(ctx, request.Provider)
		}
		if current.Revision != request.ExpectedBindingRevision {
			return ProviderRuntimeOverlayView{}, basespec.ErrConflict
		}

		if err := a.overlays.ClearProviderCredential(
			ctx,
			secret.ClearBindingRequest{
				Key:                      key,
				ExpectedArtifactRevision: request.ExpectedArtifactRevision,
				ExpectedBindingRevision:  request.ExpectedBindingRevision,
			},
		); err != nil {
			return ProviderRuntimeOverlayView{}, err
		}
		return a.GetProviderRuntimeOverlay(ctx, request.Provider)
	}

	if _, err := a.overlays.ReplaceProviderCredential(
		ctx,
		secret.ReplaceBindingRequest{
			Key:                      key,
			ExpectedArtifactRevision: request.ExpectedArtifactRevision,
			ExpectedBindingRevision:  request.ExpectedBindingRevision,
			Value:                    request.Secret,
		},
	); err != nil {
		return ProviderRuntimeOverlayView{}, err
	}
	return a.GetProviderRuntimeOverlay(ctx, request.Provider)
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
	return modelOverlayView(record.Revision, value, found), nil
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
	if err := validateExpectedArtifactRevision(
		request.ExpectedArtifactRevision,
	); err != nil {
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
	if record.Revision != request.ExpectedArtifactRevision {
		return ModelRuntimeOverlayView{}, basespec.ErrConflict
	}

	current, found, err := a.overlays.GetModelOverlay(
		ctx,
		request.Model,
	)
	if err != nil {
		return ModelRuntimeOverlayView{}, err
	}
	if found {
		if current.Revision != request.ExpectedOverlayRevision {
			return ModelRuntimeOverlayView{}, basespec.ErrConflict
		}
	} else if request.ExpectedOverlayRevision != 0 {
		return ModelRuntimeOverlayView{}, basespec.ErrConflict
	}

	if request.ExpectedOverlayRevision == ^uint64(0) {
		return ModelRuntimeOverlayView{}, fmt.Errorf(
			"%w: Model overlay revision is exhausted",
			basespec.ErrInvalid,
		)
	}

	value := modelOverlay.ModelOverlay{
		SchemaVersion:     modelOverlay.OverlaySchemaVersion,
		Revision:          request.ExpectedOverlayRevision + 1,
		Defaults:          cloneRaw(request.Defaults),
		Capabilities:      cloneRaw(request.Capabilities),
		AdapterParameters: cloneRaw(request.AdapterParameters),
	}
	if err := a.overlays.PutModelOverlay(
		ctx,
		request.Model,
		request.ExpectedArtifactRevision,
		request.ExpectedOverlayRevision,
		value,
	); err != nil {
		return ModelRuntimeOverlayView{}, err
	}

	return a.GetModelRuntimeOverlay(ctx, request.Model)
}

func (a *API) DeleteModelRuntimeOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
) error {
	if err := a.ready(ctx); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if expectedArtifactRevision == 0 ||
		expectedOverlayRevision == 0 {
		return fmt.Errorf(
			"%w: expected Artifact and Model overlay revisions are required",
			basespec.ErrInvalid,
		)
	}

	record, err := a.requireKind(
		ctx,
		ref,
		modelDomain.ModelArtifactKind,
	)
	if err != nil {
		return err
	}
	if record.Revision != expectedArtifactRevision {
		return basespec.ErrConflict
	}

	return a.overlays.DeleteModelOverlay(
		ctx,
		ref,
		expectedArtifactRevision,
		expectedOverlayRevision,
	)
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

func providerOverlayView(
	artifactRevision uint64,
	value modelOverlay.ProviderOverlay,
	found bool,
	credential secret.Binding,
	credentialFound bool,
) ProviderRuntimeOverlayView {
	output := ProviderRuntimeOverlayView{
		ArtifactRevision: artifactRevision,
	}
	if found {
		output.Revision = value.Revision
		output.Connection = cloneRaw(value.Connection)
		output.Defaults = cloneRaw(value.Defaults)
		output.Capabilities = cloneRaw(value.Capabilities)
		output.DefaultModel = cloneOptionalReference(value.DefaultModel)
		output.AdapterParameters = cloneRaw(value.AdapterParameters)
	}
	if credentialFound {
		output.CredentialRevision = credential.Revision
		output.CredentialConfigured = credential.Active()
		if credential.Active() {
			output.CredentialSHA256 = credential.SHA256
		}
	}
	return output
}

func modelOverlayView(
	artifactRevision uint64,
	value modelOverlay.ModelOverlay,
	found bool,
) ModelRuntimeOverlayView {
	output := ModelRuntimeOverlayView{
		ArtifactRevision: artifactRevision,
	}
	if found {
		output.Revision = value.Revision
		output.Defaults = cloneRaw(value.Defaults)
		output.Capabilities = cloneRaw(value.Capabilities)
		output.AdapterParameters = cloneRaw(value.AdapterParameters)
	}
	return output
}
