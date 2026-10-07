package domain

import (
	"fmt"

	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
)

func IsAgentDeclarationDocument(
	documents support.Documents,
	locator spec.Locator,
) bool {
	return documents.Matches(locator)
}

// ManagedAgentEntryPayload projects one concrete canonical Agent Entry into
// managed package bytes and an immutable Definition.
func ManagedAgentEntryPayload(
	entry declaration.Entry,
) ([]byte, definitionModel.Definition, error) {
	if err := declaration.ValidateEntryType(
		entry,
		declaration.TypeAgent,
	); err != nil {
		return nil, definitionModel.Definition{}, err
	}

	document, err := agentv1.DecodeAgentEntry(entry)
	if err != nil {
		return nil, definitionModel.Definition{}, err
	}
	if document.Locator != nil {
		return nil, definitionModel.Definition{}, fmt.Errorf(
			"%w: managed Agent declaration cannot be a source-selected alias",
			spec.ErrUnsupported,
		)
	}
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return nil, definitionModel.Definition{}, err
	}
	value, err := agentv1.DefinitionForDocument(document)
	if err != nil {
		return nil, definitionModel.Definition{}, err
	}
	if value.Kind != AgentArtifactKind ||
		value.LogicalName != spec.LogicalName(document.Name) ||
		value.LogicalVersion != "" {
		return nil, definitionModel.Definition{}, fmt.Errorf(
			"%w: managed Agent Definition identity is invalid",
			spec.ErrInvalid,
		)
	}
	return raw, value, nil
}
