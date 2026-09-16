package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

// BuiltinStore is the narrow Skill capability required by the protected
// built-in installer. It does not expose generic Store mutation internals.
type BuiltinStore interface {
	InstallBuiltInSkillPackage(
		ctx context.Context,

		request BuiltInSkillPackageInstallRequest,
	) ([]artifact.Artifact, error)

	RemoveBuiltInSkillPackage(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		address source.ManagedPackageAddress,
	) error

	EnsureBuiltInSkillSourceCurrent(ctx context.Context, rootID root.RootID, sourceID source.SourceID) error
}
