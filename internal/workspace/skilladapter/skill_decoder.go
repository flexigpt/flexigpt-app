package skilladapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

type SkillDecoder struct {
	conventions *ConventionRegistry
}

func NewSkillDecoder() *SkillDecoder {
	registry, err := NewConventionRegistry()
	if err != nil {
		panic(err)
	}
	return &SkillDecoder{conventions: registry}
}

func NewSkillDecoderWithConventions(
	registry *ConventionRegistry,
) (*SkillDecoder, error) {
	if registry == nil || len(registry.Roots()) == 0 {
		return nil, fmt.Errorf(
			"%w: Workspace Skill convention registry is empty",
			engine.ErrInvalidWorkspace,
		)
	}
	return &SkillDecoder{conventions: registry}, nil
}

func DiscoveryProfile() engine.DiscoveryProfile {
	registry, err := NewConventionRegistry()
	if err != nil {
		panic(err)
	}
	return registry.DiscoveryProfile()
}

func DiscoveryProfileWithConventions(
	registry *ConventionRegistry,
) engine.DiscoveryProfile {
	return registry.DiscoveryProfile()
}

func ArtifactSupport() engine.ArtifactSupport {
	return artifactSupport
}

func (*SkillDecoder) ID() artifactstore.DecoderID {
	return skillDecoderID
}

func (d *SkillDecoder) Recognize(
	_ context.Context,
	candidate discovery.Candidate,
) discovery.Recognition {
	if _, supported := d.conventions.Match(candidate.Locator); !supported {
		return discovery.RecognitionNone
	}
	return discovery.RecognitionPreferred
}

func (d *SkillDecoder) Decode(
	_ context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []artifactstore.Diagnostic) {
	expectedName, supported := d.conventions.ExpectedName(candidate.Locator)
	if !supported {
		return nil, nil
	}
	document, warnings, err := decodeSkillMarkdown(candidate.Locator, candidate.Content)
	if err != nil {
		return nil, engine.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			engine.DiagnosticCodeSkillInvalid,
			err.Error(),
		)
	}
	if document.Name != expectedName {
		return nil, engine.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			engine.DiagnosticCodeSkillInvalid,
			fmt.Sprintf(
				"frontmatter.name %q must match containing directory %q",
				document.Name,
				expectedName,
			),
		)
	}

	raw, err := json.Marshal(document)
	if err != nil {
		return nil, engine.WorkspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}
	raw, err = jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, engine.WorkspaceArtifactErrorDiagnostics(candidate.Locator, err)
	}

	labels := map[string]string{
		skillInsertLabelKey: document.Insert,
	}
	value := definition.Definition{
		Kind:          skillKind,
		SchemaID:      skillSchemaID,
		SchemaVersion: workspaceSkillsSchemaVersionV1,
		LogicalName:   artifactstore.LogicalName(document.Name),
		DisplayName:   document.DisplayName,
		Description:   document.Description,
		Labels:        labels,
		Body:          raw,
	}
	if err := ValidateSkillDefinition(value); err != nil {
		return nil, engine.WorkspaceArtifactDiagnostics(
			candidate.Locator,
			engine.DiagnosticCodeSkillInvalid,
			err.Error(),
		)
	}
	return []discovery.Decoded{{Definition: value}},
		skillWarningDiagnostics(
			candidate.Locator,
			warnings,
		)
}

func decodeSkillMarkdown(
	locator artifactstore.Locator,
	content []byte,
) (skillDefinition, []string, error) {
	document, warnings, err := agentskills.ParseSkillDocument(
		content,
		agentskillsSpec.ParseSkillDocumentOptions{},
	)
	if err != nil {
		return skillDefinition{}, nil, err
	}

	arguments := make(
		[]skillArgumentDefinition,
		0,
		len(document.Arguments),
	)
	for _, argument := range document.Arguments {
		arguments = append(arguments, skillArgumentDefinition{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	return skillDefinition{
		Name:           document.Name,
		DisplayName:    document.DisplayName,
		Description:    document.Description,
		Insert:         string(document.Insert),
		Arguments:      arguments,
		Tags:           append([]string(nil), document.Tags...),
		MarkdownBody:   document.MarkdownBody,
		RawFrontmatter: document.RawFrontmatter,
	}, warnings, nil
}

func skillWarningDiagnostics(
	locator artifactstore.Locator,
	warnings []string,
) []artifactstore.Diagnostic {
	output := make([]artifactstore.Diagnostic, 0, len(warnings))
	for _, warning := range warnings {
		message := strings.TrimSpace(warning)
		if message == "" {
			continue
		}
		for len(message) > artifactstore.MaxDiagnosticMessageBytes {
			_, size := utf8.DecodeLastRuneInString(message)
			message = message[:len(message)-size]
		}
		output = append(output, artifactstore.Diagnostic{
			Severity: artifactstore.DiagnosticWarning,
			Code:     "workspace.skill.parse-warning",
			Message:  message,
			Location: &artifactstore.DiagnosticLocation{
				Locator: locator,
			},
		})
	}
	return output
}
