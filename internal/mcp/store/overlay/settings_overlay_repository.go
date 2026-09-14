package overlay

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

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
	if err := value.Validate(); err != nil {
		return ServerOverlay{}, false, err
	}
	return cloneServerOverlay(value), true, nil
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
	if err := value.Validate(); err != nil {
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

func (r *SettingsOverlayRepository) PurgeRoot(
	ctx context.Context,
	rootID root.RootID,
) error {
	if err := rootID.Validate(); err != nil {
		return err
	}
	store, supported := r.values.(SettingsPrefixValueStore)
	if !supported {
		return fmt.Errorf(
			"%w: MCP installation settings store cannot purge protected Root",
			basespec.ErrUnsupported,
		)
	}
	return store.DeleteMCPInstallationPrefix(
		ctx,
		settingsOverlayPrefix+string(rootID)+"/",
	)
}

func (value ServerOverlay) Validate() error {
	if value.SchemaVersion != mcpDomain.InstallationDataSchemaVersion {
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
	return value.ServerData.Validate()
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
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		target,
		basespec.MaxLocalDataBytes,
	); err != nil {
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
	output.ServerData.Inputs = make(
		map[string]mcpDomainServer.InputBinding,
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
