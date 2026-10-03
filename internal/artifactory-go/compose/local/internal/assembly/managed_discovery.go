package assembly

import (
	"context"
	"fmt"

	managepackageimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/impl"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// pruneManagedDeclarationDiscovery removes one exact explicit declaration
// candidate from an authoritative managed Source. It runs after physical
// package removal and before managedartifact.Service performs its final
// refresh, avoiding a second refresh after every managed deletion.
func (c *Components) pruneManagedDeclarationDiscovery(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	locator spec.Locator,
) (managepackageimpl.SourceState, error) {
	if c == nil ||
		c.Sources == nil ||
		c.SourceRuntime == nil ||
		c.managedSources == nil {
		return managepackageimpl.SourceState{}, spec.ErrClosed
	}
	if ctx == nil {
		return managepackageimpl.SourceState{}, fmt.Errorf(
			"%w: managed discovery pruning context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return managepackageimpl.SourceState{}, err
	}
	if err := rootID.Validate(); err != nil {
		return managepackageimpl.SourceState{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return managepackageimpl.SourceState{}, err
	}
	if err := locator.Validate(false); err != nil {
		return managepackageimpl.SourceState{}, err
	}
	if expectedSourceRevision == 0 {
		return managepackageimpl.SourceState{}, fmt.Errorf(
			"%w: expected Source revision is required",
			spec.ErrInvalid,
		)
	}

	current, err := c.SourceRuntime.Get(ctx, rootID, sourceID)
	if err != nil {
		return managepackageimpl.SourceState{}, err
	}
	if current.Revision != expectedSourceRevision {
		return managepackageimpl.SourceState{}, spec.ErrConflict
	}
	if !current.Enabled {
		return managepackageimpl.SourceState{}, fmt.Errorf(
			"%w: managed Source is disabled",
			spec.ErrConflict,
		)
	}
	if !c.managedSources.SupportsManagedPackages(current.Kind) {
		return managepackageimpl.SourceState{}, fmt.Errorf(
			"%w: source kind %q is not writable",
			spec.ErrUnsupported,
			current.Kind,
		)
	}
	if !current.Discovery.Authoritative {
		return managepackageimpl.SourceState{}, fmt.Errorf(
			"%w: managed discovery pruning requires an authoritative Source",
			spec.ErrInvalid,
		)
	}

	next := current.Discovery.Clone()
	changed := false
	locators := make(
		[]spec.Locator,
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
		return managepackageimpl.SourceState{}, err
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
		r, err := c.getManagedSourceState(ctx, rootID, sourceID)
		if err != nil {
			return managepackageimpl.SourceState{}, err
		}
		return managepackageimpl.SourceState{
			Source:     r.Source,
			Generation: r.Generation,
		}, nil
	}

	if next.Empty() {
		next = sourceModel.DiscoverySpec{}
	} else {
		next = next.Normalized()
	}
	if err := next.Validate(); err != nil {
		return managepackageimpl.SourceState{}, err
	}

	if _, err := c.Sources.Update(
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
		return managepackageimpl.SourceState{}, err
	}
	r, err := c.getManagedSourceState(ctx, rootID, sourceID)
	if err != nil {
		return managepackageimpl.SourceState{}, err
	}
	return managepackageimpl.SourceState{
		Source:     r.Source,
		Generation: r.Generation,
	}, nil
}
