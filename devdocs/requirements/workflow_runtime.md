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
- `session.DirectCapabilitySelection` is a legacy SessionV2 representation. It is not a SessionV3
  replacement requirement.
- Do not add direct Tool, Skill, Model, Agent, MCP, Service, Activity, or Workflow capability paths.
- Inline Text remains content or declaration data. It is not a direct executable capability.

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

| Workflow target   | Runtime representation          | Setup owner                    |
| ----------------- | ------------------------------- | ------------------------------ |
| Model             | Child single-model ProgramFrame | Model and Session setup        |
| Agent             | Child Agent-loop ProgramFrame   | Agent and Model setup          |
| Tool Artifact     | Tool WorkItem                   | Tool runtime                   |
| MCP Tool          | Tool WorkItem                   | MCP runtime                    |
| Skill Tool        | Tool WorkItem                   | Skill runtime                  |
| Provider SDK Tool | Provider-managed Tool choice    | Model plan                     |
| Service           | Service WorkItem                | Registered Service adapter     |
| Activity          | Activity WorkItem               | Registered Activity adapter    |
| Nested Workflow   | Child Workflow ProgramFrame     | Workflow execution preparation |

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

| ID                  | Current status | Required end state before marking `Implemented`                                                                                                                                            | Current source evidence                                                                                                                                                                            |
| ------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ARC-CORE-001`      | Implemented    | Session Runtime v3 receives only runtime-neutral snapshots, pins, semantic bindings, and durable state. It imports no Artifact Store or setup package.                                     | `internal/agentruntime-go/sessionv3/` has no Artifact Store or setup dependency.                                                                                                                   |
| `ARC-DIRECT-001`    | Implemented    | Workflow rejects direct composition for Model, Agent, Tool, MCP, Skill, Service, Activity, and Nested Workflow before ProgramSnapshot creation.                                            | `workflow/closure.go` rejects restricted direct results. `workflowruntime/direct_targets.go` is removed. `sessionv3` has no direct target, plan, payload, semantic-binding, or Tool-origin fields. |
| `ARC-TRANSPORT-001` | Implemented    | Preparation owns the target transport matrix before WorkItem creation. Unsupported context, input-port, final-output, and streamed-output behavior cannot be silently accepted or ignored. | `workflowruntime/target_transport.go` consumes call context uses, target context-form facts, call modes, and port facts before ProgramSnapshot creation.                                           |
| `ARC-PROVIDER-001`  | Implemented    | A provider SDK Tool is represented only by an owning typed Model plan and never by a local Tool WorkItem or local Tool result.                                                             | `toolruntime/v3.go` prepares `ProviderManagedToolChoice`; `workflow_program.go` lowers it through a single-model child ProgramFrame only.                                                          |
| `ARC-ORIGIN-001`    | Implemented    | A Tool WorkItem emitted from LLM output retains immutable effective invocation and selected typed bindings. Target binders never fall back to current Session defaults.                    | `sessionv3/workflow_target_facts.go` stores `LLMToolOrigin.Plan`; `skillruntime/v3.go` consumes retained plan selections; Go and MCP binders use retained typed bindings.                          |
| `ARC-SCHEDULER-001` | Implemented    | One Runtime-owned due-time lifecycle registers, restores, fires, and cancels wait, retry, resource, execution, cleanup, and trigger due-time facts.                                        | `due_time.go` owns the private registry. Direct, event-triggered, scheduled-triggered, and restored Executions enter the same execution due-time arming path.                                      |
| `ARC-STATIC-001`    | Not applicable | Repeated-admission and source-check cleanup is not functional work in this cycle.                                                                                                          | Artifact admission is sufficient for this cycle; no validation-only work is requested.                                                                                                             |
| `ARC-COMPAT-001`    | Not applicable | Compatibility naming cleanup is not functional work in this cycle.                                                                                                                         | `PreparationRequest`, `Admission`, and `WorkflowAdmissionRequest` coexist without a demonstrated missing behavior.                                                                                 |
| `ARC-V3-001`        | Implemented    | SessionV3 reproduces required artifact-backed conversation behavior without relying on SessionV2 runtime behavior.                                                                         | `sessionv3runtime/inputs.go` reconstructs ordered completed assistant Tool-call and Tool-result facts. `normal_session.go` consumes approved App context into one durable semantic message.        |
| `ARC-REPLAY-001`    | Implemented    | Completed model, Tool, context, and message facts reproduce later conversation behavior without storing live runtime state.                                                                | `LLMOutput.ProtocolOutputs`, linked completed Tool WorkItems, Tool call identity, and persisted App-context message facts rebuild provider input after recovery.                                   |
| `ARC-SKILL-001`     | Implemented    | Semantic enabled and active Skill state is durable or deterministically reconstructable, while Skill Runtime handles remain private.                                                       | `skillruntime/v3.go` recreates private sessions from retained pinned selections, records active semantic state on Skill Tool output, and hydrates rules plus `SkillsPrompt` before each LLM bind.  |

The owning domain constructs native requests, resolves current domain state,
and converts native results. Session Runtime owns durable orchestration,
checkpointing, retries, waits, cancellation, and settlement. Workflow remains
an orchestration language and must not become a generic native-domain envelope.

| `ARC-INVOCATION-002` | Implemented | Setup lowers supported Workflow Model, Agent, and provider-managed Tool invocation forms into typed runtime fields. Core Runtime does not decode generic Workflow envelopes. | `workflowruntime/compiler_invocation.go` lowers supported forms into `WorkflowLLMInvocation`; Runtime evaluates typed templates only. |
| `ARC-CALL-POLICY-001` | Implemented | One lowering path applies call-step limits, claims, and effect facts consistently to local WorkItems and child-frame target calls. | `WorkflowChildCall` is a durable outer invocation boundary. It retains effect, effect-plan, receipt, attempt, child completion, claim, budget, port, target, and declaration-path facts without a synthetic Tool result. |
| `ARC-PATH-001` | Implemented | Every child ProgramFrame and every Workflow outcome retains the planning-owned declaration path independently from runtime identity. | Structured, machine, Saga, reconciliation, terminal-handler, retry, Fork, Join, and uncertainty paths retain the declaration path without source reparsing. |
| `ARC-V2-COEXIST-001` | Not applicable | SessionV2 and SessionV3 coexistence removal is outside this cycle. New V3 work must not add a third orchestration path or restore direct capabilities. | `cmd/agentgo/app.go` composes both runtimes. SessionV2 cleanup and migration remain excluded. |

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

| Functional capability                                                                                   | Status         | Current evidence or required result                                                                                                                                                                                              |
| ------------------------------------------------------------------------------------------------------- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Durable Session, Execution, ProgramFrame, WorkItem, Attempt, Wait, effect, and trigger state            | Implemented    | `sessionv3/` and `sessionstorev3/` persist the durable execution model                                                                                                                                                           |
| Ordered work, state updates, scopes, branching, collection work, repeat, machines, and child boundaries | Implemented    | Structured settlement clears the active parent boundary, records one outcome, advances once, and resumes the parent for Scope, Choose, Parallel, Foreach, Repeat, machine, Saga, and state-merge paths.                          |
| Timers, events, human work, channel waits, and child-completion waits                                   | Implemented    | coordination, wait, timer, and trigger runtime paths                                                                                                                                                                             |
| Planning-owned declaration path in runtime descriptors                                                  | Implemented    | Lowered blocks, child ProgramFrames, Fork and Join WorkItems, and all Workflow outcome paths retain canonical declaration paths independently from runtime identities.                                                           |
| Channel permission and port authorization semantics                                                     | Implemented    | Planning declaration paths propagate to child frames; setup validates target transport before WorkItem creation and Runtime owns durable channel delivery.                                                                       |
| Model and Agent Workflow calls with final and streamed ports                                            | Implemented    | Setup lowers explicit message, instruction, and defaults forms into typed invocation templates. Runtime stages accepted ports on child frames and does not decode generic request envelopes.                                     |
| Direct capability targets                                                                               | Implemented    | Unsupported by design. `workflow/closure.go` rejects direct composition for Model, Agent, Tool, MCP, Skill, Service, Activity, and Nested Workflow before ProgramSnapshot creation.                                              |
| Callable capability requirements                                                                        | Implemented    | `CallableInterface.capabilities` is removed from Workflow v1 grammar and all generic capability-fact runtime plumbing is removed.                                                                                                |
| Provider-managed Tool calls from Workflow                                                               | Implemented    | Provider SDK Tool Artifacts lower to `ProviderManagedToolChoice` on a typed single-model child plan and never to local Tool WorkItems.                                                                                           |
| Service and Activity context, ports, and stream delivery                                                | Implemented    | Registered binding transport facts are checked in preparation and again at dispatch; final output remains Runtime-owned.                                                                                                         |
| Target-family transport behavior                                                                        | Implemented    | Preparation compares planning-owned context uses, modes, and port facts with prepared target transport capabilities before ProgramSnapshot creation.                                                                             |
| `start` and handoff durable results                                                                     | Implemented    | `workflow_call_lifecycle.go` stores the already-created `workflow-start` receipt, handle, target, and declaration path on the parent step outcome before child creation.                                                         |
| Child-frame call policy inheritance                                                                     | Implemented    | `WorkflowChildCall` persists the outer call effect boundary for nested Workflow, Model, Agent, provider Tool, and handoff children. `start` retains its internal child call effect policy.                                       |
| Trigger lifecycle separation                                                                            | Implemented    | `StartWorkflow` creates one Execution only. `RegisterWorkflowTriggers` is the only command that creates or changes trigger registrations.                                                                                        |
| Effective invocation configuration for Tool WorkItems emitted by LLM work                               | Implemented    | `LLMToolOrigin` persists the effective dispatch plan, target fingerprint, capability facts, and selected bindings for generated Tool WorkItems.                                                                                  |
| Effects, retries, reconciliation, compensation, and saga execution                                      | Implemented    | Local WorkItems and durable child-call boundaries retain effect, attempt, confirmation, uncertainty, reconciliation, compensation, and retry-safety facts without synthesizing local Tool results.                               |
| Recover persisted ordinary retry delays                                                                 | Implemented    | `resumeRestoredWork` now invokes the Runtime due-time lifecycle for every nonterminal Execution, including `RetryWaiting` WorkItems.                                                                                             |
| Nested Workflow retry policy semantics                                                                  | Implemented    | `workflow_program.go` permits a valid one-attempt retry declaration on a nested Workflow call.                                                                                                                                   |
| Runtime-owned Workflow logical evaluation time                                                          | Implemented    | `ProgramInterpreter.InitialState` receives Runtime logical time. `WorkflowInterpreter` no longer reads the process clock during initialization.                                                                                  |
| One authoritative Runtime scheduler                                                                     | Implemented    | `due_time.go` owns one private due-time registry. Scheduled-trigger-created Executions now arm execution due-time facts after durable trigger-start persistence.                                                                 |
| Semantic reproduction of normal model and Tool continuation                                             | Implemented    | V3 replays durable provider output unions for prior assistant rounds and emits retained Tool call identity, provider call identity, choice identity, text, structured output, image, and file content into the next model input. |
| Approved App context in continuation                                                                    | Implemented    | `consumeContinuationLocked` transfers staged `Execution.Continuation.AppContext` into the next persisted message Activity and clears the staged context in the same checkpoint mutation.                                         |
| Durable semantic Skill lifecycle                                                                        | Implemented    | Completed Skill Tool output persists enabled and active pinned Skill selections. V3 reconstructs private Skill Runtime sessions after recovery and hydrates Skill rules plus `SkillsPrompt` before each model invocation.        |
| Agent Model system-prompt behavior                                                                      | Implemented    | V3 consumes the selected Agent Model capability `IncludeSystemPrompt` fact during Model patch composition while retaining explicit Session Model prompt precedence.                                                              |
| Artifact-backed Tool execution and manual triggering                                                    | Implemented    | Artifact-backed Go, provider SDK, MCP, and Skill Tool paths exist. `ExecuteToolCalls` and `ExecuteWorkItems` request ready manual work.                                                                                          |
| Normal SessionV2 functional replacement surface                                                         | Implemented    | Within the backend scope, V3 persists normal messages, Tools, Apps, approved App context, continuation, recovery facts, and projections. Frontend migration and SessionV2 removal remain excluded.                               |
| Cost budget execution                                                                                   | Not applicable | Deferred. Do not implement cost metering in this cycle. Current unsupported behavior remains outside the functional handoff scope.                                                                                               |

The table above is the authoritative functional status register for this
implementation cycle. Every `Not implemented` and `Partial` row is a complete behavior,
not a request for an isolated small edit. Implementers must update the row
status and evidence after the full behavior is complete.

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

| Subtask                                                             | Status         | Evidence or required result                                                                                                                  |
| ------------------------------------------------------------------- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Separate callable targets from requirements                         | Implemented    | `workflow.Plan.Inventory`                                                                                                                    |
| Consume planning-owned callable targets and requirements separately | Implemented    | `workflowruntime/admission.go`                                                                                                               |
| Publish complete static Artifact closure with immutable pins        | Implemented    | `workflow.Plan.Closure` and `workflow/closure.go`                                                                                            |
| Stop recursive nested Workflow plan resolution during setup         | Implemented    | `workflowruntime/admission.go` consumes closure `WorkflowPlanNode` facts                                                                     |
| Keep execution preparation outside Session Runtime commands         | Implemented    | Wrapper invokes `workflowruntime.Admission.PrepareWorkflow` or `PrepareWorkflowTriggers` before Runtime receives `ProgramSnapshot`           |
| Remove Workflow Artifact selections from core Workflow commands     | Implemented    | Core `StartWorkflowCommand` and `RegisterWorkflowTriggersCommand` accept prepared `ProgramSnapshot` only                                     |
| Rename Workflow admission to execution preparation                  | Not applicable | This is naming cleanup only. It is not functional work in this cycle.                                                                        |
| Lower one admitted static plan with canonical declaration paths     | Partial        | Lowering copies Inventory paths, but structured child ProgramFrames and several structured, uncertain, and Saga outcomes do not retain them. |

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

| Subtask                                                                   | Status         | Evidence or required result                                                                     |
| ------------------------------------------------------------------------- | -------------- | ----------------------------------------------------------------------------------------------- |
| Persist state owner identity and schema                                   | Implemented    | Workflow state-owner runtime code                                                               |
| Persist Parallel and Foreach parent-state merges                          | Implemented    | Durable StateUpdate merge WorkItems                                                             |
| Persist failed and uncertain step outcomes                                | Implemented    | Workflow outcome runtime code                                                                   |
| Make Workflow logical evaluation time Runtime-owned                       | Implemented    | Runtime supplies logical initialization time and `advanceFrameLocked` supplies transition time. |
| Persist retry boundaries and backoff waits                                | Implemented    | `RetryState` remains durable and Runtime re-arms its due-time callback after complete restore.  |
| Preserve effects, reconciliation, and compensation                        | Implemented    | Session Runtime v3 effect and Workflow code                                                     |
| Verify Service and Activity retry adaptation for every registered binding | Not applicable | Tests and validation work are excluded from this implementation cycle.                          |
| Verify cancellation races for every mutating target family                | Not applicable | Tests and validation work are excluded from this implementation cycle.                          |

### Complete normal conversation parity

| Subtask                                                               | Status      | Evidence or required result                                                                                                                                |
| --------------------------------------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Persist messages, Tool calls, waits, streaming, Apps, and projections | Implemented | Session Runtime v3 and V3 setup binder                                                                                                                     |
| Persist typed Model defaults and selected Artifact facts              | Implemented | Session defaults and dispatch plans                                                                                                                        |
| Support artifact-backed normal Session configuration                  | Implemented | V3 defaults and setup bind Artifact-backed Model, Agent, Workspace, Tool, Skill, and MCP selections.                                                       |
| Reproduce later artifact-backed conversation behavior                 | Implemented | Provider input rebuilds from durable assistant protocol output, linked completed Tool output, immutable Tool identity, and persisted approved App context. |
| Map required artifact-backed Session behavior to V3                   | Implemented | The backend normal-conversation replacement scope is implemented. SessionV2 frontend migration, stored-session conversion, and deletion remain excluded.   |
| Migrate frontend traffic and stored Sessions                          | Excluded    | Frontend integration and Session v2 migration are outside this audit scope                                                                                 |

### Remove repeated admission and setup layers

| Removal item                                                | Status         | Required result                                                                                                                                                                  |
| ----------------------------------------------------------- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Remove static declaration rechecks after Workflow admission | Not applicable | This is validation-only cleanup. Do not add it as independent work in this cycle; remove redundant checks only when functional ownership consolidation makes them unnecessary.   |
| Remove repeated source-tree walks during lowering           | Implemented    | Compiler lowers admitted source once; Inventory decorates the lowered runtime tree without another source traversal                                                              |
| Remove the runtime-owned Workflow preparation port          | Implemented    | Setup submits prepared snapshots; Session Runtime persists and interprets them                                                                                                   |
| Collapse duplicate Model dispatch-plan processing           | Implemented    | `llmExecutionPlan.activities` is prepared once and reused for provider input construction and durable dispatch-plan facts; generated Tool execution reads retained origin facts. |
| Normalize MCP Tool exposure once                            | Implemented    | `normalizeMCPSelections` persists only `all`, `none`, or `selected` before dispatch-plan construction                                                                            |
| Assign one dispatch-time pin owner per target family        | Implemented    | Go Tool freshness remains in Tool Runtime; MCP freshness is verified by MCP Runtime; generic Workflow wrappers no longer read pins                                               |
| Remove generic Workflow concrete bindings                   | Implemented    | URL normalization moved to Attachment Runtime and unified diff moved to Text Runtime; Workflow composition only registers them                                                   |
| Remove runtime reconstruction of declaration permissions    | Partial        | Channel authorization uses declaration paths, but child ProgramFrames and several step outcomes omit their declaration path                                                      |

### Verify deployment composition

| Subtask                                                 | Status      | Evidence or required result                                                               |
| ------------------------------------------------------- | ----------- | ----------------------------------------------------------------------------------------- |
| Verify Session Store v3 checkpoint and restore behavior | Implemented | `internal/sessionstorev3/` persists, restores, indexes, and projects Session v3 documents |
| Verify application bootstrap ordering                   | Implemented | `cmd/agentgo/app.go` constructs dependencies before Store v3 and Runtime v3 restore       |
| Verify runtime shutdown ordering                        | Implemented | `SessionV3Wrapper.close` closes Runtime v3 before Store v3                                |
| Verify Wails command and projection compatibility       | Excluded    | Frontend integration is outside this audit scope                                          |
