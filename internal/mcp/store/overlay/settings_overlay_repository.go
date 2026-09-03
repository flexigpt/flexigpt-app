package overlay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
)

// SettingsOverlayRepository implements OverlayRepository over application
// settings. It contains only installation metadata and opaque secret refs.
type SettingsOverlayRepository struct {
	values SettingsValueStore
}

func NewSettingsOverlayRepository(
	values SettingsValueStore,
) (*SettingsOverlayRepository, error) {
	if values == nil {
		return nil, fmt.Errorf(
			"%w: MCP installation settings store is required",
			basespec.ErrInvalid,
		)
	}
	return &SettingsOverlayRepository{values: values}, nil
}

func (r *SettingsOverlayRepository) GetServerOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ServerOverlay, bool, error) {
	key, err := serverOverlayStorageKey(ref)
	if err != nil {
		return ServerOverlay{}, false, err
	}

	raw, found, err := r.values.GetMCPInstallationValue(ctx, key)
	if err != nil || !found {
		return ServerOverlay{}, found, err
	}

	var value ServerOverlay
	if err := decodeOverlay(raw, &value); err != nil {
		return ServerOverlay{}, false, err
	}
	if err := ValidateServerOverlay(value); err != nil {
		return ServerOverlay{}, false, err
	}
	return cloneServerOverlay(value), true, nil
}

func (r *SettingsOverlayRepository) GetBundleOverlay(
	ctx context.Context,
	rootID basespec.RootID,
	collectionID basespec.CollectionID,
) (BundleOverlay, bool, error) {
	key, err := bundleOverlayStorageKey(rootID, collectionID)
	if err != nil {
		return BundleOverlay{}, false, err
	}

	raw, found, err := r.values.GetMCPInstallationValue(ctx, key)
	if err != nil || !found {
		return BundleOverlay{}, found, err
	}

	var value BundleOverlay
	if err := decodeOverlay(raw, &value); err != nil {
		return BundleOverlay{}, false, err
	}
	if err := ValidateBundleOverlay(value); err != nil {
		return BundleOverlay{}, false, err
	}
	return value, true, nil
}

func (r *SettingsOverlayRepository) PutServerOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	value ServerOverlay,
) error {
	key, err := serverOverlayStorageKey(ref)
	if err != nil {
		return err
	}
	if err := validateExpectedOverlayRevision(
		expectedRevision,
		value.Revision,
	); err != nil {
		return err
	}
	if err := ValidateServerOverlay(value); err != nil {
		return err
	}

	raw, err := encodeOverlay(value)
	if err != nil {
		return err
	}
	return r.values.PutMCPInstallationValue(
		ctx,
		key,
		expectedRevision,
		raw,
	)
}

func (r *SettingsOverlayRepository) PutBundleOverlay(
	ctx context.Context,
	rootID basespec.RootID,
	collectionID basespec.CollectionID,
	expectedRevision uint64,
	value BundleOverlay,
) error {
	key, err := bundleOverlayStorageKey(rootID, collectionID)
	if err != nil {
		return err
	}
	if err := validateExpectedOverlayRevision(
		expectedRevision,
		value.Revision,
	); err != nil {
		return err
	}
	if err := ValidateBundleOverlay(value); err != nil {
		return err
	}

	raw, err := encodeOverlay(value)
	if err != nil {
		return err
	}
	return r.values.PutMCPInstallationValue(
		ctx,
		key,
		expectedRevision,
		raw,
	)
}

func (r *SettingsOverlayRepository) DeleteServerOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	key, err := serverOverlayStorageKey(ref)
	if err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected MCP server overlay revision is required",
			basespec.ErrInvalid,
		)
	}
	return r.values.DeleteMCPInstallationValue(
		ctx,
		key,
		expectedRevision,
	)
}

func (r *SettingsOverlayRepository) DeleteBundleOverlay(
	ctx context.Context,
	rootID basespec.RootID,
	collectionID basespec.CollectionID,
	expectedRevision uint64,
) error {
	key, err := bundleOverlayStorageKey(rootID, collectionID)
	if err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected MCP bundle overlay revision is required",
			basespec.ErrInvalid,
		)
	}
	return r.values.DeleteMCPInstallationValue(
		ctx,
		key,
		expectedRevision,
	)
}

// PurgeRoot is intentionally available only through trusted protected
// hydration. It removes local overlays before a stale protected topology is
// rebuilt with a new static registration set.
func (r *SettingsOverlayRepository) PurgeRoot(
	ctx context.Context,
	rootID basespec.RootID,
) error {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return err
	}

	store, supported := r.values.(SettingsPrefixValueStore)
	if !supported {
		return fmt.Errorf(
			"%w: MCP installation settings store cannot purge a protected Root",
			basespec.ErrUnsupported,
		)
	}
	return store.DeleteMCPInstallationPrefix(
		ctx,
		settingsOverlayPrefix+string(rootID)+"/",
	)
}

func ValidateServerOverlay(value ServerOverlay) error {
	if value.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP server overlay schema %q",
			basespec.ErrInvalid,
			value.SchemaVersion,
		)
	}
	if value.Revision == 0 {
		return fmt.Errorf(
			"%w: MCP server overlay revision is required",
			basespec.ErrInvalid,
		)
	}
	return mcpStoreServer.ValidateServerData(value.ServerData)
}

func ValidateBundleOverlay(value BundleOverlay) error {
	if value.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP bundle overlay schema %q",
			basespec.ErrInvalid,
			value.SchemaVersion,
		)
	}
	if value.Revision == 0 {
		return fmt.Errorf(
			"%w: MCP bundle overlay revision is required",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func serverOverlayStorageKey(
	ref artifact.ArtifactRef,
) (string, error) {
	if err := ref.Validate(); err != nil {
		return "", err
	}
	return settingsOverlayPrefix +
		string(ref.RootID) +
		"/servers/" +
		string(ref.ArtifactID), nil
}

func bundleOverlayStorageKey(
	rootID basespec.RootID,
	collectionID basespec.CollectionID,
) (string, error) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return "", err
	}
	if err := basespec.ValidateCollectionID(collectionID); err != nil {
		return "", err
	}
	return settingsOverlayPrefix +
		string(rootID) +
		"/bundles/" +
		string(collectionID), nil
}

func validateExpectedOverlayRevision(
	expected uint64,
	next uint64,
) error {
	if next == 0 || next != expected+1 {
		return fmt.Errorf(
			"%w: invalid MCP installation overlay revision transition",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func encodeOverlay(value any) (json.RawMessage, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func decodeOverlay(raw json.RawMessage, target any) error {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf(
			"%w: decode MCP installation overlay: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("MCP installation overlay has trailing JSON")
		}
		return fmt.Errorf(
			"%w: decode MCP installation overlay: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return nil
}

func cloneServerOverlay(input ServerOverlay) ServerOverlay {
	output := input
	output.ServerData = input.ServerData
	output.ServerData.Inputs = make(
		map[string]mcpStoreServer.InputBinding,
		len(input.ServerData.Inputs),
	)
	for name, binding := range input.ServerData.Inputs {
		copyBinding := binding
		if binding.Value != nil {
			value := *binding.Value
			copyBinding.Value = &value
		}
		output.ServerData.Inputs[name] = copyBinding
	}
	output.ServerData.AdditionalPolicies = append(
		[]artifact.ArtifactRef(nil),
		input.ServerData.AdditionalPolicies...,
	)
	return output
}

func IsMCPInstallationSettingsKey(value string) bool {
	return strings.HasPrefix(value, settingsOverlayPrefix)
}
