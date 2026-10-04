package resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// WithVerificationSession runs fn under one explicit ordinary resource
// verification session. There is no dynamic fallback: callers that require
// coherent multi-read results must receive the session capability directly.
func WithVerificationSession[T any](
	ctx context.Context,
	resources SessionAPI,
	fn func(context.Context) (T, error),
) (value T, returnErr error) {
	var zero T
	if ctx == nil {
		return zero, fmt.Errorf("%w: resource verification session context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if resources == nil {
		return zero, fmt.Errorf("%w: resource verification session capability is nil", spec.ErrInvalid)
	}
	if fn == nil {
		return zero, fmt.Errorf("%w: resource verification session callback is nil", spec.ErrInvalid)
	}
	sessionCtx, session, err := resources.BeginVerificationSession(ctx)
	if err != nil {
		return zero, err
	}
	if sessionCtx == nil || session == nil {
		return zero, fmt.Errorf("%w: resource verification session is incomplete", spec.ErrInvalid)
	}
	defer func() { returnErr = errors.Join(returnErr, session.Close(context.WithoutCancel(sessionCtx))) }()
	return fn(sessionCtx)
}
