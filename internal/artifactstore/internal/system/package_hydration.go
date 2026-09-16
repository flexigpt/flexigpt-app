package system

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
)

func (c *Components) PrepareTopologyPackageHydrations(
	ctx context.Context,
	installerNames []string,
	desiredValues []topology.PackageHydration,
) (topology.PackageHydrationPreparation, error) {
	if c == nil || c.metadata == nil {
		return topology.PackageHydrationPreparation{}, basespec.ErrClosed
	}
	if ctx == nil {
		return topology.PackageHydrationPreparation{}, fmt.Errorf(
			"%w: topology package hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return topology.PackageHydrationPreparation{}, err
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return topology.PackageHydrationPreparation{}, err
	}

	installers := make(map[string]struct{}, len(installerNames))
	for index, installerName := range installerNames {
		if err := topology.ValidateHydrationInstallerName(installerName); err != nil {
			return topology.PackageHydrationPreparation{}, fmt.Errorf(
				"package hydration installer %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := installers[installerName]; duplicate {
			return topology.PackageHydrationPreparation{}, fmt.Errorf(
				"%w: duplicate package hydration installer %q",
				basespec.ErrInvalid,
				installerName,
			)
		}
		installers[installerName] = struct{}{}
	}

	desired, err := topology.NormalizePackageHydrations(desiredValues)
	if err != nil {
		return topology.PackageHydrationPreparation{}, err
	}

	preparation := topology.PackageHydrationPreparation{
		Current: make(
			map[topology.PackageHydrationKey]bool,
			len(desired),
		),
		Stale: make([]topology.PackageHydration, 0),
	}
	desiredByKey := make(
		map[topology.PackageHydrationKey]topology.PackageHydration,
		len(desired),
	)
	for _, desiredValue := range desired {
		if _, found := installers[desiredValue.Key.InstallerName]; !found {
			return topology.PackageHydrationPreparation{}, fmt.Errorf(
				"%w: package hydration %q/%q has no participating installer",
				basespec.ErrInvalid,
				desiredValue.Key.InstallerName,
				desiredValue.Key.Scope,
			)
		}
		if !c.isProtectedRoot(desiredValue.RootID) {
			return topology.PackageHydrationPreparation{}, fmt.Errorf(
				"%w: package hydration root %q is not protected",
				basespec.ErrProtected,
				desiredValue.RootID,
			)
		}
		desiredByKey[desiredValue.Key] = desiredValue
		preparation.Current[desiredValue.Key] = false
	}

	persisted, err := c.metadata.ListTopologyPackageHydrations(ctx)
	if err != nil {
		return topology.PackageHydrationPreparation{}, err
	}
	for _, persistedValue := range persisted {
		if _, managed := installers[persistedValue.Key.InstallerName]; !managed {
			continue
		}
		desiredValue, wanted := desiredByKey[persistedValue.Key]
		if !wanted {
			preparation.Stale = append(
				preparation.Stale,
				persistedValue.Clone(),
			)
			continue
		}
		if equalTopologyPackageHydration(persistedValue, desiredValue) {
			preparation.Current[persistedValue.Key] = true
		}
	}

	stale, err := topology.NormalizePackageHydrations(preparation.Stale)
	if err != nil {
		return topology.PackageHydrationPreparation{}, err
	}
	preparation.Stale = stale
	return preparation.Clone(), nil
}

func (c *Components) CommitTopologyPackageHydration(
	ctx context.Context,
	value topology.PackageHydration,
) error {
	if c == nil || c.metadata == nil || c.Roots == nil || c.Sources == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: topology package hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if !c.isProtectedRoot(value.RootID) {
		return fmt.Errorf(
			"%w: package hydration root %q is not protected",
			basespec.ErrProtected,
			value.RootID,
		)
	}
	if _, err := c.Roots.Get(ctx, value.RootID); err != nil {
		return err
	}
	if _, err := c.Sources.Get(ctx, value.RootID, value.SourceID); err != nil {
		return err
	}
	return c.metadata.PutTopologyPackageHydration(ctx, value)
}

func (c *Components) DeleteTopologyPackageHydration(
	ctx context.Context,
	value topology.PackageHydration,
) error {
	if c == nil || c.metadata == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: topology package hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	return c.metadata.DeleteTopologyPackageHydration(ctx, value.Key)
}

func equalTopologyPackageHydration(
	left topology.PackageHydration,
	right topology.PackageHydration,
) bool {
	return left.Key == right.Key &&
		left.RootID == right.RootID &&
		left.SourceID == right.SourceID &&
		left.Fingerprint == right.Fingerprint
}
