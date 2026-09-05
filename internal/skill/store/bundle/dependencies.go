package bundle

import (
	"fmt"

	artifactAPI "github.com/flexigpt/flexigpt-app/internal/artifactstore/api"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type Dependencies struct {
	Store artifactAPI.ConsumerAPI
}

func (d Dependencies) Validate() error {
	if d.Store == nil {
		return fmt.Errorf(
			"%w: skill bundle Artifact Store dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return nil
}
