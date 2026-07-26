package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
)

type Reader interface {
	Get(
		ctx context.Context,
		ref artifactstore.ArtifactRef,
	) (Artifact, error)

	ListByCollection(
		ctx context.Context,
		ref artifactstore.CollectionRef,
	) ([]Artifact, error)

	ListSuppressions(
		ctx context.Context,
		ref artifactstore.CollectionRef,
	) ([]Suppression, error)
}

type Repository interface {
	Reader

	CreateAdopted(
		ctx context.Context,
		value Artifact,
		expectedCollectionRevision uint64,
		expectedCatalogRevision uint64,
	) error

	CreatePinned(
		ctx context.Context,
		value Artifact,
		expectedCollectionRevision uint64,
		expectedCatalogRevision uint64,
	) error

	Update(
		ctx context.Context,
		value Artifact,
		expectedRevision uint64,
	) error

	Unadopt(
		ctx context.Context,
		ref artifactstore.ArtifactRef,
		expectedRevision uint64,
		suppression *Suppression,
	) error

	Suppress(
		ctx context.Context,
		value Suppression,
		expectedCollectionRevision uint64,
	) error

	Unsuppress(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		binding artifactstore.SourceBinding,
		expectedRevision uint64,
	) error

	Purge(
		ctx context.Context,
		ref artifactstore.ArtifactRef,
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
		value collection.Collection,
		occurrence catalog.Occurrence,
		def definition.Definition,
	) (Draft, bool, []artifactstore.Diagnostic)
}

type SourceStateUpdate struct {
	ArtifactID         artifactstore.ArtifactID
	RootID             artifactstore.RootID
	CollectionID       artifactstore.CollectionID
	ResolvedDefinition *artifactstore.Digest
	State              State
	Diagnostics        []artifactstore.Diagnostic
	Revision           uint64
	ModifiedAt         time.Time
	ExpectedRevision   uint64
}

func (u SourceStateUpdate) Validate() error {
	if err := artifactstore.ValidateArtifactID(u.ArtifactID); err != nil {
		return err
	}
	if err := artifactstore.ValidateRootID(u.RootID); err != nil {
		return err
	}
	if err := artifactstore.ValidateCollectionID(u.CollectionID); err != nil {
		return err
	}
	if u.ResolvedDefinition != nil {
		if err := artifactstore.ValidateDigest(*u.ResolvedDefinition); err != nil {
			return err
		}
	}
	switch u.State {
	case StateAvailable, StateIncompatible:
		if u.ResolvedDefinition == nil {
			return fmt.Errorf(
				"%w: artifact update state requires a definition",
				artifactstore.ErrInvalid,
			)
		}

	case StateMissing, StateInvalid:
		if u.ResolvedDefinition != nil {
			return fmt.Errorf(
				"%w: missing or invalid artifact state cannot retain a definition",
				artifactstore.ErrInvalid,
			)
		}

	default:
		return fmt.Errorf("%w: invalid artifact update state", artifactstore.ErrInvalid)
	}
	if err := artifactstore.ValidateDiagnostics(u.Diagnostics); err != nil {
		return err
	}
	if u.ExpectedRevision == 0 ||
		u.Revision != u.ExpectedRevision+1 ||
		u.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: invalid source-derived artifact update",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}

type Reconciliation struct {
	Creates     []Artifact
	Updates     []SourceStateUpdate
	Diagnostics []artifactstore.Diagnostic
}
