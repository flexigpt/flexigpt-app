package overlay

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelproviderv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// SettingsOverlayRepository persists Model Store runtime overlays through the
// application's settings implementation. It intentionally has no file-backed
// fallback repository.
type SettingsOverlayRepository struct {
	values SettingsValueStore
}

func NewSettingsOverlayRepository(
	values SettingsValueStore,
) (*SettingsOverlayRepository, error) {
	if values == nil {
		return nil, fmt.Errorf(
			"%w: Model runtime settings store is required",
			basespec.ErrInvalid,
		)
	}
	return &SettingsOverlayRepository{values: values}, nil
}

func (r *SettingsOverlayRepository) GetProviderOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ProviderOverlay, bool, error) {
	if r == nil || r.values == nil {
		return ProviderOverlay{}, false, basespec.ErrClosed
	}

	key, err := providerOverlayStorageKey(ref)
	if err != nil {
		return ProviderOverlay{}, false, err
	}
	raw, found, err := r.values.GetModelRuntimeValue(ctx, key)
	if err != nil || !found {
		return ProviderOverlay{}, found, err
	}

	var value ProviderOverlay
	if err := decodeOverlay(raw, &value); err != nil {
		return ProviderOverlay{}, false, err
	}
	if err := value.Validate(); err != nil {
		return ProviderOverlay{}, false, err
	}
	return value.Clone(), true, nil
}

func (r *SettingsOverlayRepository) PutProviderOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	value ProviderOverlay,
) error {
	if r == nil || r.values == nil {
		return basespec.ErrClosed
	}

	key, err := providerOverlayStorageKey(ref)
	if err != nil {
		return err
	}
	if err := validateExpectedOverlayRevision(
		expectedRevision,
		value.Revision,
	); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}

	raw, err := encodeOverlay(value)
	if err != nil {
		return err
	}
	return r.values.PutModelRuntimeValue(
		ctx,
		key,
		expectedRevision,
		raw,
	)
}

func (r *SettingsOverlayRepository) DeleteProviderOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if r == nil || r.values == nil {
		return basespec.ErrClosed
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Model Provider overlay revision is required",
			basespec.ErrInvalid,
		)
	}

	key, err := providerOverlayStorageKey(ref)
	if err != nil {
		return err
	}
	return r.values.DeleteModelRuntimeValue(
		ctx,
		key,
		expectedRevision,
	)
}

func (r *SettingsOverlayRepository) GetModelOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ModelOverlay, bool, error) {
	if r == nil || r.values == nil {
		return ModelOverlay{}, false, basespec.ErrClosed
	}

	key, err := modelOverlayStorageKey(ref)
	if err != nil {
		return ModelOverlay{}, false, err
	}
	raw, found, err := r.values.GetModelRuntimeValue(ctx, key)
	if err != nil || !found {
		return ModelOverlay{}, found, err
	}

	var value ModelOverlay
	if err := decodeOverlay(raw, &value); err != nil {
		return ModelOverlay{}, false, err
	}
	if err := value.Validate(); err != nil {
		return ModelOverlay{}, false, err
	}
	return value.Clone(), true, nil
}

func (r *SettingsOverlayRepository) PutModelOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	value ModelOverlay,
) error {
	if r == nil || r.values == nil {
		return basespec.ErrClosed
	}

	key, err := modelOverlayStorageKey(ref)
	if err != nil {
		return err
	}
	if err := validateExpectedOverlayRevision(
		expectedRevision,
		value.Revision,
	); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}

	raw, err := encodeOverlay(value)
	if err != nil {
		return err
	}
	return r.values.PutModelRuntimeValue(
		ctx,
		key,
		expectedRevision,
		raw,
	)
}

func (r *SettingsOverlayRepository) DeleteModelOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if r == nil || r.values == nil {
		return basespec.ErrClosed
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Model overlay revision is required",
			basespec.ErrInvalid,
		)
	}

	key, err := modelOverlayStorageKey(ref)
	if err != nil {
		return err
	}
	return r.values.DeleteModelRuntimeValue(
		ctx,
		key,
		expectedRevision,
	)
}

// PurgeRoot is used only by a trusted built-in catalog lifecycle hook after a
// protected topology reset. It removes overlay references, not raw secret
// material. Credential-store lifecycle remains owned by the credential system.
func (r *SettingsOverlayRepository) PurgeRoot(
	ctx context.Context,
	rootID root.RootID,
) error {
	if r == nil || r.values == nil {
		return basespec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return err
	}

	store, supported := r.values.(SettingsPrefixValueStore)
	if !supported {
		return fmt.Errorf(
			"%w: Model runtime settings store cannot purge a Root",
			basespec.ErrUnsupported,
		)
	}
	return store.DeleteModelRuntimePrefix(
		ctx,
		SettingsOverlayPrefix+string(rootID)+"/",
	)
}

