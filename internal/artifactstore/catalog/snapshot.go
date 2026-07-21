package catalog

import (
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Snapshot struct {
	RootID            artifactstore.RootID              `json:"rootID"`
	Revision          uint64                            `json:"revision"`
	RootRevision      uint64                            `json:"rootRevision"`
	SourceRevisions   map[artifactstore.SourceID]uint64 `json:"sourceRevisions"`
	SourceGenerations map[artifactstore.SourceID]string `json:"sourceGenerations"`
	PublishedAt       time.Time                         `json:"publishedAt"`
	Diagnostics       []artifactstore.Diagnostic        `json:"diagnostics,omitempty"`
	Occurrences       []Occurrence                      `json:"occurrences"`
}

func (s Snapshot) Validate() error {
	if err := artifactstore.ValidateRootID(s.RootID); err != nil {
		return err
	}
	if s.Revision == 0 || s.RootRevision == 0 {
		return fmt.Errorf("%w: catalog revisions must be positive", artifactstore.ErrInvalid)
	}
	for sourceID, revision := range s.SourceRevisions {
		if err := artifactstore.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if revision == 0 {
			return fmt.Errorf("%w: source revision must be positive", artifactstore.ErrInvalid)
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
		if generation == "" {
			return fmt.Errorf("%w: source generation is empty", artifactstore.ErrInvalid)
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
		if err := occurrence.Validate(); err != nil {
			return fmt.Errorf("occurrence %d: %w", index, err)
		}
	}
	return nil
}
