package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelAggregate "github.com/flexigpt/flexigpt-app/internal/model/aggregate"
	"github.com/flexigpt/flexigpt-app/internal/model/inferenceadapter"
	modelBuiltin "github.com/flexigpt/flexigpt-app/internal/model/store/builtin"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/model/store/consumerapi"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
)

const (
	modelSettingsIndexKey  = "model-runtime-v1:index"
	modelSettingsNamespace = "model-runtime-v1"

	//nolint:gosec // Enum.
	modelCredentialPrefix = "modelcred.v1:"
)

type ModelStoreWrapper struct {
	api         *modelConsumerAPI.API
	management  *modelConsumerAPI.CatalogStore
	settings    *modelSettingsAdapter
	credentials *modelCredentialResolver
	roots       compositionapi.RootAPI
	protection  compositionapi.ProtectionAPI
}

type modelAuthKeyStore interface {
	GetAuthKey(
		ctx context.Context,
		req *settingSpec.GetAuthKeyRequest,
	) (*settingSpec.GetAuthKeyResponse, error)

	SetAuthKey(
		ctx context.Context,
		req *settingSpec.SetAuthKeyRequest,
	) (*settingSpec.SetAuthKeyResponse, error)

	DeleteAuthKey(
		ctx context.Context,
		req *settingSpec.DeleteAuthKeyRequest,
	) (*settingSpec.DeleteAuthKeyResponse, error)
}

type modelSettingsIndex struct {
	Keys map[string]string `json:"keys"`
}

// modelSettingsAdapter is structurally equivalent to MCP's settings overlay
// adapter but uses a Model-owned logical namespace and Model-specific payload
// validation in `modelOverlay.SettingsOverlayRepository`.
type modelSettingsAdapter struct {
	store modelAuthKeyStore
	mu    sync.Mutex
}

func newModelSettingsAdapter(
	store modelAuthKeyStore,
) (*modelSettingsAdapter, error) {
	if store == nil {
		return nil, errors.New("model settings store is required")
	}
	return &modelSettingsAdapter{store: store}, nil
}

func (s *modelSettingsAdapter) GetModelRuntimeValue(
	ctx context.Context,
	key string,
) (r json.RawMessage, found bool, err error) {
	if err := validateModelSettingsKey(key); err != nil {
		return nil, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.readLocked(ctx, key)
}

func (s *modelSettingsAdapter) PutModelRuntimeValue(
	ctx context.Context,
	key string,
	expectedRevision uint64,
	value json.RawMessage,
) error {
	if err := validateModelSettingsKey(key); err != nil {
		return err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		value,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return err
	}

	nextRevision, err := modelSettingsRevision(canonical)
	if err != nil {
		return err
	}
	if nextRevision != expectedRevision+1 {
		return fmt.Errorf(
			"%w: invalid Model runtime overlay revision transition",
			basespec.ErrInvalid,
		)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, found, err := s.readLocked(ctx, key)
	if err != nil {
		return err
	}
	if found {
		currentRevision, err := modelSettingsRevision(current)
		if err != nil {
			return err
		}
		if currentRevision != expectedRevision {
			return basespec.ErrConflict
		}
	} else if expectedRevision != 0 {
		return basespec.ErrConflict
	}

	index, err := s.readIndexLocked(ctx)
	if err != nil {
		return err
	}
	index.Keys[key] = modelSettingsStorageKey(key)
	if err := s.writeIndexLocked(ctx, index); err != nil {
		return err
	}

	return s.writeLocked(ctx, key, canonical)
}

func (s *modelSettingsAdapter) DeleteModelRuntimeValue(
	ctx context.Context,
	key string,
	expectedRevision uint64,
) error {
	if err := validateModelSettingsKey(key); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Model runtime overlay revision is required",
			basespec.ErrInvalid,
		)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, found, err := s.readLocked(ctx, key)
	if err != nil {
		return err
	}
	if !found {
		return basespec.ErrConflict
	}

	currentRevision, err := modelSettingsRevision(current)
	if err != nil {
		return err
	}
	if currentRevision != expectedRevision {
		return basespec.ErrConflict
	}

	if err := s.deleteLocked(ctx, key); err != nil {
		return err
	}
	index, err := s.readIndexLocked(ctx)
	if err != nil {
		return err
	}
	delete(index.Keys, key)
	return s.writeIndexLocked(ctx, index)
}

func (s *modelSettingsAdapter) DeleteModelRuntimePrefix(
	ctx context.Context,
	prefix string,
) error {
	if err := validateModelSettingsKey(prefix); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, err := s.readIndexLocked(ctx)
	if err != nil {
		return err
	}

	var result error
	for key := range index.Keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		result = errors.Join(
			result,
			s.deleteLocked(ctx, key),
		)
		delete(index.Keys, key)
	}
	result = errors.Join(result, s.writeIndexLocked(ctx, index))
	return result
}

