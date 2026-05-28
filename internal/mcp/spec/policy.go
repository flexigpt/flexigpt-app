package spec

type MCPServerPolicy struct {
	DefaultApprovalRule  MCPApprovalRule  `json:"defaultApprovalRule"`
	DefaultExecutionMode MCPExecutionMode `json:"defaultExecutionMode"`

	RequireApprovalForUnknownRisk bool `json:"requireApprovalForUnknownRisk"`
	RequireApprovalForWrite       bool `json:"requireApprovalForWrite"`
	RequireApprovalForDestructive bool `json:"requireApprovalForDestructive"`
}

type MCPToolPolicyOverride struct {
	ToolName string `json:"toolName"`

	ApprovalRule  *MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode *MCPExecutionMode `json:"executionMode,omitempty"`

	AllowStaleDigest bool   `json:"allowStaleDigest,omitempty"`
	ExpectedDigest   string `json:"expectedDigest,omitempty"`
}

type MCPAppsPolicy struct {
	Enabled                          bool `json:"enabled"`
	AllowAppInitiatedToolCalls       bool `json:"allowAppInitiatedToolCalls"`
	RequireApprovalForOpenLink       bool `json:"requireApprovalForOpenLink"`
	RequireApprovalForContextUpdates bool `json:"requireApprovalForContextUpdates"`
}

func DefaultMCPServerPolicy() MCPServerPolicy {
	return MCPServerPolicy{
		DefaultApprovalRule:           MCPApprovalRuleAsk,
		DefaultExecutionMode:          MCPExecutionModeManual,
		RequireApprovalForUnknownRisk: true,
		RequireApprovalForWrite:       true,
		RequireApprovalForDestructive: true,
	}
}
