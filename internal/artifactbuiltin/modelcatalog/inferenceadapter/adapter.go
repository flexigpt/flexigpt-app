package inferenceadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/flexigpt/inference-go/capabilityoverride"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/modelcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	modelv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
	modelproviderv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/modelprovider/contract/v1"
)

const (
	apiKeyHeaderStr = "apiKeyHeader"

	adapterAnthropicMessages     = "anthropic.messages"
	adapterOpenAIChatCompletions = "openai.chatCompletions"
	adapterOpenAIResponses       = "openai.responses"
	adapterGoogleGenerateContent = "google.generateContent"
)

type preparedModel struct {
	presetID modelpreset.ModelPresetID
	name     spec.LogicalName
	preset   modelpreset.ModelPreset
}

// PreparePackages converts the immutable inference-go default catalog into
// independent source-backed Model Provider and Model packages.
func PreparePackages(
	ctx context.Context,
	registry *coreinterpretation.Registry,
) ([]modelcatalog.PreparedPackage, error) {
	if registry == nil {
		return nil, fmt.Errorf(
			"%w: Model catalog interpretation registry is nil",
			spec.ErrInvalid,
		)
	}

	catalog := modelpreset.DefaultCatalog()
	if err := modelpreset.ValidateCatalog(catalog); err != nil {
		return nil, fmt.Errorf(
			"validate inference-go default Model catalog: %w",
			err,
		)
	}
	if err := validatePolicy(catalog); err != nil {
		return nil, err
	}

	providerNames := make(
		[]inferenceSpec.ProviderName,
		0,
		len(catalog.Providers),
	)
	for name := range catalog.Providers {
		providerNames = append(providerNames, name)
	}
	slices.Sort(providerNames)

	modelsByProvider := make(
		map[inferenceSpec.ProviderName][]preparedModel,
		len(catalog.Providers),
	)
	seenLogicalNames := make(map[spec.LogicalName]string)

	for _, providerName := range providerNames {
		provider := catalog.Providers[providerName]
		logicalProviderName := spec.LogicalName(providerName)
		if err := logicalProviderName.Validate(); err != nil {
			return nil, fmt.Errorf(
				"generated Provider name %q: %w",
				providerName,
				err,
			)
		}

		modelIDs := make(
			[]modelpreset.ModelPresetID,
			0,
			len(provider.ModelPresets),
		)
		for modelID := range provider.ModelPresets {
			modelIDs = append(modelIDs, modelID)
		}
		slices.Sort(modelIDs)

		prepared := make([]preparedModel, 0, len(modelIDs))
		for _, modelID := range modelIDs {
			preset := provider.ModelPresets[modelID]
			name, err := generatedModelLogicalName(
				logicalProviderName,
				string(preset.Name),
			)
			if err != nil {
				return nil, fmt.Errorf(
					"generated Model name for %q/%q: %w",
					providerName,
					modelID,
					err,
				)
			}
			if previous, duplicate := seenLogicalNames[name]; duplicate {
				return nil, fmt.Errorf(
					"%w: generated Model name %q collides between %s and %s/%s",
					spec.ErrConflict,
					name,
					previous,
					providerName,
					modelID,
				)
			}
			seenLogicalNames[name] = string(providerName) +
				"/" +
				string(modelID)
			prepared = append(prepared, preparedModel{
				presetID: modelID,
				name:     name,
				preset:   preset,
			})
		}
		modelsByProvider[providerName] = prepared
	}

	output := make([]modelcatalog.PreparedPackage, 0)
	for _, providerName := range providerNames {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		provider := catalog.Providers[providerName]
		logicalProviderName := spec.LogicalName(providerName)
		models := modelsByProvider[providerName]

		defaultPresetID := defaultModelByProvider[providerName]
		defaultModelName, found := generatedNameForPreset(
			models,
			defaultPresetID,
		)
		if !found {
			return nil, fmt.Errorf(
				"%w: Provider %q default preset %q has no generated Model",
				spec.ErrInvalid,
				providerName,
				defaultPresetID,
			)
		}

		providerDocument, err := providerDocumentFromInference(
			provider,
			defaultModelName,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"convert Provider %q: %w",
				providerName,
				err,
			)
		}
		preparedProvider, err := modelcatalog.PrepareProviderPackage(
			ctx,
			providerDocument,
			registry,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, preparedProvider)

		for _, model := range models {
			document, err := modelDocumentFromInference(
				logicalProviderName,
				model.name,
				model.preset,
				modelDisabled(providerName, model.presetID),
			)
			if err != nil {
				return nil, fmt.Errorf(
					"convert Model %q/%q: %w",
					providerName,
					model.presetID,
					err,
				)
			}
			preparedModel, err := modelcatalog.PrepareModelPackage(
				ctx,
				document,
				registry,
			)
			if err != nil {
				return nil, err
			}
			output = append(output, preparedModel)
		}
	}

	return modelcatalog.NormalizePreparedPackages(output)
}

