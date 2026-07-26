package skillartifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
)

func ValidateDefinition(value definition.Definition) error {
	if value.Kind != Kind {
		return fmt.Errorf(
			"%w: Skill definition kind must be %q",
			artifactstore.ErrInvalid,
			Kind,
		)
	}
	if value.SchemaID != SchemaID {
		return fmt.Errorf(
			"%w: Skill definition schema must be %q",
			artifactstore.ErrInvalid,
			SchemaID,
		)
	}
	if value.SchemaVersion != SchemaVersion {
		return fmt.Errorf(
			"%w: Skill definition schema version must be %q",
			artifactstore.ErrInvalid,
			SchemaVersion,
		)
	}
	if len(value.Dependencies) != 0 {
		return fmt.Errorf(
			"%w: Agent Skills cannot declare generic portable dependencies",
			artifactstore.ErrInvalid,
		)
	}
	body, err := DecodeBody(value.Body)
	if err != nil {
		return err
	}
	if err := agentskills.ValidateSkillDocument(toDocument(body)); err != nil {
		return fmt.Errorf(
			"%w: invalid Agent Skill document: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	if string(value.LogicalName) != body.Name {
		return fmt.Errorf(
			"%w: Skill logical name does not match body.name",
			artifactstore.ErrInvalid,
		)
	}
	if value.DisplayName != body.DisplayName {
		return fmt.Errorf(
			"%w: Skill display name does not match body.displayName",
			artifactstore.ErrInvalid,
		)
	}
	if value.Description != body.Description {
		return fmt.Errorf(
			"%w: Skill description does not match body.description",
			artifactstore.ErrInvalid,
		)
	}
	if value.Labels[InsertLabelKey] != body.Insert {
		return fmt.Errorf(
			"%w: Skill insert label does not match body.insert",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}

func DecodeBody(raw json.RawMessage) (Body, error) {
	var output Body
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return output, fmt.Errorf(
			"%w: decode Agent Skill definition: %w",
			artifactstore.ErrInvalid,
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
			artifactstore.ErrInvalid,
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
