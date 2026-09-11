package consumerapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/workspacecollectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/attachmentdata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/collectiondata"
)

func (a *StoreAPI) CreateFilesystemWorkspace(
	ctx context.Context,
	input CreateFilesystemWorkspaceInput,
) (WorkspaceView, error) {
	if err := a.managementReady(ctx); err != nil {
		return WorkspaceView{}, err
	}

	data, err := workspaceCollectionData(input.Discovery)
	if err != nil {
		return WorkspaceView{}, err
	}
	attachmentData, err := workspaceAttachmentData(
		workspaceDomain.RolePrimary,
		WorkspaceAttachmentSettings{},
	)
	if err != nil {
		return WorkspaceView{}, err
	}

	rootID := a.workspace.workspaceRootID
	sourceValue, sourceCreated, err := a.sources.CreateWithStatus(
		ctx,
		rootID,
		source.Draft{
			ID:          source.SourceID(uuidutil.NewUUIDv7()),
			StorageKey:  workspaceStorageKey(),
			Kind:        source.SourceKindFilesystemDirectory,
			DisplayName: input.DisplayName,
			Enabled:     true,
			Config:      filesystemSourceConfig(input.RootPath),
		},
	)
	if err != nil {
		return WorkspaceView{}, err
	}

	created, _, err := a.collections.Create(
		ctx,
		rootID,
		collection.Draft{
			ID:          collection.CollectionID(uuidutil.NewUUIDv7()),
			Kind:        workspacecollectionv1.WorkspaceCollectionKind,
			DisplayName: input.DisplayName,
			Description: input.Description,
			Enabled:     true,
			Data:        data,
		},
		[]collection.AttachmentDraft{{
			SourceID: sourceValue.ID,
			Role:     workspaceDomain.RolePrimary,
			Enabled:  true,
			Data:     attachmentData,
		}},
	)
	if err != nil {
		if !sourceCreated {
			return WorkspaceView{}, err
		}
		cleanupErr := a.sources.Discard(
			context.WithoutCancel(ctx),
			rootID,
			sourceValue.ID,
			sourceValue.Revision,
		)
		return WorkspaceView{}, errors.Join(err, cleanupErr)
	}

	return a.workspaceViewForManagement(ctx, created.Ref())
}

func (a *StoreAPI) CreateEmptyWorkspace(
	ctx context.Context,
	input CreateEmptyWorkspaceInput,
) (WorkspaceView, error) {
	if err := a.managementReady(ctx); err != nil {
		return WorkspaceView{}, err
	}

	data, err := workspaceCollectionData(input.Discovery)
	if err != nil {
		return WorkspaceView{}, err
	}
	created, _, err := a.collections.Create(
		ctx,
		a.workspace.workspaceRootID,
		collection.Draft{
			ID:          collection.CollectionID(uuidutil.NewUUIDv7()),
			Kind:        workspacecollectionv1.WorkspaceCollectionKind,
			DisplayName: input.DisplayName,
			Description: input.Description,
			Enabled:     true,
			Data:        data,
		},
		nil,
	)
	if err != nil {
		return WorkspaceView{}, err
	}
	return a.workspaceViewForManagement(ctx, created.Ref())
}

func (a *StoreAPI) UpdateWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
	input UpdateWorkspaceInput,
) (WorkspaceView, error) {
	current, err := a.workspaceForManagement(ctx, ref)
	if err != nil {
		return WorkspaceView{}, err
	}
	if input.ExpectedRevision == 0 || current.Collection.Revision != input.ExpectedRevision {
		return WorkspaceView{}, basespec.ErrConflict
	}

	data, err := workspaceCollectionData(input.Discovery)
	if err != nil {
		return WorkspaceView{}, err
	}
	updated, err := a.collections.Update(
		ctx,
		ref,
		collection.Update{
			ExpectedRevision: input.ExpectedRevision,
			DisplayName:      input.DisplayName,
			Description:      input.Description,
			Enabled:          input.Enabled,
			Data:             data,
		},
	)
	if err != nil {
		return WorkspaceView{}, err
	}
	return a.workspaceViewForManagement(ctx, updated.Ref())
}

