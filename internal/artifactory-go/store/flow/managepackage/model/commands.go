package model

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type PublishRequest struct {
	RootID                  rootModel.RootID                              `json:"rootID"`
	Binding                 artifactModel.SourceBinding                   `json:"binding"`
	ExpectedKind            artifactModel.ArtifactKind                    `json:"expectedKind"`
	ExpectedLogicalName     spec.LogicalName                              `json:"expectedLogicalName"`
	ExpectedDefinition      cryptoutil.Digest                             `json:"expectedDefinition"`
	Package                 managedpackageModel.ManagedPackagePublication `json:"package"`
	AllowPackageReplacement bool                                          `json:"allowPackageReplacement,omitempty"`
}

type PublishResult struct {
	Artifact   artifactModel.Artifact `json:"artifact"`
	Source     sourceModel.Summary    `json:"source"`
	Generation string                 `json:"generation"`
	Refreshed  bool                   `json:"refreshed"`
}

type RemoveRequest struct {
	RootID                rootModel.RootID                          `json:"rootID"`
	SourceID              sourceModel.SourceID                      `json:"sourceID"`
	Package               managedpackageModel.ManagedPackageAddress `json:"package"`
	ExpectedGeneration    string                                    `json:"expectedGeneration,omitempty"`
	ExpectedArtifact      *artifactModel.ArtifactRef                `json:"expectedArtifact,omitempty"`
	PruneDiscoveryLocator *spec.Locator                             `json:"pruneDiscoveryLocator,omitempty"`
}
