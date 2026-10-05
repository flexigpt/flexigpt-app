package consumerapi

import (
	"context"
	"net/url"
	"path"

	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type workspaceRefreshCoordinator struct {
	sources          source.API
	discovery        refreshFlow.API
	workspaceSources workspaceSourceRegistry
}

func newWorkspaceRefreshCoordinator(
	sources source.API,
	discovery refreshFlow.API,
	workspaceSources workspaceSourceRegistry,
) *workspaceRefreshCoordinator {
	return &workspaceRefreshCoordinator{
		sources:          sources,
		discovery:        discovery,
		workspaceSources: workspaceSources,
	}
}

func (c *workspaceRefreshCoordinator) PrepareSelectorDiscovery(
	ctx context.Context,
	request composition.SelectorRefreshRequest,
) ([]composition.RefreshDirective, error) {
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

	next := current.Discovery.Clone()
	include := append([]string(nil), request.Selector.Include...)
	if len(include) == 0 {
		defaultInclude, err := topology.DiscoveryIncludePatternsForUse(
			topology.DiscoveryUseSelector,
		)
		if err != nil {
			return nil, err
		}
		include = defaultInclude
	}
	next.DirectoryRoots = source.AppendDirectoryRoot(
		next.DirectoryRoots,
		sourceModel.DirectoryRoot{
			Root:            base,
			Recursive:       true,
			IncludePatterns: include,
			ExcludePatterns: append([]string(nil), request.Selector.Exclude...),
		},
	)
	return c.updateDiscoveryForRefresh(ctx, current, next)
}

func (c *workspaceRefreshCoordinator) PrepareLocatedMemberDiscovery(
	ctx context.Context,
	request composition.LocatedMemberRefreshRequest,
) ([]composition.RefreshDirective, error) {
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

	next := current.Discovery.Clone()
	candidates := locatedRefreshCandidates(header.Type, target)
	for _, candidate := range candidates {
		inScope, err := next.InScope(candidate)
		if err != nil {
			return nil, err
		}
		if !inScope {
			next.ExplicitLocators = source.AppendUniqueLocator(
				next.ExplicitLocators,
				candidate,
			)
		}
	}
	return c.updateDiscoveryForRefresh(ctx, current, next)
}

func (c *workspaceRefreshCoordinator) RefreshSource(
	ctx context.Context,
	target composition.RefreshTarget,
) error {
	if c == nil || c.discovery == nil {
		return spec.ErrClosed
	}
	_, err := c.discovery.RefreshSource(
		ctx,
		target.RootID,
		target.SourceID,
	)
	return err
}

func (c *workspaceRefreshCoordinator) updateDiscoveryForRefresh(
	ctx context.Context,
	current sourceModel.Summary,
	next sourceModel.DiscoverySpec,
) ([]composition.RefreshDirective, error) {
	next = next.Normalized()
	if err := next.Validate(); err != nil {
		return nil, err
	}

	target := composition.RefreshTarget{
		RootID:   current.RootID,
		SourceID: current.ID,
	}
	if current.Discovery.Equal(next) {
		return []composition.RefreshDirective{{
			Target: target,
		}}, nil
	}

	if _, err := c.sources.Update(
		ctx,
		current.RootID,
		current.ID,
		sourceModel.Update{
			ExpectedRevision: current.Revision,
			DisplayName:      current.DisplayName,
			Enabled:          current.Enabled,
			Discovery:        &next,
		},
	); err != nil {
		return nil, err
	}
	return []composition.RefreshDirective{{
		Target:  target,
		Changed: true,
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

func locatedRefreshCandidates(
	declarationType declaration.Type,
	target spec.Locator,
) []spec.Locator {
	if declarationType != declaration.TypeSkill ||
		topology.IsSkillPackageDocument(target) {
		return []spec.Locator{target}
	}

	output := []spec.Locator{target}
	seen := map[spec.Locator]struct{}{
		target: {},
	}
	for _, skillDocument := range topology.SkillPackageDocumentFiles() {
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