func (a *StoreAPI) SetWorkspacePrimarySource(
	ctx context.Context,
	ref WorkspaceRef,
	input SetWorkspacePrimarySourceInput,
) (WorkspaceView, error) {
	current, err := a.workspaceForManagement(ctx, ref)
	if err != nil {
		return WorkspaceView{}, err
	}
	if input.ExpectedCollectionRevision == 0 ||
		current.Collection.Revision != input.ExpectedCollectionRevision {
		return WorkspaceView{}, basespec.ErrConflict
	}

	var previous *collection.Attachment
	for index := range current.Attachments {
		if current.Attachments[index].Role == workspaceDomain.RolePrimary {
			value := current.Attachments[index]
			previous = &value
			break
		}
	}

	if input.Clear {
		if input.SourceID != "" {
			return WorkspaceView{}, fmt.Errorf(
				"%w: clearing a Workspace primary Source cannot include sourceID",
				basespec.ErrInvalid,
			)
		}
		if previous == nil {
			return a.workspaceViewForManagement(ctx, ref)
		}
		if _, err := a.collections.Detach(
			ctx,
			ref,
			previous.SourceID,
			input.ExpectedCollectionRevision,
			previous.Revision,
		); err != nil {
			return WorkspaceView{}, err
		}
		return a.workspaceViewForManagement(ctx, ref)
	}

	if input.SourceID == "" {
		return WorkspaceView{}, fmt.Errorf(
			"%w: Workspace primary Source is required",
			basespec.ErrInvalid,
		)
	}
	if previous != nil && previous.SourceID == input.SourceID {
		return a.workspaceViewForManagement(ctx, ref)
	}

	sourceValue, err := a.sources.Get(ctx, ref.RootID, input.SourceID)
	if err != nil {
		return WorkspaceView{}, err
	}
	if !sourceValue.Enabled ||
		sourceValue.Kind != source.SourceKindFilesystemDirectory {
		return WorkspaceView{}, fmt.Errorf(
			"%w: Workspace primary Source must be an enabled filesystem Source",
			basespec.ErrInvalid,
		)
	}

	attachmentData, err := workspaceAttachmentData(
		workspaceDomain.RolePrimary,
		WorkspaceAttachmentSettings{},
	)
	if err != nil {
		return WorkspaceView{}, err
	}
	replacement := collection.AttachmentDraft{
		SourceID: input.SourceID,
		Role:     workspaceDomain.RolePrimary,
		Enabled:  true,
		Data:     attachmentData,
	}

	if previous == nil {
		if _, _, err := a.collections.Attach(
			ctx,
			ref,
			input.ExpectedCollectionRevision,
			replacement,
		); err != nil {
			return WorkspaceView{}, err
		}
	} else {
		if _, _, err := a.collections.ReplaceAttachment(
			ctx,
			ref,
			collection.AttachmentReplacement{
				ExpectedCollectionRevision: input.ExpectedCollectionRevision,
				PreviousSourceID:           previous.SourceID,
				PreviousAttachmentRevision: previous.Revision,
				Replacement:                replacement,
			},
		); err != nil {
			return WorkspaceView{}, err
		}
	}

	return a.workspaceViewForManagement(ctx, ref)
}

func (a *StoreAPI) AttachWorkspaceSource(
	ctx context.Context,
	ref WorkspaceRef,
	input AttachWorkspaceSourceInput,
) (WorkspaceView, error) {
	if _, err := a.workspaceForManagement(ctx, ref); err != nil {
		return WorkspaceView{}, err
	}
	if err := requireAttachableWorkspaceRole(input.Role); err != nil {
		return WorkspaceView{}, err
	}
	attachmentData, err := workspaceAttachmentData(input.Role, input.Settings)
	if err != nil {
		return WorkspaceView{}, err
	}
	if _, _, err := a.collections.Attach(
		ctx,
		ref,
		input.ExpectedCollectionRevision,
		collection.AttachmentDraft{
			SourceID: input.SourceID,
			Role:     input.Role,
			Enabled:  input.Enabled,
			Data:     attachmentData,
		},
	); err != nil {
		return WorkspaceView{}, err
	}
	return a.workspaceViewForManagement(ctx, ref)
}

