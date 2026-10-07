package domain

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	modelv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
	modelproviderv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/modelprovider/contract/v1"
)

const (
	ModelProviderArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		modelproviderv1.ModelProviderType,
	)
	ModelArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		modelv1.ModelType,
	)

	ModelProviderPackageKind managedpackageModel.PackageKind = "model-provider"
	ModelPackageKind         managedpackageModel.PackageKind = "model"

	BuiltInInitialEnabledLabel = "model.initialEnabled"

	ManagedSourceStorageKey spec.StorageKey = "model-artifacts"

	BuiltInInstallerName   = "model"
	HydrationSchemaVersion = "model.builtin-hydration/v1"
)
