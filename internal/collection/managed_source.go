package collection

import (
	"context"
	"fmt"
	"slices"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// EnsureManagedDeclarationDiscovery ensures that one managed declaration
// origin is selected by its managed Source. Domain-managed Sources become
// authoritative because their declaration universe is owned by the managed
// Collection authoring flow.
//
// The caller publishes the package after this method returns. Publication
// owns the resulting Source refresh.
func (a *API) EnsureManagedDeclarationDiscovery(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	locator spec.Locator,
	requiredDecoder spec.DecoderID,
) (sourceModel.Summary, error) {
	if a == nil {
		return sourceModel.Summary{}, spec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := locator.Validate(false); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := requiredDecoder.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}

	current, err := a.managedSource(ctx, rootID, sourceID)
	if err != nil {
		return sourceModel.Summary{}, err
	}

	next := current.Discovery.Clone()
	inScope, err := next.InScope(locator)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if !inScope {
		next.ExplicitLocators = append(next.ExplicitLocators, locator)
	}
	next.Authoritative = true

	hintFound := false
	for index := range next.DecoderHints {
		hint := &next.DecoderHints[index]
		if hint.Locator != locator || hint.Recursive {
			continue
		}
		hintFound = true
		if !slices.Contains(hint.DecoderIDs, requiredDecoder) {
			hint.DecoderIDs = append(
				hint.DecoderIDs,
				requiredDecoder,
			)
		}
	}
	if !hintFound {
		next.DecoderHints = append(next.DecoderHints, sourceModel.DecoderHint{
			Locator:    locator,
			Recursive:  false,
			DecoderIDs: []spec.DecoderID{requiredDecoder},
		})
	}

	if len(next.AllowedDecoderIDs) != 0 &&
		!slices.Contains(next.AllowedDecoderIDs, requiredDecoder) {
		next.AllowedDecoderIDs = append(
			next.AllowedDecoderIDs,
			requiredDecoder,
		)
	}

	next = next.Normalized()
	if err := next.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if current.Discovery.Equal(next) {
		return current, nil
	}

	updated, err := a.sources.Update(
		ctx,
		rootID,
		sourceID,
		sourceModel.Update{
			ExpectedRevision: current.Revision,
			DisplayName:      current.DisplayName,
			Enabled:          current.Enabled,
			Discovery:        &next,
		},
	)
	if err != nil {
		return sourceModel.Summary{}, fmt.Errorf(
			"update managed declaration discovery: %w",
			err,
		)
	}
	return updated, nil
}
