package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/consumerutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

// AddWorkspacePath registers a native directory or Workspace declaration
// file beneath an existing Root, discovers the Workspace and its initial
// Artifacts, applies Workspace declaration sources, and resolves all roots.
//
// This is intentionally native-path input owned by the Workspace domain. It
// is not a portable declaration locator.
func (a *StoreAPI) AddWorkspacePath(
	ctx context.Context,
	request WorkspacePathRegistration,
) (WorkspacePathRegistrationResult, error) {
	if a == nil {
		return WorkspacePathRegistrationResult{}, basespec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return WorkspacePathRegistrationResult{}, err
	}
	if request.WorkspaceName != "" {
		if err := request.WorkspaceName.Validate(); err != nil {
			return WorkspacePathRegistrationResult{}, err
		}
	}

	rootPath, manifest, hasManifest, err := normalizeWorkspacePath(
		request.Path,
	)
	if err != nil {
		return WorkspacePathRegistrationResult{}, err
	}
	displayName := request.SourceDisplayName
	if displayName == "" {
		displayName = "Workspace source " + filepath.Base(rootPath)
		if filepath.Base(rootPath) == string(filepath.Separator) {
			displayName = "Workspace source"
		}
	}
	if err := basespec.ValidateRequiredText(
		"Workspace Source display name",
		displayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return WorkspacePathRegistrationResult{}, err
	}

	discovery, err := a.defaultDiscovery()
	if err != nil {
		return WorkspacePathRegistrationResult{}, err
	}
	if hasManifest {
		discovery.ExplicitLocators = consumerutil.AppendUniqueLocator(
			discovery.ExplicitLocators,
			manifest,
		)
		discovery = discovery.Normalized()
	}

	config, err := json.Marshal(struct {
		RootPath string `json:"rootPath"`
	}{
		RootPath: rootPath,
	})
	if err != nil {
		return WorkspacePathRegistrationResult{}, err
	}

	summary, err := consumerutil.EnsureAndRefreshSource(
		ctx,
		a.sources,
		a.discovery,
		consumerutil.EnsureAndRefreshSourceRequest{
			RootID: request.RootID,
			Draft: source.Draft{
				ID: source.SourceID(uuidutil.NewUUIDv7()),
				StorageKey: consumerutil.FilesystemSourceStorageKey(
					"workspace-path",
					rootPath,
				),
				Kind:        source.SourceKindFilesystemDirectory,
				DisplayName: displayName,
				Enabled:     true,
				Config:      config,
				Discovery:   discovery,
			},
			ReconcileDiscovery: consumerutil.MergeDiscoveryScopes,
		},
	)
	if err != nil {
		return WorkspacePathRegistrationResult{}, err
	}

	ref, err := a.findWorkspaceFromPath(
		ctx,
		request.RootID,
		summary.ID,
		manifest,
		hasManifest,
		request.WorkspaceName,
	)
	if err != nil {
		return WorkspacePathRegistrationResult{}, err
	}
	if _, err := a.RefreshWorkspace(ctx, ref); err != nil {
		return WorkspacePathRegistrationResult{}, err
	}

	// Loading is read-only. RefreshWorkspace above expands selector and local
	// locator closure through the typed Artifact resolver.
	load, err := a.LoadWorkspace(ctx, ref)
	if err != nil {
		return WorkspacePathRegistrationResult{}, err
	}
	summary, err = a.sources.Get(ctx, request.RootID, summary.ID)
	if err != nil {
		return WorkspacePathRegistrationResult{}, err
	}
	return WorkspacePathRegistrationResult{
		Source:    summary,
		Workspace: load.Workspace,
		Load:      load,
	}, nil
}

func normalizeWorkspacePath(
	raw string,
) (
	rootPath string,
	manifest basespec.Locator,
	hasManifest bool,
	err error,
) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", "", false, fmt.Errorf(
			"%w: Workspace path is required",
			basespec.ErrInvalid,
		)
	}
	absolute, err := filepath.Abs(raw)
	if err != nil {
		return "", "", false, err
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", "", false, err
	}
	if info.IsDir() {
		return absolute, "", false, nil
	}
	if !info.Mode().IsRegular() {
		return "", "", false, fmt.Errorf(
			"%w: Workspace path is not a regular file or directory",
			basespec.ErrInvalid,
		)
	}

	rootPath = filepath.Dir(absolute)
	relative, err := filepath.Rel(rootPath, absolute)
	if err != nil {
		return "", "", false, err
	}
	manifest = basespec.Locator(filepath.ToSlash(relative))
	if err := manifest.Validate(false); err != nil {
		return "", "", false, err
	}
	return rootPath, manifest, true, nil
}

func (a *StoreAPI) findWorkspaceFromPath(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	manifest basespec.Locator,
	hasManifest bool,
	name basespec.LogicalName,
) (artifact.ArtifactRef, error) {
	records, err := a.artifacts.ListBySource(ctx, rootID, sourceID)
	if err != nil {
		return artifact.ArtifactRef{}, err
	}

	candidates := make(
		map[artifact.ArtifactRef]struct{},
	)
	for _, record := range records {
		if record.Kind != workspaceDomain.WorkspaceArtifactKind ||
			record.State != artifact.StateAvailable {
			continue
		}
		if hasManifest &&
			(record.Binding.Locator != manifest ||
				record.Binding.SubresourceLocator != "") {
			continue
		}
		if name != "" && record.LogicalName != name {
			continue
		}
		workspace, err := a.GetWorkspace(ctx, record.Ref())
		if err != nil {
			return artifact.ArtifactRef{}, err
		}
		candidates[workspace.Ref()] = struct{}{}
	}

	switch len(candidates) {
	case 0:
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: no available Workspace declaration was discovered",
			basespec.ErrReferenceUnresolved,
		)
	case 1:
		for ref := range candidates {
			return ref, nil
		}
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: Workspace candidate set is inconsistent",
			basespec.ErrInvalid,
		)
	default:
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: path contains %d Workspace declarations; provide a Workspace manifest file or workspaceName",
			basespec.ErrIdentityConflict,
			len(candidates),
		)
	}
}
