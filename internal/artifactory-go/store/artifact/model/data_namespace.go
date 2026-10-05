package model

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// DataNamespace identifies one independently owned Artifact.Data field. The
// Store does not interpret its payload; family services own its schema.
type DataNamespace string

func (n DataNamespace) Validate() error {
	return spec.ValidateRequiredText(
		"Artifact local-data namespace",
		string(n),
		spec.MaxSchemaIDBytes,
	)
}

// RemoveDataNamespaces removes selected namespaced fields while preserving
// every unrelated Artifact.Data field.
func RemoveDataNamespaces(
	raw json.RawMessage,
	namespaces []DataNamespace,
) (nextData json.RawMessage, changed bool, err error) {
	fields, err := DecodeDataObject(raw)
	if err != nil {
		return nil, false, err
	}

	removed := false
	for _, namespace := range namespaces {
		if err := namespace.Validate(); err != nil {
			return nil, false, err
		}
		if _, found := fields[string(namespace)]; !found {
			continue
		}
		delete(fields, string(namespace))
		removed = true
	}
	if !removed {
		return append(json.RawMessage(nil), raw...), false, nil
	}

	next, err := EncodeDataObject(fields)
	if err != nil {
		return nil, false, fmt.Errorf(
			"encode Artifact local-data cleanup: %w",
			err,
		)
	}
	return next, true, nil
}
