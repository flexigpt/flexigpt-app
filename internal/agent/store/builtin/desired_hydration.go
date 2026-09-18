package builtin

import (
	"context"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
)

func (i *Installer) DesiredHydration(
	ctx context.Context,
) (topology.Hydration, error) {
	if i == nil {
		return topology.Hydration{}, basespec.ErrClosed
	}
	if ctx == nil {
		return topology.Hydration{}, fmt.Errorf(
			"%w: built-in Agent hydration context is nil",
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
		RootID:        i.builtInTopology.Root.ID,
		SourceID:      i.builtInTopology.Sources[0].ID,
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
		fingerprint, err := PackageFingerprint(value)
		if err != nil {
			return nil, err
		}
		output = append(output, topology.PackageHydration{
			Key: topology.PackageHydrationKey{
				InstallerName: i.BuiltInName(),
				Scope:         scope,
			},
			RootID:      i.builtInTopology.Root.ID,
			SourceID:    i.builtInTopology.Sources[0].ID,
			Fingerprint: fingerprint,
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
		address, err := source.ParseManagedPackageAddressDirectory(
			value.Key.Scope,
		)
		if err != nil {
			return err
		}
		if address.Kind != agentDomain.BuiltinAgentCollectionPackageKind {
			continue
		}
		if err := i.agents.RemoveBuiltInAgentPackage(
			ctx,
			value.RootID,
			value.SourceID,
			address,
		); err != nil {
			return fmt.Errorf(
				"remove stale built-in Agent package %q: %w",
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
	if err := i.agents.EnsureBuiltInAgentSourceCurrent(
		ctx,
		i.builtInTopology.Root.ID,
		i.builtInTopology.Sources[0].ID,
	); err != nil {
		return err
	}

	for _, value := range i.prepared {
		if err := i.agents.ValidateBuiltInAgentPackage(
			ctx,
			value.installRequest(
				i.builtInTopology.Root.ID,
				i.builtInTopology.Sources[0].ID,
			),
		); err != nil {
			return fmt.Errorf(
				"validate built-in Agent package %q: %w",
				value.EmbeddedPackageRoot,
				err,
			)
		}
	}
	return nil
}
