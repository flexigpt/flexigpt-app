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

The current source-backed status assessment covers the supplied SessionV3
Runtime, Workflow preparation, V3 Session setup, Model, Tool, MCP, Skill,
attachment, registered-binding, Store v3, bootstrap, wrapper, Workflow
Inventory, and closure source. V2 source is used only as a backend functional
replacement reference. Absence claims apply only to that supplied source set.

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

The inline completion registers in this document contain the current
source-audited state. They supersede conflicting statuses and evidence in
earlier component tables, including the ID mappings recorded there.

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

Core Runtime owns durable Session execution, logical evaluation boundaries,
scheduling, settlement, cancellation, recovery, and lifecycle notification. It
must not import Artifact Store or setup packages.

Setup can consume existing generic Artifact facts and read-only Workflow plans.
Artifact Store must not gain preparation, binding, dispatch, scheduling,
recovery, or consumer-private APIs.

Domain setup owns native request grammar, source-family contribution semantics,
current private resources, native invocation, and native result conversion.
Session and Workflow setup compose and orchestrate neutral domain facts.

An immutable origin-plan reference must preserve invocation meaning. Replacing
a copied plan with a reference must not cause generated Tools to resolve current
Session defaults. Shared Runtime helpers must replace equivalent algorithms
without becoming a generic native-domain envelope. Lowered-tree traversal
remains separate from Artifact-source interpretation.

MCP connection, authorization, discovery, and their timers remain MCP-owned
lifecycle behavior. They are separate from the Runtime orchestration due-time
owner. V2 coexistence remains intact; this document does not authorize V2
retirement. Ownership consolidation must not add repeated admission or a
separate validation project.

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

