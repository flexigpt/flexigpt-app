package engine

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
)

func ValidateWorkspaceDefinition(
	value definition.Definition,
) error {
	if value.Kind != DefinitionKind {
		return fmt.Errorf(
			"%w: Workspace definition kind must be %q",
			ErrInvalidWorkspace,
			DefinitionKind,
		)
	}
	if value.SchemaID != DefinitionSchemaID {
		return fmt.Errorf(
			"%w: Workspace definition schema must be %q",
			ErrInvalidWorkspace,
			DefinitionSchemaID,
		)
	}
	if value.SchemaVersion != workspaceSchemaVersionV1 {
		return fmt.Errorf(
			"%w: Workspace definition schema version must be %q",
			ErrInvalidWorkspace,
			workspaceSchemaVersionV1,
		)
	}
	if value.LogicalName != workspaceDefinitionLogicalName {
		return fmt.Errorf(
			"%w: Workspace definition logical name must be %q",
			ErrInvalidWorkspace,
			workspaceDefinitionLogicalName,
		)
	}
	if value.DisplayName != workspaceDefinitionDisplayName {
		return fmt.Errorf(
			"%w: Workspace definition display name must be %q",
			ErrInvalidWorkspace,
			workspaceDefinitionDisplayName,
		)
	}
	if len(value.Dependencies) != 0 {
		return fmt.Errorf(
			"%w: Workspace definition cannot declare dependencies",
			ErrInvalidWorkspace,
		)
	}
	body, err := DecodeDefinitionBody[DefinitionDocument](value.Body)
	if err != nil {
		return err
	}
	if err := validateDiscoveryPreferences(body.Discovery); err != nil {
		return fmt.Errorf(
			"%w: invalid Workspace definition discovery preferences: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	return nil
}
