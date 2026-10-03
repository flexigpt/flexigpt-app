package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const (
	MCPArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		mcpv1.MCPType,
	)
	MCPPolicyArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		mcppolicyv1.MCPPolicyType,
	)

	ManagedMCPPackageKind       sourceModel.PackageKind = "mcp"
	ManagedMCPPolicyPackageKind sourceModel.PackageKind = "mcp-policy"
	MCPCollectionPackageKind    sourceModel.PackageKind = "mcp-collection"
	SourceDecoderID             spec.DecoderID          = "artifact.mcp-json"

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
