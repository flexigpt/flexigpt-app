package managedartifact

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// PublishRequest publishes one complete managed package and verifies that the
// package resolves to the supplied pinned Artifact.
type PublishRequest struct {
	Artifact           artifact.Artifact
	ExpectedDefinition cryptoutil.Digest
	Package            source.ManagedPackagePublication
	AllowProtected     bool
}

// PublishResult reports the reconciled Artifact and managed Source state.
type PublishResult struct {
	Artifact   artifact.Artifact
	Source     source.Summary
	Generation string
	Refreshed  bool
}

// PublishCollectionRequest publishes one complete package for a managed
// Collection Source and refreshes the Collection when required.
type PublishCollectionRequest struct {
	Collection     collection.CollectionRef
	SourceID       basespec.SourceID
	Package        source.ManagedPackagePublication
	AllowProtected bool
	ForceRefresh   bool
}

// PublishCollectionResult reports the managed Source state after publication.
type PublishCollectionResult struct {
	Source     source.Summary
	Generation string
	Refreshed  bool
}

// RemoveRequest removes the managed package represented by one pinned
// Artifact and purges the reconciled missing Artifact record.
type RemoveRequest struct {
	Artifact       artifact.Artifact
	Package        source.ManagedPackageAddress
	AllowProtected bool
}
