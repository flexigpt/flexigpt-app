package mcp

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
)

type ListServersRequest struct {
	RootID  rootModel.RootID `json:"rootID"`
	Enabled *bool            `json:"enabled,omitempty"`
}

type ServerListItem struct {
	Ref artifactModel.ArtifactRef `json:"ref"`

	Name        spec.LogicalName `json:"name"`
	DisplayName string           `json:"displayName"`
	Description string           `json:"description,omitempty"`

	State            artifactModel.State `json:"state"`
	Enabled          bool                `json:"enabled"`
	Revision         uint64              `json:"revision"`
	DefinitionDigest cryptoutil.Digest   `json:"definitionDigest,omitempty"`
	BuiltIn          bool                `json:"builtIn"`
}

type ListPoliciesRequest struct {
	RootID  rootModel.RootID `json:"rootID"`
	Enabled *bool            `json:"enabled,omitempty"`
}

type PolicyListItem struct {
	Ref artifactModel.ArtifactRef `json:"ref"`

	Name        spec.LogicalName `json:"name"`
	DisplayName string           `json:"displayName"`
	Description string           `json:"description,omitempty"`

	State            artifactModel.State `json:"state"`
	Enabled          bool                `json:"enabled"`
	Revision         uint64              `json:"revision"`
	DefinitionDigest cryptoutil.Digest   `json:"definitionDigest,omitempty"`
	BuiltIn          bool                `json:"builtIn"`
}

type InstallationInputView struct {
	Value            *string `json:"value,omitempty"`
	SecretConfigured bool    `json:"secretConfigured"`
}

type ServerInstallationDataView struct {
	SelectedConnectionProfile string                           `json:"selectedConnectionProfile,omitempty"`
	Inputs                    map[string]InstallationInputView `json:"inputs,omitempty"`
	AdditionalPolicies        []artifactModel.ArtifactRef      `json:"additionalPolicies,omitempty"`
}

type ServerInstallationView struct {
	Artifact     artifactModel.Artifact         `json:"artifact"`
	Document     serverMCPDomain.ServerDocument `json:"document"`
	Installation ServerInstallationDataView     `json:"installation"`

	// InstallationRevision is the optimistic-concurrency token for a write
	// through UpdateServerInstallation or UpdateProtectedServerInstallation.
	//
	// Mutable servers use their Artifact revision. Protected servers use the
	// persisted overlay revision, which is zero when no overlay exists yet.
	InstallationRevision uint64 `json:"installationRevision"`
	BuiltIn              bool   `json:"builtIn"`
}

type ServerSecretInputView struct {
	Name        string                    `json:"name"`
	Label       string                    `json:"label,omitempty"`
	Description string                    `json:"description,omitempty"`
	Required    bool                      `json:"required"`
	Kind        serverMCPDomain.InputKind `json:"kind"`
	Configured  bool                      `json:"configured"`
}

type ServerSecretsView struct {
	Inputs []ServerSecretInputView `json:"inputs"`
}

func installationDataView(
	value serverMCPDomain.ServerData,
) ServerInstallationDataView {
	output := ServerInstallationDataView{
		SelectedConnectionProfile: value.SelectedConnectionProfile,
		Inputs:                    make(map[string]InstallationInputView, len(value.Inputs)),
		AdditionalPolicies:        append([]artifactModel.ArtifactRef(nil), value.AdditionalPolicies...),
	}
	for name, binding := range value.Inputs {
		input := InstallationInputView{
			SecretConfigured: binding.SecretRef != "",
		}
		if binding.Value != nil {
			copyValue := *binding.Value
			input.Value = &copyValue
		}
		output.Inputs[name] = input
	}
	return output
}

// ServerRead is shared by aggregate projections. It is not a wire DTO:
// Resolved contains installation-local references and must remain backend-only.
type ServerRead struct {
	Settings ServerInstallationView   `json:"-"`
	Resolved serverMCPDomain.Resolved `json:"-"`
}

type PolicyView struct {
	Artifact artifactModel.Artifact `json:"artifact"`
	Body     mcpPolicy.MCPPolicy    `json:"body"`
	BuiltIn  bool                   `json:"builtIn"`
}