func (a *StoreAPI) UpdateWorkspaceAttachment(
	ctx context.Context,
	ref WorkspaceRef,
	input UpdateWorkspaceAttachmentInput,
) (WorkspaceView, error) {
	if _, err := a.workspaceForManagement(ctx, ref); err != nil {
		return WorkspaceView{}, err
	}
	if err := requireAttachableWorkspaceRole(input.Role); err != nil {
		return WorkspaceView{}, err
	}
	attachmentData, err := workspaceAttachmentData(input.Role, input.Settings)
	if err != nil {
		return WorkspaceView{}, err
	}
	if _, _, err := a.collections.UpdateAttachment(
		ctx,
		ref,
		input.SourceID,
		collection.AttachmentUpdate{
			ExpectedCollectionRevision: input.ExpectedCollectionRevision,
			ExpectedAttachmentRevision: input.ExpectedAttachmentRevision,
			Role:                       input.Role,
			Enabled:                    input.Enabled,
			Data:                       attachmentData,
		},
	); err != nil {
		return WorkspaceView{}, err
	}
	return a.workspaceViewForManagement(ctx, ref)
}

func (a *StoreAPI) DetachWorkspaceSource(
	ctx context.Context,
	ref WorkspaceRef,
	input DetachWorkspaceSourceInput,
) (WorkspaceView, error) {
	current, err := a.workspaceForManagement(ctx, ref)
	if err != nil {
		return WorkspaceView{}, err
	}
	for _, attachment := range current.Attachments {
		if attachment.SourceID == input.SourceID &&
			attachment.Role == workspaceDomain.RolePrimary {
			return WorkspaceView{}, fmt.Errorf(
				"%w: clear the primary Source through SetWorkspacePrimarySource",
				basespec.ErrInvalid,
			)
		}
	}
	if _, err := a.collections.Detach(
		ctx,
		ref,
		input.SourceID,
		input.ExpectedCollectionRevision,
		input.ExpectedAttachmentRevision,
	); err != nil {
		return WorkspaceView{}, err
	}
	return a.workspaceViewForManagement(ctx, ref)
}

func (a *StoreAPI) RefreshWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
) (WorkspaceRefreshResult, error) {
	if _, err := a.workspaceForManagement(ctx, ref); err != nil {
		return WorkspaceRefreshResult{}, err
	}

	result, err := a.catalogs.RefreshCollection(ctx, ref)
	if err != nil {
		return WorkspaceRefreshResult{}, err
	}

	output := WorkspaceRefreshResult{
		Workspace:        ref,
		CatalogRevision:  result.Catalog.Revision,
		Diagnostics:      diagnostic.Clone(result.Diagnostics),
		Candidates:       result.Candidates,
		CreatedArtifacts: make([]artifact.ArtifactRef, 0, len(result.CreatedArtifacts)),
		UpdatedArtifacts: make([]artifact.ArtifactRef, 0, len(result.UpdatedArtifacts)),
	}
	for _, id := range result.CreatedArtifacts {
		output.CreatedArtifacts = append(output.CreatedArtifacts, artifact.ArtifactRef{
			RootID:     ref.RootID,
			ArtifactID: id,
		})
	}
	for _, id := range result.UpdatedArtifacts {
		output.UpdatedArtifacts = append(output.UpdatedArtifacts, artifact.ArtifactRef{
			RootID:     ref.RootID,
			ArtifactID: id,
		})
	}
	return output, nil
}

func (a *StoreAPI) RetireWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
	expectedRevision uint64,
) (RetireWorkspaceResult, error) {
	if _, err := a.workspaceForManagement(ctx, ref); err != nil {
		return RetireWorkspaceResult{}, err
	}
	retired, err := a.collections.Retire(ctx, ref, expectedRevision)
	if err != nil {
		return RetireWorkspaceResult{}, err
	}
	return RetireWorkspaceResult{
		Workspace: ref,
		Revision:  retired.Revision,
	}, nil
}

