package protection

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// RootPolicy identifies Roots whose ordinary mutations are prohibited. The
// application composition owns the concrete policy and the protected-root
// declaration. Artifact Store does not know which application feature owns it.
type RootPolicy interface {
	IsProtectedRoot(r basespec.RootID) bool
}

type StaticRootPolicy struct {
	RootID basespec.RootID
}

func (p StaticRootPolicy) IsProtectedRoot(rootID basespec.RootID) bool {
	return p.RootID != "" && p.RootID == rootID
}

type privilegedInstallerContextKey struct{}

// WithPrivilegedInstaller grants the narrow application-composition capability
// used by trusted protected-topology installers and update paths. It must never
// be used by transport wrappers.
func WithPrivilegedInstaller(ctx context.Context) context.Context {
	return context.WithValue(ctx, privilegedInstallerContextKey{}, true)
}

func IsPrivilegedInstaller(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	value, _ := ctx.Value(privilegedInstallerContextKey{}).(bool)
	return value
}

func RequirePrivilegedInstaller(ctx context.Context) error {
	if ctx == nil || !IsPrivilegedInstaller(ctx) {
		return fmt.Errorf(
			"%w: protected topology installation requires trusted installer access",
			basespec.ErrProtected,
		)
	}
	return nil
}

func RequireMutableRoot(
	ctx context.Context,
	policy RootPolicy,
	rootID basespec.RootID,
) error {
	if policy == nil || !policy.IsProtectedRoot(rootID) {
		return nil
	}
	if IsPrivilegedInstaller(ctx) {
		return nil
	}
	return fmt.Errorf(
		"%w: root %q may only be mutated by a trusted protected-topology installer",
		basespec.ErrProtected,
		rootID,
	)
}
