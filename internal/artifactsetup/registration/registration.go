package registration

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	corelocator "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

type Selection struct {
	interpretations *coreinterpretation.Registry
	schemaCodecs    []schema.Codec
	decoders        []ingest.Decoder
	locators        []corelocator.Factory
}

func New() (*Selection, error) {
	interpretations, err := NewLLMInterpretationRegistry()
	if err != nil {
		return nil, err
	}

	codecs, err := LLMDeclarationSchemaCodecs()
	if err != nil {
		return nil, err
	}

	canonicalDecoders, err := LLMCanonicalDeclarationDecoders(
		interpretations,
	)
	if err != nil {
		return nil, err
	}
	sourceFormatDecoders, err := LLMSourceFormatDecoders(
		interpretations,
	)
	if err != nil {
		return nil, err
	}

	decoders := append(
		append([]ingest.Decoder(nil), canonicalDecoders...),
		sourceFormatDecoders...,
	)
	decoders, err = ingest.NormalizeDecoders(decoders)
	if err != nil {
		return nil, err
	}

	factories, err := LLMPathLocatorFactories(interpretations)
	if err != nil {
		return nil, err
	}
	locators, err := corelocator.NewRegistry(factories...)
	if err != nil {
		return nil, err
	}

	return &Selection{
		interpretations: interpretations,
		schemaCodecs:    codecs,
		decoders:        decoders,
		locators:        locators.Factories(),
	}, nil
}

func (s *Selection) Interpretations() *coreinterpretation.Registry {
	if s == nil {
		return nil
	}
	return s.interpretations
}

func (s *Selection) SchemaCodecs() []schema.Codec {
	if s == nil {
		return nil
	}
	return append([]schema.Codec(nil), s.schemaCodecs...)
}

func (s *Selection) Decoders() []ingest.Decoder {
	if s == nil {
		return nil
	}
	return append([]ingest.Decoder(nil), s.decoders...)
}

func (s *Selection) LocatorFactories() []corelocator.Factory {
	if s == nil {
		return nil
	}
	return append([]corelocator.Factory(nil), s.locators...)
}
