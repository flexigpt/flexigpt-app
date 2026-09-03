package aggregate

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpStore "github.com/flexigpt/flexigpt-app/internal/mcp/store"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
)

// BundleMutator is the narrow Store write port used by lifecycle coordination.
type BundleMutator interface {
	ReplaceDocument(
		ctx context.Context,
		request mcpStore.ReplaceDocumentRequest,
	) (mcpStore.Bundle, error)

	Refresh(
		ctx context.Context,
		ref collection.CollectionRef,
		allowProtected bool,
	) (mcpStore.Bundle, error)

	UpdateBundleEnabled(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedRevision uint64,
		enabled bool,
	) (mcpStore.Bundle, error)

	Retire(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedRevision uint64,
	) (collection.Collection, error)

	Purge(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedRevision uint64,
	) error

	UpdateProtectedBundleInstallation(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedOverlayRevision uint64,
		runtimeEnabled bool,
	) error

	UpdateServerInstallation(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedArtifactRevision uint64,
		data mcpStoreServer.ServerData,
	) (artifact.Artifact, error)

	UpdateProtectedServerInstallation(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedOverlayRevision uint64,
		runtimeEnabled bool,
		data mcpStoreServer.ServerData,
	) error
}

// RuntimeInvalidator is the only Runtime capability required by Store mutation
// coordination. Runtime does not know who persists a server.
type RuntimeInvalidator interface {
	Invalidate(
		ctx context.Context,
		server mcpServer.ServerID,
	) error

	InvalidateCollection(
		ctx context.Context,
		catalog mcpServer.CatalogID,
	) error
}

type Lifecycle struct {
	store   BundleMutator
	runtime RuntimeInvalidator
}

func NewLifecycle(
	store BundleMutator,
	runtime RuntimeInvalidator,
) (*Lifecycle, error) {
	if store == nil || runtime == nil {
		return nil, errors.New("MCP lifecycle dependencies are incomplete")
	}
	return &Lifecycle{
		store:   store,
		runtime: runtime,
	}, nil
}

func (l *Lifecycle) InvalidateServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if l == nil || l.runtime == nil {
		return mcpServer.ErrClosed
	}
	serverID, err := RuntimeServerIDForArtifact(ref)
	if err != nil {
		return err
	}
	return l.runtime.Invalidate(ctx, serverID)
}

func (l *Lifecycle) InvalidateCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	if l == nil || l.runtime == nil {
		return mcpServer.ErrClosed
	}
	catalogID, err := RuntimeCatalogIDForCollection(ref)
	if err != nil {
		return err
	}
	return l.runtime.InvalidateCollection(ctx, catalogID)
}

func (l *Lifecycle) ReplaceDocument(
	ctx context.Context,
	request mcpStore.ReplaceDocumentRequest,
) (mcpStore.Bundle, error) {
	if err := l.InvalidateCollection(ctx, request.Bundle); err != nil {
		return mcpStore.Bundle{}, err
	}
	return l.store.ReplaceDocument(ctx, request)
}

func (l *Lifecycle) RefreshBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	allowProtected bool,
) (mcpStore.Bundle, error) {
	if err := l.InvalidateCollection(ctx, ref); err != nil {
		return mcpStore.Bundle{}, err
	}
	return l.store.Refresh(ctx, ref, allowProtected)
}

func (l *Lifecycle) UpdateBundleEnabled(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
	enabled bool,
) (mcpStore.Bundle, error) {
	if err := l.InvalidateCollection(ctx, ref); err != nil {
		return mcpStore.Bundle{}, err
	}
	return l.store.UpdateBundleEnabled(ctx, ref, expectedRevision, enabled)
}

func (l *Lifecycle) RetireBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	if err := l.InvalidateCollection(ctx, ref); err != nil {
		return collection.Collection{}, err
	}
	return l.store.Retire(ctx, ref, expectedRevision)
}

func (l *Lifecycle) PurgeBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	if err := l.InvalidateCollection(ctx, ref); err != nil {
		return err
	}
	return l.store.Purge(ctx, ref, expectedRevision)
}

func (l *Lifecycle) UpdateProtectedBundleInstallation(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
) error {
	if err := l.InvalidateCollection(ctx, ref); err != nil {
		return err
	}
	return l.store.UpdateProtectedBundleInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		runtimeEnabled,
	)
}

func (l *Lifecycle) UpdateServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpStoreServer.ServerData,
) (artifact.Artifact, error) {
	if err := l.InvalidateServer(ctx, ref); err != nil {
		return artifact.Artifact{}, err
	}
	return l.store.UpdateServerInstallation(
		ctx,
		ref,
		expectedArtifactRevision,
		data,
	)
}

func (l *Lifecycle) UpdateProtectedServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
	data mcpStoreServer.ServerData,
) error {
	if err := l.InvalidateServer(ctx, ref); err != nil {
		return err
	}
	return l.store.UpdateProtectedServerInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		runtimeEnabled,
		data,
	)
}
