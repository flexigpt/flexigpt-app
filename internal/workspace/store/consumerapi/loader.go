package consumerapi

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
	workspaceProviderAPI "github.com/flexigpt/flexigpt-app/internal/workspace/store/providerapi"
)

// applyWorkspaceDeclarations converts Workspace-local declaration sources into
// Source-owned Store discovery configuration.
//
// Local path declarations reuse the physical Source that contains the
// Workspace Artifact. URL, Git, package, and command locators are deliberately
// left to locator resolver integrations outside the generic Store.
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
	desired, err := workspaceDiscoveryForDocument(
		sourceValue.Discovery,
		workspace.Artifact.Binding.Locator,
		workspace.Document,
	)
	if err != nil {
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
							workspaceProviderAPI.ContextMarkdownDecoderID,
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
						workspaceProviderAPI.ContextMarkdownDecoderID,
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
	locator artifactcontract.Locator,
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

	case artifactcontract.LocatorKindPath:
		// Resolved below.

	default:
		return "", false, nil
	}

	value, err := artifactcontract.ResolveSourceRelativePathLocator(
		locator,
		declarationLocator,
	)
	return value, true, err
}

func isContextMarkdownLocator(
	value basespec.Locator,
) bool {
	if !strings.EqualFold(path.Ext(string(value)), ".md") {
		return false
	}
	switch strings.ToUpper(path.Base(string(value))) {
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
	for index := range values {
		if values[index].Root != value.Root {
			continue
		}
		values[index].Recursive =
			values[index].Recursive || value.Recursive
		values[index].IncludePatterns = mergePatterns(
			values[index].IncludePatterns,
			value.IncludePatterns,
		)
		values[index].ExcludePatterns = mergePatterns(
			values[index].ExcludePatterns,
			value.ExcludePatterns,
		)
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

func mergePatterns(
	left []string,
	right []string,
) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	output := append([]string(nil), left...)
	for _, value := range right {
		if !slices.Contains(output, value) {
			output = append(output, value)
		}
	}
	return output
}
