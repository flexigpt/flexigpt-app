# MCP Support HLD

## 1. Design summary

### Primary user concept

- User adds an `MCP Server`
- The server exposes:
  - `Tools`
  - `Resources`
  - `Prompts`

### Core split

- `Local store`
  - persistent server config, policies, secret references, scope
- `MCP runtime`
  - live connection/session, discovery, tool execution, resource/prompt fetch
- `UI`
  - edit servers, show status, choose what is allowed in a chat, show approvals
- `Composer`
  - per-chat selection and restoration
- `Provider/inference layer`
  - hydrates selected MCP tools into model tool choices

### Important product rule

- The user should primarily think in terms of:
  - `Use this MCP Server in this chat`
  - then optionally `Allow only these tools from this server`
- `Resources` and `Prompts` should remain concrete, server-scoped concepts, not abstract “capabilities”.

## 2. Architecture overview

```text
UI
  /mcpservers page
  chat composer MCP picker
  approval modal
      ↓ Wails API
Local store
  MCP server config + policies + secret refs
      ↓ runtime control
MCP runtime
  live client sessions
  discovery snapshots
  tool/resource/prompt fetch
  tool execution
      ↓ inference hydration
Provider wrapper
  merges static tools + MCP tools + skills + web search
      ↓ LLM
Model returns tool call
      ↓
Composer runtime / approval flow
  approve → backend executes MCP tool → result shown in chat
```

The important thing is that the backend owns the live MCP connection and the UI never speaks protocol directly.

---

## 3. How this maps to your existing app

Your repo already has two useful patterns:

### Pattern A: `tools`

- static, versioned catalog
- local CRUD
- bundle/version semantics

This is not a good fit for MCP, because MCP tools are dynamic and runtime-discovered.

### Pattern B: `skills`

- persistent store plus live runtime
- stale-safe refresh
- status badges
- runtime sessions
- best-effort reconciliation

This is the right mental model for MCP.

So MCP should be built more like `skills` than like `tools`.

---

## 4. Proposed backend structure

### New backend package

Add:

- `internal/mcp/spec`
- `internal/mcp/store`
- `internal/mcp/runtime`
- `internal/mcp/policy`
- `internal/mcp/approval`
- `internal/mcp/bridge`

And a Wails wrapper:

- `cmd/agentgo/wrapper_mcp.go`

### Responsibilities

#### `internal/mcp/store`

Owns persistent data:

- server configs
- enable/disable
- scope
- policy
- secret references
- last-known metadata

#### `internal/mcp/runtime`

Owns live operations:

- start/stop local stdio server processes
- connect to remote servers
- initialize handshake
- list tools/resources/prompts
- invoke tools
- read resources
- get prompts
- cache discovery snapshots

#### `internal/mcp/policy`

Owns decision rules:

- per-server allow/deny/ask
- per-tool allow/deny/ask
- read/write/destructive risk classification
- chat-scoped overrides

#### `internal/mcp/approval`

Owns pending approval requests:

- approval IDs
- timeout
- allow once
- allow always
- deny
- audit trail

#### `internal/mcp/bridge`

Owns conversion between:

- MCP tool/resource/prompt shapes
- app-level tool choices
- provider tool definitions
- UI-friendly summaries

---

## 5. Data models

Below are the concrete shapes I would introduce.

### 5.1 Server configuration

#### Go shape

```go
type MCPTransportType string

const (
  MCPTransportStdio MCPTransportType = "stdio"
  MCPTransportHTTP  MCPTransportType = "http"
)

type MCPServerScope string

const (
  MCPServerScopeGlobal MCPServerScope = "global"
  MCPServerScopeManual MCPServerScope = "manual"
)

type MCPApprovalMode string

const (
  MCPApprovalModeAsk  MCPApprovalMode = "ask"
  MCPApprovalModeAuto  MCPApprovalMode = "auto"
  MCPApprovalModeDeny  MCPApprovalMode = "deny"
)
```

