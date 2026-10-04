package builtin

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin/catalogtest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/tool/llmtoolsadapter"
)

func TestGeneratedCatalogMatchesSources(t *testing.T) {
	goTools, err := llmtoolsadapter.New()
	if err != nil {
		t.Fatal(err)
	}

	expected, err := Compile(
		t.Context(),
		t.TempDir(),
		goTools,
	)
	if err != nil {
		t.Fatalf("compile embedded Tool packages: %v", err)
	}

	_, expectedFingerprint, err := install.CanonicalGeneratedPackageSet(expected)
	if err != nil {
		t.Fatalf("fingerprint expected generated Tool catalog: %v", err)
	}

	recompiled, err := Compile(
		t.Context(),
		t.TempDir(),
		goTools,
	)
	if err != nil {
		t.Fatalf("compile embedded Tool packages again: %v", err)
	}

	_, recompiledFingerprint, err := install.CanonicalGeneratedPackageSet(
		recompiled,
	)
	if err != nil {
		t.Fatalf(
			"fingerprint repeated generated Tool catalog: %v",
			err,
		)
	}

	if expectedFingerprint != recompiledFingerprint {
		t.Fatalf(
			"generated Tool catalog is nondeterministic: first %q, second %q",
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
			"resolve generated Tool catalog candidate path: %v",
			err,
		)
	}

	if err := catalogtest.AssertGeneratedPackageSetMatches(
		expected,
		actualErr,
		GeneratedCatalogFingerprint(),
		"internal/tool/store/builtin/catalog_generated.json",
		candidate,
	); err != nil {
		t.Fatal(err)
	}
}
