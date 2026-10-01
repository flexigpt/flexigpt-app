package inferenceadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/flexigpt/inference-go/capabilityoverride"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/secret"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelAggregate "github.com/flexigpt/flexigpt-app/internal/model/aggregate"
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

// CredentialResolver owns Artifact Store secret-binding resolution.
//
// Model Store resolves a Provider credential binding through Artifact Store.
// The resolver receives public-safe binding metadata and returns a runtime-only
// plaintext credential.
type CredentialResolver interface {
	ResolveModelCredential(
		ctx context.Context,
		binding secret.Binding,
	) (Credential, error)
}

type AdapterDefinition struct {
	ID      string
	Version string

	SDKType         inferenceSpec.ProviderSDKType
	Origin          string
	Path            string
	APIKeyHeaderKey string
	DefaultHeaders  map[string]string
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

// RuntimeConfiguration remains an alias for callers that use the runtime
// adapter directly. Model Aggregate owns the runtime result contract.
type RuntimeConfiguration = modelAggregate.RuntimeConfiguration

type RuntimeAdapter struct {
	credentials CredentialResolver
	adapters    map[string]AdapterDefinition
}

var _ modelAggregate.RuntimeResolver = (*RuntimeAdapter)(nil)

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

		output.adapters[definition.ID] = definition
	}
	return output, nil
}

func DefaultAdapterDefinitions() []AdapterDefinition {
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
	return a.ResolveRuntime(
		ctx,
		resolved,
		modelAggregate.PreparedRuntimeRequestPatch{},
	)
}

// ResolveWithRequestPatch is the direct-adapter convenience entry point. Model
// Aggregate normally prepares the typed optional patch and calls ResolveRuntime.
func (a *RuntimeAdapter) ResolveWithRequestPatch(
	ctx context.Context,
	resolved modelConsumerAPI.ResolvedModel,
	requestPatch *modelAggregate.RuntimeRequestPatch,
) (RuntimeConfiguration, error) {
	prepared, err := requestPatch.Prepare()
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	return a.ResolveRuntime(ctx, resolved, prepared)
}

