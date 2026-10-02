package resolve

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	catalog "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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
			spec.ErrSourceUnavailable,
		)
	}
	if r.sourceArtifacts == nil {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector source enumeration is unavailable",
			spec.ErrUnsupported,
		)
	}
	if r.sourceEntries == nil {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector Source entry inspection is unavailable",
			spec.ErrUnsupported,
		)
	}

	effectiveSourceID := from.Binding.SourceID
	if state.usesCompositionSource(rootID) {
		effectiveSourceID = state.compositionSourceID
	}

	selector, err := member.Selector()
	if err != nil {
		return ResolvedSelector{}, err
	}
	typeResolver, found := r.registry.Resolver(selector.Type)
	if !found || !typeResolver.SupportsSelectors() {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: Artifact type %q does not support member selectors",
			spec.ErrUnsupported,
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
		effectiveSourceID,
		base,
	)
	if err != nil {
		return ResolvedSelector{}, err
	}
	if err := baseEntry.Validate(); err != nil {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector base inspection: %w",
			spec.ErrInvalid,
			err,
		)
	}
	if baseEntry.Locator != base {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector base inspection returned %q for %q",
			spec.ErrInvalid,
			baseEntry.Locator,
			base,
		)
	}
	if !baseEntry.IsDirectory {
		return ResolvedSelector{}, fmt.Errorf(
			"%w: member selector base %q is not a directory",
			spec.ErrReferenceUnresolved,
			base,
		)
	}

	sourceSelection, err := spec.NewPathSelection(
		selector.Include,
		selector.Exclude,
	)
	if err != nil {
		return ResolvedSelector{}, err
	}
	nameSelection, err := spec.NewPathSelection(
		selector.NameInclude,
		selector.NameExclude,
	)
	if err != nil {
		return ResolvedSelector{}, err
	}

	records, err := r.sourceArtifacts.ListBySource(
		ctx,
		rootID,
		effectiveSourceID,
		catalog.ListOptions{},
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
			record.Binding.SourceID != effectiveSourceID ||
			record.Kind != artifact.ArtifactKind(selector.Type) {
			continue
		}

		relative, inside := selectorRelativePath(base, record.Binding.Locator)
		if !inside {
			continue
		}

		sourceMatched, err := sourceSelection.Match(relative)
		if err != nil {
			return ResolvedSelector{}, err
		}
		if !sourceMatched {
			continue
		}
		nameMatched, err := nameSelection.Match(
			string(record.LogicalName),
		)
		if err != nil {
			return ResolvedSelector{}, err
		}
		if !nameMatched {
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
	base spec.Locator,
	candidate spec.Locator,
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
