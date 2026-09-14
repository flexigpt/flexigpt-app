package workspacev1

import (
	"bytes"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// AsLocator returns the declaration source as a portable Locator when it is
// an exact locator form.
func (s DeclarationSource) AsLocator() (
	declaration.Locator,
	bool,
	error,
) {
	raw, err := s.CanonicalJSON()
	if err != nil {
		return declaration.Locator{}, false, err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return declaration.Locator{}, false, nil
	}
	if trimmed[0] == '"' {
		var value declaration.Locator
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return declaration.Locator{}, false, err
		}
		return value, true, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return declaration.Locator{}, false, err
	}
	if _, found := fields["kind"]; !found {
		return declaration.Locator{}, false, nil
	}
	var value declaration.Locator
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return declaration.Locator{}, false, err
	}
	return value, true, nil
}

// AsEntry returns one inline declaration source. The returned Entry is
// generic until a concrete contract integration dispatches its type.
func (s DeclarationSource) AsEntry() (
	declaration.Entry,
	bool,
	error,
) {
	raw, err := s.CanonicalJSON()
	if err != nil {
		return declaration.Entry{}, false, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		//nolint:nilerr // Needed explicit return.
		return declaration.Entry{}, false, nil
	}
	if _, found := fields["type"]; !found {
		return declaration.Entry{}, false, nil
	}
	value, err := declaration.DecodeEntryJSON(raw)
	if err != nil {
		return declaration.Entry{}, false, err
	}
	return value, true, nil
}

// AsScan returns one local or externally resolvable declaration scan.
func (s DeclarationSource) AsScan() (
	DeclarationScan,
	bool,
	error,
) {
	raw, err := s.CanonicalJSON()
	if err != nil {
		return DeclarationScan{}, false, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		//nolint:nilerr // Needed explicit return.
		return DeclarationScan{}, false, nil
	}
	if _, found := fields["base"]; !found {
		return DeclarationScan{}, false, nil
	}
	var value DeclarationScan
	if err := jsonutil.DecodeCanonicalObjectInto(
		raw,
		&value,
		basespec.MaxDefinitionBodyBytes,
	); err != nil {
		return DeclarationScan{}, false, err
	}
	return value, true, nil
}
