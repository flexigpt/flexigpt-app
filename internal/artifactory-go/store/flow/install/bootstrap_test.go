package install

import (
	"testing"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func TestBootstrapPlanChangedPackageScopesRehydratesAllPackagesAfterMarkerDrift(
	t *testing.T,
) {
	first := installModel.PackageHydrationKey{
		InstallerName: "catalog",
		Scope:         storeSpec.Locator("agent/first/unversioned"),
	}
	second := installModel.PackageHydrationKey{
		InstallerName: "catalog",
		Scope:         storeSpec.Locator("agent/second/unversioned"),
	}

	plan := bootstrapPlan{
		current: false,
		packages: []installModel.PackageHydration{
			{Key: first},
			{Key: second},
		},
		packageCurrent: map[installModel.PackageHydrationKey]bool{
			first:  true,
			second: true,
		},
	}

	scopes := plan.changedPackageScopes()
	if len(scopes) != 2 ||
		scopes[0] != first.Scope ||
		scopes[1] != second.Scope {
		t.Fatalf(
			"stale topology changed scopes = %v, want [%q %q]",
			scopes,
			first.Scope,
			second.Scope,
		)
	}

	plan.current = true
	plan.packageCurrent[second] = false
	scopes = plan.changedPackageScopes()
	if len(scopes) != 1 || scopes[0] != second.Scope {
		t.Fatalf(
			"current topology changed scopes = %v, want [%q]",
			scopes,
			second.Scope,
		)
	}
}
