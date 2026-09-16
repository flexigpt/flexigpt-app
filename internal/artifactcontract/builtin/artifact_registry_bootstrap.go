package builtin

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
)

// Installer is implemented by one artifact-family-owned built-in installerapi.
// The generic built-in layer deliberately does not inspect package contents,
// artifact definitions, or artifact-specific manifests.
type Installer interface {
	BuiltInName() string
	BuiltInPackageScopes() []basespec.Locator
	Ensure(ctx context.Context) error
}

// HydrationInstaller supplies artifact-family desired state and receives the
// result of generic hydration comparison. It does not read or write hydration
// markers and it does not reset topology roots itself.
type HydrationInstaller interface {
	Installer

	DesiredHydration(ctx context.Context) (topology.Hydration, error)

	// EnsureHydration creates or repairs this artifact family's desired
	// topology and package state. It may publish managed Source content.
	EnsureHydration(ctx context.Context, current bool) error

	// FinalizeHydration runs after package publication has completed. The
	// bootstrapper may coalesce calls for installers sharing one Source. It
	// must refresh source-backed Artifact state against
	// the final shared Source generation. It must not mutate
	// managed package content or topology.
	FinalizeHydration(ctx context.Context) error
}

type PackageHydrationInstaller interface {
	HydrationInstaller

	DesiredPackageHydrations(
		ctx context.Context,
	) ([]topology.PackageHydration, error)

	EnsurePackageHydration(
		ctx context.Context,
		topologyCurrent bool,
		current map[topology.PackageHydrationKey]bool,
		stale []topology.PackageHydration,
	) error
}

type preparedHydration struct {
	installer      string
	desired        topology.Hydration
	current        bool
	packages       []topology.PackageHydration
	stale          []topology.PackageHydration
	packageCurrent map[topology.PackageHydrationKey]bool
}

type registeredInstaller struct {
	name      string
	installer Installer
}

// BootstrapRegistry owns application-level built-in installation order and
// shared topology. It is intentionally unaware of Skills, MCPs, or any other
// artifact format.
type BootstrapRegistry struct {
	declaration topology.Declaration
	topology    topology.Ensurer
	hydrator    topology.HydrationCoordinator

	mu         sync.RWMutex
	ensureMu   sync.Mutex
	installers map[string]Installer
	scopes     map[basespec.Locator]string
}

