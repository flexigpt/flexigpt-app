package localstate

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/secretapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/secret"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

const secretCleanupBatchSize = 128

// Service owns generic Artifact Store local-state behavior:
//
//   - protected-root non-secret overlays;
//   - Artifact-local secret refs and SHA metadata;
//   - staged physical secret publication;
//   - durable physical-secret cleanup.
//
// It deliberately does not know Model, MCP, Tool, Skill, source adapters,
// package layouts, or source-definition schemas.
type Service struct {
	repository Repository
	artifacts  ArtifactReader
	clock      clockutil.Clock
	policy     root.RootPolicy
	values     secretapi.ValueStore

	namespaces      map[overlay.Namespace]struct{}
	storeNamespaces map[overlay.Namespace]struct{}

	closed    atomic.Bool
	closeOnce sync.Once
	closeErr  error
}

func NewService(
	repository Repository,
	artifacts ArtifactReader,
	timeClock clockutil.Clock,
	policy root.RootPolicy,
	namespaces []overlay.Namespace,
	storeNamespaces []overlay.Namespace,
	values secretapi.ValueStore,
) (*Service, error) {
	if repository == nil || artifacts == nil || timeClock == nil {
		return nil, fmt.Errorf(
			"%w: Artifact local-state dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if values != nil {
		if err := secretapi.ValidateValueStore(values); err != nil {
			return nil, err
		}
	}

	registered := make(
		map[overlay.Namespace]struct{},
		len(namespaces),
	)
	for index, namespace := range namespaces {
		if err := namespace.Validate(); err != nil {
			return nil, fmt.Errorf(
				"protected overlay namespace %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := registered[namespace]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate protected overlay namespace %q",
				basespec.ErrConflict,
				namespace,
			)
		}
		registered[namespace] = struct{}{}
	}

	storeRegistered := make(
		map[overlay.Namespace]struct{},
		len(storeNamespaces),
	)
	for index, namespace := range storeNamespaces {
		if err := namespace.Validate(); err != nil {
			return nil, fmt.Errorf(
				"store overlay namespace %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := storeRegistered[namespace]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate store overlay namespace %q",
				basespec.ErrConflict,
				namespace,
			)
		}
		storeRegistered[namespace] = struct{}{}
	}

	return &Service{
		repository:      repository,
		artifacts:       artifacts,
		clock:           timeClock,
		policy:          policy,
		values:          values,
		namespaces:      registered,
		storeNamespaces: storeRegistered,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	ref artifact.ArtifactRef,
	namespace overlay.Namespace,
) (overlay.Record, bool, error) {
	if err := s.ready(ctx); err != nil {
		return overlay.Record{}, false, err
	}
	if err := ref.Validate(); err != nil {
		return overlay.Record{}, false, err
	}
	if err := s.requireNamespace(namespace); err != nil {
		return overlay.Record{}, false, err
	}

	target, err := s.availableArtifact(ctx, ref)
	if err != nil {
		return overlay.Record{}, false, err
	}
	if err := s.requireProtected(target); err != nil {
		return overlay.Record{}, false, err
	}

	return s.repository.GetOverlay(ctx, ref, namespace)
}

func (s *Service) Put(
	ctx context.Context,
	request overlay.PutRequest,
) (overlay.Record, error) {
	if err := s.ready(ctx); err != nil {
		return overlay.Record{}, err
	}
	if err := request.Validate(); err != nil {
		return overlay.Record{}, err
	}
	if err := s.requireNamespace(request.Namespace); err != nil {
		return overlay.Record{}, err
	}

	target, err := s.availableArtifact(ctx, request.Artifact)
	if err != nil {
		return overlay.Record{}, err
	}
	if err := s.requireProtected(target); err != nil {
		return overlay.Record{}, err
	}
	if target.Revision != request.ExpectedArtifactRevision {
		return overlay.Record{}, basespec.ErrConflict
	}

	payload, err := overlay.CanonicalPayload(request.Payload)
	if err != nil {
		return overlay.Record{}, err
	}
	request.Payload = payload

	return s.repository.PutOverlay(
		ctx,
		request,
		clockutil.NowUTC(s.clock),
	)
}

// Delete removes one complete protected overlay and every secret binding under
// the same Artifact/namespace pair. Detached physical secret values are placed
// in the durable cleanup queue before the metadata transaction commits.
func (s *Service) Delete(
	ctx context.Context,
	ref artifact.ArtifactRef,
	namespace overlay.Namespace,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
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
	if expectedArtifactRevision == 0 ||
		expectedOverlayRevision == 0 {
		return fmt.Errorf(
			"%w: expected Artifact and overlay revisions are required",
			basespec.ErrInvalid,
		)
	}

	target, err := s.availableArtifact(ctx, ref)
	if err != nil {
		return err
	}
	if err := s.requireProtected(target); err != nil {
		return err
	}
	if target.Revision != expectedArtifactRevision {
		return basespec.ErrConflict
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

func (s *Service) GetBinding(
	ctx context.Context,
	key secret.BindingKey,
) (secret.Binding, bool, error) {
	if err := s.ready(ctx); err != nil {
		return secret.Binding{}, false, err
	}
	if err := key.Validate(); err != nil {
		return secret.Binding{}, false, err
	}
	if err := s.requireNamespace(key.Namespace); err != nil {
		return secret.Binding{}, false, err
	}
	if _, err := s.availableArtifact(ctx, key.Artifact); err != nil {
		return secret.Binding{}, false, err
	}

	return s.repository.GetBinding(ctx, key)
}

func (s *Service) ReplaceBinding(
	ctx context.Context,
	request secret.ReplaceBindingRequest,
) (secret.Binding, error) {
	if err := s.ready(ctx); err != nil {
		return secret.Binding{}, err
	}
	if err := request.Validate(); err != nil {
		return secret.Binding{}, err
	}
	if err := s.requireNamespace(request.Key.Namespace); err != nil {
		return secret.Binding{}, err
	}
	if s.values == nil {
		return secret.Binding{}, fmt.Errorf(
			"%w: Artifact Store secret value backend is not configured",
			basespec.ErrUnsupported,
		)
	}

	target, err := s.availableArtifact(ctx, request.Key.Artifact)
	if err != nil {
		return secret.Binding{}, err
	}
	if target.Revision != request.ExpectedArtifactRevision {
		return secret.Binding{}, basespec.ErrConflict
	}

	now := clockutil.NowUTC(s.clock)
	record := secret.Record{
		Ref:        secret.NewRef(),
		StoreName:  s.values.Name(),
		SHA256:     secret.SHA256(request.Value),
		State:      secret.RecordStatePending,
		CreatedAt:  now,
		ModifiedAt: now,
	}
	if err := record.Validate(); err != nil {
		return secret.Binding{}, err
	}

	if err := s.repository.CreatePendingSecret(ctx, record); err != nil {
		return secret.Binding{}, err
	}

	if err := s.values.Put(ctx, record.Ref, request.Value); err != nil {
		s.queuePendingBestEffort(ctx, record.Ref)
		return secret.Binding{}, err
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
		return secret.Binding{}, err
	}

	s.drainBestEffort(ctx)
	return binding.Clone(), nil
}

func (s *Service) ClearBinding(
	ctx context.Context,
	request secret.ClearBindingRequest,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := request.Validate(); err != nil {
		return err
	}
	if err := s.requireNamespace(request.Key.Namespace); err != nil {
		return err
	}

	target, err := s.availableArtifact(ctx, request.Key.Artifact)
	if err != nil {
		return err
	}
	if target.Revision != request.ExpectedArtifactRevision {
		return basespec.ErrConflict
	}

	if err := s.repository.ClearSecretBinding(
		ctx,
		request,
		clockutil.NowUTC(s.clock),
	); err != nil {
		return err
	}
	s.drainBestEffort(ctx)
	return nil
}

// ReadBinding resolves one active Artifact-local binding through the physical
// secret backend. It is intended only for trusted runtime composition.
func (s *Service) ReadBinding(
	ctx context.Context,
	key secret.BindingKey,
	expectedRef secret.Ref,
) (string, secret.Binding, error) {
	if err := s.ready(ctx); err != nil {
		return "", secret.Binding{}, err
	}
	if err := key.Validate(); err != nil {
		return "", secret.Binding{}, err
	}
	if err := expectedRef.Validate(); err != nil {
		return "", secret.Binding{}, err
	}
	if err := s.requireNamespace(key.Namespace); err != nil {
		return "", secret.Binding{}, err
	}
	if s.values == nil {
		return "", secret.Binding{}, fmt.Errorf(
			"%w: Artifact Store secret value backend is not configured",
			basespec.ErrUnsupported,
		)
	}
	if _, err := s.availableArtifact(ctx, key.Artifact); err != nil {
		return "", secret.Binding{}, err
	}

	binding, found, err := s.repository.GetBinding(ctx, key)
	if err != nil {
		return "", secret.Binding{}, err
	}
	if !found || !binding.Active() {
		return "", secret.Binding{}, fmt.Errorf(
			"%w: secret binding is not configured",
			basespec.ErrReferenceUnresolved,
		)
	}
	if *binding.Ref != expectedRef {
		return "", secret.Binding{}, fmt.Errorf(
			"%w: secret binding changed during runtime resolution",
			basespec.ErrConflict,
		)
	}

	value, err := s.values.Get(ctx, expectedRef)
	if err != nil {
		return "", secret.Binding{}, err
	}
	if secret.SHA256(value) != binding.SHA256 {
		return "", secret.Binding{}, fmt.Errorf(
			"%w: physical secret value does not match binding SHA-256",
			basespec.ErrDigestMismatch,
		)
	}
	return value, binding.Clone(), nil
}

// PurgeArtifactLocalState is an internal lifecycle operation. It removes every
// protected overlay and every secret binding attached to the exact Artifact.
// It does not delete the Artifact itself.
//
// Mutable Roots may use this after an explicit managed Artifact deletion.
// Protected Roots require trusted installer privilege.
func (s *Service) PurgeArtifactLocalState(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}

	target, err := s.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if s.policy != nil &&
		s.policy.IsProtectedRoot(target.RootID) &&
		!installerapi.IsPrivileged(ctx) {
		return fmt.Errorf(
			"%w: protected Artifact local-state purge requires installer privilege",
			basespec.ErrProtected,
		)
	}

	if err := s.repository.PurgeArtifactLocalState(
		ctx,
		ref,
		clockutil.NowUTC(s.clock),
	); err != nil {
		return err
	}
	s.drainBestEffort(ctx)
	return nil
}

// DrainSecretGarbage performs one bounded cleanup pass. Logical detachment is
// already durable before this method is called, so physical cleanup failures
// never restore or reactivate a detached binding.
func (s *Service) DrainSecretGarbage(
	ctx context.Context,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if s.values == nil {
		return fmt.Errorf(
			"%w: Artifact Store secret value backend is not configured",
			basespec.ErrUnsupported,
		)
	}

	values, err := s.repository.ListSecretCleanup(
		ctx,
		secretCleanupBatchSize,
	)
	if err != nil {
		return err
	}

	var output error
	for _, value := range values {
		if value.StoreName != s.values.Name() {
			_ = s.repository.RecordSecretCleanupFailure(
				context.WithoutCancel(ctx),
				value.Ref,
				"configured secret backend does not own cleanup record",
				clockutil.NowUTC(s.clock),
			)
			output = errors.Join(
				output,
				fmt.Errorf(
					"%w: cleanup record belongs to secret backend %q",
					basespec.ErrUnsupported,
					value.StoreName,
				),
			)
			continue
		}

		if err := s.values.Delete(ctx, value.Ref); err != nil {
			_ = s.repository.RecordSecretCleanupFailure(
				context.WithoutCancel(ctx),
				value.Ref,
				"physical secret deletion failed",
				clockutil.NowUTC(s.clock),
			)
			output = errors.Join(output, err)
			continue
		}

		if err := s.repository.CompleteSecretCleanup(
			ctx,
			value.Ref,
		); err != nil {
			output = errors.Join(output, err)
		}
	}
	return output
}

// RecoverPending marks interrupted staged writes for cleanup. It intentionally
// does not attempt to publish a pending value after process restart because
// the caller's expected revisions may no longer be current.
func (s *Service) RecoverPending(
	ctx context.Context,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := s.repository.RecoverPendingSecrets(
		ctx,
		clockutil.NowUTC(s.clock),
	); err != nil {
		return err
	}
	s.drainBestEffort(ctx)
	return nil
}

func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		if s.values != nil {
			s.closeErr = s.values.Close()
		}
	})
	return s.closeErr
}

