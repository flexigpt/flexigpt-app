package contextadapter

import (
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

const maxWorkspaceContextContentBytes = 2 << 20

func ValidateContextDefinition(
	value definition.Definition,
) error {
	if value.Kind != contextKind {
		return fmt.Errorf(
			"%w: Context definition kind must be %q",
			engine.ErrInvalidWorkspace,
			contextKind,
		)
	}
	if value.SchemaID != contextSchemaID {
		return fmt.Errorf(
			"%w: Context definition schema must be %q",
			engine.ErrInvalidWorkspace,
			contextSchemaID,
		)
	}
	if value.SchemaVersion != workspaceContextSchemaVersionV1 {
		return fmt.Errorf(
			"%w: Context definition schema version must be %q",
			engine.ErrInvalidWorkspace,
			workspaceContextSchemaVersionV1,
		)
	}
	if len(value.Dependencies) != 0 {
		return fmt.Errorf(
			"%w: Context definitions cannot declare dependencies",
			engine.ErrInvalidWorkspace,
		)
	}

	body, err := engine.DecodeDefinitionBody[contextDefinition](value.Body)
	if err != nil {
		return err
	}
	if err := artifactstore.ValidateRequiredText(
		"Context name",
		body.Name,
		artifactstore.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if !supportedContextRole(body.Role) {
		return fmt.Errorf(
			"%w: unsupported Context role %q",
			engine.ErrInvalidWorkspace,
			body.Role,
		)
	}
	if body.MediaType != contextMarkdownMediaType {
		return fmt.Errorf(
			"%w: unsupported Context media type %q",
			engine.ErrInvalidWorkspace,
			body.MediaType,
		)
	}
	if !utf8.ValidString(body.Content) {
		return fmt.Errorf(
			"%w: Context content must contain valid UTF-8",
			engine.ErrInvalidWorkspace,
		)
	}
	if strings.ContainsRune(body.Content, 0) {
		return fmt.Errorf(
			"%w: Context content contains a NUL byte",
			engine.ErrInvalidWorkspace,
		)
	}
	if strings.TrimSpace(body.Content) == "" {
		return fmt.Errorf(
			"%w: Context content is empty",
			engine.ErrInvalidWorkspace,
		)
	}
	if len(body.Content) > maxWorkspaceContextContentBytes {
		return fmt.Errorf(
			"%w: Context content exceeds %d bytes",
			engine.ErrInvalidWorkspace,
			maxWorkspaceContextContentBytes,
		)
	}
	if value.DisplayName != body.Name {
		return fmt.Errorf(
			"%w: Context display name does not match body.name",
			engine.ErrInvalidWorkspace,
		)
	}
	if value.LogicalName != contextLogicalName(body.Name) {
		return fmt.Errorf(
			"%w: Context logical name does not match body.name",
			engine.ErrInvalidWorkspace,
		)
	}
	if value.Labels[contextRoleLabelKey] != body.Role {
		return fmt.Errorf(
			"%w: Context role label does not match body.role",
			engine.ErrInvalidWorkspace,
		)
	}
	return nil
}

func contextLogicalName(name string) artifactstore.LogicalName {
	contextVal := "context"
	base := strings.ToLower(strings.TrimSuffix(name, path.Ext(name)))
	parts := strings.FieldsFunc(base, func(character rune) bool {
		return (character < 'a' || character > 'z') &&
			(character < '0' || character > '9')
	})

	value := strings.Join(parts, "-")
	if value == "" {
		value = contextVal
	}
	if value[0] >= '0' && value[0] <= '9' {
		value = "context-" + value
	}
	if len(value) > artifactstore.MaxLogicalNameBytes {
		value = value[:artifactstore.MaxLogicalNameBytes]
		value = strings.Trim(value, ".-")
	}
	if value == "" {
		value = contextVal
	}
	return artifactstore.LogicalName(value)
}
