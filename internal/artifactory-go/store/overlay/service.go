package overlay

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

type artifactReader interface {
	Get(ctx context.Context, ref artifactModel.ArtifactRef) (artifactModel.Artifact, error)
}

type cleanupDrainer interface {
	DrainSecretGarbage(ctx context.Context) error
}

// Service owns protected and store-scoped non-secret overlays. It has no
// plaintext secret capability and no Artifact-wide cleanup operation.
type Service struct {
	repository      Repository
	artifacts       artifactReader
	clock           clockutil.Clock
	policy          root.Policy
	namespaces      map[overlayModel.Namespace]struct{}
	storeNamespaces map[overlayModel.Namespace]struct{}
	cleanup         cleanupDrainer
}

func NewService(
	repository Repository,
	artifacts artifactReader,
	timeClock clockutil.Clock,
	policy root.Policy,
	namespaces, storeNamespaces []overlayModel.Namespace,
	cleanup cleanupDrainer,
) (*Service, error) {
	if repository == nil || artifacts == nil || timeClock == nil {
		return nil, fmt.Errorf("%w: Overlay service dependencies are incomplete", spec.ErrInvalid)
	}
	protected, err := normalizeNamespaces("protected overlay", namespaces)
	if err != nil {
		return nil, err
	}
	store, err := normalizeNamespaces("store overlay", storeNamespaces)
	if err != nil {
		return nil, err
	}
	return &Service{
		repository:      repository,
		artifacts:       artifacts,
		clock:           timeClock,
		policy:          policy,
		namespaces:      protected,
		storeNamespaces: store,
		cleanup:         cleanup,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	namespace overlayModel.Namespace,
) (overlayModel.Record, bool, error) {
	if err := s.ready(ctx); err != nil {
		return overlayModel.Record{}, false, err
	}
	if err := ref.Validate(); err != nil {
		return overlayModel.Record{}, false, err
	}
	if err := s.requireNamespace(namespace); err != nil {
		return overlayModel.Record{}, false, err
	}
	target, err := s.availableArtifact(ctx, ref)
	if err != nil {
		return overlayModel.Record{}, false, err
	}
	if err := s.requireProtected(target.RootID); err != nil {
		return overlayModel.Record{}, false, err
	}
	return s.repository.GetOverlay(ctx, ref, namespace)
}

func (s *Service) Put(ctx context.Context, request overlayModel.PutRequest) (overlayModel.Record, error) {
	if err := s.ready(ctx); err != nil {
		return overlayModel.Record{}, err
	}
	if err := request.Validate(); err != nil {
		return overlayModel.Record{}, err
	}
	if err := s.requireNamespace(request.Namespace); err != nil {
		return overlayModel.Record{}, err
	}
	target, err := s.availableArtifact(ctx, request.Artifact)
	if err != nil {
		return overlayModel.Record{}, err
	}
	if err := s.requireProtected(target.RootID); err != nil {
		return overlayModel.Record{}, err
	}
	if target.Revision != request.ExpectedArtifactRevision {
		return overlayModel.Record{}, spec.ErrConflict
	}
	payload, err := overlayModel.CanonicalPayload(request.Payload)
	if err != nil {
		return overlayModel.Record{}, err
	}
	request.Payload = payload
	return s.repository.PutOverlay(ctx, request, clockutil.NowUTC(s.clock))
}

func (s *Service) Delete(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	namespace overlayModel.Namespace,
	expectedArtifactRevision, expectedOverlayRevision uint64,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := s.requireNamespace(namespace); err != nil {
		return err
	}
	if expectedArtifactRevision == 0 || expectedOverlayRevision == 0 {
		return fmt.Errorf("%w: expected Artifact and overlay revisions are required", spec.ErrInvalid)
	}
	target, err := s.availableArtifact(ctx, ref)
	if err != nil {
		return err
	}
	if err := s.requireProtected(target.RootID); err != nil {
		return err
	}
	if target.Revision != expectedArtifactRevision {
		return spec.ErrConflict
	}
	if err := s.repository.DeleteOverlay(
		ctx,
		ref,
		namespace,
		expectedArtifactRevision,
		expectedOverlayRevision,
		clockutil.NowUTC(s.clock),
	); err != nil {
		return err
	}
	s.drainBestEffort(ctx)
	return nil
}

func (s *Service) GetStoreOverlay(
	ctx context.Context,
	namespace overlayModel.Namespace,
) (overlayModel.StoreRecord, bool, error) {
	if err := s.ready(ctx); err != nil {
		return overlayModel.StoreRecord{}, false, err
	}
	if err := s.requireStoreNamespace(namespace); err != nil {
		return overlayModel.StoreRecord{}, false, err
	}
	return s.repository.GetStoreOverlay(ctx, namespace)
}

func (s *Service) PutStoreOverlay(
	ctx context.Context,
	request overlayModel.StorePutRequest,
) (overlayModel.StoreRecord, error) {
	if err := s.ready(ctx); err != nil {
		return overlayModel.StoreRecord{}, err
	}
	if err := request.Validate(); err != nil {
		return overlayModel.StoreRecord{}, err
	}
	if err := s.requireStoreNamespace(request.Namespace); err != nil {
		return overlayModel.StoreRecord{}, err
	}
	payload, err := overlayModel.CanonicalPayload(request.Payload)
	if err != nil {
		return overlayModel.StoreRecord{}, err
	}
	request.Payload = payload
	return s.repository.PutStoreOverlay(ctx, request, clockutil.NowUTC(s.clock))
}

func (s *Service) DeleteStoreOverlay(
	ctx context.Context,
	namespace overlayModel.Namespace,
	expectedRevision uint64,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := s.requireStoreNamespace(namespace); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf("%w: expected store overlay revision is required", spec.ErrInvalid)
	}
	return s.repository.DeleteStoreOverlay(ctx, namespace, expectedRevision)
}

