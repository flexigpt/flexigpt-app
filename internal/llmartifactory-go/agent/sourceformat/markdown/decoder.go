package markdown

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	textv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const AgentMarkdownDecoderID spec.DecoderID = "agent-markdown"

const markdownMediaType = "text/markdown"

// AgentMarkdownDecoder adapts AGENT.md and *.agent.md files. YAML front
// matter provides Agent declaration fields. The Markdown body becomes a named
// Instruction in Agent.members using the <agent-name>-instructions naming
// convention. Candidate support is supplied by application registration.
type AgentMarkdownDecoder struct {
	interpretations *coreinterpretation.Registry
	candidates      support.Candidates
}

func NewAgentMarkdownDecoder(
	interpretations *coreinterpretation.Registry,
	candidates support.Candidates,
) (*AgentMarkdownDecoder, error) {
	if interpretations == nil {
		return nil, fmt.Errorf(
			"%w: Agent Markdown interpretation registry is nil",
			spec.ErrInvalid,
		)
	}
	if err := candidates.Validate(); err != nil {
		return nil, err
	}
	return &AgentMarkdownDecoder{
		interpretations: interpretations,
		candidates:      candidates.Clone(),
	}, nil
}

func (d *AgentMarkdownDecoder) ID() spec.DecoderID {
	return AgentMarkdownDecoderID
}

func (d *AgentMarkdownDecoder) Revision() string {
	return "artifact-agent-markdown/v1"
}

func (d *AgentMarkdownDecoder) Recognize(
	_ context.Context,
	candidate ingestModel.Candidate,
) ingestModel.Recognition {
	if !d.candidates.Matches(candidate.Locator) {
		return ingestModel.RecognitionNone
	}
	return ingestModel.RecognitionPreferred
}

func (d *AgentMarkdownDecoder) Decode(
	_ context.Context,
	candidate ingestModel.Candidate,
) ([]ingestModel.Decoded, []diagnostic.Diagnostic) {
	document, body, err := decodeAgentMarkdown(
		candidate.Content,
		candidate.Locator,
	)
	if err != nil {
		return nil, agentMarkdownDiagnostics(candidate.Locator, err)
	}

	if strings.TrimSpace(body) != "" {
		instructionName, err := declaration.DeriveNestedLogicalName(
			spec.LogicalName(document.Name),
			"instructions",
		)
		if err != nil {
			return nil, agentMarkdownDiagnostics(
				candidate.Locator,
				err,
			)
		}
		text, err := declaration.NewEntry(
			textv1.TextDocument{
				Type:      textv1.TextType,
				Name:      string(instructionName),
				Insert:    declaration.InsertInstructions,
				Content:   new(body),
				MediaType: markdownMediaType,
			},
		)
		if err != nil {
			return nil, agentMarkdownDiagnostics(candidate.Locator, err)
		}
		member, err := declaration.NewContainedMember(text)
		if err != nil {
			return nil, agentMarkdownDiagnostics(candidate.Locator, err)
		}
		document.Members = append(document.Members, member)
	}

	entry, err := declaration.NewEntry(document)
	if err != nil {
		return nil, agentMarkdownDiagnostics(candidate.Locator, err)
	}

	raw, err := entry.CanonicalJSON()
	if err != nil {
		return nil, agentMarkdownDiagnostics(candidate.Locator, err)
	}
	if d == nil || d.interpretations == nil {
		return nil, agentMarkdownDiagnostics(
			candidate.Locator,
			fmt.Errorf("%w: Agent Markdown interpretation registry is unavailable", spec.ErrClosed),
		)
	}
	admitted, err := d.interpretations.DefinitionsForDocument(
		agentv1.AgentSchemaKey,
		raw,
	)
	if err != nil {
		return nil, agentMarkdownDiagnostics(candidate.Locator, err)
	}

	output := make([]ingestModel.Decoded, 0, len(admitted))
	for _, named := range admitted {
		output = append(output, ingestModel.Decoded{
			SubresourceLocator: named.SubresourceLocator,
			Definition:         named.Definition,
		})
	}
	return output, nil
}

func decodeAgentMarkdown(
	content []byte,
	locator spec.Locator,
) (agentv1.AgentDocument, string, error) {
	markdown, err := normalizeMarkdownOptional(content)
	if err != nil {
		return agentv1.AgentDocument{}, "", err
	}
	frontmatter, body, err := splitAgentMarkdown(markdown)
	if err != nil {
		return agentv1.AgentDocument{}, "", err
	}

	fields := make(map[string]json.RawMessage)
	if strings.TrimSpace(frontmatter) != "" {
		raw, err := yamlutil.CanonicalObjectJSON(
			[]byte(frontmatter),
			spec.MaxDefinitionBytes,
		)
		if err != nil {
			return agentv1.AgentDocument{}, "", err
		}
		if err := jsonutil.DecodeCanonicalObjectBytesInto(
			raw,
			&fields,
			spec.MaxDefinitionBytes,
		); err != nil {
			return agentv1.AgentDocument{}, "", err
		}
	}

	if _, found := fields["type"]; !found {
		fields["type"] = json.RawMessage(`"agent"`)
	}
	if _, found := fields["name"]; !found {
		name, err := declaration.DeriveLogicalName("agent", locator)
		if err != nil {
			return agentv1.AgentDocument{}, "", err
		}
		rawName, err := json.Marshal(string(name))
		if err != nil {
			return agentv1.AgentDocument{}, "", err
		}
		fields["name"] = rawName
	}

	raw, err := jsonutil.MarshalCanonicalObject(
		fields,
		spec.MaxDefinitionBytes,
	)
	if err != nil {
		return agentv1.AgentDocument{}, "", err
	}
	entry, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return agentv1.AgentDocument{}, "", err
	}
	document, err := agentv1.DecodeAgentEntry(entry)
	if err != nil {
		return agentv1.AgentDocument{}, "", err
	}
	return document, body, nil
}

func splitAgentMarkdown(
	value string,
) (frontmatter, body string, returnErr error) {
	lines := strings.Split(value, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return "", value, nil
	}
	for index := 1; index < len(lines); index++ {
		if lines[index] != "---" && lines[index] != "..." {
			continue
		}
		return strings.Join(lines[1:index], "\n"),
			strings.Join(lines[index+1:], "\n"),
			nil
	}
	return "", "", fmt.Errorf(
		"%w: Agent Markdown front matter has no closing delimiter",
		spec.ErrInvalid,
	)
}

func agentMarkdownDiagnostics(
	locator spec.Locator,
	err error,
) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     "artifact.agent-markdown.invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: &diagnostic.Location{
			Locator: locator,
		},
	}}
}

func normalizeMarkdownOptional(
	content []byte,
) (string, error) {
	if !utf8.Valid(content) {
		return "", fmt.Errorf(
			"%w: Markdown source must contain valid UTF-8",
			spec.ErrInvalid,
		)
	}
	if bytes.ContainsRune(content, 0) {
		return "", fmt.Errorf(
			"%w: Markdown source contains a NUL byte",
			spec.ErrInvalid,
		)
	}

	value := strings.ReplaceAll(
		strings.ReplaceAll(string(content), "\r\n", "\n"),
		"\r",
		"\n",
	)
	return value, nil
}
