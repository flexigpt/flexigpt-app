package pluginv1

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

// DefinitionForDocument is Plugin's family-owned Definition reconstruction
// boundary. Package and membership code use it rather than a central
// declaration-type switch.
func DefinitionForDocument(
	document PluginDocument,
) (definitionModel.Definition, error) {
	body, err := document.CanonicalJSON()
	if err != nil {
		return definitionModel.Definition{}, err
	}
	displayName := document.DisplayName
	if displayName == "" {
		displayName = document.Name
	}
	return definitionModel.Canonicalize(definitionModel.Definition{
		Kind:          artifactModel.ArtifactKind(PluginType),
		SchemaID:      PluginSchemaKey.SchemaID,
		SchemaVersion: PluginSchemaKey.SchemaVersion,
		LogicalName:   spec.LogicalName(document.Name),
		DisplayName:   displayName,
		Description:   document.Description,
		Labels:        declaration.CloneStringMap(document.Labels),
		Body:          body,
	})
}

// FromDefinition reconstructs an owned Plugin projection from a Store-admitted
// Definition. It verifies the interpretation and identity linkage without
// re-executing declaration admission or hashing the immutable body.
//
// Call DecodePluginJSON for unadmitted external document input.
func FromDefinition(value definitionModel.Definition) (PluginDocument, error) {
	if value.Kind != artifactModel.ArtifactKind(PluginType) ||
		value.SchemaID != PluginSchemaKey.SchemaID ||
		value.SchemaVersion != PluginSchemaKey.SchemaVersion {
		return PluginDocument{}, fmt.Errorf(
			"%w: Definition is not a supported Plugin declaration",
			spec.ErrUnsupported,
		)
	}
	if err := definitionModel.ValidateAdmitted(value); err != nil {
		return PluginDocument{}, err
	}
	entry, err := declaration.DecodeCanonicalEntryJSON(value.Body)
	if err != nil {
		return PluginDocument{}, err
	}
	document, err := DecodeAdmittedPluginEntry(entry)
	if err != nil {
		return PluginDocument{}, err
	}
	if document.Type != PluginType ||
		document.Name != string(value.LogicalName) {
		return PluginDocument{}, fmt.Errorf(
			"%w: Plugin body identity differs from Definition metadata",
			spec.ErrDigestMismatch,
		)
	}
	return document, nil
}
