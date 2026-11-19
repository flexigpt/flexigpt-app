# Session Runtime v3 and Workflow Runtime v3 High-Level Design

## Execute durable conversations and Workflow programs

Session Runtime v3 is the durable execution host for:

- Normal conversation.
- Agent loops.
- Direct Model calls.
- Workflow executions.
- Tool execution.
- MCP interaction.
- Skill interaction.
- Service and Activity bindings.
- Waits and human work.
- Triggers.
- Effects, reconciliation, and compensation.
- Recovery and cancellation.

Workflow Runtime v3 is not a second persistence system or scheduler.

Workflow Runtime v3 consists of:

- A setup-owned Workflow execution preparation path.
- The Workflow ProgramFrame interpreter in Session Runtime v3.
- Setup-owned target adapters and domain binders.

Workflow Artifact planning and setup-owned execution preparation complete before
Session Runtime v3 receives a runtime Program snapshot. For Workflow work,
Session Runtime receives only a typed `ProgramSnapshot` and immutable runtime
pins. It does not parse Workflow source, resolve an Artifact plan, retain an
Artifact Store handle, or invoke Workflow execution preparation.

    Workflow Artifact contract
        ->
    Artifact admission
        ->
    Read-only Workflow plan and static Artifact closure
        ->
    Workflow execution preparation in setup
        ->
    Runtime Program snapshot
        ->
    Session Runtime v3
        ->
    Setup-owned domain binders
        ->
    Model, Agent, Tool, MCP, Skill, Workspace, Service, Activity

## Current implementation cycle constraints

This HLD and its implementation status are work in progress.

No version identifier may be bumped by this implementation cycle. This includes
Workflow `apiVersion`, ProgramFrame payload and interpreter versions, Artifact
and Store schema versions, package or module versions, and public API versions.

This implementation cycle excludes:

- Tests and all validation work.
- Frontend integration.
- SessionV2 deletion, cleanup, migration, and stored-session conversion.
- Version bumps, release work, and release notes.
- Cost-budget metering. This capability is deferred and must not be implemented in this cycle.

SessionV2 functional replacement gaps remain tracked when they prevent Session
v3 from providing required backend behavior. An implementer must change an
`Not implemented` or `Partial` status to `Implemented` only after the complete listed
behavior exists in source. A status change alone is not completion.

The requested implementation is one substantial end-to-end functional delivery,
not a sequence of small status-only edits. No phases or version changes are
part of this handoff. The open functional register below is the actionable
completion contract. Existing components are not evidence that every composed
execution path is complete.

## Functional-analysis guardrails for this WIP cycle

This HLD tracks missing behavior and remaining functional work only.

- Artifact and attachment admission own source and input admission behavior.
- Do not add Runtime or setup work merely to repeat admission checks.
- Do not create work for strictness, malformed-input handling, or validation-only changes.
- Do not create tests or validation reports in this cycle.
- A Session is a reproducible semantic checkpoint, not a serialization of live Runtime state.
- Durable Session facts can include completed semantic messages, selected Artifact facts,
  completed Tool and model outcomes, and approved next-message context needed to reproduce behavior.
- Durable Session facts must not include live clients, timers, cancellation functions, private bindings,
  credentials, OAuth tokens, Skill Runtime session IDs, callbacks, or partial in-flight loads.
- All executable Model, Agent, Tool, Skill, MCP, Service, Activity, and Workflow capabilities in scope
  are Artifact-backed or registered binding-backed.
- `session.DirectCapabilitySelection` is a legacy SessionV2 compatibility
  representation. It is not a SessionV3 replacement requirement.
- Current application registration supplies no direct capability provider.
  Current Tool Store mapping and resolution accept only Artifact-backed Tool
  targets. Consequently, no fresh normal user, Agent, built-in Tool, SDK Tool,
  MCP Tool, or Skill Tool path can create a usable direct Tool target.
- Built-in Go Tools and provider SDK Tools are Tool Artifacts. MCP Tools are
  discovered operations of selected MCP Server Artifacts. Skill lifecycle
  Tools are supplied by selected Skill Runtime sessions. None of these are
  direct capability targets.
- Do not add direct Tool, Skill, Model, Agent, MCP, Service, Activity, or Workflow capability paths.
- Inline Text remains content or declaration data. It is not a direct executable capability.

At eventual SessionV2 retirement, the legacy direct Tool compatibility fields
can be removed without a direct-target migration or translation. No current
implementation task is created for that future retirement because the supplied
application has no active fresh-user producer for a direct Tool target.

## Separate Artifact planning from execution preparation

Workflow Artifact planning owns static source facts.

It returns:

- The admitted Workflow declaration.
- The root Workflow Artifact identity.
- A complete static Artifact closure.
- Static dependency edges.
- Callable target facts.
- Requirement facts.
- Unsupported direct-composition issue facts.
- Source inventory facts for calls, contexts, ports, retries, effects, blocks, and triggers.

Workflow Artifact planning does not return:

- ProgramFrames.
- WorkItems.
- Attempts.
- Waits.
- Timers.
- Runtime semantic bindings.
- Provider-ready requests.
- Credentials.
- MCP sessions.
- Approval tokens.
- Native handles.
- Callbacks.
- Runtime-private state.

Workflow execution preparation is setup-owned.

It performs:

- Invocation input validation.
- Typed Workflow source to runtime program conversion.
- Runtime Artifact pin construction from the static closure.
- Current binding registry checks.
- Current target transport checks.
- Current domain readiness checks.
- Runtime interpreter selection.
- Semantic binding preparation.

Execution preparation does not:

- Re-admit Workflow source.
- Reparse Workflow JSON.
- Rebuild Artifact composition.
- Re-resolve nested Workflow source relationships.
- Recreate a second trigger or retry scheduler.
- Mutate Session state.

## Use one durable Session root

A Session is the only durable root.

    Session
    ├── Defaults
    ├── Timeline activities
    ├── Executions
    │   ├── ProgramFrames
    │   ├── WorkItems
    │   ├── Attempts
    │   ├── Waits
    │   ├── Effects
    │   ├── Signals
    │   ├── Channels
    │   ├── Resource leases and rate windows
    │   ├── Child Execution links
    │   └── Outcome
    ├── Trigger registrations
    ├── Conversation ownership
    ├── Command records
    └── Event sequence

A durable Session document contains semantic facts only.

A durable Session document does not contain:

- Provider clients.
- HTTP clients.
- MCP client sessions.
- Artifact Store handles.
- Native file handles.
- Native paths used as live handles.
- Credentials.
- OAuth tokens.
- Approval tokens.
- Callbacks.
- Timers.
- Context cancellation functions.
- Compiled CEL programs.
- Compiled JSON Schema programs.
- Domain-private execution bindings.

Normal conversation is Session-scoped. `Session.ID` is the only durable
normal-conversation identity.

`ConversationOwnership.ConversationID` is a Workflow handoff semantic key.
It does not create another Session root, another normal conversation identity,
or a Message, Continuation, or Execution identity field. Normal conversation
routing consults only the ownership record whose conversation key is
`string(Session.ID)`. A Workflow can retain other ownership records for its
own orchestration semantics, but they do not create frontend-facing
conversations in this cycle.

## Persist commands before external work

Every mutating command has a command identity.

The runtime:

- Canonicalizes the command payload.
- Computes a command fingerprint.
- Returns the prior receipt for an identical command identity and payload.
- Rejects a command identity reused with a different payload.
- Persists accepted command receipts.
- Persists accepted Session deletion receipts in tombstones.

The command flow is:

    Client command
        ->
    Runtime idempotency and revision checks
        ->
    Durable state mutation
        ->
    Session checkpoint
        ->
    Command receipt
        ->
    Asynchronous runtime work

The runtime must not start new external work after a checkpoint failure.

## Dispatch WorkItems through a durable lifecycle

A WorkItem has one durable lifecycle.

    Ready
        -> ResourceWaiting
        -> Ready
        -> Binding
        -> Waiting
        -> Ready
        -> Dispatched
        -> Completed
        -> Failed
        -> RetryWaiting
        -> Ready
        -> Uncertain
        -> Reconciling
        -> Completed, Failed, or Uncertain
        -> Cancelled
        -> Blocked

The runtime performs these operations:

- Persist WorkItem intent.
- Stage authorized port inputs.
- Acquire resource claims.
- Persist semantic binding facts.
- Persist an Attempt before external invocation.
- Invoke the setup-owned domain binder.
- Persist output, receipt, error, cancellation result, or uncertainty.
- Deliver confirmed output ports.
- Advance the owning ProgramFrame.
- Wake completion waits.
- Update Execution state.
- Checkpoint the Session.

The runtime owns WorkItem state. A domain binder owns external execution.

## Execute normal conversation in ProgramFrames

Normal conversation uses long-lived conversation ProgramFrames.

The normal conversation flow is:

    SubmitMessage
        ->
    Setup admits attachment snapshots
        ->
    Runtime persists user Activity
        ->
    Conversation ProgramFrame creates LLM WorkItem
        ->
    Setup binds Model, Agent, Workspace, Tool, Skill, MCP, and attachments
        ->
    Runtime persists semantic binding and Attempt
        ->
    Provider executes
        ->
    Runtime persists streamed output, final output, Tool calls, and usage
        ->
    Agent loop continues or waits

The built-in ProgramFrame interpreters are:

| Interpreter              | Purpose                              |
| ------------------------ | ------------------------------------ |
| `builtin.conversation@1` | Routes visible conversation activity |
| `builtin.agent-loop@1`   | Runs Agent Model and Tool cycles     |
| `builtin.single-model@1` | Runs one direct Model request        |
| `artifact.workflow.v1@1` | Runs typed Workflow behavior         |

The setup binder owns:

- Model resolution.
- Provider configuration.
- Agent capability resolution.
- Workspace contributions.
- Tool choice preparation.
- MCP context and Tool choice preparation.
- Skill context and Tool choice preparation.
- Attachment conversion to provider content.
- Provider output conversion.

The runtime owns:

- Activity persistence.
- WorkItem lifecycle.
- Streaming state.
- Tool continuation.
- Wait state.
- Execution state.
- Checkpointing.
- Projection state.

