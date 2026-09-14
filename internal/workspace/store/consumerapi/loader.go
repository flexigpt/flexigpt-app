package consumerapi

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/format/markdown"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

// applyWorkspaceDeclarations converts Workspace-local declaration sources into
// Source-owned Store discovery configuration.
//
// Local path declarations reuse the physical Source that contains the
// Workspace Artifact. URL, Git, package, archive, and zip sources require
// dedicated Source adapters and locator resolvers. Command locators remain
// runtime implementation data and are not opened by Workspace discovery.
func (a *StoreAPI) applyWorkspaceDeclarations(
	ctx context.Context,
	ref WorkspaceRef,
) (workspaceDomain.Workspace, error) {
	workspace, err := a.GetWorkspace(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}

	sourceValue, err := a.sources.Get(
		ctx,
		workspace.Artifact.RootID,
		workspace.Artifact.Binding.SourceID,
	)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}

	base, err := a.workspaceDiscoveryBase(
		sourceValue,
		workspace.Artifact.Binding.Locator,
	)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	desired, err := workspaceDiscoveryForDocument(
		base,
		workspace.Artifact.Binding.Locator,
		workspace.Document,
	)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	if err := a.expandWorkspaceDirectoryDeclarations(
		ctx,
		sourceValue,
		workspace.Artifact.Binding.Locator,
		workspace.Document,
		&desired,
	); err != nil {
		return workspaceDomain.Workspace{}, err
	}
	if !sourceValue.Discovery.Equal(desired) {
		update := source.Update{
			ExpectedRevision: sourceValue.Revision,
			DisplayName:      sourceValue.DisplayName,
			Enabled:          sourceValue.Enabled,
			Discovery:        &desired,
		}
		if _, err := a.sources.Update(
			ctx,
			sourceValue.RootID,
			sourceValue.ID,
			update,
		); err != nil {
			return workspaceDomain.Workspace{}, err
		}
	}
	return workspace, nil
}

// workspaceDiscoveryBase prevents old Workspace declarations from remaining
// in Source discovery after they were removed from a Workspace manifest.
//
// Sources created by this Workspace API use the workspace-* storage-key
// namespace and are owned by the Workspace Loader. Other manually registered
// Sources retain their caller-managed baseline discovery configuration.
func (a *StoreAPI) workspaceDiscoveryBase(
	current source.Summary,
	declarationLocator basespec.Locator,
) (source.DiscoverySpec, error) {
	output := current.Discovery.Clone()
	if strings.HasPrefix(
		string(current.StorageKey),
		"workspace-",
	) {
		var err error
		output, err = a.defaultDiscovery()
		if err != nil {
			return source.DiscoverySpec{}, err
		}
	}
	output.ExplicitLocators = appendUniqueLocator(
		output.ExplicitLocators,
		declarationLocator,
	)
	output = output.Normalized()
	if err := output.Validate(); err != nil {
		return source.DiscoverySpec{}, err
	}
	return output, nil
}

func (a *StoreAPI) refreshAndResolveWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
) (
	workspaceDomain.Workspace,
	refresh.RefreshRootResult,
	resolve.Graph,
	error,
) {
	if a == nil || a.resolver == nil {
		return workspaceDomain.Workspace{},
			refresh.RefreshRootResult{},
			resolve.Graph{},
			basespec.ErrClosed
	}

	workspace, err := a.applyWorkspaceDeclarations(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{},
			refresh.RefreshRootResult{},
			resolve.Graph{},
			err
	}
	result, err := a.discovery.RefreshRoot(
		ctx,
		workspace.Artifact.RootID,
	)
	if err != nil {
		return workspaceDomain.Workspace{},
			refresh.RefreshRootResult{},
			resolve.Graph{},
			err
	}
	workspace, err = a.GetWorkspace(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{},
			refresh.RefreshRootResult{},
			resolve.Graph{},
			err
	}
	graph, err := a.resolver.ResolveArtifact(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{},
			refresh.RefreshRootResult{},
			resolve.Graph{},
			err
	}
	if graph.Root == nil ||
		graph.Root.Type != declaration.TypeWorkspace ||
		graph.Root.Workspace == nil {
		return workspaceDomain.Workspace{},
			refresh.RefreshRootResult{},
			resolve.Graph{},
			fmt.Errorf(
				"%w: Artifact %q did not resolve as a Workspace",
				basespec.ErrReferenceUnresolved,
				ref.ArtifactID,
			)
	}
	return workspace, result, graph, nil
}

