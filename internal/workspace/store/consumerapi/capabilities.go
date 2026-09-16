package consumerapi

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

func (a *StoreAPI) ResolveArtifactCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	if a == nil || a.resolver == nil {
		return resolve.CapabilityPlan{}, basespec.ErrClosed
	}
	return a.resolver.ResolveCapabilities(
		ctx,
		ref,
	)
}

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
	workspace, graph, err := a.resolveCurrentWorkspace(ctx, ref)
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
		Occurrences:     make([]WorkspaceCapabilityOccurrence, 0),
		PromptArtifacts: make([]artifact.ArtifactRef, 0),
		SkillArtifacts:  make([]artifact.ArtifactRef, 0),
		MCPArtifacts:    make([]artifact.ArtifactRef, 0),
	}
	collector := &workspaceCapabilityCollector{
		capabilities:     &capabilities,
		promptSeen:       make(map[artifact.ArtifactRef]struct{}),
		skillSeen:        make(map[artifact.ArtifactRef]struct{}),
		mcpSeen:          make(map[artifact.ArtifactRef]struct{}),
		occurrenceByPath: make(map[string]int),
	}
	for index, relationship := range graph.Root.Workspace.RootResults {
		if err := collector.collectRelationship(
			capabilityPath("roots", index),
			relationship,
			true,
		); err != nil {
			return workspaceDomain.Workspace{},
				WorkspaceCapabilityPlan{},
				err
		}
	}
	sortWorkspaceArtifactRefs(capabilities.PromptArtifacts)
	sortWorkspaceArtifactRefs(capabilities.SkillArtifacts)
	sortWorkspaceArtifactRefs(capabilities.MCPArtifacts)
	capabilities.Complete = resolve.RequireComplete(
		capabilities.Occurrences,
	) == nil
	return workspace, capabilities, nil
}

type workspaceCapabilityCollector struct {
	capabilities     *WorkspaceCapabilityPlan
	promptSeen       map[artifact.ArtifactRef]struct{}
	skillSeen        map[artifact.ArtifactRef]struct{}
	mcpSeen          map[artifact.ArtifactRef]struct{}
	occurrenceByPath map[string]int
}

func (c *workspaceCapabilityCollector) collectRelationship(
	path string,
	relationship resolve.ResolvedRelationship,
	ambient bool,
) error {
	header := relationship.Declared.Header()
	occurrence := WorkspaceCapabilityOccurrence{
		Path:   path,
		Type:   header.Type,
		Name:   basespec.LogicalName(header.Name),
		Status: relationship.Status,
	}
	if relationship.Issue != nil {
		occurrence.Code = relationship.Issue.Code
		occurrence.Message = relationship.Issue.Message
	}
	if relationship.Resolved != nil {
		if ref, found := relationship.Resolved.ArtifactRef(); found {
			value := ref
			occurrence.Artifact = &value
		}
	}

	c.occurrenceByPath[path] = len(c.capabilities.Occurrences)
	c.capabilities.Occurrences = append(
		c.capabilities.Occurrences,
		occurrence,
	)

	if !relationship.IsAvailable() {
		return nil
	}
	if relationship.Resolved == nil {
		c.markUnavailable(
			path,
			"workspace.artifact.unresolved",
			"resolved Workspace relationship has no Artifact entry",
		)
		return nil
	}
	return c.collectEntry(path, relationship.Resolved, ambient)
}

func (c *workspaceCapabilityCollector) collectEntry(
	path string,
	entry *resolve.ResolvedEntry,
	ambient bool,
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
		for index, member := range entry.MemberResults {
			if err := c.collectRelationship(
				capabilityPath(path+"/members", index),
				member,
				ambient,
			); err != nil {
				return err
			}
		}
		if entry.ProgramResult != nil {
			if err := c.collectRelationship(
				path+"/program",
				*entry.ProgramResult,
				false,
			); err != nil {
				return err
			}
		}
		return nil

	case declaration.TypeInstruction, declaration.TypeContext:
		if !ambient {
			return nil
		}
		ref, err := workspaceArtifactRef(entry)
		if err != nil {
			c.markUnavailable(
				path,
				workspaceDomain.DiagnosticCodeArtifactUnresolved,
				err.Error(),
			)
			return nil
		}
		appendUniqueWorkspaceArtifact(
			&c.capabilities.PromptArtifacts,
			c.promptSeen,
			ref,
		)
		return nil

	case declaration.TypeSkill:
		if !ambient {
			return nil
		}
		ref, err := workspaceArtifactRef(entry)
		if err != nil {
			c.markUnavailable(
				path,
				workspaceDomain.DiagnosticCodeArtifactUnresolved,
				err.Error(),
			)
			return nil
		}
		appendUniqueWorkspaceArtifact(
			&c.capabilities.SkillArtifacts,
			c.skillSeen,
			ref,
		)
		return nil

	case declaration.TypeMCP:
		if !ambient {
			return nil
		}
		ref, err := workspaceArtifactRef(entry)
		if err != nil {
			c.markUnavailable(
				path,
				workspaceDomain.DiagnosticCodeArtifactUnresolved,
				err.Error(),
			)
			return nil
		}
		appendUniqueWorkspaceArtifact(
			&c.capabilities.MCPArtifacts,
			c.mcpSeen,
			ref,
		)
		return nil

	case declaration.TypeLoop:
		if entry.Loop != nil && entry.Loop.BodyResult != nil {
			return c.collectRelationship(
				path+"/body",
				*entry.Loop.BodyResult,
				false,
			)
		}
		return nil

	case declaration.TypeWorkflow:
		if entry.Workflow == nil {
			return nil
		}
		for index, node := range entry.Workflow.Nodes {
			if node.TargetResult == nil {
				continue
			}
			if err := c.collectRelationship(
				capabilityPath(path+"/nodes", index)+"/target",
				*node.TargetResult,
				false,
			); err != nil {
				return err
			}
		}
		return nil

	case declaration.TypeWorkspace:
		if entry.Workspace == nil {
			return nil
		}
		for index, root := range entry.Workspace.RootResults {
			if err := c.collectRelationship(
				capabilityPath(path+"/roots", index),
				root,
				ambient,
			); err != nil {
				return err
			}
		}
		return nil

	default:
		return nil
	}
}

func (c *workspaceCapabilityCollector) markUnavailable(
	path string,
	code string,
	message string,
) {
	index, found := c.occurrenceByPath[path]
	if !found {
		return
	}
	c.capabilities.Occurrences[index].Status = resolve.ResolutionUnavailable
	c.capabilities.Occurrences[index].Code = code
	c.capabilities.Occurrences[index].Message = message
}

func capabilityPath(
	parent string,
	index int,
) string {
	return parent + "/" + strconv.Itoa(index)
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

func sortWorkspaceArtifactRefs(values []artifact.ArtifactRef) {
	sort.Slice(values, func(left, right int) bool {
		return string(values[left].RootID)+"\x00"+string(values[left].ArtifactID) <
			string(values[right].RootID)+"\x00"+string(values[right].ArtifactID)
	})
}
