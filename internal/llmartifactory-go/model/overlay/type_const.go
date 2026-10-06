package overlay

import (
	"context"
	"encoding/json"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	modelv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
	modelproviderv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/modelprovider/contract/v1"
)

const (
	OverlaySchemaVersion = "v1"

	// Mutable Model Artifacts store non-secret runtime state in Artifact.Data.

	ProviderRuntimeDataNamespace = "flexigpt.site/model-provider-runtime-v1"
	ModelRuntimeDataNamespace    = "flexigpt.site/model-runtime-v1"

	// Protected Model Artifacts store non-secret runtime state in Artifact Store protected overlay records.

	ProviderRuntimeNamespace overlayModel.Namespace = "model.provider.runtime"
	ModelRuntimeNamespace    overlayModel.Namespace = "model.runtime"
	PreferencesNamespace     overlayModel.Namespace = "model.preferences"

	ProviderCredentialSlot secretModel.Slot = "apiKey"
)

func Namespaces() []overlayModel.Namespace {
	return []overlayModel.Namespace{
		ProviderRuntimeNamespace,
		ModelRuntimeNamespace,
	}
}

func StoreNamespaces() []overlayModel.Namespace {
	return []overlayModel.Namespace{
		PreferencesNamespace,
	}
}

func ProviderCredentialBindingKey(
	ref artifactModel.ArtifactRef,
) secretModel.BindingKey {
	return secretModel.BindingKey{
		Artifact:  ref,
		Namespace: ProviderRuntimeNamespace,
		Slot:      ProviderCredentialSlot,
	}
}

// ProviderOverlay is non-secret local runtime state for one Model Provider.
//
// Credential metadata is stored separately in Artifact Store secret bindings.
type ProviderOverlay struct {
	SchemaVersion string `json:"schemaVersion"`
	Revision      uint64 `json:"revision"`

	Connection        json.RawMessage                    `json:"connection,omitempty"`
	Defaults          json.RawMessage                    `json:"defaults,omitempty"`
	Capabilities      json.RawMessage                    `json:"capabilities,omitempty"`
	DefaultModel      *declaration.ArtifactNameReference `json:"defaultModel,omitempty"`
	AdapterParameters json.RawMessage                    `json:"adapterParameters,omitempty"`
}

// ModelOverlay is non-secret local runtime state for one Model.
type ModelOverlay struct {
	SchemaVersion string `json:"schemaVersion"`
	Revision      uint64 `json:"revision"`

	Defaults          json.RawMessage `json:"defaults,omitempty"`
	Capabilities      json.RawMessage `json:"capabilities,omitempty"`
	AdapterParameters json.RawMessage `json:"adapterParameters,omitempty"`
}

func (v ProviderOverlay) Validate() error {
	if v.SchemaVersion != OverlaySchemaVersion {
		return fmt.Errorf(
			"%w: unsupported Model Provider overlay schema %q",
			spec.ErrInvalid,
			v.SchemaVersion,
		)
	}
	if v.Revision == 0 {
		return fmt.Errorf(
			"%w: Model Provider overlay revision is required",
			spec.ErrInvalid,
		)
	}
	if v.DefaultModel != nil {
		if err := v.DefaultModel.Validate(); err != nil {
			return fmt.Errorf(
				"model Provider overlay defaultModel: %w",
				err,
			)
		}
	}

	document := modelproviderv1.ProviderDocument{
		Type:              modelproviderv1.ModelProviderType,
		Name:              "overlay-provider",
		Adapter:           "overlay.adapter",
		Connection:        cloneRaw(v.Connection),
		DefaultModel:      cloneReference(v.DefaultModel),
		Defaults:          cloneRaw(v.Defaults),
		Capabilities:      cloneRaw(v.Capabilities),
		AdapterParameters: cloneRaw(v.AdapterParameters),
	}
	if err := document.Validate(); err != nil {
		return fmt.Errorf(
			"model Provider overlay payload: %w",
			err,
		)
	}
	return nil
}

func (v ModelOverlay) Validate() error {
	if v.SchemaVersion != OverlaySchemaVersion {
		return fmt.Errorf(
			"%w: unsupported Model overlay schema %q",
			spec.ErrInvalid,
			v.SchemaVersion,
		)
	}
	if v.Revision == 0 {
		return fmt.Errorf(
			"%w: Model overlay revision is required",
			spec.ErrInvalid,
		)
	}

	document := modelv1.ModelDocument{
		Type: modelv1.ModelType,
		Name: "overlay-model",
		Provider: declaration.ArtifactNameReference{
			Name: "overlay-provider",
		},
		ProviderModelID:   "overlay-model",
		Defaults:          cloneRaw(v.Defaults),
		Capabilities:      cloneRaw(v.Capabilities),
		AdapterParameters: cloneRaw(v.AdapterParameters),
	}
	if err := document.Validate(); err != nil {
		return fmt.Errorf(
			"model overlay payload: %w",
			err,
		)
	}
	return nil
}

func (v ProviderOverlay) Clone() ProviderOverlay {
	output := v
	output.Connection = cloneRaw(v.Connection)
	output.Defaults = cloneRaw(v.Defaults)
	output.Capabilities = cloneRaw(v.Capabilities)
	output.AdapterParameters = cloneRaw(v.AdapterParameters)
	if v.DefaultModel != nil {
		value := v.DefaultModel.Clone()
		output.DefaultModel = &value
	}
	return output
}

func (v ModelOverlay) Clone() ModelOverlay {
	output := v
	output.Defaults = cloneRaw(v.Defaults)
	output.Capabilities = cloneRaw(v.Capabilities)
	output.AdapterParameters = cloneRaw(v.AdapterParameters)
	return output
}

func cloneRaw(
	value json.RawMessage,
) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

func cloneReference(
	value *declaration.ArtifactNameReference,
) *declaration.ArtifactNameReference {
	if value == nil {
		return nil
	}
	output := value.Clone()
	return &output
}

// OverlayRepository is the Model Store local-state port.
//
// Protected Artifacts use Artifact Store protected overlays. Mutable Artifacts
// use namespaced Artifact.Data. Provider API keys always use Artifact Store
// secret bindings.
type OverlayRepository interface {
	GetProviderOverlay(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (ProviderOverlay, bool, error)

	PutProviderOverlay(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		expectedArtifactRevision uint64,
		expectedOverlayRevision uint64,
		value ProviderOverlay,
	) error

	DeleteProviderOverlay(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		expectedArtifactRevision uint64,
		expectedOverlayRevision uint64,
	) error

	GetModelOverlay(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (ModelOverlay, bool, error)

	PutModelOverlay(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		expectedArtifactRevision uint64,
		expectedOverlayRevision uint64,
		value ModelOverlay,
	) error

	DeleteModelOverlay(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		expectedArtifactRevision uint64,
		expectedOverlayRevision uint64,
	) error

	GetProviderCredential(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (secretModel.Binding, bool, error)

	ReplaceProviderCredential(
		ctx context.Context,
		request secretModel.ReplaceBindingRequest,
	) (secretModel.Binding, error)

	ClearProviderCredential(
		ctx context.Context,
		request secretModel.ClearBindingRequest,
	) error

	PurgeProviderLocalState(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) error

	PurgeModelLocalState(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) error
}
