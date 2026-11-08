// Package registration assembles reusable LLM declaration-family registration
// components.
//
// It owns declaration-family schema, interpretation, decoder, and locator
// assembly. Application setup supplies document names, source candidates, and
// package layout choices. This package does not import application topology,
// runtime setup, providers, Wails, or Session Runtime.
package registration

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	agentmarkdown "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/sourceformat/markdown"
	corelocator "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	coredecoder "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/decoder"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	mcpv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/contract/v1"
	mcpconfig "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/sourceformat/config"
	mcppolicyv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcppolicy/contract/v1"
	modelv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
	modelproviderv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/modelprovider/contract/v1"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	skillv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/contract/v1"
	skillmarkdown "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/sourceformat/markdown"
	textv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/contract/v1"
	textmarkdown "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/sourceformat/markdown"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
	workspacev1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contract/v1"
)

// Config supplies application-selected source naming support to the reusable
// LLM declaration registration assembly. It contains no source driver,
// topology, persistence, runtime, or desktop dependency.
type Config struct {
	SkillDocuments support.Documents
	SkillPackage   support.PackageLayout

	CanonicalJSONCandidates support.Candidates
	CanonicalYAMLCandidates support.Candidates
	AgentMarkdownCandidates support.Candidates
	TextMarkdownCandidates  support.Candidates
	InstructionCandidates   support.Candidates
	SkillMarkdownCandidates support.Candidates
	MCPConfigCandidates     support.Candidates
}

func (c Config) Clone() Config {
	output := c
	output.SkillDocuments = c.SkillDocuments.Clone()
	output.CanonicalJSONCandidates = c.CanonicalJSONCandidates.Clone()
	output.CanonicalYAMLCandidates = c.CanonicalYAMLCandidates.Clone()
	output.AgentMarkdownCandidates = c.AgentMarkdownCandidates.Clone()
	output.TextMarkdownCandidates = c.TextMarkdownCandidates.Clone()
	output.InstructionCandidates = c.InstructionCandidates.Clone()
	output.SkillMarkdownCandidates = c.SkillMarkdownCandidates.Clone()
	output.MCPConfigCandidates = c.MCPConfigCandidates.Clone()
	return output
}

func (c Config) Validate() error {
	if err := c.SkillDocuments.Validate(); err != nil {
		return err
	}
	if err := c.SkillPackage.Validate(); err != nil {
		return err
	}
	if c.SkillPackage.Document != c.SkillDocuments.Default {
		return fmt.Errorf(
			"%w: Skill package document differs from Skill document support",
			storeSpec.ErrInvalid,
		)
	}

	for _, candidate := range []struct {
		name  string
		value support.Candidates
	}{
		{
			name:  "canonical JSON",
			value: c.CanonicalJSONCandidates,
		},
		{
			name:  "canonical YAML",
			value: c.CanonicalYAMLCandidates,
		},
		{
			name:  "Agent Markdown",
			value: c.AgentMarkdownCandidates,
		},
		{
			name:  "Text Markdown",
			value: c.TextMarkdownCandidates,
		},
		{
			name:  "instruction Markdown",
			value: c.InstructionCandidates,
		},
		{
			name:  "Skill Markdown",
			value: c.SkillMarkdownCandidates,
		},
		{
			name:  "MCP configuration",
			value: c.MCPConfigCandidates,
		},
	} {
		if err := candidate.value.Validate(); err != nil {
			return fmt.Errorf(
				"%s decoder candidates: %w",
				candidate.name,
				err,
			)
		}
	}
	return nil
}

// Bundle is one immutable declaration-family registration set. It is shared
// by generic Artifact Store opening and LLM Artifactory composition.
type Bundle struct {
	config Config

	interpretations      *coreinterpretation.Registry
	schemaCodecs         []schema.Codec
	canonicalDecoders    []ingest.Decoder
	sourceFormatDecoders []ingest.Decoder
	decoders             []ingest.Decoder
	locatorFactories     []corelocator.Factory
}

// New builds all reusable LLM declaration registration components once.
func New(config Config) (*Bundle, error) {
	config = config.Clone()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	interpretations, err := NewInterpretationRegistry(
		config.SkillDocuments,
	)
	if err != nil {
		return nil, err
	}

	schemaCodecs, err := schemaCodecsForSchemaKeys(
		interpretations,
		config.SkillDocuments,
		interpretations.SchemaKeys(),
	)
	if err != nil {
		return nil, err
	}

	canonicalDecoders, err := CanonicalDeclarationDecodersForSchemaKeys(
		interpretations,
		interpretations.SchemaKeys(),
		config.CanonicalJSONCandidates,
		config.CanonicalYAMLCandidates,
	)
	if err != nil {
		return nil, err
	}

	sourceFormatDecoders, err := SourceFormatDecoders(
		interpretations,
		config,
	)
	if err != nil {
		return nil, err
	}

	decoders, err := ingest.NormalizeDecoders(
		append(
			append(
				[]ingest.Decoder(nil),
				canonicalDecoders...,
			),
			sourceFormatDecoders...,
		),
	)
	if err != nil {
		return nil, err
	}

	locatorFactories, err := PathLocatorFactories(interpretations)
	if err != nil {
		return nil, err
	}

	return &Bundle{
		config:               config,
		interpretations:      interpretations,
		schemaCodecs:         schemaCodecs,
		canonicalDecoders:    canonicalDecoders,
		sourceFormatDecoders: sourceFormatDecoders,
		decoders:             decoders,
		locatorFactories:     locatorFactories,
	}, nil
}