func (a *StoreAPI) PurgeWorkspace(
	ctx context.Context,
	ref WorkspaceRef,
	expectedRevision uint64,
) (WorkspaceRef, error) {
	if err := a.managementReady(ctx); err != nil {
		return WorkspaceRef{}, err
	}
	retired, err := a.collections.GetRetired(ctx, ref)
	if err != nil {
		return WorkspaceRef{}, err
	}
	if retired.Kind != workspacecollectionv1.WorkspaceCollectionKind {
		return WorkspaceRef{}, fmt.Errorf(
			"%w: Collection %q is not a Workspace",
			workspaceDomain.ErrNotWorkspace,
			ref.CollectionID,
		)
	}
	if err := a.collections.Purge(ctx, ref, expectedRevision); err != nil {
		return WorkspaceRef{}, err
	}
	return ref, nil
}

func (a *StoreAPI) ListWorkspaceSourcesForManagement(
	ctx context.Context,
) ([]WorkspaceSourceSummary, error) {
	if err := a.managementReady(ctx); err != nil {
		return nil, err
	}
	return a.sources.List(ctx, a.workspace.workspaceRootID)
}

func (a *StoreAPI) RegisterWorkspaceDirectory(
	ctx context.Context,
	ref WorkspaceRef,
	input RegisterWorkspaceDirectoryInput,
) (WorkspaceDirectoryRegistrationResult, error) {
	current, err := a.workspaceForManagement(ctx, ref)
	if err != nil {
		return WorkspaceDirectoryRegistrationResult{}, err
	}
	if input.ExpectedCollectionRevision == 0 ||
		current.Collection.Revision != input.ExpectedCollectionRevision {
		return WorkspaceDirectoryRegistrationResult{}, basespec.ErrConflict
	}

	if input.Role == workspaceDomain.RolePrimary {
		if input.Settings.Recursive != nil || input.Settings.Authoritative != nil {
			return WorkspaceDirectoryRegistrationResult{}, fmt.Errorf(
				"%w: primary Source registration cannot include attachment overrides",
				basespec.ErrInvalid,
			)
		}
	} else if err := requireAttachableWorkspaceRole(input.Role); err != nil {
		return WorkspaceDirectoryRegistrationResult{}, err
	}

	created, createdNew, err := a.sources.CreateWithStatus(
		ctx,
		ref.RootID,
		source.Draft{
			ID:          source.SourceID(uuidutil.NewUUIDv7()),
			StorageKey:  workspaceStorageKey(),
			Kind:        source.SourceKindFilesystemDirectory,
			DisplayName: input.DisplayName,
			Enabled:     true,
			Config:      filesystemSourceConfig(input.RootPath),
		},
	)
	if err != nil {
		return WorkspaceDirectoryRegistrationResult{}, err
	}

	var workspaceValue WorkspaceView
	if input.Role == workspaceDomain.RolePrimary {
		workspaceValue, err = a.SetWorkspacePrimarySource(
			ctx,
			ref,
			SetWorkspacePrimarySourceInput{
				ExpectedCollectionRevision: input.ExpectedCollectionRevision,
				SourceID:                   created.ID,
			},
		)
	} else {
		workspaceValue, err = a.AttachWorkspaceSource(
			ctx,
			ref,
			AttachWorkspaceSourceInput{
				ExpectedCollectionRevision: input.ExpectedCollectionRevision,
				SourceID:                   created.ID,
				Role:                       input.Role,
				Enabled:                    true,
				Settings:                   input.Settings,
			},
		)
	}
	if err == nil {
		return WorkspaceDirectoryRegistrationResult{
			Source:    created,
			Workspace: workspaceValue,
		}, nil
	}

	if !createdNew {
		return WorkspaceDirectoryRegistrationResult{}, err
	}

	latest, readErr := a.workspaceForManagement(ctx, ref)
	attached := readErr == nil && workspaceHasSourceAttachment(latest, created.ID)
	if attached {
		return WorkspaceDirectoryRegistrationResult{}, err
	}

	cleanupErr := a.sources.Discard(
		context.WithoutCancel(ctx),
		ref.RootID,
		created.ID,
		created.Revision,
	)
	return WorkspaceDirectoryRegistrationResult{}, errors.Join(err, cleanupErr)
}

