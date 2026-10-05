package agentcatalog

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/catalogtest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
)

func TestGeneratedCatalogMatchesSources(t *testing.T) {
	registry, err := registration.NewLLMInterpretationRegistry()
	if err != nil {
		t.Fatalf("create Agent catalog interpretation registry: %v", err)
	}

	expected, err := Compile(t.Context(), t.TempDir(), registry)
	if err != nil {
		t.Fatalf("compile embedded Agent packages: %v", err)
	}

	_, expectedFingerprint, err := install.CanonicalGeneratedPackageSet(expected)
	if err != nil {
		t.Fatalf("fingerprint expected generated Agent catalog: %v", err)
	}

	recompiled, err := Compile(t.Context(), t.TempDir(), registry)
	if err != nil {
		t.Fatalf("compile embedded Agent packages again: %v", err)
	}

	_, recompiledFingerprint, err := install.CanonicalGeneratedPackageSet(recompiled)
	if err != nil {
		t.Fatalf(
			"fingerprint repeated generated Agent catalog: %v",
			err,
		)
	}

	if expectedFingerprint != recompiledFingerprint {
		t.Fatalf(
			"generated Agent catalog is nondeterministic: first %q, second %q",
			expectedFingerprint,
			recompiledFingerprint,
		)
	}

	_, actualErr := GeneratedCatalogSet()

	candidate, err := catalogtest.CandidatePath(
		"catalog_generated.next.json",
	)
	if err != nil {
		t.Fatalf(
			"resolve generated Agent catalog candidate path: %v",
			err,
		)
	}

	if err := catalogtest.AssertGeneratedPackageSetMatches(
		expected,
		actualErr,
		GeneratedCatalogFingerprint(),
		"internal/artifactbuiltin/agentcatalog/catalog_generated.json",
		candidate,
	); err != nil {
		t.Fatal(err)
	}
}