func (s *modelSettingsAdapter) readLocked(
	ctx context.Context,
	key string,
) (c json.RawMessage, found bool, err error) {
	response, err := s.store.GetAuthKey(
		ctx,
		&settingSpec.GetAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeProvider,
			KeyName: settingSpec.AuthKeyName(
				modelSettingsStorageKey(key),
			),
		},
	)
	if err != nil {
		if modelSettingMissing(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if response == nil || response.Body == nil || !response.Body.NonEmpty {
		return nil, false, nil
	}

	raw, err := jsonutil.CanonicalizeObject(
		[]byte(response.Body.Secret),
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, false, err
	}
	return raw, true, nil
}

func (s *modelSettingsAdapter) writeLocked(
	ctx context.Context,
	key string,
	value json.RawMessage,
) error {
	_, err := s.store.SetAuthKey(
		ctx,
		&settingSpec.SetAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeProvider,
			KeyName: settingSpec.AuthKeyName(
				modelSettingsStorageKey(key),
			),
			Body: &settingSpec.SetAuthKeyRequestBody{
				Secret: string(value),
			},
		},
	)
	return err
}

func (s *modelSettingsAdapter) deleteLocked(
	ctx context.Context,
	key string,
) error {
	_, err := s.store.DeleteAuthKey(
		ctx,
		&settingSpec.DeleteAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeProvider,
			KeyName: settingSpec.AuthKeyName(
				modelSettingsStorageKey(key),
			),
		},
	)
	if modelSettingMissing(err) {
		return nil
	}
	return err
}

func (s *modelSettingsAdapter) readIndexLocked(
	ctx context.Context,
) (modelSettingsIndex, error) {
	raw, found, err := s.readLocked(ctx, modelSettingsIndexKey)
	if err != nil {
		return modelSettingsIndex{}, err
	}
	if !found {
		return modelSettingsIndex{
			Keys: map[string]string{},
		}, nil
	}

	var value modelSettingsIndex
	if err := json.Unmarshal(raw, &value); err != nil {
		return modelSettingsIndex{}, err
	}
	if value.Keys == nil {
		value.Keys = map[string]string{}
	}
	return value, nil
}

func (s *modelSettingsAdapter) writeIndexLocked(
	ctx context.Context,
	index modelSettingsIndex,
) error {
	raw, err := jsonutil.MarshalCanonicalObject(
		index,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return err
	}
	return s.writeLocked(ctx, modelSettingsIndexKey, raw)
}