func (b *Bundle) Interpretations() *coreinterpretation.Registry {
	if b == nil {
		return nil
	}
	return b.interpretations
}

func (b *Bundle) SchemaCodecs() []schema.Codec {
	if b == nil {
		return nil
	}
	return append([]schema.Codec(nil), b.schemaCodecs...)
}

func (b *Bundle) Decoders() []ingest.Decoder {
	if b == nil {
		return nil
	}
	return append([]ingest.Decoder(nil), b.decoders...)
}

func (b *Bundle) SourceFormatDecoders() []ingest.Decoder {
	if b == nil {
		return nil
	}
	return append(
		[]ingest.Decoder(nil),
		b.sourceFormatDecoders...,
	)
}

func (b *Bundle) LocatorFactories() []corelocator.Factory {
	if b == nil {
		return nil
	}
	return append(
		[]corelocator.Factory(nil),
		b.locatorFactories...,
	)
}

func (b *Bundle) SchemaCodecsForSchemaKeys(
	keys []schemaModel.Key,
) ([]schema.Codec, error) {
	if b == nil || b.interpretations == nil {
		return nil, storeSpec.ErrClosed
	}
	return schemaCodecsForSchemaKeys(
		b.interpretations,
		b.config.SkillDocuments,
		keys,
	)
}

func (b *Bundle) CanonicalDeclarationDecodersForSchemaKeys(
	keys []schemaModel.Key,
) ([]ingest.Decoder, error) {
	if b == nil || b.interpretations == nil {
		return nil, storeSpec.ErrClosed
	}
	return CanonicalDeclarationDecodersForSchemaKeys(
		b.interpretations,
		keys,
		b.config.CanonicalJSONCandidates,
		b.config.CanonicalYAMLCandidates,
	)
}

type familyDefinition struct {
	registration coreinterpretation.Registration
	schema       []byte
}

func familyDefinitions(
	skillDocuments support.Documents,
) []familyDefinition {
	return []familyDefinition{
		{
			registration: textv1.Interpretation(),
			schema:       textv1.TextJSONSchema(),
		},
		{
			registration: modelv1.Interpretation(),
			schema:       modelv1.ModelJSONSchema(),
		},
		{
			registration: modelproviderv1.Interpretation(),
			schema:       modelproviderv1.ModelProviderJSONSchema(),
		},
		{
			registration: toolv1.Interpretation(),
			schema:       toolv1.ToolJSONSchema(),
		},
		{
			registration: skillv1.Interpretation(skillDocuments),
			schema:       skillv1.SkillJSONSchema(),
		},
		{
			registration: mcpv1.Interpretation(),
			schema:       mcpv1.MCPJSONSchema(),
		},
		{
			registration: mcppolicyv1.Interpretation(),
			schema:       mcppolicyv1.MCPPolicyJSONSchema(),
		},
		{
			registration: pluginv1.Interpretation(),
			schema:       pluginv1.PluginJSONSchema(),
		},
		{
			registration: agentv1.Interpretation(),
			schema:       agentv1.AgentJSONSchema(),
		},
		{
			registration: workflowv1.Interpretation(),
			schema:       workflowv1.WorkflowDefinitionJSONSchema(),
		},
		{
			registration: workspacev1.Interpretation(),
			schema:       workspacev1.WorkspaceJSONSchema(),
		},
	}
}

// NewInterpretationRegistry builds the source grammar and semantic
// interpretation registry for the selected LLM declaration families.
func NewInterpretationRegistry(
	skillDocuments support.Documents,
) (*coreinterpretation.Registry, error) {
	if err := skillDocuments.Validate(); err != nil {
		return nil, err
	}

	definitions := familyDefinitions(skillDocuments)
	registrations := make(
		[]coreinterpretation.Registration,
		0,
		len(definitions),
	)
	for _, definition := range definitions {
		registrations = append(
			registrations,
			definition.registration,
		)
	}
	return coreinterpretation.NewRegistry(registrations...)
}

