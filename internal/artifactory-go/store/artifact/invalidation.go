package artifact

import (
	"context"
	"fmt"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

// DeriveLifecycleInvalidation derives explicit source-owned Artifact updates
// for one Source lifecycle transition. It never changes local fields or
// performs persistence.
func (s *Synchronizer) DeriveLifecycleInvalidation(
	ctx context.Context,
	transition source.LifecycleTransition,
	existing []artifactModel.Artifact,
) ([]SourceStateUpdate, error) {
	if err := transition.Validate(); err != nil {
		return nil, err
	}
	ordered := append([]artifactModel.Artifact(nil), existing...)
	sort.Slice(ordered, func(left, right int) bool { return ordered[left].ID < ordered[right].ID })
	seen := make(map[artifactModel.ArtifactID]struct{}, len(ordered))
	updates := make([]SourceStateUpdate, 0, len(ordered))
	for index, current := range ordered {
		if err := current.ValidateRead(); err != nil {
			return nil, fmt.Errorf("existing Artifact %d: %w", index, err)
		}
		if current.RootID != transition.Source.RootID || current.Binding.SourceID != transition.Source.ID {
			return nil, fmt.Errorf("%w: lifecycle invalidation Artifact belongs to another Source", spec.ErrInvalid)
		}
		if _, duplicate := seen[current.ID]; duplicate {
			return nil, fmt.Errorf("%w: lifecycle invalidation repeats Artifact %q", spec.ErrInvalid, current.ID)
		}
		seen[current.ID] = struct{}{}
		if current.State == artifactModel.StateMissing && current.ResolvedDefinition == nil &&
			current.SourceContentDigest == nil {
			continue
		}
		if current.Revision == ^uint64(0) {
			return nil, fmt.Errorf("%w: Artifact revision is exhausted", spec.ErrInvalid)
		}
		updates = append(updates, SourceStateUpdate{
			ArtifactID:       current.ID,
			RootID:           current.RootID,
			Binding:          current.Binding,
			LogicalName:      current.LogicalName,
			LogicalVersion:   current.LogicalVersion,
			State:            artifactModel.StateMissing,
			Diagnostics:      lifecycleInvalidationDiagnostics(transition.Invalidation),
			ExpectedRevision: current.Revision,
			Revision:         current.Revision + 1,
			ModifiedAt:       clockutil.Advance(transition.Source.ModifiedAt, current.ModifiedAt),
		})
	}
	return updates, nil
}

func lifecycleInvalidationDiagnostics(reason source.LifecycleInvalidation) []diagnostic.Diagnostic {
	code, message := "artifact.source-disabled", "the Artifact Source was disabled"
	switch reason {
	case source.LifecycleInvalidationDiscoveryRemoved:
		code, message = "artifact.discovery-disabled", "the Artifact Source no longer has declaration discovery"
	case source.LifecycleInvalidationRetired:
		code, message = "artifact.source-retired", "the Artifact Source was retired"
	default:
	}
	return []diagnostic.Diagnostic{{Severity: diagnostic.SeverityWarning, Code: code, Message: message}}
}