| ID                        | Current status | Required end state before marking `Implemented`                                                                                                                                                                                        | Current source evidence                                                                                                                                                                                                                                                                                                               |
| ------------------------- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ARC-CORE-001`            | Implemented    | Session Runtime v3 receives only runtime-neutral snapshots, pins, semantic bindings, and durable state. It imports no Artifact Store or setup package.                                                                                 | `internal/agentruntime-go/sessionv3/` has no Artifact Store or setup dependency, and `agentruntime-go/workbinding` supplies neutral execution contracts. Setup must retain this boundary while it prepares native domain semantics.                                                                                                   |
| `ARC-DIRECT-001`          | Implemented    | Workflow rejects direct composition for Model, Agent, Tool, MCP, Skill, Service, Activity, and Nested Workflow before ProgramSnapshot creation.                                                                                        | `workflow/closure.go` rejects restricted direct results. `workflowruntime/direct_targets.go` is removed. `sessionv3` has no direct target, plan, payload, semantic-binding, or Tool-origin fields.                                                                                                                                    |
| `ARC-TRANSPORT-001`       | Partial        | Preserve the prepared transport matrix and connect nested published ports to their actual child-local channels.                                                                                                                        | `sessionv3/ports_runtime.go` installs published-port bridges, but private channel names leak into declaration expression lookup and shared mutation paths remain incomplete. See `WF-CHANNEL-003` and `RT-SETTLE-003`.                                                                                                                |
| `ARC-PROVIDER-001`        | Implemented    | A provider SDK Tool is represented only by an owning typed Model plan and never by a local Tool WorkItem or local Tool result.                                                                                                         | `toolruntime/v3.go` prepares `ProviderManagedToolChoice`; `workflow_program.go` lowers it through a single-model child ProgramFrame only.                                                                                                                                                                                             |
| `ARC-ORIGIN-001`          | Implemented    | A Tool WorkItem emitted from LLM output retains immutable effective invocation and selected typed bindings. Target binders never fall back to current Session defaults.                                                                | `sessionv3/workflow_target_facts.go` stores `LLMToolOrigin.Plan`; `skillruntime/v3.go` consumes retained plan selections; Go and MCP binders use retained typed bindings.                                                                                                                                                             |
| `ARC-SCHEDULER-001`       | Partial        | Keep one Runtime due-time owner and complete work launch, deadline arming, restored resource wakeup, and trigger-queue resumption.                                                                                                     | `due_time.go` remains the sole private timer registry, but cleanup-aware callbacks, concurrency-slot wakeup, and trigger-registration reactivation remain incomplete. See `RT-CLEANUP-003`, `RT-BUDGET-003`, and `RT-TRIGGER-003`.                                                                                                    |
| `ARC-STATIC-001`          | Not applicable | Repeated-admission and source-check cleanup is not functional work in this cycle.                                                                                                                                                      | Artifact admission is sufficient for this cycle; no validation-only work is requested.                                                                                                                                                                                                                                                |
| `ARC-COMPAT-001`          | Not applicable | Compatibility naming cleanup is not functional work in this cycle.                                                                                                                                                                     | `PreparationRequest`, `Admission`, and `WorkflowAdmissionRequest` coexist without a demonstrated missing behavior.                                                                                                                                                                                                                    |
| `ARC-V3-001`              | Partial        | Complete artifact-backed conversation configuration, controls, replay, and setup capability forwarding.                                                                                                                                | Conversation controls, setup capability forwarding, and Provider SDK Tool user configuration have implemented components, but collection policy, contribution precedence, active-operation continuation, and owning-domain preparation remain incomplete. See `V3-CONFIG-003`, `V3-CONTINUE-003`, `V3-API-003`, and `ARC-DOMAIN-003`. |
| `ARC-REPLAY-001`          | Partial        | Reproduce each message's retained semantic context and complete assistant/Tool rounds across isolation and Execution boundaries.                                                                                                       | History count, audit exclusion, isolated Agent assistant rounds, prior Execution Tool results, and retained context replay exist. Recovery clears retained bindings before reuse and rebuilds parts of requests from current facts. See `V3-REPLAY-003`.                                                                              |
| `ARC-SKILL-001`           | Partial        | Reconstruct and update Skill state for the correct invocation owner without one parallel Agent replacing another's private session.                                                                                                    | Skill Runtime sessions have Execution and ProgramFrame keys, but LLM active-state reconstruction is Execution-wide and natural owner cleanup is incomplete. See `DOMAIN-SKILL-003`.                                                                                                                                                   |
| `ARC-TOOL-SCOPE-001`      | Implemented    | Standalone Tool selection is limited to built-in Artifact-backed Go and provider SDK Tool declarations. Go Tools remain Tool Runtime work; provider SDK Tools remain typed Model-plan choices.                                         | `tool/api.go` intentionally requires the protected built-in Tool root. `toolcatalog/packages.go` creates both Go and SDK Tool Artifacts.                                                                                                                                                                                              |
| `ARC-PROVIDER-CONFIG-001` | Implemented    | Provider SDK Tool configuration for built-in SDK Tool Artifacts is typed, immutable in the Model dispatch plan, and available to Workflow provider Tool lowering. An owning Model can derive from an explicit Model or selected Agent. | `ToolSelection.UserArguments`, `ToolDispatchPlan.ProviderConfiguration`, `ProviderManagedToolChoice.Configuration`, and `ResolveWorkflowAgentOwnerModel` retain typed configuration and owner Model pins.                                                                                                                             |
| `ARC-MCP-PREPARATION-001` | Implemented    | Workflow preparation resolves current MCP server readiness, discovered Tool operation, policy, and discovery facts before it emits a ProgramSnapshot. MCP Tools remain discovered MCP Server operations, not Tool Artifacts.           | `mcpruntime.WorkflowTargetPreparer` prepares the current operation. `WorkflowMCP` retains server, provider Tool, policy, and discovery facts; dispatch rejects a changed prepared target.                                                                                                                                             |

The owning domain constructs native requests, resolves current domain state,
and converts native results. Session Runtime owns durable orchestration,
checkpointing, retries, waits, cancellation, and settlement. Workflow remains
an orchestration language and must not become a generic native-domain envelope.

| ID                    | Current status | Required end state                                                                                                                                                                                     | Current source evidence                                                                                                                                                                                                                                     |
| --------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ARC-INVOCATION-002`  | Implemented    | Setup lowers supported Workflow Model, Agent, and provider-managed Tool invocation forms into typed runtime fields. Core Runtime does not decode generic Workflow envelopes.                           | `workflowruntime/compiler_invocation.go` lowers supported forms into `WorkflowLLMInvocation`; Runtime evaluates typed templates only.                                                                                                                       |
| `ARC-CALL-POLICY-001` | Partial        | Apply one captured logical call policy consistently to local work, child work, retry, and cleanup.                                                                                                     | Some WorkItems and child-call boundaries retain policy facts, but outer child-call reservation and admission can occur after child native work becomes ready, and terminal accounting is not common to every path. See `RT-BUDGET-003` and `RT-SETTLE-003`. |
| `ARC-PATH-001`        | Implemented    | Preserve declaration identity in admitted descriptors and synthetic execution blocks.                                                                                                                  | `EvaluationBoundary` persists canonical declaration path and logical evaluation time on generated WorkItems and ProgramFrames, including structured child and retry-created boundaries.                                                                     |
| `ARC-V2-COEXIST-001`  | Not applicable | SessionV2 and SessionV3 coexistence removal is outside this cycle. New V3 work must not add a third orchestration path or restore direct capabilities.                                                 | `cmd/agentgo/app.go` composes both runtimes. SessionV2 cleanup and migration remain excluded.                                                                                                                                                               |
| `ARC-START-PORT-001`  | Implemented    | A started call has one authoritative port-delivery boundary. Final output is delivered once, streamed output remains ordered, and independent child execution routes to the true parent channel owner. | Forward-only wrapper frames retain routing facts but suppress duplicate final delivery. Recursive channel ownership resolves inherited parent port plans across independent Executions.                                                                     |
| `ARC-NESTED-PORT-001` | Implemented    | A nested Workflow target with mapped ports must publish compatible root ports. A target with no published ports accepts no mapping.                                                                    | `ProgramFramePortBoundaryPublishedWorkflow` invokes `validateWorkflowFramePortBoundary`, which rejects a mapped target with no published ports.                                                                                                             |
| `ARC-DUP-MCP-001`     | Implemented    | V3 MCP content adaptation has one MCP-domain implementation. Session Runtime receives only converted semantic values.                                                                                  | `mcpruntime.MCPContentText` and `MCPContentsText` are now the shared V3 MCP conversion boundary.                                                                                                                                                            |
| `ARC-OWNERSHIP-002`   | Partial        | Domain owners prepare domain semantics through neutral execution contracts; Session and Workflow setup compose them.                                                                                   | Attachment, Text, Service, and Activity use neutral work-binding contracts, but `sessionv3runtime/llm_plan.go` and `workflowruntime/model_agent_adapter.go` still duplicate source-family contribution interpretation. See `ARC-DOMAIN-003`.                |
| `ARC-DUP-002`         | Partial        | Use one implementation for shared execution semantics and one authoritative immutable effective plan.                                                                                                  | `ModelDispatchPlan` is the intended immutable plan, but `enrichDurableLLMDispatchPlan` copies effective state into owner records, Tool origins deep-copy plans, and equivalent Runtime algorithms remain replicated. See `ARC-DUP-003`.                     |

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
| Durable Session, Execution, ProgramFrame, WorkItem, Attempt, Wait, effect, and trigger state            | Partial        | `sessionv3/` and `sessionstorev3/` persist the durable execution model, but authoritative terminal mutation and deletion commitment remain incomplete. See `RT-SETTLE-003` and `RT-PERSIST-003`.                                                                                        |
| Immutable V3 attachment content admission                                                               | Implemented    | `sessionv3runtime/attachments.go` produces content and digests before message persistence. Semantic labels and owning-package placement remain open in `DOMAIN-ATTACH-003` and `ARC-DOMAIN-003`.                                                                                        |
| Ordered work, state updates, scopes, branching, collection work, repeat, machines, and child boundaries | Partial        | Durable structured state exists, but Repeat condition indices, empty collection settlement, Machine lifetime behavior, and exact evaluation snapshots remain incomplete. See `WF-CONTROL-003` and `WF-CHANNEL-003`.                                                                     |
| Durable Fork and Join intent, explicit trigger registration, and prepared Workflow commands             | Implemented    | Runtime persists typed child intent; direct Workflow start and trigger registration are separate operations; core Workflow commands receive prepared snapshots. Shared lifecycle defects remain tracked separately.                                                                     |
| Timers, events, human work, channel waits, and child-completion waits                                   | Partial        | Runtime owns durable wait and timer facts, but cleanup-aware callbacks, slot wakeup, registration reactivation, and one event envelope remain incomplete. See `RT-CLEANUP-003`, `RT-BUDGET-003`, and `RT-TRIGGER-003`.                                                                  |
| Planning-owned declaration path in runtime descriptors                                                  | Implemented    | WorkItems and child ProgramFrames retain planning-owned declaration paths and `EvaluationBoundary` logical-time facts through normal, retry, structured, cleanup, and recovery execution.                                                                                               |
| Channel permission and port authorization semantics                                                     | Partial        | Canonical declaration paths exist and `ports_runtime.go` maps nested invocations to private channels, but private storage names leak into expression maps and snapshots can be captured at the wrong boundary. See `WF-CHANNEL-003`.                                                    |
| Model and Agent Workflow calls with final and streamed ports                                            | Partial        | Typed lowering and literal-call target-use planning exist, but original-boundary retry capture, typed child failure preservation, active continuation, and composed settlement remain incomplete. See `WF-RETRY-003`, `V3-CONTINUE-003`, and `RT-SETTLE-003`.                           |
| Typed Model-default field surface and Store-owned semantic export                                       | Implemented    | `sessionv3/normal_model_defaults.go`, `modelruntime/runtime_request_patch.go`, and `sessionstorev3.Store.ExportSession` provide this component. Collection inheritance and exact replay remain open in `V3-CONFIG-003` and `V3-REPLAY-003`.                                             |
| Direct capability targets                                                                               | Implemented    | Unsupported by design. `workflow/closure.go` rejects direct composition for Model, Agent, Tool, MCP, Skill, Service, Activity, and Nested Workflow before ProgramSnapshot creation.                                                                                                     |
| Callable capability requirements                                                                        | Implemented    | `CallableInterface.capabilities` is removed from Workflow v1 grammar and all generic capability-fact runtime plumbing is removed.                                                                                                                                                       |
| Provider-managed Tool calls from Workflow                                                               | Implemented    | Provider SDK Tool Artifacts lower to a typed single-model child plan, retain typed user configuration, derive an owner Model from an explicit Model or Agent, and never become local Tool WorkItems.                                                                                    |
| Service and Activity context, ports, and stream delivery                                                | Implemented    | Registered binding transport facts are checked in preparation and again at dispatch; final output remains Runtime-owned.                                                                                                                                                                |
| Target-family transport behavior                                                                        | Partial        | Setup preserves most of the target transport matrix, but nested Workflow expression aliases and common settlement behavior remain incomplete. See `WF-CHANNEL-003` and `RT-SETTLE-003`.                                                                                                 |
| `start` and handoff durable results                                                                     | Partial        | `workflow_call_lifecycle.go` stores a started-call receipt and handle, but queued-message causality and successful handoff completion or relinquish behavior remain incomplete. See `WF-HANDOFF-003`.                                                                                   |
| Started-call port routing and exactly-once final-output delivery                                        | Partial        | Forward-only wrappers preserve routing and suppress duplicate delivery for covered paths, but declaration-visible channel identity and shared mutation paths remain incomplete. See `WF-CHANNEL-003` and `RT-SETTLE-003`.                                                               |
| Child-frame call policy inheritance                                                                     | Partial        | Child-call WorkItems retain policy facts, but the outer logical operation is not reserved before child native work can launch. See `RT-BUDGET-003.c`.                                                                                                                                   |
| Trigger lifecycle separation                                                                            | Implemented    | `StartWorkflow` creates one Execution only. `RegisterWorkflowTriggers` is the only command that creates or changes trigger registrations.                                                                                                                                               |
| Calendar repeated-local-time trigger identity                                                           | Implemented    | Actual fired local slots are retained by local-day slot identity during firing and restore, and successor calculation skips every already-fired repeated local slot.                                                                                                                    |
| Effective invocation configuration for Tool WorkItems emitted by LLM work                               | Implemented    | `LLMToolOrigin` persists the effective dispatch plan, target fingerprint, capability facts, and selected bindings for generated Tool WorkItems.                                                                                                                                         |
| Effects, retries, reconciliation, compensation, and saga execution                                      | Partial        | Durable effect, retry, reconciliation, and compensation structures exist, but source effect translation, MCP certainty, original call capture, causality, and cleanup execution remain incomplete. See `WF-EFFECT-003`, `WF-RETRY-003`, and `RT-CLEANUP-003`.                           |
| Recover persisted ordinary retry delays                                                                 | Implemented    | `resumeRestoredWork` now invokes the Runtime due-time lifecycle for every nonterminal Execution, including `RetryWaiting` WorkItems.                                                                                                                                                    |
| Nested Workflow retry policy semantics                                                                  | Implemented    | `workflow_program.go` permits a valid one-attempt retry declaration on a nested Workflow call.                                                                                                                                                                                          |
| Runtime-owned Workflow logical evaluation time                                                          | Implemented    | `ProgramInterpreter.InitialState` receives Runtime logical time. `WorkflowInterpreter` no longer reads the process clock during initialization.                                                                                                                                         |
| One authoritative Runtime scheduler                                                                     | Partial        | `due_time.go` remains the one private timer registry, but cleanup-aware lifecycle decisions, slot wakeup, and trigger reactivation are incomplete. See `RT-CLEANUP-003`, `RT-BUDGET-003`, and `RT-TRIGGER-003`.                                                                         |
| Semantic reproduction of normal model and Tool continuation                                             | Partial        | Durable activity history, provider output, Tool origin, retained context, and dispatch plans exist, but recovery discards retained bindings before reuse and rebuilds parts of requests from current facts. See `V3-REPLAY-003`.                                                        |
| Approved App context in continuation                                                                    | Implemented    | `consumeContinuationLocked` transfers staged `Execution.Continuation.AppContext` into the next persisted message Activity and clears the staged context in the same checkpoint mutation.                                                                                                |
| Durable semantic Skill lifecycle                                                                        | Partial        | Skill Runtime sessions have Execution and ProgramFrame keys and generated lifecycle Tools serialize in provider order, but active-state selection is Execution-wide and natural terminal cleanup is incomplete. See `DOMAIN-SKILL-003`.                                                 |
| Agent Model system-prompt behavior                                                                      | Implemented    | V3 consumes the selected Agent Model capability `IncludeSystemPrompt` fact during Model patch composition while retaining explicit Session Model prompt precedence.                                                                                                                     |
| Artifact-backed Tool execution and manual triggering                                                    | Implemented    | Standalone built-in Artifact-backed Go and provider SDK Tool declarations are supported. MCP discovered Tools are prepared by MCP Runtime. Skill lifecycle Tools are prepared by Skill Runtime. The standalone Tool scope intentionally excludes arbitrary Tool additions and removals. |
| Normal SessionV2 functional replacement surface                                                         | Partial        | Provider SDK Tool configuration, selected Artifact facts, and several durable V3 components exist, but replay, configuration, continuation, handoff, settlement, persistence, and shared backend defects prevent a complete replacement claim.                                          |
| Cost budget execution                                                                                   | Not applicable | Deferred. Do not implement cost metering in this cycle. Current unsupported behavior remains outside the functional handoff scope.                                                                                                                                                      |

