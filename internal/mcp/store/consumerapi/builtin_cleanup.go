package consumerapi

import (
	"context"
	"errors"
	"fmt"
	"path"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
)

// BuiltinPackageCleanup is the narrow MCP lifecycle capability used by the
// generated built-in package hydrator.
//
// It owns MCP overlay and secret cleanup. It does not publish packages,
// refresh Sources, decode generated declarations, or mutate topology.
type BuiltinPackageCleanup interface {
	CaptureBuiltInPackageServers(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		addresses []source.ManagedPackageAddress,
	) ([]artifact.ArtifactRef, error)

	CleanupRemovedBuiltInPackageServers(
		ctx context.Context,
		refs []artifact.ArtifactRef,
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
			basespec.ErrInvalid,
		)
	}
	return &builtinPackageCleanup{api: api}, nil
}

func (c *builtinPackageCleanup) CaptureBuiltInPackageServers(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	addresses []source.ManagedPackageAddress,
) ([]artifact.ArtifactRef, error) {
	if c == nil || c.api == nil {
		return nil, basespec.ErrClosed
	}
	if !documentTopology.IsBuiltinPackageSource(rootID, sourceID) {
		return nil, fmt.Errorf(
			"%w: MCP cleanup does not target the built-in package Source",
			basespec.ErrProtected,
		)
	}

	directories := make(map[basespec.Locator]struct{}, len(addresses))
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
		return []artifact.ArtifactRef{}, nil
	}

	entries, err := c.api.artifacts.ListBySource(
		ctx,
		rootID,
		sourceID,
		catalog.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	refs := make([]artifact.ArtifactRef, 0)
	seen := make(map[artifact.ArtifactRef]struct{})
	for _, entry := range entries {
		if entry.Kind != mcpDomain.MCPArtifactKind {
			continue
		}
		if _, found := directories[basespec.Locator(path.Dir(string(entry.Binding.Locator)))]; !found {
			continue
		}
		if !documentTopology.IsCollectionDocumentFile(
			basespec.Locator(path.Base(string(entry.Binding.Locator))),
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
	refs []artifact.ArtifactRef,
) error {
	if c == nil || c.api == nil {
		return basespec.ErrClosed
	}

	for _, ref := range refs {
		record, err := c.api.artifacts.Get(ctx, ref)
		if err != nil &&
			!errors.Is(err, basespec.ErrArtifactNotFound) &&
			!errors.Is(err, basespec.ErrRootNotFound) {
			return err
		}
		if err == nil && record.State == artifact.StateAvailable {
			continue
		}
		if err := c.api.purgeBuiltInServerInstallation(ctx, ref); err != nil {
			return err
		}
	}
	return nil
}
