package consumerutil

import (
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

// AppendUniqueLocator appends a declaration locator only when it is not
// already present.
func AppendUniqueLocator(
	values []basespec.Locator,
	value basespec.Locator,
) []basespec.Locator {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

// AppendDirectoryRoot appends one discovery root when an identical root does
// not already exist.
func AppendDirectoryRoot(
	values []source.DirectoryRoot,
	value source.DirectoryRoot,
) []source.DirectoryRoot {
	for _, current := range values {
		if current.Root != value.Root ||
			current.Recursive != value.Recursive ||
			!slices.Equal(
				current.IncludePatterns,
				value.IncludePatterns,
			) ||
			!slices.Equal(
				current.ExcludePatterns,
				value.ExcludePatterns,
			) {
			continue
		}
		return values
	}
	return append(values, value.Clone())
}

// AppendDecoderHint merges requested decoders for one locator scope.
func AppendDecoderHint(
	values []source.DecoderHint,
	value source.DecoderHint,
) []source.DecoderHint {
	for index := range values {
		if values[index].Locator != value.Locator ||
			values[index].Recursive != value.Recursive {
			continue
		}
		for _, decoderID := range value.DecoderIDs {
			if slices.Contains(values[index].DecoderIDs, decoderID) {
				continue
			}
			values[index].DecoderIDs = append(
				values[index].DecoderIDs,
				decoderID,
			)
		}
		return values
	}
	return append(values, value.Clone())
}

// MergeDiscoveryScopes preserves existing discovery closure while adding
// required static scopes, decoder hints, and allowed decoders.
//
// Source limits and expected-content digests remain current-Source state.
// They are intentionally not reconciled by this additive helper.
func MergeDiscoveryScopes(
	current source.DiscoverySpec,
	required source.DiscoverySpec,
) source.DiscoverySpec {
	output := current.Clone()
	for _, locator := range required.ExplicitLocators {
		output.ExplicitLocators = AppendUniqueLocator(
			output.ExplicitLocators,
			locator,
		)
	}
	for _, root := range required.DirectoryRoots {
		output.DirectoryRoots = AppendDirectoryRoot(
			output.DirectoryRoots,
			root,
		)
	}
	for _, hint := range required.DecoderHints {
		output.DecoderHints = AppendDecoderHint(
			output.DecoderHints,
			hint,
		)
	}
	for _, decoderID := range required.AllowedDecoderIDs {
		if !slices.Contains(output.AllowedDecoderIDs, decoderID) {
			output.AllowedDecoderIDs = append(
				output.AllowedDecoderIDs,
				decoderID,
			)
		}
	}
	output.Authoritative = output.Authoritative ||
		required.Authoritative
	return output.Normalized()
}
