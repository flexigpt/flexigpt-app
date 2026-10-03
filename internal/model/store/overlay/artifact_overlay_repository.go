package overlay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type ArtifactOverlayDependencies struct {
	Artifacts        artifact.API
	Protection       root.ProtectionAPI
	ProtectedOverlay overlay.API
	Secrets          secret.API
	LocalState       artifactcleanup.API
}

type ArtifactOverlayRepository struct {
	artifacts        artifact.API
	protection       root.ProtectionAPI
	protectedOverlay overlay.API
	secrets          secret.API
	localState       artifactcleanup.API
}

func NewArtifactOverlayRepository(
	dependencies ArtifactOverlayDependencies,
) (*ArtifactOverlayRepository, error) {
	if dependencies.Artifacts == nil ||
		dependencies.Protection == nil ||
		dependencies.ProtectedOverlay == nil ||
		dependencies.Secrets == nil ||
		dependencies.LocalState == nil {
		return nil, fmt.Errorf(
			"%w: Model Artifact overlay dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	return &ArtifactOverlayRepository{
		artifacts:        dependencies.Artifacts,
		protection:       dependencies.Protection,
		protectedOverlay: dependencies.ProtectedOverlay,
		secrets:          dependencies.Secrets,
		localState:       dependencies.LocalState,
	}, nil
}

func (r *ArtifactOverlayRepository) GetProviderOverlay(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ProviderOverlay, bool, error) {
	record, err := r.artifact(ctx, ref)
	if err != nil {
		return ProviderOverlay{}, false, err
	}

	if r.protection.IsProtectedRoot(record.RootID) {
		stored, found, err := r.protectedOverlay.Get(
			ctx,
			ref,
			ProviderRuntimeNamespace,
		)
		if err != nil || !found {
			return ProviderOverlay{}, found, err
		}
		value, err := decodeProtectedProviderOverlay(stored)
		if err != nil {
			return ProviderOverlay{}, false, err
		}
		return value.Clone(), true, nil
	}

	return decodeMutableProviderOverlay(record.Data)
}

func (r *ArtifactOverlayRepository) PutProviderOverlay(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
	value ProviderOverlay,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if err := validateOverlayTransition(
		expectedOverlayRevision,
		value.Revision,
	); err != nil {
		return err
	}

	record, err := r.artifact(ctx, ref)
	if err != nil {
		return err
	}
	if record.Revision != expectedArtifactRevision {
		return spec.ErrConflict
	}

	if r.protection.IsProtectedRoot(record.RootID) {
		payload, err := encodeProtectedProviderOverlay(value)
		if err != nil {
			return err
		}
		stored, err := r.protectedOverlay.Put(
			ctx,
			overlayModel.PutRequest{
				Artifact:                 ref,
				Namespace:                ProviderRuntimeNamespace,
				SchemaVersion:            value.SchemaVersion,
				Payload:                  payload,
				ExpectedArtifactRevision: expectedArtifactRevision,
				ExpectedOverlayRevision:  expectedOverlayRevision,
			},
		)
		if err != nil {
			return err
		}
		if stored.Revision != value.Revision {
			return fmt.Errorf(
				"%w: protected Model Provider overlay revision changed unexpectedly",
				spec.ErrConflict,
			)
		}
		return nil
	}

	current, found, err := decodeMutableProviderOverlay(record.Data)
	if err != nil {
		return err
	}
	if found {
		if current.Revision != expectedOverlayRevision {
			return spec.ErrConflict
		}
	} else if expectedOverlayRevision != 0 {
		return spec.ErrConflict
	}

	raw, err := encodeMutableProviderOverlay(value)
	if err != nil {
		return err
	}
	return r.putMutableData(
		ctx,
		record,
		ProviderRuntimeDataNamespace,
		raw,
	)
}

func (r *ArtifactOverlayRepository) DeleteProviderOverlay(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
) error {
	record, err := r.artifact(ctx, ref)
	if err != nil {
		return err
	}
	if record.Revision != expectedArtifactRevision {
		return spec.ErrConflict
	}

	if r.protection.IsProtectedRoot(record.RootID) {
		return r.protectedOverlay.Delete(
			ctx,
			ref,
			ProviderRuntimeNamespace,
			expectedArtifactRevision,
			expectedOverlayRevision,
		)
	}

	current, found, err := decodeMutableProviderOverlay(record.Data)
	if err != nil {
		return err
	}
	if !found ||
		current.Revision != expectedOverlayRevision {
		return spec.ErrConflict
	}

	return r.removeMutableData(
		ctx,
		record,
		ProviderRuntimeDataNamespace,
	)
}

func (r *ArtifactOverlayRepository) GetModelOverlay(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ModelOverlay, bool, error) {
	record, err := r.artifact(ctx, ref)
	if err != nil {
		return ModelOverlay{}, false, err
	}

	if r.protection.IsProtectedRoot(record.RootID) {
		stored, found, err := r.protectedOverlay.Get(
			ctx,
			ref,
			ModelRuntimeNamespace,
		)
		if err != nil || !found {
			return ModelOverlay{}, found, err
		}
		value, err := decodeProtectedModelOverlay(stored)
		if err != nil {
			return ModelOverlay{}, false, err
		}
		return value.Clone(), true, nil
	}

	return decodeMutableModelOverlay(record.Data)
}

func (r *ArtifactOverlayRepository) PutModelOverlay(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
	value ModelOverlay,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if err := validateOverlayTransition(
		expectedOverlayRevision,
		value.Revision,
	); err != nil {
		return err
	}

	record, err := r.artifact(ctx, ref)
	if err != nil {
		return err
	}
	if record.Revision != expectedArtifactRevision {
		return spec.ErrConflict
	}

	if r.protection.IsProtectedRoot(record.RootID) {
		payload, err := encodeProtectedModelOverlay(value)
		if err != nil {
			return err
		}
		stored, err := r.protectedOverlay.Put(
			ctx,
			overlayModel.PutRequest{
				Artifact:                 ref,
				Namespace:                ModelRuntimeNamespace,
				SchemaVersion:            value.SchemaVersion,
				Payload:                  payload,
				ExpectedArtifactRevision: expectedArtifactRevision,
				ExpectedOverlayRevision:  expectedOverlayRevision,
			},
		)
		if err != nil {
			return err
		}
		if stored.Revision != value.Revision {
			return fmt.Errorf(
				"%w: protected Model overlay revision changed unexpectedly",
				spec.ErrConflict,
			)
		}
		return nil
	}

	current, found, err := decodeMutableModelOverlay(record.Data)
	if err != nil {
		return err
	}
	if found {
		if current.Revision != expectedOverlayRevision {
			return spec.ErrConflict
		}
	} else if expectedOverlayRevision != 0 {
		return spec.ErrConflict
	}

	raw, err := encodeMutableModelOverlay(value)
	if err != nil {
		return err
	}
	return r.putMutableData(
		ctx,
		record,
		ModelRuntimeDataNamespace,
		raw,
	)
}

func (r *ArtifactOverlayRepository) DeleteModelOverlay(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
) error {
	record, err := r.artifact(ctx, ref)
	if err != nil {
		return err
	}
	if record.Revision != expectedArtifactRevision {
		return spec.ErrConflict
	}

	if r.protection.IsProtectedRoot(record.RootID) {
		return r.protectedOverlay.Delete(
			ctx,
			ref,
			ModelRuntimeNamespace,
			expectedArtifactRevision,
			expectedOverlayRevision,
		)
	}

	current, found, err := decodeMutableModelOverlay(record.Data)
	if err != nil {
		return err
	}
	if !found ||
		current.Revision != expectedOverlayRevision {
		return spec.ErrConflict
	}

	return r.removeMutableData(
		ctx,
		record,
		ModelRuntimeDataNamespace,
	)
}

func (r *ArtifactOverlayRepository) GetProviderCredential(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (secretModel.Binding, bool, error) {
	return r.secrets.GetBinding(
		ctx,
		ProviderCredentialBindingKey(ref),
	)
}

func (r *ArtifactOverlayRepository) ReplaceProviderCredential(
	ctx context.Context,
	request secretModel.ReplaceBindingRequest,
) (secretModel.Binding, error) {
	if request.Key.Namespace != ProviderRuntimeNamespace ||
		request.Key.Slot != ProviderCredentialSlot {
		return secretModel.Binding{}, fmt.Errorf(
			"%w: unsupported Model Provider secret binding slot",
			spec.ErrInvalid,
		)
	}
	return r.secrets.ReplaceBinding(ctx, request)
}

func (r *ArtifactOverlayRepository) ClearProviderCredential(
	ctx context.Context,
	request secretModel.ClearBindingRequest,
) error {
	if request.Key.Namespace != ProviderRuntimeNamespace ||
		request.Key.Slot != ProviderCredentialSlot {
		return fmt.Errorf(
			"%w: unsupported Model Provider secret binding slot",
			spec.ErrInvalid,
		)
	}
	return r.secrets.ClearBinding(ctx, request)
}

func (r *ArtifactOverlayRepository) PurgeProviderLocalState(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	return r.purgeLocalState(
		ctx,
		ref,
		ProviderRuntimeDataNamespace,
	)
}

func (r *ArtifactOverlayRepository) PurgeModelLocalState(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	return r.purgeLocalState(
		ctx,
		ref,
		ModelRuntimeDataNamespace,
	)
}

func (r *ArtifactOverlayRepository) purgeLocalState(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	dataNamespace string,
) error {
	record, err := r.artifact(ctx, ref)
	if err != nil {
		return err
	}

	if err := r.localState.PurgeArtifactLocalState(ctx, ref); err != nil {
		return err
	}
	if r.protection.IsProtectedRoot(record.RootID) {
		return nil
	}
	return r.removeMutableData(ctx, record, dataNamespace)
}

func (r *ArtifactOverlayRepository) artifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	if r == nil ||
		r.artifacts == nil ||
		r.protection == nil ||
		r.protectedOverlay == nil ||
		r.secrets == nil ||
		r.localState == nil {
		return artifactModel.Artifact{}, spec.ErrClosed
	}
	return r.artifacts.Get(ctx, ref)
}

func (r *ArtifactOverlayRepository) putMutableData(
	ctx context.Context,
	record artifactModel.Artifact,
	namespace string,
	value json.RawMessage,
) error {
	fields, err := artifactModel.DecodeDataObject(record.Data)
	if err != nil {
		return err
	}
	fields[namespace] = append(json.RawMessage(nil), value...)

	data, err := artifactModel.EncodeDataObject(fields)
	if err != nil {
		return err
	}
	_, err = r.artifacts.UpdateData(
		ctx,
		record.Ref(),
		record.Revision,
		data,
	)
	return err
}

func (r *ArtifactOverlayRepository) removeMutableData(
	ctx context.Context,
	record artifactModel.Artifact,
	namespace string,
) error {
	fields, err := artifactModel.DecodeDataObject(record.Data)
	if err != nil {
		return err
	}
	if _, found := fields[namespace]; !found {
		return nil
	}
	delete(fields, namespace)

	data, err := artifactModel.EncodeDataObject(fields)
	if err != nil {
		return err
	}
	_, err = r.artifacts.UpdateData(
		ctx,
		record.Ref(),
		record.Revision,
		data,
	)
	return err
}

type providerOverlayPayload struct {
	Connection        json.RawMessage                    `json:"connection,omitempty"`
	Defaults          json.RawMessage                    `json:"defaults,omitempty"`
	Capabilities      json.RawMessage                    `json:"capabilities,omitempty"`
	DefaultModel      *declaration.ArtifactNameReference `json:"defaultModel,omitempty"`
	AdapterParameters json.RawMessage                    `json:"adapterParameters,omitempty"`
}

type modelOverlayPayload struct {
	Defaults          json.RawMessage `json:"defaults,omitempty"`
	Capabilities      json.RawMessage `json:"capabilities,omitempty"`
	AdapterParameters json.RawMessage `json:"adapterParameters,omitempty"`
}

func encodeProtectedProviderOverlay(
	value ProviderOverlay,
) (json.RawMessage, error) {
	return jsonutil.MarshalCanonicalObject(
		providerOverlayPayload{
			Connection:        cloneRaw(value.Connection),
			Defaults:          cloneRaw(value.Defaults),
			Capabilities:      cloneRaw(value.Capabilities),
			DefaultModel:      cloneReference(value.DefaultModel),
			AdapterParameters: cloneRaw(value.AdapterParameters),
		},
		spec.MaxLocalDataBytes,
	)
}

func decodeProtectedProviderOverlay(
	record overlayModel.Record,
) (ProviderOverlay, error) {
	var payload providerOverlayPayload
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		record.Payload,
		&payload,
		spec.MaxLocalDataBytes,
	); err != nil {
		return ProviderOverlay{}, fmt.Errorf(
			"%w: decode protected Model Provider overlay: %w",
			spec.ErrInvalid,
			err,
		)
	}

	value := ProviderOverlay{
		SchemaVersion:     record.SchemaVersion,
		Revision:          record.Revision,
		Connection:        cloneRaw(payload.Connection),
		Defaults:          cloneRaw(payload.Defaults),
		Capabilities:      cloneRaw(payload.Capabilities),
		DefaultModel:      cloneReference(payload.DefaultModel),
		AdapterParameters: cloneRaw(payload.AdapterParameters),
	}
	if err := value.Validate(); err != nil {
		return ProviderOverlay{}, err
	}
	return value, nil
}

