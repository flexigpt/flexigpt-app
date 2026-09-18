package domain

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
)

// ManagedAgentDocumentPayload validates one concrete managed Agent declaration
// and projects it into its canonical source bytes and immutable Definition.
//
// Managed Agent packages must contain concrete Agent declarations. A
// source-selected Agent alias is a valid portable declaration, but it is not
// a managed Agent package body because the package would not own the selected
// declaration occurrence.
func ManagedAgentDocumentPayload(
	document agentv1.AgentDocument,
) ([]byte, definition.Definition, error) {
	if err := document.Validate(); err != nil {
		return nil, definition.Definition{}, err
	}
	if document.Locator != nil {
		return nil, definition.Definition{}, fmt.Errorf(
			"%w: managed Agent declaration cannot be a source-selected alias",
			basespec.ErrUnsupported,
		)
	}

	entry, err := declaration.NewEntry(document)
	if err != nil {
		return nil, definition.Definition{}, err
	}
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return nil, definition.Definition{}, err
	}

	value, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return nil, definition.Definition{}, err
	}
	if value.Kind != AgentArtifactKind ||
		value.LogicalName != basespec.LogicalName(document.Name) ||
		value.LogicalVersion != "" {
		return nil, definition.Definition{}, fmt.Errorf(
			"%w: managed Agent Definition identity is invalid",
			basespec.ErrInvalid,
		)
	}

	return raw, value, nil
}
