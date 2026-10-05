package domain

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/modelproviderv1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/modelv1"
)

// DocumentType is the Model-domain document discriminator exposed to callers.
// It intentionally does not expose declaration.Header or declaration.Entry.
type DocumentType string

const (
	DocumentTypeProvider DocumentType = "model.provider"
	DocumentTypeModel    DocumentType = "model"
)

// LookupScope is the Model-domain lookup scope for logical Artifact references.
type LookupScope string

const (
	LookupScopeBuiltin LookupScope = "builtin"
)

// ArtifactNameReference is the Model-domain projection of a declaration
// logical-name reference.
type ArtifactNameReference struct {
	Name  spec.LogicalName `json:"name"`
	Scope LookupScope      `json:"scope,omitempty"`
}

func (r ArtifactNameReference) Clone() ArtifactNameReference {
	return r
}

func (r ArtifactNameReference) Validate() error {
	return r.ToDeclaration().Validate()
}

func (r ArtifactNameReference) ToDeclaration() declaration.ArtifactNameReference {
	return declaration.ArtifactNameReference{
		Name:  r.Name,
		Scope: declaration.LookupScope(r.Scope),
	}
}

func ArtifactNameReferenceFromDeclaration(
	value declaration.ArtifactNameReference,
) ArtifactNameReference {
	return ArtifactNameReference{
		Name:  value.Name,
		Scope: LookupScope(value.Scope),
	}
}

func ArtifactNameReferenceFromOptionalDeclaration(
	value *declaration.ArtifactNameReference,
) *ArtifactNameReference {
	if value == nil {
		return nil
	}
	output := ArtifactNameReferenceFromDeclaration(*value)
	return &output
}

func ArtifactNameReferenceToDeclaration(
	value *ArtifactNameReference,
) *declaration.ArtifactNameReference {
	if value == nil {
		return nil
	}
	output := value.ToDeclaration()
	return &output
}

type HeaderPatch struct {
	Set    map[string]string `json:"set,omitempty"`
	Remove []string          `json:"remove,omitempty"`
}

type ConnectionPatch struct {
	Origin  *string      `json:"origin,omitempty"`
	Path    *string      `json:"path,omitempty"`
	Headers *HeaderPatch `json:"headers,omitempty"`
}

type AuthenticationMode string

const (
	AuthenticationModeNone         AuthenticationMode = "none"
	AuthenticationModeAPIKeyHeader AuthenticationMode = "apiKeyHeader"
	AuthenticationModeBearerToken  AuthenticationMode = "bearerToken"
)

type Authentication struct {
	Mode       AuthenticationMode `json:"mode"`
	HeaderName string             `json:"headerName,omitempty"`
	Prefix     *string            `json:"prefix,omitempty"`
}

// AdapterParameters is intentionally an open object. Individual adapters own
// its keys and values, but callers exchange an object value, never encoded JSON.
type AdapterParameters map[string]any

// JSONSchemaValue is an intentionally open JSON Schema value. The declaration
// contract permits either an object or a boolean JSON Schema.
type JSONSchemaValue any

type (
	ReasoningType         string
	ReasoningLevel        string
	ReasoningSummaryStyle string
	ReasoningContext      string
	ReasoningMode         string
)

type ReasoningDefaults struct {
	Type         *ReasoningType         `json:"type,omitempty"`
	Level        *ReasoningLevel        `json:"level,omitempty"`
	Tokens       *int                   `json:"tokens,omitempty"`
	SummaryStyle *ReasoningSummaryStyle `json:"summaryStyle,omitempty"`
	Context      *ReasoningContext      `json:"context,omitempty"`
	Mode         *ReasoningMode         `json:"mode,omitempty"`
}

type (
	CacheControlKind string
	CacheControlTTL  string
)

type CacheControl struct {
	Kind *CacheControlKind `json:"kind,omitempty"`
	TTL  *CacheControlTTL  `json:"ttl,omitempty"`
	Key  *string           `json:"key,omitempty"`
}

type (
	OutputVerbosity  string
	OutputFormatKind string
)

