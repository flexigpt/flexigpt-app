package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

var ErrGeneratedPackageSetNotLoaded = errors.New(
	"generated Artifact package set is not loaded",
)

// Generated payloads contain base64 file content and Definition metadata.
// This is an installation-envelope limit, not declaration vocabulary.
const maximumGeneratedPackageSetBytes = 2 * spec.MaxScanBytes

// DecodeGeneratedPackageSet decodes trusted generated installation data.
// It does not admit ordinary user-supplied declarations.
//
// The wire representation, array ordering, and fingerprint domain remain the
// existing artifact-compiled-package-set/v1 representation.
func DecodeGeneratedPackageSet(raw []byte) (installModel.CompiledPackageSet, error) {
	if len(bytes.TrimSpace(raw)) == 0 ||
		int64(len(raw)) > maximumGeneratedPackageSetBytes {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"%w: generated package set is empty or exceeds its size limit",
			spec.ErrInvalid,
		)
	}
	var value installModel.CompiledPackageSet
	if err := json.Unmarshal(raw, &value); err != nil {
		return installModel.CompiledPackageSet{}, fmt.Errorf(
			"decode generated package set JSON: %w",
			err,
		)
	}
	if err := validateGeneratedEnvelope(value); err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	return value, nil
}

func CanonicalGeneratedPackageSet(
	value installModel.CompiledPackageSet,
) (installModel.CompiledPackageSet, cryptoutil.Digest, error) {
	normalized, _, fingerprint, err := canonicalGeneratedPackageSet(value)
	return normalized, fingerprint, err
}

func RenderGeneratedPackageSetJSON(
	value installModel.CompiledPackageSet,
) ([]byte, error) {
	_, canonical, _, err := canonicalGeneratedPackageSet(value)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := json.Indent(&output, canonical, "", "  "); err != nil {
		return nil, fmt.Errorf("format generated package set JSON: %w", err)
	}
	output.WriteByte('\n')
	return output.Bytes(), nil
}

func canonicalGeneratedPackageSet(
	value installModel.CompiledPackageSet,
) (installModel.CompiledPackageSet, []byte, cryptoutil.Digest, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return installModel.CompiledPackageSet{}, nil, "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return installModel.CompiledPackageSet{}, nil, "", err
	}
	normalized, err := DecodeGeneratedPackageSet(canonical)
	if err != nil {
		return installModel.CompiledPackageSet{}, nil, "", err
	}
	return normalized, canonical, cryptoutil.DigestBytes(canonical), nil
}

func validateGeneratedEnvelope(value installModel.CompiledPackageSet) error {
	if value.Format != installModel.CompiledPackageSetFormat {
		return fmt.Errorf(
			"%w: generated package format %q",
			spec.ErrUnsupported,
			value.Format,
		)
	}
	if err := spec.ValidateRequiredText(
		"generated package set name",
		value.Name,
		spec.MaxKindBytes,
	); err != nil {
		return err
	}
	return value.Hydration.Validate()
}
