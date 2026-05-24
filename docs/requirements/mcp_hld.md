# MCP Support HLD

- [1. Introduction](#1-introduction)
  - [1.1 MCP role mapping](#11-mcp-role-mapping)
  - [1.2 Protocol and extension targets](#12-protocol-and-extension-targets)
  - [1.3 Official MCP Go SDK usage](#13-official-mcp-go-sdk-usage)
  - [1.4 Core architectural principles](#14-core-architectural-principles)
  - [1.5 High-level architecture](#15-high-level-architecture)
- [2. Functional requirements](#2-functional-requirements)
  - [FR-1. MCP server management](#fr-1-mcp-server-management)
  - [FR-2. Latest MCP transports](#fr-2-latest-mcp-transports)
  - [FR-3. MCP lifecycle and sessions](#fr-3-mcp-lifecycle-and-sessions)
  - [FR-4. MCP HTTP authorization](#fr-4-mcp-http-authorization)
  - [FR-5. Server feature discovery](#fr-5-server-feature-discovery)
  - [FR-6. Chat-level MCP selection](#fr-6-chat-level-mcp-selection)
  - [FR-7. Conversation persistence](#fr-7-conversation-persistence)
  - [FR-8. Inference hydration](#fr-8-inference-hydration)
  - [FR-9. MCP tool execution](#fr-9-mcp-tool-execution)
  - [FR-10. Approval and policy](#fr-10-approval-and-policy)
  - [FR-11. MCP resources and prompts](#fr-11-mcp-resources-and-prompts)
  - [FR-12. MCP Apps extension](#fr-12-mcp-apps-extension)
  - [FR-13. Typed Wails API boundary](#fr-13-typed-wails-api-boundary)
  - [FR-14. Documentation and settings](#fr-14-documentation-and-settings)
- [3. Non-functional requirements](#3-non-functional-requirements)
  - [NFR-1. Security](#nfr-1-security)
  - [NFR-2. Privacy](#nfr-2-privacy)
  - [NFR-3. Local-first persistence](#nfr-3-local-first-persistence)
  - [NFR-4. Maintainability](#nfr-4-maintainability)
  - [NFR-5. Reliability](#nfr-5-reliability)
  - [NFR-6. Performance](#nfr-6-performance)
  - [NFR-7. Observability](#nfr-7-observability)
  - [NFR-8. Compatibility](#nfr-8-compatibility)
- [4. Out of scope](#4-out-of-scope)
- [5. Sequential implementation steps](#5-sequential-implementation-steps)
- [Step 1. Add MCP domain contracts and typed API skeleton](#step-1-add-mcp-domain-contracts-and-typed-api-skeleton)
  - [Requirements handled](#requirements-handled)
  - [Backend packages and files](#backend-packages-and-files)
  - [Frontend files](#frontend-files)
  - [Core backend models](#core-backend-models)
  - [Core frontend models](#core-frontend-models)
  - [Conversation MCP context model](#conversation-mcp-context-model)
  - [Wails API skeleton](#wails-api-skeleton)
  - [Workflow added](#workflow-added)
- [Step 2. Add MCP local store, secrets, and server management UI](#step-2-add-mcp-local-store-secrets-and-server-management-ui)
  - [Requirements handled](#requirements-handled-1)
  - [Backend packages and files](#backend-packages-and-files-1)
  - [Frontend files](#frontend-files-1)
  - [Data storage](#data-storage)
  - [Store responsibilities](#store-responsibilities)
  - [Validation rules](#validation-rules)
  - [UI workflow: add server](#ui-workflow-add-server)
  - [UI workflow: manage policy](#ui-workflow-manage-policy)
  - [Initial policy defaults](#initial-policy-defaults)
- [Step 3. Add MCP session manager and latest transports using official Go SDK](#step-3-add-mcp-session-manager-and-latest-transports-using-official-go-sdk)
  - [Requirements handled](#requirements-handled-2)
  - [Backend packages and files](#backend-packages-and-files-2)
  - [SDK integration](#sdk-integration)
  - [Runtime status model](#runtime-status-model)
  - [Client capabilities advertised initially](#client-capabilities-advertised-initially)
  - [Session manager responsibilities](#session-manager-responsibilities)
  - [Stdio transport rules](#stdio-transport-rules)
  - [Streamable HTTP transport rules](#streamable-http-transport-rules)
  - [Workflow: connect server](#workflow-connect-server)
  - [Workflow: disconnect server](#workflow-disconnect-server)
  - [Workflow: shutdown](#workflow-shutdown)
- [Step 4. Add MCP HTTP authorization and auth extensions](#step-4-add-mcp-http-authorization-and-auth-extensions)
  - [Requirements handled](#requirements-handled-3)
  - [Backend packages and files](#backend-packages-and-files-3)
  - [Frontend files](#frontend-files-2)
  - [Auth modes](#auth-modes)
  - [Auth metadata model](#auth-metadata-model)
  - [OAuth Authorization Code + PKCE workflow](#oauth-authorization-code--pkce-workflow)
  - [Protected resource discovery rules](#protected-resource-discovery-rules)
  - [Token usage rules](#token-usage-rules)
  - [Scope challenge workflow](#scope-challenge-workflow)
  - [OAuth Client Credentials extension workflow](#oauth-client-credentials-extension-workflow)
  - [Enterprise-managed authorization](#enterprise-managed-authorization)
- [Step 5. Add runtime discovery for tools, resources, prompts, and completions](#step-5-add-runtime-discovery-for-tools-resources-prompts-and-completions)
  - [Requirements handled](#requirements-handled-4)
  - [Backend packages and files](#backend-packages-and-files-4)
  - [Frontend files](#frontend-files-3)
  - [Runtime APIs added](#runtime-apis-added)
  - [Tool capability model](#tool-capability-model)
  - [Resource models](#resource-models)
  - [Prompt model](#prompt-model)
  - [Discovery workflow](#discovery-workflow)
  - [Notification workflow](#notification-workflow)
  - [Tool risk classification](#tool-risk-classification)
  - [Task support handling](#task-support-handling)
- [Step 6. Add composer MCP selection and conversation persistence](#step-6-add-composer-mcp-selection-and-conversation-persistence)
  - [Requirements handled](#requirements-handled-5)
  - [Frontend files](#frontend-files-4)
  - [Backend files](#backend-files)
  - [Composer state](#composer-state)
  - [Bottom bar behavior](#bottom-bar-behavior)
  - [Chips bar behavior](#chips-bar-behavior)
  - [Conversation message persistence](#conversation-message-persistence)
  - [Restore workflow](#restore-workflow)
  - [Edit older message workflow](#edit-older-message-workflow)
- [Step 7. Add inference bridge for MCP tools, resources, and prompts](#step-7-add-inference-bridge-for-mcp-tools-resources-and-prompts)
  - [Requirements handled](#requirements-handled-6)
  - [Backend packages and files](#backend-packages-and-files-5)
  - [Request model update](#request-model-update)
  - [Provider tool naming](#provider-tool-naming)
  - [Tool hydration workflow](#tool-hydration-workflow)
  - [Resource hydration workflow](#resource-hydration-workflow)
  - [Resource template hydration workflow](#resource-template-hydration-workflow)
  - [Prompt hydration workflow](#prompt-hydration-workflow)
  - [Server instructions](#server-instructions)
  - [Send-turn workflow](#send-turn-workflow)
- [Step 8. Add MCP tool execution, policy, and approvals](#step-8-add-mcp-tool-execution-policy-and-approvals)
  - [Requirements handled](#requirements-handled-7)
  - [Backend packages and files](#backend-packages-and-files-6)
  - [Frontend files](#frontend-files-5)
  - [Invocation request](#invocation-request)
  - [Invocation response](#invocation-response)
  - [Approval evaluation](#approval-evaluation)
  - [Approval workflow](#approval-workflow)
  - [Backend enforcement](#backend-enforcement)
  - [Tool result handling](#tool-result-handling)
  - [Auto-execute behavior](#auto-execute-behavior)
  - [Timeline rendering](#timeline-rendering)
- [Step 9. Add MCP Apps extension support](#step-9-add-mcp-apps-extension-support)
  - [Requirements handled](#requirements-handled-8)
  - [Backend packages and files](#backend-packages-and-files-7)
  - [Frontend files](#frontend-files-6)
  - [Apps capability negotiation](#apps-capability-negotiation)
  - [UI resource model](#ui-resource-model)
  - [App instance model](#app-instance-model)
  - [App discovery](#app-discovery)
  - [App rendering workflow](#app-rendering-workflow)
  - [Interactive app workflow](#interactive-app-workflow)
  - [Sandbox requirements](#sandbox-requirements)
  - [Host context](#host-context)
  - [App security decisions](#app-security-decisions)
- [Step 10. Add docs, tests, observability, and shutdown hardening](#step-10-add-docs-tests-observability-and-shutdown-hardening)
  - [Requirements handled](#requirements-handled-9)
  - [Documentation updates](#documentation-updates)
  - [Backend tests](#backend-tests)
  - [Frontend tests](#frontend-tests)
  - [Observability](#observability)
  - [Shutdown hardening](#shutdown-hardening)

## 1. Introduction

FlexiGPT will support MCP as a local-first desktop MCP host. Users will configure MCP servers, use those servers inside chats, expose selected MCP tools to the active model, attach MCP resources and prompts as context, and render MCP Apps when a server provides interactive UI resources.

FlexiGPT remains responsible for user control, durable local configuration, provider request assembly, approval decisions, app sandboxing, and privacy boundaries. External MCP servers remain isolated protocol peers.

### 1.1 MCP role mapping

| MCP role      | FlexiGPT component                                |
| ------------- | ------------------------------------------------- |
| Host          | FlexiGPT desktop app                              |
| Client        | Go backend MCP client session managed by FlexiGPT |
| Server        | Configured external MCP server                    |
| MCP Apps Host | FlexiGPT frontend plus backend mediator           |
| MCP Apps View | Sandboxed iframe rendered by the frontend         |

The backend creates and owns one isolated MCP client session per configured MCP server. The frontend never connects directly to external MCP servers.

MCP Apps iframes use MCP-style JSON-RPC over `postMessage`, but they communicate with FlexiGPT as host, not directly with external MCP servers.

### 1.2 Protocol and extension targets

FlexiGPT targets the latest attached MCP protocol revision:

- MCP protocol: `2025-11-25`
- MCP Apps extension: `io.modelcontextprotocol/ui`, stable `2026-01-26`
- MCP HTTP authorization baseline from the MCP authorization specification
- OAuth Client Credentials auth extension where applicable and supported by the official MCP Go SDK

FlexiGPT will not support deprecated or legacy MCP transports or compatibility paths.

### 1.3 Official MCP Go SDK usage

The backend should use the official MCP Go SDK for:

- MCP protocol message types where suitable
- lifecycle and capability negotiation
- Streamable HTTP transport
- stdio transport
- request/response handling
- server notifications
- authorization helpers where available
- MCP Apps extension helper types where available

FlexiGPT should still define app-facing types in `internal/mcp/spec` and `frontend/app/spec/mcp.ts`. SDK types should not leak directly across the Wails/frontend boundary. This keeps the app contract stable if SDK internals or raw protocol shapes evolve.

### 1.4 Core architectural principles

- MCP is modeled around `MCP Server`, not around static tool definitions.
- MCP tools, resources, prompts, and Apps are dynamic runtime-discovered server features.
- MCP server configuration and policy are local durable state.
- MCP sessions, discovery snapshots, requests, notifications, and app instances are runtime state.
- The backend owns protocol sessions and execution.
- The frontend owns user interaction, selection, approvals, and rendering.
- The provider/inference layer hydrates MCP selections into provider requests.
- The frontend never directly executes MCP protocol requests against external servers.
- Approval and policy are enforced in the backend, even when the frontend presents the approval UI.
- MCP Apps are treated as untrusted UI and must be sandboxed.

### 1.5 High-level architecture

```text
Frontend
  MCP Servers page
  Chat composer MCP picker
  Timeline MCP tool/app rendering
  Approval UI
  MCP Apps iframe host bridge
      ↓ typed Wails APIs

Backend
  MCP store
  MCP auth manager
  MCP session manager
  MCP runtime
  MCP policy and approval services
  MCP inference/tool/app bridges
      ↓ official MCP Go SDK

MCP client sessions
  session per configured MCP server
      ↓

MCP servers
  Streamable HTTP servers
  stdio servers
      ↓

Inference wrapper
  static tools + MCP tools + skills + web search
      ↓

LLM providers
```

## 2. Functional requirements

### FR-1. MCP server management

Users must be able to:

- add MCP servers
- edit MCP server configuration
- enable or disable servers
- delete servers
- view connection status
- connect, disconnect, and refresh discovery
- view discovered tools, resources, resource templates, prompts, and app-capable tools
- configure per-server policy defaults
- configure per-tool policy overrides
- configure HTTP auth status where required

### FR-2. Latest MCP transports

FlexiGPT must support latest proper MCP transports through the official Go SDK:

- Streamable HTTP
- stdio

FlexiGPT must not support deprecated legacy HTTP+SSE compatibility behavior.

### FR-3. MCP lifecycle and sessions

For each connected server, the backend must:

- create one isolated MCP client session
- send `initialize`
- negotiate protocol version
- advertise supported client capabilities
- receive server capabilities
- send `notifications/initialized`
- track server info and instructions
- track negotiated protocol version
- support request timeouts
- support `ping`
- handle request cancellation with `notifications/cancelled`
- handle progress notifications
- handle logging notifications
- disconnect cleanly on shutdown

For HTTP sessions, the backend must also:

- send the negotiated `MCP-Protocol-Version` header on subsequent requests
- handle `MCP-Session-Id` if provided by the server
- include auth tokens when required

### FR-4. MCP HTTP authorization

For Streamable HTTP servers, FlexiGPT must support MCP HTTP authorization:

- parse `WWW-Authenticate` challenges
- discover OAuth Protected Resource Metadata
- discover Authorization Server Metadata or OpenID Connect discovery metadata
- use Authorization Code with PKCE for user-based auth
- include the OAuth `resource` parameter in authorization and token requests
- send bearer tokens in `Authorization` headers only
- store tokens securely using OS keyring-backed storage
- support token refresh where possible
- support runtime scope challenge handling with retry limits
- show auth status in the MCP Servers page

FlexiGPT should also support the OAuth Client Credentials extension for configured machine-to-machine MCP servers, where the server and authorization server support it and credentials are explicitly configured by the user.

### FR-5. Server feature discovery

For connected servers, FlexiGPT must discover and cache:

- tools via `tools/list`
- resources via `resources/list`
- resource templates via `resources/templates/list`
- prompts via `prompts/list`
- argument completions via `completion/complete`, when supported
- server capabilities
- server info
- server instructions

FlexiGPT must handle list-changed notifications by invalidating or refreshing the relevant discovery snapshot.

### FR-6. Chat-level MCP selection

In a chat, users must be able to:

- select MCP servers for the active conversation
- choose whether a selected server exposes no tools, all tools, or selected tools
- choose specific tools from a server
- configure execution behavior for selected tools
- attach MCP resources to the request
- attach MCP resource-template instances to the request
- select MCP prompts and fill prompt arguments
- see server status and stale references in the composer
- see selected MCP servers/tools/resources/prompts in chips
- restore MCP selections from saved conversations
- restore MCP selections when editing and resending an older user message

### FR-7. Conversation persistence

Conversation messages must persist MCP context as part of durable conversation data.

Persisted MCP context must include:

- selected servers
- tool exposure mode per server
- selected tool refs
- resource refs
- resource template refs and arguments
- prompt refs and arguments
- tool call provenance
- tool output provenance
- app UI provenance where applicable

Stored refs must be stale-safe. If a server, tool, prompt, or resource is missing later, the frontend should restore what it can and show warnings.

### FR-8. Inference hydration

When a chat turn is sent, the backend inference layer must hydrate MCP context into the provider request.

Hydration must include:

- selected MCP tools as provider tool definitions
- deterministic provider-safe tool names
- internal mapping from provider tool names back to MCP server/tool refs
- selected MCP resources as provider-ready content blocks
- selected resource-template instances as provider-ready content blocks
- selected MCP prompts as prompt messages or visible draft content
- explicitly enabled server instructions, if the user allows them

MCP prompts must not be blindly appended to the system prompt. MCP prompts return role-tagged prompt messages and content blocks. The composer must make the usage mode explicit.

### FR-9. MCP tool execution

When a model returns an MCP tool call, FlexiGPT must:

- resolve the provider tool name back to an MCP server and tool
- validate that the tool was available in the current request
- evaluate backend policy
- request user approval if required
- execute the MCP tool through the backend runtime
- return normalized tool output to the composer
- render the call and output in the timeline
- support existing FlexiGPT manual and auto-execute tool-loop behavior, bounded by MCP policy

MCP task-augmented tool calls are not supported in this HLD.

### FR-10. Approval and policy

FlexiGPT must support backend-enforced MCP policy:

- server-level default policy
- per-tool policy override
- allow once
- allow always
- deny once
- deny always
- stale digest handling
- risk classification
- approval audit trail

Approval policy and execution mode must be separate concepts:

- `approvalRule`: whether backend policy allows, denies, or asks
- `executionMode`: whether the composer runs the tool manually or automatically after policy permits it

### FR-11. MCP resources and prompts

FlexiGPT must support MCP resources and prompts as first-class server-scoped concepts.

Resources:

- list resources
- read selected resources
- normalize text and binary resource contents into provider-ready content
- preserve metadata and MIME type
- show resource chips in composer

Resource templates:

- list templates
- collect template arguments
- use completion suggestions where supported
- resolve selected template instances into resource content

Prompts:

- list prompts
- collect prompt arguments
- use completion suggestions where supported
- call `prompts/get`
- insert or attach returned prompt messages in an explicit user-visible way

### FR-12. MCP Apps extension

FlexiGPT must support MCP Apps extension `io.modelcontextprotocol/ui` when app sandboxing is available.

Support includes:

- advertise Apps capability in MCP initialize request
- discover app-capable tools through `_meta.ui.resourceUri`
- recognize `ui://` resources
- read UI resources through `resources/read`
- support MIME type `text/html;profile=mcp-app`
- render Apps in sandboxed iframes
- enforce CSP declared by UI resource metadata
- provide host context through `ui/initialize`
- send tool input and tool result notifications to Apps views
- support app-initiated allowed JSON-RPC requests
- support app-only tools through `_meta.ui.visibility`
- block cross-server app tool calls
- teardown app instances cleanly

### FR-13. Typed Wails API boundary

All frontend-to-backend MCP behavior must go through typed Wails APIs.

The frontend must not import backend implementation details or MCP SDK raw types directly.

### FR-14. Documentation and settings

FlexiGPT docs and settings surfaces must explain:

- what MCP servers are
- that MCP servers can execute local commands or contact remote services
- how MCP tools differ from static FlexiGPT tools
- how MCP resources/prompts can enter provider requests
- how MCP Apps are sandboxed
- how secrets and OAuth tokens are stored
- how approvals work
- privacy implications of MCP tool outputs and resources

## 3. Non-functional requirements

### NFR-1. Security

- Treat all MCP server-provided content as untrusted.
- Treat MCP tool descriptions and annotations as untrusted hints.
- Treat MCP prompts, resources, and tool outputs as untrusted.
- Default unknown or risky tools to approval required.
- Enforce approval in backend.
- Redact secrets and tokens from logs.
- Store secrets and OAuth tokens through OS keyring-backed storage.
- Do not expose Wails bindings to MCP Apps iframes.
- Do not render MCP App HTML directly into the React DOM.
- Enforce iframe sandbox and CSP for Apps.
- Show clear UI boundaries for untrusted Apps and tool outputs.

### NFR-2. Privacy

- Users must see which MCP servers are active in a chat.
- Users must see which MCP tools are exposed to the model.
- Users must explicitly select MCP resources and prompts used as context.
- MCP tool outputs may be resent to model providers only through visible conversation flow.
- Remote MCP auth tokens must stay local and never be sent to model providers.
- MCP server data must not be silently added to model context.

### NFR-3. Local-first persistence

- MCP server configs, policies, metadata, and conversation MCP refs are stored locally.
- Secrets and tokens are not stored in normal JSON config files.
- Runtime sessions and app instances are not durable state.
- Discovery snapshots may be cached but must be stale-safe.

### NFR-4. Maintainability

- Store, runtime, policy, approval, auth, Apps, and inference responsibilities must remain separated.
- SDK types should be wrapped by app-level contracts at the Wails boundary.
- MCP should not be merged into static tools, prompts, or skills catalogs.
- MCP Apps should be a dedicated frontend subsystem, not embedded in generic message rendering logic.

### NFR-5. Reliability

- Runtime operations must use timeouts.
- Stdio processes must be terminated on disconnect or app shutdown.
- HTTP sessions must handle session invalidation and reconnect.
- Discovery should be refreshable manually.
- List-change notifications should invalidate relevant caches.
- Request cancellation should be best effort and safe.

### NFR-6. Performance

- Discovery snapshots should be cached per server.
- Tool/resource/prompt lists should support pagination.
- UI resources may be prefetched and cached by digest where safe.
- Provider tool hydration should deduplicate and enforce provider limits.
- Apps iframe sizing should be debounced.

### NFR-7. Observability

- MCP server status must be visible.
- MCP connection and discovery errors must be explainable.
- MCP logs should be routed to a server-specific diagnostics surface.
- Approval decisions should be auditable locally.
- Tool calls should preserve provenance in message details.

### NFR-8. Compatibility

- Support latest attached MCP protocol and extensions only.
- Do not implement deprecated legacy transports.
- Do not advertise unsupported MCP client capabilities.
- Gracefully ignore unknown extension metadata.
- Keep app-level models versioned.

## 4. Out of scope

The following are not implemented in this HLD:

- Deprecated legacy HTTP+SSE transport.
- MCP task augmentation and task APIs.
  - Reason: the official Go SDK does not currently support tasks for this use case.
  - Tool metadata `execution.taskSupport` may be displayed as unsupported, but task execution is not implemented.
- MCP sampling.
  - FlexiGPT should not advertise `sampling` until a strong user-review and model-selection UX exists.
- MCP roots.
  - FlexiGPT should not advertise `roots` until per-server root grant UI and path-boundary enforcement are implemented.
- MCP elicitation.
  - FlexiGPT should not advertise `elicitation` until form and URL approval UX is implemented.
  - MCP HTTP authorization does not require advertising elicitation.
- Enterprise-managed authorization extension.
  - This requires enterprise SSO, IdP token exchange, and admin policy integration.
  - The auth package should leave extension points for this later, but not implement it now.
- Running FlexiGPT as an MCP server.
- Importing MCP tools into the static FlexiGPT tools catalog.
- Built-in MCP server marketplace, package signing, or auto-update workflows.
- Custom non-standard transports.
- Cross-server app tool calls.
- Unrestricted App network access.

## 5. Sequential implementation steps

## Step 1. Add MCP domain contracts and typed API skeleton

### Requirements handled

- FR-1 server management foundation
- FR-6 chat-level selection foundation
- FR-7 conversation persistence foundation
- FR-12 Apps data model foundation
- FR-13 typed Wails API boundary
- NFR-4 maintainability
- NFR-8 compatibility

### Backend packages and files

Add:

```text
internal/mcp/spec/
  type_const.go
  server.go
  transport.go
  auth.go
  policy.go
  discovery.go
  conversation.go
  tool.go
  resource.go
  prompt.go
  approval.go
  apps.go
  req_resp.go

cmd/agentgo/wrapper_mcp.go
```

Update:

```text
cmd/agentgo/app.go
cmd/agentgo/main.go
```

### Frontend files

Add:

```text
frontend/app/spec/mcp.ts
frontend/app/apis/wailsapi/mcp_api.ts
```

Update:

```text
frontend/app/apis/interface.ts
```

### Core backend models

```go
type MCPTransportType string

const (
  MCPTransportStreamableHTTP MCPTransportType = "streamableHttp"
  MCPTransportStdio          MCPTransportType = "stdio"
)

type MCPServerAvailability string

const (
  MCPServerAvailabilityManual     MCPServerAvailability = "manual"
  MCPServerAvailabilityAutoAttach MCPServerAvailability = "autoAttach"
)

type MCPTrustLevel string

const (
  MCPTrustLevelUntrusted MCPTrustLevel = "untrusted"
  MCPTrustLevelTrusted   MCPTrustLevel = "trusted"
)
```

```go
type MCPServerConfig struct {
  SchemaVersion string                `json:"schemaVersion"`
  ID            string                `json:"id"`
  DisplayName   string                `json:"displayName"`
  Enabled       bool                  `json:"enabled"`
  Transport     MCPTransportType      `json:"transport"`

  Stdio          *MCPStdioConfig      `json:"stdio,omitempty"`
  StreamableHTTP *MCPStreamableHTTPConfig `json:"streamableHttp,omitempty"`

  Availability  MCPServerAvailability `json:"availability"`
  TrustLevel    MCPTrustLevel         `json:"trustLevel"`

  DefaultPolicy  MCPServerPolicy       `json:"defaultPolicy"`
  AppsPolicy     *MCPAppsPolicy        `json:"appsPolicy,omitempty"`
  AuthRef        *MCPAuthRef           `json:"authRef,omitempty"`

  CreatedAt     time.Time             `json:"createdAt"`
  ModifiedAt    time.Time             `json:"modifiedAt"`
  SoftDeletedAt *time.Time            `json:"softDeletedAt,omitempty"`
}
```

```go
type MCPStdioConfig struct {
  Command          string            `json:"command"`
  Args             []string          `json:"args,omitempty"`
  WorkingDir       string            `json:"workingDir,omitempty"`
  Env              map[string]string `json:"env,omitempty"`
  SecretEnvRefs    map[string]string `json:"secretEnvRefs,omitempty"`
  StartupTimeoutMS int               `json:"startupTimeoutMS,omitempty"`
}
```

```go
type MCPHTTPAuthMode string

const (
  MCPHTTPAuthNone              MCPHTTPAuthMode = "none"
  MCPHTTPAuthOAuth             MCPHTTPAuthMode = "oauth"
  MCPHTTPAuthClientCredentials MCPHTTPAuthMode = "clientCredentials"
  MCPHTTPAuthCustomBearer      MCPHTTPAuthMode = "customBearer"
  MCPHTTPAuthCustomHeaders     MCPHTTPAuthMode = "customHeaders"
)

type MCPStreamableHTTPConfig struct {
  URL              string            `json:"url"`
  TimeoutMS        int               `json:"timeoutMS,omitempty"`
  CustomHeaders    map[string]string `json:"customHeaders,omitempty"`
  SecretHeaderRefs map[string]string `json:"secretHeaderRefs,omitempty"`
  AuthMode         MCPHTTPAuthMode   `json:"authMode"`
}
```

### Core frontend models

```ts
export type MCPTransportType = "streamableHttp" | "stdio";
export type MCPServerAvailability = "manual" | "autoAttach";
export type MCPTrustLevel = "untrusted" | "trusted";

export interface MCPServerConfig {
  schemaVersion: string;
  id: string;
  displayName: string;
  enabled: boolean;
  transport: MCPTransportType;

  stdio?: MCPStdioConfig;
  streamableHttp?: MCPStreamableHTTPConfig;

  availability: MCPServerAvailability;
  trustLevel: MCPTrustLevel;

  defaultPolicy: MCPServerPolicy;
  appsPolicy?: MCPAppsPolicy;
  authRef?: MCPAuthRef;

  createdAt: string;
  modifiedAt: string;
  softDeletedAt?: string;
}
```

### Conversation MCP context model

Use a nested context object instead of many top-level message fields:

```ts
export interface MCPConversationContext {
  servers: MCPServerSelection[];
  resources?: MCPResourceRef[];
  resourceTemplates?: MCPResourceTemplateRef[];
  prompts?: MCPPromptRef[];
}
```

```ts
export interface MCPServerSelection {
  serverID: string;
  snapshotDigest?: string;

  toolExposure: "none" | "all" | "selected";
  selectedTools?: MCPToolSelection[];

  includeServerInstructions?: boolean;
}
```

```ts
export interface MCPToolSelection {
  serverID: string;
  toolName: string;
  providerToolName?: string;
  choiceID?: string;
  digest?: string;

  approvalRule?: MCPApprovalRule;
  executionMode?: MCPExecutionMode;
}
```

### Wails API skeleton

Add typed API groups:

```ts
export interface IMCPStoreAPI {
  listMcpServers(req: ListMCPServersRequest): Promise<ListMCPServersResponse>;
  getMcpServer(serverID: string): Promise<MCPServerConfig | undefined>;
  putMcpServer(serverID: string, payload: PutMCPServerPayload): Promise<void>;
  patchMcpServerEnabled(serverID: string, enabled: boolean): Promise<void>;
  patchMcpServerPolicy(
    serverID: string,
    payload: PatchMCPServerPolicyPayload,
  ): Promise<void>;
  deleteMcpServer(serverID: string): Promise<void>;
}

export interface IMCPRuntimeAPI {
  connectMcpServer(serverID: string): Promise<void>;
  disconnectMcpServer(serverID: string): Promise<void>;
  refreshMcpServer(serverID: string): Promise<MCPServerRuntimeSnapshot>;
  getMcpServerStatus(serverID: string): Promise<MCPServerRuntimeSnapshot>;
}
```

More runtime APIs are added in later steps.

### Workflow added

No user-facing workflow is complete in this step. This step establishes stable contracts and empty wrappers so later work has a clean boundary.

## Step 2. Add MCP local store, secrets, and server management UI

### Requirements handled

- FR-1 MCP server management
- FR-4 auth metadata foundation
- FR-10 policy persistence foundation
- FR-14 docs/settings foundation
- NFR-2 privacy
- NFR-3 local-first persistence
- NFR-4 maintainability

### Backend packages and files

Add:

```text
internal/mcp/store/
  store.go
  validate.go
  clone.go
  pagination.go
  secret_refs.go
  policy.go
```

Update:

```text
cmd/agentgo/app.go
cmd/agentgo/wrapper_mcp.go
internal/setting/store/store.go
```

### Frontend files

Add:

```text
frontend/app/mcpservers/page.tsx
frontend/app/mcpservers/server_list.tsx
frontend/app/mcpservers/server_editor_modal.tsx
frontend/app/mcpservers/server_status_card.tsx
frontend/app/mcpservers/server_policy_editor.tsx
frontend/app/mcpservers/server_auth_panel.tsx
frontend/app/mcpservers/server_discovery_panel.tsx
```

Update:

```text
frontend/app/routes.ts
frontend/app/components/sidebar.tsx
frontend/app/apis/interface.ts
```

### Data storage

Add MCP data root under app data directory:

```text
mcpserversv1/
  mcpservers.json
  mcp-policies.json
  mcp-auth-metadata.json
  mcp-last-known-snapshots.json
```

Secrets and OAuth tokens must not be stored in these JSON files. They must be referenced through secret IDs and stored using existing OS keyring-backed secret storage.

### Store responsibilities

`internal/mcp/store` owns:

- server config CRUD
- enable/disable
- soft delete
- validation
- policy persistence
- secret refs
- last-known metadata persistence
- pagination

It does not own:

- live MCP sessions
- process lifecycle
- HTTP connections
- tool execution
- Apps iframe state

### Validation rules

Server config validation must reject:

- missing server ID
- duplicate server ID
- empty display name
- disabled but malformed persisted config that cannot be edited
- missing transport config for selected transport
- stdio command using shell wrappers like `sh -c`
- stdio env secret refs pointing to missing secret metadata
- HTTP URL without scheme
- HTTP URL with unsupported scheme
- custom auth fields when auth mode does not allow them

### UI workflow: add server

```text
User opens MCP Servers page
  -> clicks Add Server
  -> chooses Streamable HTTP or stdio
  -> fills config
  -> config is validated locally for basic form errors
  -> frontend calls putMcpServer
  -> backend validates and stores config
  -> UI refreshes list
```

### UI workflow: manage policy

```text
User opens a server card
  -> opens policy editor
  -> sets default approval and execution behavior
  -> optionally configures app support
  -> frontend calls patchMcpServerPolicy
  -> backend validates and persists policy
```

### Initial policy defaults

```ts
export type MCPApprovalRule = "ask" | "allow" | "deny";
export type MCPExecutionMode = "manual" | "auto";

export interface MCPServerPolicy {
  defaultApprovalRule: MCPApprovalRule;
  defaultExecutionMode: MCPExecutionMode;
  requireApprovalForUnknownRisk: boolean;
  requireApprovalForWrite: boolean;
  requireApprovalForDestructive: boolean;
}
```

Recommended defaults:

```json
{
  "defaultApprovalRule": "ask",
  "defaultExecutionMode": "manual",
  "requireApprovalForUnknownRisk": true,
  "requireApprovalForWrite": true,
  "requireApprovalForDestructive": true
}
```

## Step 3. Add MCP session manager and latest transports using official Go SDK

### Requirements handled

- FR-2 latest MCP transports
- FR-3 lifecycle and sessions
- FR-5 server status foundation
- NFR-1 security
- NFR-5 reliability
- NFR-7 observability
- NFR-8 compatibility

### Backend packages and files

Add:

```text
internal/mcp/transport/
  stdio.go
  streamable_http.go
  env.go
  headers.go

internal/mcp/session/
  manager.go
  session.go
  lifecycle.go
  capabilities.go
  notifications.go
  status.go
  timeout.go
```

Update:

```text
internal/mcp/runtime/
  runtime.go

cmd/agentgo/app.go
cmd/agentgo/wrapper_mcp.go
```

### SDK integration

Use the official MCP Go SDK to:

- create Streamable HTTP clients
- create stdio clients
- perform initialize lifecycle
- send and receive protocol messages
- handle transport-level session details where SDK supports them
- parse server capabilities and server info
- register notification handlers

FlexiGPT wrappers should convert SDK structs into app-level `mcp/spec` structs.

### Runtime status model

```ts
export type MCPServerStatus =
  | "disabled"
  | "disconnected"
  | "connecting"
  | "ready"
  | "error";

export interface MCPServerRuntimeSnapshot {
  serverID: string;
  status: MCPServerStatus;

  negotiatedProtocolVersion?: string;
  serverInfo?: MCPImplementationInfo;
  serverCapabilities?: MCPServerCapabilitiesSummary;
  instructions?: string;

  lastError?: string;
  lastConnectedAt?: string;
  lastSyncedAt?: string;

  toolCount: number;
  resourceCount: number;
  resourceTemplateCount: number;
  promptCount: number;

  snapshotDigest?: string;
}
```

### Client capabilities advertised initially

Advertise only capabilities FlexiGPT supports in this HLD.

Do not advertise:

- `tasks`
- `sampling`
- `roots`
- `elicitation`

Advertise Apps extension only when Apps support is implemented in Step 9.

Initial capability intent:

```json
{
  "capabilities": {
    "experimental": {}
  }
}
```

When Apps support is added:

```json
{
  "capabilities": {
    "extensions": {
      "io.modelcontextprotocol/ui": {
        "mimeTypes": ["text/html;profile=mcp-app"]
      }
    }
  }
}
```

If the Go SDK uses a typed extensions mechanism, use the SDK equivalent.

### Session manager responsibilities

The session manager owns:

- session map keyed by server ID
- connection state
- SDK client instance
- protocol version
- capabilities
- server info
- server instructions
- notification registration
- request timeouts
- reconnect/disconnect
- process cleanup for stdio

### Stdio transport rules

- Execute command directly.
- Never execute through shell.
- Do not inherit full environment by default.
- Pass only configured env vars.
- Resolve secret env refs immediately before launch.
- Redact secrets in logs.
- Capture stderr as logs.
- Do not treat every stderr line as failure.
- Kill process on disconnect or app shutdown.

### Streamable HTTP transport rules

- Use latest Streamable HTTP transport only.
- Do not attempt legacy HTTP+SSE fallback.
- Include `Accept` headers required by the SDK/spec.
- Include negotiated `MCP-Protocol-Version` on subsequent requests.
- Preserve and send `MCP-Session-Id` when provided by server.
- Use auth manager for bearer tokens.
- Respect request timeouts.

### Workflow: connect server

```text
Frontend calls connectMcpServer(serverID)
  -> backend loads server config
  -> backend validates enabled state and transport config
  -> auth manager prepares auth headers if needed
  -> transport creates SDK transport
  -> session manager creates SDK client
  -> client sends initialize
  -> server returns capabilities/info/instructions
  -> client sends initialized notification
  -> session manager stores runtime snapshot
  -> backend emits status event to frontend
```

### Workflow: disconnect server

```text
Frontend calls disconnectMcpServer(serverID)
  -> session manager cancels in-flight requests
  -> session manager closes SDK client/transport
  -> stdio process is terminated if present
  -> runtime status becomes disconnected
  -> backend emits status event
```

### Workflow: shutdown

```text
App shutdown starts
  -> MCP runtime rejects new requests
  -> in-flight MCP requests are cancelled best-effort
  -> app instances are torn down
  -> HTTP sessions are closed
  -> stdio processes are terminated
  -> MCP store is closed
```

Also update existing shutdown to close stores that expose `Close()` but are currently not closed.

## Step 4. Add MCP HTTP authorization and auth extensions

### Requirements handled

- FR-4 MCP HTTP authorization
- FR-1 auth status in server UI
- NFR-1 security
- NFR-2 privacy
- NFR-3 local-first persistence
- NFR-5 reliability

### Backend packages and files

Add:

```text
internal/mcp/auth/
  manager.go
  metadata.go
  oauth_pkce.go
  callback.go
  token_store.go
  scope_challenge.go
  client_credentials.go
  errors.go
```

Update:

```text
internal/mcp/transport/streamable_http.go
internal/mcp/store/auth.go
cmd/agentgo/wrapper_mcp.go
```

### Frontend files

Update:

```text
frontend/app/mcpservers/server_auth_panel.tsx
frontend/app/apis/interface.ts
```

Add if needed:

```text
frontend/app/mcpservers/oauth_callback_state.ts
```

### Auth modes

```ts
export type MCPHTTPAuthMode =
  | "none"
  | "oauth"
  | "clientCredentials"
  | "customBearer"
  | "customHeaders";
```

Recommended meaning:

- `none`: no auth
- `oauth`: MCP HTTP authorization with Authorization Code + PKCE
- `clientCredentials`: OAuth Client Credentials extension
- `customBearer`: user-provided bearer token compatibility mode
- `customHeaders`: user-provided secret headers compatibility mode

`customBearer` and `customHeaders` are not MCP auth extensions. They are compatibility modes for non-standard deployments and should be labeled as such.

### Auth metadata model

```ts
export interface MCPAuthRef {
  authMode: MCPHTTPAuthMode;
  tokenRef?: string;
  clientCredentialRef?: string;
  metadataRef?: string;
}

export interface MCPAuthStatus {
  serverID: string;
  authMode: MCPHTTPAuthMode;
  state:
    | "notRequired"
    | "required"
    | "authorized"
    | "expired"
    | "insufficientScope"
    | "error";
  scopes?: string[];
  expiresAt?: string;
  lastError?: string;
  authorizationServer?: string;
  resource?: string;
}
```

### OAuth Authorization Code + PKCE workflow

```text
Runtime attempts HTTP request without token
  -> MCP server returns 401 with WWW-Authenticate
  -> auth manager parses resource_metadata if present
  -> auth manager fetches Protected Resource Metadata
  -> auth manager discovers Authorization Server Metadata
  -> auth manager validates PKCE support
  -> frontend opens browser authorization URL
  -> user approves
  -> local callback receives authorization code
  -> auth manager exchanges code using PKCE and resource parameter
  -> token is stored in OS keyring
  -> runtime retries MCP request with bearer token
```

### Protected resource discovery rules

Auth manager must support:

- `WWW-Authenticate` `resource_metadata` URL
- well-known protected resource metadata fallback
- `authorization_servers` selection
- Authorization Server Metadata discovery
- OpenID Connect discovery fallback where required by the MCP auth spec

### Token usage rules

- Use `Authorization: Bearer <access-token>`.
- Include auth on every HTTP request requiring it.
- Never put tokens in query strings.
- Never send MCP server tokens to LLM providers.
- Never log tokens.
- Store refresh tokens in OS keyring if issued.

### Scope challenge workflow

```text
Runtime request returns 403 insufficient_scope
  -> auth manager parses required scopes
  -> status becomes insufficientScope
  -> frontend prompts user to reauthorize
  -> auth manager starts step-up OAuth flow
  -> new token is stored
  -> original request is retried with retry limit
```

### OAuth Client Credentials extension workflow

```text
User configures client credentials for a server
  -> credentials are stored in OS keyring
  -> auth manager discovers protected resource and AS metadata
  -> auth manager requests token with grant_type=client_credentials
  -> request includes resource parameter
  -> token is stored in OS keyring
  -> runtime uses bearer token for MCP HTTP requests
```

Support both SDK-supported methods where available:

- client secret authentication
- private key JWT authentication

Private key material must be stored securely and never exported in settings JSON.

### Enterprise-managed authorization

Do not implement Enterprise-Managed Authorization in this HLD. Keep `internal/mcp/auth` structured so an enterprise edition could later add:

- enterprise IdP config
- SSO subject token handling
- token exchange
- ID-JAG handling
- JWT authorization grant exchange

## Step 5. Add runtime discovery for tools, resources, prompts, and completions

### Requirements handled

- FR-5 server feature discovery
- FR-11 resources and prompts foundation
- FR-12 Apps discovery foundation
- NFR-6 performance
- NFR-7 observability

### Backend packages and files

Add:

```text
internal/mcp/runtime/
  discovery.go
  tools.go
  resources.go
  prompts.go
  completions.go
  digest.go
```

Update:

```text
cmd/agentgo/wrapper_mcp.go
```

### Frontend files

Update:

```text
frontend/app/mcpservers/server_discovery_panel.tsx
frontend/app/apis/interface.ts
```

### Runtime APIs added

```ts
export interface IMCPRuntimeAPI {
  listMcpServerTools(serverID: string): Promise<{ tools: MCPToolCapability[] }>;

  listMcpServerResources(
    serverID: string,
    cursor?: string,
  ): Promise<{ resources: MCPResourceRef[]; nextCursor?: string }>;

  listMcpServerResourceTemplates(
    serverID: string,
    cursor?: string,
  ): Promise<{
    resourceTemplates: MCPResourceTemplateRef[];
    nextCursor?: string;
  }>;

  listMcpServerPrompts(
    serverID: string,
    cursor?: string,
  ): Promise<{ prompts: MCPPromptRef[]; nextCursor?: string }>;

  completeMcpArgument(
    req: MCPCompleteArgumentRequest,
  ): Promise<MCPCompletionResult>;

  readMcpResource(
    req: MCPReadResourceRequest,
  ): Promise<MCPReadResourceResponse>;

  getMcpPrompt(req: MCPGetPromptRequest): Promise<MCPGetPromptResponse>;
}
```

### Tool capability model

```ts
export interface MCPToolCapability {
  serverID: string;
  toolName: string;
  providerToolName: string;
  choiceID: string;

  title?: string;
  displayName: string;
  description?: string;

  inputSchema: Record<string, unknown>;
  outputSchema?: Record<string, unknown>;

  annotations?: MCPToolAnnotations;
  inferredRisk: MCPToolRisk;

  approvalRule: MCPApprovalRule;
  executionMode: MCPExecutionMode;

  taskSupport: "forbidden" | "optional" | "required";

  app?: {
    resourceUri?: string;
    visibility: Array<"model" | "app">;
  };

  digest: string;
  enabled: boolean;
  stale?: boolean;
}
```

### Resource models

```ts
export interface MCPResourceRef {
  serverID: string;
  uri: string;
  name?: string;
  title?: string;
  displayName: string;
  description?: string;
  mimeType?: string;
  size?: number;
  annotations?: MCPAnnotations;
  digest?: string;
}
```

```ts
export interface MCPResourceTemplateRef {
  serverID: string;
  uriTemplate: string;
  name?: string;
  title?: string;
  displayName: string;
  description?: string;
  mimeType?: string;
  arguments?: Record<string, string>;
  annotations?: MCPAnnotations;
  digest?: string;
}
```

### Prompt model

```ts
export interface MCPPromptRef {
  serverID: string;
  promptName: string;
  title?: string;
  displayName: string;
  description?: string;
  arguments?: Record<string, string>;
  digest?: string;
}
```

### Discovery workflow

```text
User clicks Refresh on server
  -> frontend calls refreshMcpServer
  -> runtime ensures session is connected
  -> runtime calls tools/list if server supports tools
  -> runtime calls resources/list if server supports resources
  -> runtime calls resources/templates/list if server supports resources
  -> runtime calls prompts/list if server supports prompts
  -> runtime computes discovery digest
  -> runtime updates snapshot
  -> frontend renders discovered capabilities
```

### Notification workflow

```text
Server sends notifications/tools/list_changed
  -> session manager receives notification
  -> runtime invalidates tools cache
  -> backend emits MCP status/discovery event
  -> frontend shows stale discovery or auto-refreshes

Server sends notifications/resources/list_changed
  -> runtime invalidates resource and template cache

Server sends notifications/prompts/list_changed
  -> runtime invalidates prompt cache
```

### Tool risk classification

Risk is derived from:

- trusted server policy
- tool annotations
- tool name and description heuristics
- user override

Tool annotations are hints only. For untrusted servers, annotations must not auto-grant trust.

Default mapping:

- `readOnlyHint: true` -> `read`, if server trusted
- `destructiveHint: true` -> `destructive`
- `openWorldHint: true` -> `openWorld`
- no reliable signal -> `unknown`

### Task support handling

If a tool declares `execution.taskSupport`:

- show the value in UI
- if `required`, mark tool unsupported for execution in this HLD
- if `optional`, execute only in normal non-task mode
- if `forbidden` or absent, execute normally

Do not call `tools/call` with `params.task`.

## Step 6. Add composer MCP selection and conversation persistence

### Requirements handled

- FR-6 chat-level MCP selection
- FR-7 conversation persistence
- FR-11 resource/prompt selection UX
- NFR-2 privacy
- NFR-3 local-first persistence

### Frontend files

Add:

```text
frontend/app/chats/composer/mcp/
  use_composer_mcp.ts
  use_mcp_server_options.ts
  use_mcp_tool_selection.ts
  use_mcp_resource_picker.ts
  use_mcp_prompt_picker.ts
  mcp_server_picker.tsx
  mcp_tool_picker.tsx
  mcp_resource_picker.tsx
  mcp_prompt_picker.tsx
  mcp_prompt_args_form.tsx
  mcp_resource_template_args_form.tsx
  mcp_status_badge.tsx
  mcp_chips.tsx
```

Update:

```text
frontend/app/chats/composer/composer_box.tsx
frontend/app/chats/composer/editor/editor_bottom_bar.tsx
frontend/app/chats/composer/editor/editor_chips_bar.tsx
frontend/app/chats/conversation/hydration_helper.ts
frontend/app/chats/conversation/conversation_persistence_mapper.ts
frontend/app/chats/conversation/use_send_message.ts
frontend/app/spec/conversation.ts
frontend/app/spec/mcp.ts
```

### Backend files

Update conversation specs:

```text
internal/conversation/spec/req_resp.go
internal/inferencewrapper/spec/req_resp.go
```

### Composer state

`useComposerMcp()` owns:

- selected server refs
- per-server tool exposure mode
- selected MCP tools
- selected resources
- selected resource-template instances
- selected prompts
- prompt/resource-template argument values
- stale ref warnings
- runtime status display
- MCP context snapshot/restore for edit flows

### Bottom bar behavior

Add an `MCP Servers` section beside:

- Attachments
- Prompts
- Tools
- Skills
- Web Search
- System Prompt

The MCP picker shows:

- configured servers
- connection status
- transport type
- auth status
- selected state
- available tools/resources/prompts
- policy summary

### Chips bar behavior

Add MCP chips for:

- selected servers
- exposed tool mode, such as `All tools`, `3 tools`, or `No tools`
- selected resources
- selected resource templates
- selected prompts
- stale references

### Conversation message persistence

Add optional `mcp` context to stored user messages:

```ts
export interface StoreConversationMessage {
  // existing fields...
  mcp?: MCPConversationContext;
}
```

Assistant messages and tool output records should store MCP tool provenance when applicable:

```ts
export interface MCPToolCallProvenance {
  serverID: string;
  serverDisplayName?: string;

  toolName: string;
  providerToolName: string;
  toolDigest?: string;

  toolUseID?: string;
  approvalID?: string;

  appResourceUri?: string;
  appInstanceID?: string;
}
```

### Restore workflow

```text
Conversation is hydrated
  -> hydration extracts latest user message MCP context
  -> composer restores selected servers
  -> runtime status refresh starts in background
  -> stale/missing servers are marked
  -> selected tools/resources/prompts are restored where possible
  -> missing refs remain visible as stale chips
```

### Edit older message workflow

```text
User edits older message
  -> composer snapshots current MCP context
  -> older message MCP context is loaded
  -> user cancels edit
  -> prior MCP context is restored

User resends edited message
  -> edited message replaces old message
  -> later messages are dropped
  -> edited MCP context is persisted with the new branch
```

## Step 7. Add inference bridge for MCP tools, resources, and prompts

### Requirements handled

- FR-8 inference hydration
- FR-11 resources and prompts
- FR-6 selected context send flow
- NFR-2 privacy
- NFR-4 maintainability
- NFR-6 performance

### Backend packages and files

Add:

```text
internal/mcp/inferencebridge/
  bridge.go
  tool_hydration.go
  provider_names.go
  resource_hydration.go
  prompt_hydration.go
  instructions.go
  limits.go
```

Update:

```text
internal/inferencewrapper/provider_set.go
internal/inferencewrapper/spec/req_resp.go
```

### Request model update

Add MCP context to completion request:

```go
type FetchCompletionRequest struct {
  // existing fields...
  MCP *mcpSpec.MCPConversationContext `json:"mcp,omitempty"`
}
```

### Provider tool naming

Provider tool names must be deterministic and provider-safe.

Recommended pattern:

```text
mcp__<short-server-id>__<sanitized-tool-name>
```

If this exceeds provider limits or collides, append a short digest:

```text
mcp__github__create_issue__a1b2c3
```

Store mapping:

```ts
export interface MCPProviderToolMapping {
  providerToolName: string;
  choiceID: string;
  serverID: string;
  toolName: string;
  toolDigest: string;
  appResourceUri?: string;
  visibility: Array<"model" | "app">;
}
```

### Tool hydration workflow

```text
FetchCompletion receives MCP context
  -> inferencebridge loads runtime snapshot for selected servers
  -> skips disconnected or missing servers unless cached tool defs are valid
  -> resolves tool exposure mode per server
  -> filters app-only tools out of model-visible tools
  -> filters unsupported task-required tools
  -> validates provider-safe names
  -> produces provider tool definitions
  -> returns tool mapping for tool-call resolution
```

### Resource hydration workflow

```text
FetchCompletion receives selected MCP resources
  -> inferencebridge calls runtime readMcpResource
  -> runtime calls resources/read
  -> text resources become text content blocks
  -> binary resources become supported provider blocks where possible
  -> unsupported binary resources become readable text fallback
  -> metadata is preserved in debug/details
```

Resource content must be treated like attachments. The user selected it, but it may still contain untrusted data.

### Resource template hydration workflow

```text
FetchCompletion receives selected resource templates and arguments
  -> arguments are validated as present
  -> runtime resolves template by reading generated URI or SDK-supported template path
  -> resulting contents are normalized like resources
```

### Prompt hydration workflow

```text
FetchCompletion receives selected MCP prompts
  -> runtime calls prompts/get with arguments
  -> returns PromptMessage[]
  -> inferencebridge maps PromptMessage content blocks into request messages
  -> UI/debug metadata records source server and prompt
```

Do not silently append MCP prompt messages to the system prompt. If the composer offers a mode that contributes MCP prompt text to system instructions, that mode must be explicit and visible.

### Server instructions

MCP server `instructions` may be useful but should not be injected by default.

Only include server instructions when:

- the server selection has `includeServerInstructions: true`
- the UI clearly shows that server instructions are included
- policy permits it

### Send-turn workflow

```text
User sends chat turn
  -> frontend includes MCPConversationContext in completion request
  -> inferencewrapper builds normal request state
  -> inferencebridge hydrates MCP tools/resources/prompts
  -> skills and static tools are hydrated as before
  -> provider request is sent
  -> tool mapping is returned in debug/details and frontend runtime state
  -> final assistant response is persisted
```

## Step 8. Add MCP tool execution, policy, and approvals

### Requirements handled

- FR-9 MCP tool execution
- FR-10 approval and policy
- FR-7 tool provenance persistence
- NFR-1 security
- NFR-7 observability

### Backend packages and files

Add:

```text
internal/mcp/policy/
  policy.go
  risk.go
  visibility.go
  stale.go

internal/mcp/approval/
  manager.go
  audit.go
  token.go
  timeout.go

internal/mcp/toolbridge/
  resolve.go
  invoke.go
  result.go
  provenance.go
```

Update:

```text
internal/mcp/runtime/tools.go
cmd/agentgo/wrapper_mcp.go
```

### Frontend files

Update:

```text
frontend/app/chats/composer/toolruntime/execute_tool_call.ts
frontend/app/chats/composer/toolruntime/*
frontend/app/chats/messages/*
frontend/app/spec/toolruntime.ts
frontend/app/spec/mcp.ts
```

Add:

```text
frontend/app/chats/composer/mcp/mcp_approval_modal.tsx
frontend/app/chats/composer/mcp/use_mcp_approval.ts
```

### Invocation request

```ts
export interface InvokeMCPToolRequest {
  source: "model" | "user" | "app";

  serverID: string;
  toolName: string;
  providerToolName?: string;
  toolDigest?: string;

  arguments?: Record<string, unknown>;

  approvalToken?: string;

  conversationID?: string;
  messageID?: string;
  toolUseID?: string;

  appInstanceID?: string;
}
```

### Invocation response

```ts
export interface InvokeMCPToolResponse {
  serverID: string;
  toolName: string;
  providerToolName?: string;

  content: MCPContentBlock[];
  structuredContent?: Record<string, unknown>;
  isError?: boolean;

  provenance: MCPToolCallProvenance;
  app?: MCPToolAppRenderInfo;
}
```

### Approval evaluation

```ts
export interface MCPApprovalEvaluation {
  decision: "allowed" | "denied" | "approvalRequired";
  reason?: string;
  approvalID?: string;
  summary?: MCPApprovalSummary;
}
```

### Approval workflow

```text
Model returns MCP tool call
  -> frontend resolves providerToolName through tool mapping
  -> frontend calls evaluateMcpToolCall
  -> backend validates server/tool/context/policy
  -> if allowed, frontend calls invokeMcpTool
  -> if denied, frontend renders denied tool output
  -> if approvalRequired, frontend shows approval modal
  -> user chooses allow once, allow always, deny once, or deny always
  -> frontend calls resolveMcpApproval
  -> backend records decision and returns approval token if allowed
  -> frontend calls invokeMcpTool with approval token
  -> backend verifies and consumes token
  -> runtime calls tools/call through SDK
  -> result is normalized and returned
```

### Backend enforcement

Before executing an MCP tool, backend must verify:

- server exists
- server is enabled
- session is ready or can connect
- tool exists in latest or accepted cached discovery
- tool digest matches or stale policy permits execution
- tool was selected or exposed in current conversation context
- app-only/model visibility rules are respected
- approval policy permits execution
- approval token is valid if required
- task-required tools are rejected as unsupported

### Tool result handling

MCP tool results may contain:

- text
- images
- audio
- resource links
- embedded resources
- structured content
- `isError`

Normalize these into FlexiGPT tool output records while preserving MCP-specific metadata.

### Auto-execute behavior

Auto-execute may run MCP tools only when:

- `executionMode` is `auto`
- backend policy returns `allowed`
- no approval is required
- tool args validate
- tool is not unsupported due to task requirement
- composer is not blocked

Auto-execute must never bypass backend policy.

### Timeline rendering

MCP tool calls should show:

- server display name
- tool display name
- arguments
- approval status
- execution status
- result content
- structured content preview
- resource links returned by tool
- app UI if available

## Step 9. Add MCP Apps extension support

### Requirements handled

- FR-12 MCP Apps extension
- FR-9 app-capable tool result rendering
- FR-10 app-only tool policy
- NFR-1 security
- NFR-2 privacy
- NFR-4 maintainability
- NFR-6 performance

### Backend packages and files

Add:

```text
internal/mcp/apps/
  instance.go
  resource.go
  bridge.go
  visibility.go
  csp.go
  policy.go
```

Update:

```text
internal/mcp/session/capabilities.go
internal/mcp/runtime/resources.go
internal/mcp/toolbridge/result.go
cmd/agentgo/wrapper_mcp.go
```

### Frontend files

Add:

```text
frontend/app/chats/mcpapps/
  mcp_app_view.tsx
  mcp_app_sandbox.tsx
  mcp_app_postmessage_bridge.ts
  mcp_app_host_context.ts
  mcp_app_rpc_router.ts
  mcp_app_lifecycle.ts
  mcp_app_types.ts
```

Update:

```text
frontend/app/chats/messages/*
frontend/app/spec/mcp.ts
frontend/app/apis/interface.ts
```

### Apps capability negotiation

When Apps support is available and sandbox requirements are met, advertise:

```json
{
  "capabilities": {
    "extensions": {
      "io.modelcontextprotocol/ui": {
        "mimeTypes": ["text/html;profile=mcp-app"]
      }
    }
  }
}
```

If sandbox requirements are not met, do not advertise Apps support.

### UI resource model

```ts
export interface MCPUIResourceMeta {
  csp?: {
    connectDomains?: string[];
    resourceDomains?: string[];
    frameDomains?: string[];
    baseUriDomains?: string[];
  };
  permissions?: {
    camera?: {};
    microphone?: {};
    geolocation?: {};
    clipboardWrite?: {};
  };
  domain?: string;
  prefersBorder?: boolean;
}
```

```ts
export interface MCPUIResourceContent {
  serverID: string;
  uri: string;
  mimeType: "text/html;profile=mcp-app";
  html: string;
  meta?: MCPUIResourceMeta;
  digest: string;
}
```

### App instance model

```ts
export interface MCPAppInstance {
  instanceID: string;
  serverID: string;
  resourceUri: string;
  toolName?: string;
  toolUseID?: string;
  conversationID?: string;
  messageID?: string;
  createdAt: string;
  status: "initializing" | "ready" | "tearingDown" | "closed" | "error";
}
```

### App discovery

Tools may include:

```json
{
  "_meta": {
    "ui": {
      "resourceUri": "ui://weather-server/dashboard-template",
      "visibility": ["model", "app"]
    }
  }
}
```

Rules:

- `visibility` defaults to `["model", "app"]`.
- Tools without `"model"` visibility must not be exposed to the LLM.
- Tools without `"app"` visibility must not be callable by Apps.
- App tool calls are restricted to the same server connection.

### App rendering workflow

```text
Tool has _meta.ui.resourceUri
  -> model or user invokes the tool
  -> backend executes tools/call
  -> tool result returns normal MCP CallToolResult
  -> frontend sees app render info
  -> frontend creates app instance
  -> frontend/backend reads ui:// resource through resources/read
  -> frontend validates MIME type
  -> frontend renders sandboxed iframe
  -> View sends ui/initialize
  -> Host returns host capabilities and host context
  -> View sends ui/notifications/initialized
  -> Host sends ui/notifications/tool-input
  -> Host sends ui/notifications/tool-result
```

### Interactive app workflow

```text
User interacts with App View
  -> App sends JSON-RPC postMessage
  -> frontend validates message source and app instance
  -> frontend routes request through app RPC router
  -> backend verifies same-server and visibility policy
  -> backend executes allowed tools/resources through MCP runtime
  -> frontend sends JSON-RPC response back to App View
```

Supported View-to-Host requests:

- `tools/call`
- `resources/read`
- `notifications/message`
- `ui/open-link`
- `ui/message`
- `ui/update-model-context`
- `ui/request-display-mode`

Supported Host-to-View messages:

- `ui/notifications/tool-input`
- `ui/notifications/tool-input-partial`
- `ui/notifications/tool-result`
- `ui/notifications/tool-cancelled`
- `ui/notifications/host-context-changed`
- `ui/resource-teardown`

### Sandbox requirements

MCP App HTML must never run with privileged Wails bindings.

Frontend must ensure:

- App HTML is rendered only in a sandboxed iframe.
- The App iframe cannot access parent DOM.
- The App iframe cannot access Wails runtime bindings.
- `postMessage` bridge validates source window and instance ID.
- CSP is enforced based on UI resource metadata.
- Restrictive CSP defaults are used when no metadata is present.
- External links require explicit user consent.
- App teardown is attempted before iframe removal.

Restrictive default CSP:

```text
default-src 'none';
script-src 'self' 'unsafe-inline';
style-src 'self' 'unsafe-inline';
img-src 'self' data:;
media-src 'self' data:;
connect-src 'none';
frame-src 'none';
object-src 'none';
base-uri 'self';
```

The host may further restrict CSP but must not allow undeclared external domains.

### Host context

Return relevant context in `ui/initialize` response:

```ts
export interface MCPAppHostContext {
  theme?: "light" | "dark";
  displayMode?: "inline" | "fullscreen" | "pip";
  availableDisplayModes?: string[];
  containerDimensions?: {
    width?: number;
    height?: number;
    maxWidth?: number;
    maxHeight?: number;
  };
  locale?: string;
  timeZone?: string;
  userAgent?: string;
  platform?: "desktop";
  styles?: {
    variables?: Record<string, string | undefined>;
    css?: {
      fonts?: string;
    };
  };
}
```

### App security decisions

- `ui/open-link` must show full URL and require confirmation unless user policy allows it.
- `ui/message` may request adding a user message to the chat; require confirmation by default.
- `ui/update-model-context` may update future model context; show it or require policy approval.
- App `notifications/message` is logging only and must not enter model context.
- App tool calls go through the same backend policy as other MCP tool calls.

## Step 10. Add docs, tests, observability, and shutdown hardening

### Requirements handled

- FR-14 documentation and settings
- NFR-1 security
- NFR-2 privacy
- NFR-5 reliability
- NFR-7 observability
- NFR-8 compatibility

### Documentation updates

Update bundled docs:

```text
frontend/app/docs/content/02-core-concepts.md
frontend/app/docs/content/03-chats-composer-and-everyday-workflow.md
frontend/app/docs/content/04-attachments-tools-skills-prompts.md
frontend/app/docs/content/05-presets-providers-settings.md
frontend/app/docs/content/06-privacy-storage-and-troubleshooting.md
frontend/app/docs/content/11-architecture-overview.md
frontend/app/docs/content/12-backend-roles-and-responsibilities.md
frontend/app/docs/content/13-frontend-roles-and-responsibilities.md
frontend/app/docs/content/14-chats-workspace-and-composer-hld.md
```

Add an MCP architecture doc if desired:

```text
frontend/app/docs/content/15-mcp-support-hld.md
```

Docs must explain:

- MCP server concept
- HTTP and stdio server risks
- OAuth auth flow
- MCP tools vs static tools
- MCP resources and prompts
- MCP Apps sandboxing
- approvals
- privacy implications
- local storage and secrets

### Backend tests

Add tests for:

- server config validation
- store CRUD
- secret ref validation
- transport config construction
- auth metadata parsing
- policy decisions
- approval token lifecycle
- provider tool name mapping
- tool digest stale detection
- MCP content normalization
- Apps CSP construction
- app visibility enforcement

Use SDK test servers or mocked SDK interfaces for protocol behavior.

### Frontend tests

Add tests for:

- MCP server form validation
- composer MCP restore
- stale chips
- tool exposure selection
- approval modal behavior
- Apps postMessage routing
- Apps teardown
- blocked app request behavior

### Observability

Add MCP diagnostics surfaces:

- server status
- last connection error
- negotiated protocol version
- server capabilities
- auth status
- discovery digest
- tool/resource/prompt counts
- server logs from `notifications/message`
- approval audit records

### Shutdown hardening

On app shutdown:

```text
MCP runtime stops accepting new requests
  -> active app instances receive teardown
  -> in-flight MCP requests are cancelled best-effort
  -> HTTP sessions are closed
  -> stdio process stdin is closed
  -> stdio process receives terminate if needed
  -> stdio process is killed if terminate times out
  -> MCP store is closed
  -> other stores with Close() are closed
```

Also fix existing backend shutdown gaps for stores that already expose `Close()` methods.