Every LLM WorkItem retains its effective invocation configuration and
capability facts. Tool WorkItems created from that LLM result retain the
immutable origin facts required by their target binders. A binder must not
replace those selected typed bindings or invocation facts with current Session
defaults when it evaluates or executes the generated Tool WorkItem.

## Admit replay-safe attachments before persistence

Attachment admission is setup-owned.

The V3 attachment path:

- Validates command-only attachment sources.
- Reads local files and images.
- Expands directories once.
- Fetches or snapshots URL content according to the selected mode.
- Produces immutable semantic attachments.
- Persists attachment content and digest rather than a live native path.

Session Runtime v3 stores attachment snapshots. Historical replay does not reopen a local native file.

## Prepare Workflow execution from a static Artifact plan

Workflow execution preparation receives:

- Workflow selection.
- Canonical invocation input.
- A read-only Workflow plan.
- A complete static Artifact closure.
- Current setup-domain adapter registry.

Workflow execution preparation performs:

- Root Workflow selection pin verification.
- Invocation input validation.
- Source-to-runtime conversion.
- Runtime Program snapshot creation.
- Runtime target descriptor construction.
- Requirement binding checks.
- Target transport checks.
- Runtime preparation fingerprint creation.

Workflow execution preparation must not:

- Re-run Artifact composition.
- Resolve nested Workflow declarations independently.
- Rebuild target relationships from `CapabilityPlan`.
- Revalidate source grammar.
- Create WorkItems.
- Start timers.
- Persist Session state.
- Be invoked by Session Runtime.

The runtime accepts only a typed `ProgramSnapshot`.

The application setup boundary invokes execution preparation before it submits
a Workflow start or trigger-registration command. Session Runtime persists and
interprets the prepared snapshot only.

## Execute Workflow control flow in ProgramFrames

The Workflow interpreter owns durable control-flow execution.

It supports:

- Ordered steps.
- Transform.
- Set.
- Scope.
- Choose.
- Parallel.
- Foreach.
- Repeat.
- State machines.
- Wait.
- Emit.
- Saga.
- Cancel.
- Return.
- Fail.
- Break.
- Continue.
- Catch.
- Finally.
- Retry boundaries.
- Reconciliation.
- Compensation.

The interpreter emits runtime transitions. It does not call a domain directly.

A runtime transition can contain:

- WorkItem intent.
- Child ProgramFrame intent.
- State update intent.
- Wait intent.
- Cancellation intent.
- Reconciliation settlement.
- Compensation settlement.
- Terminal ProgramFrame outcome.

### Persist callable operation facts

Execution preparation lowers each callable declaration to typed runtime intent.
Before target dispatch, Session Runtime persists:

- Prepared input and explicit context.
- Selected target identity.
- Resolved target-set membership.
- Effect key and retry identity.
- Deadline and port plans.
- Immutable Artifact pins and semantic binding facts.

A resolver-backed target set starts with a typed resolver WorkItem. The runtime
persists normalized membership before target selection. Retries, waits,
recovery, and resumption reuse the persisted membership and selected target.

A successful `start` call persists its completion handle and start receipt in
the parent step outcome. A handoff persists its conversation ownership facts
before its child ProgramFrame starts.

## Preserve state ownership and Workflow outcomes

A state-owning scope provides:

- State owner identity.
- State schema.
- State snapshot.

A typed `StateUpdateWork` applies a state update atomically.

Parallel and Foreach child frames receive isolated state.

A parent state changes only through a declared parent-state merge. The runtime persists that merge as a typed StateUpdate WorkItem.

Workflow outcomes retain:

- Step identity.
- Outcome state.
- Output.
- Selected target.
- Receipt.
- Error.
- Effect state.

Outcome states are:

- `succeeded`
- `failed`
- `skipped`
- `cancelled`
- `uncertain`

A skipped step has no output.

An uncertain step records that an external effect can have occurred without durable confirmation.

## Bind each target through its owning domain

| Workflow target          | Runtime representation                                             | Setup owner                    |
| ------------------------ | ------------------------------------------------------------------ | ------------------------------ |
| Model                    | Child single-model ProgramFrame                                    | Model and Session setup        |
| Agent                    | Child Agent-loop ProgramFrame                                      | Agent and Model setup          |
| Tool Artifact            | Tool WorkItem                                                      | Tool runtime                   |
| MCP Tool                 | Tool WorkItem                                                      | MCP runtime                    |
| Skill Artifact selection | Typed Model or Agent child plan and its Skill lifecycle Tool calls | Skill runtime                  |
| Provider SDK Tool        | Provider-managed Tool choice                                       | Model plan                     |
| Service                  | Service WorkItem                                                   | Registered Service adapter     |
| Activity                 | Activity WorkItem                                                  | Registered Activity adapter    |
| Nested Workflow          | Child Workflow ProgramFrame                                        | Workflow execution preparation |

A Skill Artifact is not a standalone Workflow callable target. A typed Model
or Agent invocation can select artifact-backed Skills through its typed
defaults. The resulting Skill lifecycle Tool calls belong to that Model or
Agent plan. Do not add a direct Skill target or a non-Artifact Skill path.

Standalone Tool selection is intentionally closed to built-in Tool Artifacts.
Those declarations can contain only Go Tool or provider SDK Tool
implementations. MCP Tools are discovered operations of selected MCP Server
Artifacts. Skill lifecycle Tools are supplied by selected Skill Runtime
sessions. Do not add arbitrary Tool Artifact creation, removal, direct Tool
capabilities, or non-Artifact Tool routing.

Workflow owns orchestration. It does not own native domain behavior.

A target adapter owns:

- Native request construction.
- Current domain readiness.
- Native result conversion.
- Domain-specific authorization.
- Domain-private state.

A target adapter must not use generic Workflow data as a replacement for a native domain request grammar.

## Convert Model and Agent Workflow calls before runtime execution

Workflow Model and Agent calls use explicit typed invocation data.

The current supported forms are:

- Explicit message list.
- Explicit instructions.
- Existing Session default overrides.
- Final output ports.
- Streamed output ports.
- Agent handoff.

The runtime must receive this data as typed runtime fields.

The runtime must not parse a generic Workflow JSON envelope to discover:

- Message structure.
- Instruction structure.
- Model defaults.
- Agent defaults.
- Provider request fields.

The setup converter maps supported Workflow forms into typed runtime invocation fields. The runtime then creates child Agent-loop or single-model ProgramFrames.

When a Model or Agent call accepts a final-output or streamed-output port, the
lowered child ProgramFrame contains the accepted port plans before child work
starts. A valid port mapping must not disappear between call lowering and child
frame creation.

Arbitrary Workflow JSON must never become fallback Model or Agent chat text.

## Enforce ports, channels, and target transport

The runtime owns:

- Port plans.
- Channel logs.
- Input cursors.
- Message ordering.
- Retention.
- Byte limits.
- Parent-child ownership.
- Stream ordering.
- Durable replay facts.

Each Workflow WorkItem and child ProgramFrame retains its planning-owned
declaration path separately from its runtime path. Channel permissions and
port authorization use the declaration path.

Setup owns target transport capability.

A target transport check must validate:

- Context support.
- Input port support.
- Final output port support.
- Streamed output port support.
- Port schema compatibility.
- Channel authorization.
- Current target capability.

Target transport matrix:

| Target family       | Workflow context                                         | Input ports                                | Final output port                    | Streamed output port |
| ------------------- | -------------------------------------------------------- | ------------------------------------------ | ------------------------------------ | -------------------- |
| Model               | Explicit `messages`, `instructions`, and `defaults` only | Unavailable                                | Supported                            | Supported            |
| Agent               | Explicit `messages`, `instructions`, and `defaults` only | Unavailable                                | Supported                            | Supported            |
| Nested Workflow     | Child ProgramFrame scope                                 | Supported                                  | Supported                            | Supported            |
| Port-aware Service  | Supported when the registered binding declares it        | Supported when the registered binding does | Supported                            | Supported            |
| Port-aware Activity | Supported when the registered binding declares it        | Supported when the registered binding does | Supported                            | Supported            |
| Go Tool             | Unavailable                                              | Unavailable                                | Supported after confirmed settlement | Unavailable          |
| MCP Tool            | Unavailable                                              | Unavailable                                | Supported after confirmed settlement | Unavailable          |
| Skill Tool          | Unavailable                                              | Unavailable                                | Supported after confirmed settlement | Unavailable          |
| Provider SDK Tool   | Typed Model-plan only                                    | Typed Model-plan only                      | Provider-managed                     | Provider-managed     |

Native target contracts remain authoritative:

- Go Tool arguments remain Go Tool function arguments.
- MCP Tool arguments remain MCP `tools/call` arguments.
- Skill Tool arguments remain Skill Tool arguments.
- Provider Tool choices remain inside the Model request plan.

Workflow metadata is not inserted into generic Tool context fields, Tool
argument envelopes, Model request envelopes, Agent request envelopes, native
context sidecars, MCP metadata, MCP extensions, Skill sidecars, provider
metadata, or Artifact-domain imports.

A final call output remains separate from a streamed port. A streamed port
contains ordered stream records only. If terminal stream signalling is needed,
it uses a distinct durable stream record.

A provider-managed Tool target is lowered only through a typed Model child
plan. It never becomes a local Tool WorkItem or a synthetic local result.

## Architecture ownership and duplication register

This register records cross-cutting ownership and duplication work. It is
separate from the functional completion ledger because one architecture issue
can affect several target families and Workflow operations.

