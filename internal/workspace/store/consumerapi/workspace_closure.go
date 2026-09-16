package consumerapi

import (
	"context"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/format/markdown"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

type workspaceClosureSource struct {
	rootID   root.RootID
	sourceID source.SourceID
}

func (a *StoreAPI) expandWorkspaceLocatorClosure(
	ctx context.Context,
	graph resolve.Graph,
) ([]workspaceClosureSource, error) {
	if graph.Root == nil {
		return nil, fmt.Errorf(
			"%w: Workspace resolution graph has no root",
			basespec.ErrReferenceUnresolved,
		)
	}

	closure, err := workspaceLocatorClosure(graph.Root)
	if err != nil {
		return nil, err
	}
	keys := make([]workspaceClosureSource, 0, len(closure))
	for key := range closure {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		if keys[left].rootID != keys[right].rootID {
			return keys[left].rootID < keys[right].rootID
		}
		return keys[left].sourceID < keys[right].sourceID
	})

	changed := make([]workspaceClosureSource, 0)
	for _, key := range keys {
		current, err := a.sources.Get(ctx, key.rootID, key.sourceID)
		if err != nil {
			return nil, err
		}
		next := current.Discovery.Clone()
		for _, locator := range closure[key] {
			inScope, err := next.InScope(locator)
			if err != nil {
				return nil, err
			}
			if !inScope {
				next.ExplicitLocators = appendUniqueLocator(
					next.ExplicitLocators,
					locator,
				)
			}
			next = configureClosureDecoder(next, locator)
		}
		next = next.Normalized()
		if err := next.Validate(); err != nil {
			return nil, err
		}
		if current.Discovery.Equal(next) {
			continue
		}
		if _, err := a.sources.Update(
			ctx,
			key.rootID,
			key.sourceID,
			source.Update{
				ExpectedRevision: current.Revision,
				DisplayName:      current.DisplayName,
				Enabled:          current.Enabled,
				Discovery:        &next,
			},
		); err != nil {
			return nil, err
		}
		changed = append(changed, key)
	}
	return changed, nil
}

