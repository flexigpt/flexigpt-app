package agentcatalog

import (
	"errors"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
)

type InstallerDependencies struct {
	Hydrator installModel.CompiledHydrationCoordinator
}

type Installer struct {
	*installFlow.CatalogInstaller
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Hydrator == nil {
		return nil, errors.New("agent generated catalog installer dependencies are incomplete")
	}

	set, err := generatedCatalogValue()
	if err != nil {
		return nil, err
	}

	value, err := installFlow.NewCatalogInstaller(
		set,
		dependencies.Hydrator,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &Installer{CatalogInstaller: value}, nil
}
