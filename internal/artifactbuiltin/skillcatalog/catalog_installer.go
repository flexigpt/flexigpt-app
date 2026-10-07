package skillcatalog

import (
	"fmt"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type InstallerDependencies struct {
	Hydrator installModel.CompiledHydrationCoordinator
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*installFlow.CatalogInstaller, error) {
	if dependencies.Hydrator == nil {
		return nil, fmt.Errorf(
			"%w: skill generated catalog installer dependencies are incomplete",
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
		nil,
	)
	if err != nil {
		return nil, err
	}
	return value, nil
}
