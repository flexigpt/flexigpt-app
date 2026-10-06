//go:build generated_catalog

package modelcatalog_test

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/catalogtest"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/modelcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
	"github.com/flexigpt/flexigpt-app/internal/model/inferenceadapter/catalog"
)

func TestGeneratedCatalogMatchesSources(t *testing.T) {
	registry, err := registration.NewLLMInterpretationRegistry()
	if err != nil {
		t.Fatalf("create Model catalog interpretation registry: %v", err)
	}

	prepared, err := catalog.PreparePackages(t.Context(), registry)
	if err != nil {
		t.Fatalf("prepare inference Model catalog packages: %v", err)
	}

	expected, err := modelcatalog.Compile(
		t.Context(),
		t.TempDir(),
		registry,
		prepared,
	)
	if err != nil {
		t.Fatalf("compile generated Model packages: %v", err)
	}

	_, expectedFingerprint, err := install.CanonicalGeneratedPackageSet(
		expected,
	)
	if err != nil {
		t.Fatalf("fingerprint expected generated Model catalog: %v", err)
	}

	reprepared, err := catalog.PreparePackages(t.Context(), registry)
	if err != nil {
		t.Fatalf("prepare inference Model catalog packages again: %v", err)
	}
	recompiled, err := modelcatalog.Compile(
		t.Context(),
		t.TempDir(),
		registry,
		reprepared,
	)
	if err != nil {
		t.Fatalf("compile generated Model packages again: %v", err)
	}

	_, repeatedFingerprint, err := install.CanonicalGeneratedPackageSet(
		recompiled,
	)
	if err != nil {
		t.Fatalf("fingerprint repeated generated Model catalog: %v", err)
	}
	if expectedFingerprint != repeatedFingerprint {
		t.Fatalf(
			"generated Model catalog is nondeterministic: first %q, second %q",
			expectedFingerprint,
			repeatedFingerprint,
		)
	}

	_, actualErr := modelcatalog.GeneratedCatalogSet()

	candidate, err := catalogtest.CandidatePath(
		"catalog_generated.next.json",
	)
	if err != nil {
		t.Fatalf(
			"resolve generated Model catalog candidate path: %v",
			err,
		)
	}

	if err := catalogtest.AssertGeneratedPackageSetMatches(
		expected,
		actualErr,
		modelcatalog.GeneratedCatalogFingerprint(),
		"internal/artifactbuiltin/modelcatalog/catalog_generated.json",
		candidate,
	); err != nil {
		t.Fatal(err)
	}
}
