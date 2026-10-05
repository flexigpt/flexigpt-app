// Package registration selects the LLM declaration registrations used by the
// FlexiGPT application assembly. It deliberately exposes separately named
// registration concerns rather than a universal registration descriptor.
package registration

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	agentmarkdown "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/sourceformat/markdown"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/decoder"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
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
func NewLLMInterpretationRegistry() (*interpretation.Registry, error) {
	return interpretation.NewRegistry(llmFamilyRegistrations()...)
}

// LLMDeclarationSchemaCodecs returns the selected family schema codecs. Schema
// registration is separate from decoder and locator selection.
func LLMDeclarationSchemaCodecs() ([]schema.Codec, error) {
	registrations := llmFamilyRegistrations()
	output := make([]schema.Codec, 0, len(registrations))

	for _, registration := range registrations {
		schemaJSON, err := llmFamilySchema(registration.DeclarationType)
		if err != nil {
			return nil, err
		}

		codec, err := interpretation.NewSchemaCodec(
			registration,
			schemaJSON,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"create declaration schema codec for %q: %w",
				registration.DeclarationType,
				err,
			)
		}
		output = append(output, codec)
	}

	return output, nil
}

// LLMCanonicalDeclarationDecoders returns the canonical JSON and YAML
// declaration decoders bound to one immutable interpretation registry.
func LLMCanonicalDeclarationDecoders(
	registry *interpretation.Registry,
) ([]ingest.Decoder, error) {
	if registry == nil {
		return nil, fmt.Errorf(
			"%w: LLM declaration interpretation registry is nil",
			spec.ErrInvalid,
		)
	}

	return []ingest.Decoder{
		decoder.NewJSONDecoder(registry),
		decoder.NewYAMLDecoder(registry),
	}, nil
}

// LLMSourceFormatDecoders returns the application-selected adapters for
// non-canonical source formats. These adapters remain separate from canonical
// declaration decoding.
func LLMSourceFormatDecoders(
	registry *interpretation.Registry,
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
	registry *interpretation.Registry,
) ([]locator.Factory, error) {
	factory, err := locator.NewPathFactory(registry)
	if err != nil {
		return nil, err
	}
	return []locator.Factory{factory}, nil
}

func llmFamilyRegistrations() []interpretation.Registration {
	return []interpretation.Registration{
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

func llmFamilySchema(
	declarationType declaration.Type,
) ([]byte, error) {
	switch declarationType {
	case textv1.TextType:
		return textv1.TextJSONSchema(), nil
	case modelv1.ModelType:
		return modelv1.ModelJSONSchema(), nil
	case modelproviderv1.ModelProviderType:
		return modelproviderv1.ModelProviderJSONSchema(), nil
	case toolv1.ToolType:
		return toolv1.ToolJSONSchema(), nil
	case skillv1.SkillType:
		return skillv1.SkillJSONSchema(), nil
	case mcpv1.MCPType:
		return mcpv1.MCPJSONSchema(), nil
	case mcppolicyv1.MCPPolicyType:
		return mcppolicyv1.MCPPolicyJSONSchema(), nil
	case pluginv1.PluginType:
		return pluginv1.PluginJSONSchema(), nil
	case agentv1.AgentType:
		return agentv1.AgentJSONSchema(), nil
	case teamv1.TeamType:
		return teamv1.TeamJSONSchema(), nil
	case loopv1.LoopType:
		return loopv1.LoopJSONSchema(), nil
	case workflowv1.WorkflowType:
		return workflowv1.WorkflowJSONSchema(), nil
	case workspacev1.WorkspaceType:
		return workspacev1.WorkspaceJSONSchema(), nil
	default:
		return nil, fmt.Errorf(
			"%w: no selected LLM schema for declaration type %q",
			spec.ErrUnsupported,
			declarationType,
		)
	}
}
