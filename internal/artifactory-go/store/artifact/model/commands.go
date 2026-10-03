package model

import (
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// PublishArtifactRequest is source-first managed publication intent.
//
// Artifact Store does not create pinned placeholders. It publishes Source
// content, refreshes the Source, and verifies that the expected source-backed
// Artifact exists with the requested identity and Definition digest.
type PublishArtifactRequest struct {
	RootID                  rootModel.RootID                      `json:"rootID"`
	Binding                 SourceBinding                         `json:"binding"`
	ExpectedKind            ArtifactKind                          `json:"expectedKind"`
	ExpectedLogicalName     spec.LogicalName                      `json:"expectedLogicalName"`
	ExpectedDefinition      cryptoutil.Digest                     `json:"expectedDefinition"`
	Package                 sourceModel.ManagedPackagePublication `json:"package"`
	AllowPackageReplacement bool                                  `json:"allowPackageReplacement,omitempty"`
	AllowProtected          bool                                  `json:"allowProtected"`
}

type PublishArtifactResult struct {
	Artifact   Artifact            `json:"artifact"`
	Source     sourceModel.Summary `json:"source"`
	Generation string              `json:"generation"`
	Refreshed  bool                `json:"refreshed"`
}

// RemoveArtifactRequest removes one managed Source package. The Artifact
// record remains in the Store as missing local state until explicitly purged.
type RemoveArtifactRequest struct {
	RootID             rootModel.RootID                  `json:"rootID"`
	SourceID           sourceModel.SourceID              `json:"sourceID"`
	Package            sourceModel.ManagedPackageAddress `json:"package"`
	ExpectedGeneration string                            `json:"expectedGeneration,omitempty"`
	ExpectedArtifact   *ArtifactRef                      `json:"expectedArtifact,omitempty"`

	// PruneDiscoveryLocator removes one exact explicit declaration candidate
	// after source-side package removal and before the final Source refresh.
	// It is valid only for an authoritative managed Source owned by the
	// caller's managed authoring flow.
	PruneDiscoveryLocator *spec.Locator `json:"pruneDiscoveryLocator,omitempty"`

	AllowProtected bool `json:"allowProtected"`
}

// ExpectedGeneration optionally prevents removal from replacing a package
// mutation that occurred after a caller read its source-backed Artifact.
