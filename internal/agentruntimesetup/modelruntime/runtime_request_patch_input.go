package modelruntime

import (
	"bytes"
	"encoding/json"
	"fmt"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
)

// runtimeRequestPatchJSON is the shared wire shape used by runtime-facing
// callers. DefaultsPatch is embedded so callers use the established model
// default field names, while clear remains Runtime-owned request metadata.
type runtimeRequestPatchJSON struct {
	modelDomain.DefaultsPatch

	Clear []RuntimeDefaultField `json:"clear,omitempty"`
}

// RuntimeRequestPatchFromJSON adapts a portable request-layer model patch
// without making Session know model-domain implementation types.
//
// Validation and digest construction remain in prepareRuntimeRequestPatch.
// This function deliberately performs no duplicate semantic validation.
func RuntimeRequestPatchFromJSON(
	input json.RawMessage,
) (*RuntimeRequestPatch, error) {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		//nolint:nilnil // Ok.
		return nil, nil
	}

	canonical, err := jsonutil.CanonicalizeObject(
		trimmed,
		storeSpec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("canonicalize Model runtime request patch: %w", err)
	}
	if string(canonical) == "{}" {
		//nolint:nilnil // Ok.
		return nil, nil
	}

	value, err := jsonutil.DecodeJSONRaw[runtimeRequestPatchJSON](
		json.RawMessage(canonical),
	)
	if err != nil {
		return nil, fmt.Errorf("decode Model runtime request patch: %w", err)
	}

	return &RuntimeRequestPatch{
		Defaults: &value.DefaultsPatch,
		Clear:    value.Clear,
	}, nil
}
