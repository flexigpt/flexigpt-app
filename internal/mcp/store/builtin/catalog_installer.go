package builtin

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
)

type InstallerDependencies struct {
	Hydrator topology.CompiledHydrationCoordinator
	Cleanup  mcpConsumerAPI.BuiltinPackageCleanup
	Overlays mcpOverlay.RootPurger
}

type Installer struct {
	*builtin.CatalogInstaller
}

type lifecycle struct {
	cleanup  mcpConsumerAPI.BuiltinPackageCleanup
	overlays mcpOverlay.RootPurger
}

type lifecycleState struct {
	rootPurged bool
	servers    []artifact.ArtifactRef
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Hydrator == nil ||
		dependencies.Cleanup == nil ||
		dependencies.Overlays == nil {
		return nil, fmt.Errorf(
			"%w: MCP generated catalog installer dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}

	set, err := generatedCatalogValue()
	if err != nil {
		return nil, err
	}

	value, err := builtin.NewCatalogInstallerForSet(
		set,
		dependencies.Hydrator,
		lifecycle{
			cleanup:  dependencies.Cleanup,
			overlays: dependencies.Overlays,
		},
	)
	if err != nil {
		return nil, err
	}
	return &Installer{CatalogInstaller: value}, nil
}

func (l lifecycle) PrepareCompiledHydration(
	ctx context.Context,
	plan topology.CompiledPackagePlan,
) (any, error) {
	if !plan.TopologyCurrent {
		if err := l.overlays.PurgeRoot(
			ctx,
			plan.Registration.Set.Hydration.RootID,
		); err != nil {
			return nil, err
		}
		return lifecycleState{rootPurged: true}, nil
	}

	addresses := make(
		[]source.ManagedPackageAddress,
		0,
		len(plan.Changed)+len(plan.Stale),
	)
	byScope := make(map[basespec.Locator]source.ManagedPackageAddress)
	for _, packageValue := range plan.Registration.Set.Packages {
		scope, err := packageValue.Address.Directory()
		if err != nil {
			return nil, err
		}
		byScope[scope] = packageValue.Address
	}
	for _, scope := range plan.Changed {
		address, found := byScope[scope]
		if !found {
			return nil, fmt.Errorf(
				"%w: MCP generated package scope %q is unknown",
				basespec.ErrInvalid,
				scope,
			)
		}
		addresses = append(addresses, address)
	}
	for _, stale := range plan.Stale {
		address, err := source.ParseManagedPackageAddressDirectory(
			stale.Key.Scope,
		)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, address)
	}

	refs, err := l.cleanup.CaptureBuiltInPackageServers(
		ctx,
		plan.Registration.Set.Hydration.RootID,
		plan.Registration.Set.Hydration.SourceID,
		addresses,
	)
	if err != nil {
		return nil, err
	}
	return lifecycleState{servers: refs}, nil
}

func (l lifecycle) CompleteCompiledHydration(
	ctx context.Context,
	_ topology.CompiledPackagePlan,
	state any,
) error {
	value, ok := state.(lifecycleState)
	if !ok {
		return fmt.Errorf(
			"%w: MCP generated hydration lifecycle state is invalid",
			basespec.ErrInvalid,
		)
	}
	if value.rootPurged {
		return nil
	}
	return l.cleanup.CleanupRemovedBuiltInPackageServers(
		ctx,
		value.servers,
	)
}
