package consumerapi

import (
	"context"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
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

	current, err := a.loadEditableManagedAgent(ctx, request.Agent, 0)
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

	resolution, err := a.ResolveAgentCapabilities(ctx, request.Agent)
	if err != nil {
		return AgentExportResult{}, err
	}

	setup := importMCPSetupDescriptors(admission.MCPSetupDescriptors)
	setup = a.bindMCPSetupArtifacts(ctx, request.Agent, setup)

	return AgentExportResult{
		Type:                declaration.TypeAgent,
		Name:                current.artifact.LogicalName,
		MediaType:           "application/yaml",
		SuggestedFileName:   string(current.artifact.LogicalName) + ".agent.yaml",
		Content:             string(content),
		ContentDigest:       cryptoutil.DigestBytes(content),
		DefinitionDigest:    definitionValue.Digest,
		ArtifactRevision:    current.artifact.Revision,
		Resolution:          resolution,
		MCPSetupDescriptors: setup,
	}, nil
}
