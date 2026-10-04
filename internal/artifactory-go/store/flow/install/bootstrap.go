package install

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Installer supplies application-owned protected content. These contracts
// describe installation roles, not a universal Store plugin registration.
type Installer interface {
	BuiltInName() string
	BuiltInPackageScopes() []spec.Locator
	Ensure(ctx context.Context) error
}

type HydrationInstaller interface {
	Installer
	DesiredHydration(ctx context.Context) (installModel.Hydration, error)
	EnsureHydration(ctx context.Context, current bool) error
	FinalizeHydration(ctx context.Context) error
}

type PackageHydrationInstaller interface {
	HydrationInstaller
	DesiredPackageHydrations(ctx context.Context) ([]installModel.PackageHydration, error)
	EnsurePackageHydration(
		ctx context.Context,
		topologyCurrent bool,
		stale []installModel.PackageHydration,
	) error
}

type CompiledPackageInstaller interface {
	PackageHydrationInstaller
	CompiledRegistration(ctx context.Context) (installModel.CompiledRegistration, error)
}

type bootstrapInstaller struct {
	name      string
	scopes    []spec.Locator
	installer Installer
}

type bootstrapPlan struct {
	entry        bootstrapInstaller
	hydration    HydrationInstaller
	packageOwner PackageHydrationInstaller

	desired      installModel.Hydration
	packages     []installModel.PackageHydration
	registration *installModel.CompiledRegistration

	current        bool
	packageCurrent map[installModel.PackageHydrationKey]bool
	stale          []installModel.PackageHydration
}

// Bootstrap owns complete-plan preparation, shared-Root reset sequencing,
// Source-batched compiled installation, finalization, and marker commits.
//
// Construct it at application composition and run it before ordinary Store
// writers are started. Its mutex serializes runs of this bootstrap instance;
// it is not a substitute for deployment's startup exclusivity protocol.
type Bootstrap struct {
	declaration installModel.Declaration
	coordinator API
	installers  []bootstrapInstaller
	mu          sync.Mutex
}

func NewBootstrap(
	declaration installModel.Declaration,
	coordinator API,
	installers ...Installer,
) (*Bootstrap, error) {
	if coordinator == nil {
		return nil, fmt.Errorf("%w: bootstrap installation coordinator is nil", spec.ErrInvalid)
	}
	if err := declaration.Validate(); err != nil {
		return nil, err
	}

	owned := declaration
	owned.Sources = slices.Clone(declaration.Sources)
	for index := range owned.Sources {
		owned.Sources[index].Config = slices.Clone(declaration.Sources[index].Config)
		owned.Sources[index].Discovery = declaration.Sources[index].Discovery.Clone()
	}

	entries := make([]bootstrapInstaller, 0, len(installers))
	names := make(map[string]struct{}, len(installers))
	for index, installer := range installers {
		if installer == nil {
			return nil, fmt.Errorf("%w: installer %d is nil", spec.ErrInvalid, index)
		}
		name := installer.BuiltInName()
		if err := installModel.ValidateHydrationInstallerName(name); err != nil {
			return nil, err
		}
		if _, duplicate := names[name]; duplicate {
			return nil, fmt.Errorf("%w: duplicate installer %q", spec.ErrConflict, name)
		}
		names[name] = struct{}{}
		scopes, err := normalizePackageScopes(installer.BuiltInPackageScopes())
		if err != nil {
			return nil, err
		}
		for _, previous := range entries {
			for _, scope := range scopes {
				for _, existing := range previous.scopes {
					if packageScopesOverlap(scope, existing) {
						return nil, fmt.Errorf(
							"%w: installer %q scope %q overlaps installer %q scope %q",
							spec.ErrConflict,
							name,
							scope,
							previous.name,
							existing,
						)
					}
				}
			}
		}
		entries = append(entries, bootstrapInstaller{
			name: name, scopes: scopes, installer: installer,
		})
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].name < entries[right].name
	})
	return &Bootstrap{
		declaration: owned,
		coordinator: coordinator,
		installers:  entries,
	}, nil
}

