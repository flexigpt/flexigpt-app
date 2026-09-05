package artifact

import (
	"context"
	"path"
	"strings"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

type Decoder struct{}

var _ providerapi.Decoder = (*Decoder)(nil)

func NewDecoder() (*Decoder, error) {
	return &Decoder{}, nil
}

func (*Decoder) ID() basespec.DecoderID {
	return artifactbuiltin.AgentSkillDecoderID
}

func (*Decoder) Revision() string {
	return artifactbuiltin.AgentSkillSchemaVersion
}

func (d *Decoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	if candidate.RequestsDecoder(artifactbuiltin.AgentSkillDecoderID) && basespec.Locator(path.Base(
		string(candidate.Locator),
	)) == artifactbuiltin.AgentSkillDefinitionFileName {
		return providerapi.RecognitionPreferred
	}
	return providerapi.RecognitionNone
}

func (d *Decoder) Decode(
	_ context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []providerapi.Diagnostic) {
	if !candidate.RequestsDecoder(artifactbuiltin.AgentSkillDecoderID) ||
		basespec.Locator(path.Base(string(candidate.Locator))) != artifactbuiltin.AgentSkillDefinitionFileName {
		return nil, nil
	}

	parent := path.Dir(string(candidate.Locator))
	if parent == "." || parent == "/" || parent == "" {
		return nil, nil
	}
	expectedName := path.Base(parent)
	value, warnings, err := DecodeSkillDocument(
		candidate.Content,
		expectedName,
	)
	if err != nil {
		return nil, errorDiagnostics(candidate.Locator, err)
	}
	for index := range warnings {
		warnings[index].Location = &providerapi.DiagnosticLocation{
			Locator: candidate.Locator,
		}
	}
	return []providerapi.Decoded{{Definition: value}}, warnings
}

// DecodeSkillDocument is the shared SKILL.md parse-and-definition path used
// by discovery and managed Skill publication. It deliberately delegates
// parsing and semantic validation to agentskills-go.
func DecodeSkillDocument(
	content []byte,
	expectedName string,
) (providerapi.Definition, []providerapi.Diagnostic, error) {
	doc, warnings, err := document.ParseSkillDocument(
		content,
		document.ParseSkillDocumentOptions{
			ExpectedName: expectedName,
		},
	)
	if err != nil {
		return providerapi.Definition{}, nil, err
	}

	value, err := definitionForDocument(doc)
	if err != nil {
		return providerapi.Definition{}, nil, err
	}
	canonical, err := providerapi.Canonicalize(value)
	if err != nil {
		return providerapi.Definition{}, nil, err
	}
	return canonical, warningDiagnostics("", warnings), nil
}

func definitionForDocument(
	doc document.SkillDocument,
) (providerapi.Definition, error) {
	arguments := make([]Argument, 0, len(doc.Arguments))
	for _, argument := range doc.Arguments {
		arguments = append(arguments, Argument{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	raw, err := providerapi.EncodeBody(Body{
		Name:           doc.Name,
		DisplayName:    doc.DisplayName,
		Description:    doc.Description,
		Insert:         string(doc.Insert),
		Arguments:      arguments,
		Tags:           append([]string(nil), doc.Tags...),
		MarkdownBody:   doc.MarkdownBody,
		RawFrontmatter: doc.RawFrontmatter,
	})
	if err != nil {
		return providerapi.Definition{}, err
	}
	return providerapi.Definition{
		Kind:          artifactbuiltin.AgentSkillArtifactKind,
		SchemaID:      artifactbuiltin.AgentSkillSchemaID,
		SchemaVersion: artifactbuiltin.AgentSkillSchemaVersion,
		LogicalName:   basespec.LogicalName(doc.Name),
		DisplayName:   doc.DisplayName,
		Description:   doc.Description,
		Labels: map[string]string{
			artifactbuiltin.AgentSkillInsertLabelKey: string(doc.Insert),
		},
		Body: raw,
	}, nil
}

func errorDiagnostics(
	locator basespec.Locator,
	err error,
) []providerapi.Diagnostic {
	return []providerapi.Diagnostic{{
		Severity: providerapi.DiagnosticError,
		Code:     "agent.skill.invalid",
		Message:  providerapi.BoundedDiagnosticMessage(err.Error()),
		Location: &providerapi.DiagnosticLocation{
			Locator: locator,
		},
	}}
}

func warningDiagnostics(
	locator basespec.Locator,
	warnings []string,
) []providerapi.Diagnostic {
	output := make([]providerapi.Diagnostic, 0, len(warnings))
	for _, warning := range warnings {
		if len(output) == providerapi.MaxDiagnostics {
			break
		}
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		output = append(output, providerapi.Diagnostic{
			Severity: providerapi.DiagnosticWarning,
			Code:     "agent.skill.parse-warning",
			Message:  providerapi.BoundedDiagnosticMessage(warning),
			Location: &providerapi.DiagnosticLocation{
				Locator: locator,
			},
		})
	}
	return output
}