type OutputJSONSchemaPatch struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Schema      JSONSchemaValue `json:"schema,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type OutputFormatPatch struct {
	Kind       *OutputFormatKind      `json:"kind,omitempty"`
	JSONSchema *OutputJSONSchemaPatch `json:"jsonSchema,omitempty"`
}

type OutputPatch struct {
	Verbosity *OutputVerbosity   `json:"verbosity,omitempty"`
	Format    *OutputFormatPatch `json:"format,omitempty"`
}

// DefaultsPatch is the typed portable Model defaults object used by source
// declarations, local settings, managed authoring, and request patches.
type DefaultsPatch struct {
	Stream            *bool              `json:"stream,omitempty"`
	MaxPromptTokens   *int               `json:"maxPromptTokens,omitempty"`
	MaxOutputTokens   *int               `json:"maxOutputTokens,omitempty"`
	Temperature       *float64           `json:"temperature,omitempty"`
	SystemPrompt      *string            `json:"systemPrompt,omitempty"`
	TimeoutMS         *int               `json:"timeoutMS,omitempty"`
	Reasoning         *ReasoningDefaults `json:"reasoning,omitempty"`
	CacheControl      *CacheControl      `json:"cacheControl,omitempty"`
	Output            *OutputPatch       `json:"output,omitempty"`
	StopSequences     []string           `json:"stopSequences,omitempty"`
	AdapterParameters *AdapterParameters `json:"adapterParameters,omitempty"`
}

type (
	ModalityIn               string
	ModalityOut              string
	ToolType                 string
	ToolPolicyMode           string
	ClientToolOutputFormat   string
	OutputTokenParameterName string
	ToolChoiceParameterStyle string
)

type ReasoningTokenBudgetCapabilities struct {
	MinAllowed      *int  `json:"minAllowed,omitempty"`
	MaxAllowed      *int  `json:"maxAllowed,omitempty"`
	ZeroAllowed     *bool `json:"zeroAllowed,omitempty"`
	MinusOneAllowed *bool `json:"minusOneAllowed,omitempty"`
}

type ReasoningCapabilities struct {
	SupportsReasoningConfig          *bool                             `json:"supportsReasoningConfig,omitempty"`
	SupportedReasoningTypes          *[]ReasoningType                  `json:"supportedReasoningTypes,omitempty"`
	SupportedReasoningLevels         *[]ReasoningLevel                 `json:"supportedReasoningLevels,omitempty"`
	HybridTokenBudgetCapabilities    *ReasoningTokenBudgetCapabilities `json:"hybridTokenBudgetCapabilities,omitempty"`
	SupportsSummaryStyle             *bool                             `json:"supportsSummaryStyle,omitempty"`
	SupportsReasoningContext         *bool                             `json:"supportsReasoningContext,omitempty"`
	SupportsReasoningMode            *bool                             `json:"supportsReasoningMode,omitempty"`
	SupportsEncryptedReasoningInput  *bool                             `json:"supportsEncryptedReasoningInput,omitempty"`
	TemperatureDisallowedWhenEnabled *bool                             `json:"temperatureDisallowedWhenEnabled,omitempty"`
}

type StopSequenceCapabilities struct {
	IsSupported             *bool `json:"isSupported,omitempty"`
	DisallowedWithReasoning *bool `json:"disallowedWithReasoning,omitempty"`
	MaxSequences            *int  `json:"maxSequences,omitempty"`
}

type OutputCapabilities struct {
	SupportedOutputFormats *[]OutputFormatKind `json:"supportedOutputFormats,omitempty"`
	SupportsVerbosity      *bool               `json:"supportsVerbosity,omitempty"`
}

type ToolCapabilities struct {
	SupportedToolTypes               *[]ToolType               `json:"supportedToolTypes,omitempty"`
	SupportedToolPolicyModes         *[]ToolPolicyMode         `json:"supportedToolPolicyModes,omitempty"`
	SupportsParallelToolCalls        *bool                     `json:"supportsParallelToolCalls,omitempty"`
	MaxForcedTools                   *int                      `json:"maxForcedTools,omitempty"`
	SupportedClientToolOutputFormats *[]ClientToolOutputFormat `json:"supportedClientToolOutputFormats,omitempty"`
}

type CacheControlCapabilities struct {
	SupportsTTL    *bool               `json:"supportsTTL,omitempty"`
	SupportedKinds *[]CacheControlKind `json:"supportedKinds,omitempty"`
	SupportedTTLs  *[]CacheControlTTL  `json:"supportedTTLs,omitempty"`
	SupportsKey    *bool               `json:"supportsKey,omitempty"`
}

type CacheCapabilities struct {
	SupportsAutomaticCaching *bool                     `json:"supportsAutomaticCaching,omitempty"`
	TopLevel                 *CacheControlCapabilities `json:"topLevel,omitempty"`
	InputOutputContent       *CacheControlCapabilities `json:"inputOutputContent,omitempty"`
	ReasoningContent         *CacheControlCapabilities `json:"reasoningContent,omitempty"`
	ToolChoice               *CacheControlCapabilities `json:"toolChoice,omitempty"`
	ToolCall                 *CacheControlCapabilities `json:"toolCall,omitempty"`
	ToolOutput               *CacheControlCapabilities `json:"toolOutput,omitempty"`
}

type ParameterDialect struct {
	MaxOutputTokensParamName *OutputTokenParameterName `json:"maxOutputTokensParamName,omitempty"`
	ToolChoiceParamStyle     *ToolChoiceParameterStyle `json:"toolChoiceParamStyle,omitempty"`
}

// CapabilitiesPatch is the typed portable capabilities patch shared by Model
// declarations and Model local settings.
type CapabilitiesPatch struct {
	ModalitiesIn             *[]ModalityIn             `json:"modalitiesIn,omitempty"`
	ModalitiesOut            *[]ModalityOut            `json:"modalitiesOut,omitempty"`
	ReasoningCapabilities    *ReasoningCapabilities    `json:"reasoningCapabilities,omitempty"`
	StopSequenceCapabilities *StopSequenceCapabilities `json:"stopSequenceCapabilities,omitempty"`
	OutputCapabilities       *OutputCapabilities       `json:"outputCapabilities,omitempty"`
	ToolCapabilities         *ToolCapabilities         `json:"toolCapabilities,omitempty"`
	CacheCapabilities        *CacheCapabilities        `json:"cacheCapabilities,omitempty"`
	ParamDialect             *ParameterDialect         `json:"paramDialect,omitempty"`
}

// ProviderDocument is the typed Model-domain authoring and read projection.
// Conversion to the source declaration contract remains internal.
type ProviderDocument struct {
	Type              DocumentType           `json:"type"`
	Name              spec.LogicalName       `json:"name"`
	DisplayName       string                 `json:"displayName,omitempty"`
	Description       string                 `json:"description,omitempty"`
	Labels            map[string]string      `json:"labels,omitempty"`
	Adapter           string                 `json:"adapter"`
	Connection        *ConnectionPatch       `json:"connection,omitempty"`
	Authentication    *Authentication        `json:"authentication,omitempty"`
	DefaultModel      *ArtifactNameReference `json:"defaultModel,omitempty"`
	Defaults          *DefaultsPatch         `json:"defaults,omitempty"`
	Capabilities      *CapabilitiesPatch     `json:"capabilities,omitempty"`
	AdapterParameters *AdapterParameters     `json:"adapterParameters,omitempty"`
}

// ModelDocument is the typed Model-domain authoring and read projection.
// Conversion to the source declaration contract remains internal.
type ModelDocument struct {
	Type              DocumentType          `json:"type"`
	Name              spec.LogicalName      `json:"name"`
	DisplayName       string                `json:"displayName,omitempty"`
	Description       string                `json:"description,omitempty"`
	Labels            map[string]string     `json:"labels,omitempty"`
	Provider          ArtifactNameReference `json:"provider"`
	ProviderModelID   string                `json:"providerModelID"`
	Defaults          *DefaultsPatch        `json:"defaults,omitempty"`
	Capabilities      *CapabilitiesPatch    `json:"capabilities,omitempty"`
	AdapterParameters *AdapterParameters    `json:"adapterParameters,omitempty"`
}

type ProviderSettings struct {
	Revision          uint64                 `json:"revision"`
	Connection        *ConnectionPatch       `json:"connection,omitempty"`
	Defaults          *DefaultsPatch         `json:"defaults,omitempty"`
	Capabilities      *CapabilitiesPatch     `json:"capabilities,omitempty"`
	DefaultModel      *ArtifactNameReference `json:"defaultModel,omitempty"`
	AdapterParameters *AdapterParameters     `json:"adapterParameters,omitempty"`
}

type ModelSettings struct {
	Revision          uint64             `json:"revision"`
	Defaults          *DefaultsPatch     `json:"defaults,omitempty"`
	Capabilities      *CapabilitiesPatch `json:"capabilities,omitempty"`
	AdapterParameters *AdapterParameters `json:"adapterParameters,omitempty"`
}

func (v ProviderDocument) Validate() error {
	_, err := v.ToDeclaration()
	return err
}

func (v ProviderDocument) ToDeclaration() (
	modelproviderv1.ProviderDocument,
	error,
) {
	raw, err := jsonutil.MarshalCanonicalObject(
		v,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return modelproviderv1.ProviderDocument{}, err
	}
	return modelproviderv1.DecodeModelProviderJSON(raw)
}

func ProviderDocumentFromDeclaration(
	value modelproviderv1.ProviderDocument,
) (ProviderDocument, error) {
	raw, err := value.CanonicalJSON()
	if err != nil {
		return ProviderDocument{}, err
	}

	var output ProviderDocument
	if err := json.Unmarshal(raw, &output); err != nil {
		return ProviderDocument{}, fmt.Errorf(
			"decode typed Model Provider document: %w",
			err,
		)
	}
	if err := output.Validate(); err != nil {
		return ProviderDocument{}, err
	}
	return output, nil
}

func (v ModelDocument) Validate() error {
	_, err := v.ToDeclaration()
	return err
}

func (v ModelDocument) ToDeclaration() (modelv1.ModelDocument, error) {
	raw, err := jsonutil.MarshalCanonicalObject(
		v,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return modelv1.ModelDocument{}, err
	}
	return modelv1.DecodeModelJSON(raw)
}

func ModelDocumentFromDeclaration(
	value modelv1.ModelDocument,
) (ModelDocument, error) {
	raw, err := value.CanonicalJSON()
	if err != nil {
		return ModelDocument{}, err
	}

	var output ModelDocument
	if err := json.Unmarshal(raw, &output); err != nil {
		return ModelDocument{}, fmt.Errorf(
			"decode typed Model document: %w",
			err,
		)
	}
	if err := output.Validate(); err != nil {
		return ModelDocument{}, err
	}
	return output, nil
}
