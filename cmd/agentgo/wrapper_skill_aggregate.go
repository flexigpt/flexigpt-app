package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/flexigpt/agentskills-go/document"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/skillcatalog/inferenceadapter"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type ArtifactSkillFilter struct {
	Types          []string                    `json:"types,omitempty"`
	Inserts        []document.SkillInsert      `json:"inserts,omitempty"`
	NamePrefix     string                      `json:"namePrefix,omitempty"`
	LocationPrefix string                      `json:"locationPrefix,omitempty"`
	AllowArtifacts []artifactModel.ArtifactRef `json:"allowArtifacts,omitempty"`

	SessionID agentskillsRuntimeSpec.SessionID     `json:"sessionID,omitempty"`
	Activity  agentskillsRuntimeSpec.SkillActivity `json:"activity,omitempty"`
}

type ArtifactSkillSummary struct {
	Artifact     artifactModel.ArtifactRef
	IsEnabled    bool
	Insert       document.SkillInsert
	HasArguments bool
	HasResources bool
}

type skillArtifactResolver interface {
	ResolveSkills(
		ctx context.Context,
		refs []artifactModel.ArtifactRef,
	) (inferenceadapter.ResolvedSkills, error)
}

// SkillAggregateWrapper owns Artifact-aware Wails API projections.
// It borrows the adapter and runtime; it does not own their shutdown.
type SkillAggregateWrapper struct {
	mu       sync.RWMutex
	resolver skillArtifactResolver
	runtime  *skillRuntime.Service
}

func InitSkillAggregateWrapper(
	wrapper *SkillAggregateWrapper,
	resolver skillArtifactResolver,
	runtime *skillRuntime.Service,
) error {
	if wrapper == nil || resolver == nil || runtime == nil {
		return fmt.Errorf(
			"%w: Skill aggregate wrapper dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()
	if wrapper.resolver != nil || wrapper.runtime != nil {
		return fmt.Errorf(
			"%w: Skill aggregate wrapper is already initialized",
			spec.ErrConflict,
		)
	}
	wrapper.resolver = resolver
	wrapper.runtime = runtime
	return nil
}

func withSkillAggregate[T any](
	w *SkillAggregateWrapper,
	fn func(context.Context) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil {
			return zero, spec.ErrClosed
		}

		w.mu.RLock()
		defer w.mu.RUnlock()
		if w.resolver == nil || w.runtime == nil {
			return zero, spec.ErrClosed
		}
		return fn(context.Background())
	})
}

func (w *SkillAggregateWrapper) ResolveArtifactSkill(
	ref artifactModel.ArtifactRef,
) (inferenceadapter.ResolvedArtifactSkill, error) {
	values, err := w.ResolveArtifactSkills([]artifactModel.ArtifactRef{ref})
	if err != nil {
		return inferenceadapter.ResolvedArtifactSkill{}, err
	}
	if len(values) != 1 {
		return inferenceadapter.ResolvedArtifactSkill{}, fmt.Errorf(
			"%w: expected one resolved Artifact Skill",
			spec.ErrReferenceUnresolved,
		)
	}
	return values[0], nil
}

func (w *SkillAggregateWrapper) ResolveArtifactSkills(
	refs []artifactModel.ArtifactRef,
) ([]inferenceadapter.ResolvedArtifactSkill, error) {
	return withSkillAggregate(
		w,
		func(ctx context.Context) ([]inferenceadapter.ResolvedArtifactSkill, error) {
			selected, err := w.resolver.ResolveSkills(ctx, refs)
			if err != nil {
				return nil, err
			}
			return selected.Values, nil
		},
	)
}

func (w *SkillAggregateWrapper) GetArtifactSkillsPrompt(
	filter ArtifactSkillFilter,
) (string, error) {
	return withSkillAggregate(w, func(ctx context.Context) (string, error) {
		selected, err := w.resolver.ResolveSkills(ctx, filter.AllowArtifacts)
		if err != nil {
			return "", err
		}
		if len(filter.Inserts) != 0 && !containsInstructionInsert(filter.Inserts) {
			return "", nil
		}
		return w.runtime.SkillsPrompt(ctx, &agentskillsRuntime.SkillFilter{
			Types:          filter.Types,
			NamePrefix:     filter.NamePrefix,
			LocationPrefix: filter.LocationPrefix,
			AllowSkills:    selected.Definitions,
			SessionID:      filter.SessionID,
			Activity:       filter.Activity,
		})
	})
}

func (w *SkillAggregateWrapper) ListArtifactSkillRefs(
	filter ArtifactSkillFilter,
) ([]artifactModel.ArtifactRef, error) {
	return withSkillAggregate(
		w,
		func(ctx context.Context) ([]artifactModel.ArtifactRef, error) {
			selected, err := w.resolver.ResolveSkills(ctx, filter.AllowArtifacts)
			if err != nil {
				return nil, err
			}
			records, err := w.runtime.ListAgentSkills(
				ctx,
				&agentskillsRuntime.SkillListFilter{
					Types:          filter.Types,
					NamePrefix:     filter.NamePrefix,
					LocationPrefix: filter.LocationPrefix,
					AllowSkills:    selected.Definitions,
					Inserts:        filter.Inserts,
					SessionID:      filter.SessionID,
					Activity:       filter.Activity,
				},
			)
			if err != nil {
				return nil, err
			}
			return selected.ArtifactRefs(records), nil
		},
	)
}

func (w *SkillAggregateWrapper) DescribeArtifactSkill(
	ref artifactModel.ArtifactRef,
) (ArtifactSkillSummary, error) {
	return withSkillAggregate(
		w,
		func(ctx context.Context) (ArtifactSkillSummary, error) {
			selected, err := w.resolver.ResolveSkills(
				ctx,
				[]artifactModel.ArtifactRef{ref},
			)
			if err != nil {
				return ArtifactSkillSummary{}, err
			}
			if len(selected.Values) != 1 {
				return ArtifactSkillSummary{}, fmt.Errorf(
					"%w: expected one resolved Artifact Skill",
					spec.ErrReferenceUnresolved,
				)
			}

			resolved := selected.Values[0]
			records, err := w.runtime.ListAgentSkills(
				ctx,
				&agentskillsRuntime.SkillListFilter{
					AllowSkills: selected.Definitions,
				},
			)
			if err != nil {
				return ArtifactSkillSummary{}, err
			}
			for _, record := range records {
				if record.Def == resolved.Definition {
					return ArtifactSkillSummary{
						Artifact:     ref,
						IsEnabled:    resolved.Enabled,
						Insert:       record.Insert,
						HasArguments: len(record.Arguments) != 0,
						HasResources: record.Resources.HasResources,
					}, nil
				}
			}
			return ArtifactSkillSummary{}, fmt.Errorf(
				"%w: runtime did not index Artifact Skill %q",
				spec.ErrReferenceUnresolved,
				ref.ArtifactID,
			)
		},
	)
}

func containsInstructionInsert(values []document.SkillInsert) bool {
	for _, value := range values {
		insert, supported := document.NormalizeSkillInsert(value)
		if supported && insert == document.SkillInsertInstructions {
			return true
		}
	}
	return false
}

func (w *SkillAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	w.resolver = nil
	w.runtime = nil
}
