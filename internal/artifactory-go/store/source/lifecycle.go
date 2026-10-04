package source

import (
	"context"
	"fmt"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// LifecycleInvalidation identifies the Source-owned transition that makes its
// previously refreshed source-backed Artifacts unavailable. Artifact derives
// the resulting source state; Refresh coordinates its atomic publication.
type LifecycleInvalidation string

const (
	LifecycleInvalidationDisabled         LifecycleInvalidation = "disabled"
	LifecycleInvalidationDiscoveryRemoved LifecycleInvalidation = "discovery-removed"
	LifecycleInvalidationRetired          LifecycleInvalidation = "retired"
)

// LifecycleTransition is an explicit requested Source transition requiring
// aggregate invalidation publication. Source owns construction and validation
// of this command; it contains no provider or Refresh implementation detail.
type LifecycleTransition struct {
	Source                 sourceModel.Source
	ExpectedSourceRevision uint64
	Invalidation           LifecycleInvalidation
}

func (t LifecycleTransition) Validate() error {
	if err := t.Source.ValidateRead(); err != nil {
		return err
	}
	if t.ExpectedSourceRevision == 0 ||
		t.ExpectedSourceRevision == ^uint64(0) ||
		t.Source.Revision != t.ExpectedSourceRevision+1 {
		return fmt.Errorf("%w: invalid Source lifecycle revision transition", spec.ErrInvalid)
	}
	switch t.Invalidation {
	case LifecycleInvalidationDisabled:
		if t.Source.Enabled || t.Source.RetiredAt != nil {
			return fmt.Errorf(
				"%w: disabled Source lifecycle transition must retain an active disabled Source",
				spec.ErrInvalid,
			)
		}
	case LifecycleInvalidationDiscoveryRemoved:
		if !t.Source.Enabled || t.Source.RetiredAt != nil || !t.Source.Discovery.Empty() {
			return fmt.Errorf(
				"%w: discovery-removal lifecycle transition must retain an enabled Source with empty discovery",
				spec.ErrInvalid,
			)
		}
	case LifecycleInvalidationRetired:
		if t.Source.Enabled || t.Source.RetiredAt == nil {
			return fmt.Errorf(
				"%w: retired Source lifecycle transition must retire and disable the Source",
				spec.ErrInvalid,
			)
		}
	default:
		return fmt.Errorf("%w: unsupported Source lifecycle invalidation %q", spec.ErrInvalid, t.Invalidation)
	}
	return nil
}

// LifecyclePublisher is the narrow aggregate publication capability Source
// needs for lifecycle transitions. Refresh implements it; Source never imports
// the coordinating flow or a provider implementation.
type LifecyclePublisher interface {
	PublishSourceLifecycle(ctx context.Context, transition LifecycleTransition) error
}
