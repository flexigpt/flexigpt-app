package sessionv3runtime

import (
	"context"

	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/mcpruntime"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) mcpToolChoices(ctx context.Context, req sessionv3.WorkDispatchRequest) ([]preparedChoice, error) {
	out := []preparedChoice{}
	if b.deps.MCPRuntime == nil {
		return out, nil
	}
	for _, sel := range req.Session.Defaults.MCP {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if sel.ToolExposure == sessionv3.MCPToolExposureNone {
			continue
		}
		ref := artifactModel.ArtifactRef{
			RootID:     rootModel.RootID(sel.Server.RootID),
			ArtifactID: artifactModel.ArtifactID(sel.Server.ArtifactID),
		}
		if err := ref.Validate(); err != nil {
			continue
		}
		serverID, err := mcpruntime.ServerIDForArtifact(ref)
		if err != nil {
			continue
		}
		tools, err := b.deps.MCPRuntime.ListTools(ctx, serverID)
		if err != nil {
			continue
		}
		for _, t := range tools {
			if !t.Enabled {
				continue
			}
			if t.TaskSupport == mcpServer.MCPTaskSupportRequired {
				continue
			}
			if sel.ToolExposure != sessionv3.MCPToolExposureAll && string(sel.ToolExposure) != "" {
				found := false
				for _, s := range sel.SelectedTools {
					if s.ToolName == t.ToolName {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			args := getEmptySchema()
			if len(t.InputSchema) != 0 {
				args = t.InputSchema
			}
			choice := inferenceSpec.ToolChoice{
				Type:        inferenceSpec.ToolTypeFunction,
				ID:          t.ChoiceID,
				Name:        t.ProviderToolName,
				Description: t.Description,
				Arguments:   args,
			}
			call := sessionv3.ToolCall{
				Name:        t.ProviderToolName,
				ChoiceID:    t.ChoiceID,
				DisplayName: t.DisplayName,
				Description: t.Description,
				Binding: sessionv3.ToolBinding{
					Kind: sessionv3.ToolBindingKindMCP,
					MCP: &sessionv3.MCPToolBinding{
						ServerID:         string(t.Server),
						ProviderToolName: t.ProviderToolName,
						ChoiceID:         t.ChoiceID,
						ToolName:         t.ToolName,
						ToolDigest:       t.Digest,
						ApprovalRule:     string(t.ApprovalRule),
						ExecutionMode:    string(t.ExecutionMode),
					},
				},
			}
			out = append(out, preparedChoice{Choice: choice, Call: call})
		}
	}
	return out, nil
}
