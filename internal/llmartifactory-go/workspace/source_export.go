package workspace

import (
	"context"
	"fmt"
	"path"
	"strings"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
)

// ExportWorkspaceSource returns verified source text for a top-level Workspace
// manifest owned by directory. It supports both custom directory manifests and
// the application default Workspace policy Source.
//
// It returns original source bytes rather than a Definition body. Native paths,
// Source configuration, Artifact data, and refresh state remain internal.
func (a *Service) ExportWorkspaceSource(
	ctx context.Context,
	directory WorkspaceDirectoryRef,
	ref artifactModel.ArtifactRef,
) (WorkspaceSourceExport, error) {
	if err := directory.RootID.Validate(); err != nil {
		return WorkspaceSourceExport{}, err
	}

	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (WorkspaceSourceExport, error) {
			return a.exportWorkspaceSource(
				sessionCtx,
				directory,
				ref,
			)
		},
	)
}

func (a *Service) exportWorkspaceSource(
	ctx context.Context,
	directory WorkspaceDirectoryRef,
	ref artifactModel.ArtifactRef,
) (WorkspaceSourceExport, error) {
	sources, err := a.workspaceSources.required(ctx, directory.RootID)
	if err != nil {
		return WorkspaceSourceExport{}, err
	}

	terminal, err := a.resolver.ResolveTerminalArtifact(ctx, ref)
	if err != nil {
		return WorkspaceSourceExport{}, err
	}
	if terminal.RootID != directory.RootID {
		return WorkspaceSourceExport{}, fmt.Errorf(
			"%w: Workspace belongs to another directory Root",
			workspaceDomain.ErrReferenceUnresolved,
		)
	}

	resolved, err := a.resources.ResolveArtifact(
		ctx,
		terminal,
		resourceModel.ResolveOptions{},
	)
	if err != nil {
		return WorkspaceSourceExport{}, err
	}
	if resolved.Artifact.Kind != workspaceDomain.WorkspaceArtifactKind {
		return WorkspaceSourceExport{}, fmt.Errorf(
			"%w: Artifact is not a Workspace",
			workspaceDomain.ErrReferenceUnresolved,
		)
	}
	if !a.workspaceSourceExportOrigin(sources, resolved.Artifact) {
		return WorkspaceSourceExport{}, fmt.Errorf(
			"%w: Workspace source is not exportable from this directory",
			workspaceDomain.ErrReferenceUnresolved,
		)
	}
	if resolved.Artifact.SourceContentDigest == nil {
		return WorkspaceSourceExport{}, fmt.Errorf(
			"%w: Workspace source has no content digest",
			storeSpec.ErrDigestMismatch,
		)
	}

	entry, err := a.resources.ReadSourceEntry(
		ctx,
		resolved.Artifact.RootID,
		resolved.Artifact.Binding.SourceID,
		resolved.Artifact.Binding.Locator,
		storeSpec.MaxCandidateBytes,
	)
	if err != nil {
		return WorkspaceSourceExport{}, err
	}
	if entry.SourceGeneration != resolved.RefreshState.SourceGeneration ||
		entry.Digest != *resolved.Artifact.SourceContentDigest {
		return WorkspaceSourceExport{}, fmt.Errorf(
			"%w: Workspace source changed during export",
			storeSpec.ErrRefreshRequired,
		)
	}

	format, err := workspaceSourceExportFormat(
		resolved.Artifact.Binding.Locator,
	)
	if err != nil {
		return WorkspaceSourceExport{}, err
	}

	return WorkspaceSourceExport{
		Content:           string(entry.Content),
		SuggestedFileName: path.Base(string(resolved.Artifact.Binding.Locator)),
		Format:            format,
	}, nil
}

func (a *Service) workspaceSourceExportOrigin(
	sources workspaceSourceSet,
	artifact artifactModel.Artifact,
) bool {
	if artifact.RootID != sources.Directory.RootID ||
		artifact.Binding.SubresourceLocator != "" {
		return false
	}

	switch artifact.Binding.SourceID {
	case sources.Directory.ID:
		return a.isWorkspaceManifestLocator(
			artifact.Binding.Locator,
		)
	case sources.Policy.ID:
		return artifact.Binding.Locator == a.policySource.Locator
	default:
		return false
	}
}

func workspaceSourceExportFormat(
	locator storeSpec.Locator,
) (WorkspaceSourceFormat, error) {
	switch strings.ToLower(path.Ext(string(locator))) {
	case ".yaml", ".yml":
		return WorkspaceSourceFormatYAML, nil
	case ".json":
		return WorkspaceSourceFormatJSON, nil
	default:
		return "", fmt.Errorf(
			"%w: Workspace source format for %q is unsupported",
			storeSpec.ErrUnsupported,
			locator,
		)
	}
}
