package refresh

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// LifecyclePublication is Refresh's aggregate command for a Source lifecycle
// transition. The requested Source transition, refresh-state invalidation,
// and Artifact source-state updates must commit together.
type LifecyclePublication struct {
	Transition              source.LifecycleTransition
	ExpectedRefreshRevision uint64
	ArtifactUpdates         []artifact.SourceStateUpdate
}

func (p LifecyclePublication) Validate() error {
	if err := p.Transition.Validate(); err != nil {
		return err
	}
	seen := make(map[artifactModel.ArtifactID]struct{}, len(p.ArtifactUpdates))
	for index, update := range p.ArtifactUpdates {
		if err := update.Validate(); err != nil {
			return fmt.Errorf("lifecycle Artifact update %d: %w", index, err)
		}
		if update.RootID != p.Transition.Source.RootID ||
			update.Binding.SourceID != p.Transition.Source.ID ||
			update.State != artifactModel.StateMissing ||
			update.ResolvedDefinition != nil ||
			update.SourceContentDigest != nil {
			return fmt.Errorf(
				"%w: lifecycle Artifact update does not invalidate the transitioned Source",
				spec.ErrInvalid,
			)
		}
		if _, duplicate := seen[update.ArtifactID]; duplicate {
			return fmt.Errorf("%w: lifecycle publication repeats Artifact %q", spec.ErrInvalid, update.ArtifactID)
		}
		seen[update.ArtifactID] = struct{}{}
	}
	return nil
}