func (v ProviderOverlay) Validate() error {
	if v.SchemaVersion != OverlaySchemaVersion {
		return fmt.Errorf(
			"%w: unsupported Model Provider overlay schema %q",
			basespec.ErrInvalid,
			v.SchemaVersion,
		)
	}
	if v.Revision == 0 {
		return fmt.Errorf(
			"%w: Model Provider overlay revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := validateOpaqueCredentialReference(v.CredentialRef); err != nil {
		return err
	}
	if v.DefaultModel != nil {
		if err := v.DefaultModel.Validate(); err != nil {
			return fmt.Errorf("model provider overlay defaultModel: %w", err)
		}
	}

	// Reuse the canonical Provider declaration validator by constructing an
	// otherwise minimal synthetic declaration. The overlay shape can only
	// carry fields that are already valid Provider declaration patches.
	document := modelproviderv1.ProviderDocument{
		Type:              modelproviderv1.ModelProviderType,
		Name:              "overlay-provider",
		Adapter:           "overlay.adapter",
		Connection:        cloneRaw(v.Connection),
		DefaultModel:      cloneReference(v.DefaultModel),
		Defaults:          cloneRaw(v.Defaults),
		Capabilities:      cloneRaw(v.Capabilities),
		AdapterParameters: cloneRaw(v.AdapterParameters),
	}
	if err := document.Validate(); err != nil {
		return fmt.Errorf("model Provider overlay payload: %w", err)
	}
	return nil
}

func (v ModelOverlay) Validate() error {
	if v.SchemaVersion != OverlaySchemaVersion {
		return fmt.Errorf(
			"%w: unsupported Model overlay schema %q",
			basespec.ErrInvalid,
			v.SchemaVersion,
		)
	}
	if v.Revision == 0 {
		return fmt.Errorf(
			"%w: Model overlay revision is required",
			basespec.ErrInvalid,
		)
	}

	// Reuse the canonical Model declaration validator by constructing an
	// otherwise minimal synthetic declaration. The overlay cannot carry a
	// Provider reference or providerModelID, so it cannot mutate identity.
	document := modelv1.ModelDocument{
		Type: modelv1.ModelType,
		Name: "overlay-model",
		Provider: declaration.ArtifactNameReference{
			Name: "overlay-provider",
		},
		ProviderModelID:   "overlay-model",
		Defaults:          cloneRaw(v.Defaults),
		Capabilities:      cloneRaw(v.Capabilities),
		AdapterParameters: cloneRaw(v.AdapterParameters),
	}
	if err := document.Validate(); err != nil {
		return fmt.Errorf("model overlay payload: %w", err)
	}
	return nil
}

func providerOverlayStorageKey(
	ref artifact.ArtifactRef,
) (string, error) {
	if err := ref.Validate(); err != nil {
		return "", err
	}
	return SettingsOverlayPrefix +
		string(ref.RootID) +
		"/providers/" +
		string(ref.ArtifactID), nil
}

func modelOverlayStorageKey(
	ref artifact.ArtifactRef,
) (string, error) {
	if err := ref.Validate(); err != nil {
		return "", err
	}
	return SettingsOverlayPrefix +
		string(ref.RootID) +
		"/models/" +
		string(ref.ArtifactID), nil
}

func validateExpectedOverlayRevision(
	expected uint64,
	next uint64,
) error {
	if next == 0 || next != expected+1 {
		return fmt.Errorf(
			"%w: invalid Model runtime overlay revision transition",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func validateOpaqueCredentialReference(value string) error {
	if value == "" {
		return nil
	}
	if err := basespec.ValidateRequiredText(
		"Model credential reference",
		value,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return fmt.Errorf(
			"%w: Model credential reference cannot contain whitespace",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func encodeOverlay(value any) (json.RawMessage, error) {
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func decodeOverlay(
	raw json.RawMessage,
	target any,
) error {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return err
	}
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		canonical,
		target,
		basespec.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf(
			"%w: decode Model runtime overlay: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return nil
}

func cloneReference(
	value *declaration.ArtifactNameReference,
) *declaration.ArtifactNameReference {
	if value == nil {
		return nil
	}
	output := value.Clone()
	return &output
}

func IsModelRuntimeSettingsKey(value string) bool {
	return strings.HasPrefix(value, SettingsOverlayPrefix)
}
