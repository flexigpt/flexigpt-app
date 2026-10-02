package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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
