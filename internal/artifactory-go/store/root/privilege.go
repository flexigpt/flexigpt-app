package root

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type installerPrivilegeContextKey struct{}

// WithInstallerPrivilege marks a context as trusted protected-topology work.
// It is an in-process composition convention, not a transport authorization
// mechanism and not an unforgeable security boundary.
func WithInstallerPrivilege(ctx context.Context) context.Context {
	return context.WithValue(ctx, installerPrivilegeContextKey{}, true)
}

// IsInstallerPrivileged reports whether ctx carries the trusted installer
// convention established by application composition.
func IsInstallerPrivileged(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	granted, _ := ctx.Value(installerPrivilegeContextKey{}).(bool)
	return granted
}

// RequireInstallerPrivilege validates trusted protected-topology access.
func RequireInstallerPrivilege(ctx context.Context) error {
	if ctx == nil || !IsInstallerPrivileged(ctx) {
		return fmt.Errorf(
			"%w: protected topology installation requires trusted installer access",
			spec.ErrProtected,
		)
	}
	return nil
}
