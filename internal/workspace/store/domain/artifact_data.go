package domain

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const artifactDataNamespace = "flexigpt.dev/workspace"

// ArtifactData is Workspace-local consumer data stored in Artifact.Data.
// It contains no source ownership, typed membership, or execution behavior.
type ArtifactData struct {
	RuntimeDisabled bool `json:"runtimeDisabled,omitempty"`
}

// EncodeArtifactData creates the Workspace-owned namespaced Artifact.Data
// representation. Existing callers that need to preserve other consumer data
// should use MergeArtifactData.
func EncodeArtifactData(
	value ArtifactData,
) (json.RawMessage, error) {
	return MergeArtifactData(
		json.RawMessage(jsonutil.EmptyObject),
		value,
	)
}

// MergeArtifactData updates only Workspace-owned local state and preserves
// data owned by MCP or future Artifact consumers.
func MergeArtifactData(
	raw json.RawMessage,
	value ArtifactData,
) (json.RawMessage, error) {
	fields, err := artifact.DecodeDataObject(raw)
	if err != nil {
		return nil, err
	}
	payload, err := jsonutil.MarshalCanonicalObject(
		value,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	fields[artifactDataNamespace] = payload
	delete(fields, "runtimeDisabled")
	return artifact.EncodeDataObject(fields)
}

func DecodeArtifactData(
	raw json.RawMessage,
) (ArtifactData, error) {
	fields, err := artifact.DecodeDataObject(raw)
	if err != nil {
		return ArtifactData{}, fmt.Errorf(
			"%w: decode Workspace Artifact data: %w",
			ErrInvalidWorkspace,
			err,
		)
	}
	if payload, found := fields[artifactDataNamespace]; found {
		var value ArtifactData
		if err := jsonutil.DecodeCanonicalObjectBytesInto(
			payload,
			&value,
			basespec.MaxLocalDataBytes,
		); err != nil {
			return ArtifactData{}, fmt.Errorf(
				"%w: decode Workspace Artifact data: %w",
				ErrInvalidWorkspace,
				err,
			)
		}
		return value, nil
	}
	legacy, found := fields["runtimeDisabled"]
	if !found {
		return ArtifactData{}, nil
	}
	var runtimeDisabled bool
	if err := json.Unmarshal(legacy, &runtimeDisabled); err != nil {
		return ArtifactData{}, fmt.Errorf(
			"%w: decode Workspace Artifact data: %w",
			ErrInvalidWorkspace,
			err,
		)
	}
	return ArtifactData{RuntimeDisabled: runtimeDisabled}, nil
}
