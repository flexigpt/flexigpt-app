package consumerapi

import (
	"context"
	"net/url"
	"path"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func (a *StoreAPI) PrepareSelectorDiscovery(
	ctx context.Context,
	request resolve.SelectorRefreshRequest,
) ([]resolve.RefreshDirective, error) {
	current, err := a.sources.Get(
		ctx,
		request.Parent.RootID,
		request.Parent.Binding.SourceID,
	)
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
		include = []string{
			"**/*.json",
			"**/*.yaml",
			"**/*.yml",
			"**/SKILL.md",
			".mcp.json",
			"mcp.json",
		}
	}
	next.DirectoryRoots = appendWorkspaceDirectoryRoot(
		next.DirectoryRoots,
		source.DirectoryRoot{
			Root:            base,
			Recursive:       true,
			IncludePatterns: include,
			ExcludePatterns: append([]string(nil), request.Selector.Exclude...),
		},
	)
	return a.updateDiscoveryForRefresh(ctx, current, next)
}

func (a *StoreAPI) PrepareLocatedMemberDiscovery(
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

	current, err := a.sources.Get(
		ctx,
		request.Parent.RootID,
		request.Parent.Binding.SourceID,
	)
	if err != nil {
		return nil, err
	}
	next := current.Discovery.Clone()
	for _, candidate := range locatedRefreshCandidates(header.Type, target) {
		inScope, err := next.InScope(candidate)
		if err != nil {
			return nil, err
		}
		if !inScope {
			next.ExplicitLocators = appendUniqueLocator(
				next.ExplicitLocators,
				candidate,
			)
		}
	}
	return a.updateDiscoveryForRefresh(ctx, current, next)
}

func (a *StoreAPI) RefreshSource(
	ctx context.Context,
	target resolve.RefreshTarget,
) error {
	if a == nil || a.discovery == nil {
		return basespec.ErrClosed
	}
	_, err := a.discovery.RefreshSource(
		ctx,
		target.RootID,
		target.SourceID,
	)
	return err
}

func (a *StoreAPI) updateDiscoveryForRefresh(
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

	if _, err := a.sources.Update(
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
		path.Base(string(target)) == "SKILL.md" {
		return []basespec.Locator{target}
	}
	return []basespec.Locator{
		target,
		basespec.Locator(path.Join(string(target), "SKILL.md")),
	}
}

func appendWorkspaceDirectoryRoot(
	values []source.DirectoryRoot,
	value source.DirectoryRoot,
) []source.DirectoryRoot {
	for _, current := range values {
		if current.Root != value.Root ||
			current.Recursive != value.Recursive ||
			!slices.Equal(current.IncludePatterns, value.IncludePatterns) ||
			!slices.Equal(current.ExcludePatterns, value.ExcludePatterns) {
			continue
		}
		return values
	}
	return append(values, value.Clone())
}

func appendDecoderHint(
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

func appendUniqueLocator(
	values []basespec.Locator,
	value basespec.Locator,
) []basespec.Locator {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}
