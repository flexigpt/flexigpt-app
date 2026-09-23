package consumerapi

import (
	"context"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// ExportAgent exports the canonical portable declaration for any available
// Agent Artifact, including protected built-ins, managed imported Agents, and
// repository-backed Agents. Export is read-only and does not imply that the
// exported Agent is editable or importable as a managed Agent.
func (a *API) ExportAgent(
	ctx context.Context,
	request AgentExportRequest,
) (AgentExportResult, error) {
	if a == nil || a.artifacts == nil {
		return AgentExportResult{}, basespec.ErrClosed
	}

	current, err := a.GetAgentView(ctx, request.Agent)
	if err != nil {
		return AgentExportResult{}, err
	}
	if current.Artifact.State != artifact.StateAvailable {
		return AgentExportResult{}, fmt.Errorf(
			"%w: Agent Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			current.Artifact.ID,
		)
	}
	if current.Artifact.LogicalVersion != "" {
		return AgentExportResult{}, fmt.Errorf(
			"%w: Agent Artifact has an unexpected logical version",
			basespec.ErrDigestMismatch,
		)
	}

	definitionValue, err := a.artifacts.GetDefinition(
		ctx,
		request.Agent,
	)
	if err != nil {
		return AgentExportResult{}, err
	}

	if definitionValue.Kind != agentDomain.AgentArtifactKind {
		return AgentExportResult{}, fmt.Errorf(
			"%w: Agent Artifact Definition has another kind",
			basespec.ErrDigestMismatch,
		)
	}

	document, err := agentv1.DecodeAgentJSON(definitionValue.Body)
	if err != nil {
		return AgentExportResult{}, err
	}
	if document.Name != string(current.Artifact.LogicalName) {
		return AgentExportResult{}, fmt.Errorf(
			"%w: Agent declaration name differs from Artifact identity",
			basespec.ErrDigestMismatch,
		)
	}

	content, err := yamlutil.CanonicalObjectYAML(
		definitionValue.Body,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return AgentExportResult{}, err
	}

	var resolution *resolve.CapabilityPlan
	var resolutionIssue *resolve.ResolutionIssue
	value, err := a.ResolveAgentCapabilities(ctx, request.Agent)
	if err != nil {
		if ctx != nil && ctx.Err() != nil {
			return AgentExportResult{}, ctx.Err()
		}
		resolutionIssue = &resolve.ResolutionIssue{
			Code:    "agent.export.resolution-unavailable",
			Message: diagnostic.BoundedMessage(err.Error()),
		}
	} else {
		resolution = &value
	}

	setup := a.exportMCPSetupDescriptors(ctx, resolution)

	return AgentExportResult{
		Type:              declaration.TypeAgent,
		Name:              current.Artifact.LogicalName,
		MediaType:         "application/yaml",
		SuggestedFileName: string(current.Artifact.LogicalName) + ".agent.yaml",
		Content:           string(content),
		ContentDigest:     cryptoutil.DigestBytes(content),
		DefinitionDigest:  definitionValue.Digest,
		ArtifactRevision:  current.Artifact.Revision,
		BuiltIn:           current.BuiltIn,
		Managed:           current.Managed,

		Resolution:          resolution,
		ResolutionIssue:     resolutionIssue,
		MCPSetupDescriptors: setup,
	}, nil
}

func (a *API) exportMCPSetupDescriptors(
	ctx context.Context,
	plan *resolve.CapabilityPlan,
) []AgentMCPSetupDescriptor {
	if a == nil || a.artifacts == nil || plan == nil {
		return nil
	}

	seen := make(map[artifact.ArtifactRef]struct{})
	output := make([]AgentMCPSetupDescriptor, 0)

	for _, occurrence := range plan.Occurrences {
		if occurrence.Type != declaration.TypeMCP ||
			occurrence.Status != resolve.ResolutionAvailable ||
			occurrence.Artifact == nil {
			continue
		}

		ref := *occurrence.Artifact
		if _, duplicate := seen[ref]; duplicate {
			continue
		}
		seen[ref] = struct{}{}

		definitionValue, err := a.artifacts.GetDefinition(ctx, ref)
		if err != nil {
			continue
		}
		document, err := mcpv1.DecodeMCPJSON(definitionValue.Body)
		if err != nil {
			continue
		}

		output = append(output, importMCPSetupDescriptor(
			&ref,
			agentDomain.MCPSetupDescriptorForDocument(occurrence.Path, document),
		))
	}

	return output
}