| ID                        | Current status | Required end state before marking `Implemented`                                                                                                                                                                                        | Current source evidence                                                                                                                                                                                   |
| ------------------------- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ARC-CORE-001`            | Implemented    | Session Runtime v3 receives only runtime-neutral snapshots, pins, semantic bindings, and durable state. It imports no Artifact Store or setup package.                                                                                 | `internal/agentruntime-go/sessionv3/` has no Artifact Store or setup dependency.                                                                                                                          |
| `ARC-DIRECT-001`          | Implemented    | Workflow rejects direct composition for Model, Agent, Tool, MCP, Skill, Service, Activity, and Nested Workflow before ProgramSnapshot creation.                                                                                        | `workflow/closure.go` rejects restricted direct results. `workflowruntime/direct_targets.go` is removed. `sessionv3` has no direct target, plan, payload, semantic-binding, or Tool-origin fields.        |
| `ARC-TRANSPORT-001`       | Partial        | Preserve the prepared transport matrix and connect nested published ports to their actual child-local channels.                                                                                                                        | The no-published-port rejection and forward-only started wrapper exist. Published-channel transfer and immediate cross-Execution wakeup remain open in `WF-PORT-003`.                                     |
| `ARC-PROVIDER-001`        | Implemented    | A provider SDK Tool is represented only by an owning typed Model plan and never by a local Tool WorkItem or local Tool result.                                                                                                         | `toolruntime/v3.go` prepares `ProviderManagedToolChoice`; `workflow_program.go` lowers it through a single-model child ProgramFrame only.                                                                 |
| `ARC-ORIGIN-001`          | Implemented    | A Tool WorkItem emitted from LLM output retains immutable effective invocation and selected typed bindings. Target binders never fall back to current Session defaults.                                                                | `sessionv3/workflow_target_facts.go` stores `LLMToolOrigin.Plan`; `skillruntime/v3.go` consumes retained plan selections; Go and MCP binders use retained typed bindings.                                 |
| `ARC-SCHEDULER-001`       | Partial        | Keep one Runtime due-time owner and complete work launch, deadline arming, restored resource wakeup, and trigger-queue resumption.                                                                                                     | `due_time.go` is the common registry. `RT-TRIGGER-002`, `RT-LIMIT-002`, and `RT-CLEANUP-002` record missing composed behavior.                                                                            |
| `ARC-STATIC-001`          | Not applicable | Repeated-admission and source-check cleanup is not functional work in this cycle.                                                                                                                                                      | Artifact admission is sufficient for this cycle; no validation-only work is requested.                                                                                                                    |
| `ARC-COMPAT-001`          | Not applicable | Compatibility naming cleanup is not functional work in this cycle.                                                                                                                                                                     | `PreparationRequest`, `Admission`, and `WorkflowAdmissionRequest` coexist without a demonstrated missing behavior.                                                                                        |
| `ARC-V3-001`              | Partial        | Complete artifact-backed conversation configuration, controls, replay, and setup capability forwarding.                                                                                                                                | See `V3-CONFIG-002`, `V3-HISTORY-002`, `V3-CONTROL-002`, and `V3-SETUP-002`. Provider SDK Tool user configuration already exists and is not the remaining parity gap.                                     |
| `ARC-REPLAY-001`          | Partial        | Reproduce each message's retained semantic context and complete assistant/Tool rounds across isolation and Execution boundaries.                                                                                                       | Current input reconstruction omits historical injected context, isolated assistant rounds, and some cross-Execution Tool results. See `V3-HISTORY-002`.                                                   |
| `ARC-SKILL-001`           | Partial        | Reconstruct and update Skill state for the correct invocation owner without one parallel Agent replacing another's private session.                                                                                                    | Durable Skill results and hydration exist. Execution-keyed sessions and origin-state resets leave `DOMAIN-SKILL-002` open.                                                                                |
| `ARC-TOOL-SCOPE-001`      | Implemented    | Standalone Tool selection is limited to built-in Artifact-backed Go and provider SDK Tool declarations. Go Tools remain Tool Runtime work; provider SDK Tools remain typed Model-plan choices.                                         | `tool/api.go` intentionally requires the protected built-in Tool root. `toolcatalog/packages.go` creates both Go and SDK Tool Artifacts.                                                                  |
| `ARC-PROVIDER-CONFIG-001` | Implemented    | Provider SDK Tool configuration for built-in SDK Tool Artifacts is typed, immutable in the Model dispatch plan, and available to Workflow provider Tool lowering. An owning Model can derive from an explicit Model or selected Agent. | `ToolSelection.UserArguments`, `ToolDispatchPlan.ProviderConfiguration`, `ProviderManagedToolChoice.Configuration`, and `ResolveWorkflowAgentOwnerModel` retain typed configuration and owner Model pins. |
| `ARC-MCP-PREPARATION-001` | Implemented    | Workflow preparation resolves current MCP server readiness, discovered Tool operation, policy, and discovery facts before it emits a ProgramSnapshot. MCP Tools remain discovered MCP Server operations, not Tool Artifacts.           | `mcpruntime.WorkflowTargetPreparer` prepares the current operation. `WorkflowMCP` retains server, provider Tool, policy, and discovery facts; dispatch rejects a changed prepared target.                 |

The owning domain constructs native requests, resolves current domain state,
and converts native results. Session Runtime owns durable orchestration,
checkpointing, retries, waits, cancellation, and settlement. Workflow remains
an orchestration language and must not become a generic native-domain envelope.

| ID                    | Current status | Required end state                                                                                                                                                                                     | Current source evidence                                                                                                                                                                 |
| --------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ARC-INVOCATION-002`  | Implemented    | Setup lowers supported Workflow Model, Agent, and provider-managed Tool invocation forms into typed runtime fields. Core Runtime does not decode generic Workflow envelopes.                           | `workflowruntime/compiler_invocation.go` lowers supported forms into `WorkflowLLMInvocation`; Runtime evaluates typed templates only.                                                   |
| `ARC-CALL-POLICY-001` | Partial        | Apply one captured logical call policy consistently to local work, child work, retry, and cleanup.                                                                                                     | Child-call identity is frame-scoped, but retry capture, outer-attempt ordering, budgets, and ordinary child-call cleanup classification remain incomplete.                              |
| `ARC-PATH-001`        | Partial        | Preserve declaration identity in admitted descriptors and synthetic execution blocks.                                                                                                                  | Ordinary lowering retains paths. `startWorkflowRetry` constructs a synthetic root block without its declaration path.                                                                   |
| `ARC-V2-COEXIST-001`  | Not applicable | SessionV2 and SessionV3 coexistence removal is outside this cycle. New V3 work must not add a third orchestration path or restore direct capabilities.                                                 | `cmd/agentgo/app.go` composes both runtimes. SessionV2 cleanup and migration remain excluded.                                                                                           |
| `ARC-START-PORT-001`  | Implemented    | A started call has one authoritative port-delivery boundary. Final output is delivered once, streamed output remains ordered, and independent child execution routes to the true parent channel owner. | Forward-only wrapper frames retain routing facts but suppress duplicate final delivery. Recursive channel ownership resolves inherited parent port plans across independent Executions. |
| `ARC-NESTED-PORT-001` | Implemented    | A nested Workflow target with mapped ports must publish compatible root ports. A target with no published ports accepts no mapping.                                                                    | `ProgramFramePortBoundaryPublishedWorkflow` invokes `validateWorkflowFramePortBoundary`, which rejects a mapped target with no published ports.                                         |
| `ARC-DUP-MCP-001`     | Implemented    | V3 MCP content adaptation has one MCP-domain implementation. Session Runtime receives only converted semantic values.                                                                                  | `mcpruntime.MCPContentText` and `MCPContentsText` are now the shared V3 MCP conversion boundary.                                                                                        |
| `ARC-OWNERSHIP-002`   | Partial        | Domain owners prepare domain semantics through neutral execution contracts; Session and Workflow setup compose them.                                                                                   | Generic Session setup owns Agent and Workspace contribution behavior; concrete domain adapters import the Artifact-dependent `workflowruntime` package.                                 |
| `ARC-DUP-002`         | Partial        | Use one implementation for shared execution semantics and one authoritative immutable effective plan.                                                                                                  | Child scopes, value evaluation, root construction, cancellation, projections, and effective-plan copies contain actionable duplication. See the open register.                          |

### Workflow direct-capability and callable-capability decision

Workflow does not support direct capability targets. A direct composition
result for Model, Agent, Tool, MCP Tool, Skill Tool, Service, Activity, or
Nested Workflow is rejected during planning before a ProgramSnapshot exists.

Service and Activity are registered typed `WorkBinding` targets keyed by their
declared binding IDs. They are not Artifacts by default and are not direct
capabilities.

Inline Workflow literal values, explicit Model and Agent messages,
instructions, and defaults remain declaration data. They are not a direct
capability, Artifact pin, or generic native request envelope.

`CallableInterface.capabilities` has no owning vocabulary or runtime meaning.
The field is removed from Workflow v1 grammar. Because callable interface
objects are closed, contract admission rejects the former field once. Runtime
does not repeat this check.

Root Workflow `requirements` remain separate named Artifact-backed or
registered-binding prerequisites. They are never callable target selections.

| Cleanup task                             | Status      | Completion evidence                                                                                                                                                                   |
| ---------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Reject direct Workflow composition       | Implemented | `internal/llmartifactory-go/workflow/closure.go` returns `UnsupportedDirectCapabilityError` before `workflow.Service.ResolvePlan` returns a usable plan.                              |
| Remove direct runtime data and routing   | Implemented | Removed `internal/agentruntimesetup/workflowruntime/direct_targets.go`; `sessionv3` payloads, semantic bindings, targets, dispatch plans, and Tool origins contain no direct plan.    |
| Remove generic callable capability facts | Implemented | Removed `internal/agentruntimesetup/workflowruntime/capabilities.go`; no `WorkflowCapabilityFact`, resolver, target field, retry field, recovery field, or Tool-origin field remains. |
| Remove callable capability strings       | Implemented | `internal/llmartifactory-go/workflow/contract/v1/types.go` and `workflow-v1.schema.json` remove `CallableInterface.capabilities`.                                                     |
| Preserve root requirements               | Implemented | `internal/agentruntimesetup/workflowruntime/target_preparation.go` prepares named Artifact pins or registered binding facts without inserting them into `WorkflowProgram.Targets`.    |
| Preserve Service and Activity bindings   | Implemented | `internal/agentruntimesetup/workflowruntime/admission.go` and `binder.go` retain registered `WorkBinding` routing and typed transport checks.                                         |
| Preserve inline text                     | Implemented | `internal/agentruntime-go/sessionv3/workflow_invocation.go` accepts only explicit typed messages, instructions, and defaults. It has no fallback chat-text conversion.                |

## Persist waits, events, channels, and triggers

The runtime owns durable wait state.

A Wait can represent:

- Approval.
- Manual work.
- Human input.
- Timer.
- Event.
- Channel message.
- Child WorkItem completion.
- Child ProgramFrame completion.
- Child Execution completion.
- Recovery review.

The runtime persists timer facts, not timer objects.

The runtime recreates timers from persisted facts after restore.

The runtime owns:

- Event source and identity deduplication.
- Subscription state.
- Channel cursors.
- Channel consumption facts.
- Emit receipts.
- Trigger registrations.
- Trigger occurrences.
- Trigger queues.
- Trigger overlap behavior.
- Missed-trigger behavior.

The runtime has one scheduler for:

- Wait expiry.
- Retry delays.
- Resource waits.
- Execution deadlines.
- Cleanup deadlines.
- Trigger schedules.

Workflow setup must not create another scheduler.

Trigger registration is explicit and independent from direct Workflow start.
A direct start creates one Execution only. It does not activate event or
schedule registrations. Trigger registration persists an input-unbound program
template and is the only operation that activates or changes trigger lifecycle
state.

## Preserve effect safety during retry and recovery

A WorkItem has an effect contract and effect state.

Effect kinds:

- `read`
- `idempotent`
- `keyed`
- `nonIdempotent`

Effect states:

- `notApplicable`
- `notStarted`
- `dispatched`
- `confirmedApplied`
- `confirmedNotApplied`
- `uncertain`
- `compensating`
- `compensated`
- `compensationFailed`

Retry behavior follows effect safety:

- Read, idempotent, and keyed work can replay according to declared policy.
- A keyed operation keeps one stable idempotency key.
- Non-idempotent uncertain work does not replay automatically.
- Confirmed non-application can enable a retry.
- Recovery creates review work when certainty is unavailable.

Workflow retry uses a durable retry ProgramFrame boundary.

The retry boundary retains:

- Attempt count.
- Child frame identity.
- Backoff wait identity.
- Retry deadline.
- Last error.
- Resolved target-set facts.
- Child settlement facts.

## Reconcile uncertain external effects

A mutating WorkItem can become uncertain after:

- Process stop after dispatch.
- Cancellation during external execution.
- Execution timeout.
- External failure without confirmed non-application.
- Runtime panic after dispatch.

The reconciliation flow is:

    Uncertain mutating WorkItem
        ->
    Persist reconciliation WorkItem
        ->
    Run reconciliation child ProgramFrame
        ->
    Applied
        -> confirm original WorkItem
    Not applied
        -> mark confirmed non-application
        -> evaluate retry eligibility
    Unknown
        -> create recovery review wait

A reconciliation block receives durable operation facts.

It does not receive:

- Private clients.
- Provider handles.
- MCP sessions.
- Credentials.
- Callbacks.

## Compensate confirmed effects through durable work

A Saga records confirmed compensatable effects.

The compensation flow is:

    Failed or cancelled Saga
        ->
    Select confirmed compensatable effects
        ->
    Order reverse dependencies
        ->
    Persist Compensation WorkItem
        ->
    Run compensation child ProgramFrame
        ->
    Confirmed compensation
        -> mark effect compensated
    Failed or uncertain compensation
        -> attach typed compensation cause

The runtime preserves primary failure or cancellation as the primary outcome.

Unknown effects do not enter automatic compensation until reconciliation confirms application.

## Cancel, close, and recover safely

Cancellation:

- Stops queued normal WorkItems.
- Cancels active dispatch contexts.
- Cancels ordinary waits.
- Cancels attached child ProgramFrames.
- Cancels eligible independent child Executions.
- Preserves uncertainty for dispatched mutating work.
- Allows bounded cleanup work.
- Preserves cancellation as the primary terminal meaning.

Runtime close:

- Marks the runtime closed.
- Cancels lifecycle context.
- Cancels dispatch contexts.
- Cancels timers.
- Waits for owned asynchronous work.
- Closes process-private execution bindings.
- Releases subscribers.

Recovery:

- Restores Session documents.
- Removes process-private state.
- Marks in-flight external work uncertain unless replay-safe.
- Requeues replay-safe work.
- Starts reconciliation when declared and applicable.
- Creates recovery review waits for unresolved uncertainty.
- Rearms timers only after complete restore.
- Resumes dispatchable work.

## Persist projections and expose wrapper reads

The runtime provides durable projections for:

- Conversation.
- Activities.
- Workflow activity.
- Model operations.
- Tool calls.
- Attempts.
- Waits.
- Apps.
- Usage.
- Protocol output.
- Debugger details.
- Session snapshots.

The Wails wrapper exposes commands and projections.

The wrapper does not own:

- Runtime transitions.
- Workflow scheduling.
- Provider execution.
- Artifact resolution.
- Domain binding.
- Persistence semantics.

## Functional completion ledger

This ledger records remaining functional behavior. It does not prescribe
phases, milestones, or implementation waves.

| Functional capability                                                                                   | Status         | Current evidence or required result                                                                                                                                                                                                                                                     |
| ------------------------------------------------------------------------------------------------------- | -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Durable Session, Execution, ProgramFrame, WorkItem, Attempt, Wait, effect, and trigger state            | Implemented    | `sessionv3/` and `sessionstorev3/` persist the durable execution model                                                                                                                                                                                                                  |
| Ordered work, state updates, scopes, branching, collection work, repeat, machines, and child boundaries | Partial        | Components exist. `RT-DISPATCH-001`, `WF-CONTROL-001`, `WF-SCOPE-001`, and `WF-MACHINE-002` prevent end-to-end completion.                                                                                                                                                              |
| Timers, events, human work, channel waits, and child-completion waits                                   | Partial        | Complete machine rearming, event-driven dispatch, scoped channels, cross-Execution wakeup, and recovered waits.                                                                                                                                                                         |
| Planning-owned declaration path in runtime descriptors                                                  | Partial        | Ordinary descriptors retain paths. Synthetic retry blocks and evaluation boundaries still require completion.                                                                                                                                                                           |
| Channel permission and port authorization semantics                                                     | Partial        | Canonical paths exist. Invocation-local channel identity and published-channel transfer remain open in `WF-PORT-003`.                                                                                                                                                                   |
| Model and Agent Workflow calls with final and streamed ports                                            | Partial        | Typed lowering exists, but literal-call target-use planning, isolated Agent continuation, and composed settlement remain incomplete.                                                                                                                                                    |
| Direct capability targets                                                                               | Implemented    | Unsupported by design. `workflow/closure.go` rejects direct composition for Model, Agent, Tool, MCP, Skill, Service, Activity, and Nested Workflow before ProgramSnapshot creation.                                                                                                     |
| Callable capability requirements                                                                        | Implemented    | `CallableInterface.capabilities` is removed from Workflow v1 grammar and all generic capability-fact runtime plumbing is removed.                                                                                                                                                       |
| Provider-managed Tool calls from Workflow                                                               | Implemented    | Provider SDK Tool Artifacts lower to a typed single-model child plan, retain typed user configuration, derive an owner Model from an explicit Model or Agent, and never become local Tool WorkItems.                                                                                    |
| Service and Activity context, ports, and stream delivery                                                | Implemented    | Registered binding transport facts are checked in preparation and again at dispatch; final output remains Runtime-owned.                                                                                                                                                                |
| Target-family transport behavior                                                                        | Partial        | Preserve existing transport components and complete literal-call coverage plus actual nested published-channel routing.                                                                                                                                                                 |
| `start` and handoff durable results                                                                     | Implemented    | `workflow_call_lifecycle.go` stores the already-created `workflow-start` receipt, handle, target, and declaration path on the parent step outcome before child creation.                                                                                                                |
| Started-call port routing and exactly-once final-output delivery                                        | Implemented    | The wrapper frame retains forwarding facts, suppresses duplicate final delivery, and inner target port routing follows the parent port chain across attached and independent child Executions.                                                                                          |
| Child-frame call policy inheritance                                                                     | Partial        | Identity is scoped correctly. Captured call policy, outer-attempt ordering, retry identity, budgets, and cleanup classification remain open.                                                                                                                                            |
| Trigger lifecycle separation                                                                            | Implemented    | `StartWorkflow` creates one Execution only. `RegisterWorkflowTriggers` is the only command that creates or changes trigger registrations.                                                                                                                                               |
| Calendar repeated-local-time trigger identity                                                           | Partial        | Actual fired slots are retained. Comparing only the immediately previous local slot does not suppress all repeated slots for multi-occurrence fallback schedules.                                                                                                                       |
| Effective invocation configuration for Tool WorkItems emitted by LLM work                               | Implemented    | `LLMToolOrigin` persists the effective dispatch plan, target fingerprint, capability facts, and selected bindings for generated Tool WorkItems.                                                                                                                                         |
| Effects, retries, reconciliation, compensation, and saga execution                                      | Partial        | Complete transition application, source effect translation, stable retry facts, domain certainty, applied reconciliation settlement, and cancellation compensation.                                                                                                                     |
| Recover persisted ordinary retry delays                                                                 | Implemented    | `resumeRestoredWork` now invokes the Runtime due-time lifecycle for every nonterminal Execution, including `RetryWaiting` WorkItems.                                                                                                                                                    |
| Nested Workflow retry policy semantics                                                                  | Implemented    | `workflow_program.go` permits a valid one-attempt retry declaration on a nested Workflow call.                                                                                                                                                                                          |
| Runtime-owned Workflow logical evaluation time                                                          | Implemented    | `ProgramInterpreter.InitialState` receives Runtime logical time. `WorkflowInterpreter` no longer reads the process clock during initialization.                                                                                                                                         |
| One authoritative Runtime scheduler                                                                     | Partial        | One registry exists. Complete event-trigger launch, restored queue and resource progress, conversation arming, and cleanup dispatch.                                                                                                                                                    |
| Semantic reproduction of normal model and Tool continuation                                             | Partial        | Protocol outputs and Tool identity exist. Historical context, isolated rounds, history selection, and cross-Execution Tool-result reconstruction remain incomplete.                                                                                                                     |
| Approved App context in continuation                                                                    | Implemented    | `consumeContinuationLocked` transfers staged `Execution.Continuation.AppContext` into the next persisted message Activity and clears the staged context in the same checkpoint mutation.                                                                                                |
| Durable semantic Skill lifecycle                                                                        | Partial        | Semantic state and hydration exist. Invocation ownership, ordered lifecycle updates, and authoritative reconstruction remain open.                                                                                                                                                      |
| Agent Model system-prompt behavior                                                                      | Implemented    | V3 consumes the selected Agent Model capability `IncludeSystemPrompt` fact during Model patch composition while retaining explicit Session Model prompt precedence.                                                                                                                     |
| Artifact-backed Tool execution and manual triggering                                                    | Implemented    | Standalone built-in Artifact-backed Go and provider SDK Tool declarations are supported. MCP discovered Tools are prepared by MCP Runtime. Skill lifecycle Tools are prepared by Skill Runtime. The standalone Tool scope intentionally excludes arbitrary Tool additions and removals. |
| Normal SessionV2 functional replacement surface                                                         | Partial        | Provider SDK Tool user configuration is implemented. Remaining backend gaps are configuration inheritance, automation, replay, Skill ownership, setup forwarding, and durable settlement.                                                                                               |
| Cost budget execution                                                                                   | Not applicable | Deferred. Do not implement cost metering in this cycle. Current unsupported behavior remains outside the functional handoff scope.                                                                                                                                                      |

