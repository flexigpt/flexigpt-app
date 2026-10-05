package domain

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcppolicy/contract/v1"
)

const (
	MCPArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		mcpv1.MCPType,
	)
	MCPPolicyArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		mcppolicyv1.MCPPolicyType,
	)

	ManagedMCPPackageKind       managedpackageModel.PackageKind = "mcp"
	ManagedMCPPolicyPackageKind managedpackageModel.PackageKind = "mcp-policy"
	MCPCollectionPackageKind    managedpackageModel.PackageKind = "mcp-collection"
	SourceDecoderID             spec.DecoderID                  = "artifact.mcp-json"

	InstallationDataSchemaVersion = "v1"
	BuiltInInstallerName          = "mcp"
	HydrationSchemaVersion        = "mcp.builtin-hydration/v1"
)

func IsMCPKind(value artifactModel.ArtifactKind) bool {
	return value == MCPArtifactKind
}

func IsMCPPolicyKind(value artifactModel.ArtifactKind) bool {
	return value == MCPPolicyArtifactKind
}
