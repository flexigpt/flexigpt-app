package artifactcleanup

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

type artifactReader interface {
	Get(ctx context.Context, ref artifactModel.ArtifactRef) (artifactModel.Artifact, error)
}

type cleanupDrainer interface {
	DrainSecretGarbage(ctx context.Context) error
}

// Service owns authorization and coordination of atomic Artifact-local overlay
// and binding-reference cleanup. It never deletes the Artifact itself.
type Service struct {
	repository Repository
	artifacts  artifactReader
	policy     root.Policy
	clock      clockutil.Clock
	cleanup    cleanupDrainer
}

func NewService(
	repository Repository,
	artifacts artifactReader,
	policy root.Policy,
	timeClock clockutil.Clock,
	cleanup cleanupDrainer,
) (*Service, error) {
	if repository == nil || artifacts == nil || timeClock == nil {
		return nil, fmt.Errorf("%w: Artifact cleanup dependencies are incomplete", spec.ErrInvalid)
	}
	return &Service{
		repository: repository,
		artifacts:  artifacts,
		policy:     policy,
		clock:      timeClock,
		cleanup:    cleanup,
	}, nil
}

func (s *Service) PurgeArtifactLocalState(ctx context.Context, ref artifactModel.ArtifactRef) error {
	value, err := s.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	_, err = s.CleanupArtifactLocalState(ctx, PurgeRequest{
		Artifact:                 ref,
		ExpectedArtifactRevision: value.Revision,
		AllNamespaces:            true,
	})
	return err
}

func (s *Service) CleanupArtifactLocalState(
	ctx context.Context,
	request PurgeRequest,
) (PurgeResult, error) {
	if err := request.Validate(); err != nil {
		return PurgeResult{}, err
	}
	value, err := s.artifacts.Get(ctx, request.Artifact)
	if err != nil {
		return PurgeResult{}, err
	}
	if s.policy != nil && s.policy.IsProtectedRoot(value.RootID) && !root.IsInstallerPrivileged(ctx) {
		return PurgeResult{}, fmt.Errorf(
			"%w: protected Artifact local-state purge requires installer privilege",
			spec.ErrProtected,
		)
	}
	if value.Revision != request.ExpectedArtifactRevision {
		return PurgeResult{}, spec.ErrConflict
	}
	updated, err := s.repository.CleanupArtifactLocalState(
		ctx,
		request,
		clockutil.NowUTC(s.clock),
	)
	if err != nil {
		return PurgeResult{}, err
	}
	if updated.Ref() != request.Artifact {
		return PurgeResult{}, fmt.Errorf(
			"%w: Artifact cleanup repository returned another Artifact",
			spec.ErrInvalid,
		)
	}
	if s.cleanup != nil {
		_ = s.cleanup.DrainSecretGarbage(context.WithoutCancel(ctx))
	}
	return PurgeResult{Artifact: updated.Clone()}, nil
}
