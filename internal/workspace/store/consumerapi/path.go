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
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
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
		discovery.ExplicitLocators = appendUniqueLocator(
			discovery.ExplicitLocators,
			manifest,
		)
		discovery.DecoderHints = appendCanonicalDeclarationDecoderHint(
			discovery.DecoderHints,
			discovery.AllowedDecoderIDs,
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

	summary, _, err := a.sources.Ensure(
		ctx,
		request.RootID,
		source.Draft{
			ID:          source.SourceID(uuidutil.NewUUIDv7()),
			StorageKey:  workspacePathStorageKey(rootPath),
			Kind:        source.SourceKindFilesystemDirectory,
			DisplayName: displayName,
			Enabled:     true,
			Config:      config,
			Discovery:   discovery,
		},
	)
	if err != nil {
		return WorkspacePathRegistrationResult{}, err
	}

	if !summary.Enabled ||
		summary.DisplayName != displayName ||
		!summary.Discovery.Equal(discovery) {
		summary, err = a.sources.Update(
			ctx,
			request.RootID,
			summary.ID,
			source.Update{
				ExpectedRevision: summary.Revision,
				DisplayName:      displayName,
				Enabled:          true,
				Discovery:        &discovery,
			},
		)
		if err != nil {
			return WorkspacePathRegistrationResult{}, err
		}
	}

	if _, err := a.discovery.RefreshSource(
		ctx,
		request.RootID,
		summary.ID,
	); err != nil {
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
) (workspaceDomain.WorkspaceRef, error) {
	records, err := a.artifacts.ListBySource(ctx, rootID, sourceID)
	if err != nil {
		return workspaceDomain.WorkspaceRef{}, err
	}

	candidates := make([]artifact.Artifact, 0)
	for _, record := range records {
		if record.Kind != workspaceDomain.WorkspaceArtifactKind ||
			record.State != artifact.StateAvailable ||
			!record.Enabled {
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
		candidates = append(candidates, record)
	}

	switch len(candidates) {
	case 0:
		return workspaceDomain.WorkspaceRef{}, fmt.Errorf(
			"%w: no available Workspace declaration was discovered",
			basespec.ErrReferenceUnresolved,
		)
	case 1:
		return candidates[0].Ref(), nil
	default:
		return workspaceDomain.WorkspaceRef{}, fmt.Errorf(
			"%w: path contains %d Workspace declarations; provide a Workspace manifest file or workspaceName",
			basespec.ErrIdentityConflict,
			len(candidates),
		)
	}
}

func workspacePathStorageKey(
	rootPath string,
) basespec.StorageKey {
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(rootPath))),
		cryptoutil.DigestSHA256Prefix,
	)
	return basespec.StorageKey("workspace-path-" + digest[:24])
}
