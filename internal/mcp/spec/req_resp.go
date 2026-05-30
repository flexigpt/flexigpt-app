package spec

import "github.com/flexigpt/flexigpt-app/internal/bundleitemutils"

type PutMCPBundleRequestBody struct {
	Slug        bundleitemutils.BundleSlug `json:"slug"                  required:"true"`
	DisplayName string                     `json:"displayName"           required:"true"`
	IsEnabled   bool                       `json:"isEnabled"             required:"true"`
	Description string                     `json:"description,omitempty"`
}

type PutMCPBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	Body     *PutMCPBundleRequestBody
}

type PutMCPBundleResponse struct{}

type PatchMCPBundleRequestBody struct {
	IsEnabled bool `json:"isEnabled" required:"true"`
}

type PatchMCPBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	Body     *PatchMCPBundleRequestBody
}

type PatchMCPBundleResponse struct{}

type DeleteMCPBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
}

type DeleteMCPBundleResponse struct{}

type ListMCPBundlesRequest struct {
	BundleIDs       []bundleitemutils.BundleID `query:"bundleIDs"`
	IncludeDisabled bool                       `query:"includeDisabled"`
	PageSize        int                        `query:"pageSize"`
	PageToken       string                     `query:"pageToken"`
}

type ListMCPBundlesResponseBody struct {
	Bundles       []MCPBundle `json:"bundles"`
	NextPageToken *string     `json:"nextPageToken,omitempty"`
}

type ListMCPBundlesResponse struct {
	Body *ListMCPBundlesResponseBody
}

type GetMCPServerAuthStatusRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`
}

type GetMCPServerAuthStatusResponse struct {
	Body *MCPAuthStatus
}

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
}

type PutMCPServerRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	Body *PutMCPServerPayload
}

type PutMCPServerResponse struct{}

type GetMCPServerRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	IncludeDeleted bool `query:"includeDeleted"`
}

type GetMCPServerResponse struct {
	Body *MCPServerConfig
}

type ListMCPServersRequest struct {
	BundleID        bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerIDs       []MCPServerID            `                                query:"serverIDs"`
	Enabled         *bool                    `                                query:"enabled"`
	IncludeDisabled bool                     `                                query:"includeDisabled"`
	PageSize        int                      `                                query:"pageSize"`
	PageToken       string                   `                                query:"pageToken"`
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
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	Body *PatchMCPServerEnabledRequestBody
}

type PatchMCPServerEnabledResponse struct{}

type PatchMCPServerPolicyPayload struct {
	DefaultPolicy *MCPServerPolicy                 `json:"defaultPolicy,omitempty"`
	ToolPolicies  map[string]MCPToolPolicyOverride `json:"toolPolicies,omitempty"`
	AppsPolicy    *MCPAppsPolicy                   `json:"appsPolicy,omitempty"`
}

type PatchMCPServerPolicyRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	Body *PatchMCPServerPolicyPayload
}

type PatchMCPServerPolicyResponse struct{}

type DeleteMCPServerRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`
}

type DeleteMCPServerResponse struct{}

type ConnectMCPServerRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`
}

type ConnectMCPServerResponse struct {
	Body *MCPServerRuntimeSnapshot
}

type DisconnectMCPServerRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`
}

type DisconnectMCPServerResponse struct{}

type RefreshMCPServerRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`
}

type RefreshMCPServerResponse struct {
	Body *MCPServerRuntimeSnapshot
}

type GetMCPServerStatusRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`
}

type GetMCPServerStatusResponse struct {
	Body *MCPServerRuntimeSnapshot
}

type ListMCPServerToolsRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	PageSize  int    `query:"pageSize"`
	PageToken string `query:"pageToken"`
}

type ListMCPServerToolsResponseBody struct {
	Tools         []MCPToolCapability `json:"tools"`
	NextPageToken *string             `json:"nextPageToken,omitempty"`
}

type ListMCPServerToolsResponse struct {
	Body *ListMCPServerToolsResponseBody
}

type ListMCPServerResourcesRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	PageSize  int    `query:"pageSize"`
	PageToken string `query:"pageToken"`
}

type ListMCPServerResourcesResponseBody struct {
	Resources     []MCPResourceRef `json:"resources"`
	NextPageToken *string          `json:"nextPageToken,omitempty"`
}

type ListMCPServerResourcesResponse struct {
	Body *ListMCPServerResourcesResponseBody
}

type ListMCPServerResourceTemplatesRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	PageSize  int    `query:"pageSize"`
	PageToken string `query:"pageToken"`
}

type ListMCPServerResourceTemplatesResponseBody struct {
	ResourceTemplates []MCPResourceTemplateRef `json:"resourceTemplates"`
	NextPageToken     *string                  `json:"nextPageToken,omitempty"`
}

type ListMCPServerResourceTemplatesResponse struct {
	Body *ListMCPServerResourceTemplatesResponseBody
}

type ListMCPServerPromptsRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	PageSize  int    `query:"pageSize"`
	PageToken string `query:"pageToken"`
}

type ListMCPServerPromptsResponseBody struct {
	Prompts       []MCPPromptRef `json:"prompts"`
	NextPageToken *string        `json:"nextPageToken,omitempty"`
}

type ListMCPServerPromptsResponse struct {
	Body *ListMCPServerPromptsResponseBody
}

type EvaluateMCPToolCallRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

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
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`
}

type CancelPendingMCPOAuthAuthorizationResponse struct{}

type GetMCPServerAuthHealthRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`
}

type GetMCPServerAuthHealthResponse struct {
	Body *MCPAuthHealth
}

type PutMCPServerSecretRequestBody struct {
	Kind   MCPSecretKind `json:"kind"   required:"true"`
	Slot   string        `json:"slot"   required:"true"`
	Secret string        `json:"secret" required:"true"`
}

type PutMCPServerSecretRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	Body *PutMCPServerSecretRequestBody
}

type PutMCPServerSecretResponseBody struct {
	SecretRef string `json:"secretRef"`
	SHA256    string `json:"sha256,omitempty"`
	NonEmpty  bool   `json:"nonEmpty"`
}

type PutMCPServerSecretResponse struct {
	Body *PutMCPServerSecretResponseBody
}

type DeleteMCPServerSecretRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	Kind MCPSecretKind `required:"true" query:"kind"`
	Slot string        `required:"true" query:"slot"`
}

type DeleteMCPServerSecretResponse struct{}
