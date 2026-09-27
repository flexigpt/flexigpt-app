package builtin

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// DecodeGeneratedPackageSet reads embedded generated JSON through the same
// canonical map representation used for compiler output fingerprints.
func DecodeGeneratedPackageSet(
	raw []byte,
) (
	topology.CompiledPackageSet,
	cryptoutil.Digest,
	error,
) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return topology.CompiledPackageSet{}, "", fmt.Errorf(
			"%w: generated built-in package set is empty",
			basespec.ErrInvalid,
		)
	}

	canonical, err := canonicalJSON(raw)
	if err != nil {
		return topology.CompiledPackageSet{}, "", fmt.Errorf(
			"canonicalize generated built-in package set JSON: %w",
			err,
		)
	}

	return decodeCanonicalGeneratedPackageSet(canonical)
}

// CanonicalGeneratedPackageSet converts a typed package set to the canonical
// generic-map JSON representation, then decodes that representation back to a
// typed package set.
//
// This is intentionally the same transformation used when loading embedded
// JSON. It normalizes object key order and raw JSON body formatting without
// changing array order.
func CanonicalGeneratedPackageSet(
	value topology.CompiledPackageSet,
) (
	topology.CompiledPackageSet,
	cryptoutil.Digest,
	error,
) {
	normalized, _, fingerprint, err := canonicalGeneratedPackageSet(value)
	if err != nil {
		return topology.CompiledPackageSet{}, "", err
	}
	return normalized, fingerprint, nil
}

// RenderGeneratedPackageSetJSON emits the stable checked-in JSON form.
//
// Object keys are lexicographically sorted through canonicalJSON. Arrays keep
// the order emitted by the compiler. The two-space indentation is presentation
// only and does not affect the generated fingerprint.
func RenderGeneratedPackageSetJSON(
	value topology.CompiledPackageSet,
) ([]byte, error) {
	_, canonical, _, err := canonicalGeneratedPackageSet(value)
	if err != nil {
		return nil, err
	}

	var output bytes.Buffer
	if err := json.Indent(&output, canonical, "", "  "); err != nil {
		return nil, fmt.Errorf(
			"format generated built-in package set JSON: %w",
			err,
		)
	}
	output.WriteByte('\n')

	return output.Bytes(), nil
}

func canonicalGeneratedPackageSet(
	value topology.CompiledPackageSet,
) (
	topology.CompiledPackageSet,
	[]byte,
	cryptoutil.Digest,
	error,
) {
	raw, err := json.Marshal(value)
	if err != nil {
		return topology.CompiledPackageSet{}, nil, "", fmt.Errorf(
			"marshal generated built-in package set: %w",
			err,
		)
	}

	canonical, err := canonicalJSON(raw)
	if err != nil {
		return topology.CompiledPackageSet{}, nil, "", err
	}

	normalized, fingerprint, err := decodeCanonicalGeneratedPackageSet(canonical)
	if err != nil {
		return topology.CompiledPackageSet{}, nil, "", err
	}

	return normalized, canonical, fingerprint, nil
}

func decodeCanonicalGeneratedPackageSet(
	canonical []byte,
) (
	topology.CompiledPackageSet,
	cryptoutil.Digest,
	error,
) {
	var value topology.CompiledPackageSet
	if err := json.Unmarshal(canonical, &value); err != nil {
		return topology.CompiledPackageSet{}, "", fmt.Errorf(
			"decode canonical generated built-in package set JSON: %w",
			err,
		)
	}

	if value.Format != topology.CompiledPackageSetFormat {
		return topology.CompiledPackageSet{}, "", fmt.Errorf(
			"%w: generated package format %q; regenerate built-in catalogs",
			basespec.ErrUnsupported,
			value.Format,
		)
	}

	if value.Name == "" ||
		value.Hydration.InstallerName == "" ||
		value.Hydration.Fingerprint == "" {
		return topology.CompiledPackageSet{}, "", fmt.Errorf(
			"%w: generated built-in package set is incomplete",
			basespec.ErrInvalid,
		)
	}

	return value, cryptoutil.DigestBytes(canonical), nil
}

// Use the same JSON canonicalization policy as Definition admission.
// This is performed once when loading the embedded generated catalog.
func canonicalJSON(
	raw []byte,
) ([]byte, error) {
	return jsonutil.Canonicalize(raw)
}
