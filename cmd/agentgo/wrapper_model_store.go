package main

import (
	"context"
	"fmt"
	"slices"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	modelAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model"
)

type ModelStoreWrapper struct {
	api        *modelAPI.Service
	roots      root.API
	protection root.ProtectionAPI
}

func (w *ModelStoreWrapper) ListProviders(
	rootID rootModel.RootID,
) ([]modelAPI.ProviderListItem, error) {
	if w == nil || w.api == nil {
		return nil, spec.ErrClosed
	}

	ctx := context.Background()
	roots, err := w.managementRootIDs(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]modelAPI.ProviderListItem, 0)
	for _, currentRoot := range roots {
		values, err := w.api.Providers.List(ctx, modelAPI.ListProvidersRequest{
			RootID: currentRoot,
		})
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}
	return output, nil
}

func (w *ModelStoreWrapper) ListModels(
	rootID rootModel.RootID,
) ([]modelAPI.ModelListItem, error) {
	if w == nil || w.api == nil {
		return nil, spec.ErrClosed
	}

	ctx := context.Background()
	roots, err := w.managementRootIDs(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]modelAPI.ModelListItem, 0)
	for _, currentRoot := range roots {
		values, err := w.api.Models.List(ctx, modelAPI.ListModelsRequest{
			RootID: currentRoot,
		})
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}
	return output, nil
}

func (w *ModelStoreWrapper) GetProvider(
	ref artifactModel.ArtifactRef,
) (modelAPI.ProviderView, error) {
	if w == nil || w.api == nil {
		return modelAPI.ProviderView{}, spec.ErrClosed
	}
	return w.api.Providers.Get(context.Background(), ref)
}

func (w *ModelStoreWrapper) GetModel(
	ref artifactModel.ArtifactRef,
) (modelAPI.ModelView, error) {
	if w == nil || w.api == nil {
		return modelAPI.ModelView{}, spec.ErrClosed
	}
	return w.api.Models.Get(context.Background(), ref)
}

func (w *ModelStoreWrapper) SaveModelSettings(
	request modelAPI.SaveModelSettingsRequest,
) (modelAPI.ModelView, error) {
	if w == nil || w.api == nil {
		return modelAPI.ModelView{}, spec.ErrClosed
	}
	return w.api.Models.Settings.Save(context.Background(), request)
}

func (w *ModelStoreWrapper) ResetModelSettings(
	ref artifactModel.ArtifactRef,
	expectedModelRevision uint64,
	expectedSettingsRevision uint64,
) (modelAPI.ModelView, error) {
	if w == nil || w.api == nil {
		return modelAPI.ModelView{}, spec.ErrClosed
	}
	return w.api.Models.Settings.Reset(
		context.Background(),
		ref,
		expectedModelRevision,
		expectedSettingsRevision,
	)
}

func (w *ModelStoreWrapper) GetProviderAPIKeyStatus(
	ref artifactModel.ArtifactRef,
) (modelAPI.ProviderAPIKeyStatus, error) {
	if w == nil || w.api == nil {
		return modelAPI.ProviderAPIKeyStatus{}, spec.ErrClosed
	}
	return w.api.Providers.Credentials.Status(context.Background(), ref)
}

func (w *ModelStoreWrapper) CreateModel(
	request modelAPI.ManagedModelCreateRequest,
) (modelAPI.ManagedModelCreateResult, error) {
	if w == nil || w.api == nil {
		return modelAPI.ManagedModelCreateResult{}, spec.ErrClosed
	}

	rootID, err := w.writableManagementRoot(
		context.Background(),
		request.RootID,
	)
	if err != nil {
		return modelAPI.ManagedModelCreateResult{}, err
	}
	request.RootID = rootID
	return w.api.Models.Packages.Create(context.Background(), request)
}

func (w *ModelStoreWrapper) UpdateModel(
	request modelAPI.ManagedModelReplaceRequest,
) (modelAPI.ManagedModelReplaceResult, error) {
	if w == nil || w.api == nil {
		return modelAPI.ManagedModelReplaceResult{}, spec.ErrClosed
	}
	return w.api.Models.Packages.Replace(context.Background(), request)
}

func (w *ModelStoreWrapper) DeleteModel(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	if w == nil || w.api == nil {
		return spec.ErrClosed
	}
	return w.api.Models.Packages.Delete(
		context.Background(),
		ref,
		expectedRevision,
	)
}

func (w *ModelStoreWrapper) SetModelEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	if w == nil || w.api == nil {
		return artifactModel.Artifact{}, spec.ErrClosed
	}
	return w.api.Models.SetEnabled(
		context.Background(),
		ref,
		expectedRevision,
		enabled,
	)
}

func (w *ModelStoreWrapper) managementRootIDs(
	ctx context.Context,
	requested rootModel.RootID,
) ([]rootModel.RootID, error) {
	if requested != "" {
		if err := requested.Validate(); err != nil {
			return nil, err
		}
		return []rootModel.RootID{requested}, nil
	}

	values, err := w.roots.List(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]rootModel.RootID, 0, len(values))
	for _, value := range values {
		if value.RetiredAt != nil {
			continue
		}
		output = append(output, value.ID)
	}
	slices.Sort(output)
	return output, nil
}

func (w *ModelStoreWrapper) writableManagementRoot(
	ctx context.Context,
	requested rootModel.RootID,
) (rootModel.RootID, error) {
	roots, err := w.managementRootIDs(ctx, requested)
	if err != nil {
		return "", err
	}
	for _, rootID := range roots {
		if !w.protection.IsProtectedRoot(rootID) {
			return rootID, nil
		}
	}
	return "", fmt.Errorf(
		"%w: no writable Artifact Root is available for Model authoring",
		spec.ErrReferenceUnresolved,
	)
}

func (w *ModelStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
	w.roots = nil
	w.protection = nil
}