func (s *Service) ready(ctx context.Context) error {
	if s == nil || s.repository == nil || s.artifacts == nil || s.clock == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf("%w: Overlay context is nil", spec.ErrInvalid)
	}
	return ctx.Err()
}

func (s *Service) availableArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	value, err := s.artifacts.Get(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if value.State != artifactModel.StateAvailable {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			value.ID,
		)
	}
	return value, nil
}

func (s *Service) requireProtected(rootID rootModel.RootID) error {
	if s.policy == nil || !s.policy.IsProtectedRoot(rootID) {
		return fmt.Errorf("%w: protected Artifact overlays are valid only in protected Roots", spec.ErrUnsupported)
	}
	return nil
}

func (s *Service) requireNamespace(value overlayModel.Namespace) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if _, found := s.namespaces[value]; !found {
		return fmt.Errorf("%w: protected overlay namespace %q is not registered", spec.ErrUnsupported, value)
	}
	return nil
}

func (s *Service) requireStoreNamespace(value overlayModel.Namespace) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if _, found := s.storeNamespaces[value]; !found {
		return fmt.Errorf("%w: store overlay namespace %q is not registered", spec.ErrUnsupported, value)
	}
	return nil
}

func (s *Service) drainBestEffort(ctx context.Context) {
	if s.cleanup != nil && ctx != nil {
		_ = s.cleanup.DrainSecretGarbage(context.WithoutCancel(ctx))
	}
}

func normalizeNamespaces(label string, values []overlayModel.Namespace) (map[overlayModel.Namespace]struct{}, error) {
	output := make(map[overlayModel.Namespace]struct{}, len(values))
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("%s namespace %d: %w", label, index, err)
		}
		if _, found := output[value]; found {
			return nil, fmt.Errorf("%w: duplicate %s namespace %q", spec.ErrConflict, label, value)
		}
		output[value] = struct{}{}
	}
	return output, nil
}

var (
	_ API      = (*Service)(nil)
	_ StoreAPI = (*Service)(nil)
)
