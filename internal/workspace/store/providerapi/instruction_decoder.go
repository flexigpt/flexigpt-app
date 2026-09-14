package providerapi

import (
	"context"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/instructionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

const InstructionMarkdownDecoderID basespec.DecoderID = "workspace-instruction-markdown"

type InstructionDecoder struct{}

func NewInstructionDecoder() *InstructionDecoder {
	return &InstructionDecoder{}
}

func (*InstructionDecoder) ID() basespec.DecoderID {
	return InstructionMarkdownDecoderID
}

func (*InstructionDecoder) Revision() string {
	return "workspace-instruction-markdown/v1"
}

func (*InstructionDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	if isInstructionFile(candidate.Locator) {
		return providerapi.RecognitionPreferred
	}
	if candidate.RequestsDecoder(InstructionMarkdownDecoderID) &&
		strings.EqualFold(path.Ext(string(candidate.Locator)), ".md") {
		return providerapi.RecognitionPossible
	}
	return providerapi.RecognitionNone
}

func (*InstructionDecoder) Decode(
	_ context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	if !isInstructionFile(candidate.Locator) &&
		!candidate.RequestsDecoder(InstructionMarkdownDecoderID) {
		return nil, nil
	}
	if !strings.EqualFold(path.Ext(string(candidate.Locator)), ".md") {
		return nil, nil
	}

	content, err := normalizeMarkdown(candidate.Content)
	if err != nil {
		return nil, instructionDiagnostics(candidate.Locator, err)
	}

	name := logicalNameForLocator(
		"instruction",
		candidate.Locator,
	)
	dec := instructionv1.InstructionDocument{
		APIVersion:  instructionv1.InstructionSchemaVersion,
		Type:        instructionv1.InstructionType,
		Name:        string(name),
		Description: "Instruction source " + string(candidate.Locator),
		Content:     stringPointer(content),
		MediaType:   markdownMediaType,
	}
	entry, err := declaration.NewEntry(dec)
	if err != nil {
		return nil, instructionDiagnostics(candidate.Locator, err)
	}
	value, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return nil, instructionDiagnostics(candidate.Locator, err)
	}
	return []providerapi.Decoded{{
		Definition: value,
	}}, nil
}

func isInstructionFile(locator basespec.Locator) bool {
	switch strings.ToUpper(path.Base(string(locator))) {
	case "AGENTS.MD", "CLAUDE.MD":
		return true
	default:
		return false
	}
}

func instructionDiagnostics(
	locator basespec.Locator,
	err error,
) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     "workspace.instruction.invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: &diagnostic.Location{
			Locator: locator,
		},
	}}
}