```go
type MCPStdioConfig struct {
  Command         string            `json:"command"`
  Args            []string          `json:"args,omitempty"`
  WorkingDir      string            `json:"workingDir,omitempty"`
  Env             map[string]string `json:"env,omitempty"`
  SecretEnvRefs   map[string]string `json:"secretEnvRefs,omitempty"`
  StartupTimeoutMS int               `json:"startupTimeoutMS,omitempty"`
}

type MCPHTTPConfig struct {
  URL              string            `json:"url"`
  Headers          map[string]string `json:"headers,omitempty"`
  SecretHeaderRefs  map[string]string `json:"secretHeaderRefs,omitempty"`
  TimeoutMS        int               `json:"timeoutMS,omitempty"`
  TransportVariant  string            `json:"transportVariant,omitempty"` // streamable-http, sse
}
```

```go
type MCPToolPolicy struct {
  ApprovalMode           MCPApprovalMode `json:"approvalMode"`
  AutoExecute            bool            `json:"autoExecute"`
  RequireApprovalForWrite bool            `json:"requireApprovalForWrite"`
}
```

```go
type MCPServer struct {
  SchemaVersion string         `json:"schemaVersion"`
  ID            string         `json:"id"`
  DisplayName   string         `json:"displayName"`
  Enabled       bool           `json:"enabled"`
  Transport     MCPTransportType `json:"transport"`
  Stdio         *MCPStdioConfig `json:"stdio,omitempty"`
  HTTP          *MCPHTTPConfig  `json:"http,omitempty"`
  Scope         MCPServerScope  `json:"scope"`
  DefaultPolicy MCPToolPolicy   `json:"defaultPolicy"`
  CreatedAt     time.Time       `json:"createdAt"`
  ModifiedAt    time.Time       `json:"modifiedAt"`
  SoftDeletedAt  *time.Time      `json:"softDeletedAt,omitempty"`
}
```

### 5.2 Runtime discovery snapshot

```go
type MCPServerStatus string

const (
  MCPServerStatusDisabled   MCPServerStatus = "disabled"
  MCPServerStatusConnecting MCPServerStatus = "connecting"
  MCPServerStatusReady      MCPServerStatus = "ready"
  MCPServerStatusError      MCPServerStatus = "error"
)
```

```go
type MCPServerRuntimeSnapshot struct {
  ServerID      string `json:"serverID"`
  Status        MCPServerStatus `json:"status"`
  LastError     string `json:"lastError,omitempty"`
  LastSyncedAt  time.Time `json:"lastSyncedAt"`
  ToolCount     int    `json:"toolCount"`
  ResourceCount  int    `json:"resourceCount"`
  PromptCount    int    `json:"promptCount"`
  SnapshotDigest string `json:"snapshotDigest,omitempty"`
}
```

### 5.3 Tool capability

```go
type MCPToolRisk string

const (
  MCPToolRiskRead        MCPToolRisk = "read"
  MCPToolRiskWrite       MCPToolRisk = "write"
  MCPToolRiskDestructive MCPToolRisk = "destructive"
  MCPToolRiskUnknown     MCPToolRisk = "unknown"
)
```

```go
type MCPToolCapability struct {
  ChoiceID      string          `json:"choiceID"`
  ServerID      string          `json:"serverID"`
  ToolName      string          `json:"toolName"`
  DisplayName   string          `json:"displayName"`
  Description   string          `json:"description,omitempty"`
  InputSchema   JSONSchema      `json:"inputSchema"`
  Risk          MCPToolRisk     `json:"risk"`
  ApprovalMode  MCPApprovalMode `json:"approvalMode"`
  AutoExecute   bool            `json:"autoExecute"`
  Digest        string          `json:"digest"`
  Enabled       bool            `json:"enabled"`
}
```

### 5.4 Resources

