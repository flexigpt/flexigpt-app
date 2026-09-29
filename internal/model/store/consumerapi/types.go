package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelproviderv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
)

// AdapterDescriptor is the runtime-neutral adapter identity that participates
// in Model resolution and configuration fingerprints.
type AdapterDescriptor struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

func (d AdapterDescriptor) Validate() error {
	if err := basespec.ValidateIdentifier(
		"Model adapter ID",
		d.ID,
		basespec.MaxKindBytes,
	); err != nil {
		return err
	}
	return basespec.ValidateRequiredText(
		"Model adapter version",
		d.Version,
		basespec.MaxVersionBytes,
	)
}

// AdapterRegistry is implemented by the outer inference adapter.
//
// Model Store never imports inference-go. The outer adapter owns adapter
// installation, supported protocol IDs, and adapter versioning.
type AdapterRegistry interface {
	LookupModelAdapter(
		ctx context.Context,
		adapter string,
	) (AdapterDescriptor, bool, error)
}

type Dependencies struct {
	Sources          compositionapi.SourceAPI
	Discovery        compositionapi.DiscoveryAPI
	Artifacts        compositionapi.ArtifactAPI
	ManagedArtifacts compositionapi.ManagedArtifactAPI
	Protection       compositionapi.ProtectionAPI

	Overlays modelOverlay.OverlayRepository
	Adapters AdapterRegistry

	BuiltinRoot root.RootID
}

type ListProvidersRequest struct {
	RootID  root.RootID `json:"rootID"`
	Enabled *bool       `json:"enabled,omitempty"`
}

func (r ListProvidersRequest) Validate() error {
	return r.RootID.Validate()
}

type ListModelsRequest struct {
	RootID  root.RootID `json:"rootID"`
	Enabled *bool       `json:"enabled,omitempty"`
}

func (r ListModelsRequest) Validate() error {
	return r.RootID.Validate()
}

// ListModelsByProviderRequest is a declaration-level query. It filters by the
// exact authored Model.provider logical reference, not by a resolved runtime
// Provider Artifact.
type ListModelsByProviderRequest struct {
	RootID   root.RootID                       `json:"rootID"`
	Provider declaration.ArtifactNameReference `json:"provider"`
	Enabled  *bool                             `json:"enabled,omitempty"`
}

func (r ListModelsByProviderRequest) Validate() error {
	if err := r.RootID.Validate(); err != nil {
		return err
	}
	return r.Provider.Validate()
}

type ProviderListItem struct {
	Ref artifact.ArtifactRef `json:"ref"`

	Name        basespec.LogicalName `json:"name"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description,omitempty"`
	Adapter     string               `json:"adapter,omitempty"`

	DefaultModel *declaration.ArtifactNameReference `json:"defaultModel,omitempty"`

	State            artifact.State    `json:"state"`
	Enabled          bool              `json:"enabled"`
	Revision         uint64            `json:"revision"`
	DefinitionDigest cryptoutil.Digest `json:"definitionDigest,omitempty"`
	BuiltIn          bool              `json:"builtIn"`

	CredentialConfigured   bool   `json:"credentialConfigured"`
	RuntimeOverlayRevision uint64 `json:"runtimeOverlayRevision"`
}

type ModelListItem struct {
	Ref artifact.ArtifactRef `json:"ref"`

	Name            basespec.LogicalName               `json:"name"`
	DisplayName     string                             `json:"displayName"`
	Description     string                             `json:"description,omitempty"`
	Provider        *declaration.ArtifactNameReference `json:"provider,omitempty"`
	ProviderModelID string                             `json:"providerModelID,omitempty"`

	State            artifact.State    `json:"state"`
	Enabled          bool              `json:"enabled"`
	Revision         uint64            `json:"revision"`
	DefinitionDigest cryptoutil.Digest `json:"definitionDigest,omitempty"`
	BuiltIn          bool              `json:"builtIn"`
}

type ProviderView struct {
	Artifact         artifact.Artifact                `json:"artifact"`
	DefinitionDigest cryptoutil.Digest                `json:"definitionDigest"`
	Document         modelproviderv1.ProviderDocument `json:"document"`
	BuiltIn          bool                             `json:"builtIn"`
}

type ModelView struct {
	Artifact         artifact.Artifact     `json:"artifact"`
	DefinitionDigest cryptoutil.Digest     `json:"definitionDigest"`
	Document         modelv1.ModelDocument `json:"document"`
	BuiltIn          bool                  `json:"builtIn"`
}

// ProviderRuntimeOverlayView intentionally exposes only whether a credential
// is configured. The opaque credential reference remains internal runtime
// material and is not a frontend API value.
type ProviderRuntimeOverlayView struct {
	Revision             uint64                             `json:"revision"`
	CredentialConfigured bool                               `json:"credentialConfigured"`
	Connection           json.RawMessage                    `json:"connection,omitempty"`
	Defaults             json.RawMessage                    `json:"defaults,omitempty"`
	Capabilities         json.RawMessage                    `json:"capabilities,omitempty"`
	DefaultModel         *declaration.ArtifactNameReference `json:"defaultModel,omitempty"`
	AdapterParameters    json.RawMessage                    `json:"adapterParameters,omitempty"`
}

