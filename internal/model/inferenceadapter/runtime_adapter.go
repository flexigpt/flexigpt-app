package inferenceadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/inference-go/capabilityoverride"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/model/store/consumerapi"
)

// Credential is one resolved non-secret credential metadata/value pair.
//
// APIKey is runtime-only and must never be logged, serialized, stored in
// Artifact Data, stored in overlays, or included in fingerprints.
type Credential struct {
	APIKey  string
	Version string
}

// CredentialResolver owns opaque credential-reference resolution.
//
// Model Store stores only a CredentialRef. AgentGo composition provides the
// resolver through its encrypted settings implementation.
type CredentialResolver interface {
	ResolveModelCredential(
		ctx context.Context,
		ref string,
	) (Credential, error)
}

type AdapterDefinition struct {
	ID      string
	Version string

	SDKType          inferenceSpec.ProviderSDKType
	Origin           string
	Path             string
	APIKeyHeaderKey  string
	DefaultHeaders   map[string]string
	DefaultDefaults  map[string]any
	BaseCapabilities inferenceSpec.ModelCapabilities
}

func (d AdapterDefinition) Validate() error {
	if err := basespec.ValidateIdentifier(
		"Model adapter ID",
		d.ID,
		basespec.MaxKindBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Model adapter version",
		d.Version,
		basespec.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Model adapter origin",
		d.Origin,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Model adapter path",
		d.Path,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Model adapter API key header",
		d.APIKeyHeaderKey,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return nil
}

type RuntimeConfiguration struct {
	ProviderParam inferenceSpec.ProviderParam
	ModelParam    inferenceSpec.ModelParam
	Capabilities  inferenceSpec.ModelCapabilities
	Fingerprint   cryptoutil.Digest
}

type RuntimeAdapter struct {
	credentials CredentialResolver
	adapters    map[string]AdapterDefinition
}

func NewRuntimeAdapter(
	credentials CredentialResolver,
	definitions ...AdapterDefinition,
) (*RuntimeAdapter, error) {
	if len(definitions) == 0 {
		definitions = DefaultAdapterDefinitions()
	}

	output := &RuntimeAdapter{
		credentials: credentials,
		adapters:    make(map[string]AdapterDefinition, len(definitions)),
	}
	for index, definition := range definitions {
		if err := definition.Validate(); err != nil {
			return nil, fmt.Errorf(
				"model runtime adapter definition %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := output.adapters[definition.ID]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate Model runtime adapter %q",
				basespec.ErrConflict,
				definition.ID,
			)
		}
		definition.DefaultHeaders = maps.Clone(
			definition.DefaultHeaders,
		)
		definition.DefaultDefaults = maps.Clone(
			definition.DefaultDefaults,
		)
		output.adapters[definition.ID] = definition
	}
	return output, nil
}

func DefaultAdapterDefinitions() []AdapterDefinition {
	defaults := map[string]any{
		"stream":       true,
		"temperature":  1.0,
		"timeoutMS":    300000,
		"systemPrompt": "",
	}
	baseCapabilities := inferenceSpec.ModelCapabilities{
		ModalitiesIn: []inferenceSpec.Modality{
			inferenceSpec.ModalityTextIn,
		},
		ModalitiesOut: []inferenceSpec.Modality{
			inferenceSpec.ModalityTextOut,
		},
	}

	return []AdapterDefinition{
		{
			ID:              "anthropic.messages",
			Version:         "v1",
			SDKType:         inferenceSpec.ProviderSDKTypeAnthropic,
			Origin:          inferenceSpec.DefaultAnthropicOrigin,
			Path:            inferenceSpec.DefaultAnthropicChatCompletionPrefix,
			APIKeyHeaderKey: inferenceSpec.DefaultAnthropicAuthorizationHeaderKey,
			DefaultHeaders: map[string]string{
				inferenceSpec.DefaultContentTypeHeaderKey:      inferenceSpec.DefaultContentTypeHeader,
				inferenceSpec.DefaultAcceptHeaderKey:           inferenceSpec.DefaultContentTypeHeader,
				inferenceSpec.DefaultAnthropicVersionHeaderKey: inferenceSpec.DefaultAnthropicVersionHeader,
			},
			DefaultDefaults:  maps.Clone(defaults),
			BaseCapabilities: capabilityoverride.CloneModelCapabilities(baseCapabilities),
		},
		{
			ID:              "openai.chatCompletions",
			Version:         "v1",
			SDKType:         inferenceSpec.ProviderSDKTypeOpenAIChatCompletions,
			Origin:          inferenceSpec.DefaultOpenAIOrigin,
			Path:            inferenceSpec.DefaultOpenAIChatCompletionsPrefix,
			APIKeyHeaderKey: inferenceSpec.DefaultAuthorizationHeaderKey,
			DefaultHeaders: map[string]string{
				inferenceSpec.DefaultContentTypeHeaderKey: inferenceSpec.DefaultContentTypeHeader,
			},
			DefaultDefaults:  maps.Clone(defaults),
			BaseCapabilities: capabilityoverride.CloneModelCapabilities(baseCapabilities),
		},
		{
			ID:              "openai.responses",
			Version:         "v1",
			SDKType:         inferenceSpec.ProviderSDKTypeOpenAIResponses,
			Origin:          inferenceSpec.DefaultOpenAIOrigin,
			Path:            inferenceSpec.DefaultOpenAIResponsesPrefix,
			APIKeyHeaderKey: inferenceSpec.DefaultAuthorizationHeaderKey,
			DefaultHeaders: map[string]string{
				inferenceSpec.DefaultContentTypeHeaderKey: inferenceSpec.DefaultContentTypeHeader,
			},
			DefaultDefaults:  maps.Clone(defaults),
			BaseCapabilities: capabilityoverride.CloneModelCapabilities(baseCapabilities),
		},
		{
			ID:              "google.generateContent",
			Version:         "v1",
			SDKType:         inferenceSpec.ProviderSDKTypeGoogleGenerateContent,
			Origin:          inferenceSpec.DefaultGoogleGenerateContentOrigin,
			Path:            inferenceSpec.DefaultGoogleGenerateContentPrefix,
			APIKeyHeaderKey: inferenceSpec.DefaultGoogleGenerateContentAPIKeyHeaderKey,
			DefaultHeaders: map[string]string{
				inferenceSpec.DefaultContentTypeHeaderKey: inferenceSpec.DefaultContentTypeHeader,
			},
			DefaultDefaults:  maps.Clone(defaults),
			BaseCapabilities: capabilityoverride.CloneModelCapabilities(baseCapabilities),
		},
	}
}

func (a *RuntimeAdapter) LookupModelAdapter(
	ctx context.Context,
	adapter string,
) (modelConsumerAPI.AdapterDescriptor, bool, error) {
	if a == nil {
		return modelConsumerAPI.AdapterDescriptor{}, false, basespec.ErrClosed
	}
	if ctx == nil {
		return modelConsumerAPI.AdapterDescriptor{}, false, fmt.Errorf(
			"%w: Model adapter lookup context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return modelConsumerAPI.AdapterDescriptor{}, false, err
	}

	value, found := a.adapters[adapter]
	if !found {
		return modelConsumerAPI.AdapterDescriptor{}, false, nil
	}
	return modelConsumerAPI.AdapterDescriptor{
		ID:      value.ID,
		Version: value.Version,
	}, true, nil
}

// Resolve converts one source-backed resolved Model into ordinary inference-go
// runtime values. It does not execute a completion and it does not reach into
// Model Store itself.
func (a *RuntimeAdapter) Resolve(
	ctx context.Context,
	resolved modelConsumerAPI.ResolvedModel,
) (RuntimeConfiguration, error) {
	return a.ResolveWithRequestPatch(ctx, resolved, nil)
}

// ResolveWithRequestPatch applies the final, caller-owned portable defaults
// patch after adapter, Provider, Provider-overlay, Model, and Model-overlay
// layers. The patch cannot modify source identity or credentials.
func (a *RuntimeAdapter) ResolveWithRequestPatch(
	ctx context.Context,
	resolved modelConsumerAPI.ResolvedModel,
	requestPatchRaw json.RawMessage,
) (RuntimeConfiguration, error) {
	if a == nil {
		return RuntimeConfiguration{}, basespec.ErrClosed
	}
	if ctx == nil {
		return RuntimeConfiguration{}, fmt.Errorf(
			"%w: Model runtime resolution context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return RuntimeConfiguration{}, err
	}
	if err := cryptoutil.ValidateDigest(resolved.Fingerprint); err != nil {
		return RuntimeConfiguration{}, err
	}

	requestPatch, err := decodeRuntimeRequestPatch(requestPatchRaw)
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	definition, found := a.adapters[resolved.Provider.Document.Adapter]
	if !found {
		return RuntimeConfiguration{}, fmt.Errorf(
			"%w: Model adapter %q is not installed",
			basespec.ErrUnsupported,
			resolved.Provider.Document.Adapter,
		)
	}

	connection, err := resolveConnection(
		definition,
		resolved.Provider.Document.Connection,
		resolved.ProviderOverlay.Connection,
	)
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	authentication, err := decodeAuthentication(
		resolved.Provider.Document.Authentication,
		definition.APIKeyHeaderKey,
	)
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	var credential Credential
	if resolved.ProviderOverlay.CredentialRef != "" {
		if a.credentials == nil {
			return RuntimeConfiguration{}, fmt.Errorf(
				"%w: Model credential resolver is unavailable",
				basespec.ErrReferenceUnresolved,
			)
		}
		credential, err = a.credentials.ResolveModelCredential(
			ctx,
			resolved.ProviderOverlay.CredentialRef,
		)
		if err != nil {
			return RuntimeConfiguration{}, err
		}
		if strings.TrimSpace(credential.APIKey) == "" {
			return RuntimeConfiguration{}, fmt.Errorf(
				"%w: Model credential reference resolved to an empty value",
				basespec.ErrReferenceUnresolved,
			)
		}
	} else if authentication.Mode != "none" {
		return RuntimeConfiguration{}, fmt.Errorf(
			"%w: Model Provider %q has no configured credential",
			basespec.ErrReferenceUnresolved,
			resolved.Provider.Artifact.LogicalName,
		)
	}

	defaults, err := mergeDefaultLayers(
		definition.DefaultDefaults,
		resolved.Provider.Document.Defaults,
		resolved.ProviderOverlay.Defaults,
		resolved.Model.Document.Defaults,
		resolved.ModelOverlay.Defaults,
		requestPatch.Defaults,
	)
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	// Adapter parameters replace atomically by layer.
	for _, raw := range []json.RawMessage{
		resolved.Provider.Document.AdapterParameters,
		resolved.ProviderOverlay.AdapterParameters,
		resolved.Model.Document.AdapterParameters,
		resolved.ModelOverlay.AdapterParameters,
	} {
		if len(raw) == 0 {
			continue
		}
		value, err := decodeObjectMap(raw)
		if err != nil {
			return RuntimeConfiguration{}, err
		}
		defaults["adapterParameters"] = value
	}

	if len(requestPatch.Defaults) != 0 {
		requestDefaults, err := decodeObjectMap(requestPatch.Defaults)
		if err != nil {
			return RuntimeConfiguration{}, err
		}
		if adapterParameters, found := requestDefaults["adapterParameters"]; found {
			defaults["adapterParameters"] = adapterParameters
		}
	}

	for _, field := range requestPatch.Clear {
		delete(defaults, field)
	}

	modelParam, err := modelParamFromDefaults(
		defaults,
		resolved.Model.Document.ProviderModelID,
	)
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	capabilities, err := deriveCapabilities(
		definition.BaseCapabilities,
		resolved.Provider.Document.Capabilities,
		resolved.ProviderOverlay.Capabilities,
		resolved.Model.Document.Capabilities,
		resolved.ModelOverlay.Capabilities,
	)
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	fingerprint, err := cryptoutil.CanonicalDigest(struct {
		ResolvedFingerprint cryptoutil.Digest `json:"resolvedFingerprint"`
		AdapterVersion      string            `json:"adapterVersion"`
		CredentialVersion   string            `json:"credentialVersion,omitempty"`
		RequestPatch        cryptoutil.Digest `json:"requestPatch,omitempty"`
	}{
		ResolvedFingerprint: resolved.Fingerprint,
		AdapterVersion:      definition.Version,
		CredentialVersion:   credential.Version,
		RequestPatch:        requestPatch.Digest,
	})
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	return RuntimeConfiguration{
		ProviderParam: inferenceSpec.ProviderParam{
			Name:                     inferenceSpec.ProviderName(resolved.Provider.Artifact.LogicalName),
			SDKType:                  definition.SDKType,
			APIKey:                   credential.APIKey,
			Origin:                   connection.Origin,
			ChatCompletionPathPrefix: connection.Path,
			APIKeyHeaderKey:          authentication.HeaderName,
			DefaultHeaders:           connection.Headers,
		},
		ModelParam:   modelParam,
		Capabilities: capabilities,
		Fingerprint:  fingerprint,
	}, nil
}

type resolvedConnection struct {
	Origin  string
	Path    string
	Headers map[string]string
}

type connectionPatch struct {
	Origin  *string      `json:"origin"`
	Path    *string      `json:"path"`
	Headers *headerPatch `json:"headers"`
}

type headerPatch struct {
	Set    map[string]string `json:"set"`
	Remove []string          `json:"remove"`
}

type authentication struct {
	Mode       string  `json:"mode"`
	HeaderName string  `json:"headerName"`
	Prefix     *string `json:"prefix"`
}

func resolveConnection(
	definition AdapterDefinition,
	providerRaw json.RawMessage,
	overlayRaw json.RawMessage,
) (resolvedConnection, error) {
	output := resolvedConnection{
		Origin:  definition.Origin,
		Path:    definition.Path,
		Headers: maps.Clone(definition.DefaultHeaders),
	}

	for _, raw := range []json.RawMessage{providerRaw, overlayRaw} {
		if len(raw) == 0 {
			continue
		}
		var patch connectionPatch
		if err := decodeObject(raw, &patch); err != nil {
			return resolvedConnection{}, err
		}
		if patch.Origin != nil {
			output.Origin = *patch.Origin
		}
		if patch.Path != nil {
			output.Path = *patch.Path
		}
		if patch.Headers != nil {
			applyHeaderPatch(
				output.Headers,
				patch.Headers.Set,
				patch.Headers.Remove,
			)
		}
	}

	if strings.TrimSpace(output.Origin) == "" ||
		strings.TrimSpace(output.Path) == "" {
		return resolvedConnection{}, fmt.Errorf(
			"%w: effective Model Provider connection is incomplete",
			basespec.ErrReferenceUnresolved,
		)
	}
	if output.Headers == nil {
		output.Headers = map[string]string{}
	}
	return output, nil
}

func decodeAuthentication(
	raw json.RawMessage,
	defaultHeaderName string,
) (authentication, error) {
	output := authentication{
		Mode:       "apiKeyHeader",
		HeaderName: defaultHeaderName,
	}
	if len(raw) == 0 {
		return output, nil
	}
	if err := decodeObject(raw, &output); err != nil {
		return authentication{}, err
	}
	if output.Mode == "" {
		output.Mode = "apiKeyHeader"
	}
	if output.HeaderName == "" && output.Mode != "none" {
		output.HeaderName = defaultHeaderName
	}
	if output.Prefix != nil && *output.Prefix != "" {
		return authentication{}, fmt.Errorf(
			"%w: Model Provider authentication prefixes are not supported by inference-go ProviderParam",
			basespec.ErrUnsupported,
		)
	}
	return output, nil
}

func applyHeaderPatch(
	headers map[string]string,
	set map[string]string,
	remove []string,
) {
	if headers == nil {
		return
	}

	for _, name := range remove {
		deleteHeaderFold(headers, name)
	}
	for name, value := range set {
		deleteHeaderFold(headers, name)
		headers[name] = value
	}
}

func deleteHeaderFold(
	headers map[string]string,
	name string,
) {
	for current := range headers {
		if strings.EqualFold(current, name) {
			delete(headers, current)
		}
	}
}

func mergeDefaultLayers(
	base map[string]any,
	layers ...json.RawMessage,
) (map[string]any, error) {
	output, err := cloneObjectMap(base)
	if err != nil {
		return nil, err
	}

	for _, raw := range layers {
		if len(raw) == 0 {
			continue
		}
		next, err := decodeObjectMap(raw)
		if err != nil {
			return nil, err
		}
		for key, value := range next {
			switch key {
			case "reasoning", "cacheControl", "output":
				child, ok := value.(map[string]any)
				if !ok {
					return nil, fmt.Errorf(
						"%w: Model defaults field %q must be an object",
						basespec.ErrInvalid,
						key,
					)
				}
				existing, _ := output[key].(map[string]any)
				merged, err := mergeObjectMaps(existing, child)
				if err != nil {
					return nil, err
				}
				output[key] = merged

			case "stopSequences":
				values, ok := value.([]any)
				if !ok {
					return nil, fmt.Errorf(
						"%w: Model stopSequences must be an array",
						basespec.ErrInvalid,
					)
				}
				// The v1 declaration rule treats omitted and [] identically.
				// An empty list does not clear inherited stop sequences.
				if len(values) != 0 {
					output[key] = value
				}

			default:
				output[key] = value
			}
		}
	}
	return output, nil
}

func mergeObjectMaps(
	base map[string]any,
	next map[string]any,
) (map[string]any, error) {
	output, err := cloneObjectMap(base)
	if err != nil {
		return nil, err
	}
	maps.Copy(output, next)
	return output, nil
}

func cloneObjectMap(
	value map[string]any,
) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, err
	}
	return output, nil
}

func decodeObjectMap(
	raw json.RawMessage,
) (map[string]any, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, err
	}

	var output map[string]any
	if err := json.Unmarshal(canonical, &output); err != nil {
		return nil, err
	}
	return output, nil
}

