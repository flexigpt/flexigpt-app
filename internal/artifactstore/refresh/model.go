package refresh

import (
	"context"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type CollectionReader interface {
	Get(
		ctx context.Context,
		ref collection.CollectionRef,
	) (collection.Collection, error)

	ListAttachments(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]collection.Attachment, error)
}

type ArtifactReader interface {
	ListByCollection(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)

	ListSuppressions(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Suppression, error)
}

type Publication struct {
	Ref collection.CollectionRef

	// ExpectedCatalogRevision is zero when the root has no prior publication.
	// It prevents a concurrent refresh from replacing a newer catalog with an
	// older source observation.
	ExpectedCatalogRevision     uint64
	ExpectedCollectionRevision  uint64
	ExpectedAttachmentRevisions map[basespec.SourceID]uint64
	ExpectedSourceRevisions     map[basespec.SourceID]uint64
	SourceGenerations           map[basespec.SourceID]string
	PlanFingerprint             cryptoutil.Digest
	DecoderFingerprint          cryptoutil.Digest
	Occurrences                 []catalog.Occurrence
	ArtifactCreates             []artifact.Artifact
	ArtifactUpdates             []artifact.SourceStateUpdate
	Diagnostics                 []diagnostic.Diagnostic
	PublishedAt                 time.Time
}