func (a *StoreAPI) SetWorkspaceSourceEnabled(
	ctx context.Context,
	sourceID source.SourceID,
	expectedRevision uint64,
	enabled bool,
) (WorkspaceSourceSummary, error) {
	if err := a.managementReady(ctx); err != nil {
		return WorkspaceSourceSummary{}, err
	}
	current, err := a.sources.Get(ctx, a.workspace.workspaceRootID, sourceID)
	if err != nil {
		return WorkspaceSourceSummary{}, err
	}
	return a.sources.Update(
		ctx,
		current.RootID,
		current.ID,
		source.Update{
			ExpectedRevision: expectedRevision,
			DisplayName:      current.DisplayName,
			Enabled:          enabled,
		},
	)
}

func (a *StoreAPI) AdoptWorkspaceOccurrence(
	ctx context.Context,
	ref WorkspaceRef,
	input AdoptWorkspaceOccurrenceInput,
) (WorkspaceArtifactView, error) {
	if _, err := a.workspaceForManagement(ctx, ref); err != nil {
		return WorkspaceArtifactView{}, err
	}

	key := catalog.OccurrenceKey{
		CollectionID:       ref.CollectionID,
		SourceID:           input.Occurrence.SourceID,
		Locator:            input.Occurrence.Locator,
		SubresourceLocator: input.Occurrence.SubresourceLocator,
	}
	snapshot, err := a.catalogs.CurrentCatalog(ctx, ref)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}

	var occurrence *catalog.Occurrence
	for index := range snapshot.Occurrences {
		if snapshot.Occurrences[index].Key == key {
			value := snapshot.Occurrences[index]
			occurrence = &value
			break
		}
	}
	if occurrence == nil {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: Workspace occurrence is not in the current catalog",
			workspaceDomain.ErrReferenceUnresolved,
		)
	}
	if !workspaceArtifactKindSupported(occurrence.Kind) {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: Artifact kind %q is not supported by Workspace",
			workspaceDomain.ErrInvalidWorkspace,
			occurrence.Kind,
		)
	}

	data, err := artifactadapter.EncodeArtifactData(
		workspaceDomain.ArtifactData{
			RuntimeDisabled: input.Settings.RuntimeDisabled,
		},
	)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	value, err := a.artifacts.Adopt(
		ctx,
		catalog.AdoptRequest{
			ArtifactID:              input.ArtifactID,
			Collection:              ref,
			Occurrence:              key,
			ExpectedCatalogRevision: input.ExpectedCatalogRevision,
			Name:                    input.Name,
			Enabled:                 input.Enabled,
			Data:                    data,
		},
	)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	return workspaceArtifactViewOf(value), nil
}

func (a *StoreAPI) PinWorkspaceArtifact(
	ctx context.Context,
	ref WorkspaceRef,
	input PinWorkspaceArtifactInput,
) (WorkspaceArtifactView, error) {
	if _, err := a.workspaceForManagement(ctx, ref); err != nil {
		return WorkspaceArtifactView{}, err
	}
	if !workspaceArtifactKindSupported(input.Binding.ExpectedKind) {
		return WorkspaceArtifactView{}, fmt.Errorf(
			"%w: Artifact kind %q is not supported by Workspace",
			workspaceDomain.ErrInvalidWorkspace,
			input.Binding.ExpectedKind,
		)
	}
	data, err := artifactadapter.EncodeArtifactData(
		workspaceDomain.ArtifactData{
			RuntimeDisabled: input.Settings.RuntimeDisabled,
		},
	)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	value, err := a.artifacts.Pin(
		ctx,
		catalog.PinRequest{
			ArtifactID:                 input.ArtifactID,
			Collection:                 ref,
			ExpectedCollectionRevision: input.ExpectedCollectionRevision,
			Binding:                    input.Binding,
			Name:                       input.Name,
			Enabled:                    input.Enabled,
			Data:                       data,
		},
	)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	return workspaceArtifactViewOf(value), nil
}

func (a *StoreAPI) ListWorkspaceSuppressions(
	ctx context.Context,
	ref WorkspaceRef,
) ([]WorkspaceSuppressionView, error) {
	if _, err := a.workspaceForManagement(ctx, ref); err != nil {
		return nil, err
	}
	values, err := a.artifacts.ListSuppressions(ctx, ref)
	if err != nil {
		return nil, err
	}
	output := make([]WorkspaceSuppressionView, 0, len(values))
	for _, value := range values {
		output = append(output, WorkspaceSuppressionView{
			Workspace:  ref,
			Binding:    value.Binding,
			Revision:   value.Revision,
			CreatedAt:  value.CreatedAt,
			ModifiedAt: value.ModifiedAt,
		})
	}
	return output, nil
}

