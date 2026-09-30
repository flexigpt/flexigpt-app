package aggregate

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/inference-go/capabilityoverride"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/model/store/consumerapi"
)

const runtimeRequestPatchDigestDomain = "flexigpt.model.runtime-request-patch/v1"

// RuntimeConfiguration is the fully resolved inference runtime configuration
// for one source-backed Model.
type RuntimeConfiguration struct {
	ProviderParam       inferenceSpec.ProviderParam
	ModelParam          inferenceSpec.ModelParam
	CapabilityOverrides []*capabilityoverride.ModelCapabilitiesOverride
	Fingerprint         cryptoutil.Digest
}

// RuntimeResolver is the runtime boundary owned by Model Aggregate.
//
// The implementation receives an already validated and normalized request
// patch. It must not decode, canonicalize, or validate caller request data.
type RuntimeResolver interface {
	ResolveRuntime(
		ctx context.Context,
		resolved modelConsumerAPI.ResolvedModel,
		requestPatch PreparedRuntimeRequestPatch,
	) (RuntimeConfiguration, error)
}

// RuntimeModelRequest identifies the Model to resolve and the final portable
// request-level runtime patch.
type RuntimeModelRequest struct {
	Model        artifact.ArtifactRef `json:"model"`
	RequestPatch *RuntimeRequestPatch `json:"requestPatch,omitempty"`
}

func (r RuntimeModelRequest) Validate() error {
	return r.Model.Validate()
}

// RuntimeRequestPatch is the final caller-owned portable defaults layer.
//
// It is intentionally a typed API value. Callers must pass an object through
// Wails or Go, never an encoded JSON fragment.
// A nil *RuntimeRequestPatch means there is no request-level runtime override.
type RuntimeRequestPatch struct {
	Defaults *RuntimeDefaultsPatch `json:"defaults,omitempty"`
	Clear    []RuntimeDefaultField `json:"clear,omitempty"`
}

// RuntimeDefaultsPatch contains only portable Model defaults. Every scalar is
// a pointer so omitted values are distinct from explicit zero values.
type RuntimeDefaultsPatch struct {
	Stream            *bool                         `json:"stream,omitempty"`
	MaxPromptTokens   *int                          `json:"maxPromptTokens,omitempty"`
	MaxOutputTokens   *int                          `json:"maxOutputTokens,omitempty"`
	Temperature       *float64                      `json:"temperature,omitempty"`
	SystemPrompt      *string                       `json:"systemPrompt,omitempty"`
	TimeoutMS         *int                          `json:"timeoutMS,omitempty"`
	Reasoning         *inferenceSpec.ReasoningParam `json:"reasoning,omitempty"`
	CacheControl      *inferenceSpec.CacheControl   `json:"cacheControl,omitempty"`
	Output            *RuntimeOutputPatch           `json:"output,omitempty"`
	StopSequences     []string                      `json:"stopSequences,omitempty"`
	AdapterParameters *RuntimeAdapterParameters     `json:"adapterParameters,omitempty"`
}

// RuntimeAdapterParameters is an object because adapter-specific parameter
// names and values are owned by the selected inference adapter.
type RuntimeAdapterParameters map[string]any

type RuntimeOutputPatch struct {
	Verbosity *inferenceSpec.OutputVerbosity `json:"verbosity,omitempty"`
	Format    *RuntimeOutputFormatPatch      `json:"format,omitempty"`
}

type RuntimeOutputFormatPatch struct {
	Kind       *inferenceSpec.OutputFormatKind `json:"kind,omitempty"`
	JSONSchema *RuntimeJSONSchemaPatch         `json:"jsonSchema,omitempty"`
}

type RuntimeJSONSchemaPatch struct {
	Name        *string            `json:"name,omitempty"`
	Description *string            `json:"description,omitempty"`
	Schema      *RuntimeJSONSchema `json:"schema,omitempty"`
	Strict      *bool              `json:"strict,omitempty"`
}

// RuntimeJSONSchema remains an object because JSON Schema itself is an open
// extensibility format. It is still an object value, never encoded JSON text.
type RuntimeJSONSchema map[string]any

type RuntimeDefaultField string

const (
	RuntimeDefaultFieldAdapterParameters RuntimeDefaultField = "adapterParameters"
	RuntimeDefaultFieldCacheControl      RuntimeDefaultField = "cacheControl"
	RuntimeDefaultFieldOutput            RuntimeDefaultField = "output"
	RuntimeDefaultFieldReasoning         RuntimeDefaultField = "reasoning"
	RuntimeDefaultFieldStopSequences     RuntimeDefaultField = "stopSequences"
	RuntimeDefaultFieldTemperature       RuntimeDefaultField = "temperature"
)