func encodeProtectedModelOverlay(
	value ModelOverlay,
) (json.RawMessage, error) {
	return jsonutil.MarshalCanonicalObject(
		modelOverlayPayload{
			Defaults:          cloneRaw(value.Defaults),
			Capabilities:      cloneRaw(value.Capabilities),
			AdapterParameters: cloneRaw(value.AdapterParameters),
		},
		spec.MaxLocalDataBytes,
	)
}

func decodeProtectedModelOverlay(
	record overlayModel.Record,
) (ModelOverlay, error) {
	var payload modelOverlayPayload
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		record.Payload,
		&payload,
		spec.MaxLocalDataBytes,
	); err != nil {
		return ModelOverlay{}, fmt.Errorf(
			"%w: decode protected Model overlay: %w",
			spec.ErrInvalid,
			err,
		)
	}

	value := ModelOverlay{
		SchemaVersion:     record.SchemaVersion,
		Revision:          record.Revision,
		Defaults:          cloneRaw(payload.Defaults),
		Capabilities:      cloneRaw(payload.Capabilities),
		AdapterParameters: cloneRaw(payload.AdapterParameters),
	}
	if err := value.Validate(); err != nil {
		return ModelOverlay{}, err
	}
	return value, nil
}

func decodeMutableProviderOverlay(
	raw json.RawMessage,
) (ProviderOverlay, bool, error) {
	fields, err := artifactModel.DecodeDataObject(raw)
	if err != nil {
		return ProviderOverlay{}, false, err
	}
	value, found := fields[ProviderRuntimeDataNamespace]
	if !found {
		return ProviderOverlay{}, false, nil
	}

	var output ProviderOverlay
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		value,
		&output,
		spec.MaxLocalDataBytes,
	); err != nil {
		return ProviderOverlay{}, false, fmt.Errorf(
			"%w: decode mutable Model Provider overlay: %w",
			spec.ErrInvalid,
			err,
		)
	}
	if err := output.Validate(); err != nil {
		return ProviderOverlay{}, false, err
	}
	return output.Clone(), true, nil
}

