package system

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	managedartifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/managedartifact"
)

// pruneManagedDeclarationDiscovery removes one exact explicit declaration
// candidate from an authoritative managed Source. It runs after physical
// package removal and before managedartifact.Service performs its final
// refresh, avoiding a second refresh after every managed deletion.
func (c *Components) pruneManagedDeclarationDiscovery(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	expectedSourceRevision uint64,
	locator basespec.Locator,
) (managedartifactimpl.SourceState, error) {
	if c == nil ||
		c.Sources == nil ||
		c.SourceRuntime == nil ||
		c.managedSources == nil {
		return managedartifactimpl.SourceState{}, basespec.ErrClosed
	}
	if ctx == nil {
		return managedartifactimpl.SourceState{}, fmt.Errorf(
			"%w: managed discovery pruning context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return managedartifactimpl.SourceState{}, err
	}
	if err := rootID.Validate(); err != nil {
		return managedartifactimpl.SourceState{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return managedartifactimpl.SourceState{}, err
	}
	if err := locator.Validate(false); err != nil {
		return managedartifactimpl.SourceState{}, err
	}
	if expectedSourceRevision == 0 {
		return managedartifactimpl.SourceState{}, fmt.Errorf(
			"%w: expected Source revision is required",
			basespec.ErrInvalid,
		)
	}

	current, err := c.SourceRuntime.Get(ctx, rootID, sourceID)
	if err != nil {
		return managedartifactimpl.SourceState{}, err
	}
	if current.Revision != expectedSourceRevision {
		return managedartifactimpl.SourceState{}, basespec.ErrConflict
	}
	if !current.Enabled {
		return managedartifactimpl.SourceState{}, fmt.Errorf(
			"%w: managed Source is disabled",
			basespec.ErrConflict,
		)
	}
	if !c.managedSources.SupportsManagedPackages(current.Kind) {
		return managedartifactimpl.SourceState{}, fmt.Errorf(
			"%w: source kind %q is not writable",
			basespec.ErrUnsupported,
			current.Kind,
		)
	}
	if !current.Discovery.Authoritative {
		return managedartifactimpl.SourceState{}, fmt.Errorf(
			"%w: managed discovery pruning requires an authoritative Source",
			basespec.ErrInvalid,
		)
	}

	next := current.Discovery.Clone()
	changed := false
	locators := make(
		[]basespec.Locator,
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
		return managedartifactimpl.SourceState{}, err
	}
	if !inScope {
		hints := make(
			[]source.DecoderHint,
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
			return managedartifactimpl.SourceState{}, err
		}
		return managedartifactimpl.SourceState{
			Source:     r.Source,
			Generation: r.Generation,
		}, nil
	}

	if next.Empty() {
		next = source.DiscoverySpec{}
	} else {
		next = next.Normalized()
	}
	if err := next.Validate(); err != nil {
		return managedartifactimpl.SourceState{}, err
	}

	if _, err := c.Sources.Update(
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
		return managedartifactimpl.SourceState{}, err
	}
	r, err := c.getManagedSourceState(ctx, rootID, sourceID)
	if err != nil {
		return managedartifactimpl.SourceState{}, err
	}
	return managedartifactimpl.SourceState{
		Source:     r.Source,
		Generation: r.Generation,
	}, nil
}
