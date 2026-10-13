# Go Agent Runtime Proposal

## Purpose

Build a clean, Session-owned agent execution system for FlexiGPT.

The backend becomes the single owner of execution after a task is accepted. The frontend remains responsible for authoring, selection UI, override UI, rendering, tabs, menus, and rich interaction, but no longer owns:

- Model request progression
- Streamed-operation lifecycle
- Tool scheduling
- Tool retries
- Automatic tool execution
- Automatic continuation
- Skill session lifecycle
- MCP approval lifecycle
- Run cancellation
- Run pause and resume
- Final execution persistence

The new system is intentionally breaking. It does not migrate or preserve the old conversation execution architecture.

## Scope

The target implementation includes:

- Session and Run orchestration
- Agent selection and backend Agent resolution
- Workspace selection and backend Workspace resolution
- Deterministic frontend-visible owner and override semantics
- Model resolution and model-request defaults layering
- Tool, Skill, MCP, Workspace, attachment, and system-prompt preparation
- Provider completion execution through a thin inference runtime
- Tool execution, approval, retries, skips, explicit error completion, and continuation
- Persistence through SessionStore when persistence is enabled
- SessionStore as the sole durable execution persistence authority
- Recovery of safe state without blind replay of external work
- Full frontend migration from direct completion and tool ownership to Session Runtime ownership
- MCP App ownership bound to the originating Session, Run, and view rather than the selected frontend tab

The target does not implement production graph workflows, subagents, or agent-to-agent transfer. The Session and Run model must support those as future drivers without requiring redesign.

## Explicit non-goals

The branch must not include:

- Migration from legacy conversations
- Reading or importing `conversations_v2`
- Compatibility wrappers that allow old frontend chat execution to coexist with Session execution
- Duplicate frontend and backend tool schedulers
- Frontend-owned Skill Runtime session creation for chat
- Frontend-owned MCP approval promises for Session-owned tool calls
- Domain resolution inside the inference runtime
- Session imports inside domain packages
- Artifact storage redesign unless an adapter requires a narrow port addition
- Graph or subagent implementation disguised as incomplete sequential-chat code

A new clean Session storage namespace or schema is acceptable and preferred.

## Historical baseline

The following inventory describes the pre-delivery ownership model and the
work that motivated this proposal. The status sections near the end of this
document are authoritative for the current backend implementation.
The current codebase contains substantial pieces that should be reused, but their ownership must change.

### Existing Agent resolution surface

The frontend currently uses Agent APIs equivalent to:

```text
listAgentCatalogOptions
prepareAgentStarter
```

The current Agent preparation flow resolves a starter recipe containing variants of:

- Selected Model
- Tool selections
- Enabled and active Skills
- MCP context
- System instruction sources
- Model-default system-prompt preference
- Starter text
- Agent display and provenance metadata

The frontend currently applies this recipe directly into Composer state.

#### Target use

The frontend continues using Agent resolution for preview and selection presentation.

The backend independently resolves the selected Agent again from the authoritative Agent domain or store when preparing a Run or model operation.

The frontend does not send trusted resolved Agent content as authority. It sends:

- Agent reference
- Optional expected revision or digest
- Explicit user overrides

### Existing Workspace resolution surface

The frontend currently uses Workspace APIs equivalent to:

```text
listComposerWorkspaceDirectories
resolveWorkspaceRuntimePlan
refreshWorkspaceDirectory
registerWorkspaceDirectory
```

The existing Workspace domain and runtime provide concepts equivalent to:

```text
ResolveWorkspace
ComposeWorkspacePrompt
LoadWorkspaceSkills
```

The current Workspace plan contains:

- Effective Workspace
- Context contributions
- Instruction and user-message composition
- Workspace Skills
- Workspace MCP declarations
- Capability diagnostics
- Revision, digest, and locator information

#### Target use

The frontend continues using Workspace resolution for preview, selection UI, diagnostics, and deterministic override controls.

The backend independently resolves the selected Workspace at execution preparation time.

The frontend sends:

- Workspace reference
- Selected or pinned Context references
- Selected or pinned Workspace Skill references
- Selected or pinned Workspace MCP references
- Explicit additions, removals, and overrides
- Expected revision and digest data for stale-state detection

The backend remains authoritative for execution. It must not trust a frontend-prepared Workspace prompt, Workspace Skill list, or Workspace MCP declaration list as the actual execution source.

### Existing Session Runtime

The current Session Runtime already includes:

- Session and Run state roots
- Sequential chat driver
- Run lifecycle
- Model operations
- Tool calls and attempts
- Waiting requests
- Continuation input
- Pause and cancel controls
- Automation holds
- Limits
- Checkpointing
- Recovery
- Snapshots
- Wails event publication
- SessionStore integration
- MCP, Skill, and Go-tool adapters
- Attachment admission and materialization

This is the correct foundation.

The main missing layer is authoritative resolution and composition before a model operation.

### Existing inference runtime

The current `ProviderSetAPI.FetchCompletion` performs too much:

- Model resolution
- Provider registration
- Capability resolution
- Input assembly
- Workspace hydration
- MCP hydration
- Skill hydration
- Tool preparation
- Provider execution
- Stream adaptation

That is suitable for the old direct frontend completion path, but it is too semantically aware for the target architecture.

The target inference runtime becomes thin:

```text
Provider lifecycle
Provider registration
Provider capability resolution
Provider completion execution
Stream transport
Provider response normalization
```

It must not know:

- Sessions
- Runs
- Agents
- Workspaces
- Skill selections
- MCP selection semantics
- Artifact references
- Conversation history policy
- Attachment resolution
- Frontend overrides

Those semantics move to Session orchestration and runtime-setup adapters.

## Core design principles

### Session is the orchestration engine

Session Runtime owns:

- Accepted execution intent
- Run creation
- Operation ordering
- Lifecycle transitions
- Model-operation ownership
- Tool-call ownership
- Waits and approvals
- Continuation
- Cancellation
- Recovery
- Snapshots
- Event sequencing
- Persistence checkpoints

No frontend callback determines whether the next model or tool action occurs after acceptance.

### Frontend owns presentation and intent editing

Frontend owns:

- Agent and Workspace pickers
- Model, Tool, Skill, and MCP dropdowns
- Override controls
- Draft text
- Attachment picker UI
- Tabs
- Scroll state
- Menus
- Editor state
- Markdown and MCP App rendering
- Approval dialogs
- Rendering throttling
- Snapshot projection into UI state

Frontend does not own:

- Final effective execution composition
- Model dispatch
- Tool dispatch
- Skill session creation
- MCP tool invocation
- Provider completion lifecycle
- Run terminal state

### Backend re-resolves authoritative source state

Frontend preview is not execution authority.

At execution preparation, backend resolves:

- Selected Agent
- Selected Workspace
- Selected Model
- Selected Tools
- Selected Skills
- Selected MCP capabilities
- Selected Workspace Context
- Selected attachments
- Request defaults and overrides

The backend applies user overrides deterministically to backend-resolved source state.

### One authoritative composition path

There must be exactly one effective composition path for a model operation:

```text
Session or Run intent
  -> authoritative backend resolver
  -> prepared model operation
  -> thin inference completion runtime
```

There must not be:

```text
Frontend-composed prompt
  + backend-composed prompt
  + legacy ProviderSet Workspace hydration
```

for the same Session model operation.

### No hidden merges

Every effective configuration contribution must have:

- Source
- Canonical identity
- Owner
- Override state
- Resolution state
- Diagnostics
- Final inclusion or exclusion reason

The frontend must be able to render:

```text
Inherited from Agent
Inherited from Workspace
Added by user
Removed by user
Unavailable
Changed since selected
Blocked by policy
Used for current operation
```

No configuration element may silently appear or disappear due to array order or source refresh.

## Dependency direction

The target dependency graph is:

```text
Frontend
  -> Wails wrappers
    -> runtimesetup
      -> session
      -> inference
      -> domain adapters
        -> Agent domain
        -> Workspace domain
        -> Model domain
        -> Tool domain
        -> Skill domain
        -> MCP domain
        -> Attachment and filesystem domain
      -> sessionstore

session
  -> no domain imports
  -> no inference imports
  -> no Wails imports
  -> no frontend imports

inference
  -> inference-go and provider SDKs
  -> no Session imports
  -> no Agent imports
  -> no Workspace imports
  -> no MCP semantic imports
  -> no Tool or Skill semantic imports
  -> no conversation or domain imports

domain packages
  -> their own storage and services
  -> no Session imports
  -> no inference imports

runtimesetup
  -> the only composition layer allowed to import
     Session, inference, and domain services together
```

## Runtime setup composition layer

Create:

```text
internal/agentruntime-go/runtimesetup/
```

This package is the only place where Session, inference, and domain runtimes are stitched together.

Suggested structure:

```text
internal/agentruntime-go/runtimesetup/
  setup.go
  composition/
    resolver.go
    types.go
    diagnostics.go
  adapters/
    model/
      resolver.go
      executor.go
    agent/
      resolver.go
    workspace/
      resolver.go
    tool/
      resolver.go
      executor.go
    skill/
      resolver.go
      session_controller.go
      executor.go
    mcp/
      resolver.go
      executor.go
    attachment/
      preparer.go
  wails/
    session_runtime.go
    session_store.go
```

The exact names can vary, but the dependency rule cannot vary.

### Runtime setup responsibilities

`runtimesetup` owns:

- Construction of all required runtime ports
- Dependency validation at initialization
- Domain adapter wiring
- Agent and Workspace resolution composition
- Conversion from Session intent to prepared model operation
- Conversion from prepared model operation to thin inference request
- Provider runtime assembly
- SessionStore assembly
- Wails command and query wrappers
- Panic-safe background work integration

It must not become a second runtime state machine. Session remains the coordinator.

## Session state model

The physical state roots remain:

| Root    | Responsibility                                                                     |
| ------- | ---------------------------------------------------------------------------------- |
| Session | Conversation and task context, default intent, transcript, associated Runs         |
| Run     | One accepted unit of work, operations, waits, controls, results, execution history |

No top-level Step, ToolAttempt, AgentInvocation, WorkspaceInvocation, or GraphNode entity is introduced.

Those remain Run-owned records.

### Session configuration intent

Session configuration stores intent rather than provider-ready output.

Conceptually:

```go
type SessionConfigurationIntent struct {
    Agent     *ArtifactSelection
    Workspace *WorkspaceSelectionIntent

    Model ModelSelectionIntent

    ModelOverrides ModelOverrideIntent

    InstructionOverrides InstructionOverrideSet
    ToolOverrides        ToolOverrideSet
    SkillOverrides       SkillOverrideSet
    MCPOverrides         MCPOverrideSet
    WorkspaceOverrides   WorkspaceOverrideSet

    Metadata json.RawMessage
}
```

The actual implementation must use typed structs rather than a generic, unvalidated metadata blob.

`Metadata` remains frontend-owned presentation and provenance metadata and must not drive execution.

### Artifact selection pin

All backend-resolved selections use stable identity.

Conceptually:

```go
type ArtifactSelection struct {
    RootID           string
    ArtifactID       string
    ExpectedRevision uint64
    ExpectedDigest   string
}
```

The backend resolver uses current authoritative artifact state and records:

- Selected identity
- Selected revision or digest
- Used revision or digest
- Changed or unavailable state
- Whether the current source was explicitly adopted

The backend must not silently replace a pinned selected resource with a newly discovered resource merely because the Workspace or Agent changed.

If an expected revision or digest differs from the authoritative source, preparation must preserve the selected identity and produce a typed stale-source diagnostic. The backend must not adopt current contents until an explicit configuration command repins the selection or explicitly acknowledges use of the current source.

Runtime setup adapters must call underlying domain services directly. They must not invoke Wails wrappers internally.

## Owner and override model

### Sources

Configuration can originate from:

```text
Platform policy
Model Artifact defaults
Agent baseline
Workspace baseline
Session explicit overrides
Run or continuation explicit overrides
User-authored current-turn input
```

Each effective item preserves origin.

Examples:

```text
Tool inherited from Agent
Skill inherited from Workspace
MCP server inherited from Workspace
System instruction inherited from Agent
User-added Tool
User-removed Workspace Context
User-overridden model temperature
```

### Override modes

Every selectable collection uses deterministic canonical-key overrides.

Supported states:

```text
inherit
add
remove
replace
```

Not every collection needs every mode.

| Configuration item        | Supported override modes                                |
| ------------------------- | ------------------------------------------------------- |
| Model reference           | inherit, replace                                        |
| Model defaults            | inherit, override, clear                                |
| Agent                     | inherit, replace, clear                                 |
| Workspace                 | inherit, replace, clear                                 |
| System instruction source | inherit, add, remove                                    |
| Tool                      | inherit, add, remove                                    |
| Skill                     | inherit, add, remove                                    |
| Active Skill              | inherit, activate, deactivate                           |
| MCP server                | inherit, add, remove                                    |
| MCP tool exposure         | inherit, replace                                        |
| Workspace Context         | inherit, add, remove                                    |
| Workspace Skill           | inherit, add, remove                                    |
| Workspace MCP declaration | inherit, add, remove                                    |
| Attachments               | add, remove                                             |
| User text                 | replace only through explicit edit-and-resend semantics |

The override key must be a canonical identity, never:

- Display name
- Array position
- Frontend object identity
- Transient provider name
- UI list index

### Deterministic precedence

The effective configuration precedence is:

```text
1. Hard platform and policy restrictions
2. Explicit Run or continuation overrides
3. Explicit Session overrides
4. User-added or user-removed collection overrides
5. Workspace baseline contributions
6. Agent baseline contributions
7. Model Artifact defaults
8. Provider SDK defaults
```

For collection values, ownership is additive unless the override says remove or replace.

For model selection:

```text
Run explicit Model override
  > Session explicit Model override
  > Agent Model
  > Session default Model
```

A Workspace must not silently replace the Model selected by an Agent or user.

For model request defaults:

```text
Provider defaults
  -> Model Artifact defaults
  -> Agent Model patch
  -> Session Model patch
  -> Run or continuation Model patch
```

Existing typed clear semantics, such as `RuntimeRequestPatch.Clear`, remain the basis for explicit removal.

### System prompt ordering

System instruction composition must be ordered and visible.

Recommended order:

```text
1. Host or platform policy
2. Model default prompt, if selected
3. Agent instruction sources
4. Workspace instruction Context
5. User-added instruction sources
6. Explicit Session or Run system-prompt override
```

Each source carries provenance and a stable source key.

The frontend must show which sources are active and their order.

The backend re-resolves source text at operation preparation time.

## Agent resolution

### Frontend role