func (a *StoreAPI) SuppressWorkspaceBinding(
	ctx context.Context,
	ref WorkspaceRef,
	input SuppressWorkspaceBindingInput,
) (WorkspaceSuppressionView, error) {
	if _, err := a.workspaceForManagement(ctx, ref); err != nil {
		return WorkspaceSuppressionView{}, err
	}
	if !workspaceArtifactKindSupported(input.Binding.ExpectedKind) {
		return WorkspaceSuppressionView{}, fmt.Errorf(
			"%w: Artifact kind %q is not supported by Workspace",
			workspaceDomain.ErrInvalidWorkspace,
			input.Binding.ExpectedKind,
		)
	}
	value, err := a.artifacts.Suppress(
		ctx,
		catalog.SuppressRequest{
			Collection:                 ref,
			ExpectedCollectionRevision: input.ExpectedCollectionRevision,
			Binding:                    input.Binding,
		},
	)
	if err != nil {
		return WorkspaceSuppressionView{}, err
	}
	return WorkspaceSuppressionView{
		Workspace:  ref,
		Binding:    value.Binding,
		Revision:   value.Revision,
		CreatedAt:  value.CreatedAt,
		ModifiedAt: value.ModifiedAt,
	}, nil
}

func (a *StoreAPI) UnsuppressWorkspaceBinding(
	ctx context.Context,
	ref WorkspaceRef,
	binding artifact.SourceBinding,
	expectedRevision uint64,
) (UnsuppressWorkspaceBindingResult, error) {
	if _, err := a.workspaceForManagement(ctx, ref); err != nil {
		return UnsuppressWorkspaceBindingResult{}, err
	}
	if !workspaceArtifactKindSupported(binding.ExpectedKind) {
		return UnsuppressWorkspaceBindingResult{}, fmt.Errorf(
			"%w: Artifact kind %q is not supported by Workspace",
			workspaceDomain.ErrInvalidWorkspace,
			binding.ExpectedKind,
		)
	}
	if err := a.artifacts.Unsuppress(ctx, ref, binding, expectedRevision); err != nil {
		return UnsuppressWorkspaceBindingResult{}, err
	}
	return UnsuppressWorkspaceBindingResult{
		Workspace: ref,
		Binding:   binding,
	}, nil
}

func (a *StoreAPI) SetWorkspaceArtifactEnabled(
	ctx context.Context,
	workspace WorkspaceRef,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (WorkspaceArtifactView, error) {
	if _, err := a.workspaceArtifact(ctx, workspace, ref); err != nil {
		return WorkspaceArtifactView{}, err
	}
	value, err := a.artifacts.SetEnabled(ctx, ref, expectedRevision, enabled)
	if err != nil {
		return WorkspaceArtifactView{}, err
	}
	return workspaceArtifactViewOf(value), nil
}

func (a *StoreAPI) UnadoptWorkspaceArtifact(
	ctx context.Context,
	workspace WorkspaceRef,
	ref artifact.ArtifactRef,
	input UnadoptWorkspaceArtifactInput,
) (UnadoptWorkspaceArtifactResult, error) {
	if _, err := a.workspaceArtifact(ctx, workspace, ref); err != nil {
		return UnadoptWorkspaceArtifactResult{}, err
	}
	if err := a.artifacts.Unadopt(ctx, ref, input.ExpectedRevision, input.Suppress); err != nil {
		return UnadoptWorkspaceArtifactResult{}, err
	}
	return UnadoptWorkspaceArtifactResult{Artifact: ref}, nil
}

func (a *StoreAPI) PurgeWorkspaceArtifact(
	ctx context.Context,
	workspace WorkspaceRef,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) (artifact.ArtifactRef, error) {
	if _, err := a.workspaceArtifact(ctx, workspace, ref); err != nil {
		return artifact.ArtifactRef{}, err
	}
	if err := a.artifacts.Purge(ctx, ref, expectedRevision); err != nil {
		return artifact.ArtifactRef{}, err
	}
	return ref, nil
}

func (a *StoreAPI) managementReady(ctx context.Context) error {
	if a == nil ||
		a.sources == nil ||
		a.collections == nil ||
		a.artifacts == nil ||
		a.catalogs == nil ||
		a.resources == nil ||
		a.workspace == nil ||
		a.workspace.service == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf("%w: Workspace management context is nil", basespec.ErrInvalid)
	}
	return ctx.Err()
}

