package consumerapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/defaultpolicy"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type workspaceSourceSet struct {
	Directory    source.Summary
	Policy       source.Summary
	HasDirectory bool
	HasPolicy    bool
}

type workspaceSourceRegistry struct {
	sources compositionapi.SourceAPI
}

func newWorkspaceSourceRegistry(
	sources compositionapi.SourceAPI,
) workspaceSourceRegistry {
	return workspaceSourceRegistry{sources: sources}
}

func (r workspaceSourceRegistry) load(
	ctx context.Context,
	rootID root.RootID,
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
					basespec.ErrIdentityConflict,
					rootID,
				)
			}
			output.Directory = value
			output.HasDirectory = true

		case WorkspaceBasePolicySourceStorageKey:
			if output.HasPolicy {
				return workspaceSourceSet{}, fmt.Errorf(
					"%w: Root %q has multiple Workspace policy Sources",
					basespec.ErrIdentityConflict,
					rootID,
				)
			}
			output.Policy = value
			output.HasPolicy = true
		}
	}

	if output.HasDirectory &&
		output.Directory.Kind != source.SourceKindFilesystemDirectory {
		return workspaceSourceSet{}, fmt.Errorf(
			"%w: Workspace directory Source has kind %q",
			basespec.ErrInvalid,
			output.Directory.Kind,
		)
	}
	if output.HasPolicy &&
		output.Policy.Kind != source.SourceKindEmbeddedDirectory {
		return workspaceSourceSet{}, fmt.Errorf(
			"%w: Workspace policy Source has kind %q",
			basespec.ErrInvalid,
			output.Policy.Kind,
		)
	}
	return output, nil
}

func (r workspaceSourceRegistry) required(
	ctx context.Context,
	rootID root.RootID,
) (workspaceSourceSet, error) {
	values, err := r.load(ctx, rootID)
	if err != nil {
		return workspaceSourceSet{}, err
	}
	if !values.HasDirectory || !values.HasPolicy {
		return workspaceSourceSet{}, fmt.Errorf(
			"%w: Workspace directory Root %q is missing a required Source",
			basespec.ErrReferenceUnresolved,
			rootID,
		)
	}
	return values, nil
}

func (r workspaceSourceRegistry) refreshSource(
	ctx context.Context,
	current source.Summary,
) (source.Summary, error) {
	if string(current.StorageKey) != WorkspaceBasePolicySourceStorageKey {
		return current, nil
	}
	values, err := r.required(ctx, current.RootID)
	if err != nil {
		return source.Summary{}, err
	}
	return values.Directory, nil
}

func (a *StoreAPI) findWorkspaceRoot(
	ctx context.Context,
	rootPath string,
) (root.Root, bool, error) {
	storageKey := workspaceRootStorageKey(rootPath)
	values, err := a.roots.List(ctx)
	if err != nil {
		return root.Root{}, false, err
	}

	var (
		found root.Root
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
		return root.Root{}, false, fmt.Errorf(
			"%w: multiple Roots use Workspace storage key %q",
			basespec.ErrIdentityConflict,
			storageKey,
		)
	}
	return found, count == 1, nil
}

func (a *StoreAPI) ensureWorkspaceRoot(
	ctx context.Context,
	rootPath string,
) (root.Root, bool, error) {
	existing, found, err := a.findWorkspaceRoot(ctx, rootPath)
	if err != nil || found {
		return existing, false, err
	}

	displayName := filepath.Base(rootPath)
	if err := basespec.ValidateRequiredText(
		"Workspace Root display name",
		displayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		displayName = "Workspace directory"
	}

	created, err := a.roots.Create(ctx, root.RootDraft{
		ID:          root.RootID(uuidutil.NewUUIDv7()),
		StorageKey:  workspaceRootStorageKey(rootPath),
		DisplayName: displayName,
		Description: "Workspace directory",
	})
	if err == nil {
		return created, true, nil
	}
	if !errors.Is(err, basespec.ErrConflict) {
		return root.Root{}, false, err
	}

	// Another registration may have created the deterministic Root.
	existing, found, findErr := a.findWorkspaceRoot(ctx, rootPath)
	if findErr != nil {
		return root.Root{}, false, findErr
	}
	if !found {
		return root.Root{}, false, err
	}
	return existing, false, nil
}

func (a *StoreAPI) defaultWorkspaceRecord(
	ctx context.Context,
	values workspaceSourceSet,
) (artifact.Artifact, bool, error) {
	records, err := a.artifacts.ListBySource(
		ctx,
		values.Policy.RootID,
		values.Policy.ID,
	)
	if err != nil {
		return artifact.Artifact{}, false, err
	}

	candidates := make([]artifact.Artifact, 0, 1)
	for _, record := range records {
		if record.Binding.Locator == basespec.Locator(defaultpolicy.PolicyLocator) &&
			record.Binding.SubresourceLocator == "" &&
			record.Kind == workspaceDomain.WorkspaceArtifactKind &&
			string(record.LogicalName) == defaultpolicy.PolicyID {
			candidates = append(candidates, record)
		}
	}
	switch len(candidates) {
	case 0:
		return artifact.Artifact{}, false, nil
	case 1:
		return candidates[0], true, nil
	default:
		return artifact.Artifact{}, false, fmt.Errorf(
			"%w: policy Source produced multiple default Workspace Artifacts",
			basespec.ErrIdentityConflict,
		)
	}
}

func (a *StoreAPI) refreshEffectiveWorkspaces(
	ctx context.Context,
	rootID root.RootID,
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
			if err := a.resolver.RefreshWorkspaceWithCompositionSource(
				ctx,
				record.Ref(),
				values.Directory.ID,
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
	if record.State != artifact.StateAvailable {
		return nil
	}
	return a.resolver.RefreshWorkspaceWithCompositionSource(
		ctx,
		record.Ref(),
		values.Directory.ID,
	)
}

func (a *StoreAPI) requireEffectiveWorkspace(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
) error {
	if workspace.Artifact.State != artifact.StateAvailable ||
		!workspace.Artifact.Enabled {
		return fmt.Errorf(
			"%w: selected Workspace is disabled or unavailable",
			basespec.ErrReferenceUnresolved,
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
			basespec.ErrSourceUnavailable,
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
				basespec.ErrReferenceUnresolved,
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
				basespec.ErrReferenceUnresolved,
			)
		}
		if string(workspace.Artifact.LogicalName) != defaultpolicy.PolicyID ||
			cryptoutil.DigestBytes(workspace.Definition.Body) != a.policy.Digest {
			return fmt.Errorf(
				"%w: selected default Workspace does not match the active policy",
				basespec.ErrDigestMismatch,
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
				basespec.ErrReferenceUnresolved,
			)
		}
		return nil

	default:
		return fmt.Errorf(
			"%w: Workspace does not belong to this directory or policy Source",
			basespec.ErrReferenceUnresolved,
		)
	}
}

func (a *StoreAPI) isEffectivePhysicalWorkspace(
	ctx context.Context,
	values workspaceSourceSet,
	ref artifact.ArtifactRef,
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

func (a *StoreAPI) workspaceDirectoryRootIDs(
	ctx context.Context,
) ([]root.RootID, error) {
	values, err := a.roots.List(ctx)
	if err != nil {
		return nil, err
	}
	output := make([]root.RootID, 0)
	for _, value := range values {
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
			basespec.ErrInvalid,
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
			basespec.ErrInvalid,
		)
	}
	return absolute, nil
}
