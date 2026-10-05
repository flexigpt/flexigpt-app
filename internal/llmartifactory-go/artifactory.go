package llmartifactory

import (
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
)

// Artifactory is the LLM-domain attachment to one assembled generic Artifact
// Store. It owns LLM registration lifetime only. Generic Root, Source,
// Artifact, Definition, Resource, Secret, Overlay, and Install capabilities
// remain owned by store/compose.Store.
type Artifactory struct {
	store *compose.Store

	schemaCodecs     []schema.Codec
	decoders         []ingest.Decoder
	locatorFactories []locator.Factory

	mu     sync.RWMutex
	closed bool
}

// SchemaCodecs returns the exact LLM declaration schema registrations attached
// to this aggregate.
func (a *Artifactory) SchemaCodecs() []schema.Codec {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return nil
	}
	return append([]schema.Codec(nil), a.schemaCodecs...)
}

// Decoders returns the exact LLM declaration decoder registrations attached to
// this aggregate.
func (a *Artifactory) Decoders() []ingest.Decoder {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return nil
	}
	return append([]ingest.Decoder(nil), a.decoders...)
}

// LocatorFactories returns the registered LLM declaration locator factories.
// Family services bind these through the composition locator owner rather than
// constructing private locator registries.
func (a *Artifactory) LocatorFactories() []locator.Factory {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return nil
	}
	return append([]locator.Factory(nil), a.locatorFactories...)
}
