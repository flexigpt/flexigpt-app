package builtin

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
)

// CompiledPackageInstaller is implemented by binary-owned built-in package
// sets whose declaration admission happened during generation.
type CompiledPackageInstaller interface {
	PackageHydrationInstaller

	CompiledRegistration(
		ctx context.Context,
	) (topology.CompiledRegistration, error)
}

type compiledHydrationWork struct {
	coordinator topology.CompiledHydrationCoordinator
	handled     map[string]struct{}
	plans       []topology.CompiledPackagePlan
}

func (w compiledHydrationWork) handles(
	installer string,
) bool {
	_, found := w.handled[installer]
	return found
}

func (w compiledHydrationWork) hasPhysicalWork() bool {
	return len(w.plans) != 0
}

func (w compiledHydrationWork) apply(
	ctx context.Context,
) error {
	if len(w.plans) == 0 {
		return nil
	}
	return w.coordinator.HydrateCompiledPackages(ctx, w.plans)
}

func packageHydrationNeedsMutation(
	value preparedHydration,
) bool {
	if !value.current || len(value.stale) != 0 {
		return true
	}
	for _, current := range value.packageCurrent {
		if !current {
			return true
		}
	}
	return false
}

func (r *BootstrapRegistry) prepareCompiledHydration(
	ctx context.Context,
	entries []registeredInstaller,
	prepared []preparedHydration,
) (compiledHydrationWork, error) {
	output := compiledHydrationWork{
		handled: make(map[string]struct{}),
	}

	coordinator, supported := r.hydrator.(topology.CompiledHydrationCoordinator)
	if !supported {
		return output, nil
	}
	output.coordinator = coordinator

	byInstaller := make(map[string]preparedHydration, len(prepared))
	for _, value := range prepared {
		byInstaller[value.installer] = value
	}

	registrations := make([]topology.CompiledRegistration, 0)
	for _, entry := range entries {
		installer, compiled := entry.installer.(CompiledPackageInstaller)
		if !compiled {
			continue
		}

		registration, err := installer.CompiledRegistration(ctx)
		if err != nil {
			return compiledHydrationWork{}, err
		}
		state, found := byInstaller[entry.name]
		if !found ||
			registration.Set.Hydration.InstallerName != entry.name ||
			registration.Set.Hydration != state.desired {
			return compiledHydrationWork{}, fmt.Errorf(
				"%w: compiled installer %q has inconsistent hydration state",
				basespec.ErrInvalid,
				entry.name,
			)
		}

		output.handled[entry.name] = struct{}{}
		registrations = append(registrations, registration)

		if !packageHydrationNeedsMutation(state) {
			continue
		}

		plan := topology.CompiledPackagePlan{
			Registration:    registration,
			TopologyCurrent: state.current,
			Stale: append(
				[]topology.PackageHydration(nil),
				state.stale...,
			),
		}
		for _, packageValue := range state.packages {
			if !state.packageCurrent[packageValue.Key] {
				plan.Changed = append(
					plan.Changed,
					packageValue.Key.Scope,
				)
			}
		}
		output.plans = append(output.plans, plan)
	}

	if len(registrations) == 0 {
		return output, nil
	}

	// Registration is always process-local and cheap. It gives any Source
	// refresh during this bootstrap access to the generated declaration map.
	// Physical package work remains conditional on hydration markers.
	if err := coordinator.RegisterCompiledPackages(
		ctx,
		registrations,
	); err != nil {
		return compiledHydrationWork{}, err
	}
	return output, nil
}
