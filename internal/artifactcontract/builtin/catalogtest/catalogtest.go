package catalogtest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func CandidatePath(
	filename string,
) (string, error) {
	if filename == "" {
		return "", errors.New("generated catalog candidate filename is empty")
	}

	_, sourceFile, _, found := runtime.Caller(1)
	if !found {
		return "", errors.New("resolve generated catalog candidate directory")
	}

	return filepath.Join(filepath.Dir(sourceFile), filename), nil
}

// AssertGeneratedPackageSetMatches compares only canonical fingerprints.
//
// It does not compare structs. It does not compare raw Definition.Body bytes.
// It does not compare pretty JSON file whitespace.
func AssertGeneratedPackageSetMatches(
	expected topology.CompiledPackageSet,
	actualErr error,
	actualFingerprint cryptoutil.Digest,
	generatedFile string,
	candidateFile string,
) error {
	normalized, expectedFingerprint, err := builtin.CanonicalGeneratedPackageSet(expected)
	if err != nil {
		return fmt.Errorf(
			"canonicalize expected generated package set: %w",
			err,
		)
	}

	if actualErr == nil && actualFingerprint == expectedFingerprint {
		return nil
	}

	candidate, err := builtin.RenderGeneratedPackageSetJSON(
		normalized,
	)
	if err != nil {
		return fmt.Errorf(
			"render generated package-set JSON: %w",
			err,
		)
	}

	roundTrip, err := builtin.DecodeGeneratedPackageSet(candidate)
	if err != nil {
		return fmt.Errorf(
			"decode generated package-set JSON candidate: %w",
			err,
		)
	}

	renderedAgain, err := builtin.RenderGeneratedPackageSetJSON(
		roundTrip,
	)
	if err != nil {
		return fmt.Errorf(
			"render generated package-set JSON candidate again: %w",
			err,
		)
	}
	if cryptoutil.DigestBytes(candidate) !=
		cryptoutil.DigestBytes(renderedAgain) {
		return errors.New("generated package-set JSON is not a stable render fixed point")
	}

	if err := os.MkdirAll(filepath.Dir(candidateFile), 0o755); err != nil {
		return fmt.Errorf(
			"create generated package-set candidate directory: %w",
			err,
		)
	}
	if err := os.WriteFile(candidateFile, candidate, 0o600); err != nil {
		return fmt.Errorf(
			"write generated package-set candidate %q: %w",
			candidateFile,
			err,
		)
	}

	detail := ""
	switch {
	case actualErr != nil:
		detail = fmt.Sprintf(
			"Checked-in generated package set cannot be loaded: %v\n\n",
			actualErr,
		)
	case actualFingerprint != expectedFingerprint:
		detail = fmt.Sprintf(
			"Checked-in generated package-set fingerprint is %q, expected %q.\n\n",
			actualFingerprint,
			expectedFingerprint,
		)
	}

	return fmt.Errorf(
		"generated built-in package set %q is stale\n\n"+
			"%s"+
			"Wrote stable formatted JSON candidate:\n"+
			"  %s\n\n"+
			"Replace:\n"+
			"  %s\n"+
			"with that candidate file",
		normalized.Name,
		detail,
		candidateFile,
		generatedFile,
	)
}
