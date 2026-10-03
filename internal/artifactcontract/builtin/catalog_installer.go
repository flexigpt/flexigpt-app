package builtin

import (
	"context"
	"fmt"
	"slices"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type CatalogInstaller struct {
	registration installModel.CompiledRegistration
	hydrator     installModel.CompiledHydrationCoordinator
	scopes       []spec.Locator
}

func NewCatalogInstallerForSet(
	set installModel.CompiledPackageSet,
	hydrator installModel.CompiledHydrationCoordinator,
	lifecycle installModel.CompiledPackageLifecycle,
) (*CatalogInstaller, error) {
	if hydrator == nil {
		return nil, fmt.Errorf(
			"%w: generated catalog installer dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	scopes := make([]spec.Locator, 0, len(set.Packages))
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
		registration: installModel.CompiledRegistration{
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

func (i *CatalogInstaller) BuiltInPackageScopes() []spec.Locator {
	if i == nil {
		return nil
	}
	return append([]spec.Locator(nil), i.scopes...)
}

func (i *CatalogInstaller) DesiredHydration(
	ctx context.Context,
) (installModel.Hydration, error) {
	if i == nil {
		return installModel.Hydration{}, spec.ErrClosed
	}
	if err := installFlow.RequirePrivileged(ctx); err != nil {
		return installModel.Hydration{}, err
	}
	return i.registration.Set.Hydration, nil
}

func (i *CatalogInstaller) DesiredPackageHydrations(
	ctx context.Context,
) ([]installModel.PackageHydration, error) {
	if i == nil {
		return nil, spec.ErrClosed
	}
	if err := installFlow.RequirePrivileged(ctx); err != nil {
		return nil, err
	}

	hydration := i.registration.Set.Hydration
	output := make(
		[]installModel.PackageHydration,
		0,
		len(i.registration.Set.Packages),
	)
	for _, packageValue := range i.registration.Set.Packages {
		scope, err := packageValue.Address.Directory()
		if err != nil {
			return nil, err
		}
		output = append(output, installModel.PackageHydration{
			Key: installModel.PackageHydrationKey{
				InstallerName: hydration.InstallerName,
				Scope:         scope,
			},
			RootID:      hydration.RootID,
			SourceID:    hydration.SourceID,
			Fingerprint: packageValue.Fingerprint,
		})
	}
	return installModel.NormalizePackageHydrations(output)
}

func (i *CatalogInstaller) CompiledRegistration(
	ctx context.Context,
) (installModel.CompiledRegistration, error) {
	if i == nil {
		return installModel.CompiledRegistration{}, spec.ErrClosed
	}
	if err := installFlow.RequirePrivileged(ctx); err != nil {
		return installModel.CompiledRegistration{}, err
	}
	// Generated catalogs are immutable after construction. Avoid copying the
	// complete package payload before every bootstrap registration.
	return installModel.CompiledRegistration{
		Set:       i.registration.Set,
		Lifecycle: i.registration.Lifecycle,
	}, nil
}

func (i *CatalogInstaller) Ensure(
	ctx context.Context,
) error {
	if i == nil {
		return spec.ErrClosed
	}
	registration, err := i.CompiledRegistration(ctx)
	if err != nil {
		return err
	}
	if err := i.hydrator.RegisterCompiledPackages(
		ctx,
		[]installModel.CompiledRegistration{registration},
	); err != nil {
		return err
	}
	return i.hydrator.HydrateCompiledPackages(
		ctx,
		[]installModel.CompiledPackagePlan{{
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
	stale []installModel.PackageHydration,
) error {
	if i == nil {
		return spec.ErrClosed
	}
	registration, err := i.CompiledRegistration(ctx)
	if err != nil {
		return err
	}
	if err := i.hydrator.RegisterCompiledPackages(
		ctx,
		[]installModel.CompiledRegistration{registration},
	); err != nil {
		return err
	}
	return i.hydrator.HydrateCompiledPackages(
		ctx,
		[]installModel.CompiledPackagePlan{{
			Registration:    registration,
			TopologyCurrent: false,
			Changed:         i.BuiltInPackageScopes(),
			Stale:           append([]installModel.PackageHydration(nil), stale...),
		}},
	)
}

func (i *CatalogInstaller) FinalizeHydration(
	ctx context.Context,
) error {
	// HydrateCompiledPackages refreshes and verifies changed generated packages.
	// There is no runtime declaration resolver work left for this installer.
	return installFlow.RequirePrivileged(ctx)
}