The table above is a component summary, not proof of composed completion.
The inline current task register below controls this cycle.
Each open parent task requires its complete functional behavior, not an
isolated helper or status edit.

`SessionDefaults.ToolSelections` is the canonical V3 Tool-selection collection.
It contains the Artifact selection plus Tool-only user arguments and automatic
execution facts. `ArtifactSelection` remains generic. V3 must not retain a
second Artifact-only `Tools` collection, because two collections produce
ambiguous precedence and duplicate Tool choices.

The implemented rows describe their named components, not immunity from a shared
dispatch defect. Common settlement, recovery, cleanup, and configuration defects
remain in the supplied source. The current register reopens affected older IDs
and records missing tasks that those IDs did not cover. Full Runtime completion
requires every open inline parent task and subtask to be complete.

## Pending functional work

The following work is functional work only. It excludes tests, validation-only
hardening, frontend integration, SessionV2 cleanup and migration, version
changes, release work, and cost metering. Do not add direct capability paths.

| ID                    | Functional task and required result                                                                                                                                                                                                         | Status      | Current source evidence                                                                                                                                                                                                                                    |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `DOMAIN-PROVIDER-001` | Preserve typed configuration for built-in provider SDK Tool Artifacts and derive a pinned owner Model from an explicit Model or selected Agent. Carry immutable facts through normal Model dispatch plans and Workflow provider Tool plans. | Implemented | `toolruntime/provider_sdk.go`, `toolruntime/v3.go`, and `workflowruntime/target_preparation.go` retain `ToolSelection.UserArguments`, provider configuration, and the owner Model. This component does not close unrelated preparation or settlement work. |
| `DOMAIN-MCP-001`      | Prepare the current MCP Server Artifact and selected discovered MCP Tool operation before Workflow ProgramSnapshot creation. Persist prepared domain facts without placing MCP clients or Artifact Store handles in Session Runtime.        | Implemented | `WorkflowTargetPreparer` prepares and snapshots MCP Tool identity, policy, and discovery facts before Workflow lowering. MCP business-effect certainty remains open in `WF-EFFECT-003`.                                                                    |
| `WF-CHILD-001`        | Scope each child-call WorkItem identity to its owning ProgramFrame and logical child instance. Repeats, foreach items, parallel branches, and nested child frames must not collide.                                                         | Implemented | `workflowChildCallWorkID` includes the ProgramFrame identity and logical path.                                                                                                                                                                             |
| `WF-PORT-001`         | Give a started call one authoritative port delivery boundary. Deliver final output once and route independent-child ports through the correct parent Execution and frame.                                                                   | Implemented | Forward-only wrapper frames preserve routing but suppress duplicate final delivery; recursive port ownership reaches the correct parent Execution.                                                                                                         |
| `WF-PORT-002`         | Require a mapped nested Workflow port to be published by the target Workflow and compatible with its declared direction, access, stream behavior, and schema.                                                                               | Implemented | Published Workflow port boundaries are explicit ProgramFrame intent facts and reject undeclared mappings.                                                                                                                                                  |
| `WF-SCHEDULE-001`     | Persist the actual fired local schedule slot and use it to prevent duplicate occurrences for repeated local calendar times during normal operation and recovery.                                                                            | Implemented | Trigger registration, firing, and recovery persist actual local slots before computing successor occurrences.                                                                                                                                              |

### Open functional task and subtask register

Evidence abbreviations: `R` is `internal/agentruntime-go/sessionv3/`; `S` is
`internal/agentruntimesetup/`. Paths below are relative to those directories.
This is the sole current functional completion register. It records the
source-backed status after the supplied Runtime, Workflow preparation, V3
Session setup, Model, Tool, MCP, Skill, attachment, registered-binding, Store
v3, bootstrap, wrapper, Workflow Inventory, and closure review. V2 source is
used only as the backend functional replacement reference.

Statuses describe inspected source behavior, not execution results. Each parent
is complete only when all its subtasks are complete. An implementer can change a
status to `Implemented` only after the complete listed behavior exists in
source, and must replace required-result text with exact source evidence.
Mapped earlier rows must receive the same current status, and obsolete
contradictory completion claims must be removed. Do not add a second completion
ledger. No tests, validation reports, frontend work, migration, V2 removal,
cost metering, or version changes are required by this register.

