package resource

import (
	"context"
	"errors"
	"fmt"

	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type verificationSessionStarter interface {
	BeginVerificationSession(
		ctx context.Context,
	) (context.Context, resourceModel.VerificationSession, error)
}

// WithVerificationSession runs fn under a shared verification session when the
// resource implementation supports sessions. Older test doubles may implement
// only the ordinary Resource API.
func WithVerificationSession[T any](
	ctx context.Context,
	resources any,
	fn func(context.Context) (T, error),
) (value T, returnErr error) {
	var zero T

	if ctx == nil {
		return zero, fmt.Errorf(
			"%w: resource verification session context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if fn == nil {
		return zero, fmt.Errorf(
			"%w: resource verification session callback is nil",
			spec.ErrInvalid,
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
			spec.ErrInvalid,
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
