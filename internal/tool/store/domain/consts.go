package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
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

func ToolDocumentFile() basespec.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseToolPackage,
	)
}

func ToolCollectionDocumentFile() basespec.Locator {
	return documentTopology.MustDefaultDocumentFile(
		documentTopology.DocumentUseToolCollection,
	)
}
