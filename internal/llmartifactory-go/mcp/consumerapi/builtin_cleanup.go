package consumerapi

import (
	"context"
	"errors"
	"fmt"
	"path"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain"
)

// BuiltinPackageCleanup is the narrow MCP lifecycle capability used by the
// generated built-in package hydrator.
//
// It owns MCP overlay and secret cleanup. It does not publish packages,
// refresh Sources, decode generated declarations, or mutate topology.
type BuiltinPackageCleanup interface {
	CaptureBuiltInPackageServers(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		addresses []managedpackageModel.ManagedPackageAddress,
	) ([]artifactModel.ArtifactRef, error)

	CleanupRemovedBuiltInPackageServers(
		ctx context.Context,
		refs []artifactModel.ArtifactRef,
	) error
}

type builtinPackageCleanup struct {
	api *API
}

func NewBuiltinPackageCleanup(
	api *API,
) (BuiltinPackageCleanup, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: MCP built-in package cleanup requires an API",
			spec.ErrInvalid,
		)
	}
	return &builtinPackageCleanup{api: api}, nil
}

func (c *builtinPackageCleanup) CaptureBuiltInPackageServers(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	addresses []managedpackageModel.ManagedPackageAddress,
) ([]artifactModel.ArtifactRef, error) {
	if c == nil || c.api == nil {
		return nil, spec.ErrClosed
	}
	if !topology.IsBuiltinPackageSource(rootID, sourceID) {
		return nil, fmt.Errorf(
			"%w: MCP cleanup does not target the built-in package Source",
			spec.ErrProtected,
		)
	}

	directories := make(map[spec.Locator]struct{}, len(addresses))
	for _, address := range addresses {
		if address.Kind != mcpDomain.MCPCollectionPackageKind {
			continue
		}
		directory, err := address.Directory()
		if err != nil {
			return nil, err
		}
		directories[directory] = struct{}{}
	}
	if len(directories) == 0 {
		return []artifactModel.ArtifactRef{}, nil
	}

	entries, err := c.api.cat.ListBySource(
		ctx,
		rootID,
		sourceID,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	refs := make([]artifactModel.ArtifactRef, 0)
	seen := make(map[artifactModel.ArtifactRef]struct{})
	for _, entry := range entries {
		if entry.State != artifactModel.StateAvailable ||
			entry.Kind != mcpDomain.MCPArtifactKind {
			continue
		}

		if _, found := directories[spec.Locator(path.Dir(string(entry.Binding.Locator)))]; !found {
			continue
		}
		if !topology.IsPluginDocumentFile(
			spec.Locator(path.Base(string(entry.Binding.Locator))),
		) {
			continue
		}
		if _, duplicate := seen[entry.Ref()]; duplicate {
			continue
		}
		seen[entry.Ref()] = struct{}{}
		refs = append(refs, entry.Ref())
	}
	return refs, nil
}

func (c *builtinPackageCleanup) CleanupRemovedBuiltInPackageServers(
	ctx context.Context,
	refs []artifactModel.ArtifactRef,
) error {
	if c == nil || c.api == nil {
		return spec.ErrClosed
	}

	for _, ref := range refs {
		record, err := c.api.artifacts.Get(ctx, ref)
		if err != nil &&
			!errors.Is(err, spec.ErrArtifactNotFound) &&
			!errors.Is(err, spec.ErrRootNotFound) {
			return err
		}
		if err == nil && record.State == artifactModel.StateAvailable {
			continue
		}
		if err := c.api.purgeBuiltInServerInstallation(ctx, ref); err != nil {
			return err
		}
	}
	return nil
}
