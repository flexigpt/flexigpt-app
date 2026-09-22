package consumerapi

import (
	"context"
	"net/url"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/consumerutil"
)

type workspaceRefreshCoordinator struct {
	sources          compositionapi.SourceAPI
	discovery        compositionapi.DiscoveryAPI
	workspaceSources workspaceSourceRegistry
}

func newWorkspaceRefreshCoordinator(
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
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
	request resolve.SelectorRefreshRequest,
) ([]resolve.RefreshDirective, error) {
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
		defaultInclude, err := documentTopology.DiscoveryIncludePatternsForUse(
			documentTopology.DiscoveryUseSelector,
		)
		if err != nil {
			return nil, err
		}
		include = defaultInclude
	}
	next.DirectoryRoots = consumerutil.AppendDirectoryRoot(
		next.DirectoryRoots,
		source.DirectoryRoot{
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
	request resolve.LocatedMemberRefreshRequest,
) ([]resolve.RefreshDirective, error) {
	header := request.Member.Header()
	if header.Locator == nil {
		return nil, nil
	}

	target, local, err := localRefreshLocator(
		*header.Locator,
		request.Parent.Binding.Locator,
	)
	if err != nil {
		return nil, err
	}
	if !local {
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
			next.ExplicitLocators = consumerutil.AppendUniqueLocator(
				next.ExplicitLocators,
				candidate,
			)
		}
	}
	return c.updateDiscoveryForRefresh(ctx, current, next)
}

func (c *workspaceRefreshCoordinator) RefreshSource(
	ctx context.Context,
	target resolve.RefreshTarget,
) error {
	if c == nil || c.discovery == nil {
		return basespec.ErrClosed
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
	current source.Summary,
	next source.DiscoverySpec,
) ([]resolve.RefreshDirective, error) {
	next = next.Normalized()
	if err := next.Validate(); err != nil {
		return nil, err
	}

	target := resolve.RefreshTarget{
		RootID:   current.RootID,
		SourceID: current.ID,
	}
	if current.Discovery.Equal(next) {
		return []resolve.RefreshDirective{{
			Target: target,
		}}, nil
	}

	if _, err := c.sources.Update(
		ctx,
		current.RootID,
		current.ID,
		source.Update{
			ExpectedRevision: current.Revision,
			DisplayName:      current.DisplayName,
			Enabled:          current.Enabled,
			Discovery:        &next,
		},
	); err != nil {
		return nil, err
	}
	return []resolve.RefreshDirective{{
		Target:  target,
		Changed: true,
	}}, nil
}

func localRefreshLocator(
	locator declaration.Locator,
	declarationLocator basespec.Locator,
) (basespec.Locator, bool, error) {
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
	target basespec.Locator,
) []basespec.Locator {
	if declarationType != declaration.TypeSkill ||
		documentTopology.IsSkillPackageDocument(target) {
		return []basespec.Locator{target}
	}

	output := []basespec.Locator{target}
	seen := map[basespec.Locator]struct{}{
		target: {},
	}
	for _, skillDocument := range documentTopology.SkillPackageDocumentFiles() {
		candidate := basespec.Locator(path.Join(
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
