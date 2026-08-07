package builtin

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
)

// Installer is implemented by one artifact-family-owned built-in installer.
// The generic built-in layer deliberately does not inspect package contents,
// artifact definitions, or artifact-specific manifests.
type Installer interface {
	BuiltInName() string
	BuiltInIDs() []string
	BuiltInPackageScopes() []basespec.Locator
	Ensure(ctx context.Context) error
}

// HydrationInstaller participates in desired-state reconciliation for
// binary-owned built-ins. PrepareBuiltInHydration runs before generic topology
// installation; CommitBuiltInHydration runs only after every installer has
// completed successfully.
type HydrationInstaller interface {
	PrepareBuiltInHydration(ctx context.Context) error

	CommitBuiltInHydration(ctx context.Context) error
}

type registeredInstaller struct {
	name      string
	installer Installer
}

// BootstrapRegistry owns application-level built-in installation order and
// shared topology. It is intentionally unaware of Skills, MCPs, or any other
// artifact format.
type BootstrapRegistry struct {
	configuration Registry
	topology      topology.Ensurer

	mu         sync.RWMutex
	installers map[string]Installer
	ids        map[string]string
	scopes     map[basespec.Locator]string
}

func NewBootstrapRegistry(
	configuration Registry,
	ensurer topology.Ensurer,
) (*BootstrapRegistry, error) {
	if ensurer == nil {
		return nil, fmt.Errorf(
			"%w: built-in bootstrap topology ensurer is nil",
			basespec.ErrInvalid,
		)
	}
	if err := configuration.Validate(); err != nil {
		return nil, err
	}
	return &BootstrapRegistry{
		configuration: configuration,
		topology:      ensurer,
		installers:    map[string]Installer{},
		ids: map[string]string{
			string(configuration.Root.ID):   "protected built-in Root",
			string(configuration.Source.ID): "protected built-in Source",
		},
		scopes: map[basespec.Locator]string{},
	}, nil
}

func (r *BootstrapRegistry) Register(installer Installer) error {
	if r == nil {
		return fmt.Errorf("%w: built-in bootstrap registry is nil", basespec.ErrInvalid)
	}
	if installer == nil {
		return fmt.Errorf("%w: built-in installer is nil", basespec.ErrInvalid)
	}

	name := installer.BuiltInName()
	if name == "" || name != strings.TrimSpace(name) {
		return fmt.Errorf(
			"%w: built-in installer name is required",
			basespec.ErrInvalid,
		)
	}
	ids, err := normalizeIDs(installer.BuiltInIDs())
	if err != nil {
		return err
	}
	scopes, err := normalizePackageScopes(installer.BuiltInPackageScopes())
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
	for _, id := range ids {
		if owner, exists := r.ids[id]; exists {
			return fmt.Errorf(
				"%w: built-in installer %q reuses static ID %q owned by %s",
				basespec.ErrConflict,
				name,
				id,
				owner,
			)
		}
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

	r.installers[name] = installer
	for _, id := range ids {
		r.ids[id] = name
	}
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

	ctx = protection.WithPrivilegedInstaller(ctx)
	for _, entry := range entries {
		hydrator, supported := entry.installer.(HydrationInstaller)
		if !supported {
			continue
		}
		if err := hydrator.PrepareBuiltInHydration(ctx); err != nil {
			return fmt.Errorf(
				"prepare built-in hydration for installer %q: %w",
				entry.name,
				err,
			)
		}
	}

	if _, err := r.configuration.EnsureTopology(ctx, r.topology); err != nil {
		return fmt.Errorf("ensure built-in topology: %w", err)
	}
	for _, entry := range entries {
		if err := entry.installer.Ensure(ctx); err != nil {
			return fmt.Errorf(
				"ensure built-in installer %q: %w",
				entry.name,
				err,
			)
		}
	}
	for _, entry := range entries {
		hydrator, supported := entry.installer.(HydrationInstaller)
		if !supported {
			continue
		}
		if err := hydrator.CommitBuiltInHydration(ctx); err != nil {
			return fmt.Errorf(
				"commit built-in hydration for installer %q: %w",
				entry.name,
				err,
			)
		}
	}
	return nil
}

func normalizeIDs(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	output := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || value != strings.TrimSpace(value) {
			return nil, fmt.Errorf(
				"%w: built-in static ID is required",
				basespec.ErrInvalid,
			)
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate built-in static ID %q",
				basespec.ErrConflict,
				value,
			)
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	sort.Strings(output)
	return output, nil
}

func normalizePackageScopes(
	values []basespec.Locator,
) ([]basespec.Locator, error) {
	seen := make(map[basespec.Locator]struct{}, len(values))
	output := make([]basespec.Locator, 0, len(values))
	for _, value := range values {
		if err := basespec.ValidatePortableLocator(value, false); err != nil {
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