func workspaceLocatorClosure(
	rootEntry *resolve.ResolvedEntry,
) (map[workspaceClosureSource][]basespec.Locator, error) {
	output := make(map[workspaceClosureSource][]basespec.Locator)
	visited := make(map[*resolve.ResolvedEntry]struct{})

	add := func(
		origin *artifact.Artifact,
		entry declaration.Entry,
	) error {
		if origin == nil {
			return nil
		}
		form, err := entry.CompositionForm()
		if err != nil {
			return err
		}
		header := entry.Header()
		if form != declaration.CompositionEntryReference ||
			header.Locator == nil {
			return nil
		}

		target, local, err := localSourceLocator(
			*header.Locator,
			origin.Binding.Locator,
		)
		if err != nil || !local {
			return err
		}
		for _, candidate := range closureCandidates(header.Type, target) {
			key := workspaceClosureSource{
				rootID:   origin.RootID,
				sourceID: origin.Binding.SourceID,
			}
			if !containsLocator(output[key], candidate) {
				output[key] = append(output[key], candidate)
			}
		}
		return nil
	}

	var walkRelationship func(
		*resolve.ResolvedEntry,
		resolve.ResolvedRelationship,
	) error
	var walkEntry func(*resolve.ResolvedEntry) error

	walkRelationship = func(
		parent *resolve.ResolvedEntry,
		relationship resolve.ResolvedRelationship,
	) error {
		if parent == nil {
			return nil
		}
		origin := parent.Artifact
		if origin == nil {
			origin = parent.DeclarationOrigin
		}
		if err := add(origin, relationship.Declared); err != nil {
			return err
		}
		return walkEntry(relationship.Resolved)
	}

	walkEntry = func(value *resolve.ResolvedEntry) error {
		if value == nil {
			return nil
		}
		if _, seen := visited[value]; seen {
			return nil
		}
		visited[value] = struct{}{}

		for _, relationship := range value.MemberResults {
			if err := walkRelationship(value, relationship); err != nil {
				return err
			}
		}
		for _, relationship := range value.AllowedToolResults {
			if err := walkRelationship(value, relationship); err != nil {
				return err
			}
		}
		if value.ProgramResult != nil {
			if err := walkRelationship(value, *value.ProgramResult); err != nil {
				return err
			}
		}
		if value.Loop != nil && value.Loop.BodyResult != nil {
			if err := walkRelationship(value, *value.Loop.BodyResult); err != nil {
				return err
			}
		}
		if value.Workflow != nil {
			for _, node := range value.Workflow.Nodes {
				if node.TargetResult == nil {
					continue
				}
				if err := walkRelationship(value, *node.TargetResult); err != nil {
					return err
				}
			}
		}
		if value.Workspace != nil {
			for _, relationship := range value.Workspace.RootResults {
				if err := walkRelationship(value, relationship); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := walkEntry(rootEntry); err != nil {
		return nil, err
	}
	for key := range output {
		slices.Sort(output[key])
	}
	return output, nil
}

func closureCandidates(
	declarationType declaration.Type,
	target basespec.Locator,
) []basespec.Locator {
	if declarationType != declaration.TypeSkill ||
		path.Base(string(target)) == string(skillDomain.SkillDefinitionFileName) {
		return []basespec.Locator{target}
	}
	return []basespec.Locator{basespec.Locator(path.Join(
		string(target),
		string(skillDomain.SkillDefinitionFileName),
	))}
}

func configureClosureDecoder(
	value source.DiscoverySpec,
	locator basespec.Locator,
) source.DiscoverySpec {
	value.DecoderHints = appendCanonicalDeclarationDecoderHint(
		value.DecoderHints,
		value.AllowedDecoderIDs,
		locator,
	)

	name := strings.ToLower(path.Base(string(locator)))
	switch {
	case strings.EqualFold(name, string(skillDomain.SkillDefinitionFileName)):
		value.DecoderHints = appendDecoderHint(
			value.DecoderHints,
			source.DecoderHint{
				Locator: locator,
				DecoderIDs: []basespec.DecoderID{
					skillDomain.MarkdownDecoderID,
				},
			},
		)
		value.AllowedDecoderIDs = appendAllowedDecoder(
			value.AllowedDecoderIDs,
			skillDomain.MarkdownDecoderID,
		)

	case name == ".mcp.json" || name == "mcp.json":
		value.DecoderHints = appendDecoderHint(
			value.DecoderHints,
			source.DecoderHint{
				Locator: locator,
				DecoderIDs: []basespec.DecoderID{
					mcpDomain.SourceDecoderID,
				},
			},
		)
		value.AllowedDecoderIDs = appendAllowedDecoder(
			value.AllowedDecoderIDs,
			mcpDomain.SourceDecoderID,
		)

	case isContextMarkdownLocator(locator):
		value.DecoderHints = appendDecoderHint(
			value.DecoderHints,
			source.DecoderHint{
				Locator: locator,
				DecoderIDs: []basespec.DecoderID{
					markdown.ContextMarkdownDecoderID,
				},
			},
		)
		value.AllowedDecoderIDs = appendAllowedDecoder(
			value.AllowedDecoderIDs,
			markdown.ContextMarkdownDecoderID,
		)
	}
	return value
}

func appendAllowedDecoder(
	values []basespec.DecoderID,
	value basespec.DecoderID,
) []basespec.DecoderID {
	if len(values) == 0 || containsDecoder(values, value) {
		return values
	}
	return append(values, value)
}

func containsDecoder(
	values []basespec.DecoderID,
	target basespec.DecoderID,
) bool {
	return slices.Contains(values, target)
}

func containsLocator(
	values []basespec.Locator,
	target basespec.Locator,
) bool {
	return slices.Contains(values, target)
}
