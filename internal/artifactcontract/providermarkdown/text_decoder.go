package providermarkdown

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
)

const TextMarkdownDecoderID model.DecoderID = "text-markdown"

type TextDecoder struct{}

func NewTextDecoder() *TextDecoder {
	return &TextDecoder{}
}

func (*TextDecoder) ID() model.DecoderID {
	return TextMarkdownDecoderID
}

func (*TextDecoder) Revision() string {
	return "artifact-text-markdown/v1"
}

func (*TextDecoder) Recognize(
	_ context.Context,
	candidate provider.Candidate,
) provider.Recognition {
	if isInstructionFile(candidate.Locator) ||
		isDefaultTextFile(candidate.Locator) {
		return provider.RecognitionPreferred
	}
	if candidate.RequestsDecoder(TextMarkdownDecoderID) &&
		isTextCandidate(candidate.Locator) {
		return provider.RecognitionPossible
	}
	return provider.RecognitionNone
}

func (*TextDecoder) Decode(
	_ context.Context,
	candidate provider.Candidate,
) ([]provider.Decoded, []diagnostic.Diagnostic) {
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
	return []provider.Decoded{{Definition: value}}, nil
}

func isTextCandidate(locator model.Locator) bool {
	return documentTopology.IsTextMarkdownDocument(locator)
}

func isDefaultTextFile(locator model.Locator) bool {
	return documentTopology.IsDefaultTextMarkdownDocument(locator)
}

func isInstructionFile(locator model.Locator) bool {
	return documentTopology.IsInstructionMarkdownDocument(locator)
}

func textDiagnostics(
	locator model.Locator,
	err error,
) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     "artifact.text-markdown.invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: &diagnostic.Location{Locator: locator},
	}}
}