```go
type MCPResourceRef struct {
  ServerID    string `json:"serverID"`
  URI         string `json:"uri"`
  DisplayName string `json:"displayName"`
  MIMEType    string `json:"mimeType,omitempty"`
  Digest      string `json:"digest,omitempty"`
}
```

### 5.5 Prompts

```go
type MCPPromptRef struct {
  ServerID    string `json:"serverID"`
  PromptName  string `json:"promptName"`
  DisplayName string `json:"displayName"`
  Digest      string `json:"digest,omitempty"`
}
```

### 5.6 Chat-level selection snapshot

This is what should be stored in conversation messages and restored into the composer.

```go
type MCPConversationState struct {
  EnabledServerRefs []MCPServerRef      `json:"enabledServerRefs,omitempty"`
  ToolChoices       []MCPToolSelection  `json:"toolChoices,omitempty"`
  ResourceRefs      []MCPResourceRef    `json:"resourceRefs,omitempty"`
  PromptRefs        []MCPPromptRef      `json:"promptRefs,omitempty"`
}
```

```go
type MCPServerRef struct {
  ServerID string `json:"serverID"`
  Digest   string `json:"digest,omitempty"`
}
```

```go
type MCPToolSelection struct {
  ChoiceID    string          `json:"choiceID"`
  ServerID    string          `json:"serverID"`
  ToolName    string          `json:"toolName"`
  Digest      string          `json:"digest,omitempty"`
  AutoExecute bool            `json:"autoExecute"`
  ApprovalMode MCPApprovalMode `json:"approvalMode"`
}
```

The digest is important because it lets you harden stale refs the same way skills use `SkillID`.

---

## 6. Frontend data model

Add `frontend/app/spec/mcp.ts`.

### Suggested frontend types

```ts
export type MCPTransportType = "stdio" | "http";
export type MCPServerScope = "global" | "manual";
export type MCPApprovalMode = "ask" | "auto" | "deny";
export type MCPServerStatus = "disabled" | "connecting" | "ready" | "error";
export type MCPToolRisk = "read" | "write" | "destructive" | "unknown";

export interface MCPServer {
  schemaVersion: string;
  id: string;
  displayName: string;
  enabled: boolean;
  transport: MCPTransportType;
  stdio?: MCPStdioConfig;
  http?: MCPHTTPConfig;
  scope: MCPServerScope;
  defaultPolicy: MCPToolPolicy;
  createdAt: string;
  modifiedAt: string;
  softDeletedAt?: string;
}

export interface MCPServerRuntimeSnapshot {
  serverID: string;
  status: MCPServerStatus;
  lastError?: string;
  lastSyncedAt: string;
  toolCount: number;
  resourceCount: number;
  promptCount: number;
  snapshotDigest?: string;
}

export interface MCPToolCapability {
  choiceID: string;
  serverID: string;
  toolName: string;
  displayName: string;
  description?: string;
  inputSchema: JSONSchema;
  risk: MCPToolRisk;
  approvalMode: MCPApprovalMode;
  autoExecute: boolean;
  digest: string;
  enabled: boolean;
}

export interface MCPResourceRef {
  serverID: string;
  uri: string;
  displayName: string;
  mimeType?: string;
  digest?: string;
}

export interface MCPPromptRef {
  serverID: string;
  promptName: string;
  displayName: string;
  digest?: string;
}

export interface MCPConversationState {
  enabledServerRefs: MCPServerRef[];
  toolChoices: MCPToolSelection[];
  resourceRefs: MCPResourceRef[];
  promptRefs: MCPPromptRef[];
}
```

---

## 7. Conversation persistence changes

Your conversation storage is the place where chat-specific MCP state should live.

### Add to `StoreConversationMessage` and `ConversationMessage`

In `frontend/app/spec/conversation.ts`, add:

```ts
mcpServerRefs?: MCPServerRef[];
mcpToolChoices?: MCPToolSelection[];
mcpResourceRefs?: MCPResourceRef[];
mcpPromptRefs?: MCPPromptRef[];
```

### Add to `RestorableConversationContext`

