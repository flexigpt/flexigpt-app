package providerapi

import (
	"context"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Recognition int

const (
	RecognitionNone Recognition = iota
	RecognitionPossible
	RecognitionPreferred
)

// Candidate is one bounded source entry selected by Artifact Store discovery.
//
// Artifact Store owns snapshot opening, bounded reads, source-content digest
// calculation, and source generation confirmation. A decoder receives only
// the candidate bytes and generic source identity.
type Candidate struct {
	SourceID            basespec.SourceID
	SourceKind          basespec.SourceKind
	Locator             basespec.Locator
	SourceContentDigest cryptoutil.Digest
	Content             []byte
	RequestedDecoderIDs []basespec.DecoderID
}

func (c Candidate) RequestsDecoder(id basespec.DecoderID) bool {
	return slices.Contains(c.RequestedDecoderIDs, id)
}

// Decoded is one provider-derived definition emitted from a source candidate.
//
// This compatibility-first version retains definition.Definition and
// diagnostic.Diagnostic. The next decoder migration should replace these with
// providerapi DefinitionDraft and Diagnostic DTOs so decoder implementations
// depend only on providerapi, basespec, and the standard library.
type Decoded struct {
	SubresourceLocator basespec.SubresourceLocator
	Definition         definition.Definition
	Diagnostics        []diagnostic.Diagnostic
}

// Decoder is an Artifact Store inbound content-decoding plugin.
//
// Artifact Store owns decoder selection, read limits, content hashing,
// occurrence state transitions, definition persistence, and catalog
// publication. The decoder owns format recognition and semantic projection.
type Decoder interface {
	ID() basespec.DecoderID
	Revision() string

	Recognize(
		ctx context.Context,
		candidate Candidate,
	) Recognition

	Decode(
		ctx context.Context,
		candidate Candidate,
	) ([]Decoded, []diagnostic.Diagnostic)
}

// SchemaCanonicalizerBinder is optional. A decoder implements it when its
// source bytes must be canonicalized through the registered Artifact Store
// schema catalog before the decoder can project definitions.
//
// This replaces direct decoder dependencies on *shareable.Registry.
type SchemaCanonicalizerBinder interface {
	RequiredSchemaKeys() []SchemaKey

	BindExpectedCanonicalizer(
		schemas SchemaCatalog,
	) error
}
