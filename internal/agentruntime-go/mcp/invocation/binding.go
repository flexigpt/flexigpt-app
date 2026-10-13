package invocation

import (
	"fmt"
	"strings"

	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/mcppolicy"
)

type MappedToolBinding struct {
	Server           mcpServer.ServerID         `json:"server"`
	ProviderToolName string                     `json:"providerToolName"`
	ChoiceID         string                     `json:"choiceID"`
	ToolName         string                     `json:"toolName"`
	ToolDigest       string                     `json:"toolDigest"`
	ApprovalRule     mcppolicy.MCPApprovalRule  `json:"approvalRule"`
	ExecutionMode    mcppolicy.MCPExecutionMode `json:"executionMode"`
	AppResourceURI   string                     `json:"appResourceUri,omitempty"`
	Visibility       []string                   `json:"visibility,omitempty"`
}

func (binding MappedToolBinding) Validate() error {
	if err := binding.Server.Validate(); err != nil {
		return err
	}

	for _, field := range []struct {
		name  string
		value string
	}{
		{"provider Tool name", binding.ProviderToolName},
		{"choice ID", binding.ChoiceID},
		{"Tool name", binding.ToolName},
		{"Tool digest", binding.ToolDigest},
	} {
		if strings.TrimSpace(field.value) == "" ||
			strings.TrimSpace(field.value) != field.value {
			return fmt.Errorf(
				"%w: MCP binding %s is required",
				mcpServer.ErrMCPInvalidRuntimeRequest,
				field.name,
			)
		}
	}

	switch binding.ApprovalRule {
	case mcppolicy.MCPApprovalRuleAllow,
		mcppolicy.MCPApprovalRuleAsk,
		mcppolicy.MCPApprovalRuleDeny:
	default:
		return fmt.Errorf(
			"%w: unsupported MCP binding approval rule %q",
			mcpServer.ErrMCPInvalidRuntimeRequest,
			binding.ApprovalRule,
		)
	}

	switch binding.ExecutionMode {
	case mcppolicy.MCPExecutionModeAuto,
		mcppolicy.MCPExecutionModeManual:
	default:
		return fmt.Errorf(
			"%w: unsupported MCP binding execution mode %q",
			mcpServer.ErrMCPInvalidRuntimeRequest,
			binding.ExecutionMode,
		)
	}

	return nil
}
