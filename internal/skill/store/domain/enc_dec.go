package domain

import (
	"fmt"
	"strings"

	"github.com/flexigpt/agentskills-go/document"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/skillv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
)

// ManagedSkillDocument is a source-backed editable Skill document. It is
// produced only after the caller has selected a managed Skill Artifact.
type ManagedSkillDocument struct {
	Artifact artifact.Artifact      `json:"artifact"`
	Document document.SkillDocument `json:"document"`
}

// ParseSkillDocument parses one SKILL.md source file. It remains the only
// source-format parser used by the Skill consumer.
func ParseSkillDocument(
	content []byte,
	expectedName string,
) (document.SkillDocument, []diagnostic.Diagnostic, error) {
	doc, warnings, err := document.ParseSkillDocument(
		content,
		document.ParseSkillDocumentOptions{
			ExpectedName: expectedName,
		},
	)
	if err != nil {
		return document.SkillDocument{}, nil, err
	}
	if err := document.ValidateSkillDocument(doc); err != nil {
		return document.SkillDocument{}, nil, fmt.Errorf(
			"%w: invalid Agent Skill document: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return doc, warningDiagnostics(warnings), nil
}

// DecodeSkillDocument parses SKILL.md and projects its source-derived
// identity into the independent portable skillv1 declaration contract.
func DecodeSkillDocument(
	content []byte,
	expectedName string,
) (definition.Definition, []diagnostic.Diagnostic, error) {
	doc, warnings, err := ParseSkillDocument(content, expectedName)
	if err != nil {
		return definition.Definition{}, nil, err
	}
	value, err := definitionForSkillDocument(doc)
	if err != nil {
		return definition.Definition{}, nil, err
	}
	canonical, err := definition.Canonicalize(value)
	if err != nil {
		return definition.Definition{}, nil, err
	}
	return canonical, warnings, nil
}

// DefinitionForSkillDeclaration projects one standalone canonical skillv1
// declaration into the generic Artifact Store Definition value.
func DefinitionForSkillDeclaration(
	declaration skillv1.SkillDocument,
) (definition.Definition, error) {
	if err := declaration.Validate(); err != nil {
		return definition.Definition{}, err
	}
	body, err := declaration.CanonicalJSON()
	if err != nil {
		return definition.Definition{}, err
	}
	value := definition.Definition{
		Kind:          SkillArtifactKind,
		SchemaID:      SkillSchemaID,
		SchemaVersion: SkillSchemaVersion,
		LogicalName:   basespec.LogicalName(declaration.Name),
		DisplayName:   declaration.Name,
		Description:   declaration.Description,
		Body:          body,
		Dependencies:  nil,
	}
	return definition.Canonicalize(value)
}

// SkillDeclarationFromDefinition validates and decodes the canonical skillv1
// declaration stored inside one generic Definition.
func SkillDeclarationFromDefinition(
	value definition.Definition,
) (skillv1.SkillDocument, error) {
	if err := ValidateDefinition(value); err != nil {
		return skillv1.SkillDocument{}, err
	}
	return skillv1.DecodeSkillJSON(value.Body)
}

// ValidateDefinition verifies the generic Definition projection without
// reopening its source package. Runtime source loading remains source-backed
// through Artifact Store Resource resolution.
func ValidateDefinition(
	value definition.Definition,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Kind != SkillArtifactKind {
		return fmt.Errorf(
			"%w: Skill Definition kind must be %q",
			basespec.ErrInvalid,
			SkillArtifactKind,
		)
	}
	if value.SchemaID != SkillSchemaID {
		return fmt.Errorf(
			"%w: Skill Definition schema must be %q",
			basespec.ErrInvalid,
			SkillSchemaID,
		)
	}
	if value.SchemaVersion != SkillSchemaVersion {
		return fmt.Errorf(
			"%w: Skill Definition schema version must be %q",
			basespec.ErrInvalid,
			SkillSchemaVersion,
		)
	}
	if len(value.Dependencies) != 0 {
		return fmt.Errorf(
			"%w: Skill Definition dependencies belong to the resolver graph",
			basespec.ErrInvalid,
		)
	}

	declaration, err := skillv1.DecodeSkillJSON(value.Body)
	if err != nil {
		return err
	}
	if declaration.Name != string(value.LogicalName) {
		return fmt.Errorf(
			"%w: Skill Definition logical name does not match declaration name",
			basespec.ErrInvalid,
		)
	}
	if declaration.Description != value.Description {
		return fmt.Errorf(
			"%w: Skill Definition description does not match declaration description",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func definitionForSkillDocument(
	doc document.SkillDocument,
) (definition.Definition, error) {
	declaration := skillv1.SkillDocument{
		APIVersion:  SkillSchemaVersion,
		Type:        artifactcontract.TypeSkill,
		Name:        doc.Name,
		Description: doc.Description,
	}
	body, err := declaration.CanonicalJSON()
	if err != nil {
		return definition.Definition{}, err
	}

	return definition.Definition{
		Kind:          SkillArtifactKind,
		SchemaID:      SkillSchemaID,
		SchemaVersion: SkillSchemaVersion,
		LogicalName:   basespec.LogicalName(doc.Name),
		DisplayName:   doc.DisplayName,
		Description:   doc.Description,
		Labels: map[string]string{
			InsertLabelKey: string(doc.Insert),
		},
		Body: body,
	}, nil
}

func warningDiagnostics(
	warnings []string,
) []diagnostic.Diagnostic {
	output := make([]diagnostic.Diagnostic, 0, len(warnings))
	for _, warning := range warnings {
		if len(output) == diagnostic.MaxDiagnostics {
			break
		}
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		output = append(output, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityWarning,
			Code:     "agent.skill.parse-warning",
			Message:  diagnostic.BoundedMessage(warning),
		})
	}
	return output
}
