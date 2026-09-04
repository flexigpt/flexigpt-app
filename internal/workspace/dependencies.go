package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

// Dependencies is the application-composition boundary for Workspace.
// Refresh and catalog freshness remain provider-dispatched Artifact Store
// capabilities. Workspace never receives a raw catalog repository or a plan
// execution entry point.
type Dependencies struct {
	Sources            *source.Service
	Collections        *collection.Service
	Artifacts          *artifact.Service
	Refresh            refresh.CollectionAPI
	Resources          artifactstore.ResourceResolver
	RootMutationPolicy protection.RootPolicy
}

func (d Dependencies) Validate() error {
	if d.Sources == nil ||
		d.Collections == nil ||
		d.Artifacts == nil ||
		d.Refresh == nil ||
		d.Resources == nil ||
		d.RootMutationPolicy == nil {
		return fmt.Errorf(
			"%w: Workspace Artifact Store dependencies are incomplete",
			spec.ErrInvalidWorkspace,
		)
	}
	return nil
}