```ts
mcpServerRefs: MCPServerRef[];
mcpToolChoices: MCPToolSelection[];
mcpResourceRefs: MCPResourceRef[];
mcpPromptRefs: MCPPromptRef[];
```

### Update hydration helpers

In `frontend/app/chats/conversation/hydration_helper.ts`:

- when restoring from the latest user message, extract those MCP refs
- when building a user message from the composer, persist those refs
- when re-editing an older message, restore them the same way you already restore:
  - tool choices
  - skill refs
  - attachments

This should mirror your current tool/skill restore flow.

---

## 8. Composer design

The composer should own MCP state the same way it owns tools, skills, and prompts.

### Add a new composer subsystem

Suggested new hooks:

- `useComposerMcpServers()`
- `useComposerMcpTools()`
- `useComposerMcpResources()`
- `useComposerMcpPrompts()`

Or, if you want a simpler initial shape:

- `useComposerMcp()`

That hook should provide:

- enabled server refs for the current chat
- selected tool allowlist
- selected resource refs
- selected prompt refs
- current runtime snapshot
- `connect / refresh / disconnect`
- stale ref filtering
- tool args blocking state
- approval state

### Composer UI additions

In `EditorBottomBar`, add an MCP server button/section:

- `MCP Servers`
  - list selected servers
  - show server status
  - open server tool/resource/prompt picker
  - show approval mode summary

This should sit alongside:

- Attachments
- Prompts
- Tools
- Skills
- System Prompt

### Chips bar

In `EditorChipsBar`, add an MCP-specific chip group that shows:

- enabled servers
- selected tools
- attached resources
- selected prompts

This keeps the selected MCP state visible in the draft, just like tools/skills.

---

## 9. Backend APIs

I would split APIs into two Wails-facing wrappers:

- `IMCPStoreAPI`
- `IMCPRuntimeAPI`

This matches your store/runtime split cleanly.

### 9.1 Store API

#### Suggested methods

```ts
listMcpServers(
  includeDisabled?: boolean,
  pageSize?: number,
  pageToken?: string
): Promise<{ mcpServers: MCPServer[]; nextPageToken?: string }>

getMcpServer(serverID: string): Promise<MCPServer | undefined>

putMcpServer(serverID: string, payload: PutMCPServerPayload): Promise<void>

patchMcpServerEnabled(serverID: string, enabled: boolean): Promise<void>

patchMcpServerPolicy(serverID: string, payload: PatchMCPServerPolicyPayload): Promise<void>

deleteMcpServer(serverID: string): Promise<void>
```

#### Payloads

```ts
export interface PutMCPServerPayload {
  displayName: string;
  enabled: boolean;
  transport: MCPTransportType;
  stdio?: MCPStdioConfig;
  http?: MCPHTTPConfig;
  scope: MCPServerScope;
  defaultPolicy: MCPToolPolicy;
}

export interface PatchMCPServerPolicyPayload {
  scope?: MCPServerScope;
  defaultPolicy?: MCPToolPolicy;
}
```

### 9.2 Runtime API

```ts
connectMcpServer(serverID: string): Promise<void>
disconnectMcpServer(serverID: string): Promise<void>
refreshMcpServer(serverID: string): Promise<void>
getMcpServerStatus(serverID: string): Promise<MCPServerRuntimeSnapshot>
listMcpServerTools(serverID: string, includeDisabled?: boolean): Promise<{ tools: MCPToolCapability[] }>
listMcpServerResources(serverID: string): Promise<{ resources: MCPResourceRef[] }>
listMcpServerPrompts(serverID: string): Promise<{ prompts: MCPPromptRef[] }>
invokeMcpTool(serverID: string, toolName: string, args?: JSONRawString): Promise<InvokeMcpToolResponse>
readMcpResource(serverID: string, uri: string): Promise<MCPResourceContent>
getMcpPrompt(serverID: string, promptName: string, args?: JSONRawString): Promise<MCPPromptContent>
```