func providerDocumentFromInference(
	provider modelpreset.ProviderPreset,
	defaultModelName spec.LogicalName,
) (modelproviderv1.ProviderDocument, error) {
	adapter, err := adapterIDForSDKType(provider.SDKType)
	if err != nil {
		return modelproviderv1.ProviderDocument{}, err
	}

	headers := maps.Clone(provider.DefaultHeaders)
	for key, value := range staticProviderHeaderOverlays[provider.Name] {
		if headers == nil {
			headers = map[string]string{}
		}
		headers[key] = value
	}

	connection, err := canonicalConnection(
		provider.Origin,
		provider.ChatCompletionPathPrefix,
		headers,
	)
	if err != nil {
		return modelproviderv1.ProviderDocument{}, err
	}
	authentication, err := canonicalObject(struct {
		Mode       string `json:"mode"`
		HeaderName string `json:"headerName"`
	}{
		Mode:       apiKeyHeaderStr,
		HeaderName: provider.APIKeyHeaderKey,
	})
	if err != nil {
		return modelproviderv1.ProviderDocument{}, err
	}
	capabilities, err := canonicalCapabilities(
		provider.CapabilitiesOverride,
	)
	if err != nil {
		return modelproviderv1.ProviderDocument{}, err
	}

	defaultModel := declaration.ArtifactNameReference{
		Name:  defaultModelName,
		Scope: declaration.LookupScopeBuiltin,
	}

	return modelproviderv1.ProviderDocument{
		Type:           modelproviderv1.ModelProviderType,
		Name:           string(provider.Name),
		DisplayName:    provider.DisplayName,
		Adapter:        adapter,
		Connection:     connection,
		Authentication: authentication,
		DefaultModel:   &defaultModel,
		Capabilities:   capabilities,
	}, nil
}

func modelDocumentFromInference(
	providerName spec.LogicalName,
	modelName spec.LogicalName,
	preset modelpreset.ModelPreset,
	disabled bool,
) (modelv1.ModelDocument, error) {
	defaults, err := canonicalDefaults(preset.ModelParam)
	if err != nil {
		return modelv1.ModelDocument{}, err
	}
	capabilities, err := canonicalCapabilities(
		preset.CapabilitiesOverride,
	)
	if err != nil {
		return modelv1.ModelDocument{}, err
	}

	labels := map[string]string(nil)
	if disabled {
		labels = map[string]string{
			modelDomain.BuiltInInitialEnabledLabel: "false",
		}
	}

	return modelv1.ModelDocument{
		Type:        modelv1.ModelType,
		Name:        string(modelName),
		DisplayName: preset.DisplayName,
		Labels:      labels,
		Provider: declaration.ArtifactNameReference{
			Name:  providerName,
			Scope: declaration.LookupScopeBuiltin,
		},
		ProviderModelID: string(preset.Name),
		Defaults:        defaults,
		Capabilities:    capabilities,
	}, nil
}

