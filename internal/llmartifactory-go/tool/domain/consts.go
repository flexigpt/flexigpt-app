package domain

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
)

const (
	ToolArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		toolv1.ToolType,
	)

	ToolPackageKind       managedpackageModel.PackageKind = "tool"
	ToolPluginPackageKind managedpackageModel.PackageKind = "tool-plugin"

	BuiltInInstallerName   = "tool"
	HydrationSchemaVersion = "tool.builtin-hydration/v1"
)