func (f RuntimeDefaultField) Validate() error {
	switch f {
	case RuntimeDefaultFieldAdapterParameters,
		RuntimeDefaultFieldCacheControl,
		RuntimeDefaultFieldOutput,
		RuntimeDefaultFieldReasoning,
		RuntimeDefaultFieldStopSequences,
		RuntimeDefaultFieldTemperature:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported Model runtime request clear field %q",
			basespec.ErrInvalid,
			f,
		)
	}
}

// PreparedRuntimeRequestPatch is an immutable aggregate-owned runtime patch.
// Its fields are deliberately private so only Prepare can create a non-empty
// validated patch.
type PreparedRuntimeRequestPatch struct {
	defaults map[string]any
	clear    []RuntimeDefaultField
	digest   cryptoutil.Digest
}

func (p PreparedRuntimeRequestPatch) Digest() cryptoutil.Digest {
	return p.digest
}

// Prepare is the only request-patch normalization boundary. The typed value is
// validated once, converted once into the object form needed by declaration
// layer merging, and assigned a stable configuration digest.
//
// A nil receiver is an intentional no-op patch.
func (p *RuntimeRequestPatch) Prepare() (
	PreparedRuntimeRequestPatch,
	error,
) {
	if p == nil {
		return PreparedRuntimeRequestPatch{}, nil
	}

	c, err := normalizeRuntimeDefaultFields(p.Clear)
	if err != nil {
		return PreparedRuntimeRequestPatch{}, err
	}

	var (
		defaults          map[string]any
		canonicalDefaults []byte
	)
	if p.Defaults != nil {
		canonical, err := jsonutil.MarshalCanonicalObject(
			p.Defaults,
			basespec.MaxDefinitionBodyBytes,
		)
		if err != nil {
			return PreparedRuntimeRequestPatch{}, fmt.Errorf(
				"encode typed Model runtime request defaults: %w",
				err,
			)
		}
		if err := modelv1.ValidateDefaultsPatch(canonical); err != nil {
			return PreparedRuntimeRequestPatch{}, fmt.Errorf(
				"model runtime request defaults: %w",
				err,
			)
		}
		if err := json.Unmarshal(canonical, &defaults); err != nil {
			return PreparedRuntimeRequestPatch{}, fmt.Errorf(
				"decode typed Model runtime request defaults: %w",
				err,
			)
		}
		if len(defaults) > 0 {
			canonicalDefaults = append([]byte(nil), canonical...)
		}
	}

	if len(defaults) == 0 && len(c) == 0 {
		return PreparedRuntimeRequestPatch{}, nil
	}

	return PreparedRuntimeRequestPatch{
		defaults: defaults,
		clear:    c,
		digest:   runtimeRequestPatchDigest(canonicalDefaults, c),
	}, nil
}

func normalizeRuntimeDefaultFields(
	fields []RuntimeDefaultField,
) ([]RuntimeDefaultField, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	output := make([]RuntimeDefaultField, len(fields))
	seen := make(map[RuntimeDefaultField]struct{}, len(fields))
	for index, field := range fields {
		if err := field.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[field]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate Model runtime request clear field %q",
				basespec.ErrInvalid,
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
	c []RuntimeDefaultField,
) cryptoutil.Digest {
	input := make(
		[]byte,
		0,
		len(runtimeRequestPatchDigestDomain)+len(canonicalDefaults)+len(c)*32,
	)
	input = append(input, []byte(runtimeRequestPatchDigestDomain)...)
	input = append(input, 0)
	input = append(input, canonicalDefaults...)
	input = append(input, 0)
	for _, field := range c {
		input = append(input, []byte(string(field))...)
		input = append(input, 0)
	}
	return cryptoutil.DigestBytes(input)
}

// Apply applies this prepared patch after all source and local-overlay default
// layers have been merged.
func (p PreparedRuntimeRequestPatch) Apply(
	defaults map[string]any,
) error {
	if defaults == nil {
		return fmt.Errorf(
			"%w: effective Model defaults are nil",
			basespec.ErrInvalid,
		)
	}
	if err := ApplyRuntimeDefaults(defaults, p.defaults); err != nil {
		return err
	}
	for _, field := range p.clear {
		delete(defaults, string(field))
	}
	return nil
}

// ApplyRuntimeDefaults applies one already-validated defaults object without
// further JSON decoding, canonicalization, or request validation.
func ApplyRuntimeDefaults(
	defaults map[string]any,
	patch map[string]any,
) error {
	if defaults == nil {
		return fmt.Errorf(
			"%w: effective Model defaults are nil",
			basespec.ErrInvalid,
		)
	}

	for key, value := range patch {
		switch key {
		case "reasoning", "cacheControl", "output":
			child, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf(
					"%w: Model defaults field %q must be an object",
					basespec.ErrInvalid,
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
					basespec.ErrInvalid,
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
