package modelruntime

import (
	"encoding/json"
	"fmt"
	"maps"

	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inferencewrapper/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
)

const runtimeRequestPatchDigestDomain = "flexigpt.spec.runtime-request-patch/v1"

// preparedRuntimeRequestPatch is adapter-owned normalized request data.
// Only prepareRuntimeRequestPatch constructs non-empty values.
type preparedRuntimeRequestPatch struct {
	defaults map[string]any
	clear    []inferencewrapperSpec.RuntimeDefaultField
	digest   cryptoutil.Digest
}

// prepareRuntimeRequestPatch is the sole request-patch normalization boundary.
// It validates the typed defaults, converts them to the merge representation,
// and computes their configuration digest.
func prepareRuntimeRequestPatch(
	patch *inferencewrapperSpec.RuntimeRequestPatch,
) (preparedRuntimeRequestPatch, error) {
	if patch == nil {
		return preparedRuntimeRequestPatch{}, nil
	}

	clearFields, err := normalizeRuntimeDefaultFields(patch.Clear)
	if err != nil {
		return preparedRuntimeRequestPatch{}, err
	}

	var (
		defaults          map[string]any
		canonicalDefaults []byte
	)
	if patch.Defaults != nil {
		canonical, err := jsonutil.MarshalCanonicalObject(
			patch.Defaults,
			spec.MaxDefinitionBodyBytes,
		)
		if err != nil {
			return preparedRuntimeRequestPatch{}, fmt.Errorf(
				"encode typed Model runtime request defaults: %w",
				err,
			)
		}
		if err := modelv1.ValidateDefaultsPatch(canonical); err != nil {
			return preparedRuntimeRequestPatch{}, fmt.Errorf(
				"model runtime request defaults: %w",
				err,
			)
		}
		if err := json.Unmarshal(canonical, &defaults); err != nil {
			return preparedRuntimeRequestPatch{}, fmt.Errorf(
				"decode typed Model runtime request defaults: %w",
				err,
			)
		}
		if len(defaults) > 0 {
			canonicalDefaults = canonical
		}
	}

	if len(defaults) == 0 && len(clearFields) == 0 {
		return preparedRuntimeRequestPatch{}, nil
	}

	return preparedRuntimeRequestPatch{
		defaults: defaults,
		clear:    clearFields,
		digest:   runtimeRequestPatchDigest(canonicalDefaults, clearFields),
	}, nil
}

func normalizeRuntimeDefaultFields(
	fields []inferencewrapperSpec.RuntimeDefaultField,
) ([]inferencewrapperSpec.RuntimeDefaultField, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	output := make([]inferencewrapperSpec.RuntimeDefaultField, len(fields))
	seen := make(map[inferencewrapperSpec.RuntimeDefaultField]struct{}, len(fields))
	for index, field := range fields {
		if err := field.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[field]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate Model runtime request clear field %q",
				spec.ErrInvalid,
				field,
			)
		}
		seen[field] = struct{}{}
		output[index] = field
	}
	return output, nil
}

func runtimeRequestPatchDigest(
	canonicalDefaults []byte,
	clearFields []inferencewrapperSpec.RuntimeDefaultField,
) cryptoutil.Digest {
	input := make(
		[]byte,
		0,
		len(runtimeRequestPatchDigestDomain)+len(canonicalDefaults)+len(clearFields)*32,
	)
	input = append(input, []byte(runtimeRequestPatchDigestDomain)...)
	input = append(input, 0)
	input = append(input, canonicalDefaults...)
	input = append(input, 0)
	for _, field := range clearFields {
		input = append(input, []byte(string(field))...)
		input = append(input, 0)
	}
	return cryptoutil.DigestBytes(input)
}

// apply runs after source and local-overlay defaults have been merged.
// Explicit clears take precedence over defaults in the same request patch.
func (p preparedRuntimeRequestPatch) apply(defaults map[string]any) error {
	if err := applyRuntimeDefaults(defaults, p.defaults); err != nil {
		return err
	}
	for _, field := range p.clear {
		delete(defaults, string(field))
	}
	return nil
}

// applyRuntimeDefaults merges an already-validated defaults object without
// repeating request validation or JSON conversion.
func applyRuntimeDefaults(
	defaults map[string]any,
	patch map[string]any,
) error {
	if defaults == nil {
		return fmt.Errorf(
			"%w: effective Model defaults are nil",
			spec.ErrInvalid,
		)
	}

	for key, value := range patch {
		switch key {
		case "reasoning", "cacheControl", "output":
			child, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf(
					"%w: Model defaults field %q must be an object",
					spec.ErrInvalid,
					key,
				)
			}
			existing, _ := defaults[key].(map[string]any)
			defaults[key] = mergeRuntimeDefaultObject(existing, child)

		case "stopSequences":
			values, ok := value.([]any)
			if !ok {
				return fmt.Errorf(
					"%w: Model stopSequences must be an array",
					spec.ErrInvalid,
				)
			}
			if len(values) != 0 {
				defaults[key] = value
			}

		default:
			defaults[key] = value
		}
	}
	return nil
}

func mergeRuntimeDefaultObject(
	base map[string]any,
	patch map[string]any,
) map[string]any {
	output := maps.Clone(base)
	if output == nil {
		output = map[string]any{}
	}
	maps.Copy(output, patch)
	return output
}
