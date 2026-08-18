package mcpbundle

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/mcp/overlay"
)

type RuntimeInvalidator interface {
	Invalidate(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) error

	InvalidateCollection(
		ctx context.Context,
		ref collection.CollectionRef,
	) error
}

func (a *API) invalidateReplacePlan(
	ctx context.Context,
	plan documentReplacePlan,
) error {
	refs := make(
		map[artifact.ArtifactRef]struct{},
		len(plan.existingBySubresource)+len(plan.registrations),
	)
	for _, value := range plan.existingBySubresource {
		refs[value.Ref()] = struct{}{}
	}
	for _, registration := range plan.registrations {
		refs[artifact.ArtifactRef{
			RootID:     plan.bundle.Collection.RootID,
			ArtifactID: registration.ArtifactID,
		}] = struct{}{}
	}

	ordered := make([]artifact.ArtifactRef, 0, len(refs))
	for ref := range refs {
		ordered = append(ordered, ref)
	}
	sort.Slice(ordered, func(left, right int) bool {
		if ordered[left].RootID != ordered[right].RootID {
			return ordered[left].RootID < ordered[right].RootID
		}
		return ordered[left].ArtifactID < ordered[right].ArtifactID
	})

	for _, ref := range ordered {
		if err := a.dependencies.Runtime.Invalidate(ctx, ref); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) UpdateBundleEnabled(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
	enabled bool,
) (Bundle, error) {
	if a == nil {
		return Bundle{}, basespec.ErrClosed
	}
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return Bundle{}, err
	}
	if expectedRevision == 0 {
		return Bundle{}, fmt.Errorf(
			"%w: expected MCP Bundle revision is required",
			basespec.ErrInvalid,
		)
	}

	current, err := a.Get(ctx, ref)
	if err != nil {
		return Bundle{}, err
	}
	if current.Collection.Revision != expectedRevision {
		return Bundle{}, basespec.ErrConflict
	}
	if current.Collection.Enabled == enabled {
		return current, nil
	}

	if err := a.dependencies.Runtime.InvalidateCollection(ctx, ref); err != nil {
		return Bundle{}, err
	}
	updated, err := a.dependencies.Collections.Update(
		ctx,
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
	return a.Get(ctx, updated.Ref())
}

func (a *API) Retire(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	if a == nil {
		return collection.Collection{}, basespec.ErrClosed
	}
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return collection.Collection{}, err
	}

	records, err := a.dependencies.Artifacts.ListByCollection(ctx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if len(records) != 0 {
		return collection.Collection{}, fmt.Errorf(
			"%w: remove all MCP server and policy Artifacts before retiring the Bundle",
			basespec.ErrConflict,
		)
	}

	if err := a.dependencies.Runtime.InvalidateCollection(ctx, ref); err != nil {
		return collection.Collection{}, err
	}
	return a.dependencies.Collections.Retire(
		ctx,
		ref,
		expectedRevision,
	)
}

func (a *API) Purge(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if err := a.requireBundleMutation(ctx, ref.RootID, false); err != nil {
		return err
	}

	retired, err := a.dependencies.Collections.GetRetired(ctx, ref)
	if err != nil {
		return err
	}
	if retired.Kind != artifactbuiltin.BundleKind {
		return fmt.Errorf(
			"%w: Collection %q is not a retired MCP Bundle",
			basespec.ErrCollectionNotFound,
			ref.CollectionID,
		)
	}
	data, err := DecodeCollectionData(retired.Data)
	if err != nil {
		return err
	}

	var owned source.Summary
	if data.ManagedSourceID != "" {
		owned, err = a.dependencies.Sources.Get(
			ctx,
			ref.RootID,
			data.ManagedSourceID,
		)
		if err != nil {
			return err
		}
	}

	if err := a.dependencies.Runtime.InvalidateCollection(ctx, ref); err != nil {
		return err
	}
	if err := a.dependencies.Collections.Purge(
		ctx,
		ref,
		expectedRevision,
	); err != nil {
		return err
	}
	if data.ManagedSourceID == "" {
		return nil
	}

	if err := a.dependencies.Sources.Discard(
		ctx,
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
}

func (a *API) UpdateProtectedBundleInstallation(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
) error {
	if a == nil {
		return basespec.ErrClosed
	}

	if !a.dependencies.RootPolicy.IsProtectedRoot(ref.RootID) {
		return fmt.Errorf(
			"%w: MCP Bundle is not in a protected Root",
			basespec.ErrProtected,
		)
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if a.dependencies.Overlays == nil {
		return fmt.Errorf(
			"%w: protected MCP Bundle overlay store is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}
	if _, err := a.Get(ctx, ref); err != nil {
		return err
	}

	current, found, err := a.dependencies.Overlays.GetBundleOverlay(
		ctx,
		ref.RootID,
		ref.CollectionID,
	)
	if err != nil {
		return err
	}
	if found && current.Revision != expectedOverlayRevision {
		return basespec.ErrConflict
	}
	if !found && expectedOverlayRevision != 0 {
		return basespec.ErrConflict
	}

	nextRevision := uint64(1)
	if found {
		nextRevision = current.Revision + 1
	}

	if err := a.dependencies.Runtime.InvalidateCollection(ctx, ref); err != nil {
		return err
	}

	return a.dependencies.Overlays.PutBundleOverlay(
		ctx,
		ref.RootID,
		ref.CollectionID,
		expectedOverlayRevision,
		overlay.BundleOverlay{
			SchemaVersion:  artifactbuiltin.MCPSchemaVersion,
			Revision:       nextRevision,
			RuntimeEnabled: runtimeEnabled,
		},
	)
}
