package main

import (
	"context"
	"errors"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/skillruntime"
	skillruntimeSpec "github.com/flexigpt/flexigpt-app/internal/skillruntime/spec"
)

type aggregateSkillProvider struct {
	providers []skillruntime.Provider
}

func newAggregateSkillProvider(
	providers ...skillruntime.Provider,
) (*aggregateSkillProvider, error) {
	values := make([]skillruntime.Provider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			return nil, errors.New("skill aggregate provider contains nil")
		}
		values = append(values, provider)
	}
	if len(values) == 0 {
		return nil, errors.New("skill aggregate provider is empty")
	}
	return &aggregateSkillProvider{providers: values}, nil
}

func (a *aggregateSkillProvider) Owns(ref skillruntimeSpec.SkillRef) bool {
	for _, provider := range a.providers {
		if provider.Owns(ref) {
			return true
		}
	}
	return false
}

func (a *aggregateSkillProvider) List(
	ctx context.Context,
	scope skillruntime.Scope,
) ([]skillruntime.Skill, error) {
	var output []skillruntime.Skill
	for _, provider := range a.providers {
		values, err := provider.List(ctx, scope)
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}
	applyPrecedence(output)
	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return skillRefKey(output[left].Ref) < skillRefKey(output[right].Ref)
	})
	return output, nil
}

func (a *aggregateSkillProvider) Render(
	ctx context.Context,
	request skillruntime.RenderRequest,
) (skillruntime.RenderedSkill, error) {
	for _, provider := range a.providers {
		if provider.Owns(request.Ref) {
			return provider.Render(ctx, request)
		}
	}
	return skillruntime.RenderedSkill{
		Available: false,
		Diagnostics: []artifactstore.Diagnostic{
			{
				Severity: artifactstore.DiagnosticWarning,
				Code:     "skill.provider.identity-unresolved",
				Message:  "the requested Skill provider identity is unresolved",
			},
		},
	}, nil
}

func applyPrecedence(values []skillruntime.Skill) {
	byName := make(map[string][]int)
	for index := range values {
		if !values[index].Enabled ||
			!values[index].Available ||
			!values[index].RuntimeAllowed ||
			!values[index].CatalogCurrent {
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
		for _, index := range indexes {
			values[index].Diagnostics = artifactstore.AppendDiagnostics(
				values[index].Diagnostics,
				artifactstore.Diagnostic{
					Severity: artifactstore.DiagnosticError,
					Code:     "skill.provider.name-ambiguous",
					Message:  "multiple eligible Skills have the same name",
				},
			)
			values[index].Available = false
			values[index].RuntimeAllowed = false
		}
	}
}

func skillRefKey(ref skillruntimeSpec.SkillRef) string {
	if ref.Artifact != nil {
		return "artifact|" +
			string(ref.Artifact.RootID) + "|" +
			string(ref.Artifact.ArtifactID)
	}
	return "installed|" +
		string(ref.BundleID) + "|" +
		string(ref.SkillSlug) + "|" +
		string(ref.SkillID)
}
