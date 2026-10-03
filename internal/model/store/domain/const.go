package domain

import (
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelproviderv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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

	// ManagedSourceID is Root-local. The same stable source ID is used in
	// every mutable Root because Artifact Store source identity is scoped by
	// RootID and SourceID together.
	ManagedSourceID sourceModel.SourceID = "0192c4c0-0002-7000-8000-000000000001"

	ManagedSourceStorageKey spec.StorageKey = "model-artifacts"

	BuiltInInstallerName   = "model"
	HydrationSchemaVersion = "model.builtin-hydration/v1"
)

// ManagedSourceDraft declares the single managed Source used by the Model
// Store in one mutable Root. Packages remain independent even though their
// source storage shares one Source.
//
// Discovery starts empty. The consumer API adds one exact declaration locator
// and decoder hint before each managed publication.
func ManagedSourceDraft() sourceModel.Draft {
	return sourceModel.Draft{
		ID:          ManagedSourceID,
		StorageKey:  ManagedSourceStorageKey,
		Kind:        managedfs.Kind,
		DisplayName: "Managed Model Artifacts",
		Enabled:     true,
		Config:      json.RawMessage(`{}`),
	}
}

func ModelProviderDocumentFile() spec.Locator {
	return documentTopology.ModelProviderDocumentFile()
}

func ModelDocumentFile() spec.Locator {
	return documentTopology.ModelDocumentFile()
}
