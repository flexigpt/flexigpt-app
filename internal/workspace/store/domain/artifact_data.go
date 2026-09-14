package domain

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// ArtifactData is Workspace-local consumer data stored in Artifact.Data.
// It contains no source ownership, typed membership, or execution behavior.
type ArtifactData struct {
	RuntimeDisabled bool `json:"runtimeDisabled,omitempty"`
}

func EncodeArtifactData(
	value ArtifactData,
) (json.RawMessage, error) {
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func DecodeArtifactData(
	raw json.RawMessage,
) (ArtifactData, error) {
	if len(raw) == 0 {
		raw = json.RawMessage(jsonutil.EmptyObject)
	}
	var value ArtifactData
	if err := jsonutil.DecodeCanonicalObjectInto(
		raw,
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
