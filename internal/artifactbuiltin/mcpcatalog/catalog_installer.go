package mcpcatalog

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp"
)

type InstallerDependencies struct {
	Hydrator installModel.CompiledHydrationCoordinator
	Cleanup  mcpAPI.BuiltinPackageCleanup
}

type Installer struct {
	*installFlow.CatalogInstaller
}

type lifecycle struct {
	cleanup mcpAPI.BuiltinPackageCleanup
}

type lifecycleState struct {
	servers []artifactModel.ArtifactRef
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Hydrator == nil ||
		dependencies.Cleanup == nil {
		return nil, fmt.Errorf(
			"%w: MCP generated catalog installer dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	set, err := generatedCatalogValue()
	if err != nil {
		return nil, err
	}

	value, err := installFlow.NewCatalogInstaller(
		set,
		dependencies.Hydrator,
		lifecycle{
			cleanup: dependencies.Cleanup,
		},
	)
	if err != nil {
		return nil, err
	}
	return &Installer{CatalogInstaller: value}, nil
}

func (l lifecycle) PrepareCompiledHydration(
	ctx context.Context,
	plan installModel.CompiledPackagePlan,
) (any, error) {
	// Protected topology reset now removes overlays and queues secret cleanup
	// inside Artifact Store. There is no external settings Root purge.
	if !plan.TopologyCurrent {
		return lifecycleState{}, nil
	}

	addresses, err := addressesForPlan(plan)
	if err != nil {
		return nil, err
	}
	servers, err := l.cleanup.CaptureBuiltInPackageServers(
		ctx,
		plan.Registration.Set.Hydration.RootID,
		plan.Registration.Set.Hydration.SourceID,
		addresses,
	)
	if err != nil {
		return nil, err
	}
	return lifecycleState{servers: servers}, nil
}

func (l lifecycle) CompleteCompiledHydration(
	ctx context.Context,
	_ installModel.CompiledPackagePlan,
	state any,
) error {
	value, ok := state.(lifecycleState)
	if !ok {
		return fmt.Errorf(
			"%w: MCP generated hydration lifecycle state is invalid",
			spec.ErrInvalid,
		)
	}
	return l.cleanup.CleanupRemovedBuiltInPackageServers(
		ctx,
		value.servers,
	)
}

func addressesForPlan(
	plan installModel.CompiledPackagePlan,
) ([]managedpackageModel.ManagedPackageAddress, error) {
	output := make(
		[]managedpackageModel.ManagedPackageAddress,
		0,
		len(plan.Changed)+len(plan.Stale),
	)
	seen := make(map[managedpackageModel.ManagedPackageAddress]struct{})

	byScope := make(map[spec.Locator]managedpackageModel.ManagedPackageAddress)
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
				spec.ErrInvalid,
				scope,
			)
		}
		if _, duplicate := seen[address]; duplicate {
			continue
		}
		seen[address] = struct{}{}
		output = append(output, address)
	}

	for _, stale := range plan.Stale {
		address, err := managedpackageModel.ParseManagedPackageAddressDirectory(
			stale.Key.Scope,
		)
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[address]; duplicate {
			continue
		}
		seen[address] = struct{}{}
		output = append(output, address)
	}
	return output, nil
}