type ModelRuntimeOverlayView struct {
	Revision          uint64          `json:"revision"`
	Defaults          json.RawMessage `json:"defaults,omitempty"`
	Capabilities      json.RawMessage `json:"capabilities,omitempty"`
	AdapterParameters json.RawMessage `json:"adapterParameters,omitempty"`
}

// ProviderRuntimeOverlayUpdateRequest is complete replacement state for the
// local Provider overlay. The caller must retain fields it wishes to preserve.
//
// DefaultModel is accepted only for protected Providers. Mutable Providers use
// `SetMutableProviderDefaultModel` so the best-effort default relationship is
// stored in namespaced Artifact.Data as required by the HLD.
type ProviderRuntimeOverlayUpdateRequest struct {
	Provider          artifact.ArtifactRef               `json:"provider"`
	ExpectedRevision  uint64                             `json:"expectedRevision"`
	CredentialRef     string                             `json:"credentialRef,omitempty"`
	Connection        json.RawMessage                    `json:"connection,omitempty"`
	Defaults          json.RawMessage                    `json:"defaults,omitempty"`
	Capabilities      json.RawMessage                    `json:"capabilities,omitempty"`
	DefaultModel      *declaration.ArtifactNameReference `json:"defaultModel,omitempty"`
	AdapterParameters json.RawMessage                    `json:"adapterParameters,omitempty"`
}

type ModelRuntimeOverlayUpdateRequest struct {
	Model             artifact.ArtifactRef `json:"model"`
	ExpectedRevision  uint64               `json:"expectedRevision"`
	Defaults          json.RawMessage      `json:"defaults,omitempty"`
	Capabilities      json.RawMessage      `json:"capabilities,omitempty"`
	AdapterParameters json.RawMessage      `json:"adapterParameters,omitempty"`
}

// ResolvedModel is the internal source-backed Model Store resolution result.
// It contains independent Provider and Model source artifacts, their local
// runtime overlays, adapter metadata, and a stable configuration fingerprint.
//
// Capability merging intentionally does not happen here. The external
// inference adapter owns adapter-base capability derivation, provider/model
// capability patch application, and request validation.
type ResolvedModel struct {
	Model    modelDomain.Model
	Provider modelDomain.Provider

	ProviderOverlay modelOverlay.ProviderOverlay
	ModelOverlay    modelOverlay.ModelOverlay

	Adapter     AdapterDescriptor
	Fingerprint cryptoutil.Digest
}

type DefaultModelSource string

const (
	DefaultModelSourceMutableArtifactData DefaultModelSource = "mutableArtifactData"
	DefaultModelSourceProtectedOverlay    DefaultModelSource = "protectedOverlay"
	DefaultModelSourceProviderDeclaration DefaultModelSource = "providerDeclaration"
	DefaultModelSourceFirstEnabledModel   DefaultModelSource = "firstEnabledLinkedModel"
)

type DefaultModelResolution struct {
	Resolved ResolvedModel
	Source   DefaultModelSource
}

type ManagedProviderCreateRequest struct {
	RootID   root.RootID                      `json:"rootID"`
	Document modelproviderv1.ProviderDocument `json:"document"`
	Enabled  bool                             `json:"enabled"`
}

type ManagedProviderCreateResult struct {
	Artifact artifact.Artifact        `json:"artifact"`
	Address  artifact.ArtifactAddress `json:"address"`
}

type ManagedProviderReplaceRequest struct {
	Provider                 artifact.ArtifactRef             `json:"provider"`
	ExpectedArtifactRevision uint64                           `json:"expectedArtifactRevision"`
	Document                 modelproviderv1.ProviderDocument `json:"document"`
	Enabled                  bool                             `json:"enabled"`
}

type ManagedProviderReplaceResult struct {
	Artifact artifact.Artifact        `json:"artifact"`
	Address  artifact.ArtifactAddress `json:"address"`
}

type ManagedModelCreateRequest struct {
	RootID   root.RootID           `json:"rootID"`
	Document modelv1.ModelDocument `json:"document"`
	Enabled  bool                  `json:"enabled"`
}

type ManagedModelCreateResult struct {
	Artifact artifact.Artifact        `json:"artifact"`
	Address  artifact.ArtifactAddress `json:"address"`
}

type ManagedModelReplaceRequest struct {
	Model                    artifact.ArtifactRef  `json:"model"`
	ExpectedArtifactRevision uint64                `json:"expectedArtifactRevision"`
	Document                 modelv1.ModelDocument `json:"document"`
	Enabled                  bool                  `json:"enabled"`
}

type ManagedModelReplaceResult struct {
	Artifact artifact.Artifact        `json:"artifact"`
	Address  artifact.ArtifactAddress `json:"address"`
}

func cloneOptionalReference(
	value *declaration.ArtifactNameReference,
) *declaration.ArtifactNameReference {
	if value == nil {
		return nil
	}
	output := value.Clone()
	return &output
}

func cloneRaw(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

func validateExpectedArtifactRevision(value uint64) error {
	if value == 0 {
		return fmt.Errorf(
			"%w: expected Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	return nil
}
