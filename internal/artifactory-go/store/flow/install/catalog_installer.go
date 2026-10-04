package install

import (
	"context"
	"fmt"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// CatalogInstaller installs one application-supplied compiled package set.
// It owns no built-in content, declaration language, or family package layout.
type CatalogInstaller struct {
	registration installModel.CompiledRegistration
	hydrator     installModel.CompiledHydrationCoordinator
	scopes       []spec.Locator
}

func NewCatalogInstaller(
	set installModel.CompiledPackageSet,
	hydrator installModel.CompiledHydrationCoordinator,
	lifecycle installModel.CompiledPackageLifecycle,
) (*CatalogInstaller, error) {
	if hydrator == nil {
		return nil, fmt.Errorf("%w: compiled hydration coordinator is nil", spec.ErrInvalid)
	}
	owned := set.Clone()
	if err := validateCompiledSet(owned); err != nil {
		return nil, err
	}
	scopes := make([]spec.Locator, 0, len(owned.Packages))
	for _, value := range owned.Packages {
		scope, err := value.Address.Directory()
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
	}
	scopes, err := normalizePackageScopes(scopes)
	if err != nil {
		return nil, err
	}
	return &CatalogInstaller{
		registration: installModel.CompiledRegistration{
			Set:       owned,
			Lifecycle: lifecycle,
		},
		hydrator: hydrator,
		scopes:   scopes,
	}, nil
}

func (i *CatalogInstaller) BuiltInName() string {
	return i.registration.Set.Hydration.InstallerName
}

func (i *CatalogInstaller) BuiltInPackageScopes() []spec.Locator {
	return append([]spec.Locator(nil), i.scopes...)
}

func (i *CatalogInstaller) DesiredHydration(
	ctx context.Context,
) (installModel.Hydration, error) {
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return installModel.Hydration{}, err
	}
	return i.registration.Set.Hydration, nil
}

func (i *CatalogInstaller) DesiredPackageHydrations(
	ctx context.Context,
) ([]installModel.PackageHydration, error) {
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return nil, err
	}
	hydration := i.registration.Set.Hydration
	output := make([]installModel.PackageHydration, 0, len(i.registration.Set.Packages))
	for _, value := range i.registration.Set.Packages {
		scope, err := value.Address.Directory()
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
			Fingerprint: value.Fingerprint,
		})
	}
	return installModel.NormalizePackageHydrations(output)
}

func (i *CatalogInstaller) CompiledRegistration(
	ctx context.Context,
) (installModel.CompiledRegistration, error) {
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return installModel.CompiledRegistration{}, err
	}
	return installModel.CompiledRegistration{
		Set:       i.registration.Set.Clone(),
		Lifecycle: i.registration.Lifecycle,
	}, nil
}

func (i *CatalogInstaller) Ensure(ctx context.Context) error {
	return i.EnsurePackageHydration(ctx, false, nil)
}

func (i *CatalogInstaller) EnsureHydration(ctx context.Context, current bool) error {
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return err
	}
	if current {
		return nil
	}
	return i.Ensure(ctx)
}

// EnsurePackageHydration supports explicit trusted repair. Bootstrap instead
// combines all compiled installers into one set of Source batches.
func (i *CatalogInstaller) EnsurePackageHydration(
	ctx context.Context,
	topologyCurrent bool,
	stale []installModel.PackageHydration,
) error {
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
			TopologyCurrent: topologyCurrent,
			Changed:         i.BuiltInPackageScopes(),
			Stale:           append([]installModel.PackageHydration(nil), stale...),
		}},
	)
}

func (*CatalogInstaller) FinalizeHydration(ctx context.Context) error {
	return root.RequireInstallerPrivilege(ctx)
}
