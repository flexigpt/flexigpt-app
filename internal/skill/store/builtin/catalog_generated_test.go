package builtin

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin/catalogtest"
)

func TestGeneratedCatalogMatchesSources(t *testing.T) {
	expected, err := Compile(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("compile embedded Skill packages: %v", err)
	}

	_, expectedFingerprint, err := builtin.CanonicalGeneratedPackageSet(expected)
	if err != nil {
		t.Fatalf("fingerprint expected generated Skill catalog: %v", err)
	}

	recompiled, err := Compile(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("compile embedded Skill packages again: %v", err)
	}

	_, recompiledFingerprint, err := builtin.CanonicalGeneratedPackageSet(
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