| ID                     | Functional task or subtask                                                                                                                                       | Status          | Evidence and required completed behavior                                                                                                                                                                                                                                                                                     |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `RT-DISPATCH-001`      | Complete common WorkItem creation and settlement routing.                                                                                                        | Partial         | Several Runtime paths re-acquire authoritative objects, but coordination, manual settlement, binding failure, panic, and child dispatch do not cover every composed terminal path. See `RT-SETTLE-003` and `RT-BUDGET-003`.                                                                                                  |
| `RT-DISPATCH-001.a`    | Route compensation-intent mutation only for a Compensation WorkItem.                                                                                             | Partial         | Compensation-intent mutation is restricted to Compensation WorkItems, but `R/coordination.go` retains wait pointers across durable resolution and related mutation paths require identity reacquisition.                                                                                                                     |
| `RT-DISPATCH-001.b`    | Keep transition mutation attached to the authoritative Session objects.                                                                                          | Partial         | `advanceFrameLocked` and several lifecycle paths re-acquire durable identities, but binding error, port error, panic, invalid wait resolution, manual settlement, and cancellation do not use one complete terminal lifecycle.                                                                                               |
| `RT-SETTLE-003`        | Complete authoritative mutation and terminal propagation.                                                                                                        | Partial         | Existing lifecycle helpers do not cover every composed terminal path.                                                                                                                                                                                                                                                        |
| `RT-SETTLE-003.a`      | Reacquire Session-owned objects after mutation-capable calls.                                                                                                    | Partial         | `R/coordination.go` retains wait pointers across durable resolution; emit, manual result, and reconciliation paths can continue with earlier objects. Mutate by durable Session, Execution, frame, WorkItem, and wait identity after append, rollback, child creation, forwarding, or orchestration advancement.             |
| `RT-SETTLE-003.b`      | Use one complete terminal lifecycle and committed ready-work launch.                                                                                             | Partial         | Binding error, port error, panic, invalid wait-resolution result, manual result, and cancellation paths differ. Settle Attempts and effects, release or transfer claims, wake completion waits, advance owners, settle Executions, persist, then launch all newly ready Session work.                                        |
| `WF-RECOVERY-001`      | Complete reconciliation execution and settlement.                                                                                                                | Partial         | Reconciliation transitions and durable outcome structures exist, but source effect translation, MCP certainty, late Saga confirmation, compensation causality, cleanup execution, and common terminal propagation remain incomplete.                                                                                         |
| `WF-RECOVERY-001.a`    | Apply every returned reconciliation transition.                                                                                                                  | Partial         | `Advance` applies reconciliation-start, intent-settlement, and child-settlement transitions for covered paths, but every terminal path must use the authoritative lifecycle in `RT-SETTLE-003`.                                                                                                                              |
| `WF-RECOVERY-001.b`    | Complete applied, not-applied, and unknown outcomes.                                                                                                             | Partial         | Applied, confirmed-non-application, and unknown structures exist, but actual business-effect certainty and late cleanup behavior remain incomplete. See `WF-EFFECT-003` and `RT-CLEANUP-003`.                                                                                                                                |
| `WF-OUTCOME-003`       | Preserve complete Workflow and conversation terminal meaning.                                                                                                    | Partial         | Some inline, child, failure, and interruption paths omit semantic outcomes or facts.                                                                                                                                                                                                                                         |
| `WF-OUTCOME-003.a`     | Record every actual step boundary and retain complete settled facts.                                                                                             | Partial         | Inline failure can lack the earlier `PendingPath`; return, break, and continue bypass ordinary outcome recording. Record target, receipt, effect, aggregate, native typed error, and declaration facts from the actual operation through failure, cancellation, reconciliation, and child settlement.                        |
| `WF-OUTCOME-003.b`     | Persist interrupted assistant output once without fabricating success.                                                                                           | Partial         | Failed or cancelled Agent LLM work can retain stream text only on the WorkItem; stream merging requires a final LLM payload. Preserve a semantic assistant Activity for usable partial output and retain actual provider-managed observations without inventing a successful local business result.                          |
| `WF-CONTROL-001`       | Complete structured results, exits, state propagation, and aggregate outcomes.                                                                                   | Partial         | Structured child results and aggregate state exist, but Repeat condition indices, empty collection settlement, Machine lifetime behavior, and exact evaluation snapshots remain incomplete. See `WF-CONTROL-003` and `WF-CHANNEL-003`.                                                                                       |
| `WF-CONTROL-001.a`     | Settle control exits and block values exactly once.                                                                                                              | Partial         | Child result wrapping exists, but return, break, continue, and inline failure paths do not all record and settle their actual boundary through the common outcome path. See `WF-OUTCOME-003`.                                                                                                                                |
| `WF-CONTROL-001.b`     | Apply declared state and result semantics.                                                                                                                       | Partial         | Parallel and Foreach aggregate structures exist, but empty collection merge progress, explicit maxima, and Repeat condition scope remain incomplete. See `WF-CONTROL-003`.                                                                                                                                                   |
| `WF-CONTROL-003`       | Complete remaining structured-control value and lifetime behavior.                                                                                               | Partial         | Structured operations exist but the following boundary cases remain functional gaps.                                                                                                                                                                                                                                         |
| `WF-CONTROL-003.a`     | Supply the current Repeat index to `while` and `until`.                                                                                                          | Partial         | Body frames receive an index, but condition evaluation does not install the Repeat's current index. Preserve carried state and result semantics at the same boundary.                                                                                                                                                        |
| `WF-CONTROL-003.b`     | Settle empty captured collections and preserve explicit maxima.                                                                                                  | Partial         | Empty Foreach merge can return idle with unapplied merge progress; non-`all` empty runtime collections can remain pending. Emit required merge work and settle the collection truthfully. Preserve omitted versus explicit zero `maxItems` and `maxIterations` during lowering.                                              |
| `WF-CONTROL-003.c`     | Limit consecutive immediate Machine transitions, not lifetime event count.                                                                                       | Partial         | `Microsteps` increments for selected event transitions and is not reset at genuine suspension. Permit long-running event-driven Machines while retaining the immediate-transition bound.                                                                                                                                     |
| `WF-SCOPE-001`         | Preserve the complete semantic evaluation scope across nested work.                                                                                              | Partial         | `workflowInheritedChildScope` exists, but channel aliases, boundary snapshots, inherited subscription ownership, and Repeat condition scope remain incomplete. See `WF-CHANNEL-003` and `WF-CONTROL-003`.                                                                                                                    |
| `WF-SCOPE-001.a`       | Construct child scopes through one inheritance implementation.                                                                                                   | Partial         | `workflowInheritedChildScope` preserves several inherited facts, but controlling constructs do not consistently apply current Repeat indices and declaration-visible channel aliases.                                                                                                                                        |
| `WF-SCOPE-001.b`       | Capture complete wait-filter and output evaluation facts.                                                                                                        | Partial         | `cloneWorkflowWaitFilter` retains scope and logical time, but `Advance` can evaluate another step or final output after a stale snapshot was captured.                                                                                                                                                                       |
| `WF-CHANNEL-003`       | Preserve invocation-local coordination and declaration-visible evaluation meaning.                                                                               | Partial         | Storage namespaces and expression-visible identities are currently mixed.                                                                                                                                                                                                                                                    |
| `WF-CHANNEL-003.a`     | Keep declared channel names available to expressions after private namespacing.                                                                                  | Partial         | `namespaceWorkflowProgramChannels` changes declaration names while `captureWorkflowMessages` uses those names as CEL map keys. Preserve source aliases such as `messages.review` while private logs remain invocation-local.                                                                                                 |
| `WF-CHANNEL-003.b`     | Capture the authorized channel view at each step and block-output boundary.                                                                                      | Partial         | `Advance` captures once before settlement and `drive` can evaluate another step or final output in the same transition. Capture for the actual declaration boundary; retain the original snapshot for a captured logical call.                                                                                               |
| `WF-CHANNEL-003.c`     | Preserve inherited subscription ownership and consumption identity.                                                                                              | Partial         | Outer subscriptions are copied into children and consumption is keyed to the waiting frame. Retain the originating subscription owner so child-scope changes do not make consumed events available again for that subscription.                                                                                              |
| `WF-PREPARE-003`       | Preserve complete prepared target meaning without unrelated dependencies.                                                                                        | Partial         | Closure indexing exists, but not every transitive selected execution fact is retained.                                                                                                                                                                                                                                       |
| `WF-PREPARE-003.a`     | Retain the complete required immutable source-pin and prepared contribution snapshot.                                                                            | Partial         | `S/workflowruntime/admission.go` retains direct target and requirement facts; Agent dispatch later resolves contributions again. Preserve prepared transitive Model, Agent, Text, Tool, Skill, MCP, Workspace, and nested Workflow meaning through neutral snapshots, without private clients or an Artifact execution API.  |
| `WF-PREPARE-003.b`     | Resolve an owner Model only where provider-managed Tool execution needs it.                                                                                      | Partial         | `admitPlanned` eagerly calls `workflowOwnerModel`. Pure deterministic or registered-binding Workflows must not fail because unrelated Session conversation defaults are unavailable.                                                                                                                                         |
| `WF-CALL-002`          | Complete static and resolver-backed callable selection.                                                                                                          | Implemented     | Static target sets lower only to static members; resolver-backed sets lower only to durable resolver state. Resolver settlement resumes the preserved call. Inventory literal target extraction is fixed in `workflowLiteralTargetName`.                                                                                     |
| `WF-CALL-002.a`        | Lower allowed target sets to one canonical representation.                                                                                                       | Implemented     | Static target sets lower only to static members, and resolver-backed target sets lower only to durable resolver state.                                                                                                                                                                                                       |
| `WF-CALL-002.b`        | Resume the call after accepted resolver membership.                                                                                                              | Implemented     | Resolver settlement resumes the preserved call with persisted membership and selection facts. `workflowLiteralTargetName` extracts literal targets for the planning inventory.                                                                                                                                               |
| `WF-RETRY-002`         | Preserve one logical call through durable retry attempts.                                                                                                        | Partial         | Retry synthetic blocks retain declaration identity, but typed capture occurs after child settlement instead of at the original evaluation boundary. See `WF-RETRY-003`.                                                                                                                                                      |
| `WF-RETRY-002.a`       | Make retry-created blocks executable and source-identifiable.                                                                                                    | Partial         | Synthetic retry blocks retain declaration identity and attempt runtime identities, but they must reuse the original call-boundary snapshot. See `WF-RETRY-003.a`.                                                                                                                                                            |
| `WF-RETRY-002.b`       | Capture callable facts before the retry boundary.                                                                                                                | Partial         | `retryCaptureChildCall` stores typed invocation facts, but it captures after settlement rather than before dispatch or child materialization.                                                                                                                                                                                |
| `WF-RETRY-003`         | Preserve one captured logical call through retry and started execution.                                                                                          | Partial         | Capture helpers exist but do not capture at the original evaluation boundary.                                                                                                                                                                                                                                                |
| `WF-RETRY-003.a`       | Capture evaluated target, membership, input, context, typed messages, instructions, port inputs, key, and logical time before dispatch or child materialization. | Partial         | `retryCaptureChildCall` captures after settlement; started wrappers retain typed templates for later evaluation. Persist the original evaluated facts and reuse them without observing later channels or state.                                                                                                              |
| `WF-RETRY-003.b`       | Apply complete retry selection and effect replay semantics.                                                                                                      | Partial         | `compileWorkflowRetry` drops `match.schema`; replay decisions reject uncertainty before considering replay-safe effects. Preserve declared matching and permit only policy-authorized read, idempotent, keyed, or confirmed-not-applied retries.                                                                             |
| `WF-RETRY-003.c`       | Preserve native typed failure classification through Model and Agent child settlement.                                                                           | Partial         | Generic Agent and single-model failure replacement loses category and cause facts needed by retry and catch. Owning domain conversion supplies actual failure meaning; child orchestration retains it.                                                                                                                       |
| `WF-EFFECT-002`        | Complete effect translation, domain invocation facts, and truthful settlement.                                                                                   | Partial         | Effect structures and invocation metadata exist, but declaration conversion, MCP certainty, causal operation facts, and common settlement remain incomplete. See `WF-EFFECT-003` and `RT-SETTLE-003`.                                                                                                                        |
| `WF-EFFECT-002.a`      | Translate declaration effects and deliver stable invocation metadata.                                                                                            | Partial         | `WorkBinding.Request` receives durable execution, WorkItem, attempt, effect, idempotency, input, context, port, and stream facts, but source `non-idempotent` conversion is incorrect.                                                                                                                                       |
| `WF-EFFECT-002.b`      | Preserve actual business-effect certainty.                                                                                                                       | Partial         | Go transport errors remain uncertain and registered bindings propagate explicit evidence, but `S/mcpruntime/v3.go:Execute` returns uncertainty even for confirmed success.                                                                                                                                                   |
| `WF-EFFECT-003`        | Preserve declaration effects and actual domain business-effect facts.                                                                                            | Partial         | Effect types exist, but source conversion and MCP result certainty are incorrect.                                                                                                                                                                                                                                            |
| `WF-EFFECT-003.a`      | Translate declaration `non-idempotent` to Runtime `nonIdempotent`.                                                                                               | Partial         | `compileWorkflowEffect` and `compileWorkflowStepEffect` compare source strings with Runtime enum strings. Complete the source-to-program mapping for admitted declarations.                                                                                                                                                  |
| `WF-EFFECT-003.b`      | Settle MCP success and failure from actual domain evidence.                                                                                                      | Partial         | `S/mcpruntime/v3.go:Execute` always returns effect uncertainty. Preserve confirmed successful effects, conservative error certainty, receipts, normal Tool continuation, history, reconciliation eligibility, and Saga recording consistently.                                                                               |
| `WF-EFFECT-003.c`      | Retain compensation causality and complete operation facts.                                                                                                      | Partial         | Ordinary ordered calls do not establish equivalent effect dependencies; compensation facts omit promised operation identities. Persist causal dependencies through ordered and structured work and expose original Attempt, target, effect, receipt, and compensation identity without private state.                        |
| `RT-CLEANUP-002`       | Complete cancellation, finalization, reconciliation, and compensation under cleanup allowance.                                                                   | Implemented     | Cleanup WorkItems dispatch after cancellation or timeout. Late confirmed effects release resources, become Saga EffectRecords, and resume pending compensation. Root and child deadline-derived waits use Runtime due-time ownership.                                                                                        |
| `RT-CLEANUP-002`       | Complete cancellation, finalization, reconciliation, and compensation under cleanup allowance.                                                                   | Partial         | Cleanup markers and allowance exist, but ordinary deadline and cancellation decisions still stop eligible cleanup. See `RT-CLEANUP-003`.                                                                                                                                                                                     |
| `RT-CLEANUP-002.a`     | Permit and advance cleanup work after normal cancellation or timeout.                                                                                            | Partial         | Some Cleanup WorkItems remain dispatchable through a cleanup deadline, but binding, retry, resource, emit, stream, and synthetic-frame paths retain ordinary-work decisions.                                                                                                                                                 |
| `RT-CLEANUP-002.b`     | Finish ownership only after eligible settlement and cleanup.                                                                                                     | Partial         | Cancellation preserves a primary outcome for covered paths, but Saga ownership can finish before late confirmed effects, reconciliation, compensation, completion wakeup, and final resource release settle.                                                                                                                 |
| `RT-CLEANUP-003`       | Execute cleanup through every owning lifecycle.                                                                                                                  | Partial         | Cleanup markers and allowance exist, but ordinary deadlines and cancellation decisions still stop eligible cleanup.                                                                                                                                                                                                          |
| `RT-CLEANUP-003.a`     | Carry cleanup classification and its effective deadline through binding, execution, retry, resource waits, emits, streams, and synthetic frames.                 | Partial         | `R/runtime.go`, `coordination.go`, `retry.go`, `runtime_resources.go`, and stream paths retain ordinary-work decisions. Stop normal work while dispatching eligible cleanup through its bounded allowance, including retry wrappers.                                                                                         |
| `RT-CLEANUP-003.b`     | Retain Saga ownership until late external outcomes settle.                                                                                                       | Partial         | `beginSagaCompensation` finishes immediately for an empty initial order, before its outstanding-work check can apply. Keep pending cleanup available for late confirmed effects, reconciliation, compensation, completion wakeup, and final resource release without replacing primary cancellation.                         |
| `WF-PORT-003`          | Connect published nested ports and preserve invocation-local coordination.                                                                                       | Partial         | `ports_runtime.go` installs private nested Workflow channel namespaces and durable forwarding, but private storage names leak into declaration evaluation and shared mutation paths remain incomplete. See `WF-CHANNEL-003` and `RT-SETTLE-003`.                                                                             |
| `WF-PORT-003.a`        | Establish the declared caller-port to child-channel connection.                                                                                                  | Partial         | `preparePublishedWorkflowPortBoundaryLocked` copies staged input values and adds `ChannelForward` records, but declared source aliases must remain available to expressions.                                                                                                                                                 |
| `WF-PORT-003.b`        | Scope channels and launch work woken across Execution boundaries.                                                                                                | Partial         | Child namespaces derive from Execution and ProgramFrame identity, but forwarding and ready-work launch must use the authoritative committed terminal lifecycle.                                                                                                                                                              |
| `WF-MACHINE-002`       | Complete machine rearming and durable transition selection.                                                                                                      | Partial         | Machine waits and durable selection exist, but inherited subscription ownership, one event projection, and consecutive-immediate transition lifetime behavior remain incomplete. See `WF-CHANNEL-003`, `WF-CONTROL-003`, and `RT-TRIGGER-003`.                                                                               |
| `WF-MACHINE-002.a`     | Rearm each consumed wait occurrence without losing other subscriptions.                                                                                          | Partial         | Consumed waits receive identities, but copied child subscriptions lose originating ownership and can make consumed events available again.                                                                                                                                                                                   |
| `WF-MACHINE-002.b`     | Apply declared event and timer semantics.                                                                                                                        | Partial         | `WorkflowMachineProgress` persists ready facts, but event envelopes disagree on `name` versus `type`, and `Microsteps` counts lifetime event transitions.                                                                                                                                                                    |
| `RT-TRIGGER-002`       | Complete trigger-created work launch, queue resumption, and local-slot identity.                                                                                 | Implemented     | Session mutations launch ready trigger work. Recovery and reactivation drain eligible queues. Repeated local schedule slots are retained by local-day slot identity and skipped during successor calculation.                                                                                                                |
| `RT-TRIGGER-002.a`     | Dispatch accepted event-trigger work and resume eligible queues.                                                                                                 | Partial         | Session mutations and recovery launch some ready trigger work, but `SetTriggerRegistrationState` rearms timers without draining eligible queues.                                                                                                                                                                             |
| `RT-TRIGGER-002.b`     | Deduplicate every repeated local calendar occurrence.                                                                                                            | Partial         | Repeated local slots are retained, but trigger, wait, and Machine expressions do not receive one consistent declared event `type`.                                                                                                                                                                                           |
| `RT-TRIGGER-003`       | Resume trigger lifecycle changes and expose one event meaning.                                                                                                   | Partial         | Schedule occurrence state exists, but reactivation and trigger expression envelopes remain incomplete.                                                                                                                                                                                                                       |
| `RT-TRIGGER-003.a`     | Drain eligible queued occurrences on reactivation and apply the intended missed-occurrence policy.                                                               | Partial         | `SetTriggerRegistrationState` rearms timers without draining queues. Persist queue and registration changes before the shared ready-work launch.                                                                                                                                                                             |
| `RT-TRIGGER-003.b`     | Expose the declared event `type` consistently to trigger, wait, and Machine expressions.                                                                         | Partial         | `TriggerEventEnvelope` exposes `name`, while event-wait output supplies `type`. Preserve source, identity, data, correlation, and time through one semantic event projection.                                                                                                                                                |
| `RT-LIMIT-002`         | Complete scoped budgets, concurrency scheduling, claims, and deadlines.                                                                                          | Implemented     | Shared Execution, ProgramFrame, and WorkItem counters enforce scoped limits. Parent-frame concurrency includes descendants; nested Workflow resources are isolated by invocation namespace.                                                                                                                                  |
| `RT-LIMIT-002.a`       | Account for both local limits and shared parent usage.                                                                                                           | Partial         | `ExecutionBudgetUsage` records several shared counters, but structured frame intents do not consistently carry narrowed `step.Limits` and child depth counts synthetic frames.                                                                                                                                               |
| `RT-LIMIT-002.b`       | Apply claims and lifecycle timing to every owning construct.                                                                                                     | Partial         | Claims and namespaces exist, but Binding-to-Waiting, binding failure, and related slot-release paths do not consistently wake budget waiters.                                                                                                                                                                                |
| `RT-BUDGET-003`        | Complete scoped accounting and resource-ready scheduling.                                                                                                        | Partial         | Counters exist, but slot release, structured scope application, and child admission ordering remain incomplete.                                                                                                                                                                                                              |
| `RT-BUDGET-003.a`      | Wake concurrency waiters whenever a counted slot becomes available.                                                                                              | Partial         | `Binding` to `Waiting`, binding failure, and related paths do not consistently wake budget waiters after the state change. A durable wait must not leave otherwise available work idle.                                                                                                                                      |
| `RT-BUDGET-003.b`      | Apply structured step limits and semantic Workflow depth.                                                                                                        | Partial         | Scope, Choose, Parallel, Foreach, Repeat, Machine, and Saga frame intents do not consistently carry `step.Limits`. `childProgramFrameDepth` counts synthetic frames. Apply narrowed construct budgets and distinguish nested Workflow depth from internal frame depth.                                                       |
| `RT-BUDGET-003.c`      | Reserve the outer child-call operation before child native work can launch.                                                                                      | Partial         | Child-call and Fork work are created as separately ready work. Persist the logical outer Attempt, effective policy, accounting, and ownership before permitting its child to dispatch; avoid a child performing work before its outer budget decision.                                                                       |
| `V3-CONFIG-002`        | Complete effective artifact-backed invocation configuration.                                                                                                     | Implemented     | Explicit collection replacement or clearing, contribution ordering, selected Workspace MCP facts, immutable Model dispatch plans, and contribution provenance are retained through execution and recovery.                                                                                                                   |
| `V3-CONFIG-002`        | Complete effective artifact-backed invocation configuration.                                                                                                     | Partial         | Typed fields and dispatch plans exist, but collection policy, contribution precedence, effective Workspace application, provenance, and owning-domain preparation remain incomplete. See `V3-CONFIG-003` and `ARC-DOMAIN-003`.                                                                                               |
| `V3-CONFIG-002.a`      | Preserve contribution ownership and explicit override semantics.                                                                                                 | Partial         | Nil versus non-nil handling does not durably distinguish inherit, replace, and explicit empty clearing. Effective collections are later copied into unrelated owner records.                                                                                                                                                 |
| `V3-CONFIG-002.b`      | Preserve the selected Workspace MCP configuration.                                                                                                               | Partial         | Workspace MCP facts exist, but `appendUniqueMCPSelection` can overwrite earlier explicit selection and Agent-contributed Workspace selection is not consistently applied through the effective value.                                                                                                                        |
| `V3-CONFIG-003`        | Complete inherited and explicit invocation configuration semantics.                                                                                              | Partial         | Typed fields exist, but collection policy and precedence are incomplete.                                                                                                                                                                                                                                                     |
| `V3-CONFIG-003.a`      | Preserve inherit, replace, explicit empty clearing, and supported per-item override intent durably.                                                              | Not implemented | Effective `omitempty` slices and empty-to-nil copying cannot retain the required collection policy. Carry policy separately from the resulting collections so Agent and Workspace inheritance cannot recreate explicitly removed Tools, Skills, MCP, or instructions.                                                        |
| `V3-CONFIG-003.b`      | Apply inherited contributions before more-specific Session and invocation decisions.                                                                             | Partial         | `appendUniqueMCPSelection` overwrites earlier explicit selections; Agent-contributed Workspace selection is not applied through the effective value. Preserve selected MCP operations, resources, templates, prompts, and overrides and consume the effective Workspace.                                                     |
| `V3-CONFIG-003.c`      | Retain accurate contribution ownership and ordering.                                                                                                             | Partial         | Instruction and capability facts are assembled centrally and later copied into unrelated owner records. Preserve actual Agent, Workspace, Session, and invocation provenance and the selected Model system-prompt behavior through domain-owned preparation.                                                                 |
| `V3-HISTORY-002`       | Complete semantic conversation reconstruction.                                                                                                                   | Partial         | Durable history, provider output, Tool results, App context, and dispatch plans exist, but recovery removes retained bindings before reuse and reconstructs parts of requests from current facts. See `V3-REPLAY-003`.                                                                                                       |
| `V3-HISTORY-002.a`     | Reconstruct the exact authorized message and Tool sequence.                                                                                                      | Partial         | Authorized history reconstruction exists for covered paths, but queued handoff rounds and interrupted assistant observations do not retain complete causal semantic activities. See `WF-HANDOFF-003` and `WF-OUTCOME-003`.                                                                                                   |
| `V3-HISTORY-002.b`     | Reuse retained semantic context and provider outcomes.                                                                                                           | Partial         | Recovery retains some plan data, but `R/runtime.go:recoverLocked` clears `item.Binding` before its intended preservation condition and `bindRecoveredLLM` resolves current configuration.                                                                                                                                    |
| `V3-REPLAY-003`        | Reconstruct private execution resources from the exact retained semantic operation.                                                                              | Partial         | A recovered-plan binder exists but the Runtime and request reconstruction paths do not preserve its complete contract.                                                                                                                                                                                                       |
| `V3-REPLAY-003.a`      | Retain semantic LLM bindings while discarding private resources.                                                                                                 | Partial         | `R/runtime.go:recoverLocked` sets `item.Binding = nil` before its intended plan-preservation condition. Preserve the immutable plan for dispatched replay-safe work and the equivalent interrupted preparation path.                                                                                                         |
| `V3-REPLAY-003.b`      | Reproduce captured message order and effective non-secret Model configuration.                                                                                   | Partial         | `bindRecoveredLLM` rebuilds activity selection and resolves current Model and overlay configuration. Use captured semantic message identities and order plus retained effective Provider, Model, capability, instruction, and context facts; resolve credentials and clients privately without current-default substitution. |
| `V3-CONTROL-002`       | Complete normal conversation queuing and automation controls.                                                                                                    | Implemented     | Conversation activities queue while a Fork boundary is pending. Explicit Tool requests bypass automatic Tool holds. Disabling only Tool automation no longer requests continuation.                                                                                                                                          |
| `V3-CONTROL-002`       | Complete normal conversation queuing and automation controls.                                                                                                    | Partial         | Queue and automation-control components exist, but staged continuation is not consumed by the active Agent operation and manual Tool mutations can address historical or unrelated work. See `V3-CONTINUE-003`.                                                                                                              |
| `V3-CONTROL-002.a`     | Queue messages through the pending Fork and active Agent child boundary.                                                                                         | Partial         | Conversation activities queue through a pending Fork, but queued Agent messages retain IDs without each message's completed assistant round, defaults, budget, and isolated causal history.                                                                                                                                  |
| `V3-CONTROL-002.b`     | Separate explicit execution from automatic Tool and continuation policy.                                                                                         | Partial         | Explicit Tool controls exist, but continuation broadcasts across Agent frames and normal Tool mutation can rewrite consumed historical results.                                                                                                                                                                              |
| `V3-CONTINUE-003`      | Complete active-operation continuation and manual Tool lifecycle ownership.                                                                                      | Partial         | The command surface exists, but some commands address a conversation queue or whole Execution instead of the owning operation.                                                                                                                                                                                               |
| `V3-CONTINUE-003.a`    | Consume staged input into the active conversation operation's next Model round.                                                                                  | Partial         | `consumeContinuationLocked` queues through the conversation frame and returns without resuming an Agent waiting for continuation. Carry staged text, attachments, defaults, budget, approved App context, and settled Tool results into the correct next round exactly once.                                                 |
| `V3-CONTINUE-003.b`    | Scope manual retry, skip, completion, and continuation to the current unconsumed operation.                                                                      | Partial         | Normal Tool mutation can rewrite consumed historical results; continuation broadcasts across Agent frames. Retain consumed-result ownership, preserve actual effect certainty, and keep App-originated or unrelated Workflow work outside normal Tool-result continuation.                                                   |
| `DOMAIN-SKILL-002`     | Complete invocation-owned Skill lifecycle state.                                                                                                                 | Partial         | Skill Runtime sessions have Execution and ProgramFrame keys and lifecycle ordering, but LLM active-state reconstruction is Execution-wide and natural owner cleanup is incomplete. See `DOMAIN-SKILL-003`.                                                                                                                   |
| `DOMAIN-SKILL-002.a`   | Own private Skill sessions at the correct invocation scope.                                                                                                      | Partial         | Private sessions have frame keys, but LLM preparation selects active state through an unscoped Execution-wide lookup.                                                                                                                                                                                                        |
| `DOMAIN-SKILL-002.b`   | Apply lifecycle operations in an authoritative semantic order.                                                                                                   | Partial         | Generated lifecycle Tools serialize in provider order, but long-lived conversation Executions retain completed Agent frame sessions because natural terminal paths do not consistently notify cleanup.                                                                                                                       |
| `DOMAIN-SKILL-003`     | Reconstruct and release Skill resources for the exact invocation owner.                                                                                          | Partial         | Private sessions have frame keys, but semantic active-state lookup is still Execution-wide in LLM preparation.                                                                                                                                                                                                               |
| `DOMAIN-SKILL-003.a`   | Select the latest applicable terminal Skill state for the owning frame and enabled set.                                                                          | Partial         | `prepareLLMPlan` calls unscoped `EffectiveActiveSelections`; lookup chooses the latest global result before checking enabled-set compatibility. Filter by semantic owner and enabled pins before selecting the latest result.                                                                                                |
| `DOMAIN-SKILL-003.b`   | Close private Skill sessions at natural frame and Execution completion.                                                                                          | Partial         | Long-lived conversation Executions retain sessions for completed Agent frames; natural terminal paths do not consistently call binding cleanup. Runtime owns lifecycle notification; Skill owns private session release.                                                                                                     |
| `DOMAIN-BINDING-002`   | Complete native Tool binding facts and registered binding waits.                                                                                                 | Implemented     | Bound native Tool calls persist before dispatch. Neutral registered-binding results can request durable waits while private resume state remains outside Session persistence.                                                                                                                                                |
| `DOMAIN-BINDING-002.a` | Persist the owning domain's resolved typed Tool call before invocation.                                                                                          | Implemented     | Bound native Tool calls persist before dispatch without storing clients or authorization tokens.                                                                                                                                                                                                                             |
| `DOMAIN-BINDING-002.b` | Let registered Service and Activity bindings request durable waits.                                                                                              | Implemented     | Neutral registered-binding results can request a durable wait and retain domain-private resume state outside Session persistence.                                                                                                                                                                                            |
| `DOMAIN-ATTACH-003`    | Preserve semantic attachment identity in native Model inputs.                                                                                                    | Partial         | Content snapshots exist, but labels and root-relative directory identity are lost in conversion.                                                                                                                                                                                                                             |
| `DOMAIN-ATTACH-003.a`  | Carry meaningful file labels and directory-relative names into provider content.                                                                                 | Partial         | `v3AttachmentFromContentBlock` stores raw text and `attachmentToContentItem` sends it without its label; directory children use basenames. Preserve semantic display and relative identity without reopening historical files or persisting live native handles.                                                             |
| `V3-SETUP-002`         | Preserve optional backend capabilities through binder composition.                                                                                               | Partial         | Base optional capabilities exist, but the composed Workflow binder forwards App operations while omitting Skill template preparation and instruction preview. See `V3-API-003`.                                                                                                                                              |
| `V3-SETUP-002.a`       | Forward user-message Skill template preparation.                                                                                                                 | Partial         | The base backend contract has the capability, but the composed Workflow binder does not delegate it.                                                                                                                                                                                                                         |
| `V3-SETUP-002.b`       | Forward instruction Skill preview.                                                                                                                               | Partial         | The base backend contract has the capability, but the composed Workflow binder does not preserve it.                                                                                                                                                                                                                         |
| `V3-API-003`           | Preserve optional backend capabilities through final binder composition.                                                                                         | Partial         | Base implementations exist, but Runtime discovers capabilities from the composed binder.                                                                                                                                                                                                                                     |
| `V3-API-003.a`         | Forward user-message Skill template preparation and instruction Skill preview.                                                                                   | Not implemented | The supplied `S/workflowruntime/binder.go` forwards App operations but supplies neither Skill forwarding method. Delegate both capabilities to Base without another rendering implementation.                                                                                                                                |
| `RT-PERSIST-002`       | Complete exact checkpoint and deletion commitment semantics.                                                                                                     | Partial         | Tombstone and checkpoint structures exist, but a committed tombstone can be treated as failed after derived-index deletion failure, and idle long-lived conversations cannot be deleted. See `RT-PERSIST-003`.                                                                                                               |
| `RT-PERSIST-002.a`     | Commit deletion durably before exposing its accepted result.                                                                                                     | Partial         | Tombstones commit before in-memory deletion for covered store failures, but a post-commit index failure restores a live in-memory Session.                                                                                                                                                                                   |
| `RT-PERSIST-002.b`     | Acknowledge the exact semantic mutation, including command-only updates.                                                                                         | Partial         | Checkpoint comparison includes event sequence and snapshot identity, but deletion commitment must explicitly distinguish a committed tombstone from a repairable derived-index failure.                                                                                                                                      |
| `RT-COMMAND-003`       | Replay accepted commands independently of regenerated preparation.                                                                                               | Partial         | Several setup and attachment paths run before durable command receipt lookup.                                                                                                                                                                                                                                                |
| `RT-COMMAND-003.a`     | Retain stable public command identity and return prior accepted receipts before repeating preparation.                                                           | Partial         | Submit, continuation attachment admission, edit, and Workflow wrappers can require old files or sources again. Preserve the original caller payload identity across prepared snapshots without moving Workflow preparation into Runtime.                                                                                     |
| `RT-COMMAND-003.b`     | Keep generated occurrence time outside caller payload identity.                                                                                                  | Partial         | `AcceptTriggerEvent` fills `OccurredAt` before command fingerprinting. Resolve duplicate caller commands from the original payload and reuse persisted generated facts.                                                                                                                                                      |
| `RT-PERSIST-003`       | Complete deletion commitment and normal idle Session deletion.                                                                                                   | Partial         | Live checkpoint commitment handling does not cover the complete deletion path.                                                                                                                                                                                                                                               |
| `RT-PERSIST-003.a`     | Treat a committed tombstone as committed when derived-index deletion fails.                                                                                      | Partial         | `HandleSessionTombstone` returns an ordinary index error after document commitment, causing Runtime to restore a live in-memory Session. Report commitment explicitly, schedule derived repair, and preserve the deletion receipt and barrier.                                                                               |
| `RT-PERSIST-003.b`     | Permit deletion of an idle long-lived normal conversation.                                                                                                       | Partial         | `DeleteSession` rejects every nonterminal Execution although the conversation root intentionally remains live. Retire only an idle conversation root within deletion; retain ownership of actual active Workflow, child, wait, cleanup, and external work.                                                                   |
| `WF-HANDOFF-002`       | Make conversation ownership affect backend routing.                                                                                                              | Partial         | A live handoff record routes normal Session input to an Agent child, but queued-message facts and successful completion remain incomplete. See `WF-HANDOFF-003`.                                                                                                                                                             |
| `WF-HANDOFF-002.a`     | Route subsequent normal Session work to the Session-scoped semantic owner.                                                                                       | Partial         | `conversationMessageExecutionLocked` routes to a live Agent frame, but queued Agent-loop messages retain IDs without completed assistant rounds, immutable defaults, budget, and causal isolated history.                                                                                                                    |
| `WF-HANDOFF-002.b`     | Apply restore and retain behavior to future Session-scoped routing.                                                                                              | Partial         | `recoverConversationOwnershipLocked` handles stale records, but a successful no-Tool handoff has no successful completion or relinquish operation.                                                                                                                                                                           |
| `WF-HANDOFF-003`       | Complete durable multi-message handoff ownership.                                                                                                                | Partial         | Routing exists, but queued message facts and successful completion remain incomplete.                                                                                                                                                                                                                                        |
| `WF-HANDOFF-003.a`     | Preserve each queued message's assistant round, defaults, budget, and causal isolated history.                                                                   | Partial         | Agent-loop queues retain IDs only; the no-Tool handoff branch can discard its constructed assistant Activities when starting queued work. Retain completed Activities and use each accepted message's immutable invocation facts.                                                                                            |
| `WF-HANDOFF-003.b`     | Provide successful backend handoff completion or relinquish behavior.                                                                                            | Not implemented | A successful no-Tool handoff round remains waiting and no successful completion operation is supplied. Add a typed backend lifecycle operation that settles the handoff successfully and applies restore or retain without using cancellation as success.                                                                    |
| `ARC-OWNERSHIP-002`    | Complete domain ownership and dependency separation.                                                                                                             | Partial         | Runtime remains Artifact-independent and consumes neutral plans, but Session and Workflow setup still own duplicated source-family contribution interpretation. See `ARC-DOMAIN-003`.                                                                                                                                        |
| `ARC-OWNERSHIP-002.a`  | Move contribution preparation to its owning domain.                                                                                                              | Partial         | `S/sessionv3runtime/llm_plan.go` interprets capability occurrences, insertion modes, Workspace overrides, and contribution selection; `S/workflowruntime/model_agent_adapter.go` independently selects Agent Models.                                                                                                         |
| `ARC-OWNERSHIP-002.b`  | Separate neutral execution contracts from Artifact-dependent Workflow preparation.                                                                               | Partial         | Attachment and Text retain neutral `agentruntime-go/workbinding` contracts, but attachment materialization and native Model request conversion remain in Session setup rather than their owning domains.                                                                                                                     |
| `ARC-DOMAIN-003`       | Complete owning-domain setup boundaries.                                                                                                                         | Partial         | Core Runtime separation is correct, but Session and Workflow setup still own duplicated domain interpretation.                                                                                                                                                                                                               |
| `ARC-DOMAIN-003.a`     | Put Agent, Workspace, and Text contribution preparation in their owning setup domains.                                                                           | Not implemented | `S/sessionv3runtime/llm_plan.go` interprets capability occurrences, insertion modes, Workspace overrides, and contribution selection. `S/workflowruntime/model_agent_adapter.go` independently selects Agent Models. Supply neutral domain-prepared facts and compose them from Session and Workflow setup.                  |
| `ARC-DOMAIN-003.b`     | Put attachment admission and native Model request conversion in owning setup domains.                                                                            | Partial         | Move V3 attachment materialization from Session setup to Attachment setup. Keep semantic history selection in Session orchestration; Model setup owns provider input, output, and Model-default adaptation. MCP, Skill, Tool, Text, Service, and Activity retain their native semantics.                                     |
| `ARC-DUP-002`          | Remove duplicated execution behavior and replicated effective plans.                                                                                             | Partial         | One intended effective plan exists, but durable data and execution algorithms are still copied across owners. See `ARC-DUP-003`.                                                                                                                                                                                             |
| `ARC-DUP-002.a`        | Consolidate shared algorithms and Runtime lifecycle paths.                                                                                                       | Partial         | WorkItem lifecycle helpers share some paths, but terminal scheduling, settlement, retry and catch matching, error cloning, and operation-input extraction remain replicated.                                                                                                                                                 |
| `ARC-DUP-002.b`        | Retain one canonical effective plan with truthful contribution provenance.                                                                                       | Partial         | Fresh binding has an intended `ModelDispatchPlan`, but enriched copies are stored in Workspace and Agent records, and Tool origins deep-copy complete plans.                                                                                                                                                                 |
| `ARC-DUP-003`          | Remove replicated effective state and duplicated execution algorithms.                                                                                           | Partial         | One intended effective plan exists, but durable data and execution algorithms are still copied across owners.                                                                                                                                                                                                                |
| `ARC-DUP-003.a`        | Retain one canonical effective plan with truthful contribution provenance.                                                                                       | Partial         | `enrichDurableLLMDispatchPlan` copies effective collections into Workspace, Agent, and Agent.Workspace records. Retain owner references and actual contribution facts rather than full duplicated effective plans.                                                                                                           |
| `ARC-DUP-003.b`        | Reference immutable origin plans and attachment content rather than copying them per consumer.                                                                   | Partial         | `LLMToolOriginForWorkItem` deep-copies the full plan into each Tool; message and attachment plans repeat content. Retain an immutable canonical operation-plan reference plus only required Tool-specific facts and one authoritative content snapshot.                                                                      |
| `ARC-DUP-003.c`        | Share Runtime semantic algorithms and one lowered-tree traversal model.                                                                                          | Partial         | Consolidate `evaluate` and `evaluateWithActivation`, retry and catch matching, equivalent error cloning and operation-input extraction, channel and resource namespacing visitors, call lookup, and equivalent retention calculations. Keep terminal scheduling and settlement in the common Runtime lifecycle.              |

