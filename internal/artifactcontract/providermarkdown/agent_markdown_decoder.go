package providermarkdown

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const AgentMarkdownDecoderID basespec.DecoderID = "agent-markdown"

const markdownMediaType = "text/markdown"

// AgentMarkdownDecoder adapts AGENT.md and *.agent.md files. YAML front
// matter provides Agent declaration fields. The Markdown body becomes a named
// Instruction in Agent.members using the <agent-name>-instructions naming
// convention.
type AgentMarkdownDecoder struct{}

func NewAgentMarkdownDecoder() *AgentMarkdownDecoder {
	return &AgentMarkdownDecoder{}
}

func (*AgentMarkdownDecoder) ID() basespec.DecoderID {
	return AgentMarkdownDecoderID
}

func (*AgentMarkdownDecoder) Revision() string {
	return "artifact-agent-markdown/v1"
}

func (*AgentMarkdownDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	if !isAgentMarkdownCandidate(candidate.Locator) {
		return providerapi.RecognitionNone
	}
	return providerapi.RecognitionPreferred
}

func (*AgentMarkdownDecoder) Decode(
	_ context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	document, body, err := decodeAgentMarkdown(
		candidate.Content,
		candidate.Locator,
	)
	if err != nil {
		return nil, agentMarkdownDiagnostics(candidate.Locator, err)
	}

	if strings.TrimSpace(body) != "" {
		instructionName, err := declaration.DeriveNestedLogicalName(
			basespec.LogicalName(document.Name),
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
	definitionValue, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return nil, agentMarkdownDiagnostics(candidate.Locator, err)
	}
	namedEntries, err := declaration.WalkNamedEntries(entry)
	if err != nil {
		return nil, agentMarkdownDiagnostics(candidate.Locator, err)
	}

	output := make([]providerapi.Decoded, 0, len(namedEntries))
	for index, named := range namedEntries {
		value := definitionValue
		if index != 0 {
			value, err = decoder.DefinitionForNamedEntry(named)
			if err != nil {
				return nil, agentMarkdownDiagnostics(
					candidate.Locator,
					err,
				)
			}
		}
		output = append(output, providerapi.Decoded{
			SubresourceLocator: named.SubresourceLocator,
			Definition:         value,
		})
	}
	return output, nil
}

func isAgentMarkdownCandidate(
	locator basespec.Locator,
) bool {
	name := strings.ToLower(path.Base(string(locator)))
	return name == "agent.md" ||
		strings.HasSuffix(name, ".agent.md")
}

func decodeAgentMarkdown(
	content []byte,
	locator basespec.Locator,
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
			basespec.MaxDefinitionBytes,
		)
		if err != nil {
			return agentv1.AgentDocument{}, "", err
		}
		if err := jsonutil.DecodeCanonicalObjectBytesInto(
			raw,
			&fields,
			basespec.MaxDefinitionBytes,
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
		basespec.MaxDefinitionBytes,
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
		basespec.ErrInvalid,
	)
}

func agentMarkdownDiagnostics(
	locator basespec.Locator,
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
			basespec.ErrInvalid,
		)
	}
	if bytes.ContainsRune(content, 0) {
		return "", fmt.Errorf(
			"%w: Markdown source contains a NUL byte",
			basespec.ErrInvalid,
		)
	}

	value := strings.ReplaceAll(
		strings.ReplaceAll(string(content), "\r\n", "\n"),
		"\r",
		"\n",
	)
	return value, nil
}

// sourceEntryDeclarationLocator points back to the physical source entry
// containing a source-format declaration. It keeps source material out of
// Definition.Body while preserving declaration-relative locator semantics.
func sourceEntryDeclarationLocator(
	locator basespec.Locator,
) *declaration.Locator {
	value := declaration.ScalarLocator(
		"./" + path.Base(string(locator)),
	)
	return &value
}
