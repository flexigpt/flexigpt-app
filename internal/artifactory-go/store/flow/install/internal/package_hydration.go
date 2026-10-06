package internal

import (
	"context"
	"fmt"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (c *Service) PrepareTopologyPackageHydrations(
	ctx context.Context,
	installerNames []string,
	desiredValues []installModel.PackageHydration,
) (installModel.PackageHydrationPreparation, error) {
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return installModel.PackageHydrationPreparation{}, err
	}

	installers := make(map[string]struct{}, len(installerNames))
	for index, installerName := range installerNames {
		if err := installModel.ValidateHydrationInstallerName(installerName); err != nil {
			return installModel.PackageHydrationPreparation{}, fmt.Errorf(
				"package hydration installer %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := installers[installerName]; duplicate {
			return installModel.PackageHydrationPreparation{}, fmt.Errorf(
				"%w: duplicate package hydration installer %q",
				spec.ErrInvalid,
				installerName,
			)
		}
		installers[installerName] = struct{}{}
	}

	desired, err := installModel.NormalizePackageHydrations(desiredValues)
	if err != nil {
		return installModel.PackageHydrationPreparation{}, err
	}

	preparation := installModel.PackageHydrationPreparation{
		Current: make(
			map[installModel.PackageHydrationKey]bool,
			len(desired),
		),
		Stale: make([]installModel.PackageHydration, 0),
	}
	desiredByKey := make(
		map[installModel.PackageHydrationKey]installModel.PackageHydration,
		len(desired),
	)
	for _, desiredValue := range desired {
		if _, found := installers[desiredValue.Key.InstallerName]; !found {
			return installModel.PackageHydrationPreparation{}, fmt.Errorf(
				"%w: package hydration %q/%q has no participating installer",
				spec.ErrInvalid,
				desiredValue.Key.InstallerName,
				desiredValue.Key.Scope,
			)
		}
		if !c.isProtectedRoot(desiredValue.RootID) {
			return installModel.PackageHydrationPreparation{}, fmt.Errorf(
				"%w: package hydration root %q is not protected",
				spec.ErrProtected,
				desiredValue.RootID,
			)
		}
		desiredByKey[desiredValue.Key] = desiredValue
		preparation.Current[desiredValue.Key] = false
	}

	persisted, err := c.metadata.ListTopologyPackageHydrations(ctx)
	if err != nil {
		return installModel.PackageHydrationPreparation{}, err
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

	stale, err := installModel.NormalizePackageHydrations(preparation.Stale)
	if err != nil {
		return installModel.PackageHydrationPreparation{}, err
	}
	preparation.Stale = stale
	return preparation.Clone(), nil
}

func (c *Service) CommitTopologyPackageHydration(
	ctx context.Context,
	value installModel.PackageHydration,
) error {
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if !c.isProtectedRoot(value.RootID) {
		return fmt.Errorf(
			"%w: package hydration root %q is not protected",
			spec.ErrProtected,
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

func (c *Service) DeleteTopologyPackageHydration(
	ctx context.Context,
	value installModel.PackageHydration,
) error {
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	return c.metadata.DeleteTopologyPackageHydration(ctx, value.Key)
}

func equalTopologyPackageHydration(
	left installModel.PackageHydration,
	right installModel.PackageHydration,
) bool {
	return left.Key == right.Key &&
		left.RootID == right.RootID &&
		left.SourceID == right.SourceID &&
		left.Fingerprint == right.Fingerprint
}
