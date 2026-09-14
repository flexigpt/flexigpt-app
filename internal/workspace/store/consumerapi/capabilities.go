package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

func (a *StoreAPI) ResolveWorkspaceCapabilities(
	ctx context.Context,
	ref WorkspaceRef,
) (WorkspaceCapabilityPlan, error) {
	_, capabilities, err := a.resolveWorkspaceCapabilities(ctx, ref)
	if err != nil {
		return WorkspaceCapabilityPlan{}, err
	}
	return capabilities, nil
}

func (a *StoreAPI) resolveWorkspaceCapabilities(
	ctx context.Context,
	ref WorkspaceRef,
) (
	workspaceDomain.Workspace,
	WorkspaceCapabilityPlan,
	error,
) {
	workspace, _, graph, err := a.refreshAndResolveWorkspace(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, WorkspaceCapabilityPlan{}, err
	}
	if graph.Root == nil || graph.Root.Workspace == nil {
		return workspaceDomain.Workspace{}, WorkspaceCapabilityPlan{}, fmt.Errorf(
			"%w: Workspace did not resolve to Workspace roots",
			basespec.ErrReferenceUnresolved,
		)
	}

	capabilities := WorkspaceCapabilityPlan{
		Workspace:       workspace.Ref(),
		PromptArtifacts: make([]artifact.ArtifactRef, 0),
		SkillArtifacts:  make([]artifact.ArtifactRef, 0),
		MCPArtifacts:    make([]artifact.ArtifactRef, 0),
	}
	collector := workspaceCapabilityCollector{
		capabilities: &capabilities,
		promptSeen:   make(map[artifact.ArtifactRef]struct{}),
		skillSeen:    make(map[artifact.ArtifactRef]struct{}),
		mcpSeen:      make(map[artifact.ArtifactRef]struct{}),
	}
	for _, entry := range graph.Root.Workspace.Roots {
		if err := collector.collect(entry); err != nil {
			return workspaceDomain.Workspace{},
				WorkspaceCapabilityPlan{},
				err
		}
	}
	return workspace, capabilities, nil
}

type workspaceCapabilityCollector struct {
	capabilities *WorkspaceCapabilityPlan
	promptSeen   map[artifact.ArtifactRef]struct{}
	skillSeen    map[artifact.ArtifactRef]struct{}
	mcpSeen      map[artifact.ArtifactRef]struct{}
}

func (c workspaceCapabilityCollector) collect(
	entry *resolve.ResolvedEntry,
) error {
	if entry == nil {
		return fmt.Errorf(
			"%w: Workspace resolved an empty entry",
			basespec.ErrReferenceUnresolved,
		)
	}

	switch entry.Type {
	case declaration.TypeCollection,
		declaration.TypeAgent,
		declaration.TypeTeam:
		for _, member := range entry.Members {
			if err := c.collect(member); err != nil {
				return err
			}
		}
		return nil

	case declaration.TypeInstruction, declaration.TypeContext:
		ref, err := workspaceArtifactRef(entry)
		if err != nil {
			return err
		}
		appendUniqueWorkspaceArtifact(
			&c.capabilities.PromptArtifacts,
			c.promptSeen,
			ref,
		)
		return nil

	case declaration.TypeSkill:
		ref, err := workspaceArtifactRef(entry)
		if err != nil {
			return err
		}
		appendUniqueWorkspaceArtifact(
			&c.capabilities.SkillArtifacts,
			c.skillSeen,
			ref,
		)
		return nil

	case declaration.TypeMCP:
		ref, err := workspaceArtifactRef(entry)
		if err != nil {
			return err
		}
		appendUniqueWorkspaceArtifact(
			&c.capabilities.MCPArtifacts,
			c.mcpSeen,
			ref,
		)
		return nil

	default:
		// Workflows and Loops are executable graph structure. They are not
		// ambient Workspace prompt, Skill, or MCP capabilities. Their future
		// consumer receives the original resolved graph.
		return nil
	}
}

func workspaceArtifactRef(
	entry *resolve.ResolvedEntry,
) (artifact.ArtifactRef, error) {
	ref, found := entry.ArtifactRef()
	if !found {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: current Workspace runtime requires a named source-backed %q Artifact",
			basespec.ErrReferenceUnresolved,
			entry.Type,
		)
	}
	return ref, nil
}

func appendUniqueWorkspaceArtifact(
	values *[]artifact.ArtifactRef,
	seen map[artifact.ArtifactRef]struct{},
	ref artifact.ArtifactRef,
) {
	if _, found := seen[ref]; found {
		return
	}
	seen[ref] = struct{}{}
	*values = append(*values, ref)
}
