package builtin

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
)

// Installer is implemented by one artifact-family-owned built-in installerapi.
// The generic built-in layer deliberately does not inspect package contents,
// artifact definitions, or artifact-specific manifests.
type Installer interface {
	BuiltInName() string
	BuiltInPackageScopes() []model.Locator
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

	// FinalizeHydration runs after package publication has completed for every
	// registered installer. It must refresh or verify source-backed Artifact
	// state against the final shared Source generation. Shared source refresh
	// work must be idempotent. It must not mutate managed package content or
	// topology.
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
	scopes     map[model.Locator]string
}

func NewDefaultBootstrapRegistry(
	ensurer topology.Ensurer,
	hydrator topology.HydrationCoordinator,
) (*BootstrapRegistry, error) {
	return NewBootstrapRegistry(
		documentTopology.BuiltinTopologyDeclaration(),
		ensurer,
		hydrator,
	)
}

func NewBootstrapRegistry(
	declaration topology.Declaration,
	ensurer topology.Ensurer,
	hydrator topology.HydrationCoordinator,
) (*BootstrapRegistry, error) {
	if ensurer == nil || hydrator == nil {
		return nil, fmt.Errorf(
			"%w: built-in bootstrap dependencies are incomplete",
			model.ErrInvalid,
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
		scopes:      map[model.Locator]string{},
	}, nil
}

func (r *BootstrapRegistry) Register(inst Installer) error {
	if r == nil {
		return fmt.Errorf("%w: built-in bootstrap registry is nil", model.ErrInvalid)
	}
	if inst == nil {
		return fmt.Errorf("%w: built-in installer is nil", model.ErrInvalid)
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
			model.ErrConflict,
			name,
		)
	}

	existingScopes := make([]model.Locator, 0, len(r.scopes))
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
				model.ErrConflict,
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
		return fmt.Errorf("%w: built-in bootstrap registry is nil", model.ErrInvalid)
	}
	if ctx == nil {
		return fmt.Errorf("%w: built-in bootstrap context is nil", model.ErrInvalid)
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
				model.ErrInvalid,
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
					model.ErrInvalid,
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

	compiled, err := r.prepareCompiledHydration(
		ctx,
		entries,
		prepared,
	)
	if err != nil {
		return err
	}

	physicalWork := false
	for _, entry := range entries {
		if compiled.handles(entry.name) {
			continue
		}

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
			if !packageHydrationNeedsMutation(preparedValue) {
				continue
			}
			physicalWork = true
			if err := packageInstaller.EnsurePackageHydration(
				ctx,
				preparedValue.current,
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
			if current {
				continue
			}
			physicalWork = true
			if err := hydrated.EnsureHydration(ctx, current); err != nil {
				return fmt.Errorf(
					"ensure built-in installer %q: %w",
					entry.name,
					err,
				)
			}
			continue
		}
		physicalWork = true
		if err := entry.installer.Ensure(ctx); err != nil {
			return fmt.Errorf(
				"ensure built-in installer %q: %w",
				entry.name,
				err,
			)
		}
	}
	if err := compiled.apply(ctx); err != nil {
		return fmt.Errorf("hydrate compiled built-in packages: %w", err)
	}
	if compiled.hasPhysicalWork() {
		physicalWork = true
	}

	// Installers can share a protected managed Source. A later installer can
	// advance the shared Source revision after an earlier installer publishes
	// content. Every hydration-aware installer therefore finalizes only after
	// all package publication completes. Finalizers must make shared refresh
	// work idempotent through EnsureSourceCurrent or an equivalent operation.
	desiredByInstaller := make(
		map[string]topology.Hydration,
		len(prepared),
	)
	for _, value := range prepared {
		desiredByInstaller[value.installer] = value.desired
	}

	// No physical package work means there is nothing to refresh or validate.
	// Hydration markers are still committed below when their fingerprints changed.
	if physicalWork {
		for _, entry := range entries {
			hydrated, supported := entry.installer.(HydrationInstaller)
			if !supported {
				continue
			}
			if _, found := desiredByInstaller[entry.name]; !found {
				return fmt.Errorf(
					"%w: hydration installer %q has no desired Source",
					model.ErrInvalid,
					entry.name,
				)
			}

			if err := hydrated.FinalizeHydration(ctx); err != nil {
				return fmt.Errorf(
					"finalize built-in installer %q: %w",
					entry.name,
					err,
				)
			}
		}
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
	values []model.Locator,
) ([]model.Locator, error) {
	seen := make(map[model.Locator]struct{}, len(values))
	output := make([]model.Locator, 0, len(values))
	for _, value := range values {
		if err := value.ValidatePortable(false); err != nil {
			return nil, err
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate built-in package scope %q",
				model.ErrConflict,
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
				model.ErrConflict,
				output[index-1],
				output[index],
			)
		}
	}
	return output, nil
}

func packageScopesOverlap(
	left model.Locator,
	right model.Locator,
) bool {
	return left == right ||
		strings.HasPrefix(string(left), string(right)+"/") ||
		strings.HasPrefix(string(right), string(left)+"/")
}
