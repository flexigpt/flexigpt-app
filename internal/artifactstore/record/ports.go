package record

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
)

type Repository interface {
	Get(
		ctx context.Context,
		id artifactstore.RecordID,
	) (Record, error)

	ListByRoot(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]Record, error)

	Update(
		ctx context.Context,
		value Record,
		expectedRevision uint64,
	) error

	Delete(
		ctx context.Context,
		id artifactstore.RecordID,
		expectedRevision uint64,
	) error
}

type Draft struct {
	Name    string
	Enabled bool
	Data    json.RawMessage
}

type Policy interface {
	Derive(
		ctx context.Context,
		value root.Root,
		occurrence catalog.Occurrence,
		def definition.Definition,
	) (Draft, bool, []artifactstore.Diagnostic)
}

// SourceStateUpdate is the source-derived subset of a record update.
//
// Refresh may update this state, but must not alter record-owned fields.
type SourceStateUpdate struct {
	RecordID           artifactstore.RecordID
	RootID             artifactstore.RootID
	ResolvedDefinition *artifactstore.Digest
	State              State
	Diagnostics        []artifactstore.Diagnostic
	Revision           uint64
	ModifiedAt         time.Time
	ExpectedRevision   uint64
}

func (u SourceStateUpdate) Validate() error {
	if err := artifactstore.ValidateRecordID(u.RecordID); err != nil {
		return err
	}
	if err := artifactstore.ValidateRootID(u.RootID); err != nil {
		return err
	}
	if u.ResolvedDefinition != nil {
		if err := artifactstore.ValidateDigest(*u.ResolvedDefinition); err != nil {
			return err
		}
	}
	if err := validateState(u.State, u.ResolvedDefinition); err != nil {
		return err
	}
	if err := artifactstore.ValidateDiagnostics(u.Diagnostics); err != nil {
		return err
	}
	if u.Revision == 0 {
		return fmt.Errorf(
			"%w: record revision must be positive",
			artifactstore.ErrInvalid,
		)
	}
	if u.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: record modified time is required",
			artifactstore.ErrInvalid,
		)
	}
	if u.ExpectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected record revision is required",
			artifactstore.ErrInvalid,
		)
	}
	if u.Revision != u.ExpectedRevision+1 {
		return fmt.Errorf(
			"%w: record revision must advance by one",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}

type Reconciliation struct {
	Creates     []Record
	Updates     []SourceStateUpdate
	Diagnostics []artifactstore.Diagnostic
}
