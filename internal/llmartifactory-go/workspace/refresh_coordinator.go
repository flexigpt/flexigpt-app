package workspace

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	corerefresh "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/refresh"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type workspaceRefreshCoordinator struct {
	sources          source.API
	workspaceSources workspaceSourceRegistry
	support          Support
}

func newWorkspaceRefreshCoordinator(
	sources source.API,
	workspaceSources workspaceSourceRegistry,
	support Support,
) *workspaceRefreshCoordinator {
	return &workspaceRefreshCoordinator{
		sources:          sources,
		workspaceSources: workspaceSources,
		support:          support,
	}
}

func (c *workspaceRefreshCoordinator) RequirementsForSelector(
	ctx context.Context,
	request composition.SelectorDiscoveryRequest,
) ([]corerefresh.Requirement, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	patterns, found := c.support.selectorPatternsFor(
		request.Selector.Type,
	)
	if !found {
		return nil, fmt.Errorf(
			"%w: Workspace selector type %q has no discovery contract",
			storeSpec.ErrUnsupported,
			request.Selector.Type,
		)
	}

	current, err := c.sources.Get(
		ctx,
		request.Parent.RootID,
		request.Parent.Binding.SourceID,
	)
	if err != nil {
		return nil, err
	}
	current, err = c.workspaceSources.refreshSource(ctx, current)
	if err != nil {
		return nil, err
	}

	base, err := declaration.ResolveSourceRelativePathLocator(
		request.Selector.Base,
		request.Parent.Binding.Locator,
	)
	if err != nil {
		return nil, err
	}

	// Selector.Include remains a composition-level filter. It cannot widen
	// source admission beyond the declaration-type patterns selected by the
	// application contract.
	exclude := appendUniqueDiscoveryPatterns(
		c.support.directoryExcludePatterns(),
		request.Selector.Exclude,
	)

	return []corerefresh.Requirement{{
		RootID:   current.RootID,
		SourceID: current.ID,
		Discovery: sourceModel.DiscoveryRequirement{
			DirectoryRoots: []sourceModel.DirectoryRoot{{
				Root:            base,
				Recursive:       true,
				IncludePatterns: patterns,
				ExcludePatterns: exclude,
			}},
			RequireAuthoritative: true,
		},
	}}, nil
}

func (c *workspaceRefreshCoordinator) RequirementsForLocatedMember(
	ctx context.Context,
	request composition.LocatedMemberDiscoveryRequest,
) ([]corerefresh.Requirement, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	header := request.Member.Header()
	if header.Locator == nil {
		return nil, nil
	}

	// A contained source-backed Text declaration reads raw source content
	// through Text materialization. It must not trigger declaration discovery
	// for every file below its locator.
	if header.Type == declaration.TypeText {
		return nil, nil
	}

	target, lo, err := localRefreshLocator(
		*header.Locator,
		request.Parent.Binding.Locator,
	)
	if err != nil {
		return nil, err
	}
	if !lo {
		// A URL, Git, package, archive, or future locator kind has no local
		// Source discovery behavior until its Source adapter is registered.
		return nil, nil
	}

	current, err := c.sources.Get(
		ctx,
		request.Parent.RootID,
		request.Parent.Binding.SourceID,
	)
	if err != nil {
		return nil, err
	}
	current, err = c.workspaceSources.refreshSource(ctx, current)
	if err != nil {
		return nil, err
	}

	requirement := sourceModel.DiscoveryRequirement{
		RequireAuthoritative: true,
	}
	candidates := c.locatedRefreshCandidates(header.Type, target)
	for _, candidate := range candidates {
		inScope, err := current.Discovery.InScope(candidate)
		if err != nil {
			return nil, err
		}
		if !inScope {
			requirement.ExplicitLocators = append(
				requirement.ExplicitLocators,
				candidate,
			)
		}
	}
	if len(requirement.ExplicitLocators) == 0 {
		return nil, nil
	}
	return []corerefresh.Requirement{{
		RootID:    current.RootID,
		SourceID:  current.ID,
		Discovery: requirement,
	}}, nil
}

func localRefreshLocator(
	locator declaration.Locator,
	declarationLocator storeSpec.Locator,
) (storeSpec.Locator, bool, error) {
	if err := locator.Validate(); err != nil {
		return "", false, err
	}
	switch locator.Kind {
	case "":
		parsed, err := url.Parse(locator.Scalar)
		if err != nil {
			return "", false, err
		}
		if parsed.Scheme != "" {
			return "", false, nil
		}
	case declaration.LocatorKindPath:
	default:
		return "", false, nil
	}

	value, err := declaration.ResolveSourceRelativePathLocator(
		locator,
		declarationLocator,
	)
	return value, true, err
}

func (c *workspaceRefreshCoordinator) locatedRefreshCandidates(
	declarationType declaration.Type,
	target storeSpec.Locator,
) []storeSpec.Locator {
	if declarationType != declaration.TypeSkill ||
		c.support.SkillDocuments.Matches(target) {
		return []storeSpec.Locator{target}
	}

	output := []storeSpec.Locator{target}
	seen := map[storeSpec.Locator]struct{}{
		target: {},
	}
	for _, skillDocument := range c.support.SkillDocuments.Files {
		candidate := storeSpec.Locator(path.Join(
			string(target),
			string(skillDocument),
		))
		if _, duplicate := seen[candidate]; duplicate {
			continue
		}
		seen[candidate] = struct{}{}
		output = append(output, candidate)
	}
	return output
}

func appendUniqueDiscoveryPatterns(
	groups ...[]string,
) []string {
	seen := make(map[string]struct{})
	output := make([]string, 0)
	for _, group := range groups {
		for _, pattern := range group {
			if _, duplicate := seen[pattern]; duplicate {
				continue
			}
			seen[pattern] = struct{}{}
			output = append(output, pattern)
		}
	}
	return output
}