// SchemaCodecsForSchemaKeys returns the selected LLM declaration schema
// codecs. It is used by generic Artifact Store schema assembly and package
// hydration without importing application topology.
func SchemaCodecsForSchemaKeys(
	skillDocuments support.Documents,
	keys []schemaModel.Key,
) ([]schema.Codec, error) {
	interpretations, err := NewInterpretationRegistry(skillDocuments)
	if err != nil {
		return nil, err
	}
	return schemaCodecsForSchemaKeys(
		interpretations,
		skillDocuments,
		keys,
	)
}

func schemaCodecsForSchemaKeys(
	interpretations *coreinterpretation.Registry,
	skillDocuments support.Documents,
	keys []schemaModel.Key,
) ([]schema.Codec, error) {
	if interpretations == nil {
		return nil, fmt.Errorf(
			"%w: LLM declaration interpretation registry is nil",
			storeSpec.ErrInvalid,
		)
	}

	selected, err := interpretations.RequireSchemaKeys(keys)
	if err != nil {
		return nil, err
	}

	all, err := allSchemaCodecs(skillDocuments)
	if err != nil {
		return nil, err
	}
	byKey := make(map[schemaModel.Key]schema.Codec, len(all))
	for _, codec := range all {
		byKey[codec.Key()] = codec
	}

	output := make([]schema.Codec, 0, len(selected))
	for _, key := range selected {
		codec, found := byKey[key]
		if !found {
			return nil, fmt.Errorf(
				"%w: LLM declaration schema %q/%q/%q",
				storeSpec.ErrUnsupported,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}
		output = append(output, codec)
	}
	return schema.NormalizeCodecs(output)
}

func allSchemaCodecs(
	skillDocuments support.Documents,
) ([]schema.Codec, error) {
	definitions := familyDefinitions(skillDocuments)
	output := make([]schema.Codec, 0, len(definitions))
	for _, definition := range definitions {
		codec, err := coreinterpretation.NewSchemaCodec(
			definition.registration,
			definition.schema,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"create %q schema codec: %w",
				definition.registration.DeclarationType,
				err,
			)
		}
		output = append(output, codec)
	}
	return schema.NormalizeCodecs(output)
}

// CanonicalDeclarationDecodersForSchemaKeys returns canonical JSON and YAML
// declaration decoders restricted to the requested complete schema keys.
func CanonicalDeclarationDecodersForSchemaKeys(
	interpretations *coreinterpretation.Registry,
	keys []schemaModel.Key,
	jsonCandidates support.Candidates,
	yamlCandidates support.Candidates,
) ([]ingest.Decoder, error) {
	if interpretations == nil {
		return nil, fmt.Errorf(
			"%w: LLM declaration interpretation registry is nil",
			storeSpec.ErrInvalid,
		)
	}
	if err := jsonCandidates.Validate(); err != nil {
		return nil, err
	}
	if err := yamlCandidates.Validate(); err != nil {
		return nil, err
	}

	selected, err := interpretations.RequireSchemaKeys(keys)
	if err != nil {
		return nil, err
	}

	jsonDecoder, err := coredecoder.NewJSONDecoderForSchemaKeys(
		interpretations,
		selected,
		jsonCandidates,
	)
	if err != nil {
		return nil, err
	}
	yamlDecoder, err := coredecoder.NewYAMLDecoderForSchemaKeys(
		interpretations,
		selected,
		yamlCandidates,
	)
	if err != nil {
		return nil, err
	}
	return []ingest.Decoder{
		jsonDecoder,
		yamlDecoder,
	}, nil
}

// SourceFormatDecoders returns application-selected non-canonical source
// format adapters. Their declaration semantics remain inside LLM contracts.
func SourceFormatDecoders(
	interpretations *coreinterpretation.Registry,
	config Config,
) ([]ingest.Decoder, error) {
	if interpretations == nil {
		return nil, fmt.Errorf(
			"%w: LLM declaration interpretation registry is nil",
			storeSpec.ErrInvalid,
		)
	}
	config = config.Clone()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	agentDecoder, err := agentmarkdown.NewAgentMarkdownDecoder(
		interpretations,
		config.AgentMarkdownCandidates,
	)
	if err != nil {
		return nil, err
	}
	skillDecoder, err := skillmarkdown.NewDecoder(
		config.SkillMarkdownCandidates,
		config.SkillDocuments,
		config.SkillPackage,
	)
	if err != nil {
		return nil, err
	}

	return []ingest.Decoder{
		agentDecoder,
		textmarkdown.NewTextDecoder(
			config.TextMarkdownCandidates,
			config.InstructionCandidates,
		),
		skillDecoder,
		mcpconfig.NewDecoder(config.MCPConfigCandidates),
	}, nil
}

// PathLocatorFactories returns LLM-owned local declaration locator support.
func PathLocatorFactories(
	interpretations *coreinterpretation.Registry,
) ([]corelocator.Factory, error) {
	factory, err := corelocator.NewPathFactory(interpretations)
	if err != nil {
		return nil, err
	}
	return []corelocator.Factory{factory}, nil
}
