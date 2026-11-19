package mcpruntime

import (
	"context"
	"errors"
	"fmt"

	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

type WorkflowTargetPreparer struct {
	runtime *mcpConnection.MCPRuntimeManager
}

func NewWorkflowTargetPreparer(
	runtime *mcpConnection.MCPRuntimeManager,
) (*WorkflowTargetPreparer, error) {
	if runtime == nil {
		return nil, errors.New("workflow MCP target preparer requires an MCP runtime")
	}
	return &WorkflowTargetPreparer{runtime: runtime}, nil
}

func (p *WorkflowTargetPreparer) PrepareWorkflowMCPTool(
	ctx context.Context,
	server sessionv3Runtime.ArtifactPin,
	toolName string,
) (sessionv3Runtime.WorkflowMCP, error) {
	if p == nil || p.runtime == nil {
		return sessionv3Runtime.WorkflowMCP{}, errors.New(
			"workflow MCP target preparer is unavailable",
		)
	}
	return prepareWorkflowMCPTool(ctx, p.runtime, server, toolName)
}

func prepareWorkflowMCPTool(
	ctx context.Context,
	runtime *mcpConnection.MCPRuntimeManager,
	server sessionv3Runtime.ArtifactPin,
	toolName string,
) (sessionv3Runtime.WorkflowMCP, error) {
	if runtime == nil {
		return sessionv3Runtime.WorkflowMCP{}, errors.New(
			"workflow MCP runtime is unavailable",
		)
	}
	if server.RootID == "" ||
		server.ArtifactID == "" ||
		server.Revision == 0 ||
		server.Digest == "" {
		return sessionv3Runtime.WorkflowMCP{}, errors.New(
			"workflow MCP server Artifact pin is incomplete",
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(server.RootID),
		ArtifactID: artifactModel.ArtifactID(server.ArtifactID),
	}
	if err := ref.Validate(); err != nil {
		return sessionv3Runtime.WorkflowMCP{}, err
	}

	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return sessionv3Runtime.WorkflowMCP{}, err
	}
	status, err := runtime.Status(ctx, serverID)
	if err != nil {
		return sessionv3Runtime.WorkflowMCP{}, err
	}
	if status == nil || status.Status != mcpServer.MCPServerStatusReady {
		return sessionv3Runtime.WorkflowMCP{}, fmt.Errorf(
			"workflow MCP server %q is not connected",
			serverID,
		)
	}
	if status.SnapshotDigest == "" {
		return sessionv3Runtime.WorkflowMCP{}, errors.New(
			"workflow MCP server has no discovery snapshot digest",
		)
	}

	tools, err := runtime.ListTools(ctx, serverID)
	if err != nil {
		return sessionv3Runtime.WorkflowMCP{}, err
	}
	for _, tool := range tools {
		if tool.ToolName != toolName {
			continue
		}
		if !v3MCPToolAvailable(tool) {
			return sessionv3Runtime.WorkflowMCP{}, fmt.Errorf(
				"workflow MCP tool %q is unavailable",
				toolName,
			)
		}
		return sessionv3Runtime.WorkflowMCP{
			Server:           server,
			ToolName:         tool.ToolName,
			ProviderToolName: tool.ProviderToolName,
			ChoiceID:         tool.ChoiceID,
			ToolDigest:       tool.Digest,
			ApprovalRule:     string(tool.ApprovalRule),
			ExecutionMode:    string(tool.ExecutionMode),
			DiscoveryDigest:  status.SnapshotDigest,
		}, nil
	}
	return sessionv3Runtime.WorkflowMCP{}, fmt.Errorf(
		"workflow MCP tool %q is unavailable",
		toolName,
	)
}
