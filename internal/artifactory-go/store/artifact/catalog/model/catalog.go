// Package model contains Artifact Store's internal committed read spec.
//
// It is not a transport model and is never returned directly by Agent, Skill,
// Plugin, MCP, Tool, or Workspace consumer APIs.
//
// The Artifact Store owns this projection because it can read Artifact,
// Source, and immutable Definition metadata in one durable read. Consumer
// domains own their own public list responses.
package model

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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
	Kind            artifactModel.ArtifactKind
	Enabled         *bool
	LogicalNames    []spec.LogicalName
}

// DefinitionMetadata is the small Definition projection needed by ordinary
// domain listings. Definition.Body, labels, and dependency selectors are not
// present unless ListOptions.IncludeDocument is true.
type DefinitionMetadata struct {
	Digest        cryptoutil.Digest
	SchemaID      schemaModel.SchemaID
	SchemaVersion string
	Description   string
}

// SourceMetadata is the small Source projection needed by domain list
// classification such as managed versus built-in.
type SourceMetadata struct {
	ID         sourceModel.SourceID
	Kind       sourceModel.SourceKind
	StorageKey spec.StorageKey
	Enabled    bool
}

// Entry is one committed Artifact catalog row.
//
// Artifact and Definition admission happened before this row became visible.
// Metadata reads deliberately do not reconstruct Artifact.Data, diagnostics,
// complete Definitions, schema validation, or source verification.
type Entry struct {
	ID      artifactModel.ArtifactID
	RootID  rootModel.RootID
	Binding artifactModel.SourceBinding

	Kind           artifactModel.ArtifactKind
	LogicalName    spec.LogicalName
	LogicalVersion spec.LogicalVersion

	DisplayName string
	State       artifactModel.State
	Enabled     bool
	Revision    uint64

	Source     SourceMetadata
	Definition *DefinitionMetadata

	// Document is populated only when ListOptions.IncludeDocument is true.
	// It is immutable Store data selected by Ref.RootID and Definition.Digest.
	Document *definitionModel.Definition
}

func (e Entry) Ref() artifactModel.ArtifactRef {
	return artifactModel.ArtifactRef{
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
