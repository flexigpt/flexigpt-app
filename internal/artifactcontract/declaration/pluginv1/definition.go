package pluginv1

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

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
	document, err := definitionModel.DecodeBody[PluginDocument](value.Body)
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
