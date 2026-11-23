package sessionv3runtime

import (
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

func cloneWorkspaceMCPSelection(
	value sessionv3Runtime.MCPServerSelection,
) sessionv3Runtime.MCPServerSelection {
	output := value
	output.SelectedTools = append(
		[]sessionv3Runtime.MCPToolSelection(nil),
		value.SelectedTools...,
	)
	output.Resources = append(
		[]sessionv3Runtime.MCPResourceSelection(nil),
		value.Resources...,
	)
	output.ResourceTemplates = append(
		[]sessionv3Runtime.MCPResourceTemplateSelection(nil),
		value.ResourceTemplates...,
	)
	output.Prompts = append(
		[]sessionv3Runtime.MCPPromptSelection(nil),
		value.Prompts...,
	)
	return output
}