// ResolveWorkspaceGraph applies local declaration sources, refreshes the
// Root, and resolves every Workspace root in declaration order.
//
// The returned graph is a Go consumer API. It is intentionally not exposed
// through the Wails wrapper because Loop bodies may create graph cycles.
func (a *StoreAPI) ResolveWorkspaceGraph(
	ctx context.Context,
	ref WorkspaceRef,
) (resolve.Graph, error) {
	_, _, graph, err := a.refreshAndResolveWorkspace(ctx, ref)
	if err != nil {
		return resolve.Graph{}, err
	}
	return graph, nil
}

func (a *StoreAPI) expandWorkspaceDirectoryDeclarations(
	ctx context.Context,
	sourceValue source.Summary,
	declarationLocator basespec.Locator,
	document workspacev1.WorkspaceDocument,
	discovery *source.DiscoverySpec,
) error {
	if discovery == nil {
		return fmt.Errorf("%w: Workspace discovery target is nil", basespec.ErrInvalid)
	}
	for index, declarationSource := range document.Declarations {
		locator, isLocator, err := declarationSource.AsLocator()
		if err != nil {
			return err
		}
		if !isLocator {
			continue
		}

		local, isLocal, err := localSourceLocator(
			locator,
			declarationLocator,
		)
		if err != nil {
			return err
		}
		if !isLocal {
			continue
		}
		entry, err := a.resources.StatSourceEntry(
			ctx,
			sourceValue.RootID,
			sourceValue.ID,
			local,
		)
		if err != nil {
			return fmt.Errorf(
				"workspace declarations[%d]: %w",
				index,
				err,
			)
		}
		if !entry.IsDirectory {
			continue
		}
		next := discovery.Clone()
		next.DirectoryRoots = appendDirectoryRoot(
			next.DirectoryRoots,
			workspaceDirectoryRoot(local),
		)
		next.DecoderHints = appendDecoderHint(
			next.DecoderHints,
			source.DecoderHint{
				Locator:   local,
				Recursive: true,
				DecoderIDs: []basespec.DecoderID{
					markdown.ContextMarkdownDecoderID,
				},
			},
		)
		*discovery = next
	}
	return nil
}

func workspaceDirectoryRoot(
	base basespec.Locator,
) source.DirectoryRoot {
	return source.DirectoryRoot{
		Root:      base,
		Recursive: true,
		IncludePatterns: []string{
			"**/*.json",
			"**/*.yaml",
			"**/*.yml",
			"**/*.md",
			"**/llms.txt",
		},
	}
}

