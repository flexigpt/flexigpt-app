package discovery

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
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

func (p Plan) BySource() map[basespec.SourceID]SourcePlan {
	output := make(map[basespec.SourceID]SourcePlan, len(p.Sources))
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

func (p Plan) Fingerprint() (cryptoutil.Digest, error) {
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

	raw, err := json.Marshal(rev)
	if err != nil {
		return "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(canonical), nil
}

func (p Plan) Validate() error {
	if err := basespec.ValidateOptionalText(
		"discovery plan revision",
		p.Revision,
		basespec.MaxVersionBytes,
	); err != nil {
		return err
	}
	seen := make(map[basespec.SourceID]struct{}, len(p.Sources))
	for index, sourcePlan := range p.Sources {
		if err := sourcePlan.Validate(); err != nil {
			return fmt.Errorf("source plan %d: %w", index, err)
		}
		if _, duplicate := seen[sourcePlan.SourceID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate source plan for %q",
				basespec.ErrInvalid,
				sourcePlan.SourceID,
			)
		}
		seen[sourcePlan.SourceID] = struct{}{}
	}
	return nil
}
