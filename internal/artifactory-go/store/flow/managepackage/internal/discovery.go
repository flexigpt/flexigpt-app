package internal

import (
	"context"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (s *Service) pruneDiscoveryLocator(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	locator storeSpec.Locator,
) (SourceState, error) {
	if err := rootID.Validate(); err != nil {
		return SourceState{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return SourceState{}, err
	}
	if err := locator.Validate(false); err != nil {
		return SourceState{}, err
	}
	if expectedSourceRevision == 0 {
		return SourceState{}, fmt.Errorf(
			"%w: expected Source revision is required",
			storeSpec.ErrInvalid,
		)
	}

	current, err := s.dependencies.Runtime.Get(
		ctx,
		rootID,
		sourceID,
	)
	if err != nil {
		return SourceState{}, err
	}
	if current.Revision != expectedSourceRevision {
		return SourceState{}, storeSpec.ErrConflict
	}
	if !current.Enabled {
		return SourceState{}, fmt.Errorf(
			"%w: managed Source is disabled",
			storeSpec.ErrConflict,
		)
	}
	if !s.dependencies.Packages.SupportsManagedPackages(current.Kind) {
		return SourceState{}, fmt.Errorf(
			"%w: source kind %q is not writable",
			storeSpec.ErrUnsupported,
			current.Kind,
		)
	}
	if !current.Discovery.Authoritative {
		return SourceState{}, fmt.Errorf(
			"%w: managed discovery pruning requires an authoritative Source",
			storeSpec.ErrInvalid,
		)
	}

	next := current.Discovery.Clone()
	changed := false

	locators := make(
		[]storeSpec.Locator,
		0,
		len(next.ExplicitLocators),
	)
	for _, value := range next.ExplicitLocators {
		if value == locator {
			changed = true
			continue
		}
		locators = append(locators, value)
	}
	next.ExplicitLocators = locators

	if _, found := next.ExpectedContentDigests[locator]; found {
		delete(next.ExpectedContentDigests, locator)
		changed = true
	}

	inScope, err := next.InScope(locator)
	if err != nil {
		return SourceState{}, err
	}
	if !inScope {
		hints := make(
			[]sourceModel.DecoderHint,
			0,
			len(next.DecoderHints),
		)
		for _, hint := range next.DecoderHints {
			if hint.Locator == locator && !hint.Recursive {
				changed = true
				continue
			}
			hints = append(hints, hint.Clone())
		}
		next.DecoderHints = hints
	}

	if !changed {
		return s.sourceState(ctx, rootID, sourceID)
	}

	if next.Empty() {
		next = sourceModel.DiscoverySpec{}
	} else {
		next = next.Normalized()
	}
	if err := next.Validate(); err != nil {
		return SourceState{}, err
	}

	if _, err := s.dependencies.Sources.Update(
		ctx,
		rootID,
		sourceID,
		sourceModel.Update{
			ExpectedRevision: current.Revision,
			DisplayName:      current.DisplayName,
			Enabled:          current.Enabled,
			Discovery:        &next,
		},
	); err != nil {
		return SourceState{}, err
	}

	return s.sourceState(ctx, rootID, sourceID)
}