Frontend:

- Shows Agent catalog
- Previews Agent starter recipe
- Displays inherited values
- Lets users create overrides
- Never treats frontend-prepared Agent output as execution authority

### Backend role

The backend Agent adapter:

1. Resolves the selected Agent Artifact.
2. Reads its current recipe.
3. Checks expected revision and digest.
4. Converts the Agent recipe into baseline contributions:
   - Model
   - Request defaults
   - Tool defaults
   - Skill defaults
   - MCP defaults
   - Instruction sources
   - Starter-text metadata
5. Applies Session and Run override sets.
6. Produces resolution diagnostics and provenance.

Starter text remains frontend-only authoring behavior. The backend records Agent starter provenance but never overwrites visible user text.

## Workspace resolution

### Frontend role

Frontend:

- Shows Workspace directories and effective Workspaces
- Previews Context, Skill, and MCP contributions
- Displays changed and unavailable diagnostics
- Lets users select, remove, add, or replace Workspace-derived items
- Shows source ownership
- Persists only Session intent and explicit overrides

### Backend role

The backend Workspace adapter:

1. Resolves the selected Workspace from the Workspace domain.
2. Resolves effective Context, Skills, and MCP declarations.
3. Applies canonical selection and override sets.
4. Composes:
   - System instruction contributions
   - User-side context contributions
   - Effective Workspace Skill references
   - Workspace MCP candidates
   - Diagnostics
   - Provenance
5. Produces a prepared Workspace contribution for the model operation.

### Workspace MCP rule

A Workspace-discovered MCP declaration is only a candidate.

It does not automatically:

- Connect a server
- Install a server
- Expose all tools
- Authorize invocation
- Provide credentials
- Bypass MCP policy
- Become visible to the model

Workspace selection and MCP selection remain separate, override-controlled decisions.

## Model operation preparation

### Distinct stages

The system has two distinct concepts:

| Stage                 | Purpose                                                                   |
| --------------------- | ------------------------------------------------------------------------- |
| Intent admission      | Accept typed Session or Run configuration and user input                  |
| Operation preparation | Resolve authoritative sources and construct one immutable model operation |

These are not duplicate validations.

### Intent admission

At the command boundary:

- Wails validates the incoming DTO once
- Session validates command, state, and revision once
- Attachment admission snapshots local references
- Intent and overrides are stored
- No provider-specific execution occurs under the Session lock

### Operation preparation

Before each model operation:

- Backend resolves Agent, Workspace, Model, Tool, Skill, and MCP sources once
- Backend applies overrides once
- Backend produces an immutable prepared operation
- Backend records a redacted resolution summary and provenance
- Backend invokes the thin inference runtime

No downstream helper re-validates the same Session request shape.

### Prepared model operation

Conceptually:

```go
type PreparedModelOperation struct {
    ProviderRequest PreparedProviderRequest

    Inputs      []InputUnion
    ToolChoices []ToolChoice

    ToolBindings []PreparedToolBinding
    MCPMappings  []MCPProviderToolMapping

    ResolutionSummary OperationResolutionSummary

    EffectiveSkillRefs []ArtifactReference
    SkillSessionID     string
}
```

The durable summary contains only safe semantic data:

- Selected and used Agent
- Selected and used Workspace
- Selected and used Model
- Source revisions and digests
- Included and excluded Context
- Effective Tools, Skills, and MCP capabilities
- Diagnostics
- Token and truncation information
- Binding identities

It must not persist:

- Credentials
- Provider API keys
- OAuth tokens
- Approval tokens
- Raw attachment bytes
- Private provider request headers
- Ephemeral runtime session handles

### Operation immutability

Once a model operation starts:

- Model and provider configuration are immutable
- Tool bindings are immutable
- Resolved Workspace and Agent contributions are immutable
- Request defaults are immutable
- Attachment snapshots are immutable
- Current user input is immutable

New overrides or continuation input apply only to the next model operation.

## Thin inference runtime

### Target role

The inference runtime becomes a provider execution component.

It owns:

- Provider registration
- Provider removal
- Provider credential installation
- Capability resolver invocation
- Provider SDK request execution
- Streaming callbacks
- Provider response conversion
- Provider debug transport

It accepts already-prepared content.

It must not resolve:

- Artifact Models
- Agents
- Workspaces
- Tools
- Skills
- MCP selections
- Attachments
- Session history policy
- Session or Run metadata

### Target request shape

The thin inference API receives something equivalent to:

```go
type PreparedCompletionRequest struct {
    Provider  ResolvedProviderConfiguration
    Model     inferenceSpec.ModelParam
    Inputs    []inferenceSpec.InputUnion
    Tools     []inferenceSpec.ToolChoice

    OnText     func(string) error
    OnThinking func(string) error
}
```

The exact private provider configuration may remain non-serializable and setup-owned.

The request must not contain:

```text
ArtifactRef
WorkspaceSelection
AgentSelection
ToolSelection
SkillRef selection intent
MCPConversationContext
ConversationMessage
SessionID
RunID
```

