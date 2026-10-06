package model

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

// ProviderService owns operations on model.provider Artifacts. Its named
// children own package authoring, non-secret settings, credentials, and
// runtime capability resolution respectively.
type ProviderService struct {
	owner *Service

	Packages     *ProviderPackageService
	Settings     *ProviderSettingsService
	Credentials  *ProviderCredentialService
	Capabilities *ProviderCapabilityService
}

// ModelService owns operations on model Artifacts. Package authoring,
// non-secret settings, and runtime capability resolution remain explicit
// sub-responsibilities rather than methods on the deployment aggregate.
type ModelService struct {
	owner *Service

	Packages     *ModelPackageService
	Settings     *ModelSettingsService
	Capabilities *ModelCapabilityService
}

// ProviderPreferenceService owns default-provider selection rules. Durable
// preference persistence remains injected by the outer application aggregate.
type ProviderPreferenceService struct {
	owner *Service
}

type ProviderPackageService struct {
	owner *Service
}

type ProviderSettingsService struct {
	owner *Service
}

type ProviderCredentialService struct {
	owner *Service
}

type ProviderCapabilityService struct {
	owner *Service
}

type ModelPackageService struct {
	owner *Service
}

type ModelSettingsService struct {
	owner *Service
}

type ModelCapabilityService struct {
	owner *Service
}

func newProviderService(owner *Service) *ProviderService {
	value := &ProviderService{owner: owner}
	value.Packages = &ProviderPackageService{owner: owner}
	value.Settings = &ProviderSettingsService{owner: owner}
	value.Credentials = &ProviderCredentialService{owner: owner}
	value.Capabilities = &ProviderCapabilityService{owner: owner}
	return value
}

func newModelService(owner *Service) *ModelService {
	value := &ModelService{owner: owner}
	value.Packages = &ModelPackageService{owner: owner}
	value.Settings = &ModelSettingsService{owner: owner}
	value.Capabilities = &ModelCapabilityService{owner: owner}
	return value
}

func (s *ProviderService) List(
	ctx context.Context,
	request ListProvidersRequest,
) ([]ProviderListItem, error) {
	return s.owner.listProviders(ctx, request)
}

func (s *ProviderService) Get(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ProviderView, error) {
	return s.owner.getProvider(ctx, ref)
}

func (s *ProviderService) SetEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	return s.owner.setProviderEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (s *ProviderPackageService) Create(
	ctx context.Context,
	request ManagedProviderCreateRequest,
) (ManagedProviderCreateResult, error) {
	return s.owner.createProvider(ctx, request)
}

func (s *ProviderPackageService) Replace(
	ctx context.Context,
	request ManagedProviderReplaceRequest,
) (ManagedProviderReplaceResult, error) {
	return s.owner.replaceProvider(ctx, request)
}

func (s *ProviderPackageService) Delete(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return s.owner.deleteProvider(ctx, ref, expectedRevision)
}

func (s *ProviderSettingsService) Save(
	ctx context.Context,
	request SaveProviderSettingsRequest,
) (ProviderView, error) {
	return s.owner.saveProviderSettings(ctx, request)
}

func (s *ProviderSettingsService) Reset(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedProviderRevision uint64,
	expectedSettingsRevision uint64,
) (ProviderView, error) {
	return s.owner.resetProviderSettings(
		ctx,
		ref,
		expectedProviderRevision,
		expectedSettingsRevision,
	)
}

func (s *ProviderCredentialService) Status(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ProviderAPIKeyStatus, error) {
	return s.owner.getProviderAPIKeyStatus(ctx, ref)
}

func (s *ProviderCredentialService) Set(
	ctx context.Context,
	request SetProviderAPIKeyRequest,
) (ProviderAPIKeyStatus, error) {
	return s.owner.setProviderAPIKey(ctx, request)
}

func (s *ProviderCredentialService) Clear(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedProviderRevision uint64,
	expectedCredentialRevision uint64,
) (ProviderAPIKeyStatus, error) {
	return s.owner.clearProviderAPIKey(
		ctx,
		ref,
		expectedProviderRevision,
		expectedCredentialRevision,
	)
}

func (s *ProviderCapabilityService) Resolve(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ResolvedProvider, error) {
	return s.owner.resolveProviderArtifact(ctx, ref)
}

func (s *ProviderCapabilityService) ResolveDefaultModel(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (DefaultModelResolution, error) {
	return s.owner.resolveProviderDefaultModel(ctx, ref)
}

func (s *ModelService) List(
	ctx context.Context,
	request ListModelsRequest,
) ([]ModelListItem, error) {
	return s.owner.listModels(ctx, request)
}

func (s *ModelService) ListByProvider(
	ctx context.Context,
	request ListModelsByProviderRequest,
) ([]ModelListItem, error) {
	return s.owner.listModelsByProvider(ctx, request)
}

func (s *ModelService) Get(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ModelView, error) {
	return s.owner.getModel(ctx, ref)
}

func (s *ModelService) SetEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	return s.owner.setModelEnabled(ctx, ref, expectedRevision, enabled)
}

func (s *ModelPackageService) Create(
	ctx context.Context,
	request ManagedModelCreateRequest,
) (ManagedModelCreateResult, error) {
	return s.owner.createModel(ctx, request)
}

func (s *ModelPackageService) Replace(
	ctx context.Context,
	request ManagedModelReplaceRequest,
) (ManagedModelReplaceResult, error) {
	return s.owner.replaceModel(ctx, request)
}

func (s *ModelPackageService) Delete(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return s.owner.deleteModel(ctx, ref, expectedRevision)
}

func (s *ModelSettingsService) Save(
	ctx context.Context,
	request SaveModelSettingsRequest,
) (ModelView, error) {
	return s.owner.saveModelSettings(ctx, request)
}

func (s *ModelSettingsService) Reset(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedModelRevision uint64,
	expectedSettingsRevision uint64,
) (ModelView, error) {
	return s.owner.resetModelSettings(
		ctx,
		ref,
		expectedModelRevision,
		expectedSettingsRevision,
	)
}

func (s *ModelCapabilityService) Resolve(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ResolvedModel, error) {
	return s.owner.resolveModelArtifact(ctx, ref)
}
