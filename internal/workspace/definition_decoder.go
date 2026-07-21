package workspace

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type DefinitionDecoder struct{}

func NewDefinitionDecoder() *DefinitionDecoder {
	return &DefinitionDecoder{}
}

func (*DefinitionDecoder) ID() artifactstore.DecoderID {
	return DefinitionDecoderID
}

func (*DefinitionDecoder) Recognize(
	_ context.Context,
	candidate discovery.Candidate,
) discovery.Recognition {
	if candidate.Locator != DefinitionLocator {
		return discovery.RecognitionNone
	}
	return discovery.RecognitionPreferred
}

func (*DefinitionDecoder) Decode(
	_ context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []artifactstore.Diagnostic) {
	canonical, err := jsoncanon.CanonicalizeObject(
		candidate.Content,
		artifactstore.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, definitionDiagnostics(candidate.Locator, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var document DefinitionDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, definitionDiagnostics(candidate.Locator, err)
	}
	if err := validateDiscoveryPreferences(document.Discovery); err != nil {
		return nil, definitionDiagnostics(candidate.Locator, err)
	}

	body, err := json.Marshal(document)
	if err != nil {
		return nil, definitionDiagnostics(candidate.Locator, err)
	}
	body, err = jsoncanon.CanonicalizeObject(
		body,
		artifactstore.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, definitionDiagnostics(candidate.Locator, err)
	}

	value := definition.Definition{
		Kind:          DefinitionKind,
		SchemaID:      DefinitionSchemaID,
		SchemaVersion: workspaceSchemaVersionV1,
		LogicalName:   workspaceDefinitionLogicalName,
		DisplayName:   workspaceDefinitionDisplayName,
		Body:          body,
	}
	return []discovery.Decoded{{
		Definition: value,
	}}, nil
}

func definitionDiagnostics(
	locator artifactstore.Locator,
	err error,
) []artifactstore.Diagnostic {
	message := err.Error()
	if len(message) > artifactstore.MaxDiagnosticMessageBytes {
		message = message[:artifactstore.MaxDiagnosticMessageBytes]
	}
	return []artifactstore.Diagnostic{{
		Severity: artifactstore.DiagnosticError,
		Code:     diagnosticCodeDefinitionInvalid,
		Message:  message,
		Location: &artifactstore.DiagnosticLocation{
			Locator: locator,
		},
	}}
}

var _ discovery.Decoder = (*DefinitionDecoder)(nil)