func (a *StoreAPI) workspaceForManagement(
	ctx context.Context,
	ref WorkspaceRef,
) (workspaceDomain.Workspace, error) {
	if err := a.managementReady(ctx); err != nil {
		return workspaceDomain.Workspace{}, err
	}
	return a.workspace.service.Get(ctx, ref)
}

func (a *StoreAPI) workspaceViewForManagement(
	ctx context.Context,
	ref WorkspaceRef,
) (WorkspaceView, error) {
	value, err := a.workspaceForManagement(ctx, ref)
	if err != nil {
		return WorkspaceView{}, err
	}
	return a.workspaceViewForAPI(ctx, value)
}

func workspaceCollectionData(
	input WorkspaceDiscovery,
) (json.RawMessage, error) {
	value := workspaceDomain.CollectionData{
		Discovery: workspaceDomain.DiscoveryPreferences{
			AdditionalLocators: append([]basespec.Locator(nil), input.AdditionalLocators...),
			IncludeReadme:      input.IncludeReadme,
		},
	}
	for _, root := range input.AdditionalRoots {
		value.Discovery.AdditionalRoots = append(
			value.Discovery.AdditionalRoots,
			workspaceDomain.DiscoveryRoot{
				Root:            root.Root,
				Recursive:       root.Recursive,
				IncludePatterns: append([]string(nil), root.IncludePatterns...),
			},
		)
	}
	return collectiondata.EncodeCollectionData(value)
}

func workspaceAttachmentData(
	role collection.AttachmentRole,
	settings WorkspaceAttachmentSettings,
) (json.RawMessage, error) {
	if _, supported := attachmentdata.AttachmentOperationFor(role); !supported {
		return nil, fmt.Errorf(
			"%w: unsupported Workspace attachment role %q",
			workspaceDomain.ErrInvalidWorkspace,
			role,
		)
	}
	value := workspaceDomain.AttachmentData{
		Recursive:     cloneWorkspaceOptionalBool(settings.Recursive),
		Authoritative: cloneWorkspaceOptionalBool(settings.Authoritative),
	}
	if err := attachmentdata.ValidateAttachmentDataForRole(role, value); err != nil {
		return nil, err
	}
	return attachmentdata.EncodeAttachmentData(value)
}

func requireAttachableWorkspaceRole(
	role collection.AttachmentRole,
) error {
	operation, supported := attachmentdata.AttachmentOperationFor(role)
	if !supported || !operation.CanAttach {
		return fmt.Errorf(
			"%w: Workspace attachment role %q cannot be attached directly",
			workspaceDomain.ErrInvalidWorkspace,
			role,
		)
	}
	return nil
}

func workspaceArtifactKindSupported(
	kind artifact.ArtifactKind,
) bool {
	return kind == artifactbuiltin.WorkspaceContextArtifactKind ||
		kind == artifactbuiltin.AgentSkillArtifactKind
}

func workspaceHasSourceAttachment(
	workspace workspaceDomain.Workspace,
	sourceID source.SourceID,
) bool {
	for _, attachment := range workspace.Attachments {
		if attachment.SourceID == sourceID {
			return true
		}
	}
	return false
}

func filesystemSourceConfig(rootPath string) json.RawMessage {
	raw, err := json.Marshal(struct {
		RootPath string `json:"rootPath"`
	}{
		RootPath: rootPath,
	})
	if err != nil {
		panic(err)
	}
	return raw
}

func workspaceStorageKey() basespec.StorageKey {
	return basespec.StorageKey(
		"s" + strings.ReplaceAll(uuidutil.NewUUIDv7(), "-", ""),
	)
}

func cloneWorkspaceOptionalBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	output := *value
	return &output
}
