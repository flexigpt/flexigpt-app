package engine

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
	if err := ValidateWorkspaceDefinition(value); err != nil {
		return nil, definitionDiagnostics(candidate.Locator, err)
	}
	return []discovery.Decoded{{
		Definition: value,
	}}, nil
}

func definitionDiagnostics(
	locator artifactstore.Locator,
	err error,
) []artifactstore.Diagnostic {
	return WorkspaceArtifactDiagnostics(
		locator,
		DiagnosticCodeDefinitionInvalid,
		err.Error(),
	)
}

var _ discovery.Decoder = (*DefinitionDecoder)(nil)
