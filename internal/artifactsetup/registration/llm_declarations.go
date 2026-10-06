// Package registration selects the LLM declaration registrations used by the
// FlexiGPT application assembly. It deliberately exposes separately named
// registration concerns rather than a universal registration descriptor.
package registration

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	agentmarkdown "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/sourceformat/markdown"
	corelocator "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	coredecoder "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/decoder"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	loopv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/loop/contract/v1"
	mcpv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/contract/v1"
	mcpconfig "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/sourceformat/config"
	mcppolicyv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcppolicy/contract/v1"
	modelv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
	modelproviderv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/modelprovider/contract/v1"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	skillv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/contract/v1"
	skillmarkdown "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/sourceformat/markdown"
	teamv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/team/contract/v1"
	textv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/contract/v1"
	textmarkdown "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/sourceformat/markdown"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
	workspacev1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contract/v1"
)

// NewLLMInterpretationRegistry assembles the selected application declaration
// families. Family semantics remain owned by the family contract packages.
func NewLLMInterpretationRegistry() (*coreinterpretation.Registry, error) {
	return coreinterpretation.NewRegistry(llmFamilyRegistrations()...)
}

// LLMDeclarationSchemaCodecs returns the selected family schema codecs. Schema
// registration is separate from decoder and locator selection.
func LLMDeclarationSchemaCodecs() ([]schema.Codec, error) {
	keys := make([]schemaModel.Key, 0, len(llmFamilyRegistrations()))
	for _, registration := range llmFamilyRegistrations() {
		keys = append(keys, registration.SchemaKey)
	}
	return LLMDeclarationSchemaCodecsForSchemaKeys(keys)
}

// LLMDeclarationSchemaCodecsForSchemaKeys returns only the requested LLM
// declaration schema codecs. Built-in package compilation uses this narrower
// selection so unrelated family schema changes do not alter another package
// set's hydration fingerprint.
func LLMDeclarationSchemaCodecsForSchemaKeys(
	keys []schemaModel.Key,
) ([]schema.Codec, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf(
			"%w: LLM declaration schema selection is empty",
			spec.ErrInvalid,
		)
	}

	all, err := allLLMDeclarationSchemaCodecs()
	if err != nil {
		return nil, err
	}
	byKey := make(map[schemaModel.Key]schema.Codec, len(all))
	for _, codec := range all {
		byKey[codec.Key()] = codec
	}

	selected := make([]schema.Codec, 0, len(keys))
	seen := make(map[schemaModel.Key]struct{}, len(keys))
	for _, key := range keys {
		if err := key.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf(
				"%w: LLM declaration schema %q/%q/%q is repeated",
				spec.ErrConflict,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}
		codec, found := byKey[key]
		if !found {
			return nil, fmt.Errorf(
				"%w: LLM declaration schema %q/%q/%q",
				spec.ErrUnsupported,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}
		seen[key] = struct{}{}
		selected = append(selected, codec)
	}
	return schema.NormalizeCodecs(selected)
}

func allLLMDeclarationSchemaCodecs() ([]schema.Codec, error) {
	output := make([]schema.Codec, 0, len(llmFamilyRegistrations()))
	add := func(
		registration coreinterpretation.Registration,
		raw []byte,
	) error {
		codec, err := coreinterpretation.NewSchemaCodec(
			registration,
			raw,
		)
		if err != nil {
			return err
		}
		output = append(output, codec)
		return nil
	}

	if err := add(textv1.Interpretation(), textv1.TextJSONSchema()); err != nil {
		return nil, fmt.Errorf("create Text schema codec: %w", err)
	}
	if err := add(modelv1.Interpretation(), modelv1.ModelJSONSchema()); err != nil {
		return nil, fmt.Errorf("create Model schema codec: %w", err)
	}
	if err := add(
		modelproviderv1.Interpretation(),
		modelproviderv1.ModelProviderJSONSchema(),
	); err != nil {
		return nil, fmt.Errorf("create Model Provider schema codec: %w", err)
	}
	if err := add(toolv1.Interpretation(), toolv1.ToolJSONSchema()); err != nil {
		return nil, fmt.Errorf("create Tool schema codec: %w", err)
	}
	if err := add(skillv1.Interpretation(), skillv1.SkillJSONSchema()); err != nil {
		return nil, fmt.Errorf("create Skill schema codec: %w", err)
	}
	if err := add(mcpv1.Interpretation(), mcpv1.MCPJSONSchema()); err != nil {
		return nil, fmt.Errorf("create MCP schema codec: %w", err)
	}
	if err := add(
		mcppolicyv1.Interpretation(),
		mcppolicyv1.MCPPolicyJSONSchema(),
	); err != nil {
		return nil, fmt.Errorf("create MCP Policy schema codec: %w", err)
	}
	if err := add(pluginv1.Interpretation(), pluginv1.PluginJSONSchema()); err != nil {
		return nil, fmt.Errorf("create Plugin schema codec: %w", err)
	}
	if err := add(agentv1.Interpretation(), agentv1.AgentJSONSchema()); err != nil {
		return nil, fmt.Errorf("create Agent schema codec: %w", err)
	}
	if err := add(teamv1.Interpretation(), teamv1.TeamJSONSchema()); err != nil {
		return nil, fmt.Errorf("create Team schema codec: %w", err)
	}
	if err := add(loopv1.Interpretation(), loopv1.LoopJSONSchema()); err != nil {
		return nil, fmt.Errorf("create Loop schema codec: %w", err)
	}
	if err := add(
		workflowv1.Interpretation(),
		workflowv1.WorkflowJSONSchema(),
	); err != nil {
		return nil, fmt.Errorf("create Workflow schema codec: %w", err)
	}
	if err := add(
		workspacev1.Interpretation(),
		workspacev1.WorkspaceJSONSchema(),
	); err != nil {
		return nil, fmt.Errorf("create Workspace schema codec: %w", err)
	}
	return output, nil
}

