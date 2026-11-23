package mcpruntime

import (
	"context"
	"errors"
	"fmt"
	"maps"

	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
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

// PrepareWorkflowMCPContribution captures immutable MCP discovery, policy,
// prompt context, and provider Tool choices for Workflow Model and Agent plans.
func (p *WorkflowTargetPreparer) PrepareWorkflowMCPContribution(
	ctx context.Context,
	server sessionv3Runtime.ArtifactPin,
	selection sessionv3Runtime.MCPServerSelection,
	owner sessionv3Runtime.ConfigurationOwner,
) (sessionv3Runtime.WorkflowPreparedMCP, error) {
	if p == nil || p.runtime == nil {
		return sessionv3Runtime.WorkflowPreparedMCP{}, errors.New(
			"workflow mcp target preparer is unavailable",
		)
	}
	return prepareMCPContribution(ctx, p.runtime, server, selection, owner)
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

func prepareMCPContribution(
	ctx context.Context,
	runtime *mcpConnection.MCPRuntimeManager,
	server sessionv3Runtime.ArtifactPin,
	selection sessionv3Runtime.MCPServerSelection,
	owner sessionv3Runtime.ConfigurationOwner,
) (sessionv3Runtime.WorkflowPreparedMCP, error) {
	if runtime == nil {
		return sessionv3Runtime.WorkflowPreparedMCP{}, errors.New(
			"workflow mcp runtime is unavailable",
		)
	}
	if server.RootID == "" ||
		server.ArtifactID == "" ||
		server.Revision == 0 ||
		server.Digest == "" {
		return sessionv3Runtime.WorkflowPreparedMCP{}, errors.New(
			"workflow mcp server artifact pin is incomplete",
		)
	}

	serverID, err := workflowMCPServerID(server)
	if err != nil {
		return sessionv3Runtime.WorkflowPreparedMCP{}, err
	}
	status, err := runtime.Status(ctx, serverID)
	if err != nil {
		return sessionv3Runtime.WorkflowPreparedMCP{}, err
	}
	if status == nil || status.Status != mcpServer.MCPServerStatusReady {
		return sessionv3Runtime.WorkflowPreparedMCP{}, fmt.Errorf(
			"workflow mcp server %q is not connected",
			serverID,
		)
	}
	if status.SnapshotDigest == "" {
		return sessionv3Runtime.WorkflowPreparedMCP{}, errors.New(
			"workflow mcp server has no discovery snapshot digest",
		)
	}

	exposure := selection.ToolExposure
	if exposure == "" {
		exposure = sessionv3Runtime.MCPToolExposureAll
	}
	normalizedSelection := selection
	normalizedSelection.ToolExposure = exposure

	serverDisplay, contextValues, err := collectMCPServerContext(
		ctx,
		runtime,
		serverID,
		status,
		normalizedSelection,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowPreparedMCP{}, err
	}
	choices, err := collectMCPServerChoices(
		ctx,
		runtime,
		serverID,
		&server,
		normalizedSelection,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowPreparedMCP{}, err
	}

	preparedTools := make([]sessionv3Runtime.ToolDispatchPlan, 0, len(choices))
	for _, item := range choices {
		rawChoice, encodeErr := jsonutil.EncodeToJSONRaw(item.Choice)
		if encodeErr != nil {
			return sessionv3Runtime.WorkflowPreparedMCP{}, encodeErr
		}
		canonicalChoice, canonicalErr := jsonutil.CanonicalValueFrom(rawChoice)
		if canonicalErr != nil {
			return sessionv3Runtime.WorkflowPreparedMCP{}, canonicalErr
		}
		preparedTools = append(preparedTools, sessionv3Runtime.ToolDispatchPlan{
			Call:   item.Call,
			Choice: canonicalChoice,
		})
	}

	templates := make([]sessionv3Runtime.MCPResourceTemplateSelection, len(selection.ResourceTemplates))
	for index := range selection.ResourceTemplates {
		templates[index] = selection.ResourceTemplates[index]
		templates[index].Arguments = maps.Clone(selection.ResourceTemplates[index].Arguments)
	}
	prompts := make([]sessionv3Runtime.MCPPromptSelection, len(selection.Prompts))
	for index := range selection.Prompts {
		prompts[index] = selection.Prompts[index]
		prompts[index].Arguments = maps.Clone(selection.Prompts[index].Arguments)
	}

	output := sessionv3Runtime.WorkflowPreparedMCP{
		Server:                    server,
		IncludeServerInstructions: selection.IncludeServerInstructions,
		ToolExposure:              exposure,
		Tools:                     append([]sessionv3Runtime.MCPToolSelection(nil), selection.SelectedTools...),
		Resources:                 append([]sessionv3Runtime.MCPResourceSelection(nil), selection.Resources...),
		ResourceTemplates:         templates,
		Prompts:                   prompts,
		Owner:                     owner,
		Prepared:                  true,
		DiscoveryDigest:           status.SnapshotDigest,
		ServerDisplay:             serverDisplay,
		Context:                   contextValues,
		PreparedTools:             preparedTools,
	}
	return output, output.Validate()
}
