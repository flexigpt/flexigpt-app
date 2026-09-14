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
	InstallBuiltInSkill(
		ctx context.Context,
		request BuiltInSkillInstallRequest,
	) (artifact.Artifact, error)
	InstallBuiltInSkillCollection(
		ctx context.Context,
		request BuiltInSkillCollectionInstallRequest,
	) (artifact.Artifact, error)

	EnsureBuiltInSkillSourceCurrent(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) error
}
