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
	sessionCtx, session, err := resources.BeginVerificationSession(ctx)
	if err != nil {
		return zero, err
	}
	if sessionCtx == nil || session == nil {
		return zero, fmt.Errorf("%w: resource verification session is incomplete", spec.ErrInvalid)
	}
	defer func() {
		returnErr = errors.Join(
			returnErr,
			session.Close(context.WithoutCancel(sessionCtx)),
		)
		if returnErr != nil {
			value = zero
		}
	}()
	return fn(sessionCtx)
}
