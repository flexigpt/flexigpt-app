package builtin

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/model/store/consumerapi"
)

type InstallerDependencies struct {
	Hydrator installModel.CompiledHydrationCoordinator
	Cleanup  modelConsumerAPI.BuiltinPackageCleanup
}

type Installer struct {
	*install.CatalogInstaller
}

type lifecycle struct {
	cleanup modelConsumerAPI.BuiltinPackageCleanup
}

type lifecycleState struct {
	addresses []managedpackageModel.ManagedPackageAddress
	previous  []modelConsumerAPI.BuiltinArtifactSnapshot
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Hydrator == nil ||
		dependencies.Cleanup == nil {
		return nil, fmt.Errorf(
			"%w: Model generated catalog installer dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	set, err := generatedCatalogValue()
	if err != nil {
		return nil, err
	}

	value, err := install.NewCatalogInstaller(
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
	addresses, err := addressesForPlan(plan)
	if err != nil {
		return nil, err
	}
	previous, err := l.cleanup.CaptureBuiltInPackageArtifacts(
		ctx,
		plan.Registration.Set.Hydration.RootID,
		plan.Registration.Set.Hydration.SourceID,
		addresses,
	)
	if err != nil {
		return nil, err
	}

	return lifecycleState{
		addresses: addresses,
		previous:  previous,
	}, nil
}

func (l lifecycle) CompleteCompiledHydration(
	ctx context.Context,
	plan installModel.CompiledPackagePlan,
	state any,
) error {
	value, ok := state.(lifecycleState)
	if !ok {
		return fmt.Errorf(
			"%w: Model generated hydration lifecycle state is invalid",
			spec.ErrInvalid,
		)
	}

	return l.cleanup.ReconcileBuiltInPackageArtifacts(
		ctx,
		plan.Registration.Set.Hydration.RootID,
		plan.Registration.Set.Hydration.SourceID,
		value.addresses,
		value.previous,
	)
}

func addressesForPlan(
	plan installModel.CompiledPackagePlan,
) ([]managedpackageModel.ManagedPackageAddress, error) {
	byScope := make(map[spec.Locator]managedpackageModel.ManagedPackageAddress)
	for _, packageValue := range plan.Registration.Set.Packages {
		scope, err := packageValue.Address.Directory()
		if err != nil {
			return nil, err
		}
		byScope[scope] = packageValue.Address
	}

	output := make(
		[]managedpackageModel.ManagedPackageAddress,
		0,
		len(plan.Changed)+len(plan.Stale),
	)
	seen := make(map[managedpackageModel.ManagedPackageAddress]struct{})

	for _, scope := range plan.Changed {
		address, found := byScope[scope]
		if !found {
			return nil, fmt.Errorf(
				"%w: generated Model package scope %q is unknown",
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
