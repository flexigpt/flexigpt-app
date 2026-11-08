// Package registration selects the LLM declaration registrations used by the
// FlexiGPT application assembly. It deliberately exposes separately named
// registration concerns rather than a universal registration descriptor.
package registration

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/llmsupport"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	corelocator "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/registration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
)

// NewLLMInterpretationRegistry maps application topology support into the
// reusable LLM declaration factory. Declaration-family assembly itself lives
// in internal/llmartifactory-go/registration.
func NewLLMInterpretationRegistry() (*coreinterpretation.Registry, error) {
	config, err := newLLMDeclarationConfig()
	if err != nil {
		return nil, err
	}
	return registration.NewInterpretationRegistry(
		config.SkillDocuments,
	)
}

func LLMDeclarationSchemaCodecs() ([]schema.Codec, error) {
	bundle, err := newLLMDeclarationBundle()
	if err != nil {
		return nil, err
	}
	return bundle.SchemaCodecs(), nil
}

func LLMDeclarationSchemaCodecsForSchemaKeys(
	keys []schemaModel.Key,
) ([]schema.Codec, error) {
	bundle, err := newLLMDeclarationBundle()
	if err != nil {
		return nil, err
	}
	return bundle.SchemaCodecsForSchemaKeys(keys)
}

func LLMCanonicalDeclarationDecoders(
	registry *coreinterpretation.Registry,
) ([]ingest.Decoder, error) {
	config, err := newLLMDeclarationConfig()
	if err != nil {
		return nil, err
	}
	return registration.CanonicalDeclarationDecodersForSchemaKeys(
		registry,
		registry.SchemaKeys(),
		config.CanonicalJSONCandidates,
		config.CanonicalYAMLCandidates,
	)
}

func LLMCanonicalDeclarationDecodersForSchemaKeys(
	registry *coreinterpretation.Registry,
	keys []schemaModel.Key,
) ([]ingest.Decoder, error) {
	config, err := newLLMDeclarationConfig()
	if err != nil {
		return nil, err
	}
	return registration.CanonicalDeclarationDecodersForSchemaKeys(
		registry,
		keys,
		config.CanonicalJSONCandidates,
		config.CanonicalYAMLCandidates,
	)
}

func LLMSourceFormatDecoders(
	registry *coreinterpretation.Registry,
) ([]ingest.Decoder, error) {
	config, err := newLLMDeclarationConfig()
	if err != nil {
		return nil, err
	}
	return registration.SourceFormatDecoders(
		registry,
		config,
	)
}

func LLMPathLocatorFactories(
	registry *coreinterpretation.Registry,
) ([]corelocator.Factory, error) {
	return registration.PathLocatorFactories(registry)
}

func decoderCandidates(
	decoderID storeSpec.DecoderID,
) (support.Candidates, error) {
	patterns, err := topology.DocumentPatternsForDecoder(decoderID)
	if err != nil {
		return support.Candidates{}, err
	}
	value := support.Candidates{Patterns: patterns}
	if err := value.Validate(); err != nil {
		return support.Candidates{}, err
	}
	return value, nil
}

func documentSupport(
	use string,
) (support.Documents, error) {
	locator, err := topology.DefaultDocumentFile(use)
	if err != nil {
		return support.Documents{}, err
	}
	decoderID, err := topology.DefaultDocumentDecoderID(use)
	if err != nil {
		return support.Documents{}, err
	}
	files, err := topology.DocumentFiles(use)
	if err != nil {
		return support.Documents{}, err
	}
	patterns, err := topology.DocumentPatterns(use)
	if err != nil {
		return support.Documents{}, err
	}

	value := support.Documents{
		Default:  support.Document{Locator: locator, DecoderID: decoderID},
		Files:    files,
		Patterns: patterns,
	}
	return value, value.Validate()
}

func documentUseCandidates(
	use string,
) (support.Candidates, error) {
	patterns, err := topology.DocumentPatterns(use)
	if err != nil {
		return support.Candidates{}, err
	}
	value := support.Candidates{Patterns: patterns}
	if err := value.Validate(); err != nil {
		return support.Candidates{}, err
	}
	return value, nil
}

func newLLMDeclarationBundle() (*registration.Bundle, error) {
	config, err := newLLMDeclarationConfig()
	if err != nil {
		return nil, err
	}
	return registration.New(config)
}

// newLLMDeclarationConfig is the application composition boundary. It maps
// topology-selected names and package layouts into the generic LLM
// declaration factory without owning declaration grammar or schema assembly.
func newLLMDeclarationConfig() (registration.Config, error) {
	skillDocuments, err := documentSupport(
		topology.DocumentUseSkillPackage,
	)
	if err != nil {
		return registration.Config{}, err
	}
	skillSupport, err := llmsupport.Skill()
	if err != nil {
		return registration.Config{}, err
	}

	jsonCandidates, err := decoderCandidates(
		"artifact-declaration-json",
	)
	if err != nil {
		return registration.Config{}, err
	}
	yamlCandidates, err := decoderCandidates(
		"artifact-declaration-yaml",
	)
	if err != nil {
		return registration.Config{}, err
	}
	agentMarkdownCandidates, err := decoderCandidates(
		"agent-markdown",
	)
	if err != nil {
		return registration.Config{}, err
	}
	textMarkdownCandidates, err := decoderCandidates(
		"text-markdown",
	)
	if err != nil {
		return registration.Config{}, err
	}
	instructionCandidates, err := documentUseCandidates(
		topology.DocumentUseWorkspaceInstructions,
	)
	if err != nil {
		return registration.Config{}, err
	}
	skillMarkdownCandidates, err := decoderCandidates(
		"agent.skill-markdown",
	)
	if err != nil {
		return registration.Config{}, err
	}
	mcpConfigCandidates, err := decoderCandidates(
		"artifact.mcp-json",
	)
	if err != nil {
		return registration.Config{}, err
	}

	config := registration.Config{
		SkillDocuments:          skillDocuments,
		SkillPackage:            skillSupport.Package,
		CanonicalJSONCandidates: jsonCandidates,
		CanonicalYAMLCandidates: yamlCandidates,
		AgentMarkdownCandidates: agentMarkdownCandidates,
		TextMarkdownCandidates:  textMarkdownCandidates,
		InstructionCandidates:   instructionCandidates,
		SkillMarkdownCandidates: skillMarkdownCandidates,
		MCPConfigCandidates:     mcpConfigCandidates,
	}
	return config, config.Validate()
}
