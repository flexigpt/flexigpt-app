package markdown

import (
	"context"
	"path"

	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	textv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/contract/v1"
)

const markdownMediaType = "text/markdown"

const TextMarkdownDecoderID spec.DecoderID = "text-markdown"

type TextDecoder struct {
	candidates   support.Candidates
	instructions support.Candidates
}

func NewTextDecoder(
	candidates support.Candidates,
	instructions support.Candidates,
) *TextDecoder {
	return &TextDecoder{
		candidates:   candidates.Clone(),
		instructions: instructions.Clone(),
	}
}

func (t *TextDecoder) ID() spec.DecoderID {
	return TextMarkdownDecoderID
}

func (t *TextDecoder) Revision() string {
	return "artifact-text-markdown/v1"
}

func (t *TextDecoder) Recognize(
	_ context.Context,
	candidate ingestModel.Candidate,
) ingestModel.Recognition {
	if candidate.RequestsDecoder(TextMarkdownDecoderID) ||
		t.candidates.Matches(candidate.Locator) {
		return ingestModel.RecognitionPreferred
	}
	return ingestModel.RecognitionNone
}

func (t *TextDecoder) Decode(
	_ context.Context,
	candidate ingestModel.Candidate,
) ([]ingestModel.Decoded, []diagnostic.Diagnostic) {
	if !t.candidates.Matches(candidate.Locator) &&
		!candidate.RequestsDecoder(TextMarkdownDecoderID) {
		return nil, nil
	}

	insert := declaration.InsertUserMessage
	prefix := "text"
	if t.instructions.Matches(candidate.Locator) {
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
	_, err = declaration.NewEntry(document)
	if err != nil {
		return nil, textDiagnostics(candidate.Locator, err)
	}
	value, err := textv1.DefinitionForDocument(document)
	if err != nil {
		return nil, textDiagnostics(candidate.Locator, err)
	}
	return []ingestModel.Decoded{{Definition: value}}, nil
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

// sourceEntryDeclarationLocator points back to the physical source entry
// containing a source-format declaration. It keeps source material out of
// Definition.Body while preserving declaration-relative locator semantics.
func sourceEntryDeclarationLocator(
	locator spec.Locator,
) *declaration.Locator {
	value := declaration.ScalarLocator(
		"./" + path.Base(string(locator)),
	)
	return &value
}
