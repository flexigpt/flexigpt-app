package installation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const SchemaVersion = "v1"

type InputBinding struct {
	Value     *string `json:"value,omitempty"`
	SecretRef string  `json:"secretRef,omitempty"`
}

type ServerData struct {
	SchemaVersion string `json:"schemaVersion"`

	SelectedConnectionProfile string                  `json:"selectedConnectionProfile,omitempty"`
	Inputs                    map[string]InputBinding `json:"inputs,omitempty"`
	AdditionalPolicies        []artifact.ArtifactRef  `json:"additionalPolicies,omitempty"`
}

// ServerOverlay intentionally stores ServerData as a named nested value.
// Anonymous embedding would produce two schemaVersion fields at JSON encoding
// time and causes the embedded ServerData schemaVersion to decode as empty.
type ServerOverlay struct {
	SchemaVersion  string     `json:"schemaVersion"`
	Revision       uint64     `json:"revision"`
	RuntimeEnabled bool       `json:"runtimeEnabled"`
	ServerData     ServerData `json:"serverData"`
}

type BundleOverlay struct {
	SchemaVersion  string `json:"schemaVersion"`
	Revision       uint64 `json:"revision"`
	RuntimeEnabled bool   `json:"runtimeEnabled"`
}

type OverlayRepository interface {
	GetServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (ServerOverlay, bool, error)

	GetBundleOverlay(
		ctx context.Context,
		rootID basespec.RootID,
		collectionID basespec.CollectionID,
	) (BundleOverlay, bool, error)

	PutServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		value ServerOverlay,
	) error

	PutBundleOverlay(
		ctx context.Context,
		rootID basespec.RootID,
		collectionID basespec.CollectionID,
		expectedRevision uint64,
		value BundleOverlay,
	) error

	DeleteServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error

	DeleteBundleOverlay(
		ctx context.Context,
		rootID basespec.RootID,
		collectionID basespec.CollectionID,
		expectedRevision uint64,
	) error
}

func DefaultServerData() ServerData {
	return ServerData{
		SchemaVersion: SchemaVersion,
		Inputs:        map[string]InputBinding{},
	}
}

func EncodeServerData(
	input ServerData,
) (json.RawMessage, error) {
	value := input
	value.Inputs = maps.Clone(input.Inputs)
	value.AdditionalPolicies = append(
		[]artifact.ArtifactRef(nil),
		input.AdditionalPolicies...,
	)
	if err := ValidateServerData(value); err != nil {
		return nil, err
	}
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

func DecodeServerData(
	raw json.RawMessage,
) (ServerData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return ServerData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()

	var value ServerData
	if err := decoder.Decode(&value); err != nil {
		return ServerData{}, fmt.Errorf(
			"%w: decode MCP installation data: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("MCP installation data has trailing JSON")
		}
		return ServerData{}, fmt.Errorf("%w: %w", basespec.ErrInvalid, err)
	}
	if err := ValidateServerData(value); err != nil {
		return ServerData{}, err
	}
	value.Inputs = maps.Clone(value.Inputs)
	value.AdditionalPolicies = append(
		[]artifact.ArtifactRef(nil),
		value.AdditionalPolicies...,
	)
	return value, nil
}

func ValidateServerData(value ServerData) error {
	if value.SchemaVersion != SchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP installation schema %q",
			basespec.ErrInvalid,
			value.SchemaVersion,
		)
	}

	seen := make(map[artifact.ArtifactRef]struct{})
	for name, binding := range value.Inputs {
		if name == "" {
			return fmt.Errorf(
				"%w: MCP installation input name is empty",
				basespec.ErrInvalid,
			)
		}
		if binding.Value != nil && binding.SecretRef != "" {
			return fmt.Errorf(
				"%w: MCP input %q has both value and secretRef",
				basespec.ErrInvalid,
				name,
			)
		}
	}
	for _, ref := range value.AdditionalPolicies {
		if err := ref.Validate(); err != nil {
			return err
		}
		if _, duplicate := seen[ref]; duplicate {
			return fmt.Errorf(
				"%w: duplicate additional MCP policy",
				basespec.ErrInvalid,
			)
		}
		seen[ref] = struct{}{}
	}
	return nil
}
