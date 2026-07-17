package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

const (
	projectedSkillInsertInstructions = "instructions"
	projectedSkillInsertUserMessage  = "user-message"
)

var projectedSkillNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type workspaceDefinitionProjector struct{}

func (workspaceDefinitionProjector) Kind() artifactstoreSpec.ArtifactKind {
	return KindWorkspaceDefinition
}

func (workspaceDefinitionProjector) Project(
	_ context.Context,
	input ProjectionInput,
) (any, []artifactstoreSpec.Diagnostic) {
	projected, err := parseWorkspaceDefinition(input.Definition)
	if err != nil {
		return nil, projectorDiagnostics(input, err)
	}
	projected.RecordID = input.Record.RecordID
	return projected, nil
}

type skillProjector struct{}

func (skillProjector) Kind() artifactstoreSpec.ArtifactKind {
	return KindSkillDefinition
}

func (skillProjector) Project(
	_ context.Context,
	input ProjectionInput,
) (any, []artifactstoreSpec.Diagnostic) {
	projected, err := parseSkillDefinition(input.Definition)
	if err != nil {
		return nil, projectorDiagnostics(input, err)
	}
	projected.RecordID = input.Record.RecordID
	projected.DefinitionDigest = input.Definition.Digest
	projected.SourceID = input.Record.SourceID
	projected.Locator = input.Record.Locator
	return projected, nil
}

type documentProjector struct {
	kind artifactstoreSpec.ArtifactKind
}

func (p documentProjector) Kind() artifactstoreSpec.ArtifactKind {
	return p.kind
}

func (p documentProjector) Project(
	_ context.Context,
	input ProjectionInput,
) (any, []artifactstoreSpec.Diagnostic) {
	projected, err := parseDocumentDefinition(input.Definition)
	if err != nil {
		return nil, projectorDiagnostics(input, err)
	}
	projected.RecordID = input.Record.RecordID
	projected.DefinitionDigest = input.Definition.Digest
	projected.Kind = input.Record.Kind
	projected.SourceID = input.Record.SourceID
	projected.Locator = input.Record.Locator
	return projected, nil
}

func defaultProjectors() map[artifactstoreSpec.ArtifactKind]ResourceProjector {
	return map[artifactstoreSpec.ArtifactKind]ResourceProjector{
		KindWorkspaceDefinition: workspaceDefinitionProjector{},
		KindSkillDefinition:     skillProjector{},
		KindInstructionDocument: documentProjector{kind: KindInstructionDocument},
		KindContextDocument:     documentProjector{kind: KindContextDocument},
	}
}

func validateWorkspaceCanonicalDefinition(
	definition artifactstoreSpec.CanonicalDefinition,
) error {
	switch definition.Kind {
	case KindWorkspaceDefinition:
		_, err := parseWorkspaceDefinition(definition)
		return err
	case KindSkillDefinition:
		_, err := parseSkillDefinition(definition)
		return err
	case KindInstructionDocument, KindContextDocument:
		_, err := parseDocumentDefinition(definition)
		return err
	default:
		return nil
	}
}

func parseWorkspaceDefinition(
	definition artifactstoreSpec.CanonicalDefinition,
) (ProjectedWorkspaceDefinition, error) {
	var document struct {
		Discovery DiscoveryPreferences `json:"discovery"`
	}
	if err := decodeStrictJSONObject(
		definition.DefinitionJSON,
		&document,
		false,
	); err != nil {
		return ProjectedWorkspaceDefinition{}, fmt.Errorf(
			"decode workspace definition: %w",
			err,
		)
	}
	if err := validateDiscoveryPreferences(document.Discovery); err != nil {
		return ProjectedWorkspaceDefinition{}, fmt.Errorf(
			"workspace discovery preferences: %w",
			err,
		)
	}
	return ProjectedWorkspaceDefinition{
		Discovery:  document.Discovery,
		Definition: append(json.RawMessage(nil), definition.DefinitionJSON...),
	}, nil
}

