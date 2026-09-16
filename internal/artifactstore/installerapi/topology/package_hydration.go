package topology

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type PackageHydrationKey struct {
	InstallerName string
	Scope         basespec.Locator
}

func (k PackageHydrationKey) Validate() error {
	if err := ValidateHydrationInstallerName(k.InstallerName); err != nil {
		return err
	}
	return k.Scope.ValidatePortable(false)
}

type PackageHydration struct {
	Key         PackageHydrationKey
	RootID      root.RootID
	SourceID    source.SourceID
	Fingerprint cryptoutil.Digest
}

func (h PackageHydration) Validate() error {
	if err := h.Key.Validate(); err != nil {
		return err
	}
	if err := h.RootID.Validate(); err != nil {
		return err
	}
	if err := h.SourceID.Validate(); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(h.Fingerprint); err != nil {
		return fmt.Errorf("package hydration fingerprint: %w", err)
	}
	return nil
}

func (h PackageHydration) Clone() PackageHydration {
	return h
}

type PackageHydrationPreparation struct {
	Current map[PackageHydrationKey]bool
	Stale   []PackageHydration
}

func (p PackageHydrationPreparation) Clone() PackageHydrationPreparation {
	output := PackageHydrationPreparation{
		Current: maps.Clone(p.Current),
		Stale:   make([]PackageHydration, len(p.Stale)),
	}
	for index, value := range p.Stale {
		output.Stale[index] = value.Clone()
	}
	return output
}

func (p PackageHydrationPreparation) CurrentFor(
	key PackageHydrationKey,
) bool {
	return p.Current[key]
}

type PackageHydrationCoordinator interface {
	PrepareTopologyPackageHydrations(
		ctx context.Context,
		desired []PackageHydration,
	) (PackageHydrationPreparation, error)

	CommitTopologyPackageHydration(
		ctx context.Context,
		value PackageHydration,
	) error

	DeleteTopologyPackageHydration(
		ctx context.Context,
		value PackageHydration,
	) error
}

type PackageHydrationStore interface {
	ListTopologyPackageHydrations(
		ctx context.Context,
	) ([]PackageHydration, error)

	PutTopologyPackageHydration(
		ctx context.Context,
		value PackageHydration,
	) error

	DeleteTopologyPackageHydration(
		ctx context.Context,
		key PackageHydrationKey,
	) error
}

func NormalizePackageHydrations(
	values []PackageHydration,
) ([]PackageHydration, error) {
	output := make([]PackageHydration, len(values))
	seen := make(map[PackageHydrationKey]struct{}, len(values))
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[value.Key]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate package hydration %q/%q",
				basespec.ErrConflict,
				value.Key.InstallerName,
				value.Key.Scope,
			)
		}
		seen[value.Key] = struct{}{}
		output[index] = value.Clone()
	}
	slices.SortFunc(output, func(left, right PackageHydration) int {
		if left.Key.InstallerName != right.Key.InstallerName {
			if left.Key.InstallerName < right.Key.InstallerName {
				return -1
			}
			return 1
		}
		if left.Key.Scope < right.Key.Scope {
			return -1
		}
		if left.Key.Scope > right.Key.Scope {
			return 1
		}
		return 0
	})
	return output, nil
}