func decodeObject(
	raw json.RawMessage,
	target any,
) error {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return err
	}
	return json.Unmarshal(canonical, target)
}

type runtimeRequestPatch struct {
	Defaults  json.RawMessage   `json:"defaults,omitempty"`
	Clear     []string          `json:"clear,omitempty"`
	Canonical json.RawMessage   `json:"-"`
	Digest    cryptoutil.Digest `json:"-"`
}

func decodeRuntimeRequestPatch(
	raw json.RawMessage,
) (runtimeRequestPatch, error) {
	if len(raw) == 0 {
		return runtimeRequestPatch{}, nil
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return runtimeRequestPatch{}, fmt.Errorf(
			"model runtime request patch: %w",
			err,
		)
	}

	var output runtimeRequestPatch
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		canonical,
		&output,
		basespec.MaxDefinitionBodyBytes,
	); err != nil {
		return runtimeRequestPatch{}, fmt.Errorf(
			"%w: decode Model runtime request patch: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if err := modelv1.ValidateDefaultsPatch(output.Defaults); err != nil {
		return runtimeRequestPatch{}, err
	}

	allowedClear := map[string]struct{}{
		"adapterParameters": {},
		"cacheControl":      {},
		"output":            {},
		"reasoning":         {},
		"stopSequences":     {},
		"temperature":       {},
	}
	seen := make(map[string]struct{}, len(output.Clear))
	for _, field := range output.Clear {
		if _, allowed := allowedClear[field]; !allowed {
			return runtimeRequestPatch{}, fmt.Errorf(
				"%w: unsupported Model runtime request clear field %q",
				basespec.ErrInvalid,
				field,
			)
		}
		if _, duplicate := seen[field]; duplicate {
			return runtimeRequestPatch{}, fmt.Errorf(
				"%w: duplicate Model runtime request clear field %q",
				basespec.ErrInvalid,
				field,
			)
		}
		seen[field] = struct{}{}
	}

	output.Canonical = canonical
	output.Digest = cryptoutil.DigestBytes(canonical)
	return output, nil
}

