package skillartifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
)

func ValidateDefinition(value definition.Definition) error {
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
	value definition.Definition,
) (agentskillsSpec.SkillDocument, error) {
	if value.Kind != Kind {
		return agentskillsSpec.SkillDocument{}, fmt.Errorf(
			"%w: Skill definition kind must be %q",
			basespec.ErrInvalid,
			Kind,
		)
	}
	if value.SchemaID != SchemaID {
		return agentskillsSpec.SkillDocument{}, fmt.Errorf(
			"%w: Skill definition schema must be %q",
			basespec.ErrInvalid,
			SchemaID,
		)
	}
	if value.SchemaVersion != SchemaVersion {
		return agentskillsSpec.SkillDocument{}, fmt.Errorf(
			"%w: Skill definition schema version must be %q",
			basespec.ErrInvalid,
			SchemaVersion,
		)
	}
	if len(value.Dependencies) != 0 {
		return agentskillsSpec.SkillDocument{}, fmt.Errorf(
			"%w: Agent Skills cannot declare generic portable dependencies",
			basespec.ErrInvalid,
		)
	}
	body, err := DecodeBody(value.Body)
	if err != nil {
		return agentskillsSpec.SkillDocument{}, err
	}
	document := toDocument(body)
	if err := agentskills.ValidateSkillDocument(document); err != nil {
		return agentskillsSpec.SkillDocument{}, fmt.Errorf(
			"%w: invalid Agent Skill document: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if string(value.LogicalName) != body.Name {
		return agentskillsSpec.SkillDocument{}, fmt.Errorf(
			"%w: Skill logical name does not match body.name",
			basespec.ErrInvalid,
		)
	}
	if value.DisplayName != body.DisplayName {
		return agentskillsSpec.SkillDocument{}, fmt.Errorf(
			"%w: Skill display name does not match body.displayName",
			basespec.ErrInvalid,
		)
	}
	if value.Description != body.Description {
		return agentskillsSpec.SkillDocument{}, fmt.Errorf(
			"%w: Skill description does not match body.description",
			basespec.ErrInvalid,
		)
	}
	if value.Labels[InsertLabelKey] != body.Insert {
		return agentskillsSpec.SkillDocument{}, fmt.Errorf(
			"%w: Skill insert label does not match body.insert",
			basespec.ErrInvalid,
		)
	}
	return document, nil
}

func DecodeBody(raw json.RawMessage) (Body, error) {
	var output Body
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return output, fmt.Errorf(
			"%w: decode Agent Skill definition: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("definition contains trailing JSON")
		}
		return output, fmt.Errorf(
			"%w: decode Agent Skill definition: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return output, nil
}

func toDocument(value Body) agentskillsSpec.SkillDocument {
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
