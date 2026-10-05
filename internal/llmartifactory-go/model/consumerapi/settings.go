package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/overlay"
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
	if record.State != artifactModel.StateAvailable {
		return ProviderView{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Revision != request.ExpectedProviderRevision {
		return ProviderView{}, spec.ErrConflict
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
			return ProviderView{}, spec.ErrConflict
		}
	} else if request.ExpectedSettingsRevision != 0 {
		return ProviderView{}, spec.ErrConflict
	}

	if request.ExpectedSettingsRevision == ^uint64(0) {
		return ProviderView{}, fmt.Errorf(
			"%w: Model Provider settings revision is exhausted",
			spec.ErrInvalid,
		)
	}

	connection, err := encodeOptionalObject(request.Connection)
	if err != nil {
		return ProviderView{}, fmt.Errorf(
			"encode Model Provider settings connection: %w",
			err,
		)
	}
	defaults, err := encodeOptionalObject(request.Defaults)
	if err != nil {
		return ProviderView{}, fmt.Errorf(
			"encode Model Provider settings defaults: %w",
			err,
		)
	}
	capabilities, err := encodeOptionalObject(request.Capabilities)
	if err != nil {
		return ProviderView{}, fmt.Errorf(
			"encode Model Provider settings capabilities: %w",
			err,
		)
	}
	adapterParameters, err := encodeOptionalObject(
		request.AdapterParameters,
	)
	if err != nil {
		return ProviderView{}, fmt.Errorf(
			"encode Model Provider settings adapter parameters: %w",
			err,
		)
	}

	value := modelOverlay.ProviderOverlay{
		SchemaVersion: modelOverlay.OverlaySchemaVersion,
		Revision:      request.ExpectedSettingsRevision + 1,
		Connection:    connection,
		Defaults:      defaults,
		Capabilities:  capabilities,
		DefaultModel: modelDomain.ArtifactNameReferenceToDeclaration(
			request.DefaultModel,
		),
		AdapterParameters: adapterParameters,
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
	ref artifactModel.ArtifactRef,
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
			spec.ErrInvalid,
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
		return ProviderView{}, spec.ErrConflict
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
	ref artifactModel.ArtifactRef,
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
	if record.State != artifactModel.StateAvailable {
		return ProviderAPIKeyStatus{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
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
			spec.ErrInvalid,
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
	if record.State != artifactModel.StateAvailable {
		return ProviderAPIKeyStatus{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Revision != request.ExpectedProviderRevision {
		return ProviderAPIKeyStatus{}, spec.ErrConflict
	}

	if _, err := a.overlays.ReplaceProviderCredential(
		ctx,
		secretModel.ReplaceBindingRequest{
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
	ref artifactModel.ArtifactRef,
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
	if record.State != artifactModel.StateAvailable {
		return ProviderAPIKeyStatus{}, fmt.Errorf(
			"%w: Model Provider Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Revision != expectedProviderRevision {
		return ProviderAPIKeyStatus{}, spec.ErrConflict
	}

	current, found, err := a.overlays.GetProviderCredential(ctx, ref)
	if err != nil {
		return ProviderAPIKeyStatus{}, err
	}
	if !found {
		if expectedAPIKeyRevision != 0 {
			return ProviderAPIKeyStatus{}, spec.ErrConflict
		}
		return ProviderAPIKeyStatus{
			ProviderRevision: record.Revision,
		}, nil
	}
	if current.Revision != expectedAPIKeyRevision {
		return ProviderAPIKeyStatus{}, spec.ErrConflict
	}

	if err := a.overlays.ClearProviderCredential(
		ctx,
		secretModel.ClearBindingRequest{
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
	if record.State != artifactModel.StateAvailable {
		return ModelView{}, fmt.Errorf(
			"%w: Model Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if record.Revision != request.ExpectedModelRevision {
		return ModelView{}, spec.ErrConflict
	}

	current, found, err := a.overlays.GetModelOverlay(ctx, request.Model)
	if err != nil {
		return ModelView{}, err
	}
	if found {
		if current.Revision != request.ExpectedSettingsRevision {
			return ModelView{}, spec.ErrConflict
		}
	} else if request.ExpectedSettingsRevision != 0 {
		return ModelView{}, spec.ErrConflict
	}

	if request.ExpectedSettingsRevision == ^uint64(0) {
		return ModelView{}, fmt.Errorf(
			"%w: Model settings revision is exhausted",
			spec.ErrInvalid,
		)
	}

	defaults, err := encodeOptionalObject(request.Defaults)
	if err != nil {
		return ModelView{}, fmt.Errorf(
			"encode Model settings defaults: %w",
			err,
		)
	}
	capabilities, err := encodeOptionalObject(request.Capabilities)
	if err != nil {
		return ModelView{}, fmt.Errorf(
			"encode Model settings capabilities: %w",
			err,
		)
	}
	adapterParameters, err := encodeOptionalObject(
		request.AdapterParameters,
	)
	if err != nil {
		return ModelView{}, fmt.Errorf(
			"encode Model settings adapter parameters: %w",
			err,
		)
	}

	value := modelOverlay.ModelOverlay{
		SchemaVersion:     modelOverlay.OverlaySchemaVersion,
		Revision:          request.ExpectedSettingsRevision + 1,
		Defaults:          defaults,
		Capabilities:      capabilities,
		AdapterParameters: adapterParameters,
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
	ref artifactModel.ArtifactRef,
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
			spec.ErrInvalid,
		)
	}

	record, err := a.requireKind(ctx, ref, modelDomain.ModelArtifactKind)
	if err != nil {
		return ModelView{}, err
	}
	if record.Revision != expectedModelRevision {
		return ModelView{}, spec.ErrConflict
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
	ref artifactModel.ArtifactRef,
) (ProviderSettings, error) {
	value, found, err := a.overlays.GetProviderOverlay(ctx, ref)
	if err != nil {
		return ProviderSettings{}, err
	}
	if !found {
		return ProviderSettings{}, nil
	}

	connection, err := decodeOptionalObject[modelDomain.ConnectionPatch](
		value.Connection,
	)
	if err != nil {
		return ProviderSettings{}, err
	}
	defaults, err := decodeOptionalObject[modelDomain.DefaultsPatch](
		value.Defaults,
	)
	if err != nil {
		return ProviderSettings{}, err
	}
	capabilities, err := decodeOptionalObject[modelDomain.CapabilitiesPatch](value.Capabilities)
	if err != nil {
		return ProviderSettings{}, err
	}
	adapterParameters, err := decodeOptionalObject[modelDomain.AdapterParameters](value.AdapterParameters)
	if err != nil {
		return ProviderSettings{}, err
	}

	return ProviderSettings{
		Revision:     value.Revision,
		Connection:   connection,
		Defaults:     defaults,
		Capabilities: capabilities,
		DefaultModel: modelDomain.ArtifactNameReferenceFromOptionalDeclaration(
			value.DefaultModel,
		),
		AdapterParameters: adapterParameters,
	}, nil
}

func (a *API) modelSettings(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ModelSettings, error) {
	value, found, err := a.overlays.GetModelOverlay(ctx, ref)
	if err != nil {
		return ModelSettings{}, err
	}
	if !found {
		return ModelSettings{}, nil
	}

	defaults, err := decodeOptionalObject[modelDomain.DefaultsPatch](
		value.Defaults,
	)
	if err != nil {
		return ModelSettings{}, err
	}
	capabilities, err := decodeOptionalObject[modelDomain.CapabilitiesPatch](value.Capabilities)
	if err != nil {
		return ModelSettings{}, err
	}
	adapterParameters, err := decodeOptionalObject[modelDomain.AdapterParameters](value.AdapterParameters)
	if err != nil {
		return ModelSettings{}, err
	}

	return ModelSettings{
		Revision:          value.Revision,
		Defaults:          defaults,
		Capabilities:      capabilities,
		AdapterParameters: adapterParameters,
	}, nil
}

func encodeOptionalObject[T any](
	value *T,
) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}

	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		spec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func decodeOptionalObject[T any](
	raw json.RawMessage,
) (*T, error) {
	if len(raw) == 0 {
		//nolint:nilnil // Ok.
		return nil, nil
	}

	var output T
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		raw,
		&output,
		spec.MaxLocalDataBytes,
	); err != nil {
		return nil, fmt.Errorf(
			"decode typed Model local settings: %w",
			err,
		)
	}
	return &output, nil
}

func (a *API) purgeProviderLocalState(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	return a.overlays.PurgeProviderLocalState(ctx, ref)
}

func (a *API) purgeModelLocalState(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	return a.overlays.PurgeModelLocalState(ctx, ref)
}
