package compose

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/internal"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type Config struct {
	Artifacts artifact.API
	Refresh   refreshFlow.API

	Sources         source.API
	Runtime         source.Runtime
	ContentMutation source.ContentMutation
	Packages        managedpackage.Runtime

	Policy rootModel.RootPolicy
}

func Open(
	config Config,
) (managepackageFlow.API, error) {
	if config.Artifacts == nil ||
		config.Refresh == nil ||
		config.Sources == nil ||
		config.Runtime == nil ||
		config.ContentMutation == nil ||
		config.Packages == nil {
		return nil, fmt.Errorf(
			"%w: manage package composition dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	service, err := internal.NewService(
		internal.Dependencies{
			Artifacts:       config.Artifacts,
			Refresh:         config.Refresh,
			Sources:         config.Sources,
			Runtime:         config.Runtime,
			ContentMutation: config.ContentMutation,
			Packages:        config.Packages,
			Policy:          config.Policy,
		},
	)
	if err != nil {
		return nil, err
	}

	return service, nil
}
