package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const (
	MCPArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		mcpv1.MCPType,
	)
	MCPPolicyArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		mcppolicyv1.MCPPolicyType,
	)

	ManagedMCPPackageKind       source.PackageKind = "mcp"
	ManagedMCPPolicyPackageKind source.PackageKind = "mcp-policy"
	MCPCollectionPackageKind    source.PackageKind = "mcp-collection"
	SourceDecoderID             spec.DecoderID     = "artifact.mcp-json"

	InstallationDataSchemaVersion = "v1"
	BuiltInInstallerName          = "mcp"
	HydrationSchemaVersion        = "mcp.builtin-hydration/v1"
)

func IsMCPKind(value artifact.ArtifactKind) bool {
	return value == MCPArtifactKind
}

func IsMCPPolicyKind(value artifact.ArtifactKind) bool {
	return value == MCPPolicyArtifactKind
}