Those belong to runtime-setup adapters and Session orchestration.

### Existing code movement

The following semantic logic must leave the inference package and move into `runtimesetup` adapters:

- Current Session inference adapter
- Workspace hydration
- MCP inference bridge hydration
- Tool choice hydration from Artifact Tools
- Skill prompt and tool hydration
- Artifact reference conversion
- Conversation and Session input conversion
- Attachment materialization integration

The inference package may retain provider lifecycle APIs needed by Model provider publication.

## Session orchestration lifecycle

```text
Create Session
  -> configure intent, Agent, Workspace, and overrides
  -> StartRun
  -> prepare first model operation
  -> execute model
  -> record output and tool proposals
  -> execute or wait for tool calls
  -> consume results
  -> resolve next operation
  -> terminal result
```

### Initial submission

`StartRun`:

1. Validates command and revisions at entry.
2. Snapshots user attachments.
3. Captures current Session configuration intent.
4. Creates a Run.
5. Prepares the first operation through runtime setup.
6. Accepts only after required preparation succeeds.
7. Dispatches model work.

Frontend clears the submitted draft only after receiving an accepted receipt.

### Tool calls

Tool proposals are created only from prepared bindings.

A tool call must be bound as one of:

```text
Go tool
MCP tool
Skill Runtime tool
Provider-managed tool
Unresolved call
```

No name-prefix inference is allowed as an authorization mechanism.

Tool execution remains Session-owned.

### Approval and waits

Waits are Run-owned records.

Frontend renders and resolves them.

Approval tokens, provider state, and executor private references remain backend-private and non-durable.

A wait resolution is scoped to:

- Exact tool call
- Canonical tool binding
- Arguments
- Model operation
- Authorization or approval state
- Runtime generation where applicable

### Continuation

Continuation input carries:

- User text
- User attachments
- Agent and Workspace override changes
- Model override changes
- MCP App context updates
- Tool-result selection
- Explicit continuation request

The latest accepted continuation revision is consumed exactly once.

### Automation

Session driver owns:

- Automatic tool execution
- Automatic continuation
- Holds
- Pause and resume
- Cancellation

Frontend only sends commands.

The frontend must never contain a second auto-execution drainer.

## Skills

### Effective Skill set

Effective Skills are computed from:

```text
Agent baseline Skills
+ Workspace baseline instruction Skills
+ Session additions
+ Run additions
- Session removals
- Run removals
```

Active Skills must always be a subset of enabled Skills.

### Skill Runtime ownership

Session owns the Run-associated Skill Runtime session.

The Skill adapter owns:

- Artifact Skill resolution
- Actual Skill definitions
- Session allow-list generation
- Active Skill state
- Built-in Skill tool preparation
- Skill lifecycle tool execution

Inference receives only already-prepared Skill prompt and tool choices.

## MCP

### MCP preparation

The MCP adapter resolves selected effective MCP state into:

- Instructions
- Selected tool choices
- Selected resource content
- Selected templates
- Selected prompts
- Provider tool mappings
- Invocation bindings
- Diagnostics

Inference receives only resulting prepared text, inputs, and tool choices.

### MCP invocation

Session tool execution routes MCP calls through the MCP adapter.

The MCP adapter remains authoritative for:

- Connection generation
- Discovery digest
- Policy
- Approval
- Authorization token handling
- Stale-reference detection
- Tool invocation
- App policy

### MCP Apps

Each MCP App instance must include:

```text
Session ID
Run ID
Originating operation ID
Tool call ID
Server ID
View instance ID
```

No App request may route through the currently selected frontend tab.

MCP App actions become:

| App action           | Target                                            |
| -------------------- | ------------------------------------------------- |
| Tool invocation      | Explicit Session and Run-scoped backend operation |
| Resource read        | Explicit server and view-scoped backend operation |
| UI draft proposal    | Frontend draft action                             |
| Model context update | Staged continuation input                         |
| External URL         | Frontend user-approved browser action             |

## Attachments

Attachments are admitted once at Session command boundaries.

Backend attachment preparation:

- Validates canonical local or URL reference shape
- Creates immutable local file snapshots
- Expands directories once
- Stores canonical attachment mode
- Records bounded overflow information
- Preserves user-selected mode only if valid for canonical type
- Never stores hydrated bytes in Session state

Historical replay must not silently read changed local files as if they were original historical inputs.

If source content is unavailable after recovery, the model receives a safe display and provenance representation rather than silently substituted bytes.

## SessionStore

### Storage model

SessionStore stores semantic Session and Run projections.

It stores:

- Session configuration intent
- Agent and Workspace selections and overrides
- Visible transcript
- Semantic operation history
- Tool calls and results
- Waits
- Durable resolution summaries
- Durable diagnostics
- Run outcomes
- Session and Run revisions

It does not store:

- Raw provider credentials
- Authorization tokens
- Hydrated attachment bytes
- OAuth state
- App iframe payloads
- Provider transport debug payloads unless explicitly safe and redacted

