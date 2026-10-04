package secret

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/value"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

const secretCleanupBatchSize = 128

// CleanupAPI is a narrow best-effort post-commit cleanup capability.
type CleanupAPI interface {
	DrainSecretGarbage(ctx context.Context) error
}

// LifecycleService owns pending-write recovery and durable physical cleanup.
// It borrows its ValueStore; deployment owns physical backend closure.
type LifecycleService struct {
	repository LifecycleRepository
	clock      clockutil.Clock
	values     value.ValueStore
}

func NewLifecycleService(
	repository LifecycleRepository,
	timeClock clockutil.Clock,
	values value.ValueStore,
) (*LifecycleService, error) {
	if repository == nil || timeClock == nil {
		return nil, fmt.Errorf("%w: secret lifecycle dependencies are incomplete", spec.ErrInvalid)
	}
	if values != nil {
		if err := value.ValidateValueStore(values); err != nil {
			return nil, err
		}
	}
	return &LifecycleService{repository: repository, clock: timeClock, values: values}, nil
}

func (s *LifecycleService) DrainSecretGarbage(ctx context.Context) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if s.values == nil {
		return fmt.Errorf("%w: Artifact Store secret value backend is not configured", spec.ErrUnsupported)
	}
	values, err := s.repository.ListSecretCleanup(ctx, secretCleanupBatchSize)
	if err != nil {
		return err
	}
	var output error
	for _, record := range values {
		if record.StoreName != s.values.Name() {
			_ = s.repository.RecordSecretCleanupFailure(
				context.WithoutCancel(ctx),
				record.Ref,
				"configured secret backend does not own cleanup record",
				clockutil.NowUTC(s.clock),
			)
			output = errors.Join(
				output,
				fmt.Errorf("%w: cleanup record belongs to secret backend %q", spec.ErrUnsupported, record.StoreName),
			)
			continue
		}
		if err := s.values.Delete(ctx, record.Ref); err != nil {
			_ = s.repository.RecordSecretCleanupFailure(
				context.WithoutCancel(ctx),
				record.Ref,
				"physical secret deletion failed",
				clockutil.NowUTC(s.clock),
			)
			output = errors.Join(output, err)
			continue
		}
		if err := s.repository.CompleteSecretCleanup(ctx, record.Ref); err != nil {
			output = errors.Join(output, err)
		}
	}
	return output
}

func (s *LifecycleService) RecoverPending(ctx context.Context) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := s.repository.RecoverPendingSecrets(ctx, clockutil.NowUTC(s.clock)); err != nil {
		return err
	}
	if s.values == nil {
		return nil
	}
	_ = s.DrainSecretGarbage(context.WithoutCancel(ctx))
	return nil
}

func (s *LifecycleService) ready(ctx context.Context) error {
	if s == nil || s.repository == nil || s.clock == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf("%w: secret lifecycle context is nil", spec.ErrInvalid)
	}
	return ctx.Err()
}
