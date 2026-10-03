package root

import (
	"context"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

// ProtectionAPI is a composition/runtime capability. It is not a normal
// entity mutation API.
type ProtectionAPI interface {
	IsProtectedRoot(rootID rootModel.RootID) bool
	RequirePrivilegedInstaller(ctx context.Context) error
}
