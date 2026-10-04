//go:build generated_catalog

package builtin_test

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin/catalogtest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/model/inferenceadapter/catalog"
	modelBuiltin "github.com/flexigpt/flexigpt-app/internal/model/store/builtin"
)

func TestGeneratedCatalogMatchesSources(t *testing.T) {
	prepared, err := catalog.PreparePackages(t.Context())
	if err != nil {
		t.Fatalf("prepare inference Model catalog packages: %v", err)
	}

	expected, err := modelBuiltin.Compile(
		t.Context(),
		t.TempDir(),
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

	reprepared, err := catalog.PreparePackages(t.Context())
	if err != nil {
		t.Fatalf("prepare inference Model catalog packages again: %v", err)
	}
	recompiled, err := modelBuiltin.Compile(
		t.Context(),
		t.TempDir(),
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

	_, actualErr := modelBuiltin.GeneratedCatalogSet()

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
		modelBuiltin.GeneratedCatalogFingerprint(),
		"internal/model/store/builtin/catalog_generated.json",
		candidate,
	); err != nil {
		t.Fatal(err)
	}
}