// ResolveProviderRuntime resolves only the Provider portion of an installed
// Provider. It intentionally allows a missing API key so provider lifecycle
// propagation can register endpoint/header changes before a key is configured.
func (a *RuntimeAdapter) ResolveProviderRuntime(
	ctx context.Context,
	resolved modelConsumerAPI.ResolvedProvider,
) (inferenceSpec.ProviderParam, error) {
	if a == nil {
		return inferenceSpec.ProviderParam{}, basespec.ErrClosed
	}
	if ctx == nil {
		return inferenceSpec.ProviderParam{}, fmt.Errorf(
			"%w: Model Provider runtime resolution context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return inferenceSpec.ProviderParam{}, err
	}

	value, err := a.resolveProviderRuntime(ctx, resolved, false)
	if err != nil {
		return inferenceSpec.ProviderParam{}, err
	}
	return value.param, nil
}

// ResolveRuntime applies an aggregate-prepared portable defaults patch after
// adapter, Provider, Provider-overlay, Model, and Model-overlay layers.
func (a *RuntimeAdapter) ResolveRuntime(
	ctx context.Context,
	resolved modelConsumerAPI.ResolvedModel,
	requestPatch modelAggregate.PreparedRuntimeRequestPatch,
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

	providerRuntime, err := a.resolveProviderRuntime(
		ctx,
		modelConsumerAPI.ResolvedProvider{
			Provider:           resolved.Provider,
			ProviderOverlay:    resolved.ProviderOverlay,
			ProviderCredential: resolved.ProviderCredential,
			Adapter:            resolved.Adapter,
		},
		true,
	)
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	defaults, err := mergeDefaultLayers(
		resolved.Provider.Document.Defaults,
		resolved.ProviderOverlay.Defaults,
		resolved.Model.Document.Defaults,
		resolved.ModelOverlay.Defaults,
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

	if err := requestPatch.Apply(defaults); err != nil {
		return RuntimeConfiguration{}, err
	}

	modelParam, err := modelParamFromDefaults(
		defaults,
		resolved.Model.Document.ProviderModelID,
	)
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	capabilityOverrides, err := collectCapabilityOverrides(
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
		AdapterVersion:      providerRuntime.adapterVersion,
		CredentialVersion:   providerRuntime.credentialVersion,
		RequestPatch:        requestPatch.Digest(),
	})
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	return RuntimeConfiguration{
		ProviderParam:       providerRuntime.param,
		ModelParam:          modelParam,
		CapabilityOverrides: capabilityOverrides,
		Fingerprint:         fingerprint,
	}, nil
}

type providerRuntimeConfiguration struct {
	param             inferenceSpec.ProviderParam
	adapterVersion    string
	credentialVersion string
}

func (a *RuntimeAdapter) resolveProviderRuntime(
	ctx context.Context,
	resolved modelConsumerAPI.ResolvedProvider,
	requireCredential bool,
) (providerRuntimeConfiguration, error) {
	definition, found := a.adapters[resolved.Provider.Document.Adapter]
	if !found {
		return providerRuntimeConfiguration{}, fmt.Errorf(
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
		return providerRuntimeConfiguration{}, err
	}
	authentication, err := decodeAuthentication(
		resolved.Provider.Document.Authentication,
		definition.APIKeyHeaderKey,
	)
	if err != nil {
		return providerRuntimeConfiguration{}, err
	}

	var credential Credential
	if resolved.ProviderCredential != nil &&
		resolved.ProviderCredential.Active() {
		if a.credentials == nil {
			return providerRuntimeConfiguration{}, fmt.Errorf(
				"%w: Model credential resolver is unavailable",
				basespec.ErrReferenceUnresolved,
			)
		}
		credential, err = a.credentials.ResolveModelCredential(
			ctx,
			resolved.ProviderCredential.Clone(),
		)
		if err != nil {
			return providerRuntimeConfiguration{}, err
		}
		if credential.APIKey == "" {
			return providerRuntimeConfiguration{}, fmt.Errorf(
				"%w: Model credential binding resolved to an empty value",
				basespec.ErrReferenceUnresolved,
			)
		}
	} else if requireCredential && authentication.Mode != "none" {
		return providerRuntimeConfiguration{}, fmt.Errorf(
			"%w: Model Provider %q has no configured credential",
			basespec.ErrReferenceUnresolved,
			resolved.Provider.Artifact.LogicalName,
		)
	}

	return providerRuntimeConfiguration{
		param: inferenceSpec.ProviderParam{
			Name: inferenceSpec.ProviderName(
				resolved.Provider.Artifact.LogicalName,
			),
			SDKType:                  definition.SDKType,
			APIKey:                   credential.APIKey,
			Origin:                   connection.Origin,
			ChatCompletionPathPrefix: connection.Path,
			APIKeyHeaderKey:          authentication.HeaderName,
			DefaultHeaders:           connection.Headers,
		},
		adapterVersion:    definition.Version,
		credentialVersion: credential.Version,
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
	layers ...json.RawMessage,
) (map[string]any, error) {
	output := map[string]any{}

	for _, raw := range layers {
		if len(raw) == 0 {
			continue
		}
		next, err := decodeObjectMap(raw)
		if err != nil {
			return nil, err
		}
		if err := modelAggregate.ApplyRuntimeDefaults(output, next); err != nil {
			return nil, err
		}
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

func collectCapabilityOverrides(
	raws ...json.RawMessage,
) ([]*capabilityoverride.ModelCapabilitiesOverride, error) {
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
			return nil, err
		}
		var value capabilityoverride.ModelCapabilitiesOverride
		if err := json.Unmarshal(canonical, &value); err != nil {
			return nil, err
		}
		if err := capabilityoverride.ValidateModelCapabilitiesOverride(
			&value,
		); err != nil {
			return nil, err
		}
		overrides = append(overrides, &value)
	}
	return overrides, nil
}