func (b *Bootstrap) Ensure(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: bootstrap context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Obtain and validate every application-supplied plan before preparation
	// is allowed to reset a Root or touch physical package content.
	plans, err := b.preparePlans(ctx)
	if err != nil {
		return err
	}

	registrations := make([]installModel.CompiledRegistration, 0)
	desired := make([]installModel.Hydration, 0)
	packageInstallers := make([]string, 0)
	desiredPackages := make([]installModel.PackageHydration, 0)
	for _, plan := range plans {
		if plan.hydration != nil {
			desired = append(desired, plan.desired)
		}
		if plan.packageOwner != nil {
			packageInstallers = append(packageInstallers, plan.entry.name)
			desiredPackages = append(desiredPackages, plan.packages...)
		}
		if plan.registration != nil {
			registrations = append(registrations, *plan.registration)
		}
	}

	// Definition-owned trusted admission must also succeed before reset.
	if len(registrations) != 0 {
		if err := b.coordinator.RegisterCompiledPackages(ctx, registrations); err != nil {
			return fmt.Errorf("register compiled installation evidence: %w", err)
		}
	}

	current := map[string]bool{}
	if len(desired) != 0 {
		current, err = b.coordinator.PrepareTopologyHydrations(ctx, desired)
		if err != nil {
			return fmt.Errorf("prepare topology hydrations: %w", err)
		}
	}

	packagePreparation := installModel.PackageHydrationPreparation{
		Current: map[installModel.PackageHydrationKey]bool{},
	}
	if len(packageInstallers) != 0 {
		packagePreparation, err = b.coordinator.PrepareTopologyPackageHydrations(
			ctx,
			packageInstallers,
			desiredPackages,
		)
		if err != nil {
			return fmt.Errorf("prepare package hydrations: %w", err)
		}
	}

	for index := range plans {
		plan := &plans[index]
		if plan.hydration == nil {
			continue
		}
		value, found := current[plan.entry.name]
		if !found {
			return fmt.Errorf(
				"%w: hydration coordinator omitted installer %q",
				spec.ErrInvalid,
				plan.entry.name,
			)
		}
		plan.current = value
		plan.packageCurrent = make(map[installModel.PackageHydrationKey]bool, len(plan.packages))
		for _, value := range plan.packages {
			plan.packageCurrent[value.Key] = plan.current &&
				packagePreparation.CurrentFor(value.Key)
		}
		for _, stale := range packagePreparation.Stale {
			if stale.Key.InstallerName == plan.entry.name {
				plan.stale = append(plan.stale, stale)
			}
		}
	}

	if _, err := b.coordinator.EnsureProtectedTopology(ctx, b.declaration); err != nil {
		return fmt.Errorf("ensure protected topology: %w", err)
	}

	physicalWork := false
	compiledPlans := make([]installModel.CompiledPackagePlan, 0)
	for _, plan := range plans {
		if plan.registration != nil {
			if !plan.needsPackageMutation() {
				continue
			}
			compiled := installModel.CompiledPackagePlan{
				Registration:    *plan.registration,
				TopologyCurrent: plan.current,
				Stale:           slices.Clone(plan.stale),
			}
			for _, value := range plan.packages {
				if !plan.packageCurrent[value.Key] {
					compiled.Changed = append(compiled.Changed, value.Key.Scope)
				}
			}
			compiledPlans = append(compiledPlans, compiled)
			continue
		}

		switch {
		case plan.packageOwner != nil:
			if !plan.needsPackageMutation() {
				continue
			}
			physicalWork = true
			err = plan.packageOwner.EnsurePackageHydration(ctx, plan.current, plan.stale)
		case plan.hydration != nil:
			if plan.current {
				continue
			}
			physicalWork = true
			err = plan.hydration.EnsureHydration(ctx, false)
		default:
			physicalWork = true
			err = plan.entry.installer.Ensure(ctx)
		}
		if err != nil {
			return fmt.Errorf("ensure installer %q: %w", plan.entry.name, err)
		}
	}

	if len(compiledPlans) != 0 {
		if err := b.coordinator.HydrateCompiledPackages(ctx, compiledPlans); err != nil {
			return fmt.Errorf("hydrate compiled packages: %w", err)
		}
		physicalWork = true
	}

	// A later installer may advance a Source shared with an earlier one.
	// Finalization therefore follows all physical publication.
	if physicalWork {
		for _, plan := range plans {
			if plan.hydration == nil {
				continue
			}
			if err := plan.hydration.FinalizeHydration(ctx); err != nil {
				return fmt.Errorf("finalize installer %q: %w", plan.entry.name, err)
			}
		}
	}

	for _, plan := range plans {
		if plan.hydration != nil && !plan.current {
			if err := b.coordinator.CommitTopologyHydration(ctx, plan.desired); err != nil {
				return fmt.Errorf("commit installer %q hydration: %w", plan.entry.name, err)
			}
		}
		for _, value := range plan.packages {
			if plan.packageCurrent[value.Key] {
				continue
			}
			if err := b.coordinator.CommitTopologyPackageHydration(ctx, value); err != nil {
				return fmt.Errorf("commit package %q hydration: %w", value.Key.Scope, err)
			}
		}
		for _, stale := range plan.stale {
			if err := b.coordinator.DeleteTopologyPackageHydration(ctx, stale); err != nil {
				return fmt.Errorf("delete stale package hydration: %w", err)
			}
		}
	}
	return nil
}

