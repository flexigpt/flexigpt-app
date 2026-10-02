package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
)

const (
	ToolArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		toolv1.ToolType,
	)

	ToolPackageKind           source.PackageKind = "tool"
	ToolCollectionPackageKind source.PackageKind = "tool-collection"

	BuiltInInstallerName   = "tool"
	HydrationSchemaVersion = "tool.builtin-hydration/v1"
)

func ToolDocumentFile() model.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseToolPackage,
	)
}

func ToolCollectionDocumentFile() model.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseToolCollection,
	)
}
