package markdown

import (
	"context"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/contextv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

const ContextMarkdownDecoderID basespec.DecoderID = "context-markdown"

type ContextDecoder struct{}

func NewContextDecoder() *ContextDecoder {
	return &ContextDecoder{}
}

func (*ContextDecoder) ID() basespec.DecoderID {
	return ContextMarkdownDecoderID
}

func (*ContextDecoder) Revision() string {
	return "workspace-context-markdown/v1"
}

func (*ContextDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	if !isMarkdownContextCandidate(candidate.Locator) {
		return providerapi.RecognitionNone
	}
	if isDefaultContextFile(candidate.Locator) {
		return providerapi.RecognitionPreferred
	}
	if candidate.RequestsDecoder(ContextMarkdownDecoderID) {
		return providerapi.RecognitionPreferred
	}
	return providerapi.RecognitionNone
}

func (*ContextDecoder) Decode(
	_ context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	if !isMarkdownContextCandidate(candidate.Locator) {
		return nil, nil
	}
	if !candidate.RequestsDecoder(ContextMarkdownDecoderID) && !isDefaultContextFile(candidate.Locator) {
		return nil, nil
	}

	name, err := declaration.DeriveLogicalName(
		"context",
		candidate.Locator,
	)
	if err != nil {
		return nil, contextDiagnostics(candidate.Locator, err)
	}
	dec := contextv1.ContextDocument{
		APIVersion:  contextv1.ContextSchemaVersion,
		Type:        contextv1.ContextType,
		Name:        string(name),
		Description: "Context source " + string(candidate.Locator),
		MediaType:   markdownMediaType,
		Locator:     sourceEntryDeclarationLocator(candidate.Locator),
	}
	entry, err := declaration.NewEntry(dec)
	if err != nil {
		return nil, contextDiagnostics(candidate.Locator, err)
	}
	value, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return nil, contextDiagnostics(candidate.Locator, err)
	}
	return []providerapi.Decoded{{
		Definition: value,
	}}, nil
}

func isMarkdownContextCandidate(
	locator basespec.Locator,
) bool {
	if isAgentMarkdownCandidate(locator) {
		return false
	}
	if strings.EqualFold(path.Base(string(locator)), "llms.txt") {
		return true
	}
	if !strings.EqualFold(path.Ext(string(locator)), ".md") {
		return false
	}
	switch strings.ToUpper(path.Base(string(locator))) {
	case "AGENTS.MD", "CLAUDE.MD", "SKILL.MD":
		return false
	default:
		return true
	}
}

func isDefaultContextFile(
	locator basespec.Locator,
) bool {
	switch strings.ToUpper(path.Base(string(locator))) {
	case "README.MD", "LLMS.TXT":
		return true
	default:
		return false
	}
}

func contextDiagnostics(
	locator basespec.Locator,
	err error,
) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     "workspace.context.invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: &diagnostic.Location{
			Locator: locator,
		},
	}}
}