### Breaking storage policy

This branch uses a new Session storage schema and version.

It does not:

- Read legacy conversations
- Import old conversations
- Mutate old conversation files
- Retain old frontend persistence mappers
- Attempt heuristic conversion

### Fast reads

Read paths must be fast:

- Runtime snapshots come from in-memory state
- No Agent, Workspace, Model, or MCP resolution occurs on snapshot reads
- No provider calls occur on reads
- No filesystem attachment scan occurs on reads
- SessionStore listing and search use index or projection data
- No persistence I/O occurs under Session coordinator locks
- Event delivery is notification-only
- Missed events recover through snapshots

## Recovery

Recovery restores durable semantic state only.

| Previous state                         | Recovery behavior                                                      |
| -------------------------------------- | ---------------------------------------------------------------------- |
| Idle or terminal Session               | Restore normally                                                       |
| Waiting approval or input              | Restore waiting state                                                  |
| Paused Run                             | Restore paused state                                                   |
| Queued or running model operation      | Mark interrupted and require explicit recovery decision                |
| Running tool attempt                   | Mark uncertain and do not replay                                       |
| Completed selected tool result         | Reuse result                                                           |
| Expired or private authorization state | Mark unavailable or invalid and require a new flow                     |
| Changed Agent or Workspace source      | Show changed or unavailable diagnostics and do not silently substitute |

No external tool work is replayed blindly.

## Validation and dependency rules

### Validation boundaries

Validation occurs once per trust boundary.

Allowed validation boundaries:

1. Wails or public request entry
2. Runtime construction and setup
3. Session restore from persistence
4. Domain resolution of externally stored artifacts
5. Third-party, provider, or MCP response adaptation
6. Exact authorization and freshness checks immediately before external side effects

After a value is normalized at its boundary, internal helpers must trust it.

Do not repeat the following throughout internal helpers:

```text
Validate()
nil checks
empty checks
context checks
schema checks
artifact identity checks
```

### Dependency rules

All required services are validated at initialization.

Runtime code must not repeatedly do:

```go
if dependency == nil
if service == nil
if resolver == nil
```

inside normal execution paths.

If a host intentionally wants disabled behavior, it must inject a concrete implementation at setup time. The runtime does not silently install fallback no-op services.

No built-in generic no-op provider is added in this branch.

### Context rules

Wails and public command entry validates context ownership and lifecycle.

Internal code:

- Receives a valid context
- Does not repeatedly check for nil context
- Does not call `ctx.Err()` in every helper
- Honors cancellation at actual I/O, dispatch, timer, and long-running work boundaries
- Does not use context checks as ordinary input validation

### Public failure contract

Public commands and queries must fail predictably through typed errors, typed failure receipts, or explicit rejected command results.

External adaptation boundaries, including persisted data, provider responses, MCP responses, and domain-loaded data, must use checked conversions and checked type assertions.

Stale, missing, malformed, unauthorized, or unavailable external resources must become visible diagnostics, waits, or typed operation failures. They must not panic, silently disappear, or be silently replaced.

## Crash safety and background work

Every background goroutine must be panic-safe.

Rules:

1. No raw background `go func()` launch exists in runtime code.
2. All background work uses one common safe-launch utility.
3. The safe launcher:
   - Recovers panic
   - Captures stack information
   - Emits a diagnostic
   - Invokes operation-specific failure handling where applicable
   - Does not leave a Run permanently running after a panic
4. A panic in any of the following must not crash the application process:
   - Model dispatch
   - Tool dispatch
   - Wait resolution
   - Checkpoint dispatch
   - MCP refresh
   - Skill-session cleanup
   - Storage or index maintenance
5. A panic affecting active Session work must transition the relevant operation or Run into a visible failed or recovery-required state.

The common utility should live in a neutral shared runtime utility package or be exposed by Session without forcing inference to import Session semantics.

Inference must not import Session merely to launch a safe goroutine.

## Frontend architecture

### Session Runtime client

Frontend uses only:

- Session commands
- Session and Run snapshot queries
- Session runtime events
- SessionStore read, search, and export APIs

### Required command and query surface

The Session Runtime Wails surface must provide typed command or query contracts equivalent to:

```text
CreateSession
UpdateSessionConfiguration
StartRun
UpdateContinuationInput
ContinueRun
ExecuteCalls
RetryCall
SkipCall
CompleteCallAsError
ResolveWait
SetAutomationHold
PauseRun
ResumeRun
CancelRun
EditAndResend
GetSessionSnapshot
GetRunSnapshot
SessionStore query, search, and export APIs
```

The exact Go, Wails, and frontend names may differ only where repository naming conventions require it. The behavior and ownership semantics must remain equivalent.

For chat execution, frontend no longer directly calls:

```text
completionAPI.fetchCompletion
toolManagementAPI.invokeGoToolTarget
mcpManagementAPI.invokeMCPTool
skillManagementAPI.createArtifactSkillSession
skillManagementAPI.invokeSkillTool
```

