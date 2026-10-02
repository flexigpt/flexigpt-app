package install

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
)

type privilegedContextKey struct{}

// WithPrivilege grants the narrow application-composition capability used by
// trusted protected-topology installers. It must not be used by transports or
// ordinary application consumers.
func WithPrivilege(ctx context.Context) context.Context {
	return context.WithValue(ctx, privilegedContextKey{}, true)
}

func RequirePrivileged(ctx context.Context) error {
	if ctx == nil || !IsPrivileged(ctx) {
		return fmt.Errorf(
			"%w: protected topology installation requires trusted installer access",
			model.ErrProtected,
		)
	}
	return nil
}

func IsPrivileged(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	value, _ := ctx.Value(privilegedContextKey{}).(bool)
	return value
}
