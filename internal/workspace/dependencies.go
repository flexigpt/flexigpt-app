package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type Dependencies struct {
	Store artifactstore.ConsumerAPI
}

func (d Dependencies) Validate() error {
	if d.Store == nil {
		return fmt.Errorf(
			"%w: Workspace Artifact Store dependencies are incomplete",
			spec.ErrInvalidWorkspace,
		)
	}
	return nil
}