Architecture completion must not introduce a second scheduler, a second durable
execution root, native request interpretation in Workflow, or an Artifact Store
execution API. Existing completed SDK Tool configuration, no-published-port
rejection, forward-only start wrappers, and shared V3 MCP output conversion are
retained; their presence does not close the distinct open tasks above.

No item in this document authorizes a version bump, SessionV2 cleanup, frontend
integration, test work, validation-only work, migration, stored-session
conversion, release work, or cost metering.

### Completion and replacement readiness

Every current open parent and subtask must be completed in one substantial
end-to-end functional delivery. A status can change to `Implemented` only when
the complete listed behavior exists in source and the row contains exact source
evidence. Earlier mapped parent and subtask rows must receive the same current
status. This document has one completion authority and must not gain another
competing ledger.

When every current open task is implemented, V3 is intended to replace the
supported Artifact-backed and registered-binding V2 backend behavior while
adding Workflow orchestration. This is a backend functional replacement
decision. It does not claim that application traffic or stored Sessions have
migrated.

Frontend adaptation, stored-session conversion or archival policy, application
traffic switching, and eventual V2 retirement remain separate migration work.
Cost metering remains deferred. These exclusions do not authorize an in-scope
functional task to remain open or any version identifier to change.

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

| Subtask                                                                                               | Status         | Evidence or required result                                                                                                                                                  |
| ----------------------------------------------------------------------------------------------------- | -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Persist Session, Execution, ProgramFrame, WorkItem, Attempt, Wait, effect, channel, and trigger facts | Implemented    | `internal/agentruntime-go/sessionv3/`                                                                                                                                        |
| Enforce checkpoint-before-dispatch behavior                                                           | Implemented    | Runtime checkpoint and dispatch gates                                                                                                                                        |
| Persist and restore Session Store v3 documents and tombstones                                         | Partial        | `internal/sessionstorev3/store.go` persists documents and tombstones, but committed tombstone handling after derived-index failure remains incomplete. See `RT-PERSIST-003`. |
| Verify restore behavior through integration tests                                                     | Not applicable | Tests are outside this implementation cycle.                                                                                                                                 |