type defaultsWire struct {
	Stream            bool                          `json:"stream"`
	MaxPromptTokens   int                           `json:"maxPromptTokens"`
	MaxOutputTokens   int                           `json:"maxOutputTokens"`
	Temperature       *float64                      `json:"temperature"`
	SystemPrompt      string                        `json:"systemPrompt"`
	TimeoutMS         int                           `json:"timeoutMS"`
	Reasoning         *inferenceSpec.ReasoningParam `json:"reasoning"`
	CacheControl      *inferenceSpec.CacheControl   `json:"cacheControl"`
	Output            *outputWire                   `json:"output"`
	StopSequences     []string                      `json:"stopSequences"`
	AdapterParameters json.RawMessage               `json:"adapterParameters"`
}

type outputWire struct {
	Verbosity *inferenceSpec.OutputVerbosity `json:"verbosity"`
	Format    *outputFormatWire              `json:"format"`
}

type outputFormatWire struct {
	Kind       inferenceSpec.OutputFormatKind `json:"kind"`
	JSONSchema *jsonSchemaWire                `json:"jsonSchema"`
}

type jsonSchemaWire struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
	Strict      bool            `json:"strict"`
}

func modelParamFromDefaults(
	values map[string]any,
	modelName string,
) (inferenceSpec.ModelParam, error) {
	raw, err := json.Marshal(values)
	if err != nil {
		return inferenceSpec.ModelParam{}, err
	}

	var wire defaultsWire
	if err := json.Unmarshal(raw, &wire); err != nil {
		return inferenceSpec.ModelParam{}, err
	}
	if modelName == "" {
		return inferenceSpec.ModelParam{}, fmt.Errorf(
			"%w: Model providerModelID is empty",
			basespec.ErrInvalid,
		)
	}
	if wire.TimeoutMS < 0 {
		return inferenceSpec.ModelParam{}, fmt.Errorf(
			"%w: Model timeoutMS cannot be negative",
			basespec.ErrInvalid,
		)
	}

	output := inferenceSpec.ModelParam{
		Name:            inferenceSpec.ModelName(modelName),
		Stream:          wire.Stream,
		MaxPromptLength: wire.MaxPromptTokens,
		MaxOutputLength: wire.MaxOutputTokens,
		Temperature:     wire.Temperature,
		Reasoning:       wire.Reasoning,
		SystemPrompt:    wire.SystemPrompt,
		Timeout:         timeoutSeconds(wire.TimeoutMS),
		CacheControl:    wire.CacheControl,
		StopSequences:   append([]string(nil), wire.StopSequences...),
	}

	if wire.Output != nil {
		value := &inferenceSpec.OutputParam{
			Verbosity: wire.Output.Verbosity,
		}
		if wire.Output.Format != nil {
			format := &inferenceSpec.OutputFormat{
				Kind: wire.Output.Format.Kind,
			}
			if wire.Output.Format.JSONSchema != nil {
				var schema map[string]any
				if len(wire.Output.Format.JSONSchema.Schema) != 0 {
					if err := json.Unmarshal(
						wire.Output.Format.JSONSchema.Schema,
						&schema,
					); err != nil {
						return inferenceSpec.ModelParam{}, fmt.Errorf(
							"decode Model JSON schema output configuration: %w",
							err,
						)
					}
				}
				format.JSONSchemaParam = &inferenceSpec.JSONSchemaParam{
					Name:        wire.Output.Format.JSONSchema.Name,
					Description: wire.Output.Format.JSONSchema.Description,
					Schema:      schema,
					Strict:      wire.Output.Format.JSONSchema.Strict,
				}
			}
			value.Format = format
		}
		output.OutputParam = value
	}

	if len(wire.AdapterParameters) != 0 {
		canonical, err := jsonutil.CanonicalizeObject(
			wire.AdapterParameters,
			basespec.MaxDefinitionBodyBytes,
		)
		if err != nil {
			return inferenceSpec.ModelParam{}, err
		}
		text := string(canonical)
		output.AdditionalParametersRawJSON = &text
	}

	return output, nil
}

func timeoutSeconds(milliseconds int) int {
	if milliseconds <= 0 {
		return 0
	}
	return (milliseconds + 999) / 1000
}

func deriveCapabilities(
	base inferenceSpec.ModelCapabilities,
	raws ...json.RawMessage,
) (inferenceSpec.ModelCapabilities, error) {
	overrides := make(
		[]*capabilityoverride.ModelCapabilitiesOverride,
		0,
		len(raws),
	)
	for _, raw := range raws {
		if len(raw) == 0 {
			continue
		}
		canonical, err := jsonutil.CanonicalizeObject(
			raw,
			basespec.MaxDefinitionBodyBytes,
		)
		if err != nil {
			return inferenceSpec.ModelCapabilities{}, err
		}
		var value capabilityoverride.ModelCapabilitiesOverride
		if err := json.Unmarshal(canonical, &value); err != nil {
			return inferenceSpec.ModelCapabilities{}, err
		}
		if err := capabilityoverride.ValidateModelCapabilitiesOverride(
			&value,
		); err != nil {
			return inferenceSpec.ModelCapabilities{}, err
		}
		overrides = append(overrides, &value)
	}
	return capabilityoverride.DeriveModelCapabilities(base, overrides...), nil
}
