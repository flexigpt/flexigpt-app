package source

import (
	"fmt"
	"strings"

	"github.com/flexigpt/agentskills-go/document"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	skillv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/contract/v1"
)

// ManagedSkillDocument is a source-backed editable Skill document. It is
// produced only after the caller has selected a managed Skill Artifact.
type ManagedSkillDocument struct {
	Artifact artifactModel.Artifact `json:"artifact"`
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
			spec.ErrInvalid,
			err,
		)
	}
	return doc, warningDiagnostics(warnings), nil
}

// DecodeSkillDocument parses SKILL.md and projects its source-derived
// identity into the independent portable skillv1 declaration contract.
func DecodeSkillDocument(
	documents support.Documents,
	content []byte,
	expectedName string,
) (definitionModel.Definition, []diagnostic.Diagnostic, error) {
	if err := documents.Validate(); err != nil {
		return definitionModel.Definition{}, nil, err
	}

	doc, warnings, err := ParseSkillDocument(content, expectedName)
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	value, err := definitionForSkillDocument(documents, doc)
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	canonical, err := definitionModel.Canonicalize(value)
	if err != nil {
		return definitionModel.Definition{}, nil, err
	}
	return canonical, warnings, nil
}

// DefinitionForSkillDeclaration projects one standalone canonical skillv1
// declaration into the generic Artifact Store Definition value.
func DefinitionForSkillDeclaration(
	doc skillv1.SkillDocument,
) (definitionModel.Definition, error) {
	body, err := doc.CanonicalJSON()
	if err != nil {
		return definitionModel.Definition{}, err
	}
	displayName := doc.DisplayName
	if displayName == "" {
		displayName = doc.Name
	}
	value := definitionModel.Definition{
		Kind:          SkillArtifactKind,
		SchemaID:      SkillSchemaID,
		SchemaVersion: SkillSchemaVersion,
		LogicalName:   spec.LogicalName(doc.Name),
		DisplayName:   displayName,
		Description:   doc.Description,
		Labels:        declaration.CloneStringMap(doc.Labels),
		Body:          body,
		Dependencies:  nil,
	}
	return definitionModel.Canonicalize(value)
}

// SkillDeclarationFromDefinition validates and decodes the canonical skillv1
// declaration stored inside one generic Definition.
func SkillDeclarationFromDefinition(
	value definitionModel.Definition,
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
	value definitionModel.Definition,
) error {
	if value.Kind != SkillArtifactKind {
		return fmt.Errorf(
			"%w: Skill Definition kind must be %q",
			spec.ErrInvalid,
			SkillArtifactKind,
		)
	}
	if value.SchemaID != SkillSchemaID {
		return fmt.Errorf(
			"%w: Skill Definition schema must be %q",
			spec.ErrInvalid,
			SkillSchemaID,
		)
	}
	if value.SchemaVersion != SkillSchemaVersion {
		return fmt.Errorf(
			"%w: Skill Definition schema version must be %q",
			spec.ErrInvalid,
			SkillSchemaVersion,
		)
	}
	if len(value.Dependencies) != 0 {
		return fmt.Errorf(
			"%w: Skill Definition dependencies belong to the resolver graph",
			spec.ErrInvalid,
		)
	}

	doc, err := skillv1.DecodeAdmittedSkillJSON(value.Body)
	if err != nil {
		return err
	}
	if doc.Name != string(value.LogicalName) {
		return fmt.Errorf(
			"%w: Skill Definition logical name does not match declaration name",
			spec.ErrInvalid,
		)
	}
	if doc.Description != value.Description {
		return fmt.Errorf(
			"%w: Skill Definition description does not match declaration description",
			spec.ErrInvalid,
		)
	}
	return nil
}

func definitionForSkillDocument(
	documents support.Documents,
	doc document.SkillDocument,
) (definitionModel.Definition, error) {
	documentFile := SkillDefinitionFileName(documents)
	sourceLocator := declaration.ScalarLocator(
		"./" + string(documentFile),
	)
	insert := portableSkillInsert(doc.Insert)
	decl := skillv1.SkillDocument{
		Type:        declaration.TypeSkill,
		Name:        doc.Name,
		DisplayName: doc.DisplayName,
		Description: doc.Description,
		Locator:     &sourceLocator,

		Insert: insert,
	}
	body, err := decl.CanonicalJSON()
	if err != nil {
		return definitionModel.Definition{}, err
	}

	return definitionModel.Definition{
		Kind:          SkillArtifactKind,
		SchemaID:      SkillSchemaID,
		SchemaVersion: SkillSchemaVersion,
		LogicalName:   spec.LogicalName(doc.Name),
		DisplayName:   doc.DisplayName,
		Description:   doc.Description,
		Labels: map[string]string{
			InsertLabelKey: string(doc.Insert),
		},
		Body: body,
	}, nil
}

func portableSkillInsert(
	value document.SkillInsert,
) declaration.InsertTarget {
	normalized, supported := document.NormalizeSkillInsert(value)
	if !supported {
		return ""
	}
	target := declaration.InsertTarget(normalized)
	if target.Validate() != nil {
		return ""
	}
	return target
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
