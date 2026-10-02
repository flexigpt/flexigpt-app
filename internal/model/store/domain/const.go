package domain

import (
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelproviderv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const (
	ModelProviderArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		modelproviderv1.ModelProviderType,
	)
	ModelArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		modelv1.ModelType,
	)

	ModelProviderPackageKind source.PackageKind = "model-provider"
	ModelPackageKind         source.PackageKind = "model"

	BuiltInInitialEnabledLabel = "model.initialEnabled"

	// ManagedSourceID is Root-local. The same stable source ID is used in
	// every mutable Root because Artifact Store source identity is scoped by
	// RootID and SourceID together.
	ManagedSourceID source.SourceID = "0192c4c0-0002-7000-8000-000000000001"

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
func ManagedSourceDraft() source.Draft {
	return source.Draft{
		ID:          ManagedSourceID,
		StorageKey:  ManagedSourceStorageKey,
		Kind:        source.SourceKindManagedDirectory,
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
