package artifact

import (
	"fmt"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

func ValidateDefinition(value providerapi.Definition) error {
	_, err := DocumentFromDefinition(value)
	return err
}

// DocumentFromDefinition validates the canonical Artifact definition and
// returns the corresponding agentskills-go document. Consumers must use this
// projection instead of decoding the Skill body independently.
//
// Raw SKILL.md parsing remains exclusively in DecodeSkillDocument, which
// delegates to agentskills-go.ParseSkillDocument.
func DocumentFromDefinition(
	value providerapi.Definition,
) (document.SkillDocument, error) {
	if value.Kind != artifactbuiltin.AgentSkillArtifactKind {
		return document.SkillDocument{}, fmt.Errorf(
			"%w: Skill definition kind must be %q",
			basespec.ErrInvalid,
			artifactbuiltin.AgentSkillArtifactKind,
		)
	}
	if value.SchemaID != artifactbuiltin.AgentSkillSchemaID {
		return document.SkillDocument{}, fmt.Errorf(
			"%w: Skill definition schema must be %q",
			basespec.ErrInvalid,
			artifactbuiltin.AgentSkillSchemaID,
		)
	}
	if value.SchemaVersion != artifactbuiltin.AgentSkillSchemaVersion {
		return document.SkillDocument{}, fmt.Errorf(
			"%w: Skill definition schema version must be %q",
			basespec.ErrInvalid,
			artifactbuiltin.AgentSkillSchemaVersion,
		)
	}
	if len(value.Dependencies) != 0 {
		return document.SkillDocument{}, fmt.Errorf(
			"%w: Agent Skills cannot declare generic portable dependencies",
			basespec.ErrInvalid,
		)
	}
	body, err := providerapi.DecodeBody[Body](value.Body)
	if err != nil {
		return document.SkillDocument{}, err
	}
	doc := toDocument(body)
	if err := document.ValidateSkillDocument(doc); err != nil {
		return document.SkillDocument{}, fmt.Errorf(
			"%w: invalid Agent Skill document: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if string(value.LogicalName) != body.Name {
		return document.SkillDocument{}, fmt.Errorf(
			"%w: Skill logical name does not match body.name",
			basespec.ErrInvalid,
		)
	}
	if value.DisplayName != body.DisplayName {
		return document.SkillDocument{}, fmt.Errorf(
			"%w: Skill display name does not match body.displayName",
			basespec.ErrInvalid,
		)
	}
	if value.Description != body.Description {
		return document.SkillDocument{}, fmt.Errorf(
			"%w: Skill description does not match body.description",
			basespec.ErrInvalid,
		)
	}
	if value.Labels[artifactbuiltin.AgentSkillInsertLabelKey] != body.Insert {
		return document.SkillDocument{}, fmt.Errorf(
			"%w: Skill insert label does not match body.insert",
			basespec.ErrInvalid,
		)
	}
	return doc, nil
}

func toDocument(value Body) document.SkillDocument {
	arguments := make(
		[]document.SkillArgument,
		0,
		len(value.Arguments),
	)
	for _, argument := range value.Arguments {
		arguments = append(arguments, document.SkillArgument{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	return document.SkillDocument{
		Name:           value.Name,
		DisplayName:    value.DisplayName,
		Description:    value.Description,
		Insert:         document.SkillInsert(value.Insert),
		Arguments:      arguments,
		Tags:           append([]string(nil), value.Tags...),
		MarkdownBody:   value.MarkdownBody,
		RawFrontmatter: value.RawFrontmatter,
	}
}
