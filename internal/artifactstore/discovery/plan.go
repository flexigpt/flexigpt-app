package discovery

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Plan struct {
	Sources []SourcePlan
}

func (p Plan) Validate() error {
	seen := make(map[artifactstore.SourceID]struct{}, len(p.Sources))
	for index, sourcePlan := range p.Sources {
		if err := sourcePlan.Validate(); err != nil {
			return fmt.Errorf("source plan %d: %w", index, err)
		}
		if _, duplicate := seen[sourcePlan.SourceID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate source plan for %q",
				artifactstore.ErrInvalid,
				sourcePlan.SourceID,
			)
		}
		seen[sourcePlan.SourceID] = struct{}{}
	}
	return nil
}

func (p Plan) BySource() map[artifactstore.SourceID]SourcePlan {
	output := make(map[artifactstore.SourceID]SourcePlan, len(p.Sources))
	for _, value := range p.Sources {
		value = value.Normalized()
		output[value.SourceID] = value
	}
	return output
}
