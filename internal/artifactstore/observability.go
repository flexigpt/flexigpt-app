package artifactstore

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

// ListTransferProvenance returns app-local transfer audit entries for an
// active record. Portable definition bodies are not included.
func (s *Store) ListTransferProvenance(
	ctx context.Context,
	recordID spec.RecordID,
) ([]spec.TransferProvenance, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	if _, err := s.GetRecord(ctx, recordID); err != nil {
		return nil, err
	}
	return s.repository.ListTransferProvenance(ctx, recordID)
}

// ListDependencySnapshots returns retained dependency-resolution snapshots for
// an active record. Callers must inspect CatalogGeneration to determine which
// snapshot set they need.
func (s *Store) ListDependencySnapshots(
	ctx context.Context,
	recordID spec.RecordID,
) ([]spec.ArtifactDependencySnapshot, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	if _, err := s.GetRecord(ctx, recordID); err != nil {
		return nil, err
	}
	return s.repository.ListDependencySnapshots(ctx, recordID)
}
