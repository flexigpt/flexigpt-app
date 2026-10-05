package llmartifactory

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
)

// Open attaches the LLM artifact-domain registration set to an already opened
// generic Artifact Store. Generic Store ownership remains with deployment
// assembly; this constructor does not open or close provider resources.
func Open(ctx context.Context, config Config) (*Artifactory, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: LLM Artifactory construction context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if config.Store == nil {
		return nil, fmt.Errorf(
			"%w: LLM Artifactory generic Store is nil",
			spec.ErrInvalid,
		)
	}

	codecs, err := schema.NormalizeCodecs(config.SchemaCodecs)
	if err != nil {
		return nil, err
	}
	decoders, err := ingest.NormalizeDecoders(config.Decoders)
	if err != nil {
		return nil, err
	}
	locators, err := locator.NewRegistry(config.LocatorFactories...)
	if err != nil {
		return nil, err
	}

	return &Artifactory{
		store:            config.Store,
		schemaCodecs:     codecs,
		decoders:         decoders,
		locatorFactories: locators.Factories(),
	}, nil
}
