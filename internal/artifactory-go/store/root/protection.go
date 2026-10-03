package root

import (
	"context"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

// System is the trusted Root capability used by protected topology
// installation. Ordinary callers use API and cannot create protected Roots.
type System interface {
	EnsureSystem(
		ctx context.Context,
		draft rootModel.RootDraft,
	) (rootModel.Root, error)
}

// ProtectionAPI is a composition/runtime capability. It is not a normal
// entity mutation API.
type ProtectionAPI interface {
	IsProtectedRoot(rootID rootModel.RootID) bool
	RequirePrivilegedInstaller(ctx context.Context) error
}
