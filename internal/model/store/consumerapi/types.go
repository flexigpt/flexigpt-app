package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/secret"
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

	State    artifact.State `json:"state"`
	Enabled  bool           `json:"enabled"`
	Revision uint64         `json:"revision"`
	BuiltIn  bool           `json:"builtIn"`
}

type ModelListItem struct {
	Ref artifact.ArtifactRef `json:"ref"`

	Name            basespec.LogicalName               `json:"name"`
	DisplayName     string                             `json:"displayName"`
	Description     string                             `json:"description,omitempty"`
	Provider        *modelDomain.ArtifactNameReference `json:"provider,omitempty"`
	ProviderModelID string                             `json:"providerModelID,omitempty"`

	State    artifact.State `json:"state"`
	Enabled  bool           `json:"enabled"`
	Revision uint64         `json:"revision"`
	BuiltIn  bool           `json:"builtIn"`
}

// These aliases preserve the consumer API names while making the public
// values Model-domain projections rather than persistence payloads.
type (
	ProviderSettings = modelDomain.ProviderSettings
	ModelSettings    = modelDomain.ModelSettings
)

type ProviderView struct {
	Artifact         artifact.Artifact                  `json:"artifact"`
	DefinitionDigest cryptoutil.Digest                  `json:"definitionDigest"`
	Document         modelDomain.ProviderDocument       `json:"document"`
	Settings         ProviderSettings                   `json:"settings"`
	DefaultModel     *modelDomain.ArtifactNameReference `json:"defaultModel,omitempty"`
	BuiltIn          bool                               `json:"builtIn"`
}

type ModelView struct {
	Artifact         artifact.Artifact         `json:"artifact"`
	DefinitionDigest cryptoutil.Digest         `json:"definitionDigest"`
	Document         modelDomain.ModelDocument `json:"document"`
	Settings         ModelSettings             `json:"settings"`
	BuiltIn          bool                      `json:"builtIn"`
}

// SaveProviderSettingsRequest is complete replacement state for local
// Provider settings. Omitted fields inherit from the Provider declaration.
type SaveProviderSettingsRequest struct {
	Provider                 artifact.ArtifactRef               `json:"provider"`
	ExpectedProviderRevision uint64                             `json:"expectedProviderRevision"`
	ExpectedSettingsRevision uint64                             `json:"expectedSettingsRevision"`
	Connection               *modelDomain.ConnectionPatch       `json:"connection,omitempty"`
	Defaults                 *modelDomain.DefaultsPatch         `json:"defaults,omitempty"`
	Capabilities             *modelDomain.CapabilitiesPatch     `json:"capabilities,omitempty"`
	DefaultModel             *modelDomain.ArtifactNameReference `json:"defaultModel,omitempty"`
	AdapterParameters        *modelDomain.AdapterParameters     `json:"adapterParameters,omitempty"`
}

// SaveModelSettingsRequest is complete replacement state for local Model
// settings. Omitted fields inherit from the Model declaration.
type SaveModelSettingsRequest struct {
	Model                    artifact.ArtifactRef           `json:"model"`
	ExpectedModelRevision    uint64                         `json:"expectedModelRevision"`
	ExpectedSettingsRevision uint64                         `json:"expectedSettingsRevision"`
	Defaults                 *modelDomain.DefaultsPatch     `json:"defaults,omitempty"`
	Capabilities             *modelDomain.CapabilitiesPatch `json:"capabilities,omitempty"`
	AdapterParameters        *modelDomain.AdapterParameters `json:"adapterParameters,omitempty"`
}

// ProviderAPIKeyStatus is the only readable API-key information. API key
// plaintext and secret hashes are intentionally never exposed.
type ProviderAPIKeyStatus struct {
	ProviderRevision uint64 `json:"providerRevision"`
	APIKeyRevision   uint64 `json:"apiKeyRevision"`
	Configured       bool   `json:"configured"`
}

type SetProviderAPIKeyRequest struct {
	Provider                 artifact.ArtifactRef `json:"provider"`
	ExpectedProviderRevision uint64               `json:"expectedProviderRevision"`
	ExpectedAPIKeyRevision   uint64               `json:"expectedAPIKeyRevision"`
	APIKey                   string               `json:"apiKey"`
}

// ResolvedProvider is the source-backed Provider runtime input. It contains
// no plaintext secret. Runtime adapters resolve the secret only when they need
// to build an in-memory inference ProviderParam.
type ResolvedProvider struct {
	Provider           modelDomain.Provider
	ProviderOverlay    modelOverlay.ProviderOverlay
	ProviderCredential *secret.Binding
	Adapter            AdapterDescriptor
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

	ProviderOverlay    modelOverlay.ProviderOverlay
	ProviderCredential *secret.Binding
	ModelOverlay       modelOverlay.ModelOverlay

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
	RootID   root.RootID                  `json:"rootID"`
	Document modelDomain.ProviderDocument `json:"document"`
	Enabled  bool                         `json:"enabled"`
}

type ManagedProviderCreateResult struct {
	Artifact artifact.Artifact        `json:"artifact"`
	Address  artifact.ArtifactAddress `json:"address"`
}

type ManagedProviderReplaceRequest struct {
	Provider                 artifact.ArtifactRef         `json:"provider"`
	ExpectedArtifactRevision uint64                       `json:"expectedArtifactRevision"`
	Document                 modelDomain.ProviderDocument `json:"document"`
	Enabled                  bool                         `json:"enabled"`
}

type ManagedProviderReplaceResult struct {
	Artifact artifact.Artifact        `json:"artifact"`
	Address  artifact.ArtifactAddress `json:"address"`
}

type ManagedModelCreateRequest struct {
	RootID   root.RootID               `json:"rootID"`
	Document modelDomain.ModelDocument `json:"document"`
	Enabled  bool                      `json:"enabled"`
}

type ManagedModelCreateResult struct {
	Artifact artifact.Artifact        `json:"artifact"`
	Address  artifact.ArtifactAddress `json:"address"`
}

type ManagedModelReplaceRequest struct {
	Model                    artifact.ArtifactRef      `json:"model"`
	ExpectedArtifactRevision uint64                    `json:"expectedArtifactRevision"`
	Document                 modelDomain.ModelDocument `json:"document"`
	Enabled                  bool                      `json:"enabled"`
}

type ManagedModelReplaceResult struct {
	Artifact artifact.Artifact        `json:"artifact"`
	Address  artifact.ArtifactAddress `json:"address"`
}

func cloneOptionalModelReference(
	value *modelDomain.ArtifactNameReference,
) *modelDomain.ArtifactNameReference {
	if value == nil {
		return nil
	}
	output := value.Clone()
	return &output
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
