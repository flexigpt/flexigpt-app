package skillartifact

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Decoder struct {
	policy LocatorPolicy
}

func NewDecoder(policy LocatorPolicy) (*Decoder, error) {
	if policy == nil {
		return nil, fmt.Errorf(
			"%w: agent Skill locator policy is nil",
			basespec.ErrInvalid,
		)
	}
	return &Decoder{policy: policy}, nil
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
	if _, supported := d.policy.ExpectedName(candidate.Locator); supported {
		return discovery.RecognitionPreferred
	}
	if candidate.RequestsDecoder(DecoderID) &&
		strings.EqualFold(
			path.Base(string(candidate.Locator)),
			DefinitionFileName,
		) {
		return discovery.RecognitionPreferred
	}
	return discovery.RecognitionNone
}

func (d *Decoder) Decode(
	_ context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []diagnostic.Diagnostic) {
	expectedName, supported := d.policy.ExpectedName(candidate.Locator)
	if !supported {
		if !candidate.RequestsDecoder(DecoderID) ||
			!strings.EqualFold(
				path.Base(string(candidate.Locator)),
				DefinitionFileName,
			) {
			return nil, nil
		}
		parent := path.Dir(string(candidate.Locator))
		if parent == "." || parent == "/" || parent == "" {
			return nil, nil
		}
		expectedName = path.Base(parent)
	}

	document, warnings, err := agentskills.ParseSkillDocument(
		candidate.Content,
		agentskillsSpec.ParseSkillDocumentOptions{},
	)
	if err != nil {
		return nil, errorDiagnostics(candidate.Locator, err)
	}
	if document.Name != expectedName {
		return nil, errorDiagnostics(
			candidate.Locator,
			fmt.Errorf(
				"frontmatter.name %q must match containing directory %q",
				document.Name,
				expectedName,
			),
		)
	}

	arguments := make([]Argument, 0, len(document.Arguments))
	for _, argument := range document.Arguments {
		arguments = append(arguments, Argument{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	body := Body{
		Name:           document.Name,
		DisplayName:    document.DisplayName,
		Description:    document.Description,
		Insert:         string(document.Insert),
		Arguments:      arguments,
		Tags:           append([]string(nil), document.Tags...),
		MarkdownBody:   document.MarkdownBody,
		RawFrontmatter: document.RawFrontmatter,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, errorDiagnostics(candidate.Locator, err)
	}
	raw, err = jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, errorDiagnostics(candidate.Locator, err)
	}
	value := definition.Definition{
		Kind:          Kind,
		SchemaID:      SchemaID,
		SchemaVersion: SchemaVersion,
		LogicalName:   basespec.LogicalName(document.Name),
		DisplayName:   document.DisplayName,
		Description:   document.Description,
		Labels: map[string]string{
			InsertLabelKey: string(document.Insert),
		},
		Body: raw,
	}
	if err := ValidateDefinition(value); err != nil {
		return nil, errorDiagnostics(candidate.Locator, err)
	}
	return []discovery.Decoded{{Definition: value}},
		warningDiagnostics(candidate.Locator, warnings)
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
