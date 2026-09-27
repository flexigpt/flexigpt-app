package builtin

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
)

type InstallerDependencies struct {
	Hydrator topology.CompiledHydrationCoordinator
}

type Installer struct {
	*builtin.CatalogInstaller
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Hydrator == nil {
		return nil, fmt.Errorf(
			"%w: Tool generated catalog installer dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}

	set, err := GeneratedCatalogSet()
	if err != nil {
		return nil, err
	}
	value, err := builtin.NewCatalogInstallerForSet(
		set,
		dependencies.Hydrator,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &Installer{CatalogInstaller: value}, nil
}
