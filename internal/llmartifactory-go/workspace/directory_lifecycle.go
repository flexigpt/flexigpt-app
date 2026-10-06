package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type workspaceSourceSet struct {
	Directory    sourceModel.Summary
	Policy       sourceModel.Summary
	HasDirectory bool
	HasPolicy    bool
}

type workspaceSourceRegistry struct {
	sources source.API
}

func newWorkspaceSourceRegistry(
	sources source.API,
) workspaceSourceRegistry {
	return workspaceSourceRegistry{sources: sources}
}

func (r workspaceSourceRegistry) load(
	ctx context.Context,
	rootID rootModel.RootID,
) (workspaceSourceSet, error) {
	values, err := r.sources.List(ctx, rootID)
	if err != nil {
		return workspaceSourceSet{}, err
	}

	var output workspaceSourceSet
	for _, value := range values {
		switch string(value.StorageKey) {
		case WorkspaceDirectorySourceStorageKey:
			if output.HasDirectory {
				return workspaceSourceSet{}, fmt.Errorf(
					"%w: Root %q has multiple Workspace directory Sources",
					spec.ErrIdentityConflict,
					rootID,
				)
			}
			output.Directory = value
			output.HasDirectory = true

		case WorkspaceBasePolicySourceStorageKey:
			if output.HasPolicy {
				return workspaceSourceSet{}, fmt.Errorf(
					"%w: Root %q has multiple Workspace policy Sources",
					spec.ErrIdentityConflict,
					rootID,
				)
			}
			output.Policy = value
			output.HasPolicy = true
		}
	}

	if output.HasDirectory &&
		output.Directory.Kind != fsdir.Kind {
		return workspaceSourceSet{}, fmt.Errorf(
			"%w: Workspace directory Source has kind %q",
			spec.ErrInvalid,
			output.Directory.Kind,
		)
	}
	if output.HasPolicy &&
		output.Policy.Kind != iofs.Kind {
		return workspaceSourceSet{}, fmt.Errorf(
			"%w: Workspace policy Source has kind %q",
			spec.ErrInvalid,
			output.Policy.Kind,
		)
	}
	return output, nil
}

func (r workspaceSourceRegistry) required(
	ctx context.Context,
	rootID rootModel.RootID,
) (workspaceSourceSet, error) {
	values, err := r.load(ctx, rootID)
	if err != nil {
		return workspaceSourceSet{}, err
	}
	if !values.HasDirectory || !values.HasPolicy {
		return workspaceSourceSet{}, fmt.Errorf(
			"%w: Workspace directory Root %q is missing a required Source",
			spec.ErrReferenceUnresolved,
			rootID,
		)
	}
	return values, nil
}

func (r workspaceSourceRegistry) refreshSource(
	ctx context.Context,
	current sourceModel.Summary,
) (sourceModel.Summary, error) {
	if string(current.StorageKey) != WorkspaceBasePolicySourceStorageKey {
		return current, nil
	}
	values, err := r.required(ctx, current.RootID)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	return values.Directory, nil
}

func (a *Service) findWorkspaceRoot(
	ctx context.Context,
	rootPath string,
) (rootModel.Root, bool, error) {
	storageKey := workspaceRootStorageKey(rootPath)
	values, err := a.roots.List(ctx)
	if err != nil {
		return rootModel.Root{}, false, err
	}

	var (
		found rootModel.Root
		count int
	)
	for _, value := range values {
		if value.StorageKey != storageKey {
			continue
		}
		found = value
		count++
	}
	if count > 1 {
		return rootModel.Root{}, false, fmt.Errorf(
			"%w: multiple Roots use Workspace storage key %q",
			spec.ErrIdentityConflict,
			storageKey,
		)
	}
	return found, count == 1, nil
}

func (a *Service) ensureWorkspaceRoot(
	ctx context.Context,
	rootPath string,
) (rootModel.Root, bool, error) {
	existing, found, err := a.findWorkspaceRoot(ctx, rootPath)
	if err != nil || found {
		return existing, false, err
	}

	displayName := filepath.Base(rootPath)
	if err := spec.ValidateRequiredText(
		"Workspace Root display name",
		displayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		displayName = "Workspace directory"
	}

	created, err := a.roots.Create(ctx, rootModel.RootDraft{
		ID:          rootModel.RootID(uuidutil.NewUUIDv7()),
		StorageKey:  workspaceRootStorageKey(rootPath),
		DisplayName: displayName,
		Description: "Workspace directory",
	})
	if err == nil {
		return created, true, nil
	}
	if !errors.Is(err, spec.ErrConflict) {
		return rootModel.Root{}, false, err
	}

	// Another registration may have created the deterministic Root.
	existing, found, findErr := a.findWorkspaceRoot(ctx, rootPath)
	if findErr != nil {
		return rootModel.Root{}, false, findErr
	}
	if !found {
		return rootModel.Root{}, false, err
	}
	return existing, false, nil
}

func (a *Service) defaultWorkspaceRecord(
	ctx context.Context,
	values workspaceSourceSet,
) (artifactModel.Artifact, bool, error) {
	entries, err := a.cat.ListBySource(
		ctx,
		values.Policy.RootID,
		values.Policy.ID,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return artifactModel.Artifact{}, false, err
	}

	refs := make([]artifactModel.ArtifactRef, 0, 1)
	for _, entry := range entries {
		if entry.Binding.Locator == a.policySource.Locator &&
			entry.Binding.SubresourceLocator == "" &&
			entry.Kind == workspaceDomain.WorkspaceArtifactKind &&
			string(entry.LogicalName) == a.policy.ID {
			refs = append(refs, entry.Ref())
		}
	}
	switch len(refs) {
	case 0:
		return artifactModel.Artifact{}, false, nil
	case 1:
		values, err := a.artifacts.GetMany(ctx, refs)
		if err != nil {
			return artifactModel.Artifact{}, false, err
		}
		return values[0], true, nil
	default:
		return artifactModel.Artifact{}, false, fmt.Errorf(
			"%w: policy Source produced multiple default Workspace Artifacts",
			spec.ErrIdentityConflict,
		)
	}
}

