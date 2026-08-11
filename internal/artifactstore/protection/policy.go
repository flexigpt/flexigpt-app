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

// RootDeletionPolicy is an optional lifecycle policy for Roots that must
// remain present while still allowing ordinary mutation of their descendants.
//
// A retained Root is intentionally different from a protected topology Root:
// protected Roots reject ordinary Source, Collection, and Artifact mutations;
// retained Roots reject only Root retirement and purge.
type RootDeletionPolicy interface {
	IsRootDeletionProtected(rootID basespec.RootID) bool
}

type StaticRootPolicy struct {
	RootID         basespec.RootID
	RetainedRootID basespec.RootID
}

func (p StaticRootPolicy) IsProtectedRoot(rootID basespec.RootID) bool {
	return p.RootID != "" && p.RootID == rootID
}

func (p StaticRootPolicy) IsRootDeletionProtected(
	rootID basespec.RootID,
) bool {
	return p.RetainedRootID != "" && p.RetainedRootID == rootID
}

type privilegedInstallerContextKey struct{}

// WithPrivilegedInstaller grants the narrow application-composition capability
// used by trusted protected-topology installers and update paths. It must never
// be used by transport wrappers.
func WithPrivilegedInstaller(ctx context.Context) context.Context {
	return context.WithValue(ctx, privilegedInstallerContextKey{}, true)
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

// RequireRootDeletion permits normal mutable-root checks and additionally
// rejects retirement or purge of a retained application Root. Retention is
// not bypassed by installer context because it is an application data-retention
// policy rather than protected-topology installation access.
func RequireRootDeletion(
	ctx context.Context,
	policy RootPolicy,
	rootID basespec.RootID,
) error {
	if deletionPolicy, supported := policy.(RootDeletionPolicy); supported &&
		deletionPolicy.IsRootDeletionProtected(rootID) {
		return fmt.Errorf(
			"%w: root %q is retained and cannot be retired or purged",
			basespec.ErrProtected,
			rootID,
		)
	}
	return RequireMutableRoot(ctx, policy, rootID)
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

func IsPrivilegedInstaller(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	value, _ := ctx.Value(privilegedInstallerContextKey{}).(bool)
	return value
}