func workspaceDiscoveryForDocument(
	current source.DiscoverySpec,
	declarationLocator basespec.Locator,
	document workspacev1.WorkspaceDocument,
) (source.DiscoverySpec, error) {
	output := current.Clone()

	for index, declaration := range document.Declarations {
		locator, isLocator, err := declaration.AsLocator()
		if err != nil {
			return source.DiscoverySpec{}, err
		}
		if isLocator {
			value, local, err := localSourceLocator(
				locator,
				declarationLocator,
			)
			if err != nil {
				return source.DiscoverySpec{}, err
			}
			if !local {
				return source.DiscoverySpec{}, fmt.Errorf(
					"%w: Workspace declaration source %d requires a locator resolver",
					basespec.ErrLocatorUnresolved,
					index,
				)
			}
			output.ExplicitLocators = appendUniqueLocator(
				output.ExplicitLocators,
				value,
			)
			if isContextMarkdownLocator(value) {
				output.DecoderHints = appendDecoderHint(
					output.DecoderHints,
					source.DecoderHint{
						Locator:   value,
						Recursive: false,
						DecoderIDs: []basespec.DecoderID{
							markdown.ContextMarkdownDecoderID,
						},
					},
				)
			}
			continue
		}

		scan, isScan, err := declaration.AsScan()
		if err != nil {
			return source.DiscoverySpec{}, err
		}
		if isScan {
			base, local, err := localSourceLocator(
				scan.Base,
				declarationLocator,
			)
			if err != nil {
				return source.DiscoverySpec{}, err
			}
			if !local {
				return source.DiscoverySpec{}, fmt.Errorf(
					"%w: Workspace declaration scan %d requires a locator resolver",
					basespec.ErrLocatorUnresolved,
					index,
				)
			}
			output.DirectoryRoots = appendDirectoryRoot(
				output.DirectoryRoots,
				source.DirectoryRoot{
					Root:            base,
					Recursive:       true,
					IncludePatterns: append([]string(nil), scan.Include...),
					ExcludePatterns: append([]string(nil), scan.Exclude...),
				},
			)
			output.DecoderHints = appendDecoderHint(
				output.DecoderHints,
				source.DecoderHint{
					Locator:   base,
					Recursive: true,
					DecoderIDs: []basespec.DecoderID{
						markdown.ContextMarkdownDecoderID,
					},
				},
			)
			continue
		}

		// Inline declaration sources are emitted as named subresources by the
		// canonical Workspace declaration decoder. They add no Source scan
		// configuration of their own.
		if _, inline, err := declaration.AsEntry(); err != nil {
			return source.DiscoverySpec{}, err
		} else if inline {
			continue
		}

		return source.DiscoverySpec{}, fmt.Errorf(
			"%w: Workspace declaration source %d has no supported form",
			basespec.ErrInvalid,
			index,
		)
	}

	output = output.Normalized()
	if err := output.Validate(); err != nil {
		return source.DiscoverySpec{}, err
	}
	return output, nil
}

func localSourceLocator(
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
		// Resolved below.

	default:
		return "", false, nil
	}

	value, err := declaration.ResolveSourceRelativePathLocator(
		locator,
		declarationLocator,
	)
	return value, true, err
}

func isContextMarkdownLocator(
	value basespec.Locator,
) bool {
	base := strings.ToLower(path.Base(string(value)))
	if base == "agent.md" ||
		strings.HasSuffix(base, ".agent.md") {
		return false
	}
	if strings.EqualFold(path.Base(string(value)), "llms.txt") {
		return true
	}
	if !strings.EqualFold(path.Ext(string(value)), ".md") {
		return false
	}
	switch strings.ToUpper(base) {
	case "AGENTS.MD", "CLAUDE.MD", "SKILL.MD":
		return false
	default:
		return true
	}
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

func appendDirectoryRoot(
	values []source.DirectoryRoot,
	value source.DirectoryRoot,
) []source.DirectoryRoot {
	for _, current := range values {
		if current.Root == value.Root &&
			current.Recursive == value.Recursive &&
			slices.Equal(
				current.IncludePatterns,
				value.IncludePatterns,
			) &&
			slices.Equal(
				current.ExcludePatterns,
				value.ExcludePatterns,
			) {
			return values
		}
	}
	// Include/exclude pairs are one declaration scan's semantics. Merging
	// scans sharing a base changes exclusion behavior, so preserve scopes.
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
			if !slices.Contains(
				values[index].DecoderIDs,
				decoderID,
			) {
				values[index].DecoderIDs = append(
					values[index].DecoderIDs,
					decoderID,
				)
			}
		}
		return values
	}
	return append(values, value.Clone())
}
