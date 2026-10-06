package source

import (
	"context"
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

// MergeDiscoveryRequirement adds family-required declaration discovery without
// deleting or narrowing unrelated Source discovery configuration.
//
// An unrestricted Source remains unrestricted. If a Source already has an
// explicit AllowedDecoderIDs restriction, required hinted decoders are added
// to that existing allow-list so the prepared hint remains executable.
func MergeDiscoveryRequirement(
	current sourceModel.DiscoverySpec,
	requirement sourceModel.DiscoveryRequirement,
) (sourceModel.DiscoverySpec, error) {
	if err := current.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	if err := requirement.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}

	next := current.Clone()
	for _, locator := range requirement.ExplicitLocators {
		next.ExplicitLocators = AppendUniqueLocator(
			next.ExplicitLocators,
			locator,
		)
	}
	for _, directory := range requirement.DirectoryRoots {
		next.DirectoryRoots = AppendDirectoryRoot(
			next.DirectoryRoots,
			directory,
		)
	}
	for _, hint := range requirement.DecoderHints {
		next.DecoderHints = AppendDecoderHint(
			next.DecoderHints,
			hint,
		)
		if len(next.AllowedDecoderIDs) == 0 {
			continue
		}
		for _, decoderID := range hint.DecoderIDs {
			if slices.Contains(next.AllowedDecoderIDs, decoderID) {
				continue
			}
			next.AllowedDecoderIDs = append(
				next.AllowedDecoderIDs,
				decoderID,
			)
		}
	}
	next.Authoritative = next.Authoritative ||
		requirement.RequireAuthoritative

	next = next.Normalized()
	if err := next.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	return next, nil
}

// PrepareDiscovery performs a revision-checked additive declaration-discovery
// update. It does not refresh the Source; refresh remains an explicit flow.
func (s *Service) PrepareDiscovery(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	preparation sourceModel.DiscoveryPreparation,
) (sourceModel.Summary, error) {
	if err := rootID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := preparation.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := root.RequireMutableRoot(ctx, s.policy, rootID); err != nil {
		return sourceModel.Summary{}, err
	}

	current, err := s.repository.Get(ctx, rootID, sourceID)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if current.Revision != preparation.ExpectedRevision {
		return sourceModel.Summary{}, spec.ErrConflict
	}

	nextDiscovery, err := MergeDiscoveryRequirement(
		current.Discovery,
		preparation.Requirement,
	)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if current.Discovery.Equal(nextDiscovery) {
		return current.Summary(), nil
	}
	if current.Revision == ^uint64(0) {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: Source revision is exhausted",
			spec.ErrInvalid,
		)
	}

	next := current.Clone()
	next.Discovery = nextDiscovery
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)

	if err := next.ValidateRead(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := s.repository.Update(
		ctx,
		next,
		preparation.ExpectedRevision,
	); err != nil {
		return sourceModel.Summary{}, err
	}
	return next.Summary(), nil
}