func (a *Service) refreshEffectiveWorkspaces(
	ctx context.Context,
	rootID rootModel.RootID,
) error {
	values, err := a.workspaceSources.required(ctx, rootID)
	if err != nil {
		return err
	}
	if !values.Directory.Enabled || !values.Policy.Enabled {
		return nil
	}

	intent, err := a.listManifestIntent(ctx, rootID, values.Directory.ID)
	if err != nil {
		return err
	}
	physical, err := a.listPhysicalWorkspaces(
		ctx,
		rootID,
		values.Directory.ID,
	)
	if err != nil {
		return err
	}

	if len(physical) != 0 {
		for _, record := range physical {
			if err := a.refresher.Refresh(
				ctx,
				record.Ref(),
			); err != nil {
				return err
			}
		}
		return nil
	}
	if len(intent) != 0 {
		return nil
	}

	record, found, err := a.defaultWorkspaceRecord(ctx, values)
	if err != nil || !found {
		return err
	}
	if record.State != artifactModel.StateAvailable {
		return nil
	}
	return a.refresher.Refresh(
		ctx,
		record.Ref(),
	)
}

func (a *Service) requireEffectiveWorkspace(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
) error {
	if workspace.Artifact.State != artifactModel.StateAvailable ||
		!workspace.Artifact.Enabled {
		return fmt.Errorf(
			"%w: selected Workspace is disabled or unavailable",
			spec.ErrReferenceUnresolved,
		)
	}

	values, err := a.workspaceSources.required(
		ctx,
		workspace.Artifact.RootID,
	)
	if err != nil {
		return err
	}
	if !values.Directory.Enabled || !values.Policy.Enabled {
		return fmt.Errorf(
			"%w: Workspace directory is disabled",
			spec.ErrSourceUnavailable,
		)
	}

	switch workspace.Artifact.Binding.SourceID {
	case values.Directory.ID:
		effective, err := a.isEffectivePhysicalWorkspace(
			ctx,
			values,
			workspace.Ref(),
		)
		if err != nil {
			return err
		}
		if !effective {
			return fmt.Errorf(
				"%w: Workspace is not an effective physical manifest Workspace",
				spec.ErrReferenceUnresolved,
			)
		}
		return nil

	case values.Policy.ID:
		record, found, err := a.defaultWorkspaceRecord(ctx, values)
		if err != nil {
			return err
		}
		if !found || record.Ref() != workspace.Ref() {
			return fmt.Errorf(
				"%w: selected Workspace is not the Root-local default policy Workspace",
				spec.ErrReferenceUnresolved,
			)
		}
		if string(workspace.Artifact.LogicalName) != a.policy.ID ||
			cryptoutil.DigestBytes(workspace.Definition.Body) != a.policy.Digest {
			return fmt.Errorf(
				"%w: selected default Workspace does not match the active policy",
				spec.ErrDigestMismatch,
			)
		}
		intent, err := a.listManifestIntent(
			ctx,
			workspace.Artifact.RootID,
			values.Directory.ID,
		)
		if err != nil {
			return err
		}
		physical, err := a.listPhysicalWorkspaces(
			ctx,
			workspace.Artifact.RootID,
			values.Directory.ID,
		)
		if err != nil {
			return err
		}
		if len(intent) != 0 || len(physical) != 0 {
			return fmt.Errorf(
				"%w: default Workspace is inactive because a physical Workspace manifest is present",
				spec.ErrReferenceUnresolved,
			)
		}
		return nil

	default:
		return fmt.Errorf(
			"%w: Workspace does not belong to this directory or policy Source",
			spec.ErrReferenceUnresolved,
		)
	}
}

func (a *Service) isEffectivePhysicalWorkspace(
	ctx context.Context,
	values workspaceSourceSet,
	ref artifactModel.ArtifactRef,
) (bool, error) {
	records, err := a.listPhysicalWorkspaces(
		ctx,
		ref.RootID,
		values.Directory.ID,
	)
	if err != nil {
		return false, err
	}
	for _, record := range records {
		if !record.Enabled {
			continue
		}
		if record.Ref() == ref {
			return true, nil
		}
		terminal, err := a.resolver.ResolveTerminalArtifact(
			ctx,
			record.Ref(),
		)
		if err != nil {
			if contextErr := ctx.Err(); contextErr != nil {
				return false, contextErr
			}
			continue
		}
		if terminal == ref {
			return true, nil
		}
	}
	return false, nil
}

func (a *Service) workspaceDirectoryRootIDs(
	ctx context.Context,
) ([]rootModel.RootID, error) {
	values, err := a.roots.List(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]rootModel.RootID, 0)
	for _, value := range values {
		if !strings.HasPrefix(
			string(value.StorageKey),
			WorkspaceRootStorageKeyPrefix,
		) {
			continue
		}
		sources, err := a.workspaceSources.load(ctx, value.ID)
		if err != nil {
			return nil, err
		}

		if sources.HasDirectory && sources.HasPolicy {
			output = append(output, value.ID)
		}
	}
	slices.Sort(output)
	return output, nil
}

func normalizeWorkspaceDirectoryPath(raw string) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf(
			"%w: Workspace directory path is required",
			spec.ErrInvalid,
		)
	}
	absolute, err := filepath.Abs(raw)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf(
			"%w: Workspace path is not a directory",
			spec.ErrInvalid,
		)
	}
	return absolute, nil
}
