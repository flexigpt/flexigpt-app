package refresh

import (
	"context"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type RootReader interface {
	GetRoot(
		ctx context.Context,
		id artifactstore.RootID,
		includeDeleted bool,
	) (catalog.Root, error)

	ListAttachments(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]catalog.Attachment, error)

	GetCurrentCatalog(
		ctx context.Context,
		rootID artifactstore.RootID,
	) (catalog.Snapshot, error)
}

type SourceReader interface {
	Get(
		ctx context.Context,
		id artifactstore.SourceID,
	) (source.Source, error)
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
	RecordUpdates           []record.UpdatePublication
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
	}
	for _, occurrence := range p.Occurrences {
		if err := occurrence.Validate(); err != nil {
			return err
		}
	}
	for _, value := range p.RecordCreates {
		if err := value.Validate(); err != nil {
			return err
		}
	}
	for _, update := range p.RecordUpdates {
		if update.ExpectedRevision == 0 {
			return fmt.Errorf(
				"%w: expected record revision is required",
				artifactstore.ErrInvalid,
			)
		}
		if err := update.Record.Validate(); err != nil {
			return err
		}
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