func NewBootstrapRegistry(
	declaration topology.Declaration,
	ensurer topology.Ensurer,
	hydrator topology.HydrationCoordinator,
) (*BootstrapRegistry, error) {
	if ensurer == nil || hydrator == nil {
		return nil, fmt.Errorf(
			"%w: built-in bootstrap dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if err := declaration.Validate(); err != nil {
		return nil, err
	}
	return &BootstrapRegistry{
		declaration: declaration,
		topology:    ensurer,
		hydrator:    hydrator,
		installers:  map[string]Installer{},
		scopes:      map[basespec.Locator]string{},
	}, nil
}

func (r *BootstrapRegistry) Register(inst Installer) error {
	if r == nil {
		return fmt.Errorf("%w: built-in bootstrap registry is nil", basespec.ErrInvalid)
	}
	if inst == nil {
		return fmt.Errorf("%w: built-in installer is nil", basespec.ErrInvalid)
	}

	name := inst.BuiltInName()
	if err := topology.ValidateHydrationInstallerName(name); err != nil {
		return fmt.Errorf("built-in installer name: %w", err)
	}
	scopes, err := normalizePackageScopes(inst.BuiltInPackageScopes())
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.installers[name]; exists {
		return fmt.Errorf(
			"%w: built-in installer %q is already registered",
			basespec.ErrConflict,
			name,
		)
	}

	existingScopes := make([]basespec.Locator, 0, len(r.scopes))
	for scope := range r.scopes {
		existingScopes = append(existingScopes, scope)
	}
	slices.Sort(existingScopes)
	for _, scope := range scopes {
		for _, existing := range existingScopes {
			if !packageScopesOverlap(scope, existing) {
				continue
			}
			return fmt.Errorf(
				"%w: built-in installer %q package scope %q overlaps %q owned by %s",
				basespec.ErrConflict,
				name,
				scope,
				existing,
				r.scopes[existing],
			)
		}
	}

	r.installers[name] = inst
	for _, scope := range scopes {
		r.scopes[scope] = name
	}
	return nil
}

func (r *BootstrapRegistry) Ensure(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("%w: built-in bootstrap registry is nil", basespec.ErrInvalid)
	}
	if ctx == nil {
		return fmt.Errorf("%w: built-in bootstrap context is nil", basespec.ErrInvalid)
	}

	// One bootstrapper owns one protected topology in this process. Serializing
	// hydration prevents concurrent callers from racing resets, publication,
	// final refresh, and hydration-marker commits.
	r.ensureMu.Lock()
	defer r.ensureMu.Unlock()

	r.mu.RLock()
	entries := make([]registeredInstaller, 0, len(r.installers))
	for name, installer := range r.installers {
		entries = append(entries, registeredInstaller{
			name:      name,
			installer: installer,
		})
	}
	r.mu.RUnlock()

	sort.Slice(entries, func(left, right int) bool {
		return entries[left].name < entries[right].name
	})

	ctx = installerapi.WithPrivilege(ctx)
	prepared := make([]preparedHydration, 0, len(entries))

	for _, entry := range entries {
		inst, supported := entry.installer.(HydrationInstaller)
		if !supported {
			continue
		}
		desired, err := inst.DesiredHydration(ctx)
		if err != nil {
			return fmt.Errorf(
				"build desired hydration for installer %q: %w",
				entry.name,
				err,
			)
		}
		if desired.InstallerName != entry.name {
			return fmt.Errorf(
				"%w: built-in installer %q returned hydration name %q",
				basespec.ErrInvalid,
				entry.name,
				desired.InstallerName,
			)
		}
		prepared = append(prepared, preparedHydration{
			installer: entry.name,
			desired:   desired,
		})
	}

	packageCoordinator, packageAware := r.hydrator.(topology.PackageHydrationCoordinator)
	packageInstallerNames := make(
		[]string,
		0,
	)
	if packageAware {

		desiredPackages := make([]topology.PackageHydration, 0)
		for _, entry := range entries {
			installer, supported := entry.installer.(PackageHydrationInstaller)
			if !supported {
				continue
			}
			packageInstallerNames = append(packageInstallerNames, entry.name)
			values, err := installer.DesiredPackageHydrations(ctx)
			if err != nil {
				return fmt.Errorf(
					"build package hydration for installer %q: %w",
					entry.name,
					err,
				)
			}
			desiredPackages = append(desiredPackages, values...)
			for index := range prepared {
				if prepared[index].installer == entry.name {
					prepared[index].packages = values
					break
				}
			}
		}
		if _, err := topology.NormalizePackageHydrations(desiredPackages); err != nil {
			return err
		}
	}

	if len(prepared) != 0 {
		desiredValues := make([]topology.Hydration, 0, len(prepared))
		for _, value := range prepared {
			desiredValues = append(desiredValues, value.desired)
		}
		currentByInstaller, err := r.hydrator.PrepareTopologyHydrations(
			ctx,
			desiredValues,
		)
		if err != nil {
			return fmt.Errorf(
				"prepare topology hydrations: %w",
				err,
			)
		}
		for index := range prepared {
			current, found := currentByInstaller[prepared[index].installer]
			if !found {
				return fmt.Errorf(
					"%w: hydration coordinator omitted installer %q",
					basespec.ErrInvalid,
					prepared[index].installer,
				)
			}
			prepared[index].current = current
		}
	}
	if packageAware {
		desiredPackages := make([]topology.PackageHydration, 0)
		for _, value := range prepared {
			desiredPackages = append(desiredPackages, value.packages...)
		}
		preparation, err := packageCoordinator.PrepareTopologyPackageHydrations(
			ctx,
			packageInstallerNames,
			desiredPackages,
		)
		if err != nil {
			return fmt.Errorf("prepare package hydrations: %w", err)
		}
		for index := range prepared {
			prepared[index].packageCurrent = make(
				map[topology.PackageHydrationKey]bool,
			)
			for _, value := range prepared[index].packages {
				prepared[index].packageCurrent[value.Key] = preparation.CurrentFor(value.Key)
			}
			for _, stale := range preparation.Stale {
				if stale.Key.InstallerName != prepared[index].installer {
					continue
				}
				prepared[index].stale = append(
					prepared[index].stale,
					stale,
				)
			}
			if !prepared[index].current {
				for key := range prepared[index].packageCurrent {
					prepared[index].packageCurrent[key] = false
				}
			}
		}
	}

	if _, err := r.topology.EnsureProtectedTopology(
		ctx,
		r.declaration,
	); err != nil {
		return fmt.Errorf("ensure built-in topology: %w", err)
	}
	for _, entry := range entries {
		hydrated, supported := entry.installer.(HydrationInstaller)
		if packageInstaller, packageSupported := entry.installer.(PackageHydrationInstaller); packageSupported &&
			packageAware {
			var preparedValue preparedHydration
			for _, value := range prepared {
				if value.installer == entry.name {
					preparedValue = value
					break
				}
			}
			if err := packageInstaller.EnsurePackageHydration(
				ctx,
				preparedValue.current,
				preparedValue.packageCurrent,
				preparedValue.stale,
			); err != nil {
				return fmt.Errorf("ensure built-in installer %q: %w", entry.name, err)
			}
			continue
		}
		if supported {
			var current bool
			for _, value := range prepared {
				if value.installer == entry.name {
					current = value.current
					break
				}
			}
			if err := hydrated.EnsureHydration(ctx, current); err != nil {
				return fmt.Errorf(
					"ensure built-in installer %q: %w",
					entry.name,
					err,
				)
			}
			continue
		}
		if err := entry.installer.Ensure(ctx); err != nil {
			return fmt.Errorf(
				"ensure built-in installer %q: %w",
				entry.name,
				err,
			)
		}
	}

	// Installers can share a protected managed Source. A later installer may
	// advance the shared Source revision after an earlier installer refreshed
	// its Artifact state. Refresh every hydration-aware installer only after
	// all package publication has completed.
	desiredByInstaller := make(
		map[string]topology.Hydration,
		len(prepared),
	)
	for _, value := range prepared {
		desiredByInstaller[value.installer] = value.desired
	}
	finalizedSources := make(map[string]struct{}, len(prepared))
	for _, entry := range entries {
		hydrated, supported := entry.installer.(HydrationInstaller)
		if !supported {
			continue
		}
		desired, found := desiredByInstaller[entry.name]
		if !found {
			return fmt.Errorf(
				"%w: hydration installer %q has no desired Source",
				basespec.ErrInvalid,
				entry.name,
			)
		}
		sourceKey := string(desired.RootID) + "\x00" + string(desired.SourceID)
		if _, finalized := finalizedSources[sourceKey]; finalized {
			continue
		}
		if err := hydrated.FinalizeHydration(ctx); err != nil {
			return fmt.Errorf(
				"finalize built-in installer %q: %w",
				entry.name,
				err,
			)
		}
		finalizedSources[sourceKey] = struct{}{}
	}

	for _, value := range prepared {
		if value.current {
			continue
		}
		if err := r.hydrator.CommitTopologyHydration(ctx, value.desired); err != nil {
			return fmt.Errorf(
				"commit topology hydration for installer %q: %w",
				value.installer,
				err,
			)
		}
	}
	if packageAware {
		for _, value := range prepared {
			for _, packageValue := range value.packages {
				if value.packageCurrent[packageValue.Key] {
					continue
				}
				if err := packageCoordinator.CommitTopologyPackageHydration(
					ctx,
					packageValue,
				); err != nil {
					return fmt.Errorf(
						"commit package hydration %q/%q: %w",
						packageValue.Key.InstallerName,
						packageValue.Key.Scope,
						err,
					)
				}
			}
			for _, stale := range value.stale {
				if err := packageCoordinator.DeleteTopologyPackageHydration(ctx, stale); err != nil {
					return fmt.Errorf("delete stale package hydration: %w", err)
				}
			}
		}
	}
	return nil
}

func normalizePackageScopes(
	values []basespec.Locator,
) ([]basespec.Locator, error) {
	seen := make(map[basespec.Locator]struct{}, len(values))
	output := make([]basespec.Locator, 0, len(values))
	for _, value := range values {
		if err := value.ValidatePortable(false); err != nil {
			return nil, err
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate built-in package scope %q",
				basespec.ErrConflict,
				value,
			)
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	slices.Sort(output)
	for index := 1; index < len(output); index++ {
		if packageScopesOverlap(output[index-1], output[index]) {
			return nil, fmt.Errorf(
				"%w: overlapping built-in package scopes %q and %q",
				basespec.ErrConflict,
				output[index-1],
				output[index],
			)
		}
	}
	return output, nil
}

func packageScopesOverlap(
	left basespec.Locator,
	right basespec.Locator,
) bool {
	return left == right ||
		strings.HasPrefix(string(left), string(right)+"/") ||
		strings.HasPrefix(string(right), string(left)+"/")
}