### Prepare Workflow execution from a complete static plan

| Subtask                                                             | Status         | Evidence or required result                                                                                                                                                                                      |
| ------------------------------------------------------------------- | -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Separate callable targets from requirements                         | Implemented    | `workflow.Plan.Inventory`                                                                                                                                                                                        |
| Consume planning-owned callable targets and requirements separately | Partial        | `workflowruntime/admission.go` separates direct target and requirement facts, but does not retain every transitive selected source pin and prepared contribution snapshot. See `WF-PREPARE-003`.                 |
| Publish complete static Artifact closure with immutable pins        | Implemented    | `workflow.Plan.Closure` and `workflow/closure.go`                                                                                                                                                                |
| Stop recursive nested Workflow plan resolution during setup         | Implemented    | `workflowruntime/admission.go` consumes closure `WorkflowPlanNode` facts                                                                                                                                         |
| Keep execution preparation outside Session Runtime commands         | Implemented    | Wrapper invokes `workflowruntime.Admission.PrepareWorkflow` or `PrepareWorkflowTriggers` before Runtime receives `ProgramSnapshot`                                                                               |
| Remove Workflow Artifact selections from core Workflow commands     | Implemented    | Core `StartWorkflowCommand` and `RegisterWorkflowTriggersCommand` accept prepared `ProgramSnapshot` only                                                                                                         |
| Rename Workflow admission to execution preparation                  | Not applicable | This is naming cleanup only. It is not functional work in this cycle.                                                                                                                                            |
| Lower one admitted static plan with canonical declaration paths     | Implemented    | `workflowruntime/declaration_paths.go` decorates the lowered tree. Structured child frames, WorkItems, handlers, retries, sagas, reconciliation, and uncertain outcomes retain planning-owned declaration paths. |

