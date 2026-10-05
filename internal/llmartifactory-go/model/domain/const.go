package domain

import (
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
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

// ManagedSourceDraft declares the single managed Source used by the Model
// Store in one mutable Root. Packages remain independent even though their
// source storage shares one Source.
//
// "candidateID" is used only when Source.Ensure creates a Root-local Source.
// Existing Roots reuse the Source returned by storage-key lookup.
func ManagedSourceDraft(
	candidateID sourceModel.SourceID,
) sourceModel.Draft {
	return sourceModel.Draft{
		ID:          candidateID,
		StorageKey:  ManagedSourceStorageKey,
		Kind:        managedfs.Kind,
		DisplayName: "Managed Model Artifacts",
		Enabled:     true,
		Config:      json.RawMessage(`{}`),
	}
}

func ModelProviderDocumentFile() spec.Locator {
	return topology.ModelProviderDocumentFile()
}

func ModelDocumentFile() spec.Locator {
	return topology.ModelDocumentFile()
}
