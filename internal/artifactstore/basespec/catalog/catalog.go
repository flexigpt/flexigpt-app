// Package catalog contains Artifact Store's internal committed read model.
//
// It is not a transport model and is never returned directly by Agent, Skill,
// Collection, MCP, Tool, or Workspace consumer APIs.
//
// The Artifact Store owns this projection because it can read Artifact,
// Source, and immutable Definition metadata in one durable read. Consumer
// domains own their own public list responses.
package catalog

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// ListOptions controls extra immutable material attached to normal Artifact
// list operations.
//
// The default is metadata-only. IncludeDocument performs one bulk immutable
// Definition read after the metadata query. It never changes Artifact identity,
// Source state, local enablement, or list filtering semantics.
type ListOptions struct {
	IncludeDocument bool
	Kind            artifact.ArtifactKind
	Enabled         *bool
	LogicalNames    []basespec.LogicalName
}

// DefinitionMetadata is the small Definition projection needed by ordinary
// domain listings. Definition.Body, labels, and dependency selectors are not
// present unless ListOptions.IncludeDocument is true.
type DefinitionMetadata struct {
	Digest        cryptoutil.Digest
	SchemaID      schema.SchemaID
	SchemaVersion string
	Description   string
}

// SourceMetadata is the small Source projection needed by domain list
// classification such as managed versus built-in.
type SourceMetadata struct {
	ID         source.SourceID
	Kind       source.SourceKind
	StorageKey basespec.StorageKey
	Enabled    bool
}

// Entry is one committed Artifact catalog row.
//
// Artifact and Definition admission happened before this row became visible.
// Metadata reads deliberately do not reconstruct Artifact.Data, diagnostics,
// complete Definitions, schema validation, or source verification.
type Entry struct {
	ID      artifact.ArtifactID
	RootID  root.RootID
	Binding artifact.SourceBinding

	Kind           artifact.ArtifactKind
	LogicalName    basespec.LogicalName
	LogicalVersion basespec.LogicalVersion

	DisplayName string
	State       artifact.State
	Enabled     bool
	Revision    uint64

	Source     SourceMetadata
	Definition *DefinitionMetadata

	// Document is populated only when ListOptions.IncludeDocument is true.
	// It is immutable Store data selected by Ref.RootID and Definition.Digest.
	Document *definition.Definition
}

func (e Entry) Ref() artifact.ArtifactRef {
	return artifact.ArtifactRef{
		RootID:     e.RootID,
		ArtifactID: e.ID,
	}
}

func (e Entry) Clone() Entry {
	output := e
	if e.Definition != nil {
		value := *e.Definition
		output.Definition = &value
	}
	if e.Document != nil {
		value := e.Document.Clone()
		output.Document = &value
	}
	return output
}