#### Why separate store and runtime APIs?

Because this mirrors the real split:

- store is config/policy persistence
- runtime is live protocol execution

That makes the code easier to reason about and keeps the UI honest about what is configured vs what is currently reachable.

---

## 10. Backend inference integration

This is the most important plumbing.

Your current provider wrapper already hydrates:

- static tools
- skills
- attachments
- model params

MCP should be added there, not in the frontend.

### Update `internal/inferencewrapper/spec/req_resp.go`

Add request fields:

```go
McpServerRefs []mcpSpec.MCPServerRef `json:"mcpServerRefs,omitempty"`
McpToolChoices []mcpSpec.MCPToolSelection `json:"mcpToolChoices,omitempty"`
McpResourceRefs []mcpSpec.MCPResourceRef `json:"mcpResourceRefs,omitempty"`
McpPromptRefs []mcpSpec.MCPPromptRef `json:"mcpPromptRefs,omitempty"`
```

### Update `ProviderSetAPI`

In `internal/inferencewrapper/provider_set.go`:

- resolve MCP server refs against `MCPStore`/`MCPRuntime`
- convert selected MCP tools into provider tool choices
- fetch attached resources and convert them to input blocks
- fetch selected prompts and append them to the system prompt or prompt stack

#### Suggested flow inside `FetchCompletion`

1. Resolve model param
2. Build history/current inputs
3. Hydrate static tool choices
4. Hydrate MCP tool choices
5. Hydrate skills prompt/tools
6. Fetch MCP resources and materialize them into inputs
7. Fetch MCP prompts and append them to prompt text
8. Call provider

### Tool choice mapping

MCP tools should be exposed to the model as tool choices with:

- unique `choiceID`
- name
- description
- JSON schema
- risk classification
- approval mode

If a model returns a tool call with a given `choiceID`, the runtime can resolve it back to:

- MCP server
- MCP tool name
- current digest
- policy

This is how you keep the tool-call loop deterministic.

---

## 11. Tool-call execution flow

This is the runtime path after the model returns a tool call.

### Current system

Your `executeComposerToolCall.ts` currently routes:

- skills tool names → `skillStoreAPI.invokeSkillTool(...)`
- everything else → `toolRuntimeAPI.invokeTool(...)`

### Proposed MCP branch

Add a third branch:

- MCP tool call → `mcpRuntimeAPI.invokeMcpTool(...)`

### Approval flow

The frontend should still render the approval UI, but the backend must enforce policy.

#### Flow

```text
Model returns MCP tool call
  ↓
Composer runtime checks tool policy
  ↓
If approval needed:
  - emit approval event
  - show modal
  - wait for user action
  ↓
If approved:
  - frontend calls backend invokeMcpTool(...)
  - backend resolves server + tool + digest
  - backend executes through MCP runtime
  - tool output returns to composer
  ↓
Composer appends tool output and continues
```

#### Approval decisions

Support:

- `Allow once`
- `Always allow this tool`
- `Deny`
- optionally `Always deny this tool`

The backend should store these decisions in the MCP policy store.

---

## 12. Server, tool, resource, prompt flows

### 12.1 Add server

```text
User opens MCP Servers page
  -> clicks Add Server
  -> enters stdio or remote config
  -> saves
Backend
  -> validates config
  -> stores config
  -> optionally connects immediately
  -> discovers capabilities
UI
  -> shows ready/error status
```

### 12.2 Use a server in a chat

```text
User opens chat composer
  -> selects MCP Server
  -> server refs are added to draft state
  -> selected server/tool refs are persisted in the user message
Send
  -> current draft sends server refs to backend
Backend
  -> resolves live MCP server session
  -> exposes allowed tools to provider
```

### 12.3 Use only selected tools from a server

```text
User enables MCP Server
  -> opens Tools from this server
  -> selects explicit allowlist
  -> maybe sets approval mode per tool
Send
  -> backend only exposes allowed tools
  -> other server tools are hidden from the model
```

