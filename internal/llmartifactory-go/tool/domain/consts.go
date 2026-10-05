package domain

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
)

const (
	ToolArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		toolv1.ToolType,
	)

	ToolPackageKind       managedpackageModel.PackageKind = "tool"
	ToolPluginPackageKind managedpackageModel.PackageKind = "tool-collection"

	BuiltInInstallerName   = "tool"
	HydrationSchemaVersion = "tool.builtin-hydration/v1"
)

func ToolDocumentFile() spec.Locator {
	return topology.MustDefaultDocumentFile(
		topology.DocumentUseToolPackage,
	)
}

func ToolPluginDocumentFile() spec.Locator {
	return topology.MustDefaultDocumentFile(
		topology.DocumentUseToolPlugin,
	)
}
