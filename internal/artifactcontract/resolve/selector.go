package resolve

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

func (r *Resolver) expandSelector(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	member declaration.Entry,
	from *artifact.Artifact,
	depth int,
) (ResolvedSelector, error) {
	if from == nil {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector requires a source-backed declaration origin",
			basespec.ErrSourceUnavailable,
		)
	}
	if r.sourceArtifacts == nil {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector source enumeration is unavailable",
			basespec.ErrUnsupported,
		)
	}
	if r.sourceEntries == nil {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector Source entry inspection is unavailable",
			basespec.ErrUnsupported,
		)
	}

	selector, err := member.Selector()
	if err != nil {
		return ResolvedSelector{}, err
	}
	typeResolver, found := r.registry.Resolver(selector.Type)
	if !found || !typeResolver.SupportsSelectors() {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: Artifact type %q does not support member selectors",
			basespec.ErrUnsupported,
			selector.Type,
		)
	}

	base, err := declaration.ResolveSourceRelativePathLocator(
		selector.Base,
		from.Binding.Locator,
	)
	if err != nil {
		return ResolvedSelector{}, err
	}

	baseEntry, err := r.sourceEntries.StatSourceEntry(
		ctx,
		rootID,
		from.Binding.SourceID,
		base,
	)
	if err != nil {
		return ResolvedSelector{}, err
	}
	if err := baseEntry.Validate(); err != nil {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector base inspection: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if baseEntry.Locator != base {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector base inspection returned %q for %q",
			basespec.ErrInvalid,
			baseEntry.Locator,
			base,
		)
	}
	if !baseEntry.IsDirectory {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector base %q is not a directory",
			basespec.ErrReferenceUnresolved,
			base,
		)
	}

	records, err := r.sourceArtifacts.ListBySource(
		ctx,
		rootID,
		from.Binding.SourceID,
	)
	if err != nil {
		return ResolvedSelector{}, err
	}
	sort.SliceStable(records, func(left, right int) bool {
		if records[left].Binding.Locator != records[right].Binding.Locator {
			return records[left].Binding.Locator < records[right].Binding.Locator
		}
		if records[left].Binding.SubresourceLocator !=
			records[right].Binding.SubresourceLocator {
			return records[left].Binding.SubresourceLocator <
				records[right].Binding.SubresourceLocator
		}
		return records[left].ID < records[right].ID
	})

	output := ResolvedSelector{
		Type:    selector.Type,
		Base:    base,
		Matches: make([]ResolvedSelectorMatch, 0),
	}
	for _, record := range records {
		if record.RootID != rootID ||
			record.Binding.SourceID != from.Binding.SourceID ||
			record.Kind != artifact.ArtifactKind(selector.Type) {
			continue
		}

		relative, inside := selectorRelativePath(base, record.Binding.Locator)
		if !inside ||
			!matchesSelectorPatterns(relative, selector.Include, selector.Exclude) ||
			!matchesSelectorPatterns(
				string(record.LogicalName),
				selector.NameInclude,
				selector.NameExclude,
			) {
			continue
		}

		match := ResolvedSelectorMatch{
			Artifact: record.Ref(),
		}
		if record.State != artifact.StateAvailable {
			match.Status = ResolutionUnavailable
			match.Issue = &ResolutionIssue{
				Code:    "artifact.reference-unresolved",
				Message: "selected Artifact is unavailable",
			}
			output.Matches = append(output.Matches, match)
			continue
		}

		resolved, err := r.resolveArtifact(
			ctx,
			state,
			record.Ref(),
			selector.Type,
			"",
			depth+1,
		)
		if err == nil {
			match.Status = ResolutionAvailable
			match.Resolved = resolved
			output.Matches = append(output.Matches, match)
			continue
		}

		status, issue, partial := resolutionFailure(err)
		if !partial {
			return ResolvedSelector{}, err
		}
		match.Status = status
		match.Issue = &issue
		output.Matches = append(output.Matches, match)
	}
	return output, nil
}

func selectorRelativePath(
	base basespec.Locator,
	candidate basespec.Locator,
) (string, bool) {
	if base == "." {
		return string(candidate), true
	}
	if candidate == base {
		return ".", true
	}

	relative, found := strings.CutPrefix(
		string(candidate),
		string(base)+"/",
	)
	return relative, found && relative != ""
}

func matchesSelectorPatterns(
	value string,
	include []string,
	exclude []string,
) bool {
	if len(include) != 0 && !matchesAnyGlob(include, value) {
		return false
	}
	return !matchesAnyGlob(exclude, value)
}

func matchesAnyGlob(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if globMatchesPath(pattern, value) {
			return true
		}
	}
	return false
}

func globMatchesPath(pattern, value string) bool {
	pattern = strings.TrimPrefix(path.Clean(pattern), "./")
	value = strings.TrimPrefix(path.Clean(value), "./")
	if pattern == "." {
		return value == "."
	}
	return globSegmentsMatch(
		strings.Split(pattern, "/"),
		strings.Split(value, "/"),
	)
}

func globSegmentsMatch(
	pattern []string,
	value []string,
) bool {
	if len(pattern) == 0 {
		return len(value) == 0
	}
	if pattern[0] == "**" {
		for index := 0; index <= len(value); index++ {
			if globSegmentsMatch(pattern[1:], value[index:]) {
				return true
			}
		}
		return false
	}
	if len(value) == 0 {
		return false
	}
	matched, err := path.Match(pattern[0], value[0])
	if err != nil || !matched {
		return false
	}
	return globSegmentsMatch(pattern[1:], value[1:])
}
