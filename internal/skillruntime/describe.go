package skillruntime

import (
	"context"
	"fmt"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// ArtifactSkillSummary is the runtime-derived summary used by durable
// selection validators. It contains no filesystem path and no process-local
// SkillDef identity.
type ArtifactSkillSummary struct {
	Artifact     artifact.ArtifactRef
	IsEnabled    bool
	Insert       agentskillsSpec.SkillInsert
	HasArguments bool
	HasResources bool
}

// DescribeArtifactSkill resolves and indexes a selected ArtifactRef through
// the ownership router before reading Agent Skills metadata. The Artifact
// record and its Collection membership, rather than reference shape, decide
// the owning feature adapter.
func (s *SkillRuntime) DescribeArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ArtifactSkillSummary, error) {
	if err := s.ensureConfigured(); err != nil {
		return ArtifactSkillSummary{}, err
	}
	if err := ref.Validate(); err != nil {
		return ArtifactSkillSummary{}, err
	}

	resolved, found := s.resolveArtifactSkill(ctx, ref)
	if !found {
		return ArtifactSkillSummary{}, fmt.Errorf(
			"%w: skill Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}

	records, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
		AllowSkills: []agentskillsSpec.SkillDef{resolved.Definition},
		Activity:    agentskillsSpec.SkillActivityAny,
	})
	if err != nil {
		return ArtifactSkillSummary{}, err
	}

	for _, record := range records {
		if record.Def != resolved.Definition {
			continue
		}
		return ArtifactSkillSummary{
			Artifact:     ref,
			IsEnabled:    true,
			Insert:       record.Insert,
			HasArguments: len(record.Arguments) != 0,
			HasResources: record.Resources.HasResources,
		}, nil
	}

	return ArtifactSkillSummary{}, fmt.Errorf(
		"%w: runtime did not index skill Artifact %q",
		basespec.ErrReferenceUnresolved,
		ref.ArtifactID,
	)
}
