package builtin

import (
	"context"
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
)

type CatalogInstaller struct {
	registration topology.CompiledRegistration
	hydrator     topology.CompiledHydrationCoordinator
	scopes       []model.Locator
}

func NewCatalogInstallerForSet(
	set topology.CompiledPackageSet,
	hydrator topology.CompiledHydrationCoordinator,
	lifecycle topology.CompiledPackageLifecycle,
) (*CatalogInstaller, error) {
	if hydrator == nil {
		return nil, fmt.Errorf(
			"%w: generated catalog installer dependencies are incomplete",
			model.ErrInvalid,
		)
	}

	scopes := make([]model.Locator, 0, len(set.Packages))
	for _, packageValue := range set.Packages {
		scope, err := packageValue.Address.Directory()
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)

	// `set` is transferred into this installer. Generated callers retain the
	// binary-owned immutable catalog, while the public GeneratedCatalogSet
	// helpers still return defensive clones for ordinary callers.
	return &CatalogInstaller{
		registration: topology.CompiledRegistration{
			Set:       set,
			Lifecycle: lifecycle,
		},
		hydrator: hydrator,
		scopes:   scopes,
	}, nil
}

func (i *CatalogInstaller) BuiltInName() string {
	if i == nil {
		return ""
	}
	return i.registration.Set.Hydration.InstallerName
}

func (i *CatalogInstaller) BuiltInPackageScopes() []model.Locator {
	if i == nil {
		return nil
	}
	return append([]model.Locator(nil), i.scopes...)
}

func (i *CatalogInstaller) DesiredHydration(
	ctx context.Context,
) (topology.Hydration, error) {
	if i == nil {
		return topology.Hydration{}, model.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return topology.Hydration{}, err
	}
	return i.registration.Set.Hydration, nil
}

func (i *CatalogInstaller) DesiredPackageHydrations(
	ctx context.Context,
) ([]topology.PackageHydration, error) {
	if i == nil {
		return nil, model.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return nil, err
	}

	hydration := i.registration.Set.Hydration
	output := make(
		[]topology.PackageHydration,
		0,
		len(i.registration.Set.Packages),
	)
	for _, packageValue := range i.registration.Set.Packages {
		scope, err := packageValue.Address.Directory()
		if err != nil {
			return nil, err
		}
		output = append(output, topology.PackageHydration{
			Key: topology.PackageHydrationKey{
				InstallerName: hydration.InstallerName,
				Scope:         scope,
			},
			RootID:      hydration.RootID,
			SourceID:    hydration.SourceID,
			Fingerprint: packageValue.Fingerprint,
		})
	}
	return topology.NormalizePackageHydrations(output)
}

func (i *CatalogInstaller) CompiledRegistration(
	ctx context.Context,
) (topology.CompiledRegistration, error) {
	if i == nil {
		return topology.CompiledRegistration{}, model.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return topology.CompiledRegistration{}, err
	}
	// Generated catalogs are immutable after construction. Avoid copying the
	// complete package payload before every bootstrap registration.
	return topology.CompiledRegistration{
		Set:       i.registration.Set,
		Lifecycle: i.registration.Lifecycle,
	}, nil
}

func (i *CatalogInstaller) Ensure(
	ctx context.Context,
) error {
	if i == nil {
		return model.ErrClosed
	}
	registration, err := i.CompiledRegistration(ctx)
	if err != nil {
		return err
	}
	if err := i.hydrator.RegisterCompiledPackages(
		ctx,
		[]topology.CompiledRegistration{registration},
	); err != nil {
		return err
	}
	return i.hydrator.HydrateCompiledPackages(
		ctx,
		[]topology.CompiledPackagePlan{{
			Registration:    registration,
			TopologyCurrent: false,
			Changed:         i.BuiltInPackageScopes(),
		}},
	)
}

func (i *CatalogInstaller) EnsureHydration(
	ctx context.Context,
	current bool,
) error {
	if current {
		return nil
	}
	return i.Ensure(ctx)
}

// EnsurePackageHydration is retained for direct trusted repair calls. Normal
// bootstrap batches all generated package plans through
// CompiledPackageInstaller instead.
func (i *CatalogInstaller) EnsurePackageHydration(
	ctx context.Context,
	_ bool,
	stale []topology.PackageHydration,
) error {
	if i == nil {
		return model.ErrClosed
	}
	registration, err := i.CompiledRegistration(ctx)
	if err != nil {
		return err
	}
	if err := i.hydrator.RegisterCompiledPackages(
		ctx,
		[]topology.CompiledRegistration{registration},
	); err != nil {
		return err
	}
	return i.hydrator.HydrateCompiledPackages(
		ctx,
		[]topology.CompiledPackagePlan{{
			Registration:    registration,
			TopologyCurrent: false,
			Changed:         i.BuiltInPackageScopes(),
			Stale:           append([]topology.PackageHydration(nil), stale...),
		}},
	)
}

func (i *CatalogInstaller) FinalizeHydration(
	ctx context.Context,
) error {
	// HydrateCompiledPackages refreshes and verifies changed generated packages.
	// There is no runtime declaration resolver work left for this installer.
	return installerapi.RequirePrivileged(ctx)
}
