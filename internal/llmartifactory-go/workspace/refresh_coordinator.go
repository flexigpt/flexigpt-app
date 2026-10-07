package workspace

import (
	"context"
	"net/url"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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

	include := append([]string(nil), request.Selector.Include...)
	if len(include) == 0 {
		include = append([]string(nil), c.support.SelectorIncludePatterns...)
	}
	return []corerefresh.Requirement{{
		RootID:   current.RootID,
		SourceID: current.ID,
		Discovery: sourceModel.DiscoveryRequirement{
			DirectoryRoots: []sourceModel.DirectoryRoot{{
				Root:            base,
				Recursive:       true,
				IncludePatterns: include,
				ExcludePatterns: append([]string(nil), request.Selector.Exclude...),
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
	declarationLocator spec.Locator,
) (spec.Locator, bool, error) {
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
	target spec.Locator,
) []spec.Locator {
	if declarationType != declaration.TypeSkill ||
		c.support.SkillDocuments.Matches(target) {
		return []spec.Locator{target}
	}

	output := []spec.Locator{target}
	seen := map[spec.Locator]struct{}{
		target: {},
	}
	for _, skillDocument := range c.support.SkillDocuments.Files {
		candidate := spec.Locator(path.Join(
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
