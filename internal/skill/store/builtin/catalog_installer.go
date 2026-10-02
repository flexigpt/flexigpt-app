package builtin

import (
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/topology"
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
		return nil, errors.New("skill generated catalog installer dependencies are incomplete")
	}

	set, err := generatedCatalogValue()
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