### 12.4 Attach a resource

```text
User browses Resources from the server
  -> selects a resource
  -> resource ref is stored in draft state
Send
  -> backend fetches resource content
  -> backend materializes it into prompt input blocks
```

### 12.5 Insert a prompt

```text
User browses Prompts from the server
  -> selects a prompt
  -> prompt ref is stored in draft state
Send
  -> backend fetches prompt definition/content
  -> backend appends it to the prompt stack
```

---

## 13. Local store vs runtime vs UI, cleanly separated

### Local store

Owns:

- server config
- server scope
- approval defaults
- tool policy
- secret refs
- user-installed servers

Does not own:

- live protocol sessions
- current tool/resource/prompt list
- execution state

### MCP runtime

Owns:

- live server process or HTTP connection
- initialize handshake
- discovered capabilities
- current server status
- tool execution
- resource read
- prompt fetch

Does not own:

- user editing forms
- persistent chat selections
- approval dialogs

### UI

Owns:

- server config forms
- status rendering
- allowlist editors
- chat selection
- approval modal
- visibility and filtering

Does not own:

- protocol logic
- server process lifecycle
- actual tool execution

---

## 14. Suggested file placement

### Backend

- `internal/mcp/spec/*`
- `internal/mcp/store/*`
- `internal/mcp/runtime/*`
- `internal/mcp/policy/*`
- `internal/mcp/approval/*`
- `internal/mcp/bridge/*`
- `cmd/agentgo/wrapper_mcp.go`

Update:

- `cmd/agentgo/app.go`
- `cmd/agentgo/main.go`
- `internal/inferencewrapper/provider_set.go`
- `internal/inferencewrapper/spec/req_resp.go`
- `internal/setting/store/store.go` if you want to reuse keyring storage for MCP secrets

### Frontend

- `frontend/app/mcpservers/page.tsx`
- `frontend/app/mcpservers/*`
- `frontend/app/chats/composer/mcp/*`
- `frontend/app/spec/mcp.ts`
- `frontend/app/apis/interface.ts`

Update:

- `frontend/app/routes.ts`
- `frontend/app/components/sidebar.tsx`
- `frontend/app/chats/composer/editor/editor_bottom_bar.tsx`
- `frontend/app/chats/composer/editor/editor_chips_bar.tsx`
- `frontend/app/chats/composer/composer_box.tsx`
- `frontend/app/chats/conversation/hydration_helper.ts`
- `frontend/app/chats/conversation/use_send_message.ts`
- `frontend/app/chats/composer/toolruntime/execute_tool_call.ts`

---

## 15. Security rules

These should be explicit in the design.

### Local stdio servers

- execute directly, not through shell
- no `sh -c`
- do not inherit full environment by default
- pass only the required env vars
- kill the process on disconnect/app shutdown

### Remote servers

- use HTTPS by default
- show the URL clearly in UI
- store secrets securely
- redact logs

### Tool output

Treat MCP tool output as untrusted:

- label it clearly in chat
- do not silently merge secrets into prompts
- enforce approval for risky tools

---

## 16. MVP recommendation

If you want the smallest good implementation:

### Phase 1

- support MCP Server config storage
- support stdio server runtime
- connect/disconnect/status
- list tools
- execute tools
- server-level selection in chat
- approval modal
- provider hydration of MCP tools

### Phase 2

- resources
- prompts
- per-tool policies
- stale digest hardening
- better status caching

### Phase 3

- remote HTTP transport
- manifest import
- signed/configured server packages
- optional update checks

---

## 17. Final recommendation

The right product model is:

- User adds and manages an `MCP Server`
- The server exposes `Tools`, `Resources`, and `Prompts`
- The app stores server config locally
- The backend runtime connects to the server and discovers capabilities
- The composer persists which MCP servers/tools/resources/prompts are used in a chat
- The provider wrapper hydrates selected MCP tools into the model request
- Tool calls are approved and executed through the backend runtime