func encodeMutableProviderOverlay(
	value ProviderOverlay,
) (json.RawMessage, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	return jsonutil.MarshalCanonicalObject(
		value,
		spec.MaxLocalDataBytes,
	)
}

func decodeMutableModelOverlay(
	raw json.RawMessage,
) (ModelOverlay, bool, error) {
	fields, err := artifactModel.DecodeDataObject(raw)
	if err != nil {
		return ModelOverlay{}, false, err
	}
	value, found := fields[ModelRuntimeDataNamespace]
	if !found {
		return ModelOverlay{}, false, nil
	}

	var output ModelOverlay
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		value,
		&output,
		spec.MaxLocalDataBytes,
	); err != nil {
		return ModelOverlay{}, false, fmt.Errorf(
			"%w: decode mutable Model overlay: %w",
			spec.ErrInvalid,
			err,
		)
	}
	if err := output.Validate(); err != nil {
		return ModelOverlay{}, false, err
	}
	return output.Clone(), true, nil
}

func encodeMutableModelOverlay(
	value ModelOverlay,
) (json.RawMessage, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	return jsonutil.MarshalCanonicalObject(
		value,
		spec.MaxLocalDataBytes,
	)
}

func validateOverlayTransition(
	expected uint64,
	next uint64,
) error {
	if expected == ^uint64(0) ||
		next == 0 ||
		next != expected+1 {
		return fmt.Errorf(
			"%w: invalid Model runtime overlay revision transition",
			spec.ErrInvalid,
		)
	}
	return nil
}
