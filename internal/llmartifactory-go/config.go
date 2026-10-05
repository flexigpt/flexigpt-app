package llmartifactory

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
)

// Config contains separately named LLM-domain registrations. It deliberately
// does not combine source drivers, persistence, runtime execution, content,
// schemas, decoders, and locator behavior into a universal provider object.
//
// Store is borrowed. Closing Artifactory never closes the generic Store.
type Config struct {
	Store *compose.Store

	SchemaCodecs     []schema.Codec
	Decoders         []ingest.Decoder
	LocatorFactories []locator.Factory
}
