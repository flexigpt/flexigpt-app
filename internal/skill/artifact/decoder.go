package artifact

import (
	"context"
	"path"
	"strings"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
)

type Decoder struct{}

func NewDecoder() (*Decoder, error) {
	return &Decoder{}, nil
}

func (*Decoder) ID() basespec.DecoderID {
	return DecoderID
}

func (*Decoder) Revision() string {
	return SchemaVersion
}

func (d *Decoder) Recognize(
	_ context.Context,
	candidate discovery.Candidate,
) discovery.Recognition {
	if candidate.RequestsDecoder(DecoderID) && basespec.Locator(path.Base(
		string(candidate.Locator))) == artifactbuiltin.AgentSkillDefinitionFileName {
		return discovery.RecognitionPreferred
	}
	return discovery.RecognitionNone
}

func (d *Decoder) Decode(
	_ context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []diagnostic.Diagnostic) {
	if !candidate.RequestsDecoder(DecoderID) ||
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
		warnings[index].Location = &diagnostic.DiagnosticLocation{
			Locator: candidate.Locator,
		}
	}
	return []discovery.Decoded{{Definition: value}}, warnings
}

// DecodeSkillDocument is the shared SKILL.md parse-and-definition path used
// by discovery and managed Skill publication. It deliberately delegates
// parsing and semantic validation to agentskills-go.
func DecodeSkillDocument(
	content []byte,
	expectedName string,
) (definition.Definition, []diagnostic.Diagnostic, error) {
	document, warnings, err := agentskills.ParseSkillDocument(
		content,
		agentskillsSpec.ParseSkillDocumentOptions{
			ExpectedName: expectedName,
		},
	)
	if err != nil {
		return definition.Definition{}, nil, err
	}

	value, err := definitionForDocument(document)
	if err != nil {
		return definition.Definition{}, nil, err
	}
	canonical, err := definition.Canonicalize(value)
	if err != nil {
		return definition.Definition{}, nil, err
	}
	return canonical, warningDiagnostics("", warnings), nil
}

func definitionForDocument(
	document agentskillsSpec.SkillDocument,
) (definition.Definition, error) {
	arguments := make([]Argument, 0, len(document.Arguments))
	for _, argument := range document.Arguments {
		arguments = append(arguments, Argument{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	raw, err := definition.EncodeBody(Body{
		Name:           document.Name,
		DisplayName:    document.DisplayName,
		Description:    document.Description,
		Insert:         string(document.Insert),
		Arguments:      arguments,
		Tags:           append([]string(nil), document.Tags...),
		MarkdownBody:   document.MarkdownBody,
		RawFrontmatter: document.RawFrontmatter,
	})
	if err != nil {
		return definition.Definition{}, err
	}
	return definition.Definition{
		Kind:          Kind,
		SchemaID:      SchemaID,
		SchemaVersion: SchemaVersion,
		LogicalName:   basespec.LogicalName(document.Name),
		DisplayName:   document.DisplayName,
		Description:   document.Description,
		Labels:        map[string]string{InsertLabelKey: string(document.Insert)},
		Body:          raw,
	}, nil
}

func errorDiagnostics(
	locator basespec.Locator,
	err error,
) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.DiagnosticError,
		Code:     "agent.skill.invalid",
		Message:  diagnostic.BoundedDiagnosticMessage(err.Error()),
		Location: &diagnostic.DiagnosticLocation{
			Locator: locator,
		},
	}}
}

func warningDiagnostics(
	locator basespec.Locator,
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
			Severity: diagnostic.DiagnosticWarning,
			Code:     "agent.skill.parse-warning",
			Message:  diagnostic.BoundedDiagnosticMessage(warning),
			Location: &diagnostic.DiagnosticLocation{
				Locator: locator,
			},
		})
	}
	return output
}
