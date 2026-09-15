package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

const (
	MCPArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		mcpv1.MCPType,
	)
	MCPPolicyArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		mcppolicyv1.MCPPolicyType,
	)

	SourceDecoderID basespec.DecoderID = "artifact.mcp-json"

	ManagedMCPPackageKind       source.PackageKind = "mcp"
	ManagedMCPPolicyPackageKind source.PackageKind = "mcp-policy"
	MCPCollectionPackageKind    source.PackageKind = "mcp-collection"

	ManagedMCPDocumentFile       basespec.Locator = "mcp.json"
	ManagedMCPPolicyDocumentFile basespec.Locator = "mcp-policy.json"
	MCPCollectionDocumentFile    basespec.Locator = "collection.yaml"

	InstallationDataSchemaVersion = "v1"
	RuntimeExtensionMetadataKey   = "flexigpt.site/mcp-runtime-v1"
	BuiltInInstallerName          = "mcp"
	HydrationSchemaVersion        = "mcp.builtin-hydration/v1"
)

func IsMCPKind(value artifact.ArtifactKind) bool {
	return value == MCPArtifactKind
}

func IsMCPPolicyKind(value artifact.ArtifactKind) bool {
	return value == MCPPolicyArtifactKind
}
