package providermarkdown

import (
	"context"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

const TextMarkdownDecoderID basespec.DecoderID = "text-markdown"

type TextDecoder struct{}

func NewTextDecoder() *TextDecoder {
	return &TextDecoder{}
}

func (*TextDecoder) ID() basespec.DecoderID {
	return TextMarkdownDecoderID
}

func (*TextDecoder) Revision() string {
	return "artifact-text-markdown/v1"
}

func (*TextDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	if isInstructionFile(candidate.Locator) ||
		isDefaultTextFile(candidate.Locator) {
		return providerapi.RecognitionPreferred
	}
	if candidate.RequestsDecoder(TextMarkdownDecoderID) &&
		isTextCandidate(candidate.Locator) {
		return providerapi.RecognitionPossible
	}
	return providerapi.RecognitionNone
}

func (*TextDecoder) Decode(
	_ context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	if !isTextCandidate(candidate.Locator) {
		return nil, nil
	}

	insert := declaration.InsertUserMessage
	prefix := "text"
	if isInstructionFile(candidate.Locator) {
		insert = declaration.InsertInstructions
		prefix = "instructions"
	}
	name, err := declaration.DeriveLogicalName(prefix, candidate.Locator)
	if err != nil {
		return nil, textDiagnostics(candidate.Locator, err)
	}
	document := textv1.TextDocument{
		Type:        textv1.TextType,
		Name:        string(name),
		Description: "Text source " + string(candidate.Locator),
		Locator:     sourceEntryDeclarationLocator(candidate.Locator),
		Insert:      insert,
		MediaType:   markdownMediaType,
	}
	entry, err := declaration.NewEntry(document)
	if err != nil {
		return nil, textDiagnostics(candidate.Locator, err)
	}
	value, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return nil, textDiagnostics(candidate.Locator, err)
	}
	return []providerapi.Decoded{{Definition: value}}, nil
}

func isTextCandidate(locator basespec.Locator) bool {
	if isAgentMarkdownCandidate(locator) {
		return false
	}
	if strings.EqualFold(path.Base(string(locator)), "llms.txt") {
		return true
	}
	return strings.EqualFold(path.Ext(string(locator)), ".md") &&
		!strings.EqualFold(path.Base(string(locator)), "SKILL.md")
}

func isDefaultTextFile(locator basespec.Locator) bool {
	switch strings.ToUpper(path.Base(string(locator))) {
	case "README.MD", "LLMS.TXT", "AGENTS.MD", "CLAUDE.MD":
		return true
	default:
		return false
	}
}

func isInstructionFile(locator basespec.Locator) bool {
	switch strings.ToUpper(path.Base(string(locator))) {
	case "AGENTS.MD", "CLAUDE.MD":
		return true
	default:
		return false
	}
}

func textDiagnostics(
	locator basespec.Locator,
	err error,
) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     "artifact.text-markdown.invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: &diagnostic.Location{Locator: locator},
	}}
}
