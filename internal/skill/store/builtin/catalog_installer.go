package builtin

import (
	"errors"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
)

type InstallerDependencies struct {
	Hydrator topology.CompiledHydrationCoordinator
	Packages fs.FS
}

type Installer struct {
	*builtin.CatalogInstaller
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Hydrator == nil || dependencies.Packages == nil {
		return nil, errors.New("skill generated catalog installer dependencies are incomplete")
	}

	set, err := GeneratedCatalogSet()
	if err != nil {
		return nil, err
	}

	value, err := builtin.NewCatalogInstallerForSet(
		set,
		dependencies.Packages,
		dependencies.Hydrator,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &Installer{CatalogInstaller: value}, nil
}
