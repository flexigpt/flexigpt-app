package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Service struct {
	repository  artifact.Repository
	definitions artifact.DefinitionReader
	clock       clockutil.Clock
	policy      rootModel.RootPolicy
}

func NewService(
	repository artifact.Repository,
	definitions artifact.DefinitionReader,
	timeClock clockutil.Clock,
	policy rootModel.RootPolicy,
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
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	if err := ref.Validate(); err != nil {
		return artifactModel.Artifact{}, err
	}
	return s.repository.Get(ctx, ref)
}

func (s *Service) FindByOrigin(
	ctx context.Context,
	rootID rootModel.RootID,
	binding artifactModel.SourceBinding,
	kind artifactModel.ArtifactKind,
) (artifactModel.Artifact, error) {
	if err := rootID.Validate(); err != nil {
		return artifactModel.Artifact{}, err
	}
	if err := binding.Validate(); err != nil {
		return artifactModel.Artifact{}, err
	}
	if err := kind.Validate(); err != nil {
		return artifactModel.Artifact{}, err
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
	ref artifactModel.ArtifactRef,
) (definitionModel.Definition, error) {
	record, err := s.Get(ctx, ref)
	if err != nil {
		return definitionModel.Definition{}, err
	}
	if record.ResolvedDefinition == nil {
		return definitionModel.Definition{}, fmt.Errorf(
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
		return definitionModel.Definition{}, err
	}
	if value.Digest != *record.ResolvedDefinition ||
		(record.State != artifactModel.StateIncompatible &&
			value.Kind != record.Kind) {
		return definitionModel.Definition{}, fmt.Errorf(
			"%w: Artifact Definition does not match Artifact state",
			spec.ErrDigestMismatch,
		)
	}
	return value.Clone(), nil
}

func (s *Service) SetEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	// Enablement is universal local Artifact metadata. In particular, a user
	// may disable a protected built-in Artifact without receiving write access
	// to its source package, Definition, display metadata, generic Data, or
	// lifecycle operations.
	return s.updateLocal(
		ctx,
		ref,
		expectedRevision,
		false,
		func(value *artifactModel.Artifact) {
			value.Enabled = enabled
		},
	)
}

func (s *Service) SetDisplayName(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	displayName string,
) (artifactModel.Artifact, error) {
	if err := spec.ValidateRequiredText(
		"Artifact display name",
		displayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return artifactModel.Artifact{}, err
	}
	return s.updateLocal(
		ctx,
		ref,
		expectedRevision,
		true,
		func(value *artifactModel.Artifact) {
			value.DisplayName = displayName
		},
	)
}

func (s *Service) UpdateData(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	data json.RawMessage,
) (artifactModel.Artifact, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		data,
		spec.MaxLocalDataBytes,
	)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	return s.updateLocal(
		ctx,
		ref,
		expectedRevision,
		true,
		func(value *artifactModel.Artifact) {
			value.Data = json.RawMessage(canonical)
		},
	)
}

func (s *Service) Purge(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := root.RequireMutableRoot(
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
	if current.State != artifactModel.StateMissing {
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
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	requireMutableRoot bool,
	mutate func(*artifactModel.Artifact),
) (artifactModel.Artifact, error) {
	if requireMutableRoot {
		if err := root.RequireMutableRoot(
			ctx,
			s.policy,
			ref.RootID,
		); err != nil {
			return artifactModel.Artifact{}, err
		}
	}
	if err := ref.Validate(); err != nil {
		return artifactModel.Artifact{}, err
	}
	if expectedRevision == 0 {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: expected Artifact revision is required",
			spec.ErrInvalid,
		)
	}

	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if current.Revision != expectedRevision {
		return artifactModel.Artifact{}, spec.ErrConflict
	}

	next := current.Clone()
	mutate(&next)
	if current.Enabled == next.Enabled &&
		current.DisplayName == next.DisplayName &&
		bytes.Equal(current.Data, next.Data) {
		return current, nil
	}
	if current.Revision == ^uint64(0) {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: Artifact revision is exhausted",
			spec.ErrInvalid,
		)
	}
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return artifactModel.Artifact{}, err
	}
	if err := s.repository.UpdateLocal(
		ctx,
		next,
		expectedRevision,
	); err != nil {
		return artifactModel.Artifact{}, err
	}
	return next.Clone(), nil
}
