package discovery

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Plan struct {
	// Revision is consumer-owned discovery-policy identity. Consumers must
	// change it whenever planner behavior changes in a way that can alter
	// discovery scope, decoder eligibility, or interpretation.
	//
	// It is intentionally included in the catalog plan fingerprint while
	// source snapshot generations remain concurrency preconditions only.
	Revision string       `json:"revision,omitempty"`
	Sources  []SourcePlan `json:"sources"`
}

func (p Plan) Validate() error {
	if err := artifactstore.ValidateOptionalText(
		"discovery plan revision",
		p.Revision,
		artifactstore.MaxVersionBytes,
	); err != nil {
		return err
	}
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

type fp struct {
	Revision string       `json:"revision,omitempty"`
	Sources  []SourcePlan `json:"sources"`
}

func (p Plan) Fingerprint() (artifactstore.Digest, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	values := make([]SourcePlan, len(p.Sources))
	for index, value := range p.Sources {
		values[index] = value.Normalized()
		// Snapshot generations are published separately. They are a
		// concurrency precondition, not a discovery capability input.
		values[index].ExpectedGeneration = ""
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].SourceID < values[right].SourceID
	})
	rev := fp{
		Revision: p.Revision,
		Sources:  values,
	}
	//nolint:musttag // Rev.
	raw, err := json.Marshal(rev)
	if err != nil {
		return "", err
	}
	canonical, err := jsoncanon.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return artifactstore.DigestBytes(canonical), nil
}
