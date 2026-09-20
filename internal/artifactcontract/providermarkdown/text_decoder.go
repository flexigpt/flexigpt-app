package providermarkdown

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
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
	return documentTopology.IsTextMarkdownDocument(locator)
}

func isDefaultTextFile(locator basespec.Locator) bool {
	return documentTopology.IsDefaultTextMarkdownDocument(locator)
}

func isInstructionFile(locator basespec.Locator) bool {
	return documentTopology.IsInstructionMarkdownDocument(locator)
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
