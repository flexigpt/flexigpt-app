package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpbundlev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	mcpDomainBundle "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/bundle"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
)

func (a *API) UpdateBundleEnabled(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
	enabled bool,
) (Bundle, error) {
	commit, err := a.PrepareUpdateBundleEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
	if err != nil {
		return Bundle{}, err
	}
	return commit(ctx)
}

func (a *API) PrepareUpdateBundleEnabled(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
	enabled bool,
) (BundleMutationCommit, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return nil, err
	}
	if expectedRevision == 0 {
		return nil, fmt.Errorf(
			"%w: expected MCP Bundle revision is required",
			basespec.ErrInvalid,
		)
	}

	current, err := a.Get(ctx, ref)
	if err != nil {
		return nil, err
	}
	if current.Collection.Revision != expectedRevision {
		return nil, basespec.ErrConflict
	}
	if current.Collection.Enabled == enabled {
		return func(context.Context) (Bundle, error) {
			return current, nil
		}, nil
	}

	return func(commitCtx context.Context) (Bundle, error) {
		updated, err := a.collections.Update(
			commitCtx,
			ref,
			collection.Update{
				ExpectedRevision: expectedRevision,
				DisplayName:      current.Collection.DisplayName,
				Description:      current.Collection.Description,
				Enabled:          enabled,
				Data:             current.Collection.Data,
			},
		)
		if err != nil {
			return Bundle{}, err
		}
		return a.Get(commitCtx, updated.Ref())
	}, nil
}

func (a *API) Retire(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	commit, err := a.PrepareRetire(ctx, ref, expectedRevision)
	if err != nil {
		return collection.Collection{}, err
	}
	return commit(ctx)
}

func (a *API) PrepareRetire(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (CollectionMutationCommit, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return nil, err
	}
	if expectedRevision == 0 {
		return nil, fmt.Errorf(
			"%w: expected MCP Bundle revision is required",
			basespec.ErrInvalid,
		)
	}

	bundle, err := a.Get(ctx, ref)
	if err != nil {
		return nil, err
	}
	if bundle.Collection.Revision != expectedRevision {
		return nil, basespec.ErrConflict
	}

	records, err := a.artifacts.ListByCollection(ctx, ref)
	if err != nil {
		return nil, err
	}
	if len(records) != 0 {
		return nil, fmt.Errorf(
			"%w: remove all MCP server and policy Artifacts before retiring the Bundle",
			basespec.ErrConflict,
		)
	}
	return func(commitCtx context.Context) (collection.Collection, error) {
		return a.collections.Retire(commitCtx, ref, expectedRevision)
	}, nil
}

func (a *API) Purge(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	commit, err := a.PreparePurge(ctx, ref, expectedRevision)
	if err != nil {
		return err
	}
	return commit(ctx)
}

func (a *API) PreparePurge(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (MutationCommit, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return nil, err
	}
	if expectedRevision == 0 {
		return nil, fmt.Errorf(
			"%w: expected MCP Bundle revision is required",
			basespec.ErrInvalid,
		)
	}

	retired, err := a.collections.GetRetired(ctx, ref)
	if err != nil {
		return nil, err
	}
	if retired.Kind != mcpbundlev1.MCPBundleKind {
		return nil, fmt.Errorf(
			"%w: Collection %q is not a retired MCP Bundle",
			basespec.ErrCollectionNotFound,
			ref.CollectionID,
		)
	}
	if retired.Revision != expectedRevision {
		return nil, basespec.ErrConflict
	}
	data, err := mcpDomainBundle.DecodeCollectionData(retired.Data)
	if err != nil {
		return nil, err
	}

	var owned source.Summary
	if data.ManagedSourceID != "" {
		owned, err = a.sources.Get(
			ctx,
			ref.RootID,
			data.ManagedSourceID,
		)
		if err != nil {
			return nil, err
		}
	}

	return func(commitCtx context.Context) error {
		if err := a.collections.Purge(
			commitCtx,
			ref,
			expectedRevision,
		); err != nil {
			return err
		}
		if data.ManagedSourceID == "" {
			return nil
		}
		if err := a.sources.Discard(
			commitCtx,
			ref.RootID,
			owned.ID,
			owned.Revision,
		); err != nil {
			return fmt.Errorf(
				"MCP Bundle metadata was purged but managed Source cleanup remains pending: %w",
				err,
			)
		}
		return nil
	}, nil
}

func (a *API) UpdateProtectedBundleInstallation(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
) error {
	commit, err := a.PrepareUpdateProtectedBundleInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		runtimeEnabled,
	)
	if err != nil {
		return err
	}
	return commit(ctx)
}

func (a *API) PrepareUpdateProtectedBundleInstallation(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
) (MutationCommit, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}

	if !a.protection.IsProtectedRoot(ref.RootID) {
		return nil, fmt.Errorf(
			"%w: MCP Bundle is not in a protected Root",
			basespec.ErrProtected,
		)
	}
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	if a.overlays == nil {
		return nil, fmt.Errorf(
			"%w: protected MCP Bundle overlay store is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}
	if _, err := a.Get(ctx, ref); err != nil {
		return nil, err
	}

	current, found, err := a.overlays.GetBundleOverlay(
		ctx,
		ref.RootID,
		ref.CollectionID,
	)
	if err != nil {
		return nil, err
	}
	if found && current.Revision != expectedOverlayRevision {
		return nil, basespec.ErrConflict
	}
	if !found && expectedOverlayRevision != 0 {
		return nil, basespec.ErrConflict
	}

	nextRevision := uint64(1)
	if found {
		nextRevision = current.Revision + 1
	}

	next := mcpOverlay.BundleOverlay{
		SchemaVersion:  artifactbuiltin.MCPSchemaVersion,
		Revision:       nextRevision,
		RuntimeEnabled: runtimeEnabled,
	}
	return func(commitCtx context.Context) error {
		return a.overlays.PutBundleOverlay(
			commitCtx,
			ref.RootID,
			ref.CollectionID,
			expectedOverlayRevision,
			next,
		)
	}, nil
}