// LLMCanonicalDeclarationDecoders returns the canonical JSON and YAML
// declaration decoders bound to one immutable interpretation registry.
func LLMCanonicalDeclarationDecoders(
	registry *coreinterpretation.Registry,
) ([]ingest.Decoder, error) {
	if registry == nil {
		return nil, fmt.Errorf(
			"%w: LLM declaration interpretation registry is nil",
			spec.ErrInvalid,
		)
	}

	return LLMCanonicalDeclarationDecodersForSchemaKeys(
		registry,
		registry.SchemaKeys(),
	)
}

// LLMCanonicalDeclarationDecodersForSchemaKeys returns canonical declaration
// decoders restricted to one package set's selected schema keys. The supplied
// registry remains complete so contained declaration semantic reconstruction
// continues to use family-owned validators.
func LLMCanonicalDeclarationDecodersForSchemaKeys(
	registry *coreinterpretation.Registry,
	keys []schemaModel.Key,
) ([]ingest.Decoder, error) {
	if registry == nil {
		return nil, fmt.Errorf(
			"%w: LLM declaration interpretation registry is nil",
			spec.ErrInvalid,
		)
	}
	selected, err := registry.RequireSchemaKeys(keys)
	if err != nil {
		return nil, err
	}

	jsonDecoder, err := coredecoder.NewJSONDecoderForSchemaKeys(
		registry,
		selected,
	)
	if err != nil {
		return nil, err
	}
	yamlDecoder, err := coredecoder.NewYAMLDecoderForSchemaKeys(
		registry,
		selected,
	)
	if err != nil {
		return nil, err
	}
	return []ingest.Decoder{jsonDecoder, yamlDecoder}, nil
}

// LLMSourceFormatDecoders returns the application-selected adapters for
// non-canonical source formats. These adapters remain separate from canonical
// declaration decoding.
func LLMSourceFormatDecoders(
	registry *coreinterpretation.Registry,
) ([]ingest.Decoder, error) {
	if registry == nil {
		return nil, fmt.Errorf(
			"%w: LLM declaration interpretation registry is nil",
			spec.ErrInvalid,
		)
	}

	agentDecoder, err := agentmarkdown.NewAgentMarkdownDecoder(registry)
	if err != nil {
		return nil, err
	}

	return []ingest.Decoder{
		agentDecoder,
		textmarkdown.NewTextDecoder(),
		skillmarkdown.NewDecoder(),
		mcpconfig.NewDecoder(),
	}, nil
}

// LLMPathLocatorFactories returns the selected committed-state local path
// locator support. Additional locator kinds remain explicit application input.
func LLMPathLocatorFactories(
	registry *coreinterpretation.Registry,
) ([]corelocator.Factory, error) {
	factory, err := corelocator.NewPathFactory(registry)
	if err != nil {
		return nil, err
	}
	return []corelocator.Factory{factory}, nil
}

func llmFamilyRegistrations() []coreinterpretation.Registration {
	return []coreinterpretation.Registration{
		textv1.Interpretation(),
		modelv1.Interpretation(),
		modelproviderv1.Interpretation(),
		toolv1.Interpretation(),
		skillv1.Interpretation(),
		mcpv1.Interpretation(),
		mcppolicyv1.Interpretation(),
		pluginv1.Interpretation(),
		agentv1.Interpretation(),
		teamv1.Interpretation(),
		loopv1.Interpretation(),
		workflowv1.Interpretation(),
		workspacev1.Interpretation(),
	}
}
