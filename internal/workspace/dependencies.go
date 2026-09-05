package workspace

import (
	"fmt"

	artifactAPI "github.com/flexigpt/flexigpt-app/internal/artifactstore/api"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type Dependencies struct {
	Store artifactAPI.ConsumerAPI
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