The table above is the authoritative functional status register for this
implementation cycle. Every `Not implemented` and `Partial` row is a complete behavior,
not a request for an isolated small edit. Implementers must update the row
status and evidence after the full behavior is complete.

`SessionDefaults.ToolSelections` is the canonical V3 Tool-selection collection.
It contains the Artifact selection plus Tool-only user arguments and automatic
execution facts. `ArtifactSelection` remains generic. V3 must not retain a
second Artifact-only `Tools` collection, because two collections produce
ambiguous precedence and duplicate Tool choices.

The implemented rows describe their named components, not immunity from a shared
dispatch defect. `RT-DISPATCH-001` is a common execution blocker. Full Runtime
completion requires every open parent task and subtask below to be complete.

## Pending functional work

The following work is functional work only. It excludes tests, validation-only
hardening, frontend integration, SessionV2 cleanup and migration, version
changes, release work, and cost metering. Do not add direct capability paths.

| ID                    | Functional task and required result                                                                                                                                                                                                         | Status      | Current source evidence                                                                                                                                                      |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `DOMAIN-PROVIDER-001` | Preserve typed configuration for built-in provider SDK Tool Artifacts and derive a pinned owner Model from an explicit Model or selected Agent. Carry immutable facts through normal Model dispatch plans and Workflow provider Tool plans. | Implemented | `ToolSelection.UserArguments`, `ToolDispatchPlan.ProviderConfiguration`, `ProviderManagedToolChoice.Configuration`, and Agent owner-Model resolution provide the typed path. |
| `DOMAIN-MCP-001`      | Prepare the current MCP Server Artifact and selected discovered MCP Tool operation before Workflow ProgramSnapshot creation. Persist prepared domain facts without placing MCP clients or Artifact Store handles in Session Runtime.        | Implemented | `WorkflowTargetPreparer` prepares and snapshots MCP Tool identity, policy, and discovery facts before Workflow lowering.                                                     |
| `WF-CHILD-001`        | Scope each child-call WorkItem identity to its owning ProgramFrame and logical child instance. Repeats, foreach items, parallel branches, and nested child frames must not collide.                                                         | Implemented | `workflowChildCallWorkID` includes the ProgramFrame identity and logical path.                                                                                               |
| `WF-PORT-001`         | Give a started call one authoritative port delivery boundary. Deliver final output once and route independent-child ports through the correct parent Execution and frame.                                                                   | Implemented | Forward-only wrapper frames preserve routing but suppress duplicate final delivery; recursive port ownership reaches the correct parent Execution.                           |
| `WF-PORT-002`         | Require a mapped nested Workflow port to be published by the target Workflow and compatible with its declared direction, access, stream behavior, and schema.                                                                               | Implemented | Published Workflow port boundaries are explicit ProgramFrame intent facts and reject undeclared mappings.                                                                    |
| `WF-SCHEDULE-001`     | Persist the actual fired local schedule slot and use it to prevent duplicate occurrences for repeated local calendar times during normal operation and recovery.                                                                            | Implemented | Trigger registration, firing, and recovery persist actual local slots before computing successor occurrences.                                                                |

### Open functional task and subtask register

Evidence abbreviations: `R` is `internal/agentruntime-go/sessionv3/`; `S` is
`internal/agentruntimesetup/`. Paths below are relative to those directories.
Statuses describe inspected source behavior, not execution results.
Each parent is complete only when all its subtasks are complete. Implementers
must update parent and subtask statuses and replace required results with exact
source evidence. No tests or validation work are required by this register.

