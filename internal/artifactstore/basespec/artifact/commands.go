package artifact

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// PublishArtifactRequest is source-first managed publication intent.
//
// Artifact Store does not create pinned placeholders. It publishes Source
// content, refreshes the Source, and verifies that the expected source-backed
// Artifact exists with the requested identity and Definition digest.
type PublishArtifactRequest struct {
	RootID                  root.RootID                      `json:"rootID"`
	Binding                 SourceBinding                    `json:"binding"`
	ExpectedKind            ArtifactKind                     `json:"expectedKind"`
	ExpectedLogicalName     basespec.LogicalName             `json:"expectedLogicalName"`
	ExpectedDefinition      cryptoutil.Digest                `json:"expectedDefinition"`
	Package                 source.ManagedPackagePublication `json:"package"`
	AllowPackageReplacement bool                             `json:"allowPackageReplacement,omitempty"`
	AllowProtected          bool                             `json:"allowProtected"`
}

type PublishArtifactResult struct {
	Artifact   Artifact       `json:"artifact"`
	Source     source.Summary `json:"source"`
	Generation string         `json:"generation"`
	Refreshed  bool           `json:"refreshed"`
}

// RemoveArtifactRequest removes one managed Source package. The Artifact
// record remains in the Store as missing local state until explicitly purged.
type RemoveArtifactRequest struct {
	RootID             root.RootID                  `json:"rootID"`
	SourceID           source.SourceID              `json:"sourceID"`
	Package            source.ManagedPackageAddress `json:"package"`
	ExpectedGeneration string                       `json:"expectedGeneration,omitempty"`
	ExpectedArtifact   *ArtifactRef                 `json:"expectedArtifact,omitempty"`
	AllowProtected     bool                         `json:"allowProtected"`
}

// ExpectedGeneration optionally prevents removal from replacing a package
// mutation that occurred after a caller read its source-backed Artifact.