func (p Publication) Validate() error {
	if err := p.Ref.Validate(); err != nil {
		return err
	}
	if p.ExpectedCollectionRevision == 0 {
		return fmt.Errorf(
			"%w: expected collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := cryptoutil.ValidateDigest(p.PlanFingerprint); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(p.DecoderFingerprint); err != nil {
		return err
	}
	knownSources := make(map[basespec.SourceID]struct{}, len(p.ExpectedSourceRevisions))
	for sourceID, revision := range p.ExpectedSourceRevisions {
		if err := basespec.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if revision == 0 {
			return fmt.Errorf(
				"%w: expected source revision must be positive",
				basespec.ErrInvalid,
			)
		}
		knownSources[sourceID] = struct{}{}
	}
	for sourceID := range knownSources {
		if _, exists := p.ExpectedAttachmentRevisions[sourceID]; !exists {
			return fmt.Errorf(
				"%w: expected source revision has no collection attachment",
				basespec.ErrInvalid,
			)
		}
	}
	for sourceID, revision := range p.ExpectedAttachmentRevisions {
		if err := basespec.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if revision == 0 {
			return fmt.Errorf(
				"%w: expected attachment revision must be positive",
				basespec.ErrInvalid,
			)
		}
		if _, exists := knownSources[sourceID]; !exists {
			return fmt.Errorf("%w: attachment has no source revision", basespec.ErrInvalid)
		}
	}
	for sourceID, generation := range p.SourceGenerations {
		if err := basespec.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if _, exists := knownSources[sourceID]; !exists {
			return fmt.Errorf(
				"%w: source generation belongs to an unattached source %q",
				basespec.ErrInvalid,
				sourceID,
			)
		}
		if err := basespec.ValidateSourceGeneration(generation); err != nil {
			return err
		}
	}

	validOccurrences := make(
		map[basespec.SourceBinding]catalog.Occurrence,
		len(p.Occurrences),
	)
	seenOccurrences := make(map[catalog.OccurrenceKey]struct{}, len(p.Occurrences))
	for index, occurrence := range p.Occurrences {
		if occurrence.RootID != p.Ref.RootID ||
			occurrence.CollectionID != p.Ref.CollectionID {
			return fmt.Errorf(
				"%w: occurrence %d belongs to another collection",
				basespec.ErrInvalid,
				index,
			)
		}

		if _, exists := knownSources[occurrence.Key.SourceID]; !exists {
			return fmt.Errorf(
				"%w: occurrence %d belongs to an unattached source",
				basespec.ErrInvalid,
				index,
			)
		}

		if _, exists := p.SourceGenerations[occurrence.Key.SourceID]; !exists {
			return fmt.Errorf(
				"%w: occurrence %d has no source generation",
				basespec.ErrInvalid,
				index,
			)
		}

		if _, duplicate := seenOccurrences[occurrence.Key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate occurrence %d",
				basespec.ErrInvalid,
				index,
			)
		}

		seenOccurrences[occurrence.Key] = struct{}{}
		if err := occurrence.Validate(); err != nil {
			return err
		}

		if occurrence.State == catalog.OccurrenceValid &&
			occurrence.DefinitionDigest != nil {
			binding := basespec.SourceBinding{
				SourceID:           occurrence.Key.SourceID,
				Locator:            occurrence.Key.Locator,
				SubresourceLocator: occurrence.Key.SubresourceLocator,
				ExpectedKind:       occurrence.Kind,
			}
			validOccurrences[binding] = occurrence
		}
	}

	seenArtifacts := make(map[basespec.ArtifactID]struct{})
	validateArtifact := func(value artifact.Artifact) error {
		if err := value.Validate(); err != nil {
			return err
		}
		if value.RootID != p.Ref.RootID ||
			value.CollectionID != p.Ref.CollectionID {
			return fmt.Errorf(
				"%w: artifact belongs to another collection",
				basespec.ErrInvalid,
			)
		}

		if _, exists := knownSources[value.Binding.SourceID]; !exists {
			return fmt.Errorf(
				"%w: artifact belongs to an unattached source",
				basespec.ErrInvalid,
			)
		}

		if _, duplicate := seenArtifacts[value.ID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate artifact publication %q",
				basespec.ErrInvalid,
				value.ID,
			)
		}

		seenArtifacts[value.ID] = struct{}{}
		return nil
	}

	for index, value := range p.ArtifactCreates {
		if err := validateArtifact(value); err != nil {
			return fmt.Errorf("artifact create %d: %w", index, err)
		}

		if value.Adoption != artifact.AdoptionObserved {
			return fmt.Errorf(
				"%w: refresh publication cannot create a non-observed artifact",
				basespec.ErrInvalid,
			)
		}
		if value.State != artifact.StateAvailable {
			return fmt.Errorf(
				"%w: refresh publication can only create available observed artifacts",
				basespec.ErrInvalid,
			)
		}

		occurrence, exists := validOccurrences[value.Binding]
		if !exists || occurrence.DefinitionDigest == nil {
			return fmt.Errorf(
				"%w: artifact create %d has no current valid occurrence",
				basespec.ErrInvalid,
				index,
			)
		}
		if value.ResolvedDefinition == nil ||
			*value.ResolvedDefinition != *occurrence.DefinitionDigest {
			return fmt.Errorf(
				"%w: artifact create %d definition does not match its occurrence",
				basespec.ErrInvalid,
				index,
			)
		}
		if value.Kind != occurrence.Kind {
			return fmt.Errorf(
				"%w: artifact create %d kind does not match its occurrence",
				basespec.ErrInvalid,
				index,
			)
		}
	}

	for index, update := range p.ArtifactUpdates {
		if err := update.Validate(); err != nil {
			return fmt.Errorf("artifact update %d: %w", index, err)
		}
		if update.RootID != p.Ref.RootID ||
			update.CollectionID != p.Ref.CollectionID {
			return fmt.Errorf(
				"%w: artifact update belongs to another collection",
				basespec.ErrInvalid,
			)
		}

		if _, duplicate := seenArtifacts[update.ArtifactID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate artifact publication %q",
				basespec.ErrInvalid,
				update.ArtifactID,
			)
		}

		seenArtifacts[update.ArtifactID] = struct{}{}
	}

	if err := diagnostic.ValidateDiagnostics(p.Diagnostics); err != nil {
		return err
	}
	if p.PublishedAt.IsZero() {
		return fmt.Errorf(
			"%w: publication time is required",
			basespec.ErrInvalid,
		)
	}
	return nil
}

type Publisher interface {
	Publish(
		ctx context.Context,
		publication Publication,
	) (catalog.Snapshot, error)
}

type Result struct {
	Catalog          catalog.Snapshot
	CreatedArtifacts []basespec.ArtifactID
	UpdatedArtifacts []basespec.ArtifactID
	Diagnostics      []diagnostic.Diagnostic
	Candidates       int
}

type Runner interface {
	Refresh(
		ctx context.Context,
		ref collection.CollectionRef,
		plan discovery.Plan,
		policy artifact.Policy,
	) (Result, error)
}
