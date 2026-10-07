//go:build generated_catalog

// Package modelcatalog is intentionally behind generated_catalog flag. This takes a lot of time to run and is run
// specially when needed.
package modelcatalog

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/catalogtest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
)

func TestGeneratedCatalogMatchesSources(t *testing.T) {
	registry, err := registration.NewLLMInterpretationRegistry()
	if err != nil {
		t.Fatalf("create Model catalog interpretation registry: %v", err)
	}

	prepared, err := modelruntime.PreparePackages(t.Context())
	if err != nil {
		t.Fatalf("prepare inference Model catalog packages: %v", err)
	}

	expected, err := Compile(
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

	reprepared, err := modelruntime.PreparePackages(t.Context())
	if err != nil {
		t.Fatalf("prepare inference Model catalog packages again: %v", err)
	}
	recompiled, err := Compile(
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

	actualFingerprint, actualErr := generatedCatalog.Fingerprint()

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
		actualFingerprint,
		"internal/artifactbuiltin/modelcatalog/catalog_generated.json",
		candidate,
	); err != nil {
		t.Fatal(err)
	}
}
