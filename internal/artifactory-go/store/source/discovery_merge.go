package source

import (
	"slices"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func AppendUniqueLocator(
	values []spec.Locator,
	value spec.Locator,
) []spec.Locator {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func AppendDirectoryRoot(
	values []sourceModel.DirectoryRoot,
	value sourceModel.DirectoryRoot,
) []sourceModel.DirectoryRoot {
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

func AppendDecoderHint(
	values []sourceModel.DecoderHint,
	value sourceModel.DecoderHint,
) []sourceModel.DecoderHint {
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

// MergeDiscoveryScopes preserves current dynamic discovery state while adding
// required static scopes, hints, and allowed decoders.
func MergeDiscoveryScopes(
	current sourceModel.DiscoverySpec,
	required sourceModel.DiscoverySpec,
) sourceModel.DiscoverySpec {
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
