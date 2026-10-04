package secret

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/value"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

type bindingArtifactReader interface {
	Get(ctx context.Context, a artifactModel.ArtifactRef) (artifactModel.Artifact, error)
}

// BindingService implements public-safe binding metadata and staged physical
// replacement. It deliberately has neither plaintext reads nor lifecycle APIs.
type BindingService struct {
	repository BindingRepository
	artifacts  bindingArtifactReader
	clock      clockutil.Clock
	values     value.ValueStore
	namespaces map[overlayModel.Namespace]struct{}
	cleanup    CleanupAPI
}

func NewBindingService(
	repository BindingRepository,
	artifacts bindingArtifactReader,
	timeClock clockutil.Clock,
	namespaces []overlayModel.Namespace,
	values value.ValueStore,
	cleanup CleanupAPI,
) (*BindingService, error) {
	if repository == nil || artifacts == nil || timeClock == nil {
		return nil, fmt.Errorf("%w: secret binding dependencies are incomplete", spec.ErrInvalid)
	}
	if values != nil {
		if err := value.ValidateValueStore(values); err != nil {
			return nil, err
		}
	}
	registered, err := namespaceSet(namespaces)
	if err != nil {
		return nil, err
	}
	return &BindingService{
		repository: repository,
		artifacts:  artifacts,
		clock:      timeClock,
		values:     values,
		namespaces: registered,
		cleanup:    cleanup,
	}, nil
}

func (s *BindingService) GetBinding(
	ctx context.Context,
	key secretModel.BindingKey,
) (secretModel.Binding, bool, error) {
	if err := s.ready(ctx); err != nil {
		return secretModel.Binding{}, false, err
	}
	if err := key.Validate(); err != nil {
		return secretModel.Binding{}, false, err
	}
	if err := requireNamespace(s.namespaces, key.Namespace); err != nil {
		return secretModel.Binding{}, false, err
	}
	if _, err := s.availableArtifact(ctx, key.Artifact); err != nil {
		return secretModel.Binding{}, false, err
	}
	return s.repository.GetBinding(ctx, key)
}

func (s *BindingService) ReplaceBinding(
	ctx context.Context,
	request secretModel.ReplaceBindingRequest,
) (secretModel.Binding, error) {
	if err := s.ready(ctx); err != nil {
		return secretModel.Binding{}, err
	}
	if err := request.Validate(); err != nil {
		return secretModel.Binding{}, err
	}
	if err := requireNamespace(s.namespaces, request.Key.Namespace); err != nil {
		return secretModel.Binding{}, err
	}
	if s.values == nil {
		return secretModel.Binding{}, fmt.Errorf(
			"%w: Artifact Store secret value backend is not configured",
			spec.ErrUnsupported,
		)
	}
	target, err := s.availableArtifact(ctx, request.Key.Artifact)
	if err != nil {
		return secretModel.Binding{}, err
	}
	if target.Revision != request.ExpectedArtifactRevision {
		return secretModel.Binding{}, spec.ErrConflict
	}
	now := clockutil.NowUTC(s.clock)
	record := secretModel.Record{
		Ref:        secretModel.NewRef(),
		StoreName:  s.values.Name(),
		SHA256:     secretModel.SHA256(request.Value),
		State:      secretModel.RecordStatePending,
		CreatedAt:  now,
		ModifiedAt: now,
	}
	if err := record.Validate(); err != nil {
		return secretModel.Binding{}, err
	}
	if err := s.repository.CreatePendingSecret(ctx, record); err != nil {
		return secretModel.Binding{}, err
	}
	if err := s.values.Put(ctx, record.Ref, request.Value); err != nil {
		s.queuePendingBestEffort(ctx, record.Ref)
		return secretModel.Binding{}, err
	}
	binding, err := s.repository.AttachSecretBinding(
		ctx,
		AttachBindingRequest{
			Key:                      request.Key,
			ExpectedArtifactRevision: request.ExpectedArtifactRevision,
			ExpectedBindingRevision:  request.ExpectedBindingRevision,
			Record:                   record,
		},
		clockutil.NowUTC(s.clock),
	)
	if err != nil {
		s.queuePendingBestEffort(ctx, record.Ref)
		return secretModel.Binding{}, err
	}
	s.drainBestEffort(ctx)
	return binding.Clone(), nil
}

func (s *BindingService) ClearBinding(ctx context.Context, request secretModel.ClearBindingRequest) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := request.Validate(); err != nil {
		return err
	}
	if err := requireNamespace(s.namespaces, request.Key.Namespace); err != nil {
		return err
	}
	target, err := s.availableArtifact(ctx, request.Key.Artifact)
	if err != nil {
		return err
	}
	if target.Revision != request.ExpectedArtifactRevision {
		return spec.ErrConflict
	}
	if err := s.repository.ClearSecretBinding(ctx, request, clockutil.NowUTC(s.clock)); err != nil {
		return err
	}
	s.drainBestEffort(ctx)
	return nil
}

func (s *BindingService) ready(ctx context.Context) error {
	if s == nil || s.repository == nil || s.artifacts == nil || s.clock == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf("%w: secret binding context is nil", spec.ErrInvalid)
	}
	return ctx.Err()
}

func (s *BindingService) availableArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	v, err := s.artifacts.Get(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if v.State != artifactModel.StateAvailable {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			v.ID,
		)
	}
	return v, nil
}

func (s *BindingService) queuePendingBestEffort(ctx context.Context, ref secretModel.Ref) {
	_ = s.repository.QueueSecretForCleanup(context.WithoutCancel(ctx), ref, clockutil.NowUTC(s.clock))
	s.drainBestEffort(ctx)
}

func (s *BindingService) drainBestEffort(ctx context.Context) {
	if s.cleanup != nil && ctx != nil {
		_ = s.cleanup.DrainSecretGarbage(context.WithoutCancel(ctx))
	}
}

var _ API = (*BindingService)(nil)