| ID                     | Functional task or subtask                                                                     | Status          | Evidence and required completed behavior                                                                                                                                                                                                                                                                                                                                      |
| ---------------------- | ---------------------------------------------------------------------------------------------- | --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `RT-DISPATCH-001`      | Complete common WorkItem creation and settlement routing.                                      | Partial         | `R/runtime.go`: `advanceFrameLocked`; `R/compensation.go`: `markCompensationIntentLocked`.                                                                                                                                                                                                                                                                                    |
| `RT-DISPATCH-001.a`    | Route compensation-intent mutation only for a Compensation WorkItem.                           | Partial         | Ordinary LLM, Tool, Transform, and Wait creation currently reaches an unconditional compensation-payload dereference. Ordinary execution must reach its owning executor.                                                                                                                                                                                                      |
| `RT-DISPATCH-001.b`    | Keep transition mutation attached to the authoritative Session objects.                        | Partial         | Reacquire Execution, frame, and WorkItem values by identity after appending child Executions or work. A slice append must not leave subsequent settlement or persistence operating on a detached prior value.                                                                                                                                                                 |
| `WF-RECOVERY-001`      | Complete reconciliation execution and settlement.                                              | Partial         | `R/workflow_program.go`, `workflow_reconciliation.go`, and `runtime_reconciliation.go`.                                                                                                                                                                                                                                                                                       |
| `WF-RECOVERY-001.a`    | Apply every returned reconciliation transition.                                                | Partial         | `Advance` discards transitions from reconciliation start, intent settlement, and child settlement. Persist their state, work, children, review waits, and settlements through the ordinary transition path.                                                                                                                                                                   |
| `WF-RECOVERY-001.b`    | Complete applied, not-applied, and unknown outcomes.                                           | Partial         | Applied results must deliver confirmed ports, retain receipt and effect facts, register eligible Saga effects, and wake completion waits. Non-application must resume the original retry or failure boundary; unknown must retain uncertainty.                                                                                                                                |
| `WF-CONTROL-001`       | Complete structured results, exits, state propagation, and aggregate outcomes.                 | Partial         | `R/workflow_structured.go`, `workflow_handlers.go`, `workflow_saga.go`, and `workflow_program.go`.                                                                                                                                                                                                                                                                            |
| `WF-CONTROL-001.a`     | Settle control exits and block values exactly once.                                            | Partial         | Remove double wrapping of `WorkflowBlockResult`; process return, break, and continue before an idle pending-path or normal-output path. Clear completed boundaries and preserve the correct declaration outcome.                                                                                                                                                              |
| `WF-CONTROL-001.b`     | Apply declared state and result semantics.                                                     | Partial         | Merge inherited Saga-body and successful finally state; retain state updates before Repeat break; report actual exhaustion reason; preserve aggregate branch/item outcomes for collect behavior and retain uncertainty rather than converting it to cancellation or failure-only data.                                                                                        |
| `WF-SCOPE-001`         | Preserve the complete semantic evaluation scope across nested work.                            | Partial         | `R/workflow_structured.go`, `workflow_coordination.go`, `workflow_messages.go`, and Repeat control evaluation.                                                                                                                                                                                                                                                                |
| `WF-SCOPE-001.a`       | Construct child scopes through one inheritance implementation.                                 | Partial         | Retain available item, index, result, event, error, outcome, operation, completed, ports, Saga ownership, and accepted target-set facts. Apply only the controlling construct's declared replacements. Repeat conditions receive the current iteration index.                                                                                                                 |
| `WF-SCOPE-001.b`       | Capture complete wait-filter and output evaluation facts.                                      | Partial         | Filters currently retain only input, state, messages, and steps. Retain logical time and applicable scope facts. Empty blocks and final outputs need their block declaration boundary rather than a nonexistent next step.                                                                                                                                                    |
| `WF-CALL-002`          | Complete static and resolver-backed callable selection.                                        | Partial         | `S/workflowruntime/compiler.go`; `R/workflow_target_resolver.go` and `workflow_program.go`.                                                                                                                                                                                                                                                                                   |
| `WF-CALL-002.a`        | Lower allowed target sets to one canonical representation.                                     | Partial         | Static sets currently populate both mutually exclusive runtime fields. Preserve the source set and callable interface without competing membership representations.                                                                                                                                                                                                           |
| `WF-CALL-002.b`        | Resume the call after accepted resolver membership.                                            | Partial         | Resolver settlement leaves `PendingPath` set and returns to an idle path. Continue the original suspended call using persisted membership and selection facts. Coordinate with Artifact task `ART-INV-001` for literal `TargetUses`.                                                                                                                                          |
| `WF-RETRY-002`         | Preserve one logical call through durable retry attempts.                                      | Partial         | `R/workflow_retry.go`, `retry_program.go`, `workflow_call_lifecycle.go`, and child-call construction.                                                                                                                                                                                                                                                                         |
| `WF-RETRY-002.a`       | Make retry-created blocks executable and source-identifiable.                                  | Partial         | Synthetic retry blocks lack root declaration identity. Preserve the declaration path while giving each attempt its own runtime identity.                                                                                                                                                                                                                                      |
| `WF-RETRY-002.b`       | Capture callable facts before the retry boundary.                                              | Partial         | Reuse evaluated input, context, typed LLM invocation, target membership, selected target, effect key, and overall deadline. Persist the outer call attempt before child invocation can run; a new attempt frame must not create a new business operation.                                                                                                                     |
| `WF-EFFECT-002`        | Complete effect translation, domain invocation facts, and truthful settlement.                 | Partial         | `S/workflowruntime/compiler.go`, `binder.go`; `S/toolruntime/v3.go`; `S/mcpruntime/v3.go`; `R/effect_state.go` and `runtime.go`.                                                                                                                                                                                                                                              |
| `WF-EFFECT-002.a`      | Translate declaration effects and deliver stable invocation metadata.                          | Partial         | Map source `non-idempotent` to runtime `nonIdempotent`. Registered keyed bindings must receive the captured key and operation identity through typed invocation facts, not generic context or rewritten native arguments.                                                                                                                                                     |
| `WF-EFFECT-002.b`      | Preserve actual business-effect certainty.                                                     | Partial         | Pre-invocation denials must not become applied effects. Generic invocation errors must not fabricate applied or not-applied receipts. Preserve domain-confirmed error results, uncertainty, retry eligibility, and compensation eligibility; deliver successful final ports only for the declared successful result meaning.                                                  |
| `RT-CLEANUP-002`       | Complete cancellation, finalization, reconciliation, and compensation under cleanup allowance. | Partial         | `R/runtime.go`, `runtime_child_cancel.go`, `budget.go`, and `workflow_handlers.go`.                                                                                                                                                                                                                                                                                           |
| `RT-CLEANUP-002.a`     | Permit and advance cleanup work after normal cancellation or timeout.                          | Partial         | Execution and settlement currently stop on cancellation even for cleanup. Use the cleanup deadline instead of an elapsed normal deadline. Ordinary child-call boundaries must not be unconditionally classified as cleanup.                                                                                                                                                   |
| `RT-CLEANUP-002.b`     | Finish ownership only after eligible settlement and cleanup.                                   | Partial         | Keep dispatched effect outcomes observable; register late confirmed compensatable effects; reconcile uncertainty where declared; wait for independent-child cleanup; preserve primary cancellation through finally. Consolidate root, child, and deadline cancellation behavior.                                                                                              |
| `WF-PORT-003`          | Connect published nested ports and preserve invocation-local coordination.                     | Partial         | `R/workflow_published_ports.go`, `workflow_ports.go`, `ports_runtime.go`, `coordination.go`, and child materialization.                                                                                                                                                                                                                                                       |
| `WF-PORT-003.a`        | Establish the declared caller-port to child-channel connection.                                | Not implemented | Transfer input values to the target's published local boundary and forward declared child output/stream records to the caller channel. Preserve one final-delivery owner; do not substitute an internal block-result envelope for a declared final value.                                                                                                                     |
| `WF-PORT-003.b`        | Scope channels and launch work woken across Execution boundaries.                              | Partial         | Distinct nested Workflow invocations must not share a channel merely because names match. Structured blocks share only their owning invocation's channels. A streamed delivery must promptly dispatch newly ready parent work, not wait for child completion.                                                                                                                 |
| `WF-MACHINE-002`       | Complete machine rearming and durable transition selection.                                    | Partial         | `R/workflow_machine.go`, `subscription_runtime.go`, and event envelope construction.                                                                                                                                                                                                                                                                                          |
| `WF-MACHINE-002.a`     | Rearm each consumed wait occurrence without losing other subscriptions.                        | Partial         | An unmatched event can leave its subscription unarmed or recreate an existing WorkItem ID. Allocate a new wait occurrence and preserve the other active waits.                                                                                                                                                                                                                |
| `WF-MACHINE-002.b`     | Apply declared event and timer semantics.                                                      | Partial         | Select immediate, event, then timer candidates from durable facts; reset the immediate-transition allowance after external suspension. Expose the declaration event envelope, including its documented type meaning, consistently to filters, waits, triggers, and machine actions.                                                                                           |
| `RT-TRIGGER-002`       | Complete trigger-created work launch, queue resumption, and local-slot identity.               | Partial         | `R/runtime_triggers.go`; `R/runtime.go`: `changeSession`; `R/runtime_durability.go`.                                                                                                                                                                                                                                                                                          |
| `RT-TRIGGER-002.a`     | Dispatch accepted event-trigger work and resume eligible queues.                               | Partial         | Event acceptance persists new Executions but does not launch their ready work. Restore and reactivation must drain eligible persisted occurrences through the same Runtime dispatch path.                                                                                                                                                                                     |
| `RT-TRIGGER-002.b`     | Deduplicate every repeated local calendar occurrence.                                          | Partial         | Actual fired-slot persistence exists. A comparison with only the previous local minute fails for schedules with several occurrences during a fallback hour. Retain a suitable local-slot identity or watermark through firing and restore.                                                                                                                                    |
| `RT-LIMIT-002`         | Complete scoped budgets, concurrency scheduling, claims, and deadlines.                        | Partial         | `R/budget.go`, `runtime_resources.go`, `runtime_fork_join.go`, `coordination.go`, and dispatch paths.                                                                                                                                                                                                                                                                         |
| `RT-LIMIT-002.a`       | Account for both local limits and shared parent usage.                                         | Partial         | A narrowed frame/item limit must not compare only against all earlier Execution usage. Enforce local and aggregate counts, token/message/event usage, and narrowed concurrency. Queue capacity waits and reserve capacity across binding and dispatch instead of failing ordinary work.                                                                                       |
| `RT-LIMIT-002.b`       | Apply claims and lifecycle timing to every owning construct.                                   | Partial         | Emit currently bypasses claim acquisition. Wake resource waiters after restored leases are released; apply wait deadlines; arm conversation and child due times consistently. Keep nested resource namespaces and independent-child resources separate from unrelated parent declarations.                                                                                    |
| `V3-CONFIG-002`        | Complete effective artifact-backed invocation configuration.                                   | Partial         | `S/sessionv3runtime/llm_plan.go`, `workspace_overrides.go`, and Tool choice preparation.                                                                                                                                                                                                                                                                                      |
| `V3-CONFIG-002.a`      | Preserve contribution ownership and explicit override semantics.                               | Partial         | `ToolSelections` is now the one canonical Tool collection and carries Agent auto-execution facts. Complete inherited collection removal, replacement provenance, and explicit contribution ownership. Keep Model default layering owned by the Model adapter.                                                                                                                 |
| `V3-CONFIG-002.b`      | Preserve the selected Workspace MCP configuration.                                             | Partial         | Workspace preparation computes selected MCP facts but later constructs `all` exposure records. Retain selected Tools, instructions choice, resources, templates, prompts, and overrides through the final effective plan.                                                                                                                                                     |
| `V3-HISTORY-002`       | Complete semantic conversation reconstruction.                                                 | Partial         | `S/sessionv3runtime/inputs.go`, `llm.go`, `llm_plan.go`, and retained dispatch plans.                                                                                                                                                                                                                                                                                         |
| `V3-HISTORY-002.a`     | Reconstruct the exact authorized message and Tool sequence.                                    | Partial         | Correct the extra previous-user-turn selection, retain applicable system/developer messages, exclude unrelated audit-only Workflow input, include isolated Agent assistant rounds, and attach completed Tool results from the historical origin Execution.                                                                                                                    |
| `V3-HISTORY-002.b`     | Reuse retained semantic context and provider outcomes.                                         | Partial         | Replay original per-message Workspace/MCP context instead of dropping or refetching it. Recreate private execution resources from retained effective semantic plans after recovery. Preserve usable partial provider response, usage, and completed protocol facts when an execution error accompanies a response.                                                            |
| `V3-CONTROL-002`       | Complete normal conversation queuing and automation controls.                                  | Partial         | `R/program.go`, `normal_session.go`, `runtime.go`, and `runtime_fork_join.go`.                                                                                                                                                                                                                                                                                                |
| `V3-CONTROL-002.a`     | Queue messages through the pending Fork and active Agent child boundary.                       | Partial         | A second submission while a conversation Fork is pending must queue rather than attempt another overlapping Fork. Preserve immutable invocation defaults and process each selected message in order.                                                                                                                                                                          |
| `V3-CONTROL-002.b`     | Separate explicit execution from automatic Tool and continuation policy.                       | Partial         | Explicitly requested Tools must remain executable under an automatic Tool hold. Releasing only a Tool hold must not request Model continuation or bypass a continuation hold. Preserve one-batch and indefinite semantics.                                                                                                                                                    |
| `DOMAIN-SKILL-002`     | Complete invocation-owned Skill lifecycle state.                                               | Partial         | `S/skillruntime/v3.go` and `v3_hydration.go`.                                                                                                                                                                                                                                                                                                                                 |
| `DOMAIN-SKILL-002.a`   | Own private Skill sessions at the correct invocation scope.                                    | Partial         | Execution-keyed sessions allow parallel Agents to replace or close one another's sessions. Share state only where the conversation/invocation contract explicitly shares it; keep private handles in the Skill owner.                                                                                                                                                         |
| `DOMAIN-SKILL-002.b`   | Apply lifecycle operations in an authoritative semantic order.                                 | Partial         | Do not reset every generated Tool to the old LLM-origin active state. Combine the immutable origin allow-list with current durable lifecycle state, serialize conflicting updates, and reconstruct by semantic settlement order rather than WorkItem creation order. Retain state changes accompanying a confirmed domain error.                                              |
| `DOMAIN-BINDING-002`   | Complete native Tool binding facts and registered binding waits.                               | Partial         | `S/sessionv3runtime/mcp_exec.go`; `S/workflowruntime/admission.go` and `binder.go`; `R/normal_app.go`.                                                                                                                                                                                                                                                                        |
| `DOMAIN-BINDING-002.a` | Persist the owning domain's resolved typed Tool call before invocation.                        | Partial         | Workflow Tool binding currently replaces the call only in private requests. Persist the native semantic binding needed by projections, replay, and MCP App creation without storing clients or authorization tokens.                                                                                                                                                          |
| `DOMAIN-BINDING-002.b` | Let registered Service and Activity bindings request durable waits.                            | Not implemented | `WorkBinding.ResolveWait` has routing, but `WorkBinding.Bind` cannot return a wait. Supply one neutral binding result and preserve domain-private resume state outside the Session document.                                                                                                                                                                                  |
| `V3-SETUP-002`         | Preserve optional backend capabilities through binder composition.                             | Partial         | `S/sessionv3runtime/binder.go` implements Skill template and preview ports; `S/workflowruntime/binder.go` does not forward them.                                                                                                                                                                                                                                              |
| `V3-SETUP-002.a`       | Forward user-message Skill template preparation.                                               | Not implemented | Runtime receives the composed binder and therefore cannot discover the base preparer's implemented capability. Forward through the existing optional backend contract.                                                                                                                                                                                                        |
| `V3-SETUP-002.b`       | Forward instruction Skill preview.                                                             | Not implemented | Preserve the base preview capability through the composed binder without creating a second rendering implementation.                                                                                                                                                                                                                                                          |
| `RT-PERSIST-002`       | Complete exact checkpoint and deletion commitment semantics.                                   | Partial         | `R/runtime.go`: `DeleteSession`, `persistLocked`, and command retention; `internal/sessionstorev3/store.go`: `HandleCheckpoint`.                                                                                                                                                                                                                                              |
| `RT-PERSIST-002.a`     | Commit deletion durably before exposing its accepted result.                                   | Partial         | Release the lock for an unknown non-tombstoned Session. A failed tombstone write must remain retryable and must not become an in-memory duplicate success. Preserve the authoritative Session until the durable deletion barrier is committed.                                                                                                                                |
| `RT-PERSIST-002.b`     | Acknowledge the exact semantic mutation, including command-only updates.                       | Partial         | Revision/timestamp-only comparison drops some command records. Include the durable sequence/identity contract; distinguish document commitment from derived-index failure; never acknowledge an ignored stale snapshot as current. Make command retention satisfy the documented idempotency lifetime rather than silently forgetting accepted identities after 1024 records. |
| `WF-HANDOFF-002`       | Make conversation ownership affect backend routing.                                            | Partial         | `R/runtime_children.go` records ownership, but normal submission and `conversationExecutionLocked` do not consume it.                                                                                                                                                                                                                                                         |
| `WF-HANDOFF-002.a`     | Route subsequent normal Session work to the Session-scoped semantic owner.                     | Partial         | `applySessionConversationOwnerDefaults` uses only the ownership record keyed by `string(Session.ID)`. Complete routing into a live handoff Agent-loop frame where that frame remains the active owner. Do not add another durable conversation identifier.                                                                                                                    |
| `WF-HANDOFF-002.b`     | Apply restore and retain behavior to future Session-scoped routing.                            | Partial         | Completion already restores or retains ownership. Complete recovery-time selection and live-owner routing while preserving `Session.ID` as the sole normal conversation identity.                                                                                                                                                                                             |
| `ARC-OWNERSHIP-002`    | Complete domain ownership and dependency separation.                                           | Partial         | `S/sessionv3runtime/llm_plan.go`, `S/workflowruntime/model_agent_adapter.go`, and concrete domain binding imports.                                                                                                                                                                                                                                                            |
| `ARC-OWNERSHIP-002.a`  | Move contribution preparation to its owning domain.                                            | Partial         | Agent owns reusable contribution policy and Model selection; Workspace owns Workspace selection semantics; Attachment owns snapshot materialization; MCP and Skill own their preparation. Session/Workflow setup assemble typed results and do not independently reinterpret those rules.                                                                                     |
| `ARC-OWNERSHIP-002.b`  | Separate neutral execution contracts from Artifact-dependent Workflow preparation.             | Partial         | Attachment and Text bindings import `workflowruntime`, which also imports Artifact planning. Put registered binding contracts and common prepared-choice values at an appropriate neutral boundary. Domain execution and core Runtime must not depend on Artifact Store; setup source adapters may read generic Artifact facts.                                               |
| `ARC-DUP-002`          | Remove duplicated execution behavior and replicated effective plans.                           | Partial         | The following consolidation is part of functional ownership completion, not SessionV2 cleanup or validation work.                                                                                                                                                                                                                                                             |
| `ARC-DUP-002.a`        | Consolidate shared algorithms and Runtime lifecycle paths.                                     | Partial         | Unify child-scope construction, value evaluation, semantic error cloning, reconciliation/compensation input extraction, root Execution construction, cancellation settlement, and conversation projection. Index lowered call descriptors once rather than repeatedly searching the whole tree for each call.                                                                 |
| `ARC-DUP-002.b`        | Retain one canonical effective plan with truthful contribution provenance.                     | Partial         | `llm_dispatch_plan_facts.go` copies the complete effective collections into Agent and Workspace projections; each Tool origin copies the plan again. Use immutable references to authoritative retained facts where possible, preserve origin identity, and do not mislabel Session contributions as Agent or Workspace-owned facts.                                          |

