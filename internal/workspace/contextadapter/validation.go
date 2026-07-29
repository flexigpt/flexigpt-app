package contextadapter

import (
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

const maxWorkspaceContextContentBytes = 2 << 20

func ValidateContextDefinition(
	value definition.Definition,
) error {
	if value.Kind != contextKind {
		return fmt.Errorf(
			"%w: Context definition kind must be %q",
			spec.ErrInvalidWorkspace,
			contextKind,
		)
	}
	if value.SchemaID != contextSchemaID {
		return fmt.Errorf(
			"%w: Context definition schema must be %q",
			spec.ErrInvalidWorkspace,
			contextSchemaID,
		)
	}
	if value.SchemaVersion != workspaceContextSchemaVersionV1 {
		return fmt.Errorf(
			"%w: Context definition schema version must be %q",
			spec.ErrInvalidWorkspace,
			workspaceContextSchemaVersionV1,
		)
	}
	if len(value.Dependencies) != 0 {
		return fmt.Errorf(
			"%w: Context definitions cannot declare dependencies",
			spec.ErrInvalidWorkspace,
		)
	}

	body, err := spec.DecodeDefinitionBody[contextDefinition](value.Body)
	if err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Context name",
		body.Name,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if !supportedContextRole(body.Role) {
		return fmt.Errorf(
			"%w: unsupported Context role %q",
			spec.ErrInvalidWorkspace,
			body.Role,
		)
	}
	if body.MediaType != contextMarkdownMediaType {
		return fmt.Errorf(
			"%w: unsupported Context media type %q",
			spec.ErrInvalidWorkspace,
			body.MediaType,
		)
	}
	if !utf8.ValidString(body.Content) {
		return fmt.Errorf(
			"%w: Context content must contain valid UTF-8",
			spec.ErrInvalidWorkspace,
		)
	}
	if strings.ContainsRune(body.Content, 0) {
		return fmt.Errorf(
			"%w: Context content contains a NUL byte",
			spec.ErrInvalidWorkspace,
		)
	}
	if strings.TrimSpace(body.Content) == "" {
		return fmt.Errorf(
			"%w: Context content is empty",
			spec.ErrInvalidWorkspace,
		)
	}
	if len(body.Content) > maxWorkspaceContextContentBytes {
		return fmt.Errorf(
			"%w: Context content exceeds %d bytes",
			spec.ErrInvalidWorkspace,
			maxWorkspaceContextContentBytes,
		)
	}
	if value.DisplayName != body.Name {
		return fmt.Errorf(
			"%w: Context display name does not match body.name",
			spec.ErrInvalidWorkspace,
		)
	}
	if value.LogicalName != contextLogicalName(body.Name) {
		return fmt.Errorf(
			"%w: Context logical name does not match body.name",
			spec.ErrInvalidWorkspace,
		)
	}
	if value.Labels[contextRoleLabelKey] != body.Role {
		return fmt.Errorf(
			"%w: Context role label does not match body.role",
			spec.ErrInvalidWorkspace,
		)
	}
	return nil
}

func contextLogicalName(name string) basespec.LogicalName {
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
	if len(value) > basespec.MaxLogicalNameBytes {
		value = value[:basespec.MaxLogicalNameBytes]
		value = strings.Trim(value, ".-")
	}
	if value == "" {
		value = contextVal
	}
	return basespec.LogicalName(value)
}