Those remain management and domain APIs rather than chat-loop orchestration APIs.

### Event model

Frontend subscribes to:

```text
session-runtime:event
```

Events are notification signals.

Frontend maintains a snapshot-backed local projection:

```text
event received
  -> inspect sequence
  -> fetch Session or Run snapshot if needed
  -> update UI
```

A missed event never causes lost execution state.

### Agent and Workspace UI

Frontend renders:

```text
Agent baseline
Workspace baseline
User override
Effective result
Unavailable or changed diagnostic
```

For each effective item, the user can see:

- Source
- Canonical identity
- Inherited, add, remove, or replace state
- Final effective status
- Actual backend-used revision or digest after preparation

The frontend preview is useful but not authoritative.

### Composer behavior after migration

Composer retains:

- Local draft
- Attachment UI
- Menu state
- Focus
- Selection
- Rich rendering
- Temporary editor state

Composer sends command intent.

Composer does not:

- Start completion requests
- Run tools
- Schedule auto-execution
- Decide continuation
- Create Skill sessions
- Own approval promises
- Persist execution history directly

## Status

### Backend Session-owned execution delivery

- [x] Session configuration uses typed Agent, Workspace, Model, Tool, Skill,
      MCP, instruction, model-default, and App-context intent.
- [x] Session configuration uses canonical identity keys instead of list order,
      display labels, or frontend object identity.
- [x] Agent and Workspace are independently resolved in backend runtime setup.
- [x] Backend operation preparation captures immutable source composition,
      Tool bindings, Skill selections, MCP mappings, attachments, and provenance.
- [x] Initial Runs prepare authoritative execution before acceptance.
- [x] Continuation Runs prepare authoritative execution before provider calls.
- [x] Inference receives only provider-ready configuration, inputs, and Tool
      choices.
- [x] Inference has no Session, Agent, Workspace, Tool, Skill, MCP, Artifact,
      attachment, or conversation-composition imports.
- [x] Session Runtime remains the only accepted-command execution coordinator.
- [x] Skill session creation and Tool execution are Session-owned.
- [x] MCP approval resolution and invocation are Session-owned.
- [x] Session-scoped App instances carry Session, Run, operation, Tool-call,
      server, and view identities.
- [x] App Tool calls route through Session Tool-call records and normal MCP
      policy and approval enforcement.
- [x] App context updates stage continuation input through Session commands.
- [x] Legacy direct Wails completion, Tool invocation, Skill runtime session,
      and MCP Tool invocation ownership has been removed from chat execution.
- [x] Legacy Conversation collection runtime has been removed from backend
      startup and shutdown.

### Present now

| Area                                   | Current status                                     |
| -------------------------------------- | -------------------------------------------------- |
| Session and Run core state model       | Present                                            |
| Sequential chat driver                 | Present                                            |
| Model operation records                | Present                                            |
| Tool call, attempt, and result model   | Present                                            |
| Wait model                             | Present                                            |
| Pause, cancel, and limits              | Present                                            |
| Checkpoint and recovery framework      | Present                                            |
| SessionStore foundation                | Present                                            |
| Wails Session Runtime wrapper          | Present                                            |
| Attachment materializer                | Present                                            |
| MCP runtime and domain                 | Present                                            |
| Skill runtime and domain               | Present                                            |
| Agent preview and preparation APIs     | Present                                            |
| Workspace preview and planning APIs    | Present                                            |
| Frontend Agent picker                  | Present                                            |
| Frontend Workspace picker              | Present                                            |
| Legacy direct frontend completion path | Historical only; removed from backend chat surface |

### Historical components that needed movement or refactoring

The backend items in this inventory are now delivered as recorded below.
Frontend migration remains separately tracked.

| Area                             | Current state              | Target                  |
| -------------------------------- | -------------------------- | ----------------------- |
| `ProviderSetAPI.FetchCompletion` | Provider-only              | Delivered               |
| Workspace hydration              | Runtime setup adapter      | Delivered               |
| MCP inference hydration          | Runtime setup adapter      | Delivered               |
| Tool choice hydration            | Runtime setup adapter      | Delivered               |
| Skill prompt and tool hydration  | Runtime setup adapter      | Delivered               |
| Session inference adapter        | Runtime setup coordinator  | Delivered               |
| Frontend tool scheduler          | Frontend migration pending | Remove for Session chat |
| Frontend auto-exec drainer       | Frontend migration pending | Remove for Session chat |
| Frontend Skill session lifecycle | Frontend migration pending | Session-owned           |
| Frontend MCP approval lifecycle  | Frontend migration pending | Session wait-owned      |
| Conversation persistence         | SessionStore projection    | Delivered               |

### Historical gap inventory and current status

