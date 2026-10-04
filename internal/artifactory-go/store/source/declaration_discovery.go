package source

import (
	"slices"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// AddDeclarationDiscovery adds one declaration origin and decoder hint to an
// owned discovery configuration.
//
// "authoritative" is the caller's discovery-ownership policy, not mutation
// authorization. This function performs no persistence, refresh, or package
// publication. Existing unrestricted decoder selection stays unrestricted.
func AddDeclarationDiscovery(
	current sourceModel.DiscoverySpec,
	locator spec.Locator,
	decoder spec.DecoderID,
	authoritative bool,
) (sourceModel.DiscoverySpec, error) {
	if err := current.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	if err := locator.Validate(false); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	if err := decoder.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}

	next := current.Clone()
	selected, err := next.InScope(locator)
	if err != nil {
		return sourceModel.DiscoverySpec{}, err
	}
	if !selected {
		next.ExplicitLocators = AppendUniqueLocator(next.ExplicitLocators, locator)
	}
	next.DecoderHints = AppendDecoderHint(next.DecoderHints, sourceModel.DecoderHint{
		Locator:    locator,
		DecoderIDs: []spec.DecoderID{decoder},
	})
	if len(next.AllowedDecoderIDs) != 0 &&
		!slices.Contains(next.AllowedDecoderIDs, decoder) {
		next.AllowedDecoderIDs = append(next.AllowedDecoderIDs, decoder)
	}
	next.Authoritative = next.Authoritative || authoritative
	return next.Normalized(), nil
}
