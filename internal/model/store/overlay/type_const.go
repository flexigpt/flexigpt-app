package overlay

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

const (
	SettingsOverlayPrefix = "model.runtime.v1/"
	OverlaySchemaVersion  = "v1"
)

// SettingsValueStore is the model-specific settings persistence boundary.
//
// The application adapter owns encrypted or otherwise protected persistence.
// This package stores only opaque credential references and non-secret
// runtime-overlay metadata.
type SettingsValueStore interface {
	GetModelRuntimeValue(
		ctx context.Context,
		key string,
	) (json.RawMessage, bool, error)

	PutModelRuntimeValue(
		ctx context.Context,
		key string,
		expectedRevision uint64,
		value json.RawMessage,
	) error

	DeleteModelRuntimeValue(
		ctx context.Context,
		key string,
		expectedRevision uint64,
	) error
}

// SettingsPrefixValueStore is needed only by trusted built-in topology reset
// code. Ordinary model consumers never receive this capability.
type SettingsPrefixValueStore interface {
	SettingsValueStore

	DeleteModelRuntimePrefix(
		ctx context.Context,
		prefix string,
	) error
}

// ProviderOverlay is mutable local runtime configuration for one Provider
// Artifact. It never carries raw credentials or source-definition mutation.
//
// DefaultModel is valid only for protected Providers. Mutable Providers use
// namespaced Artifact.Data for this best-effort relationship.
type ProviderOverlay struct {
	SchemaVersion string `json:"schemaVersion"`
	Revision      uint64 `json:"revision"`

	CredentialRef string `json:"credentialRef,omitempty"`

	Connection        json.RawMessage                    `json:"connection,omitempty"`
	Defaults          json.RawMessage                    `json:"defaults,omitempty"`
	Capabilities      json.RawMessage                    `json:"capabilities,omitempty"`
	DefaultModel      *declaration.ArtifactNameReference `json:"defaultModel,omitempty"`
	AdapterParameters json.RawMessage                    `json:"adapterParameters,omitempty"`
}

// ModelOverlay is mutable local runtime configuration for one Model Artifact.
// It cannot change the Provider relationship or providerModelID.
type ModelOverlay struct {
	SchemaVersion string `json:"schemaVersion"`
	Revision      uint64 `json:"revision"`

	Defaults          json.RawMessage `json:"defaults,omitempty"`
	Capabilities      json.RawMessage `json:"capabilities,omitempty"`
	AdapterParameters json.RawMessage `json:"adapterParameters,omitempty"`
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

func cloneRaw(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

type OverlayRepository interface {
	GetProviderOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (ProviderOverlay, bool, error)

	PutProviderOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		value ProviderOverlay,
	) error

	DeleteProviderOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error

	GetModelOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (ModelOverlay, bool, error)

	PutModelOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		value ModelOverlay,
	) error

	DeleteModelOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error
}

type RootPurger interface {
	PurgeRoot(
		ctx context.Context,
		rootID root.RootID,
	) error
}