Architecture completion must not introduce a second scheduler, a second durable
execution root, native request interpretation in Workflow, or an Artifact Store
execution API. Existing completed SDK Tool configuration, no-published-port
rejection, forward-only start wrappers, and shared V3 MCP output conversion are
retained; their presence does not close the distinct open tasks above.

### Current implementation status

This table supersedes earlier status wording in this document when the same
identifier appears more than once. Status is based on source inspection only.
Tests and validation work are intentionally not included.

| ID                   | Status      | Current implementation evidence and remaining functional result                                                                                                                                                                                                                                                     |
| -------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `RT-DISPATCH-001`    | Partial     | Non-Compensation WorkItems no longer enter compensation-intent mutation. `advanceFrameLocked` still needs complete authoritative-reference consolidation after all child Execution and frame mutations.                                                                                                             |
| `WF-RECOVERY-001`    | Partial     | Reconciliation transitions now enter Runtime transition handling. Applied reconciliation now delivers ports, records confirmed effects, and wakes completion waits. Retry and cancellation interaction must still preserve every original-call outcome.                                                             |
| `WF-CONTROL-001`     | Partial     | Child result wrapping, propagated exits, Saga merge state, finally merge state, and Repeat break state are implemented. Aggregate collect outcomes and all uncertainty propagation remain incomplete.                                                                                                               |
| `WF-SCOPE-001`       | Implemented | `workflowInheritedChildScope` is the common child-scope constructor. Wait filters retain scope and logical time. Empty blocks use their declaration path for message capture.                                                                                                                                       |
| `WF-CALL-002`        | Implemented | Static target sets lower only to static members; resolver-backed sets lower only to durable resolver state. Resolver settlement resumes the preserved call. Inventory literal target extraction is fixed in `workflowLiteralTargetName`.                                                                            |
| `WF-RETRY-002`       | Partial     | Retry synthetic blocks retain declaration identity. Retry state now captures evaluated target, input, context, keyed effect key, and target-set facts for local, Service, Activity, and child-call work. Typed Model or Agent message capture remains incomplete.                                                   |
| `WF-EFFECT-002`      | Partial     | Runtime accepts explicit domain effect-state facts. Go Tool transport errors become uncertain. MCP pre-invocation denial becomes confirmed non-application. MCP error responses are conservative uncertain outcomes. Remaining registered-domain effect evidence and complete retry policy propagation remain open. |
| `RT-CLEANUP-002`     | Partial     | Cleanup WorkItems can dispatch after cancellation and independent Executions no longer finish before outstanding cleanup. Complete late-effect reconciliation, compensation after cancellation, and root/child timeout unification remain open.                                                                     |
| `WF-PORT-003`        | Partial     | Cross-Execution channel wakeup now launches ready work immediately. Published-port local-channel mapping and per-invocation channel namespaces remain open.                                                                                                                                                         |
| `WF-MACHINE-002`     | Partial     | Consumed machine waits receive unique WorkItem identities, preserving other active waits. Full durable immediate/event/timer arbitration and event-envelope normalization remain open.                                                                                                                              |
| `RT-TRIGGER-002`     | Implemented | Session mutations launch ready trigger work. Recovery and reactivation drain eligible queues. Repeated local schedule slots are retained by local-day slot identity and skipped during successor calculation.                                                                                                       |
| `RT-LIMIT-002`       | Partial     | Execution and frame concurrency now reserve Binding and Dispatched capacity and queue instead of terminally blocking work. Scoped counters, nested resource namespaces, and full recovery resource wakeup remain open.                                                                                              |
| `V3-CONFIG-002`      | Partial     | Agent Tool `autoExecute`, Agent/Workspace/Session instruction ordering, and Workspace MCP selection retention are implemented. Explicit inherited collection removal and complete provenance ownership remain open.                                                                                                 |
| `V3-HISTORY-002`     | Partial     | History count, audit exclusion, isolated Agent assistant rounds, prior Execution Tool results, retained context replay, and partial provider output retention are implemented. Full reconstruction of all source-backed private preparation resources remains open.                                                 |
| `V3-CONTROL-002`     | Implemented | Conversation activities queue while a Fork boundary is pending. Explicit Tool requests bypass automatic Tool holds. Disabling only Tool automation no longer requests continuation.                                                                                                                                 |
| `DOMAIN-SKILL-002`   | Implemented | Skill Runtime sessions are keyed by Execution plus owning ProgramFrame. Generated lifecycle Tools serialize in provider order. Active state derives from the latest terminal semantic lifecycle result for that invocation frame.                                                                                   |
| `DOMAIN-BINDING-002` | Implemented | Bound native Tool calls persist before dispatch. Neutral registered-binding results can request a durable wait and retain private resume state outside Session persistence.                                                                                                                                         |
| `V3-SETUP-002`       | Implemented | The composed Workflow binder forwards user-message Skill template preparation and instruction Skill preview to the base binder.                                                                                                                                                                                     |
| `RT-PERSIST-002`     | Implemented | Tombstones commit before in-memory deletion. Failed tombstone writes restore the live Session. Checkpoint comparison includes event sequence and exact snapshot identity. Derived-index failures are explicitly durable and schedule repair. Command records are not silently truncated.                            |
| `WF-HANDOFF-002`     | Partial     | Conversation input and Execution records retain a conversation identity. New normal conversation Executions are separated by conversation identity and use the active handoff Agent as their effective selection. Direct routing to a live handoff child and full multi-owner conflict policy remain open.          |
| `ARC-OWNERSHIP-002`  | Partial     | Attachment and Text use neutral `agentruntime-go/workbinding` contracts and no longer import Artifact-aware Workflow setup. Agent and Workspace contribution ownership still needs consolidation.                                                                                                                   |
| `ARC-DUP-002`        | Partial     | Child scope construction, trigger dispatch, and registered binding contracts are consolidated. Duplicate evaluation, cancellation, projection, and effective-plan ownership paths remain open.                                                                                                                      |

No item in this table authorizes a version bump, SessionV2 cleanup, frontend
integration, test work, validation-only work, migration, stored-session
conversion, release work, or cost metering.

## Validate at the owning boundary

