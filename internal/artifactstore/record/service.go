package record

import (
	"context"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
)

type Service struct {
	repository  Repository
	definitions definition.Reader
	clock       artifactstore.Clock
}

func NewService(
	repository Repository,
	definitions definition.Reader,
	clock artifactstore.Clock,
) (*Service, error) {
	if repository == nil || definitions == nil || clock == nil {
		return nil, fmt.Errorf(
			"%w: record service dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	return &Service{
		repository:  repository,
		definitions: definitions,
		clock:       clock,
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

func (s *Service) Pin(
	ctx context.Context,
	id artifactstore.RecordID,
	expectedRevision uint64,
	digest artifactstore.Digest,
) (Record, error) {
	if expectedRevision == 0 {
		return Record{}, fmt.Errorf(
			"%w: expected record revision is required",
			artifactstore.ErrInvalid,
		)
	}
	if err := artifactstore.ValidateDigest(digest); err != nil {
		return Record{}, err
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
	definitionValue, err := definition.ReadCanonical(ctx, s.definitions, digest)
	if err != nil {
		return Record{}, err
	}
	if definitionValue.Kind != current.Kind {
		return Record{}, fmt.Errorf(
			"%w: definition kind %q cannot be pinned to record kind %q",
			artifactstore.ErrInvalid,
			definitionValue.Kind,
			current.Kind,
		)
	}
	next := current
	next.Mode = ModePinned
	next.PinnedDefinition = &digest
	next.ResolvedDefinition = &digest
	next.State = StateAvailable
	next.Diagnostics = nil
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

func (s *Service) Follow(
	ctx context.Context,
	id artifactstore.RecordID,
	expectedRevision uint64,
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
	if current.Mode == ModeLinked {
		return current, nil
	}
	next := current
	next.Mode = ModeLinked
	next.PinnedDefinition = nil
	next.State = StateStale
	next.Diagnostics = []artifactstore.Diagnostic{{
		Severity: artifactstore.DiagnosticInfo,
		Code:     "artifact.record.refresh-required",
		Message:  "refresh the root to resolve the current source definition",
	}}
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
