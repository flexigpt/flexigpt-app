package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

// BuiltinStore is the narrow Agent Store capability used by the trusted
// protected-topology Agent installer.
type BuiltinStore interface {
	InstallBuiltInAgentPackage(
		ctx context.Context,
		request BuiltInAgentPackageInstallRequest,
	) ([]artifact.Artifact, error)

	ValidateBuiltInAgentPackage(
		ctx context.Context,
		request BuiltInAgentPackageInstallRequest,
	) error

	RemoveBuiltInAgentPackage(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		address source.ManagedPackageAddress,
	) error

	EnsureBuiltInAgentSourceCurrent(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) error
}
