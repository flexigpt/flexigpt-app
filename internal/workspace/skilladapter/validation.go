package skilladapter

import (
	"fmt"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

func ValidateSkillDefinition(
	value definition.Definition,
) error {
	if value.Kind != skillKind {
		return fmt.Errorf(
			"%w: Skill definition kind must be %q",
			engine.ErrInvalidWorkspace,
			skillKind,
		)
	}
	if value.SchemaID != skillSchemaID {
		return fmt.Errorf(
			"%w: Skill definition schema must be %q",
			engine.ErrInvalidWorkspace,
			skillSchemaID,
		)
	}
	if value.SchemaVersion != workspaceSkillsSchemaVersionV1 {
		return fmt.Errorf(
			"%w: Skill definition schema version must be %q",
			engine.ErrInvalidWorkspace,
			workspaceSkillsSchemaVersionV1,
		)
	}
	if len(value.Dependencies) != 0 {
		return fmt.Errorf(
			"%w: Workspace Skills cannot declare portable dependencies",
			engine.ErrInvalidWorkspace,
		)
	}
	body, err := engine.DecodeDefinitionBody[skillDefinition](value.Body)
	if err != nil {
		return err
	}
	if err := validateSkillBody(body); err != nil {
		return err
	}
	if string(value.LogicalName) != body.Name {
		return fmt.Errorf(
			"%w: Skill logical name does not match body.name",
			engine.ErrInvalidWorkspace,
		)
	}
	if value.DisplayName != body.DisplayName {
		return fmt.Errorf(
			"%w: Skill display name does not match body.displayName",
			engine.ErrInvalidWorkspace,
		)
	}
	if value.Description != body.Description {
		return fmt.Errorf(
			"%w: Skill description does not match body.description",
			engine.ErrInvalidWorkspace,
		)
	}
	if value.Labels[skillInsertLabelKey] != body.Insert {
		return fmt.Errorf(
			"%w: Skill insert label does not match body.insert",
			engine.ErrInvalidWorkspace,
		)
	}
	return nil
}

func validateSkillBody(value skillDefinition) error {
	document := agentSkillDocument(value)
	if err := agentskills.ValidateSkillDocument(document); err != nil {
		return fmt.Errorf(
			"%w: invalid Workspace Skill document: %w",
			engine.ErrInvalidWorkspace,
			err,
		)
	}
	return nil
}

func agentSkillDocument(
	value skillDefinition,
) agentskillsSpec.SkillDocument {
	arguments := make(
		[]agentskillsSpec.SkillArgument,
		0,
		len(value.Arguments),
	)
	for _, argument := range value.Arguments {
		arguments = append(arguments, agentskillsSpec.SkillArgument{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	return agentskillsSpec.SkillDocument{
		Name:           value.Name,
		DisplayName:    value.DisplayName,
		Description:    value.Description,
		Insert:         agentskillsSpec.SkillInsert(value.Insert),
		Arguments:      arguments,
		Tags:           append([]string(nil), value.Tags...),
		MarkdownBody:   value.MarkdownBody,
		RawFrontmatter: value.RawFrontmatter,
	}
}
