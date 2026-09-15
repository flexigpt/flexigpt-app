package decoder

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/codec"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
)

// DefinitionForEntry projects one named canonical declaration Entry into the
// generic immutable Artifact Store Definition model.
//
// Entry bytes are already canonical. Projection validates the concrete
// declaration type but retains those exact canonical bytes as Definition.Body.
func DefinitionForEntry(
	entry declaration.Entry,
) (definition.Definition, error) {
	if err := ValidateEntryTree(entry); err != nil {
		return definition.Definition{}, err
	}
	return definitionForEntry(entry)
}

// DefinitionForNamedEntry projects one already tree-validated named entry.
// Canonical and source-format decoders use this after validating the complete
// containing declaration and walking its named entries.
func DefinitionForNamedEntry(
	named declaration.NamedEntry,
) (definition.Definition, error) {
	if err := named.Validate(); err != nil {
		return definition.Definition{}, err
	}
	if err := ValidateEntryTree(named.Entry); err != nil {
		return definition.Definition{}, err
	}
	return definitionForNamedEntry(named)
}

// definitionForNamedEntry is used by canonicalDecoder after the complete
// containing document has already passed ValidateEntryTree.
func definitionForNamedEntry(
	named declaration.NamedEntry,
) (definition.Definition, error) {
	if err := named.Validate(); err != nil {
		return definition.Definition{}, err
	}
	return definitionForEntry(named.Entry)
}

func definitionForEntry(
	entry declaration.Entry,
) (definition.Definition, error) {
	if err := entry.Validate(); err != nil {
		return definition.Definition{}, err
	}

	header := entry.Header()
	body, err := entry.CanonicalJSON()
	if err != nil {
		return definition.Definition{}, err
	}

	key, found := codec.SchemaKeyForType(header.Type)
	if !found {
		return definition.Definition{}, fmt.Errorf(
			"%w: unsupported canonical declaration type %q",
			basespec.ErrUnsupported,
			header.Type,
		)
	}

	var logicalVersion basespec.LogicalVersion
	if header.Type == declaration.TypeCollection {
		var fields struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(body, &fields); err != nil {
			return definition.Definition{}, fmt.Errorf(
				"%w: decode Collection version: %w",
				basespec.ErrInvalid,
				err,
			)
		}
		logicalVersion = basespec.LogicalVersion(fields.Version)
	}

	return definitionForDocument(
		header,
		key,
		logicalVersion,
		body,
	)
}

func definitionForDocument(
	header declaration.Header,
	key schema.Key,
	logicalVersion basespec.LogicalVersion,
	body []byte,
) (definition.Definition, error) {
	value := definition.Definition{
		Kind:           artifact.ArtifactKind(key.Kind),
		SchemaID:       key.SchemaID,
		SchemaVersion:  key.SchemaVersion,
		LogicalName:    basespec.LogicalName(header.Name),
		LogicalVersion: logicalVersion,
		DisplayName:    header.Name,
		Description:    header.Description,
		Body:           json.RawMessage(body),
	}
	return definition.Canonicalize(value)
}
