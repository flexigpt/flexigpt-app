package mcpcatalog

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/catalogtest"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
)

func TestGeneratedCatalogMatchesSources(t *testing.T) {
	registry, err := registration.NewLLMInterpretationRegistry()
	if err != nil {
		t.Fatalf("create MCP catalog interpretation registry: %v", err)
	}

	expected, err := Compile(t.Context(), t.TempDir(), registry)
	if err != nil {
		t.Fatalf("compile embedded MCP packages: %v", err)
	}

	_, expectedFingerprint, err := installFlow.CanonicalGeneratedPackageSet(expected)
	if err != nil {
		t.Fatalf("fingerprint expected generated MCP catalog: %v", err)
	}

	recompiled, err := Compile(t.Context(), t.TempDir(), registry)
	if err != nil {
		t.Fatalf("compile embedded MCP packages again: %v", err)
	}

	_, recompiledFingerprint, err := installFlow.CanonicalGeneratedPackageSet(
		recompiled,
	)
	if err != nil {
		t.Fatalf(
			"fingerprint repeated generated MCP catalog: %v",
			err,
		)
	}

	if expectedFingerprint != recompiledFingerprint {
		t.Fatalf(
			"generated MCP catalog is nondeterministic: first %q, second %q",
			expectedFingerprint,
			recompiledFingerprint,
		)
	}

	actualFingerprint, actualErr := generatedCatalog.Fingerprint()

	candidate, err := catalogtest.CandidatePath(
		"catalog_generated.next.json",
	)
	if err != nil {
		t.Fatalf(
			"resolve generated MCP catalog candidate path: %v",
			err,
		)
	}

	if err := catalogtest.AssertGeneratedPackageSetMatches(
		expected,
		actualErr,
		actualFingerprint,
		"internal/artifactbuiltin/mcpcatalog/catalog_generated.json",
		candidate,
	); err != nil {
		t.Fatal(err)
	}
}
