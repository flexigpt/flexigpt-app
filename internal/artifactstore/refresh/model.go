package refresh

import (
	"context"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
)

type RootReader interface {
	Get(
		ctx context.Context,
		id artifactstore.RootID,
	) (root.Root, error)

	ListAttachments(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]root.Attachment, error)
}

type RecordReader interface {
	ListByRoot(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]record.Record, error)
}

type Publication struct {
	RootID                  artifactstore.RootID
	ExpectedRootRevision    uint64
	ExpectedSourceRevisions map[artifactstore.SourceID]uint64
	SourceGenerations       map[artifactstore.SourceID]string
	Occurrences             []catalog.Occurrence
	RecordCreates           []record.Record
	RecordUpdates           []record.SourceStateUpdate
	Diagnostics             []artifactstore.Diagnostic
	PublishedAt             time.Time
}

func (p Publication) Validate() error {
	if err := artifactstore.ValidateRootID(p.RootID); err != nil {
		return err
	}
	if p.ExpectedRootRevision == 0 {
		return fmt.Errorf(
			"%w: expected root revision is required",
			artifactstore.ErrInvalid,
		)
	}
	knownSources := make(map[artifactstore.SourceID]struct{}, len(p.ExpectedSourceRevisions))
	for sourceID, revision := range p.ExpectedSourceRevisions {
		if err := artifactstore.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if revision == 0 {
			return fmt.Errorf(
				"%w: expected source revision must be positive",
				artifactstore.ErrInvalid,
			)
		}
		knownSources[sourceID] = struct{}{}
	}
	for sourceID, generation := range p.SourceGenerations {
		if err := artifactstore.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if _, exists := knownSources[sourceID]; !exists {
			return fmt.Errorf(
				"%w: source generation belongs to an unattached source %q",
				artifactstore.ErrInvalid,
				sourceID,
			)
		}
		if err := artifactstore.ValidateSourceGeneration(generation); err != nil {
			return err
		}
	}
	seenOccurrences := make(map[catalog.OccurrenceKey]struct{}, len(p.Occurrences))
	for index, occurrence := range p.Occurrences {
		if occurrence.RootID != p.RootID {
			return fmt.Errorf(
				"%w: occurrence %d belongs to another root",
				artifactstore.ErrInvalid,
				index,
			)
		}
		if _, exists := knownSources[occurrence.Key.SourceID]; !exists {
			return fmt.Errorf(
				"%w: occurrence %d belongs to an unattached source",
				artifactstore.ErrInvalid,
				index,
			)
		}
		if _, exists := p.SourceGenerations[occurrence.Key.SourceID]; !exists {
			return fmt.Errorf(
				"%w: occurrence %d has no source generation",
				artifactstore.ErrInvalid,
				index,
			)
		}
		if _, duplicate := seenOccurrences[occurrence.Key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate occurrence %d",
				artifactstore.ErrInvalid,
				index,
			)
		}
		seenOccurrences[occurrence.Key] = struct{}{}
		if err := occurrence.Validate(); err != nil {
			return err
		}
	}
	seenRecords := make(map[artifactstore.RecordID]struct{})
	validateRecord := func(value record.Record) error {
		if err := value.Validate(); err != nil {
			return err
		}
		if value.RootID != p.RootID {
			return fmt.Errorf(
				"%w: record belongs to another root",
				artifactstore.ErrInvalid,
			)
		}
		if _, exists := knownSources[value.Occurrence.SourceID]; !exists {
			return fmt.Errorf(
				"%w: record belongs to an unattached source",
				artifactstore.ErrInvalid,
			)
		}
		if _, duplicate := seenRecords[value.ID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate record publication %q",
				artifactstore.ErrInvalid,
				value.ID,
			)
		}
		seenRecords[value.ID] = struct{}{}
		return nil
	}
	for index, value := range p.RecordCreates {
		if err := validateRecord(value); err != nil {
			return fmt.Errorf("record create %d: %w", index, err)
		}
	}
	for index, update := range p.RecordUpdates {
		if err := update.Validate(); err != nil {
			return fmt.Errorf("record update %d: %w", index, err)
		}
		if update.RootID != p.RootID {
			return fmt.Errorf(
				"%w: record update belongs to another root",
				artifactstore.ErrInvalid,
			)
		}
		if _, duplicate := seenRecords[update.RecordID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate record publication %q",
				artifactstore.ErrInvalid,
				update.RecordID,
			)
		}
		seenRecords[update.RecordID] = struct{}{}
	}
	if err := artifactstore.ValidateDiagnostics(p.Diagnostics); err != nil {
		return err
	}
	if p.PublishedAt.IsZero() {
		return fmt.Errorf(
			"%w: publication time is required",
			artifactstore.ErrInvalid,
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
	Catalog        catalog.Snapshot
	CreatedRecords []artifactstore.RecordID
	UpdatedRecords []artifactstore.RecordID
	Diagnostics    []artifactstore.Diagnostic
	Candidates     int
}

type Runner interface {
	Refresh(
		ctx context.Context,
		rootID artifactstore.RootID,
		plan discovery.Plan,
		policy record.Policy,
	) (Result, error)
}
