package toolcatalog

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/catalogtest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
	"github.com/flexigpt/flexigpt-app/internal/tool/llmtoolsadapter"
)

func TestGeneratedCatalogMatchesSources(t *testing.T) {
	goTools, err := llmtoolsadapter.New()
	if err != nil {
		t.Fatal(err)
	}

	registry, err := registration.NewLLMInterpretationRegistry()
	if err != nil {
		t.Fatalf("create Tool catalog interpretation registry: %v", err)
	}

	expected, err := Compile(
		t.Context(),
		t.TempDir(),
		goTools,
		registry,
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
		registry,
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
		"internal/artifactbuiltin/toolcatalog/catalog_generated.json",
		candidate,
	); err != nil {
		t.Fatal(err)
	}
}
