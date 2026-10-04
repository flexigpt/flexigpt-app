package managepackage

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/internal"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Config contains only the named capabilities required by ManagePackage.
type Config struct {
	Artifacts       artifact.API
	Refresh         refreshFlow.API
	Sources         source.API
	Runtime         source.Runtime
	ContentMutation source.ContentMutation
	Packages        managedpackage.Runtime
	Policy          root.Policy
}

func NewService(config Config) (API, error) {
	if config.Artifacts == nil || config.Refresh == nil || config.Sources == nil || config.Runtime == nil ||
		config.ContentMutation == nil ||
		config.Packages == nil {
		return nil, fmt.Errorf("%w: manage package dependencies are incomplete", spec.ErrInvalid)
	}
	return internal.NewService(
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
}