func (s *Service) ready(ctx context.Context) error {
	if s == nil ||
		s.repository == nil ||
		s.artifacts == nil ||
		s.clock == nil ||
		s.closed.Load() {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Artifact local-state context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}

func (s *Service) availableArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	value, err := s.artifacts.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if value.State != artifact.StateAvailable {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			value.ID,
		)
	}
	return value, nil
}

func (s *Service) requireProtected(
	value artifact.Artifact,
) error {
	if s.policy == nil ||
		!s.policy.IsProtectedRoot(value.RootID) {
		return fmt.Errorf(
			"%w: protected Artifact overlays are valid only in protected Roots",
			basespec.ErrUnsupported,
		)
	}
	return nil
}

func (s *Service) requireNamespace(
	namespace overlay.Namespace,
) error {
	if err := namespace.Validate(); err != nil {
		return err
	}
	if _, found := s.namespaces[namespace]; !found {
		return fmt.Errorf(
			"%w: protected overlay namespace %q is not registered",
			basespec.ErrUnsupported,
			namespace,
		)
	}
	return nil
}

func (s *Service) queuePendingBestEffort(
	ctx context.Context,
	ref secret.Ref,
) {
	_ = s.repository.QueueSecretForCleanup(
		context.WithoutCancel(ctx),
		ref,
		clockutil.NowUTC(s.clock),
	)
	s.drainBestEffort(ctx)
}

func (s *Service) drainBestEffort(
	ctx context.Context,
) {
	if s == nil || s.values == nil || ctx == nil {
		return
	}
	_ = s.DrainSecretGarbage(context.WithoutCancel(ctx))
}
