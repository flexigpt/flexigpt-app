package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpv1"
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

	CanonicalDecoderID basespec.DecoderID = "artifact.mcp-json"
	PolicyDecoderID    basespec.DecoderID = "artifact.mcp-policy-json"
	ConfigDecoderID    basespec.DecoderID = "mcp-config-json"
	LegacyDecoderID    basespec.DecoderID = "mcp-legacy-package-json"

	ManagedMCPPackageKind       source.PackageKind = "mcp"
	ManagedMCPPolicyPackageKind source.PackageKind = "mcp-policy"
	LegacyMCPPackageKind        source.PackageKind = "mcp-package"

	ManagedMCPDocumentFile       basespec.Locator = "mcp.json"
	ManagedMCPPolicyDocumentFile basespec.Locator = "mcp-policy.json"
	LegacyMCPDocumentFile        basespec.Locator = "mcps.json"

	InstallationDataSchemaVersion = "v2"
	RuntimeExtensionMetadataKey   = "flexigpt.dev/mcp-runtime-v1"
	BuiltInInstallerName          = "mcp"
	BuiltInRegistrySchemaVersion  = "v2"
	HydrationSchemaVersion        = "mcp.builtin-hydration/v2"
)

func IsMCPKind(value artifact.ArtifactKind) bool {
	return value == MCPArtifactKind
}

func IsMCPPolicyKind(value artifact.ArtifactKind) bool {
	return value == MCPPolicyArtifactKind
}
