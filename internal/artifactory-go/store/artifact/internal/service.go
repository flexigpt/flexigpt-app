package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definition "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/impl"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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
			spec.ErrInvalid,
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
			spec.ErrDefinitionNotFound,
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
			spec.ErrDigestMismatch,
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
	// Enablement is universal local Artifact metadata. In particular, a user
	// may disable a protected built-in Artifact without receiving write access
	// to its source package, Definition, display metadata, generic Data, or
	// lifecycle operations.
	return s.updateLocal(
		ctx,
		ref,
		expectedRevision,
		false,
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
	if err := spec.ValidateRequiredText(
		"Artifact display name",
		displayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return artifact.Artifact{}, err
	}
	return s.updateLocal(
		ctx,
		ref,
		expectedRevision,
		true,
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
		spec.MaxLocalDataBytes,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	return s.updateLocal(
		ctx,
		ref,
		expectedRevision,
		true,
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
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := rootimpl.RequireMutableRoot(
		ctx,
		s.policy,
		ref.RootID,
	); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Artifact revision is required",
			spec.ErrInvalid,
		)
	}

	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return spec.ErrConflict
	}
	if current.State != artifact.StateMissing {
		return fmt.Errorf(
			"%w: source-backed Artifact %q must be missing before purge",
			spec.ErrConflict,
			current.ID,
		)
	}

	return s.repository.Purge(ctx, ref, expectedRevision)
}

func (s *Service) updateLocal(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	requireMutableRoot bool,
	mutate func(*artifact.Artifact),
) (artifact.Artifact, error) {
	if requireMutableRoot {
		if err := rootimpl.RequireMutableRoot(
			ctx,
			s.policy,
			ref.RootID,
		); err != nil {
			return artifact.Artifact{}, err
		}
	}
	if err := ref.Validate(); err != nil {
		return artifact.Artifact{}, err
	}
	if expectedRevision == 0 {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: expected Artifact revision is required",
			spec.ErrInvalid,
		)
	}

	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if current.Revision != expectedRevision {
		return artifact.Artifact{}, spec.ErrConflict
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
			spec.ErrInvalid,
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
