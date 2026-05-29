package spec

type PutMCPServerPayload struct {
	DisplayName string           `json:"displayName" required:"true"`
	Enabled     bool             `json:"enabled"     required:"true"`
	Transport   MCPTransportType `json:"transport"   required:"true"`

	Stdio          *MCPStdioConfig          `json:"stdio,omitempty"`
	StreamableHTTP *MCPStreamableHTTPConfig `json:"streamableHttp,omitempty"`

	Availability MCPServerAvailability `json:"availability,omitempty"`
	TrustLevel   MCPTrustLevel         `json:"trustLevel,omitempty"`

	DefaultPolicy *MCPServerPolicy                 `json:"defaultPolicy,omitempty"`
	ToolPolicies  map[string]MCPToolPolicyOverride `json:"toolPolicies,omitempty"`
	AppsPolicy    *MCPAppsPolicy                   `json:"appsPolicy,omitempty"`
	AuthRef       *MCPAuthRef                      `json:"authRef,omitempty"`
}

type PutMCPServerRequest struct {
	ServerID MCPServerID `path:"serverID" required:"true"`
	Body     *PutMCPServerPayload
}

type PutMCPServerResponse struct{}

type GetMCPServerRequest struct {
	ServerID       MCPServerID `path:"serverID" required:"true"`
	IncludeDeleted bool        `                                query:"includeDeleted"`
}

type GetMCPServerResponse struct {
	Body *MCPServerConfig
}

type ListMCPServersRequest struct {
	ServerIDs []MCPServerID `query:"serverIDs"`
	Enabled   *bool         `query:"enabled"`
	PageSize  int           `query:"pageSize"`
	PageToken string        `query:"pageToken"`
}

type ListMCPServersResponseBody struct {
	Servers       []MCPServerConfig `json:"servers"`
	NextPageToken *string           `json:"nextPageToken,omitempty"`
}

type ListMCPServersResponse struct {
	Body *ListMCPServersResponseBody
}

type PatchMCPServerEnabledRequestBody struct {
	Enabled bool `json:"enabled" required:"true"`
}

type PatchMCPServerEnabledRequest struct {
	ServerID MCPServerID `path:"serverID" required:"true"`
	Body     *PatchMCPServerEnabledRequestBody
}

type PatchMCPServerEnabledResponse struct{}

type PatchMCPServerPolicyPayload struct {
	DefaultPolicy *MCPServerPolicy                 `json:"defaultPolicy,omitempty"`
	ToolPolicies  map[string]MCPToolPolicyOverride `json:"toolPolicies,omitempty"`
	AppsPolicy    *MCPAppsPolicy                   `json:"appsPolicy,omitempty"`
}

type PatchMCPServerPolicyRequest struct {
	ServerID MCPServerID `path:"serverID" required:"true"`
	Body     *PatchMCPServerPolicyPayload
}

type PatchMCPServerPolicyResponse struct{}

type DeleteMCPServerRequest struct {
	ServerID MCPServerID `path:"serverID" required:"true"`
}

type DeleteMCPServerResponse struct{}

type ConnectMCPServerRequest struct {
	ServerID MCPServerID `path:"serverID" required:"true"`
}

type ConnectMCPServerResponse struct {
	Body *MCPServerRuntimeSnapshot
}

type DisconnectMCPServerRequest struct {
	ServerID MCPServerID `path:"serverID" required:"true"`
}

type DisconnectMCPServerResponse struct{}

type RefreshMCPServerRequest struct {
	ServerID MCPServerID `path:"serverID" required:"true"`
}

type RefreshMCPServerResponse struct {
	Body *MCPServerRuntimeSnapshot
}

type GetMCPServerStatusRequest struct {
	ServerID MCPServerID `path:"serverID" required:"true"`
}

type GetMCPServerStatusResponse struct {
	Body *MCPServerRuntimeSnapshot
}

type ListMCPServerToolsRequest struct {
	ServerID  MCPServerID `path:"serverID" required:"true"`
	PageSize  int         `                                query:"pageSize"`
	PageToken string      `                                query:"pageToken"`
}

type ListMCPServerToolsResponseBody struct {
	Tools         []MCPToolCapability `json:"tools"`
	NextPageToken *string             `json:"nextPageToken,omitempty"`
}

type ListMCPServerToolsResponse struct {
	Body *ListMCPServerToolsResponseBody
}

type ListMCPServerResourcesRequest struct {
	ServerID  MCPServerID `path:"serverID" required:"true"`
	PageSize  int         `                                query:"pageSize"`
	PageToken string      `                                query:"pageToken"`
}

type ListMCPServerResourcesResponseBody struct {
	Resources     []MCPResourceRef `json:"resources"`
	NextPageToken *string          `json:"nextPageToken,omitempty"`
}

type ListMCPServerResourcesResponse struct {
	Body *ListMCPServerResourcesResponseBody
}

type ListMCPServerResourceTemplatesRequest struct {
	ServerID  MCPServerID `path:"serverID" required:"true"`
	PageSize  int         `                                query:"pageSize"`
	PageToken string      `                                query:"pageToken"`
}

type ListMCPServerResourceTemplatesResponseBody struct {
	ResourceTemplates []MCPResourceTemplateRef `json:"resourceTemplates"`
	NextPageToken     *string                  `json:"nextPageToken,omitempty"`
}

type ListMCPServerResourceTemplatesResponse struct {
	Body *ListMCPServerResourceTemplatesResponseBody
}

type ListMCPServerPromptsRequest struct {
	ServerID  MCPServerID `path:"serverID" required:"true"`
	PageSize  int         `                                query:"pageSize"`
	PageToken string      `                                query:"pageToken"`
}

type ListMCPServerPromptsResponseBody struct {
	Prompts       []MCPPromptRef `json:"prompts"`
	NextPageToken *string        `json:"nextPageToken,omitempty"`
}

type ListMCPServerPromptsResponse struct {
	Body *ListMCPServerPromptsResponseBody
}

type EvaluateMCPToolCallRequest struct {
	Body *InvokeMCPToolRequestBody
}

type EvaluateMCPToolCallResponse struct {
	Body *MCPApprovalEvaluation
}

type ResolveMCPApprovalRequestBody struct {
	ApprovalID string                `json:"approvalID" required:"true"`
	Resolution MCPApprovalResolution `json:"resolution" required:"true"`
}

type ResolveMCPApprovalRequest struct {
	Body *ResolveMCPApprovalRequestBody
}

type ResolveMCPApprovalResponse struct {
	Body *MCPApprovalToken
}

type ListPendingMCPOAuthAuthorizationsRequest struct{}

type ListPendingMCPOAuthAuthorizationsResponseBody struct {
	Authorizations []MCPOAuthAuthorization `json:"authorizations"`
}

type ListPendingMCPOAuthAuthorizationsResponse struct {
	Body *ListPendingMCPOAuthAuthorizationsResponseBody
}

type CancelPendingMCPOAuthAuthorizationRequest struct {
	ServerID MCPServerID `path:"serverID" required:"true"`
}

type CancelPendingMCPOAuthAuthorizationResponse struct{}
