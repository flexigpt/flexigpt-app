package manageddiscovery

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
)

// RemoveLocator removes one exact managed declaration candidate after its
// package has been removed. Remaining managed declaration discovery stays
// authoritative and is refreshed before returning.
func RemoveLocator(
	ctx context.Context,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	rootID root.RootID,
	sourceID source.SourceID,
	locator basespec.Locator,
) error {
	if sources == nil || discovery == nil {
		return basespec.ErrClosed
	}

	current, err := sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return err
	}
	next := current.Discovery.Clone()

	removed := false
	locators := make([]basespec.Locator, 0, len(next.ExplicitLocators))
	for _, value := range next.ExplicitLocators {
		if value == locator {
			removed = true
			continue
		}
		locators = append(locators, value)
	}
	if !removed {
		return nil
	}
	next.ExplicitLocators = locators
	delete(next.ExpectedContentDigests, locator)

	stillInScope, err := next.InScope(locator)
	if err != nil {
		return err
	}
	if !stillInScope {
		hints := make([]source.DecoderHint, 0, len(next.DecoderHints))
		for _, hint := range next.DecoderHints {
			if hint.Locator == locator {
				continue
			}
			hints = append(hints, hint)
		}
		next.DecoderHints = hints
	}

	if next.Empty() {
		next = source.DiscoverySpec{}
	} else {
		next.Authoritative = true
		next = next.Normalized()
		if err := next.Validate(); err != nil {
			return err
		}
	}

	if current.Discovery.Equal(next) {
		return nil
	}
	if _, err := sources.Update(
		ctx,
		rootID,
		sourceID,
		source.Update{
			ExpectedRevision: current.Revision,
			DisplayName:      current.DisplayName,
			Enabled:          current.Enabled,
			Discovery:        &next,
		},
	); err != nil {
		return err
	}

	if next.Empty() {
		return nil
	}
	_, err = discovery.RefreshSource(ctx, rootID, sourceID)
	return err
}