func adapterIDForSDKType(
	value inferenceSpec.ProviderSDKType,
) (string, error) {
	switch value {
	case inferenceSpec.ProviderSDKTypeAnthropic:
		return adapterAnthropicMessages, nil
	case inferenceSpec.ProviderSDKTypeOpenAIChatCompletions:
		return adapterOpenAIChatCompletions, nil
	case inferenceSpec.ProviderSDKTypeOpenAIResponses:
		return adapterOpenAIResponses, nil
	case inferenceSpec.ProviderSDKTypeGoogleGenerateContent:
		return adapterGoogleGenerateContent, nil
	default:
		return "", fmt.Errorf(
			"%w: unsupported inference Provider SDK type %q",
			spec.ErrUnsupported,
			value,
		)
	}
}

func canonicalConnection(
	origin string,
	path string,
	headers map[string]string,
) (json.RawMessage, error) {
	type headerPatch struct {
		Set map[string]string `json:"set,omitempty"`
	}
	type connection struct {
		Origin  string       `json:"origin"`
		Path    string       `json:"path"`
		Headers *headerPatch `json:"headers,omitempty"`
	}

	value := connection{
		Origin: origin,
		Path:   path,
	}
	if len(headers) != 0 {
		value.Headers = &headerPatch{
			Set: maps.Clone(headers),
		}
	}
	return canonicalObject(value)
}

func canonicalDefaults(
	value inferenceSpec.ModelParam,
) (json.RawMessage, error) {
	fields := map[string]any{
		"stream":          value.Stream,
		"maxPromptTokens": value.MaxPromptLength,
		"maxOutputTokens": value.MaxOutputLength,
		"systemPrompt":    value.SystemPrompt,
		"timeoutMS":       timeoutMilliseconds(value.Timeout),
	}
	if value.Temperature != nil {
		fields["temperature"] = *value.Temperature
	}
	if value.Reasoning != nil {
		fields["reasoning"] = value.Reasoning
	}
	if value.CacheControl != nil {
		fields["cacheControl"] = value.CacheControl
	}
	if output := outputValue(value.OutputParam); output != nil {
		fields["output"] = output
	}
	if len(value.StopSequences) != 0 {
		fields["stopSequences"] = append(
			[]string(nil),
			value.StopSequences...,
		)
	}
	if value.AdditionalParametersRawJSON != nil &&
		strings.TrimSpace(*value.AdditionalParametersRawJSON) != "" {
		raw, err := jsonutil.CanonicalizeObject(
			[]byte(*value.AdditionalParametersRawJSON),
			spec.MaxDefinitionBodyBytes,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"canonicalize Model adapterParameters: %w",
				err,
			)
		}
		fields["adapterParameters"] = json.RawMessage(raw)
	}
	return canonicalObject(fields)
}

func outputValue(
	value *inferenceSpec.OutputParam,
) map[string]any {
	if value == nil {
		return nil
	}

	output := make(map[string]any)
	if value.Verbosity != nil {
		output["verbosity"] = string(*value.Verbosity)
	}
	if value.Format != nil {
		format := map[string]any{
			"kind": string(value.Format.Kind),
		}
		if value.Format.JSONSchemaParam != nil {
			format["jsonSchema"] = map[string]any{
				"name":        value.Format.JSONSchemaParam.Name,
				"description": value.Format.JSONSchemaParam.Description,
				"schema":      value.Format.JSONSchemaParam.Schema,
				"strict":      value.Format.JSONSchemaParam.Strict,
			}
		}
		output["format"] = format
	}
	if len(output) == 0 {
		return nil
	}
	return output
}

