package builtin

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin/catalogtest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
)

func TestGeneratedCatalogMatchesSources(t *testing.T) {
	expected, err := Compile(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("compile embedded Skill packages: %v", err)
	}

	_, expectedFingerprint, err := install.CanonicalGeneratedPackageSet(expected)
	if err != nil {
		t.Fatalf("fingerprint expected generated Skill catalog: %v", err)
	}

	recompiled, err := Compile(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("compile embedded Skill packages again: %v", err)
	}

	_, recompiledFingerprint, err := install.CanonicalGeneratedPackageSet(
		recompiled,
	)
	if err != nil {
		t.Fatalf(
			"fingerprint repeated generated Skill catalog: %v",
			err,
		)
	}

	if expectedFingerprint != recompiledFingerprint {
		t.Fatalf(
			"generated Skill catalog is nondeterministic: first %q, second %q",
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
			"resolve generated Skill catalog candidate path: %v",
			err,
		)
	}

	if err := catalogtest.AssertGeneratedPackageSetMatches(
		expected,
		actualErr,
		GeneratedCatalogFingerprint(),
		"internal/skill/store/builtin/catalog_generated.json",
		candidate,
	); err != nil {
		t.Fatal(err)
	}
}
