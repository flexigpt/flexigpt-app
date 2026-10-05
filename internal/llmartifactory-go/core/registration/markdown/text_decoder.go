package markdown

import (
	"context"

	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/decoder"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/textv1"
)

const TextMarkdownDecoderID spec.DecoderID = "text-markdown"

type TextDecoder struct{}

func NewTextDecoder() *TextDecoder {
	return &TextDecoder{}
}

func (*TextDecoder) ID() spec.DecoderID {
	return TextMarkdownDecoderID
}

func (*TextDecoder) Revision() string {
	return "artifact-text-markdown/v1"
}

func (*TextDecoder) Recognize(
	_ context.Context,
	candidate ingestModel.Candidate,
) ingestModel.Recognition {
	if isInstructionFile(candidate.Locator) ||
		isDefaultTextFile(candidate.Locator) {
		return ingestModel.RecognitionPreferred
	}
	if candidate.RequestsDecoder(TextMarkdownDecoderID) &&
		isTextCandidate(candidate.Locator) {
		return ingestModel.RecognitionPossible
	}
	return ingestModel.RecognitionNone
}

func (*TextDecoder) Decode(
	_ context.Context,
	candidate ingestModel.Candidate,
) ([]ingestModel.Decoded, []diagnostic.Diagnostic) {
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
	return []ingestModel.Decoded{{Definition: value}}, nil
}

func isTextCandidate(locator spec.Locator) bool {
	return topology.IsTextMarkdownDocument(locator)
}

func isDefaultTextFile(locator spec.Locator) bool {
	return topology.IsDefaultTextMarkdownDocument(locator)
}

func isInstructionFile(locator spec.Locator) bool {
	return topology.IsInstructionMarkdownDocument(locator)
}

func textDiagnostics(
	locator spec.Locator,
	err error,
) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     "artifact.text-markdown.invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: &diagnostic.Location{Locator: locator},
	}}
}
