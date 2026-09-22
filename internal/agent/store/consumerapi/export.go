package consumerapi

import (
	"bytes"
	"context"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

func (a *API) ExportManagedAgent(
	ctx context.Context,
	request AgentExportRequest,
) (AgentExportResult, error) {
	if a == nil || a.managedAgentProfile == nil {
		return AgentExportResult{}, basespec.ErrClosed
	}

	current, err := a.loadExportableManagedAgent(ctx, request.Agent)
	if err != nil {
		return AgentExportResult{}, err
	}
	definitionValue, err := a.artifacts.GetDefinition(
		ctx,
		request.Agent,
	)
	if err != nil {
		return AgentExportResult{}, err
	}

	canonical, err := a.managedAgentProfile.Validate(
		definitionValue.Body,
	)
	if err != nil {
		return AgentExportResult{}, fmt.Errorf(
			"%w: Agent does not satisfy the managed import profile: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	entry, err := declaration.DecodeCanonicalEntryJSON(canonical)
	if err != nil {
		return AgentExportResult{}, err
	}
	normalizedEntry, err := agentDomain.NormalizeManagedAgentImport(entry)
	if err != nil {
		return AgentExportResult{}, err
	}
	normalizedCanonical, err := normalizedEntry.CanonicalJSON()
	if err != nil {
		return AgentExportResult{}, err
	}
	if !bytes.Equal(canonical, normalizedCanonical) {
		return AgentExportResult{}, fmt.Errorf(
			"%w: managed Agent Definition is not normalized",
			basespec.ErrDigestMismatch,
		)
	}
	canonical, err = a.managedAgentProfile.Validate(normalizedCanonical)
	if err != nil {
		return AgentExportResult{}, err
	}
	entry, err = declaration.DecodeCanonicalEntryJSON(canonical)
	if err != nil {
		return AgentExportResult{}, err
	}

	document, err := agentv1.DecodeAgentEntry(entry)
	if err != nil {
		return AgentExportResult{}, err
	}
	admission, err := agentDomain.ValidateManagedAgentImport(document)
	if err != nil {
		return AgentExportResult{}, err
	}
	if admission.HasErrors() {
		return AgentExportResult{}, fmt.Errorf(
			"%w: Agent does not satisfy managed import admission",
			basespec.ErrInvalid,
		)
	}

	content, err := yamlutil.CanonicalObjectYAML(
		canonical,
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

	setup := importMCPSetupDescriptors(admission.MCPSetupDescriptors)
	setup = a.bindMCPSetupArtifacts(ctx, request.Agent, setup)

	return AgentExportResult{
		Type:              declaration.TypeAgent,
		Name:              current.artifact.LogicalName,
		MediaType:         "application/yaml",
		SuggestedFileName: string(current.artifact.LogicalName) + ".agent.yaml",
		Content:           string(content),
		ContentDigest:     cryptoutil.DigestBytes(content),
		DefinitionDigest:  definitionValue.Digest,
		ArtifactRevision:  current.artifact.Revision,

		Resolution:          resolution,
		ResolutionIssue:     resolutionIssue,
		MCPSetupDescriptors: setup,
	}, nil
}