func parseSkillDefinition(
	definition artifactstoreSpec.CanonicalDefinition,
) (ProjectedSkill, error) {
	var document struct {
		Markdown    string          `json:"markdown"`
		Frontmatter json.RawMessage `json:"frontmatter"`
	}
	if err := decodeStrictJSONObject(
		definition.DefinitionJSON,
		&document,
		true,
	); err != nil {
		return ProjectedSkill{}, fmt.Errorf("decode skill definition: %w", err)
	}
	if strings.TrimSpace(document.Markdown) == "" {
		return ProjectedSkill{}, errors.New("skill Markdown is empty")
	}
	if len(document.Frontmatter) == 0 {
		return ProjectedSkill{}, errors.New("skill frontmatter is missing")
	}

	fields := map[string]json.RawMessage{}
	if err := decodeStrictJSONObject(document.Frontmatter, &fields, false); err != nil {
		return ProjectedSkill{}, fmt.Errorf("decode skill frontmatter: %w", err)
	}
	name, err := requiredStringField(fields, "name")
	if err != nil {
		return ProjectedSkill{}, err
	}
	if !projectedSkillNameRE.MatchString(name) {
		return ProjectedSkill{}, errors.New(
			"skill name must contain lowercase letters, numbers, and hyphens and be at most 64 characters",
		)
	}
	if artifactstoreSpec.LogicalName(name) != definition.LogicalName {
		return ProjectedSkill{}, fmt.Errorf(
			"skill frontmatter name %q does not match logical name %q",
			name,
			definition.LogicalName,
		)
	}

	description, err := optionalStringField(fields, "description")
	if err != nil {
		return ProjectedSkill{}, err
	}
	displayName, err := optionalStringField(fields, "displayName")
	if err != nil {
		return ProjectedSkill{}, err
	}
	insert, err := optionalStringField(fields, "insert")
	if err != nil {
		return ProjectedSkill{}, err
	}
	if insert == "" {
		insert = projectedSkillInsertInstructions
	}
	switch insert {
	case projectedSkillInsertInstructions, projectedSkillInsertUserMessage:
	default:
		return ProjectedSkill{}, fmt.Errorf("unsupported skill insert %q", insert)
	}

	var arguments []ProjectedSkillArgument
	if rawArguments, exists := fields["arguments"]; exists {
		if err := json.Unmarshal(rawArguments, &arguments); err != nil {
			return ProjectedSkill{}, fmt.Errorf("decode skill arguments: %w", err)
		}
	}
	seenArguments := make(map[string]struct{}, len(arguments))
	for index, argument := range arguments {
		if strings.TrimSpace(argument.Name) == "" ||
			strings.TrimSpace(argument.Name) != argument.Name {
			return ProjectedSkill{}, fmt.Errorf(
				"skill arguments[%d].name must be non-empty and trimmed",
				index,
			)
		}
		if _, duplicate := seenArguments[argument.Name]; duplicate {
			return ProjectedSkill{}, fmt.Errorf(
				"duplicate skill argument %q",
				argument.Name,
			)
		}
		seenArguments[argument.Name] = struct{}{}
	}

	if displayName == "" {
		displayName = definition.DisplayName
	}
	if description == "" {
		description = definition.Description
	}
	return ProjectedSkill{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Insert:      insert,
		Arguments:   append([]ProjectedSkillArgument(nil), arguments...),
		Markdown:    document.Markdown,
		Frontmatter: append(json.RawMessage(nil), document.Frontmatter...),
	}, nil
}

func parseDocumentDefinition(
	definition artifactstoreSpec.CanonicalDefinition,
) (ProjectedDocument, error) {
	var document struct {
		Markdown string `json:"markdown"`
	}
	if err := decodeStrictJSONObject(
		definition.DefinitionJSON,
		&document,
		true,
	); err != nil {
		return ProjectedDocument{}, fmt.Errorf("decode document definition: %w", err)
	}
	if strings.TrimSpace(document.Markdown) == "" {
		return ProjectedDocument{}, errors.New("document Markdown is empty")
	}
	return ProjectedDocument{
		Name:     string(definition.LogicalName),
		Markdown: document.Markdown,
	}, nil
}

func requiredStringField(
	object map[string]json.RawMessage,
	field string,
) (string, error) {
	value, err := optionalStringField(object, field)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return value, nil
}

func optionalStringField(
	object map[string]json.RawMessage,
	field string,
) (string, error) {
	raw, exists := object[field]
	if !exists {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s must be a string", field)
	}
	if strings.TrimSpace(value) != value {
		return "", fmt.Errorf("%s must be trimmed", field)
	}
	return value, nil
}

func projectorDiagnostics(
	input ProjectionInput,
	err error,
) []artifactstoreSpec.Diagnostic {
	diagnostics := workspaceDiagnostics("workspace.projector.invalid", err.Error())
	diagnostics[0].Location = &artifactstoreSpec.DiagnosticLocation{
		Locator:            input.Record.Locator,
		SubresourceLocator: input.Record.SubresourceLocator,
	}
	return diagnostics
}

var (
	_ ResourceProjector = workspaceDefinitionProjector{}
	_ ResourceProjector = skillProjector{}
	_ ResourceProjector = documentProjector{}
)