| Validation class                                                | Owner                                         |
| --------------------------------------------------------------- | --------------------------------------------- |
| Workflow source grammar and static semantic validity            | Workflow Artifact contract                    |
| Static Artifact relationship closure                            | Workflow Artifact planning                    |
| Workflow execution preparation and source-to-program conversion | Workflow setup                                |
| Invocation input validation                                     | Workflow setup                                |
| Current Artifact pin and enablement                             | Workflow setup or target-specific binder      |
| Direct composition rejection for Workflow targets               | Workflow Artifact planning and Workflow setup |
| Domain readiness and native request validity                    | Owning domain adapter                         |
| Durable Session and execution integrity                         | Session Runtime v3                            |
| Session document and tombstone persistence                      | Session Store v3                              |
| Wails request transport                                         | Wrapper and frontend transport                |

The runtime validates durable runtime objects. It does not revalidate Workflow source grammar.

## Implement and verify the durable runtime

Status values:

| Status          | Meaning                                                                                  |
| --------------- | ---------------------------------------------------------------------------------------- |
| Implemented     | The inspected source contains the behavior. Test execution was not reviewed.             |
| Partial         | The inspected source contains part of the behavior or has an identified boundary gap.    |
| Not implemented | The required behavior is absent from the inspected source.                               |
| Not applicable  | The item is explicitly excluded from this implementation cycle or is not a Runtime task. |

### Persist and recover Session execution state

| Subtask                                                                                               | Status         | Evidence or required result                                                                  |
| ----------------------------------------------------------------------------------------------------- | -------------- | -------------------------------------------------------------------------------------------- |
| Persist Session, Execution, ProgramFrame, WorkItem, Attempt, Wait, effect, channel, and trigger facts | Implemented    | `internal/agentruntime-go/sessionv3/`                                                        |
| Enforce checkpoint-before-dispatch behavior                                                           | Implemented    | Runtime checkpoint and dispatch gates                                                        |
| Persist and restore Session Store v3 documents and tombstones                                         | Implemented    | `internal/sessionstorev3/store.go` persists documents and tombstones and restores both forms |
| Verify restore behavior through integration tests                                                     | Not applicable | Tests are outside this implementation cycle.                                                 |

### Prepare Workflow execution from a complete static plan

| Subtask                                                             | Status         | Evidence or required result                                                                                                                                                                                      |
| ------------------------------------------------------------------- | -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Separate callable targets from requirements                         | Implemented    | `workflow.Plan.Inventory`                                                                                                                                                                                        |
| Consume planning-owned callable targets and requirements separately | Implemented    | `workflowruntime/admission.go`                                                                                                                                                                                   |
| Publish complete static Artifact closure with immutable pins        | Implemented    | `workflow.Plan.Closure` and `workflow/closure.go`                                                                                                                                                                |
| Stop recursive nested Workflow plan resolution during setup         | Implemented    | `workflowruntime/admission.go` consumes closure `WorkflowPlanNode` facts                                                                                                                                         |
| Keep execution preparation outside Session Runtime commands         | Implemented    | Wrapper invokes `workflowruntime.Admission.PrepareWorkflow` or `PrepareWorkflowTriggers` before Runtime receives `ProgramSnapshot`                                                                               |
| Remove Workflow Artifact selections from core Workflow commands     | Implemented    | Core `StartWorkflowCommand` and `RegisterWorkflowTriggersCommand` accept prepared `ProgramSnapshot` only                                                                                                         |
| Rename Workflow admission to execution preparation                  | Not applicable | This is naming cleanup only. It is not functional work in this cycle.                                                                                                                                            |
| Lower one admitted static plan with canonical declaration paths     | Implemented    | `workflowruntime/declaration_paths.go` decorates the lowered tree. Structured child frames, WorkItems, handlers, retries, sagas, reconciliation, and uncertain outcomes retain planning-owned declaration paths. |

### Bind target domains without Workflow leakage

| Subtask                                                                      | Status      | Evidence or required result                                                                                                                                  |
| ---------------------------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Bind Model and Agent through existing setup APIs                             | Implemented | Model and Agent invocation adapter                                                                                                                           |
| Bind Go Tools, MCP Tools, and Skill Tools through native arguments           | Implemented | V3 Tool, MCP, and Skill setup executors                                                                                                                      |
| Keep Provider SDK Tools provider-managed                                     | Implemented | Workflow SDK Tool targets produce a typed Model child plan and provider-managed choice only                                                                  |
| Bind Service and Activity through registered adapters                        | Implemented | Workflow binding registry                                                                                                                                    |
| Prepare Service and Activity context and port transport before work creation | Implemented | Binding capability facts are consumed by Workflow preparation and durable target descriptors                                                                 |
| Preserve Model and Agent output and stream port plans                        | Implemented | Child ProgramFrame intents retain accepted final and streamed output plans before child work begins                                                          |
| Prepare target-family transport behavior before WorkItem creation            | Implemented | Preparation validates the target matrix and Runtime rejects any unsupported adapter port route                                                               |
| Remove direct capability routing                                             | Implemented | `workflowruntime/direct_targets.go`, direct binder routing, direct bootstrap wiring, direct target payloads, and direct semantic binding fields are removed. |
| Remove generic callable capability framework                                 | Implemented | `WorkflowCapabilityRequirement`, `WorkflowCapabilityFact`, generic fact resolvers, and callable `capabilities` schema/model fields are removed.              |
| Lower provider-managed Workflow Tools through typed Model child plans        | Implemented | SDK Tool choices remain inside typed Model plans and never enter local Tool dispatch                                                                         |
| Move concrete URL and diff behavior out of generic Workflow setup            | Implemented | `attachmentruntime` and `textruntime` own behavior; Workflow composition only registers bindings                                                             |

### Execute Workflow reliability behavior

| Subtask                                                                   | Status         | Evidence or required result                                                                                                                                                   |
| ------------------------------------------------------------------------- | -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Persist state owner identity and schema                                   | Implemented    | Workflow state-owner runtime code                                                                                                                                             |
| Persist Parallel and Foreach parent-state merges                          | Implemented    | Durable StateUpdate merge WorkItems                                                                                                                                           |
| Persist failed and uncertain step outcomes                                | Partial        | Preserve complete target, receipt, effect, aggregate, and declaration facts through ordinary, reconciliation, cancellation, and structured settlement; see the open register. |
| Make Workflow logical evaluation time Runtime-owned                       | Implemented    | Runtime supplies logical initialization time and `advanceFrameLocked` supplies transition time.                                                                               |
| Persist retry boundaries and backoff waits                                | Implemented    | `RetryState` remains durable and Runtime re-arms its due-time callback after complete restore.                                                                                |
| Preserve effects, reconciliation, and compensation                        | Partial        | Complete `WF-RECOVERY-001`, `WF-RETRY-002`, `WF-EFFECT-002`, and `RT-CLEANUP-002`.                                                                                            |
| Verify Service and Activity retry adaptation for every registered binding | Not applicable | Tests and validation work are excluded from this implementation cycle.                                                                                                        |
| Verify cancellation races for every mutating target family                | Not applicable | Tests and validation work are excluded from this implementation cycle.                                                                                                        |

### Complete normal conversation parity

| Subtask                                                               | Status      | Evidence or required result                                                                                                                                       |
| --------------------------------------------------------------------- | ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Persist messages, Tool calls, waits, streaming, Apps, and projections | Implemented | Session Runtime v3 and V3 setup binder                                                                                                                            |
| Persist typed Model defaults and selected Artifact facts              | Implemented | Session defaults and dispatch plans                                                                                                                               |
| Support artifact-backed normal Session configuration                  | Partial     | Complete inherited collection overrides, contribution policy, Workspace MCP selections, and instruction ownership order in `V3-CONFIG-002`.                       |
| Reproduce later artifact-backed conversation behavior                 | Partial     | Complete historical context, isolated rounds, history boundaries, cross-Execution Tool results, and retained-plan reconstruction in `V3-HISTORY-002`.             |
| Map required artifact-backed Session behavior to V3                   | Partial     | Backend replacement requires the open conversation, control, Skill, binding, setup-forwarding, and persistence tasks. SDK Tool user configuration already exists. |
| Migrate frontend traffic and stored Sessions                          | Excluded    | Frontend integration and Session v2 migration are outside this audit scope                                                                                        |

### Remove repeated admission and setup layers

| Removal item                                                | Status         | Required result                                                                                                                                                                    |
| ----------------------------------------------------------- | -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Remove static declaration rechecks after Workflow admission | Not applicable | This is validation-only cleanup. Do not add it as independent work in this cycle; remove redundant checks only when functional ownership consolidation makes them unnecessary.     |
| Remove repeated source-tree walks during lowering           | Implemented    | Compiler lowers admitted source once; Inventory decorates the lowered runtime tree without another source traversal                                                                |
| Remove the runtime-owned Workflow preparation port          | Implemented    | Setup submits prepared snapshots; Session Runtime persists and interprets them                                                                                                     |
| Collapse duplicate Model dispatch-plan processing           | Implemented    | `llmExecutionPlan.activities` is prepared once and reused for provider input construction and durable dispatch-plan facts; generated Tool execution reads retained origin facts.   |
| Normalize MCP Tool exposure once                            | Implemented    | `normalizeMCPSelections` persists only `all`, `none`, or `selected` before dispatch-plan construction                                                                              |
| Assign one dispatch-time pin owner per target family        | Implemented    | Go Tool freshness remains in Tool Runtime; MCP freshness is verified by MCP Runtime; generic Workflow wrappers no longer read pins                                                 |
| Remove generic Workflow concrete bindings                   | Implemented    | URL normalization moved to Attachment Runtime and unified diff moved to Text Runtime; Workflow composition only registers them                                                     |
| Remove runtime reconstruction of declaration permissions    | Implemented    | Canonical declaration paths propagate through child ProgramFrames, WorkItems, outcomes, Fork and Join boundaries. Runtime uses persisted plans and paths without source reparsing. |

### Verify deployment composition

| Subtask                                                 | Status      | Evidence or required result                                                               |
| ------------------------------------------------------- | ----------- | ----------------------------------------------------------------------------------------- |
| Verify Session Store v3 checkpoint and restore behavior | Implemented | `internal/sessionstorev3/` persists, restores, indexes, and projects Session v3 documents |
| Verify application bootstrap ordering                   | Implemented | `cmd/agentgo/app.go` constructs dependencies before Store v3 and Runtime v3 restore       |
| Verify runtime shutdown ordering                        | Implemented | `SessionV3Wrapper.close` closes Runtime v3 before Store v3                                |
| Verify Wails command and projection compatibility       | Excluded    | Frontend integration is outside this audit scope                                          |
