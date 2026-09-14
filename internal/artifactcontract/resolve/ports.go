package resolve

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

// ArtifactReader is the narrow Root-scoped Artifact read boundary required by graph resolution.
// "compositionapi.ArtifactAPI" satisfies this interface structurally without resolve importing Store composition code.
type ArtifactReader interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)

	FindByIdentity(
		ctx context.Context,
		rootID root.RootID,
		kind artifact.ArtifactKind,
		logicalName basespec.LogicalName,
	) ([]artifact.Artifact, error)

	GetDefinition(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (definition.Definition, error)
}