func (b *Bootstrap) preparePlans(ctx context.Context) ([]bootstrapPlan, error) {
	output := make([]bootstrapPlan, 0, len(b.installers))
	for _, entry := range b.installers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		plan := bootstrapPlan{entry: entry}
		plan.hydration, _ = entry.installer.(HydrationInstaller)
		plan.packageOwner, _ = entry.installer.(PackageHydrationInstaller)

		if plan.hydration != nil {
			desired, err := plan.hydration.DesiredHydration(ctx)
			if err != nil {
				return nil, fmt.Errorf("installer %q desired hydration: %w", entry.name, err)
			}
			if err := desired.Validate(); err != nil {
				return nil, err
			}
			if desired.InstallerName != entry.name ||
				desired.RootID != b.declaration.Root.ID {
				return nil, fmt.Errorf("%w: installer %q hydration identity differs", spec.ErrInvalid, entry.name)
			}
			sourceFound := false
			for _, source := range b.declaration.Sources {
				if source.ID == desired.SourceID {
					sourceFound = true
					break
				}
			}
			if !sourceFound {
				return nil, fmt.Errorf("%w: installer %q names an undeclared Source", spec.ErrInvalid, entry.name)
			}
			plan.desired = desired
		}

		if plan.packageOwner != nil {
			values, err := plan.packageOwner.DesiredPackageHydrations(ctx)
			if err != nil {
				return nil, err
			}
			values, err = installModel.NormalizePackageHydrations(values)
			if err != nil {
				return nil, err
			}
			for _, value := range values {
				if value.Key.InstallerName != entry.name ||
					value.RootID != plan.desired.RootID ||
					value.SourceID != plan.desired.SourceID ||
					!ownsPackageScope(entry.scopes, value.Key.Scope) {
					return nil, fmt.Errorf(
						"%w: installer %q returned an unowned package hydration",
						spec.ErrInvalid,
						entry.name,
					)
				}
			}
			plan.packages = values
		}

		if compiled, supported := entry.installer.(CompiledPackageInstaller); supported {
			registration, err := compiled.CompiledRegistration(ctx)
			if err != nil {
				return nil, err
			}
			registration.Set = registration.Set.Clone()
			if err := validateCompiledSet(registration.Set); err != nil {
				return nil, err
			}
			if registration.Set.Hydration != plan.desired ||
				len(registration.Set.Packages) != len(plan.packages) {
				return nil, fmt.Errorf(
					"%w: compiled installer %q has inconsistent hydration",
					spec.ErrInvalid,
					entry.name,
				)
			}
			byScope := make(map[spec.Locator]installModel.PackageHydration, len(plan.packages))
			for _, value := range plan.packages {
				byScope[value.Key.Scope] = value
			}
			for _, value := range registration.Set.Packages {
				scope, err := value.Address.Directory()
				if err != nil {
					return nil, err
				}
				marker, found := byScope[scope]
				if !found || marker.Fingerprint != value.Fingerprint {
					return nil, fmt.Errorf("%w: compiled package hydration differs", spec.ErrInvalid)
				}
			}
			plan.registration = &registration
		}
		output = append(output, plan)
	}
	return output, nil
}

func (p bootstrapPlan) needsPackageMutation() bool {
	if !p.current || len(p.stale) != 0 {
		return true
	}
	for _, current := range p.packageCurrent {
		if !current {
			return true
		}
	}
	return false
}

func normalizePackageScopes(values []spec.Locator) ([]spec.Locator, error) {
	output := slices.Clone(values)
	for _, value := range output {
		if err := value.ValidatePortable(false); err != nil {
			return nil, err
		}
	}
	slices.Sort(output)
	for index, scope := range output {
		for _, previous := range output[:index] {
			if packageScopesOverlap(previous, scope) {
				return nil, fmt.Errorf(
					"%w: overlapping package scopes %q and %q",
					spec.ErrConflict,
					previous,
					scope,
				)
			}
		}
	}
	return output, nil
}

func packageScopesOverlap(left, right spec.Locator) bool {
	return left == right ||
		strings.HasPrefix(string(left), string(right)+"/") ||
		strings.HasPrefix(string(right), string(left)+"/")
}

func ownsPackageScope(scopes []spec.Locator, requested spec.Locator) bool {
	for _, scope := range scopes {
		if requested == scope || strings.HasPrefix(string(requested), string(scope)+"/") {
			return true
		}
	}
	return false
}
