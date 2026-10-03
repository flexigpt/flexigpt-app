package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const (
	ToolArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		toolv1.ToolType,
	)

	ToolPackageKind           managedpackageModel.PackageKind = "tool"
	ToolCollectionPackageKind managedpackageModel.PackageKind = "tool-collection"

	BuiltInInstallerName   = "tool"
	HydrationSchemaVersion = "tool.builtin-hydration/v1"
)

func ToolDocumentFile() spec.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseToolPackage,
	)
}

func ToolCollectionDocumentFile() spec.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseToolCollection,
	)
}