### Bind target domains without Workflow leakage

| Subtask                                                                      | Status      | Evidence or required result                                                                                                                                                                    |
| ---------------------------------------------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Bind Model and Agent through existing setup APIs                             | Partial     | Typed Model and Agent invocation adapters exist, but prepared contribution meaning and Agent Model selection are reconstructed through setup paths. See `WF-PREPARE-003` and `ARC-DOMAIN-003`. |
| Bind Go Tools, MCP Tools, and Skill Tools through native arguments           | Implemented | V3 Tool, MCP, and Skill setup executors                                                                                                                                                        |
| Keep Provider SDK Tools provider-managed                                     | Implemented | Workflow SDK Tool targets produce a typed Model child plan and provider-managed choice only                                                                                                    |
| Bind Service and Activity through registered adapters                        | Implemented | Workflow binding registry                                                                                                                                                                      |
| Prepare Service and Activity context and port transport before work creation | Implemented | Binding capability facts are consumed by Workflow preparation and durable target descriptors                                                                                                   |
| Preserve Model and Agent output and stream port plans                        | Partial     | Child intents retain accepted plans, but original call-boundary capture and declaration-visible channel snapshots remain incomplete. See `WF-RETRY-003` and `WF-CHANNEL-003`.                  |
| Prepare target-family transport behavior before WorkItem creation            | Partial     | Preparation validates covered transport facts, but nested channel aliases and common settlement paths remain incomplete. See `WF-CHANNEL-003` and `RT-SETTLE-003`.                             |
| Remove direct capability routing                                             | Implemented | `workflowruntime/direct_targets.go`, direct binder routing, direct bootstrap wiring, direct target payloads, and direct semantic binding fields are removed.                                   |
| Remove generic callable capability framework                                 | Implemented | `WorkflowCapabilityRequirement`, `WorkflowCapabilityFact`, generic fact resolvers, and callable `capabilities` schema/model fields are removed.                                                |
| Lower provider-managed Workflow Tools through typed Model child plans        | Implemented | SDK Tool choices remain inside typed Model plans and never enter local Tool dispatch                                                                                                           |
| Move concrete URL and diff behavior out of generic Workflow setup            | Implemented | `attachmentruntime` and `textruntime` own behavior; Workflow composition only registers bindings                                                                                               |