| Area                                      | Current status                                |
| ----------------------------------------- | --------------------------------------------- |
| Typed Agent and Workspace intent          | Delivered                                     |
| Typed owner and override model            | Delivered                                     |
| Backend composition resolver              | Delivered                                     |
| Thin inference request API                | Delivered                                     |
| Runtime setup package                     | Delivered                                     |
| Prepared model-operation pipeline         | Delivered                                     |
| Durable resolution summary and provenance | Delivered                                     |
| Frontend Session client                   | Pending                                       |
| Frontend owner and override presentation  | Pending                                       |
| Session-scoped MCP App routing            | Delivered                                     |
| Breaking SessionStore schema              | Delivered                                     |
| Removal of old frontend direct execution  | Backend delivered; frontend migration pending |

### Backend background-execution delivery

- [x] Common neutral panic-safe background launcher delivered.
- [x] Supplied Session, MCP, SessionStore-maintenance, and Wails-event launches use the common utility.
- [x] Panicked Session model, Tool, and wait work has operation-specific settlement.
- [x] Panicked external Tool attempts are recorded as uncertain without fabricated selected results.
- [x] Stale Skill-session cleanup remains inside tracked Session dispatch work.
- [x] Session snapshots expose a durable event watermark independent of Session revision.
- [x] Focused launcher and Session panic-settlement tests delivered.
- [x] Authoritative runtime setup, typed intent, immutable preparation, and thin inference complete.
- [x] Breaking SessionStore schema, Wails execution removal, and Session-scoped MCP App routing complete.
- [ ] Frontend Session command/snapshot/event migration complete.

### Backend inference and persistence delivery

- [x] Provider-only inference implementation delivered.
- [x] Domain-aware inference hydration and Session adapter removed from inference.
- [x] Model, Skill, and MCP preparation contracts relocated into runtimesetup.
- [x] Breaking SessionStore envelope and sessions_v1 namespace delivered.
- [x] Durable Session deletion barriers delivered.
- [x] Checked semantic projections exclude private authorization, hydrated inputs, Skill handles, render payloads, and unfiltered runtime metadata.
- [x] Session listing and search use derived index records.
- [x] FTS maintenance uses the common panic-safe launcher and existing filesystem utilities.
- [x] Domain-runtime implementation sources adapted to relocated contracts.
- [x] Authoritative Agent/Workspace composition and typed override resolution complete.
- [x] Immutable operation preparation and initial-admission integration complete.
- [x] Application/Wails assembly and legacy execution removal complete.
- [x] Session-scoped MCP App routing complete.
- [ ] Frontend Session migration complete.

### Backend runtime-setup delivery

- [x] `internal/agentruntime-go/runtimesetup` is the only Session, inference, and domain composition root.
- [x] Inference receives only provider-ready completion data.
- [x] Domain-aware operation composition lives in runtime-setup adapters.
- [x] Typed intent, overrides, canonical identities, provenance, diagnostics, and immutable prepared operations delivered.
- [x] Session Runtime remains the only execution coordinator after command acceptance.
- [x] Session-scoped MCP App routing delivered.
- [x] Explicit Session Tool, Skill, and MCP Artifact pins are checked against authoritative current Artifact state before use.
- [x] Prepared MCP resolution summaries preserve the effective source owner and backend-used Artifact provenance.
- [x] Attachment admission keeps backend-derived supported modes authoritative.
- [x] Direct chat completion and direct Tool, Skill, and MCP execution removed from Session orchestration.
- [x] Session creation and Session-configuration updates use the same
      authoritative typed-intent preparation boundary as Run admission.
- [x] Prepared Go Tool bindings retain the exact resolved Tool Artifact
      identity and reject changed implementations at side-effect time.
- [x] Run-owned active Skill state survives continuation preparation when the
      effective Skill allow-list is unchanged.
- [x] Explicit MCP Tool, resource, template, and prompt omissions become
      resolution diagnostics rather than silently disappearing.
- [x] Runtime event unsubscribe is race-safe, and wait expiry cancels an
      in-flight wait-resolution dispatch.

## Definition of done

The branch is complete when all of the following are true:

1. A new chat can be created solely through Session commands.
2. Frontend Agent and Workspace selectors produce typed intent and deterministic overrides.
3. Backend independently resolves Agent and Workspace and produces authoritative effective state.
4. Effective Model, prompts, Tools, Skills, MCP, Workspace Context, and attachments are prepared before inference.
5. Thin inference runtime receives only prepared provider-ready data.
6. No chat path calls old frontend direct completion, tool, Skill, or MCP execution APIs.
7. No frontend auto-execution drainer remains for Session chats.
8. Tool calls are bound only from prepared backend bindings.
9. Session snapshots fully drive frontend rendering after missed events or reloads.
10. SessionStore persists new semantic Session and Run state without legacy migration.
11. Recovery never blindly replays interrupted external work.
12. All runtime background goroutines are panic-safe.
13. No domain package imports Session or inference semantics.
14. Inference package has no Agent, Workspace, Session, MCP, Skill, or Tool composition semantics.
15. Query and read paths perform no domain resolution or provider I/O.
16. No required capability remains implemented as a TODO, stub, fallback, compatibility bridge, or frontend-owned execution path.
