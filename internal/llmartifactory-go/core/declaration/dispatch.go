package declaration

import (
	"bytes"
	"encoding/json"
	"fmt"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
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
				storeSpec.ErrInvalid,
			)
		}
		slot := dispatchVersion{kind: key.Kind, version: key.SchemaVersion}
		if previous, duplicate := output.byVersion[slot]; duplicate {
			return nil, fmt.Errorf(
				"%w: schemas %q and %q both dispatch from type %q and apiVersion %q",
				storeSpec.ErrConflict,
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
	if header.versionSpecified {
		key, found := d.byVersion[dispatchVersion{
			kind:    header.kind,
			version: header.version,
		}]
		if !found {
			return schemaModel.Key{}, fmt.Errorf(
				"%w: declaration type %q apiVersion %q",
				storeSpec.ErrUnsupported,
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
			storeSpec.ErrUnsupported,
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
		(header.versionSpecified &&
			header.version != expected.SchemaVersion) {
		return fmt.Errorf(
			"%w: declaration header does not match expected schema %q/%q/%q",
			storeSpec.ErrInvalid,
			expected.Kind,
			expected.SchemaID,
			expected.SchemaVersion,
		)
	}
	return nil
}

type dispatchHeader struct {
	kind             schemaModel.Kind
	version          string
	versionSpecified bool
}

func readDispatchHeader(raw []byte) (dispatchHeader, error) {
	if len(raw) == 0 || len(raw) > storeSpec.MaxDefinitionBytes {
		return dispatchHeader{}, fmt.Errorf(
			"%w: declaration document is empty or exceeds its size limit",
			storeSpec.ErrInvalid,
		)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return dispatchHeader{}, fmt.Errorf(
			"%w: declaration document must be a JSON object",
			storeSpec.ErrInvalid,
		)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return dispatchHeader{}, fmt.Errorf(
			"%w: decode declaration header: %w",
			storeSpec.ErrInvalid,
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
				storeSpec.ErrInvalid,
			)
		}
		return dispatchHeader{}, fmt.Errorf(
			"%w: declarations require type and optional apiVersion",
			storeSpec.ErrInvalid,
		)
	}
	if !hasType {
		return dispatchHeader{}, fmt.Errorf(
			"%w: declaration requires type",
			storeSpec.ErrInvalid,
		)
	}

	var header struct {
		Type       schemaModel.Kind `json:"type"`
		APIVersion string           `json:"apiVersion"`
	}
	if err := json.Unmarshal(trimmed, &header); err != nil {
		return dispatchHeader{}, fmt.Errorf(
			"%w: decode type/apiVersion header: %w",
			storeSpec.ErrInvalid,
			err,
		)
	}
	if err := storeSpec.ValidateIdentifier(
		"declaration type",
		string(header.Type),
		storeSpec.MaxKindBytes,
	); err != nil {
		return dispatchHeader{}, err
	}

	if hasVersion {
		if err := storeSpec.ValidateRequiredText(
			"declaration apiVersion",
			header.APIVersion,
			storeSpec.MaxVersionBytes,
		); err != nil {
			return dispatchHeader{}, err
		}
	}

	return dispatchHeader{
		kind:             header.Type,
		version:          header.APIVersion,
		versionSpecified: hasVersion,
	}, nil
}

// PrepareSourceDocument validates a standalone source header against its
// selected schema key, then removes dispatch-only apiVersion metadata.
//
// "requireAPIVersion" is owned by the declaration contract. Canonical Definition
// bodies and contained entries are not standalone source documents and must
// not be passed through this function merely to reconstruct admitted data.
func PrepareSourceDocument(
	expected schemaModel.Key,
	raw []byte,
	requireAPIVersion bool,
) ([]byte, error) {
	if err := expected.Validate(); err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		storeSpec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, err
	}

	header, err := readDispatchHeader(canonical)
	if err != nil {
		return nil, err
	}
	if err := ValidateSchemaHeader(expected, canonical); err != nil {
		return nil, err
	}
	if requireAPIVersion && !header.versionSpecified {
		return nil, fmt.Errorf(
			"%w: declaration type %q requires apiVersion %q",
			storeSpec.ErrInvalid,
			expected.Kind,
			expected.SchemaVersion,
		)
	}
	if !header.versionSpecified {
		return canonical, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &fields); err != nil {
		return nil, fmt.Errorf(
			"%w: decode declaration for apiVersion normalization: %w",
			storeSpec.ErrInvalid,
			err,
		)
	}
	delete(fields, "apiVersion")
	return jsonutil.MarshalCanonicalObject(
		fields,
		storeSpec.MaxDefinitionBytes,
	)
}
