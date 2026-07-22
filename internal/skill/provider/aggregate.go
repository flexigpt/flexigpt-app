package provider

import (
	"context"
	"errors"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Aggregate struct {
	providers []Provider
}

func NewAggregate(
	providers ...Provider,
) (*Aggregate, error) {
	values := make([]Provider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			return nil, errors.New("Skill aggregate provider contains nil")
		}
		values = append(values, provider)
	}
	if len(values) == 0 {
		return nil, errors.New("Skill aggregate provider is empty")
	}
	return &Aggregate{providers: values}, nil
}

func (a *Aggregate) Owns(identity string) bool {
	for _, provider := range a.providers {
		if provider.Owns(identity) {
			return true
		}
	}
	return false
}

func (a *Aggregate) List(
	ctx context.Context,
	scope Scope,
) ([]Skill, error) {
	var output []Skill
	for _, provider := range a.providers {
		values, err := provider.List(ctx, scope)
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}
	applyPrecedence(output)
	sort.Slice(output, func(left, right int) bool {
		if output[left].Shadowed != output[right].Shadowed {
			return !output[left].Shadowed
		}
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		if originRank(output[left]) != originRank(output[right]) {
			return originRank(output[left]) > originRank(output[right])
		}
		if output[left].Priority != output[right].Priority {
			return output[left].Priority > output[right].Priority
		}
		return output[left].Identity < output[right].Identity
	})
	return output, nil
}

func (a *Aggregate) Render(
	ctx context.Context,
	request RenderRequest,
) (RenderedSkill, error) {
	for _, provider := range a.providers {
		if provider.Owns(request.Identity) {
			return provider.Render(ctx, request)
		}
	}
	return RenderedSkill{
		Available: false,
		Diagnostics: []artifactstore.Diagnostic{
			unavailableDiagnostic(
				"skill.provider.identity-unresolved",
				"the requested Skill provider identity is unresolved",
			),
		},
	}, nil
}

func applyPrecedence(values []Skill) {
	byName := make(map[string][]int)
	for index := range values {
		if !values[index].Enabled || !values[index].Available {
			continue
		}
		byName[values[index].Name] = append(
			byName[values[index].Name],
			index,
		)
	}
	for _, indexes := range byName {
		if len(indexes) < 2 {
			continue
		}
		bestRank := -1
		bestPriority := 0
		best := make([]int, 0, len(indexes))
		for _, index := range indexes {
			rank := originRank(values[index])
			priority := values[index].Priority
			if len(best) == 0 ||
				rank > bestRank ||
				(rank == bestRank && priority > bestPriority) {
				bestRank = rank
				bestPriority = priority
				best = []int{index}
				continue
			}
			if rank == bestRank && priority == bestPriority {
				best = append(best, index)
			}
		}
		if len(best) > 1 {
			for _, index := range best {
				values[index].Diagnostics = artifactstore.AppendDiagnostics(
					values[index].Diagnostics,
					artifactstore.Diagnostic{
						Severity: artifactstore.DiagnosticError,
						Code:     "skill.provider.precedence-ambiguous",
						Message:  "multiple Skills have the same highest precedence",
					},
				)
			}
			continue
		}
		winner := values[best[0]].Identity
		for _, index := range indexes {
			if index == best[0] {
				continue
			}
			values[index].Shadowed = true
			values[index].ShadowedBy = winner
		}
	}
}

func originRank(value Skill) int {
	if value.Origin == OriginWorkspace {
		return 2
	}
	return 1
}

var _ Provider = (*Aggregate)(nil)
