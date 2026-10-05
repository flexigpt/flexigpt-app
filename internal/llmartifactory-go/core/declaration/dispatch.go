package declaration

import (
	"bytes"
	"encoding/json"
	"fmt"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type dispatchVersion struct {
	kind    schemaModel.Kind
	version string
}

// Dispatcher is an immutable declaration-language dispatch table.
//
// Its participating schema keys are explicit construction input. Ambiguity
// between complete keys sharing type/apiVersion is a declaration-dispatch
// error, not a restriction on the generic Store schema catalog.
type Dispatcher struct {
	byVersion map[dispatchVersion]schemaModel.Key
	byType    map[schemaModel.Kind][]schemaModel.Key
}

func NewDispatcher(keys []schemaModel.Key) (*Dispatcher, error) {
	output := &Dispatcher{
		byVersion: make(map[dispatchVersion]schemaModel.Key, len(keys)),
		byType:    make(map[schemaModel.Kind][]schemaModel.Key),
	}
	for index, key := range keys {
		if err := key.Validate(); err != nil {
			return nil, fmt.Errorf("declaration dispatch key %d: %w", index, err)
		}
		if key.Entity != schemaModel.EntityArtifact {
			return nil, fmt.Errorf(
				"%w: declaration dispatch requires Artifact schemas",
				spec.ErrInvalid,
			)
		}
		slot := dispatchVersion{kind: key.Kind, version: key.SchemaVersion}
		if previous, duplicate := output.byVersion[slot]; duplicate {
			return nil, fmt.Errorf(
				"%w: schemas %q and %q both dispatch from type %q and apiVersion %q",
				spec.ErrConflict,
				previous.SchemaID,
				key.SchemaID,
				key.Kind,
				key.SchemaVersion,
			)
		}
		output.byVersion[slot] = key
		output.byType[key.Kind] = append(output.byType[key.Kind], key)
	}
	return output, nil
}

// Resolve selects one complete expected schema key without executing a schema
// or canonicalizing the document.
//
// Empty or omitted apiVersion retains the existing omitted-version rule:
// exactly one participating schema must exist for that type.
func (d *Dispatcher) Resolve(raw []byte) (schemaModel.Key, error) {
	header, err := readDispatchHeader(raw)
	if err != nil {
		return schemaModel.Key{}, err
	}
	if header.version != "" {
		key, found := d.byVersion[dispatchVersion(header)]
		if !found {
			return schemaModel.Key{}, fmt.Errorf(
				"%w: declaration type %q apiVersion %q",
				spec.ErrUnsupported,
				header.kind,
				header.version,
			)
		}
		return key, nil
	}

	candidates := d.byType[header.kind]
	if len(candidates) != 1 {
		return schemaModel.Key{}, fmt.Errorf(
			"%w: declaration type %q without apiVersion resolves to %d registered schemas",
			spec.ErrUnsupported,
			header.kind,
			len(candidates),
		)
	}
	return candidates[0], nil
}

// ValidateSchemaHeader checks a known declaration schema's header contract.
//
// Declaration codecs use this for direct expected-key callers as well as
// dispatched decoding. Generic schema providers must not duplicate it.
func ValidateSchemaHeader(expected schemaModel.Key, raw []byte) error {
	header, err := readDispatchHeader(raw)
	if err != nil {
		return err
	}
	if expected.Entity != schemaModel.EntityArtifact ||
		header.kind != expected.Kind ||
		(header.version != "" && header.version != expected.SchemaVersion) {
		return fmt.Errorf(
			"%w: declaration header does not match expected schema %q/%q/%q",
			spec.ErrInvalid,
			expected.Kind,
			expected.SchemaID,
			expected.SchemaVersion,
		)
	}
	return nil
}

type dispatchHeader struct {
	kind    schemaModel.Kind
	version string
}

func readDispatchHeader(raw []byte) (dispatchHeader, error) {
	if len(raw) == 0 || len(raw) > spec.MaxDefinitionBytes {
		return dispatchHeader{}, fmt.Errorf(
			"%w: declaration document is empty or exceeds its size limit",
			spec.ErrInvalid,
		)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return dispatchHeader{}, fmt.Errorf(
			"%w: declaration document must be a JSON object",
			spec.ErrInvalid,
		)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return dispatchHeader{}, fmt.Errorf(
			"%w: decode declaration header: %w",
			spec.ErrInvalid,
			err,
		)
	}

	_, hasType := fields["type"]
	_, hasVersion := fields["apiVersion"]
	_, hasKind := fields["kind"]
	_, hasSchemaID := fields["schemaID"]
	_, hasSchemaVersion := fields["schemaVersion"]

	if hasKind || hasSchemaID || hasSchemaVersion {
		if hasType || hasVersion {
			return dispatchHeader{}, fmt.Errorf(
				"%w: declaration mixes type/apiVersion and kind/schemaID/schemaVersion headers",
				spec.ErrInvalid,
			)
		}
		return dispatchHeader{}, fmt.Errorf(
			"%w: declarations require type and optional apiVersion",
			spec.ErrInvalid,
		)
	}
	if !hasType {
		return dispatchHeader{}, fmt.Errorf(
			"%w: declaration requires type",
			spec.ErrInvalid,
		)
	}

	var header struct {
		Type       schemaModel.Kind `json:"type"`
		APIVersion string           `json:"apiVersion"`
	}
	if err := json.Unmarshal(trimmed, &header); err != nil {
		return dispatchHeader{}, fmt.Errorf(
			"%w: decode type/apiVersion header: %w",
			spec.ErrInvalid,
			err,
		)
	}
	if err := spec.ValidateIdentifier(
		"declaration type",
		string(header.Type),
		spec.MaxKindBytes,
	); err != nil {
		return dispatchHeader{}, err
	}

	return dispatchHeader{kind: header.Type, version: header.APIVersion}, nil
}
