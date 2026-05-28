package spec

type MCPApprovalSummary struct {
	ServerID          MCPServerID   `json:"serverID"`
	ServerDisplayName string        `json:"serverDisplayName,omitempty"`
	ToolName          string        `json:"toolName"`
	ToolDigest        string        `json:"toolDigest,omitempty"`
	Risk              MCPToolRisk   `json:"risk"`
	Arguments         JSONRawString `json:"arguments,omitempty"`
}

type MCPApprovalEvaluation struct {
	Decision   MCPApprovalDecision `json:"decision"`
	Reason     string              `json:"reason,omitempty"`
	ApprovalID string              `json:"approvalID,omitempty"`
	Summary    *MCPApprovalSummary `json:"summary,omitempty"`
}

type MCPApprovalToken struct {
	ApprovalID string `json:"approvalID"`
	Token      string `json:"token"`
	ExpiresAt  string `json:"expiresAt"`
}
