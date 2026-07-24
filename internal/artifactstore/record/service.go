package record

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Service struct {
	repository Repository
	clock      artifactstore.Clock
}

func NewService(
	repository Repository,
	clock artifactstore.Clock,
) (*Service, error) {
	if repository == nil || clock == nil {
		return nil, fmt.Errorf(
			"%w: record service dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	return &Service{
		repository: repository,
		clock:      clock,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	id artifactstore.RecordID,
) (Record, error) {
	return s.repository.Get(ctx, id)
}

func (s *Service) ListByRoot(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]Record, error) {
	return s.repository.ListByRoot(ctx, rootID)
}

func (s *Service) SetEnabled(
	ctx context.Context,
	id artifactstore.RecordID,
	expectedRevision uint64,
	enabled bool,
) (Record, error) {
	if expectedRevision == 0 {
		return Record{}, fmt.Errorf(
			"%w: expected record revision is required",
			artifactstore.ErrInvalid,
		)
	}
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return Record{}, err
	}
	if current.Revision != expectedRevision {
		return Record{}, fmt.Errorf(
			"%w: record %q changed since it was read",
			artifactstore.ErrConflict,
			id,
		)
	}
	if current.Enabled == enabled {
		return current, nil
	}
	next := current
	next.Enabled = enabled
	next.Revision++
	next.ModifiedAt = s.nextTime(current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return Record{}, err
	}
	if err := s.repository.Update(ctx, next, expectedRevision); err != nil {
		return Record{}, err
	}
	return next, nil
}

// UpdateData replaces the record-local JSON object while preserving all
// source-derived state. The caller owns the data schema and must provide the
// expected record revision.
func (s *Service) UpdateData(
	ctx context.Context,
	id artifactstore.RecordID,
	expectedRevision uint64,
	data json.RawMessage,
) (Record, error) {
	if expectedRevision == 0 {
		return Record{}, fmt.Errorf(
			"%w: expected record revision is required",
			artifactstore.ErrInvalid,
		)
	}
	canonical, err := jsoncanon.CanonicalizeObject(
		data,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return Record{}, fmt.Errorf("%w: record data: %w", artifactstore.ErrInvalid, err)
	}
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return Record{}, err
	}
	if current.Revision != expectedRevision {
		return Record{}, fmt.Errorf(
			"%w: record %q changed since it was read",
			artifactstore.ErrConflict,
			id,
		)
	}
	if jsoncanon.Equal(current.Data, canonical) {
		return current, nil
	}
	next := current
	next.Data = json.RawMessage(canonical)
	next.Revision++
	next.ModifiedAt = s.nextTime(current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return Record{}, err
	}
	if err := s.repository.Update(ctx, next, expectedRevision); err != nil {
		return Record{}, err
	}
	return next, nil
}

func (s *Service) Delete(
	ctx context.Context,
	id artifactstore.RecordID,
	expectedRevision uint64,
) error {
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected record revision is required",
			artifactstore.ErrInvalid,
		)
	}
	return s.repository.Delete(ctx, id, expectedRevision)
}

func (s *Service) nextTime(previous time.Time) time.Time {
	next := s.clock.Now().UTC()
	if !next.After(previous) {
		return previous.Add(1)
	}
	return next
}