### Execute Workflow reliability behavior

| Subtask                                                                   | Status         | Evidence or required result                                                                                                                                                                                                        |
| ------------------------------------------------------------------------- | -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Persist state owner identity and schema                                   | Implemented    | Workflow state-owner runtime code                                                                                                                                                                                                  |
| Persist Parallel and Foreach parent-state merges                          | Partial        | Durable StateUpdate merge WorkItems exist, but empty captured collections can leave merge progress unapplied. See `WF-CONTROL-003.b`.                                                                                              |
| Persist failed and uncertain step outcomes                                | Partial        | Preserve complete target, receipt, effect, aggregate, native typed error, and declaration facts through ordinary, reconciliation, cancellation, interruption, and structured settlement. See `WF-OUTCOME-003` and `RT-SETTLE-003`. |
| Make Workflow logical evaluation time Runtime-owned                       | Implemented    | Runtime supplies logical initialization time and `advanceFrameLocked` supplies transition time.                                                                                                                                    |
| Persist retry boundaries and backoff waits                                | Implemented    | `RetryState` remains durable and Runtime re-arms its due-time callback after complete restore.                                                                                                                                     |
| Preserve effects, reconciliation, and compensation                        | Partial        | Complete `WF-RECOVERY-001`, `WF-RETRY-003`, `WF-EFFECT-003`, `RT-CLEANUP-003`, and `RT-SETTLE-003`.                                                                                                                                |
| Verify Service and Activity retry adaptation for every registered binding | Not applicable | Tests and validation work are excluded from this implementation cycle.                                                                                                                                                             |
| Verify cancellation races for every mutating target family                | Not applicable | Tests and validation work are excluded from this implementation cycle.                                                                                                                                                             |

### Complete normal conversation parity

| Subtask                                                               | Status      | Evidence or required result                                                                                                                                                                                                     |
| --------------------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Persist messages, Tool calls, waits, streaming, Apps, and projections | Implemented | Session Runtime v3 and V3 setup binder                                                                                                                                                                                          |
| Persist typed Model defaults and selected Artifact facts              | Implemented | Session defaults and dispatch plans                                                                                                                                                                                             |
| Support artifact-backed normal Session configuration                  | Partial     | Complete inherited collection policy, contribution precedence, effective Workspace application, and contribution provenance in `V3-CONFIG-003` and `ARC-DOMAIN-003`.                                                            |
| Reproduce later artifact-backed conversation behavior                 | Partial     | Complete retained LLM bindings, captured message order, exact non-secret Model configuration, queued handoff rounds, and interrupted output semantics in `V3-REPLAY-003`, `WF-HANDOFF-003`, and `WF-OUTCOME-003`.               |
| Map required artifact-backed Session behavior to V3                   | Partial     | Backend replacement requires the open settlement, cleanup, budget, preparation, effect, retry, replay, configuration, Skill, continuation, handoff, persistence, API composition, attachment, ownership, and duplication tasks. |
| Migrate frontend traffic and stored Sessions                          | Excluded    | Frontend integration and Session v2 migration are outside this audit scope                                                                                                                                                      |

### Remove repeated admission and setup layers

| Removal item                                                | Status         | Required result                                                                                                                                                                                                                  |
| ----------------------------------------------------------- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Remove static declaration rechecks after Workflow admission | Not applicable | This is validation-only cleanup. Do not add it as independent work in this cycle; remove redundant checks only when functional ownership consolidation makes them unnecessary.                                                   |
| Remove repeated source-tree walks during lowering           | Implemented    | Compiler lowers admitted source once; Inventory decorates the lowered runtime tree without another source traversal                                                                                                              |
| Remove the runtime-owned Workflow preparation port          | Implemented    | Setup submits prepared snapshots; Session Runtime persists and interprets them                                                                                                                                                   |
| Collapse duplicate Model dispatch-plan processing           | Partial        | `llmExecutionPlan.activities` has shared preparation, but owner records and Tool origins still copy complete effective plans. See `ARC-DUP-003`.                                                                                 |
| Normalize MCP Tool exposure once                            | Implemented    | `normalizeMCPSelections` persists only `all`, `none`, or `selected` before dispatch-plan construction                                                                                                                            |
| Assign one dispatch-time pin owner per target family        | Implemented    | Go Tool freshness remains in Tool Runtime; MCP freshness is verified by MCP Runtime; generic Workflow wrappers no longer read pins                                                                                               |
| Remove generic Workflow concrete bindings                   | Partial        | URL normalization moved to Attachment Runtime and unified diff moved to Text Runtime, but Session and Workflow setup still duplicate Agent, Workspace, Text, attachment, and Model request interpretation. See `ARC-DOMAIN-003`. |
| Remove runtime reconstruction of declaration permissions    | Partial        | Canonical declaration paths propagate, but private namespacing changes declaration-visible channel keys and snapshots are not always captured at the actual evaluation boundary. See `WF-CHANNEL-003`.                           |

### Verify deployment composition

| Subtask                                                 | Status      | Evidence or required result                                                                                                                                                               |
| ------------------------------------------------------- | ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Verify Session Store v3 checkpoint and restore behavior | Partial     | `internal/sessionstorev3/` persists, restores, indexes, and projects Session v3 documents, but committed tombstone behavior after index failure remains incomplete. See `RT-PERSIST-003`. |
| Verify application bootstrap ordering                   | Implemented | `cmd/agentgo/app.go` constructs dependencies before Store v3 and Runtime v3 restore                                                                                                       |
| Verify runtime shutdown ordering                        | Implemented | `SessionV3Wrapper.close` closes Runtime v3 before Store v3                                                                                                                                |
| Verify Wails command and projection compatibility       | Excluded    | Frontend integration is outside this audit scope                                                                                                                                          |
