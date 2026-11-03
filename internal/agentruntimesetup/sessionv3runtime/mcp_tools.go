package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"

	mcpApps "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/apps"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/mcpruntime"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) mcpToolChoices(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
) ([]preparedChoice, error) {
	if b.mcpTools != nil {
		values, err := b.mcpTools.Choices(ctx, req.Session.Defaults.MCP)
		return preparedChoicesFromDomain(values), err
	}

	out := []preparedChoice{}
	if b.deps.MCPRuntime == nil {
		return out, nil
	}
	for _, sel := range req.Session.Defaults.MCP {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if sel.ToolExposure == sessionv3Runtime.MCPToolExposureNone {
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
			if sel.ToolExposure != sessionv3Runtime.MCPToolExposureAll && string(sel.ToolExposure) != "" {
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
			call := sessionv3Runtime.ToolCall{
				Name:        t.ProviderToolName,
				ChoiceID:    t.ChoiceID,
				DisplayName: t.DisplayName,
				Description: t.Description,
				Binding: sessionv3Runtime.ToolBinding{
					Kind: sessionv3Runtime.ToolBindingKindMCP,
					MCP: &sessionv3Runtime.MCPToolBinding{
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

func (b *Binder) bindMCPToolCall(
	ctx context.Context,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.ToolCall, error) {
	if payload == nil || payload.MCP == nil {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"workflow MCP Tool target is missing",
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(payload.MCP.Server.RootID),
		ArtifactID: artifactModel.ArtifactID(payload.MCP.Server.ArtifactID),
	}
	serverID, err := mcpruntime.ServerIDForArtifact(ref)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}

	status, err := b.deps.MCPRuntime.Status(ctx, serverID)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}
	if status == nil || status.Status != mcpServer.MCPServerStatusReady {
		return sessionv3Runtime.ToolCall{}, fmt.Errorf(
			"workflow MCP server %q is not connected",
			serverID,
		)
	}

	tools, err := b.deps.MCPRuntime.ListTools(ctx, serverID)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}

	var selected *mcpServer.MCPToolCapability
	for index := range tools {
		if tools[index].ToolName == payload.MCP.ToolName {
			selected = &tools[index]
			break
		}
	}
	if selected == nil ||
		!selected.Enabled ||
		selected.TaskSupport == mcpServer.MCPTaskSupportRequired ||
		!mcpApps.ToolVisibleToModel(selected.App) {
		return sessionv3Runtime.ToolCall{}, fmt.Errorf(
			"workflow MCP Tool %q is unavailable",
			payload.MCP.ToolName,
		)
	}

	call := payload.Call
	call.Name = selected.ProviderToolName
	call.ChoiceID = selected.ChoiceID
	call.DisplayName = selected.DisplayName
	call.Description = selected.Description
	call.Binding = sessionv3Runtime.ToolBinding{
		Kind: sessionv3Runtime.ToolBindingKindMCP,
		MCP: &sessionv3Runtime.MCPToolBinding{
			ServerArtifact:   &payload.MCP.Server,
			ServerID:         string(serverID),
			ProviderToolName: selected.ProviderToolName,
			ChoiceID:         selected.ChoiceID,
			ToolName:         selected.ToolName,
			ToolDigest:       selected.Digest,
			ApprovalRule:     string(selected.ApprovalRule),
			ExecutionMode:    string(selected.ExecutionMode),
		},
	}
	return call, nil
}