type ManagedMCPPolicyUpsertRequest struct {
	Plugin                 artifactModel.ArtifactRef `json:"plugin"`
	ExpectedPluginRevision uint64                    `json:"expectedPluginRevision"`
	Name                   spec.LogicalName          `json:"name"`
	Description            string                    `json:"description,omitempty"`
	Policy                 mcpPolicy.MCPPolicy       `json:"policy"`
	Enabled                bool                      `json:"enabled"`
}

type ManagedMCPPolicyUpsertResult struct {
	Artifact          artifactModel.Artifact        `json:"artifact"`
	Address           artifactModel.ArtifactAddress `json:"address"`
	Plugin            pluginAPI.PluginView          `json:"plugin"`
	MembershipCreated bool                          `json:"membershipCreated"`
}

type ManagedMCPCreateRequest struct {
	Plugin                 artifactModel.ArtifactRef      `json:"plugin"`
	ExpectedPluginRevision uint64                         `json:"expectedPluginRevision"`
	Document               serverMCPDomain.ServerDocument `json:"document"`
	Enabled                bool                           `json:"enabled"`
}

type ManagedMCPCreateResult struct {
	Artifact          artifactModel.Artifact        `json:"artifact"`
	Address           artifactModel.ArtifactAddress `json:"address"`
	Plugin            pluginAPI.PluginView          `json:"plugin"`
	MembershipCreated bool                          `json:"membershipCreated"`
}

type ManagedMCPReplaceRequest struct {
	Plugin                   artifactModel.ArtifactRef      `json:"plugin"`
	ExpectedPluginRevision   uint64                         `json:"expectedPluginRevision"`
	Artifact                 artifactModel.ArtifactRef      `json:"artifact"`
	ExpectedArtifactRevision uint64                         `json:"expectedArtifactRevision"`
	Document                 serverMCPDomain.ServerDocument `json:"document"`
	Enabled                  bool                           `json:"enabled"`
}

type ManagedMCPReplaceResult struct {
	Artifact artifactModel.Artifact        `json:"artifact"`
	Address  artifactModel.ArtifactAddress `json:"address"`
	Plugin   pluginAPI.PluginView          `json:"plugin"`
}

type BuiltInArtifactExpectation struct {
	Locator          spec.Locator               `json:"locator"`
	Subresource      spec.SubresourceLocator    `json:"subresource"`
	Kind             artifactModel.ArtifactKind `json:"kind"`
	LogicalName      spec.LogicalName           `json:"logicalName"`
	DefinitionDigest cryptoutil.Digest          `json:"definitionDigest"`
}

type ServerStore interface {
	ResolveMCPServer(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (ServerRead, error)

	GetServerSettings(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (ServerInstallationView, error)

	SaveServerSettings(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		expectedSettingsRevision uint64,
		data serverMCPDomain.ServerData,
	) error
}

// ManagementStore is the aggregate-facing MCP persistence port. It includes
// mutation operations whose callers must coordinate runtime invalidation.
type ManagementStore interface {
	ServerStore

	GetMCPPolicy(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (PolicyView, error)

	ListMCPPluginServers(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) ([]ServerRead, error)

	ListMCPServersReferencingPolicy(
		ctx context.Context,
		rootID rootModel.RootID,
		policyName spec.LogicalName,
	) ([]artifactModel.ArtifactRef, error)

	CreateMCPServer(
		ctx context.Context,
		request ManagedMCPCreateRequest,
	) (ManagedMCPCreateResult, error)

	UpdateMCPServer(
		ctx context.Context,
		request ManagedMCPReplaceRequest,
	) (ManagedMCPReplaceResult, error)

	DeleteMCPServer(ctx context.Context, ref artifactModel.ArtifactRef, expectedRevision uint64) error
	SaveMCPPolicy(
		ctx context.Context,
		request ManagedMCPPolicyUpsertRequest,
	) (ManagedMCPPolicyUpsertResult, error)
	DeleteMCPPolicy(ctx context.Context, ref artifactModel.ArtifactRef, expectedRevision uint64) error
}
