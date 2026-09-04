package bundle

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/managedartifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type Dependencies struct {
	Sources            *source.Service
	Collections        *collection.Service
	Artifacts          *artifact.Service
	Refresh            refresh.CollectionAPI
	ManagedArtifacts   *managedartifact.Service
	Resources          artifactstore.ResourceResolver
	RootMutationPolicy protection.RootPolicy
}

func (d Dependencies) Validate() error {
	if d.Sources == nil ||
		d.Collections == nil ||
		d.Artifacts == nil ||
		d.Refresh == nil ||
		d.ManagedArtifacts == nil ||
		d.Resources == nil ||
		d.RootMutationPolicy == nil {
		return fmt.Errorf(
			"%w: skill bundle Artifact Store dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return nil
}
