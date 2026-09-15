package providerapi

import (
	"context"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
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
	Locator             basespec.Locator
	SourceContentDigest cryptoutil.Digest
	Content             []byte
	RequestedDecoderIDs []basespec.DecoderID
}

// SourceContent is one bounded, snapshot-backed source file requested by a
// source-aware decoder. The discovery engine owns the read, digest, and
// snapshot lifetime.
type SourceContent struct {
	Locator basespec.Locator
	Content []byte
	Digest  cryptoutil.Digest
}

// SourceEntryReader permits a source-aware decoder to read explicitly
// referenced sibling source files without exposing Source configuration,
// snapshots, or native paths.
type SourceEntryReader interface {
	ReadSourceEntry(ctx context.Context, locator basespec.Locator) (SourceContent, error)
}

// ArtifactResourceClaim states that a matching independently decoded source
// entry is resource material for the claiming declaration rather than a
// second declaration of the same Artifact identity.
//
// Claims are Source-local. They suppress only available observations with the
// exact Kind and LogicalName. Other Artifacts emitted by the target file
// remain independently discoverable.
type ArtifactResourceClaim struct {
	Locator     basespec.Locator
	Recursive   bool
	Kind        artifact.ArtifactKind
	LogicalName basespec.LogicalName
}

func (c ArtifactResourceClaim) Validate() error {
	if err := c.Locator.Validate(true); err != nil {
		return err
	}
	if err := c.Kind.Validate(); err != nil {
		return err
	}
	return c.LogicalName.Validate()
}

func (c Candidate) RequestsDecoder(id basespec.DecoderID) bool {
	return slices.Contains(c.RequestedDecoderIDs, id)
}

// Decoded is one provider-derived definition emitted from a source candidate.
type Decoded struct {
	SubresourceLocator basespec.SubresourceLocator

	// OriginLocator and OriginContentDigest override the candidate file as
	// the physical declaration origin. They are used by source-aware format
	// adapters such as a Skill Collection manifest that emits Skills from
	// referenced SKILL.md files.
	//
	// An origin override is an inseparable locator/digest pair. Supplying only
	// one member would attach content evidence to the wrong physical entry.
	OriginLocator       basespec.Locator
	OriginContentDigest *cryptoutil.Digest

	// ResourceClaims prevent a separately scanned resource from becoming a
	// duplicate available Artifact of the same identity.
	ResourceClaims []ArtifactResourceClaim

	Definition  definition.Definition
	Diagnostics []diagnostic.Diagnostic
}

// Decoder is an Artifact Store inbound content-decoding plugin.
//
// Artifact Store owns decoder selection, bounded reads, content hashing,
// source refresh synchronization, Definition persistence, and Artifact
// lifecycle publication. The decoder owns format recognition and projection.
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
