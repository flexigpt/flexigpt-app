package provider

import (
	"context"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
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
	SourceID            source.SourceID
	SourceKind          source.SourceKind
	Locator             model.Locator
	SourceContentDigest cryptoutil.Digest
	Content             []byte
	RequestedDecoderIDs []model.DecoderID
}

// SourceContent is one bounded, snapshot-backed source file requested by a
// source-aware decoder. The discovery engine owns the read, digest, and
// snapshot lifetime.
type SourceContent struct {
	Locator model.Locator
	Content []byte
	Digest  cryptoutil.Digest
}

// SourceEntryReader permits a source-aware decoder to read explicitly
// referenced sibling source files without exposing Source configuration,
// snapshots, or native paths.
type SourceEntryReader interface {
	ReadSourceEntry(ctx context.Context, locator model.Locator) (SourceContent, error)
}

func (c Candidate) RequestsDecoder(id model.DecoderID) bool {
	return slices.Contains(c.RequestedDecoderIDs, id)
}

// Decoded is one provider-derived definition emitted from a source candidate.
type Decoded struct {
	SubresourceLocator model.SubresourceLocator

	// OriginLocator and OriginContentDigest override the candidate file as
	// the physical declaration origin. They are used by source-aware format
	// adapters such as a Skill Collection manifest that emits Skills from
	// referenced SKILL.md files.
	//
	// An origin override is an inseparable locator/digest pair. Supplying only
	// one member would attach content evidence to the wrong physical entry.
	OriginLocator       model.Locator
	OriginContentDigest *cryptoutil.Digest

	Definition  definition.Definition
	Diagnostics []diagnostic.Diagnostic
}

// Decoder is an Artifact Store inbound content-decoding plugin.
//
// Artifact Store owns decoder selection, bounded reads, content hashing,
// source refresh synchronization, Definition persistence, and Artifact
// lifecycle publication. The decoder owns format recognition and projection.
type Decoder interface {
	ID() model.DecoderID
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

// SourceAwareDecoder is implemented only by formats whose declarations refer
// to sibling source files. Ordinary decoders remain candidate-byte-only.
type SourceAwareDecoder interface {
	Decoder

	DecodeWithSource(
		ctx context.Context,
		candidate Candidate,
		reader SourceEntryReader,
	) ([]Decoded, []diagnostic.Diagnostic)
}

// SchemaCanonicalizerBinder is optional. A decoder implements it when its
// source bytes must be canonicalized through the registered Artifact Store
// schema catalog before the decoder can project definitions.
//
// This replaces direct decoder dependencies on *shareable.Registry.
type SchemaCanonicalizerBinder interface {
	RequiredSchemaKeys() []schema.Key

	BindExpectedCanonicalizer(
		schemas SchemaCatalog,
	) error
}
