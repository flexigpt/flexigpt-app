package compose

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

// protection exposes only the trusted protected-root convention required by
// topology installers. It does not widen an ordinary entity API.
type protection struct {
	policy root.Policy
}

func (p protection) IsProtectedRoot(rootID rootModel.RootID) bool {
	return p.policy != nil && p.policy.IsProtectedRoot(rootID)
}

func (protection) RequireInstallerPrivilege(ctx context.Context) error {
	return root.RequireInstallerPrivilege(ctx)
}