func modelSettingsRevision(
	raw json.RawMessage,
) (uint64, error) {
	var value struct {
		Revision uint64 `json:"revision"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, err
	}
	return value.Revision, nil
}

func modelSettingsStorageKey(
	logical string,
) string {
	sum := sha256.Sum256([]byte(logical))
	return modelSettingsNamespace + ":" + hex.EncodeToString(sum[:])
}

func validateModelSettingsKey(value string) error {
	if strings.TrimSpace(value) == "" ||
		strings.TrimSpace(value) != value ||
		strings.ContainsRune(value, 0) {
		return fmt.Errorf(
			"%w: invalid Model settings key",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func modelSettingMissing(err error) bool {
	if err == nil {
		return false
	}
	value := strings.ToLower(err.Error())
	return strings.Contains(value, "not found") ||
		strings.Contains(value, "does not exist")
}

type modelCredentialReference struct {
	Provider artifact.ArtifactRef `json:"provider"`
}

type modelCredentialResolver struct {
	store modelAuthKeyStore
}

func newModelCredentialResolver(
	store modelAuthKeyStore,
) (*modelCredentialResolver, error) {
	if store == nil {
		return nil, errors.New("model credential store is required")
	}
	return &modelCredentialResolver{store: store}, nil
}

func modelCredentialRef(
	provider artifact.ArtifactRef,
) (string, error) {
	if err := provider.Validate(); err != nil {
		return "", err
	}
	raw, err := jsonutil.MarshalCanonicalObject(
		modelCredentialReference{
			Provider: provider,
		},
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return "", err
	}
	return modelCredentialPrefix +
		base64.RawURLEncoding.EncodeToString(raw), nil
}

func parseModelCredentialRef(
	value string,
) (modelCredentialReference, error) {
	if !strings.HasPrefix(value, modelCredentialPrefix) {
		return modelCredentialReference{}, fmt.Errorf(
			"%w: unsupported Model credential reference",
			basespec.ErrInvalid,
		)
	}
	raw, err := base64.RawURLEncoding.DecodeString(
		strings.TrimPrefix(value, modelCredentialPrefix),
	)
	if err != nil {
		return modelCredentialReference{}, err
	}

	var output modelCredentialReference
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		raw,
		&output,
		basespec.MaxLocalDataBytes,
	); err != nil {
		return modelCredentialReference{}, err
	}
	if err := output.Provider.Validate(); err != nil {
		return modelCredentialReference{}, err
	}
	return output, nil
}

func modelCredentialStorageKey(
	value string,
) string {
	sum := sha256.Sum256([]byte(value))
	return modelCredentialPrefix + hex.EncodeToString(sum[:])
}

func (r *modelCredentialResolver) ResolveModelCredential(
	ctx context.Context,
	ref string,
) (inferenceadapter.Credential, error) {
	if r == nil || r.store == nil {
		return inferenceadapter.Credential{}, basespec.ErrClosed
	}
	if _, err := parseModelCredentialRef(ref); err != nil {
		return inferenceadapter.Credential{}, err
	}

	response, err := r.store.GetAuthKey(
		ctx,
		&settingSpec.GetAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeProvider,
			KeyName: settingSpec.AuthKeyName(
				modelCredentialStorageKey(ref),
			),
		},
	)
	if err != nil {
		if modelSettingMissing(err) {
			return inferenceadapter.Credential{}, fmt.Errorf(
				"%w: Model credential is unavailable",
				basespec.ErrReferenceUnresolved,
			)
		}
		return inferenceadapter.Credential{}, err
	}
	if response == nil || response.Body == nil || !response.Body.NonEmpty {
		return inferenceadapter.Credential{}, fmt.Errorf(
			"%w: Model credential is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}
	return inferenceadapter.Credential{
		APIKey:  response.Body.Secret,
		Version: response.Body.SHA256,
	}, nil
}

func (r *modelCredentialResolver) SetProviderCredential(
	ctx context.Context,
	provider artifact.ArtifactRef,
	secret string,
) (string, error) {
	if r == nil || r.store == nil {
		return "", basespec.ErrClosed
	}
	ref, err := modelCredentialRef(provider)
	if err != nil {
		return "", err
	}

	_, err = r.store.SetAuthKey(
		ctx,
		&settingSpec.SetAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeProvider,
			KeyName: settingSpec.AuthKeyName(
				modelCredentialStorageKey(ref),
			),
			Body: &settingSpec.SetAuthKeyRequestBody{
				Secret: secret,
			},
		},
	)
	if err != nil {
		return "", err
	}
	return ref, nil
}

func (r *modelCredentialResolver) DeleteProviderCredential(
	ctx context.Context,
	provider artifact.ArtifactRef,
) error {
	if r == nil || r.store == nil {
		return basespec.ErrClosed
	}
	ref, err := modelCredentialRef(provider)
	if err != nil {
		return err
	}
	_, err = r.store.DeleteAuthKey(
		ctx,
		&settingSpec.DeleteAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeProvider,
			KeyName: settingSpec.AuthKeyName(
				modelCredentialStorageKey(ref),
			),
		},
	)
	if modelSettingMissing(err) {
		return nil
	}
	return err
}

func InitModelWrappers(
	ctx context.Context,
	storeWrapper *ModelStoreWrapper,
	aggregateWrapper *ModelAggregateWrapper,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	roots compositionapi.RootAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	hydrator topology.CompiledHydrationCoordinator,
	settingsStore modelAuthKeyStore,
) (builtin.HydrationInstaller, error) {
	if storeWrapper == nil || aggregateWrapper == nil {
		return nil, errors.New("model wrapper receivers are incomplete")
	}
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		managedArtifacts == nil ||
		roots == nil ||
		protection == nil ||
		hydrator == nil ||
		settingsStore == nil {
		return nil, errors.New("model wrapper dependencies are incomplete")
	}

	settings, err := newModelSettingsAdapter(settingsStore)
	if err != nil {
		return nil, err
	}
	overlays, err := modelOverlay.NewSettingsOverlayRepository(settings)
	if err != nil {
		return nil, err
	}
	credentials, err := newModelCredentialResolver(settingsStore)
	if err != nil {
		return nil, err
	}
	runtimeAdapter, err := inferenceadapter.NewRuntimeAdapter(credentials)
	if err != nil {
		return nil, err
	}

	api, err := modelConsumerAPI.New(modelConsumerAPI.Dependencies{
		Sources:          sources,
		Discovery:        discovery,
		Artifacts:        artifacts,
		ManagedArtifacts: managedArtifacts,
		Protection:       protection,
		Overlays:         overlays,
		Adapters:         runtimeAdapter,
		BuiltinRoot:      documentTopology.BuiltinRootID(),
	})
	if err != nil {
		return nil, err
	}
	management, err := modelConsumerAPI.NewManagementStore(api)
	if err != nil {
		return nil, err
	}
	catalog, err := modelConsumerAPI.NewCatalogStore(api)
	if err != nil {
		return nil, err
	}
	aggregateService, err := modelAggregate.New(
		management,
		runtimeAdapter,
	)
	if err != nil {
		return nil, err
	}
	cleanup, err := modelConsumerAPI.NewBuiltinPackageCleanup(api)
	if err != nil {
		return nil, err
	}
	installer, err := modelBuiltin.NewInstaller(
		modelBuiltin.InstallerDependencies{
			Hydrator: hydrator,
			Cleanup:  cleanup,
			Overlays: overlays,
		},
	)
	if err != nil {
		return nil, err
	}

	storeWrapper.api = api
	storeWrapper.management = catalog
	storeWrapper.settings = settings
	storeWrapper.credentials = credentials
	storeWrapper.roots = roots
	storeWrapper.protection = protection
	aggregateWrapper.service = aggregateService

	_ = ctx
	return installer, nil
}

func (w *ModelStoreWrapper) ListModelProviders(
	rootID root.RootID,
) ([]modelConsumerAPI.ProviderListItem, error) {
	if w == nil || w.management == nil {
		return nil, basespec.ErrClosed
	}
	ctx := context.Background()
	roots, err := w.managementRootIDs(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]modelConsumerAPI.ProviderListItem, 0)
	for _, currentRoot := range roots {
		values, err := w.management.ListProviders(ctx, currentRoot)
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}
	return output, nil
}

func (w *ModelStoreWrapper) ListModels(
	rootID root.RootID,
) ([]modelConsumerAPI.ModelListItem, error) {
	if w == nil || w.management == nil {
		return nil, basespec.ErrClosed
	}
	ctx := context.Background()
	roots, err := w.managementRootIDs(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]modelConsumerAPI.ModelListItem, 0)
	for _, currentRoot := range roots {
		values, err := w.management.ListModels(ctx, currentRoot)
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}
	return output, nil
}

func (w *ModelStoreWrapper) GetModelProvider(
	ref artifact.ArtifactRef,
) (modelConsumerAPI.ProviderView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ProviderView{}, basespec.ErrClosed
	}
	return w.api.GetProvider(context.Background(), ref)
}

func (w *ModelStoreWrapper) GetModel(
	ref artifact.ArtifactRef,
) (modelConsumerAPI.ModelView, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ModelView{}, basespec.ErrClosed
	}
	return w.api.GetModel(context.Background(), ref)
}

func (w *ModelStoreWrapper) CreateModelProvider(
	request modelConsumerAPI.ManagedProviderCreateRequest,
) (modelConsumerAPI.ManagedProviderCreateResult, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ManagedProviderCreateResult{}, basespec.ErrClosed
	}
	rootID, err := w.writableManagementRoot(context.Background(), request.RootID)
	if err != nil {
		return modelConsumerAPI.ManagedProviderCreateResult{}, err
	}
	request.RootID = rootID

	return w.api.CreateProvider(context.Background(), request)
}

func (w *ModelStoreWrapper) ReplaceModelProvider(
	request modelConsumerAPI.ManagedProviderReplaceRequest,
) (modelConsumerAPI.ManagedProviderReplaceResult, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ManagedProviderReplaceResult{}, basespec.ErrClosed
	}
	return w.api.ReplaceProvider(context.Background(), request)
}

func (w *ModelStoreWrapper) DeleteModelProvider(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if w == nil || w.api == nil {
		return basespec.ErrClosed
	}
	return w.api.DeleteProvider(
		context.Background(),
		ref,
		expectedRevision,
	)
}

func (w *ModelStoreWrapper) CreateManagedModel(
	request modelConsumerAPI.ManagedModelCreateRequest,
) (modelConsumerAPI.ManagedModelCreateResult, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ManagedModelCreateResult{}, basespec.ErrClosed
	}
	rootID, err := w.writableManagementRoot(context.Background(), request.RootID)
	if err != nil {
		return modelConsumerAPI.ManagedModelCreateResult{}, err
	}
	request.RootID = rootID

	return w.api.CreateModel(context.Background(), request)
}

func (w *ModelStoreWrapper) ReplaceManagedModel(
	request modelConsumerAPI.ManagedModelReplaceRequest,
) (modelConsumerAPI.ManagedModelReplaceResult, error) {
	if w == nil || w.api == nil {
		return modelConsumerAPI.ManagedModelReplaceResult{}, basespec.ErrClosed
	}
	return w.api.ReplaceModel(context.Background(), request)
}

func (w *ModelStoreWrapper) DeleteManagedModel(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if w == nil || w.api == nil {
		return basespec.ErrClosed
	}
	return w.api.DeleteModel(
		context.Background(),
		ref,
		expectedRevision,
	)
}

func (w *ModelStoreWrapper) SetModelProviderEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if w == nil || w.api == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	return w.api.SetProviderEnabled(
		context.Background(),
		ref,
		expectedRevision,
		enabled,
	)
}

func (w *ModelStoreWrapper) SetModelEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if w == nil || w.api == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	return w.api.SetModelEnabled(
		context.Background(),
		ref,
		expectedRevision,
		enabled,
	)
}

func (w *ModelStoreWrapper) SetModelProviderCredential(
	provider artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	secret string,
) (modelConsumerAPI.ProviderRuntimeOverlayView, error) {
	if w == nil || w.api == nil || w.credentials == nil {
		return modelConsumerAPI.ProviderRuntimeOverlayView{}, basespec.ErrClosed
	}

	ctx := context.Background()
	current, err := w.api.GetProviderRuntimeOverlay(
		ctx,
		provider,
	)
	if err != nil {
		return modelConsumerAPI.ProviderRuntimeOverlayView{}, err
	}
	if current.Revision != expectedOverlayRevision {
		return modelConsumerAPI.ProviderRuntimeOverlayView{}, basespec.ErrConflict
	}

	ref := ""
	if strings.TrimSpace(secret) == "" {
		if err := w.credentials.DeleteProviderCredential(ctx, provider); err != nil {
			return modelConsumerAPI.ProviderRuntimeOverlayView{}, err
		}
	} else {
		ref, err = w.credentials.SetProviderCredential(ctx, provider, secret)
		if err != nil {
			return modelConsumerAPI.ProviderRuntimeOverlayView{}, err
		}
	}

	return w.api.UpdateProviderRuntimeOverlay(
		ctx,
		modelConsumerAPI.ProviderRuntimeOverlayUpdateRequest{
			Provider:          provider,
			ExpectedRevision:  expectedOverlayRevision,
			CredentialRef:     ref,
			Connection:        current.Connection,
			Defaults:          current.Defaults,
			Capabilities:      current.Capabilities,
			DefaultModel:      current.DefaultModel,
			AdapterParameters: current.AdapterParameters,
		},
	)
}

type ModelSelectionView struct {
	Revision     uint64                `json:"revision"`
	DefaultModel *artifact.ArtifactRef `json:"defaultModel,omitempty"`
}

type ModelSelectionUpdateRequest struct {
	ExpectedRevision uint64                `json:"expectedRevision"`
	DefaultModel     *artifact.ArtifactRef `json:"defaultModel,omitempty"`
}

type storedModelSelection struct {
	SchemaVersion string                `json:"schemaVersion"`
	Revision      uint64                `json:"revision"`
	DefaultModel  *artifact.ArtifactRef `json:"defaultModel,omitempty"`
}

const (
	modelSelectionSettingsKey   = "model.runtime.v1/selection"
	modelSelectionSchemaVersion = "v1"
)

func (w *ModelStoreWrapper) GetModelSelection() (ModelSelectionView, error) {
	if w == nil || w.settings == nil {
		return ModelSelectionView{}, basespec.ErrClosed
	}

	raw, found, err := w.settings.GetModelRuntimeValue(
		context.Background(),
		modelSelectionSettingsKey,
	)
	if err != nil {
		return ModelSelectionView{}, err
	}
	if !found {
		return ModelSelectionView{}, nil
	}

	var stored storedModelSelection
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		raw,
		&stored,
		basespec.MaxLocalDataBytes,
	); err != nil {
		return ModelSelectionView{}, err
	}
	if stored.SchemaVersion != modelSelectionSchemaVersion || stored.Revision == 0 {
		return ModelSelectionView{}, fmt.Errorf(
			"%w: invalid Model selection settings",
			basespec.ErrInvalid,
		)
	}
	if stored.DefaultModel != nil {
		if err := stored.DefaultModel.Validate(); err != nil {
			return ModelSelectionView{}, err
		}
	}

	return ModelSelectionView{
		Revision:     stored.Revision,
		DefaultModel: stored.DefaultModel,
	}, nil
}

func (w *ModelStoreWrapper) UpdateModelSelection(
	request ModelSelectionUpdateRequest,
) (ModelSelectionView, error) {
	if w == nil || w.api == nil || w.settings == nil {
		return ModelSelectionView{}, basespec.ErrClosed
	}

	current, err := w.GetModelSelection()
	if err != nil {
		return ModelSelectionView{}, err
	}
	if request.ExpectedRevision != current.Revision {
		return ModelSelectionView{}, basespec.ErrConflict
	}
	if request.DefaultModel != nil {
		if _, err := w.api.GetModel(context.Background(), *request.DefaultModel); err != nil {
			return ModelSelectionView{}, err
		}
	}

	next := storedModelSelection{
		SchemaVersion: modelSelectionSchemaVersion,
		Revision:      current.Revision + 1,
		DefaultModel:  request.DefaultModel,
	}
	raw, err := jsonutil.MarshalCanonicalObject(next, basespec.MaxLocalDataBytes)
	if err != nil {
		return ModelSelectionView{}, err
	}
	if err := w.settings.PutModelRuntimeValue(
		context.Background(),
		modelSelectionSettingsKey,
		current.Revision,
		raw,
	); err != nil {
		return ModelSelectionView{}, err
	}

	return ModelSelectionView{
		Revision:     next.Revision,
		DefaultModel: next.DefaultModel,
	}, nil
}

func (w *ModelStoreWrapper) managementRootIDs(
	ctx context.Context,
	requested root.RootID,
) ([]root.RootID, error) {
	if w == nil || w.roots == nil || w.protection == nil {
		return nil, basespec.ErrClosed
	}
	if requested != "" {
		if err := requested.Validate(); err != nil {
			return nil, err
		}
		return []root.RootID{requested}, nil
	}

	values, err := w.roots.List(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]root.RootID, 0, len(values))
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
	requested root.RootID,
) (root.RootID, error) {
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
		basespec.ErrReferenceUnresolved,
	)
}

func (w *ModelStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
	w.management = nil
	w.settings = nil
	w.credentials = nil
	w.roots = nil
	w.protection = nil
}
