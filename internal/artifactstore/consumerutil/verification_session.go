package consumerutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
)

// verificationSessionStarter is intentionally optional. Production Resource
// services expose BeginVerificationSession, while small test doubles can keep
// implementing only the ordinary ResourceReader methods.
type verificationSessionStarter interface {
	BeginVerificationSession(
		ctx context.Context,
	) (context.Context, resource.VerificationSession, error)
}

// WithResourceVerificationSession runs one read-only multi-artifact operation
// under a shared source verification session when the supplied resource
// implementation supports it.
//
// The session is confirmed and closed before this function returns. Therefore
// callers never receive successful batch results from a source generation that
// changed during the batch.
func WithResourceVerificationSession[T any](
	ctx context.Context,
	resources any,
	fn func(context.Context) (T, error),
) (value T, returnErr error) {
	var zero T

	if ctx == nil {
		return zero, fmt.Errorf(
			"%w: resource verification session context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if fn == nil {
		return zero, fmt.Errorf(
			"%w: resource verification session callback is nil",
			basespec.ErrInvalid,
		)
	}

	starter, supported := resources.(verificationSessionStarter)
	if !supported {
		return fn(ctx)
	}

	sessionCtx, session, err := starter.BeginVerificationSession(ctx)
	if err != nil {
		return zero, err
	}
	if sessionCtx == nil || session == nil {
		return zero, fmt.Errorf(
			"%w: resource verification session is incomplete",
			basespec.ErrInvalid,
		)
	}

	defer func() {
		returnErr = errors.Join(
			returnErr,
			session.Close(context.WithoutCancel(sessionCtx)),
		)
	}()

	return fn(sessionCtx)
}
