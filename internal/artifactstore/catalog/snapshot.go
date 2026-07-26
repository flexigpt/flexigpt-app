package catalog

import (
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Snapshot struct {
	RootID              artifactstore.RootID              `json:"rootID"`
	CollectionID        artifactstore.CollectionID        `json:"collectionID"`
	Revision            uint64                            `json:"revision"`
	CollectionRevision  uint64                            `json:"collectionRevision"`
	AttachmentRevisions map[artifactstore.SourceID]uint64 `json:"attachmentRevisions"`
	SourceRevisions     map[artifactstore.SourceID]uint64 `json:"sourceRevisions"`
	SourceGenerations   map[artifactstore.SourceID]string `json:"sourceGenerations"`
	PlanFingerprint     artifactstore.Digest              `json:"planFingerprint"`
	DecoderFingerprint  artifactstore.Digest              `json:"decoderFingerprint"`
	PublishedAt         time.Time                         `json:"publishedAt"`
	Diagnostics         []artifactstore.Diagnostic        `json:"diagnostics,omitempty"`
	Occurrences         []Occurrence                      `json:"occurrences"`
}

func (s Snapshot) Validate() error {
	if err := artifactstore.ValidateRootID(s.RootID); err != nil {
		return err
	}
	if err := artifactstore.ValidateCollectionID(s.CollectionID); err != nil {
		return err
	}
	if s.Revision == 0 || s.CollectionRevision == 0 {
		return fmt.Errorf("%w: catalog revisions must be positive", artifactstore.ErrInvalid)
	}
	if err := artifactstore.ValidateDigest(s.PlanFingerprint); err != nil {
		return fmt.Errorf("catalog plan fingerprint: %w", err)
	}
	if err := artifactstore.ValidateDigest(s.DecoderFingerprint); err != nil {
		return fmt.Errorf("catalog decoder fingerprint: %w", err)
	}
	for sourceID, revision := range s.AttachmentRevisions {
		if err := artifactstore.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if revision == 0 {
			return fmt.Errorf("%w: attachment revision must be positive", artifactstore.ErrInvalid)
		}
	}
	for sourceID := range s.AttachmentRevisions {
		if _, exists := s.SourceRevisions[sourceID]; !exists {
			return fmt.Errorf(
				"%w: collection attachment has no corresponding source revision",
				artifactstore.ErrInvalid,
			)
		}
	}
	for sourceID, revision := range s.SourceRevisions {
		if err := artifactstore.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if revision == 0 {
			return fmt.Errorf("%w: source revision must be positive", artifactstore.ErrInvalid)
		}
		if _, attached := s.AttachmentRevisions[sourceID]; !attached {
			return fmt.Errorf(
				"%w: source revision has no collection attachment",
				artifactstore.ErrInvalid,
			)
		}
	}
	for sourceID, generation := range s.SourceGenerations {
		if err := artifactstore.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if _, exists := s.SourceRevisions[sourceID]; !exists {
			return fmt.Errorf(
				"%w: source generation has no source revision",
				artifactstore.ErrInvalid,
			)
		}
		if err := artifactstore.ValidateSourceGeneration(generation); err != nil {
			return err
		}
	}
	if s.PublishedAt.IsZero() {
		return fmt.Errorf("%w: catalog publication time is required", artifactstore.ErrInvalid)
	}
	if err := artifactstore.ValidateDiagnostics(s.Diagnostics); err != nil {
		return err
	}
	seenOccurrences := make(map[OccurrenceKey]struct{}, len(s.Occurrences))
	for index, occurrence := range s.Occurrences {
		if _, duplicate := seenOccurrences[occurrence.Key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate occurrence %d",
				artifactstore.ErrInvalid,
				index,
			)
		}
		seenOccurrences[occurrence.Key] = struct{}{}
		if occurrence.RootID != s.RootID {
			return fmt.Errorf(
				"%w: occurrence %d belongs to another root",
				artifactstore.ErrInvalid,
				index,
			)
		}
		if occurrence.CollectionID != s.CollectionID ||
			occurrence.Key.CollectionID != s.CollectionID {
			return fmt.Errorf(
				"%w: occurrence %d belongs to another collection",
				artifactstore.ErrInvalid,
				index,
			)
		}
		if _, exists := s.SourceRevisions[occurrence.Key.SourceID]; !exists {
			return fmt.Errorf(
				"%w: occurrence %d has no source revision",
				artifactstore.ErrInvalid,
				index,
			)
		}
		if _, exists := s.SourceGenerations[occurrence.Key.SourceID]; !exists {
			return fmt.Errorf(
				"%w: occurrence %d has no source generation",
				artifactstore.ErrInvalid,
				index,
			)
		}
		if err := occurrence.Validate(); err != nil {
			return fmt.Errorf("occurrence %d: %w", index, err)
		}
	}
	return nil
}
