package providerapi

import (
	"context"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/contextv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

const ContextMarkdownDecoderID basespec.DecoderID = "workspace-context-markdown"

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
	if strings.EqualFold(
		path.Base(string(candidate.Locator)),
		"README.md",
	) {
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
	if !candidate.RequestsDecoder(ContextMarkdownDecoderID) &&
		!strings.EqualFold(
			path.Base(string(candidate.Locator)),
			"README.md",
		) {
		return nil, nil
	}

	content, err := normalizeMarkdown(candidate.Content)
	if err != nil {
		return nil, contextDiagnostics(candidate.Locator, err)
	}
	name := logicalNameForLocator("context", candidate.Locator)
	dec := contextv1.ContextDocument{
		APIVersion:  contextv1.ContextSchemaVersion,
		Type:        contextv1.ContextType,
		Name:        string(name),
		Description: "Context source " + string(candidate.Locator),
		Content:     stringPointer(content),
		MediaType:   markdownMediaType,
	}
	entry, err := declaration.NewEntry(dec)
	if err != nil {
		return nil, contextDiagnostics(candidate.Locator, err)
	}
	value, err := DefinitionForEntry(entry)
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