func canonicalCapabilities(
	value *capabilityoverride.ModelCapabilitiesOverride,
) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func canonicalObject(value any) (json.RawMessage, error) {
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func timeoutMilliseconds(value int) int {
	if value <= 0 {
		return 0
	}
	maximum := int(^uint(0) >> 1)
	if value > maximum/1000 {
		return maximum
	}
	return value * 1000
}

func generatedNameForPreset(
	values []preparedModel,
	id modelpreset.ModelPresetID,
) (spec.LogicalName, bool) {
	for _, value := range values {
		if value.presetID == id {
			return value.name, true
		}
	}
	return "", false
}

func modelDisabled(
	provider inferenceSpec.ProviderName,
	id modelpreset.ModelPresetID,
) bool {
	disabled := disabledModelsByProvider[provider]
	_, found := disabled[id]
	return found
}

func generatedModelLogicalName(
	provider spec.LogicalName,
	providerModelID string,
) (spec.LogicalName, error) {
	if err := provider.Validate(); err != nil {
		return "", err
	}
	if err := spec.ValidateRequiredText(
		"generated Model providerModelID",
		providerModelID,
		spec.MaxURIBytes,
	); err != nil {
		return "", err
	}

	var normalized strings.Builder
	normalized.WriteString(string(provider))
	normalized.WriteByte('-')

	separator := false
	for _, character := range strings.ToLower(providerModelID) {
		switch {
		case character >= 'a' && character <= 'z':
			normalized.WriteRune(character)
			separator = false
		case character >= '0' && character <= '9':
			normalized.WriteRune(character)
			separator = false
		case character == '.', character == '_', character == '-':
			normalized.WriteRune(character)
			separator = false
		case unicode.IsLetter(character), unicode.IsDigit(character):
			if !separator {
				normalized.WriteByte('-')
				separator = true
			}
		default:
			if !separator {
				normalized.WriteByte('-')
				separator = true
			}
		}
	}

	candidate := strings.Trim(
		normalized.String(),
		".-_",
	)
	if value := spec.LogicalName(candidate); value.Validate() == nil {
		return value, nil
	}

	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes(
			[]byte(string(provider)+"\x00"+providerModelID),
		)),
		cryptoutil.DigestSHA256Prefix,
	)
	suffix := "-model-" + digest[:16]
	maximumPrefix := spec.MaxLogicalNameBytes - len(suffix)
	prefix := string(provider)
	if len(prefix) > maximumPrefix {
		prefix = prefix[:maximumPrefix]
	}
	prefix = strings.TrimRight(prefix, ".-_")
	value := spec.LogicalName(prefix + suffix)
	if err := value.Validate(); err != nil {
		return "", err
	}
	return value, nil
}

func validatePolicy(
	catalog modelpreset.Catalog,
) error {
	if len(catalog.Providers) == 0 {
		return fmt.Errorf(
			"%w: inference Model catalog is empty",
			spec.ErrInvalid,
		)
	}

	for providerName, provider := range catalog.Providers {
		defaultID, found := defaultModelByProvider[providerName]
		if !found {
			return fmt.Errorf(
				"%w: Provider %q has no application default Model policy",
				spec.ErrInvalid,
				providerName,
			)
		}
		if _, found := provider.ModelPresets[defaultID]; !found {
			return fmt.Errorf(
				"%w: Provider %q default Model %q is absent from inference catalog",
				spec.ErrInvalid,
				providerName,
				defaultID,
			)
		}
	}

	for providerName, defaultID := range defaultModelByProvider {
		provider, found := catalog.Providers[providerName]
		if !found {
			return fmt.Errorf(
				"%w: application default Model policy references unknown Provider %q",
				spec.ErrInvalid,
				providerName,
			)
		}
		if _, found := provider.ModelPresets[defaultID]; !found {
			return fmt.Errorf(
				"%w: application default Model policy references unknown Model %q/%q",
				spec.ErrInvalid,
				providerName,
				defaultID,
			)
		}
	}

	for providerName, disabled := range disabledModelsByProvider {
		provider, found := catalog.Providers[providerName]
		if !found {
			return fmt.Errorf(
				"%w: disabled Model policy references unknown Provider %q",
				spec.ErrInvalid,
				providerName,
			)
		}
		for modelID := range disabled {
			if _, found := provider.ModelPresets[modelID]; !found {
				return fmt.Errorf(
					"%w: disabled Model policy references unknown Model %q/%q",
					spec.ErrInvalid,
					providerName,
					modelID,
				)
			}
		}
	}

	for providerName := range staticProviderHeaderOverlays {
		if _, found := catalog.Providers[providerName]; !found {
			return fmt.Errorf(
				"%w: static Provider header policy references unknown Provider %q",
				spec.ErrInvalid,
				providerName,
			)
		}
	}

	return nil
}
