package resource

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// ResolveOptions controls Store-owned source verification.
//
// Catalog and definition verification are always performed. Source content
// verification additionally opens the exact catalogued Source generation and
// confirms the occurrence bytes before returning.
type ResolveOptions struct {
	VerifySourceContent bool
}

// ResolvedArtifact is an immutable, Store-verified resource projection.
//
// It intentionally contains a Source summary rather than Source configuration
// or a Source snapshot. Native path and source-content access remain methods
// on Service so snapshot lifecycle and verification stay inside Artifact Store.
type ResolvedArtifact struct {
	Artifact         artifact.Artifact
	Collection       collection.Collection
	Definition       providerapi.Definition
	Occurrence       catalog.Occurrence
	Source           source.Summary
	CatalogRevision  uint64
	SourceGeneration string
}

func (r ResolvedArtifact) Validate() error {
	if err := r.Artifact.Validate(); err != nil {
		return err
	}
	if err := r.Collection.Validate(); err != nil {
		return err
	}
	if err := r.Definition.Validate(); err != nil {
		return err
	}
	if err := r.Occurrence.Validate(); err != nil {
		return err
	}
	if err := r.Source.Validate(); err != nil {
		return err
	}
	if r.CatalogRevision == 0 {
		return fmt.Errorf(
			"%w: resolved Artifact catalog revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateSourceGeneration(r.SourceGeneration); err != nil {
		return err
	}

	if r.Artifact.RootID != r.Collection.RootID ||
		r.Artifact.CollectionID != r.Collection.ID {
		return fmt.Errorf(
			"%w: resolved Artifact belongs to another Collection",
			basespec.ErrInvalid,
		)
	}
	if r.Source.RootID != r.Collection.RootID ||
		r.Source.ID != r.Artifact.Binding.SourceID {
		return fmt.Errorf(
			"%w: resolved Source does not match the Artifact binding",
			basespec.ErrInvalid,
		)
	}
	if r.Occurrence.RootID != r.Artifact.RootID ||
		r.Occurrence.CollectionID != r.Artifact.CollectionID ||
		r.Occurrence.Key.CollectionID != r.Artifact.CollectionID ||
		r.Occurrence.Key.SourceID != r.Artifact.Binding.SourceID ||
		r.Occurrence.Key.Locator != r.Artifact.Binding.Locator ||
		r.Occurrence.Key.SubresourceLocator !=
			r.Artifact.Binding.SubresourceLocator {
		return fmt.Errorf(
			"%w: resolved occurrence does not match the Artifact binding",
			basespec.ErrInvalid,
		)
	}
	if r.Occurrence.State != catalog.OccurrenceValid ||
		r.Occurrence.Kind != r.Artifact.Kind ||
		r.Occurrence.DefinitionDigest == nil ||
		r.Occurrence.SourceContentDigest == nil ||
		r.Artifact.ResolvedDefinition == nil {
		return fmt.Errorf(
			"%w: resolved Artifact has no current valid occurrence",
			basespec.ErrReferenceUnresolved,
		)
	}
	if r.Definition.Kind != r.Artifact.Kind ||
		r.Definition.Digest != *r.Artifact.ResolvedDefinition ||
		r.Definition.Digest != *r.Occurrence.DefinitionDigest {
		return fmt.Errorf(
			"%w: resolved definition does not match Artifact state",
			basespec.ErrDigestMismatch,
		)
	}
	return nil
}

func (r ResolvedArtifact) Clone() ResolvedArtifact {
	output := r
	output.Artifact = r.Artifact.Clone()
	output.Collection = r.Collection.Clone()
	output.Definition = r.Definition.Clone()
	output.Occurrence = catalog.CloneOccurrence(r.Occurrence)
	output.Source = cloneSourceSummary(r.Source)
	return output
}

// VerifiedEntry contains bytes read from one exact current Collection Source
// generation.
type VerifiedEntry struct {
	Collection       collection.CollectionRef
	SourceID         basespec.SourceID
	CatalogRevision  uint64
	SourceRevision   uint64
	SourceGeneration string
	Content          []byte
	Digest           cryptoutil.Digest
}

func (e VerifiedEntry) Validate() error {
	if err := e.Collection.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateSourceID(e.SourceID); err != nil {
		return err
	}
	if e.CatalogRevision == 0 || e.SourceRevision == 0 {
		return fmt.Errorf(
			"%w: verified entry revisions are required",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateSourceGeneration(e.SourceGeneration); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(e.Digest); err != nil {
		return err
	}
	if cryptoutil.DigestBytes(e.Content) != e.Digest {
		return fmt.Errorf(
			"%w: verified entry content does not match its digest",
			basespec.ErrDigestMismatch,
		)
	}
	return nil
}

func (e VerifiedEntry) Clone() VerifiedEntry {
	output := e
	output.Content = append([]byte(nil), e.Content...)
	return output
}

func cloneSourceSummary(input source.Summary) source.Summary {
	output := input
	if input.RetiredAt != nil {
		value := *input.RetiredAt
		output.RetiredAt = &value
	}
	return output
}
