package builtin

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

func (i *Installer) DesiredHydration(
	ctx context.Context,
) (topology.Hydration, error) {
	if i == nil {
		return topology.Hydration{}, basespec.ErrClosed
	}
	if ctx == nil {
		return topology.Hydration{}, fmt.Errorf(
			"%w: built-in Skill hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return topology.Hydration{}, err
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return topology.Hydration{}, err
	}

	value := topology.Hydration{
		InstallerName: i.BuiltInName(),
		RootID:        i.rootID,
		SourceID:      i.sourceID,
		Fingerprint:   i.fingerprint,
	}
	if err := value.Validate(); err != nil {
		return topology.Hydration{}, err
	}
	return value, nil
}

func (i *Installer) DesiredPackageHydrations(
	ctx context.Context,
) ([]topology.PackageHydration, error) {
	if i == nil {
		return nil, basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return nil, err
	}
	output := make([]topology.PackageHydration, 0, len(i.prepared))
	for _, value := range i.prepared {
		scope, err := value.PackageAddress.Directory()
		if err != nil {
			return nil, err
		}
		digest, err := PackageFingerprint(value)
		if err != nil {
			return nil, err
		}
		output = append(output, topology.PackageHydration{
			Key: topology.PackageHydrationKey{
				InstallerName: i.BuiltInName(),
				Scope:         scope,
			},
			RootID:      i.rootID,
			SourceID:    i.sourceID,
			Fingerprint: digest,
		})
	}
	return topology.NormalizePackageHydrations(output)
}

func (i *Installer) EnsureHydration(
	ctx context.Context,
	_ bool,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	return i.EnsureBuiltInArtifacts(ctx)
}

func (i *Installer) EnsurePackageHydration(
	ctx context.Context,
	_ bool,
	stale []topology.PackageHydration,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	for _, value := range stale {
		address, err := source.ParseManagedPackageAddressDirectory(value.Key.Scope)
		if err != nil {
			return err
		}
		if address.Kind != skillDomain.BuiltinSkillCollectionPackageKind {
			continue
		}
		if err := i.skills.RemoveBuiltInSkillPackage(
			ctx,
			value.RootID,
			value.SourceID,
			address,
		); err != nil {
			return fmt.Errorf(
				"remove stale built-in Skill package %q: %w",
				value.Key.Scope,
				err,
			)
		}
	}
	for _, value := range i.prepared {
		if err := i.installPreparedPackage(ctx, value); err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) FinalizeHydration(
	ctx context.Context,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	return i.skills.EnsureBuiltInSkillSourceCurrent(
		ctx,
		i.rootID,
		i.sourceID,
	)
}
