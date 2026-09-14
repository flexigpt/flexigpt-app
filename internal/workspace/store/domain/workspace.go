package domain

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
)

type WorkspaceRef = artifact.ArtifactRef

const WorkspaceArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
	workspacev1.WorkspaceType,
)

// Workspace joins the ordinary Workspace Artifact record to its typed
// workspacev1 Definition. It is a consumer value, not an Artifact Store
// aggregate or parent entity.
type Workspace struct {
	Artifact   artifact.Artifact
	Definition definition.Definition
	Document   workspacev1.WorkspaceDocument
}

func NewWorkspace(
	record artifact.Artifact,
	value definition.Definition,
) (Workspace, error) {
	if err := record.Validate(); err != nil {
		return Workspace{}, err
	}
	if err := value.Validate(); err != nil {
		return Workspace{}, err
	}
	if record.Kind != WorkspaceArtifactKind {
		return Workspace{}, fmt.Errorf(
			"%w: Artifact %q has kind %q",
			ErrNotWorkspace,
			record.ID,
			record.Kind,
		)
	}
	if record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != value.Digest {
		return Workspace{}, fmt.Errorf(
			"%w: Workspace Artifact Definition is unavailable",
			ErrReferenceUnresolved,
		)
	}
	if value.Kind != WorkspaceArtifactKind ||
		value.SchemaID != workspacev1.WorkspaceSchemaKey.SchemaID ||
		value.SchemaVersion != workspacev1.WorkspaceSchemaKey.SchemaVersion {
		return Workspace{}, fmt.Errorf(
			"%w: Artifact Definition is not a workspacev1 declaration",
			ErrInvalidWorkspace,
		)
	}

	document, err := workspacev1.DecodeWorkspaceJSON(value.Body)
	if err != nil {
		return Workspace{}, fmt.Errorf(
			"%w: decode Workspace Definition: %w",
			ErrInvalidWorkspace,
			err,
		)
	}
	if document.Name != string(value.LogicalName) {
		return Workspace{}, fmt.Errorf(
			"%w: Workspace declaration name differs from Definition logical name",
			ErrInvalidWorkspace,
		)
	}
	if document.Description != value.Description {
		return Workspace{}, fmt.Errorf(
			"%w: Workspace declaration description differs from Definition",
			ErrInvalidWorkspace,
		)
	}

	output := Workspace{
		Artifact:   record.Clone(),
		Definition: value.Clone(),
		Document:   document,
	}
	if err := output.Validate(); err != nil {
		return Workspace{}, err
	}
	return output, nil
}

func (w Workspace) Ref() WorkspaceRef {
	return w.Artifact.Ref()
}

func (w Workspace) Validate() error {
	if err := w.Artifact.Validate(); err != nil {
		return err
	}
	if err := w.Definition.Validate(); err != nil {
		return err
	}
	if w.Artifact.Kind != WorkspaceArtifactKind ||
		w.Definition.Kind != WorkspaceArtifactKind ||
		w.Artifact.RootID == "" {
		return fmt.Errorf(
			"%w: invalid Workspace Artifact projection",
			ErrInvalidWorkspace,
		)
	}
	if err := w.Document.Validate(); err != nil {
		return err
	}
	return nil
}

func (w Workspace) RootID() string {
	return string(w.Artifact.RootID)
}
