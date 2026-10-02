package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

// ExportAgent exports canonical portable YAML for any available Agent Artifact,
// including protected built-ins, managed imported Agents, and
// repository-backed Agents. Export is read-only and does not imply that the
// exported Agent is editable or importable as a managed Agent.
// JSON export is intentionally not exposed by the management API.
func (a *API) ExportAgent(
	ctx context.Context,
	request AgentExportRequest,
) (AgentExportResult, error) {
	if a == nil || a.artifacts == nil {
		return AgentExportResult{}, model.ErrClosed
	}

	record, err := a.getAgentRecord(ctx, request.Agent)
	if err != nil {
		return AgentExportResult{}, err
	}
	if record.State != artifact.StateAvailable {
		return AgentExportResult{}, fmt.Errorf(
			"%w: Agent Artifact %q is unavailable",
			model.ErrReferenceUnresolved,
			record.ID,
		)
	}

	definitionValue, err := a.artifacts.GetDefinition(
		ctx,
		request.Agent,
	)
	if err != nil {
		return AgentExportResult{}, err
	}

	content, err := yamlutil.CanonicalObjectYAML(
		definitionValue.Body,
		model.MaxDefinitionBytes,
	)
	if err != nil {
		return AgentExportResult{}, err
	}

	managed, err := a.agentManaged(ctx, record, nil)
	if err != nil {
		return AgentExportResult{}, err
	}

	return AgentExportResult{
		Type:              declaration.TypeAgent,
		Name:              record.LogicalName,
		MediaType:         "application/yaml",
		SuggestedFileName: string(record.LogicalName) + ".agent.yaml",
		Content:           string(content),
		ContentDigest:     cryptoutil.DigestBytes(content),
		DefinitionDigest:  definitionValue.Digest,
		ArtifactRevision:  record.Revision,
		BuiltIn:           record.RootID == agentBuiltinRootID(),
		Managed:           managed,
	}, nil
}
