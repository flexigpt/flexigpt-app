package record

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
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
		root catalog.Root,
		occurrence catalog.Occurrence,
		value definition.Definition,
	) (Draft, bool, []artifactstore.Diagnostic)
}

type UpdatePublication struct {
	Record           Record
	ExpectedRevision uint64
}

type Reconciliation struct {
	Creates     []Record
	Updates     []UpdatePublication
	Diagnostics []artifactstore.Diagnostic
}
