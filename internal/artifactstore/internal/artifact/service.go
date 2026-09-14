package artifactimpl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/root"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Service struct {
	repository  Repository
	definitions DefinitionReader
	clock       clockutil.Clock
	policy      root.RootPolicy
}

func NewService(
	repository Repository,
	definitions DefinitionReader,
	timeClock clockutil.Clock,
	policy root.RootPolicy,
) (*Service, error) {
	if repository == nil ||
		definitions == nil ||
		timeClock == nil {
		return nil, fmt.Errorf(
			"%w: Artifact service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		repository:  repository,
		definitions: definitions,
		clock:       timeClock,
		policy:      policy,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	if err := ref.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	return s.repository.Get(ctx, ref)
}

func (s *Service) ListByRoot(
	ctx context.Context,
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	return s.repository.ListByRoot(ctx, rootID)
}

func (s *Service) ListBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]artifact.Artifact, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := sourceID.Validate(); err != nil {
		return nil, err
	}
	return s.repository.ListBySource(ctx, rootID, sourceID)
}

func (s *Service) FindByIdentity(
	ctx context.Context,
	rootID root.RootID,
	kind artifact.ArtifactKind,
	logicalName basespec.LogicalName,
) ([]artifact.Artifact, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := kind.Validate(); err != nil {
		return nil, err
	}
	if err := logicalName.Validate(); err != nil {
		return nil, err
	}
	return s.repository.FindByIdentity(
		ctx,
		rootID,
		kind,
		logicalName,
	)
}

func (s *Service) FindByOrigin(
	ctx context.Context,
	rootID root.RootID,
	binding artifact.SourceBinding,
	kind artifact.ArtifactKind,
) (artifact.Artifact, error) {
	if err := rootID.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := binding.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := kind.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	return s.repository.FindByOrigin(
		ctx,
		rootID,
		binding,
		kind,
	)
}

func (s *Service) GetDefinition(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (definition.Definition, error) {
	record, err := s.Get(ctx, ref)
	if err != nil {
		return definition.Definition{}, err
	}
	if record.ResolvedDefinition == nil {
		return definition.Definition{}, fmt.Errorf(
			"%w: Artifact %q has no current Definition",
			basespec.ErrDefinitionNotFound,
			record.ID,
		)
	}
	value, err := s.definitions.GetDefinition(
		ctx,
		record.RootID,
		*record.ResolvedDefinition,
	)
	if err != nil {
		return definition.Definition{}, err
	}
	if value.Digest != *record.ResolvedDefinition ||
		(record.State != artifact.StateIncompatible &&
			value.Kind != record.Kind) {
		return definition.Definition{}, fmt.Errorf(
			"%w: Artifact Definition does not match Artifact state",
			basespec.ErrDigestMismatch,
		)
	}
	return value.Clone(), nil
}

func (s *Service) SetEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	return s.updateLocal(
		ctx,
		ref,
		expectedRevision,
		func(value *artifact.Artifact) {
			value.Enabled = enabled
		},
	)
}

func (s *Service) SetDisplayName(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	displayName string,
) (artifact.Artifact, error) {
	if err := basespec.ValidateRequiredText(
		"Artifact display name",
		displayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return artifact.Artifact{}, err
	}
	return s.updateLocal(
		ctx,
		ref,
		expectedRevision,
		func(value *artifact.Artifact) {
			value.DisplayName = displayName
		},
	)
}

func (s *Service) UpdateData(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	data json.RawMessage,
) (artifact.Artifact, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		data,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	return s.updateLocal(
		ctx,
		ref,
		expectedRevision,
		func(value *artifact.Artifact) {
			value.Data = json.RawMessage(canonical)
		},
	)
}

func (s *Service) Purge(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := rootimpl.RequireMutableRoot(
		ctx,
		s.policy,
		ref.RootID,
	); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	return s.repository.Purge(ctx, ref, expectedRevision)
}

func (s *Service) updateLocal(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	mutate func(*artifact.Artifact),
) (artifact.Artifact, error) {
	if err := rootimpl.RequireMutableRoot(
		ctx,
		s.policy,
		ref.RootID,
	); err != nil {
		return artifact.Artifact{}, err
	}
	if err := ref.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if expectedRevision == 0 {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: expected Artifact revision is required",
			basespec.ErrInvalid,
		)
	}

	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if current.Revision != expectedRevision {
		return artifact.Artifact{}, basespec.ErrConflict
	}

	next := current.Clone()
	mutate(&next)
	if current.Enabled == next.Enabled &&
		current.DisplayName == next.DisplayName &&
		bytes.Equal(current.Data, next.Data) {
		return current, nil
	}
	if current.Revision == ^uint64(0) {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact revision is exhausted",
			basespec.ErrInvalid,
		)
	}
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if err := s.repository.UpdateLocal(
		ctx,
		next,
		expectedRevision,
	); err != nil {
		return artifact.Artifact{}, err
	}
	return next.Clone(), nil
}
