// Package refresh owns explicit declaration-discovery preparation and refresh
// closure. Normal composition resolution remains read-only.
package refresh

import (
	"context"
	"fmt"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type Requirement struct {
	RootID   rootModel.RootID
	SourceID sourceModel.SourceID

	Discovery sourceModel.DiscoveryRequirement
}

func (r Requirement) Validate() error {
	if err := r.RootID.Validate(); err != nil {
		return err
	}
	if err := r.SourceID.Validate(); err != nil {
		return err
	}
	return r.Discovery.Validate()
}

// Planner remains family- or application-owned. This package coordinates only
// Source-owned discovery preparation and explicit refresh sequencing.
type Planner interface {
	ReachableRequirements(
		ctx context.Context,
		root artifactModel.ArtifactRef,
	) ([]Requirement, error)
}

type Service struct {
	sources source.API
	refresh refreshFlow.API
	planner Planner

	maxPasses int
}

func New(
	sources source.API,
	refreshes refreshFlow.API,
	planner Planner,
	maxPasses int,
) (*Service, error) {
	if sources == nil || refreshes == nil || planner == nil {
		return nil, fmt.Errorf(
			"%w: composition refresh dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	if maxPasses <= 0 || maxPasses > spec.MaxDiscoveryDepth {
		return nil, fmt.Errorf(
			"%w: composition refresh pass limit is invalid",
			spec.ErrInvalid,
		)
	}
	return &Service{
		sources:   sources,
		refresh:   refreshes,
		planner:   planner,
		maxPasses: maxPasses,
	}, nil
}

func (s *Service) Refresh(
	ctx context.Context,
	root artifactModel.ArtifactRef,
) error {
	if s == nil || s.sources == nil || s.refresh == nil || s.planner == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: composition refresh context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := root.Validate(); err != nil {
		return err
	}

	refreshed := make(map[sourceKey]struct{})
	for pass := 0; pass < s.maxPasses; pass++ {
		requirements, err := s.planner.ReachableRequirements(ctx, root)
		if err != nil {
			return err
		}
		if len(requirements) == 0 {
			return nil
		}

		ordered, err := normalizeRequirements(requirements)
		if err != nil {
			return err
		}

		changed := false
		for _, requirement := range ordered {
			current, err := s.sources.Get(
				ctx,
				requirement.RootID,
				requirement.SourceID,
			)
			if err != nil {
				return err
			}

			next, err := s.sources.PrepareDiscovery(
				ctx,
				requirement.RootID,
				requirement.SourceID,
				sourceModel.DiscoveryPreparation{
					ExpectedRevision: current.Revision,
					Intent:           sourceModel.DiscoveryPreparationAdditive,
					Requirement:      requirement.Discovery,
				},
			)
			if err != nil {
				return err
			}

			key := sourceKey{
				rootID:   next.RootID,
				sourceID: next.ID,
			}
			if _, alreadyRefreshed := refreshed[key]; alreadyRefreshed &&
				next.Revision == current.Revision {
				continue
			}
			if _, err := s.refresh.RefreshSource(
				ctx,
				next.RootID,
				next.ID,
			); err != nil {
				return err
			}
			refreshed[key] = struct{}{}
			changed = true
		}
		if !changed {
			return nil
		}
	}

	return fmt.Errorf(
		"%w: composition refresh closure exceeds %d passes",
		spec.ErrLocatorLimitExceeded,
		s.maxPasses,
	)
}

type sourceKey struct {
	rootID   rootModel.RootID
	sourceID sourceModel.SourceID
}

func normalizeRequirements(
	values []Requirement,
) ([]Requirement, error) {
	merged := make(map[sourceKey]Requirement, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return nil, err
		}
		key := sourceKey{
			rootID:   value.RootID,
			sourceID: value.SourceID,
		}
		current, found := merged[key]
		if !found {
			merged[key] = value
			continue
		}
		current.Discovery = mergeRequirements(
			current.Discovery,
			value.Discovery,
		)
		merged[key] = current
	}

	output := make([]Requirement, 0, len(merged))
	for _, value := range merged {
		output = append(output, value)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		return output[left].SourceID < output[right].SourceID
	})
	return output, nil
}

func mergeRequirements(
	left sourceModel.DiscoveryRequirement,
	right sourceModel.DiscoveryRequirement,
) sourceModel.DiscoveryRequirement {
	current := sourceModel.DiscoverySpec{
		ExplicitLocators: append(
			[]spec.Locator(nil),
			left.ExplicitLocators...,
		),
		DirectoryRoots: append([]sourceModel.DirectoryRoot(nil), left.DirectoryRoots...),
		DecoderHints:   append([]sourceModel.DecoderHint(nil), left.DecoderHints...),
		Authoritative:  left.RequireAuthoritative,
	}
	required := sourceModel.DiscoverySpec{
		ExplicitLocators: append(
			[]spec.Locator(nil),
			right.ExplicitLocators...,
		),
		DirectoryRoots: append([]sourceModel.DirectoryRoot(nil), right.DirectoryRoots...),
		DecoderHints:   append([]sourceModel.DecoderHint(nil), right.DecoderHints...),
		Authoritative:  right.RequireAuthoritative,
	}

	merged := source.MergeDiscoveryScopes(current, required)
	return sourceModel.DiscoveryRequirement{
		ExplicitLocators:     merged.ExplicitLocators,
		DirectoryRoots:       merged.DirectoryRoots,
		DecoderHints:         merged.DecoderHints,
		RequireAuthoritative: merged.Authoritative,
	}
}
