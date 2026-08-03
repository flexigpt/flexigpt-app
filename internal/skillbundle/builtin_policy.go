package skillbundle

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
)

// builtInSkillArtifactPolicy deliberately creates no observed Artifacts.
// Protected topology has one declared static Artifact ID for every member.
// Unknown source content may remain visible as a catalog occurrence, but it
// must never silently become a canonical built-in Artifact.
type builtInSkillArtifactPolicy struct{}

func (builtInSkillArtifactPolicy) Derive(
	_ context.Context,
	_ collection.Collection,
	_ catalog.Occurrence,
	_ definition.Definition,
) (artifact.Draft, bool, []diagnostic.Diagnostic, error) {
	return artifact.Draft{}, false, nil, nil
}
