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

Workflow Artifact admission and Workflow Artifact planning occur before Session Runtime v3 receives a runtime Program snapshot.

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

## Separate Artifact planning from execution preparation

Workflow Artifact planning owns static source facts.

It returns:

- The admitted Workflow declaration.
- The root Workflow Artifact identity.
- A complete static Artifact closure.
- Static dependency edges.
- Callable target facts.
- Requirement facts.
- Direct-capability evidence.
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

The runtime accepts only a typed `ProgramSnapshot`.

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

Setup owns target transport capability.

A target transport check must validate:

- Context support.
- Input port support.
- Final output port support.
- Streamed output port support.
- Port schema compatibility.
- Channel authorization.
- Current target capability.

Tool-family rules:

| Target family     | Workflow context | Input ports     | Final output port                    | Streamed output port |
| ----------------- | ---------------- | --------------- | ------------------------------------ | -------------------- |
| Go Tool           | Unsupported      | Unsupported     | Supported after confirmed settlement | Unsupported          |
| MCP Tool          | Unsupported      | Unsupported     | Supported after confirmed settlement | Unsupported          |
| Skill Tool        | Unsupported      | Unsupported     | Supported after confirmed settlement | Unsupported          |
| Provider SDK Tool | Model-plan only  | Model-plan only | Provider-managed                     | Provider-managed     |

Service and Activity rules depend on registered binding capability.

Model and Agent rules depend on their typed setup adapter capability.

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

## Validate at the owning boundary

| Validation class                                     | Owner                                    |
| ---------------------------------------------------- | ---------------------------------------- |
| Workflow source grammar and static semantic validity | Workflow Artifact contract               |
| Static Artifact relationship closure                 | Workflow Artifact planning               |
| Runtime source-to-program conversion                 | Workflow setup                           |
| Invocation input validation                          | Workflow setup                           |
| Current Artifact pin and enablement                  | Workflow setup or target-specific binder |
| Direct capability routing                            | Owning setup adapter                     |
| Domain readiness and native request validity         | Owning domain adapter                    |
| Durable Session and execution integrity              | Session Runtime v3                       |
| Session document and tombstone persistence           | Session Store v3                         |
| Wails request transport                              | Wrapper and frontend transport           |

The runtime validates durable runtime objects. It does not revalidate Workflow source grammar.

## Implement and verify the durable runtime

Status values:

| Status      | Meaning                                                                               |
| ----------- | ------------------------------------------------------------------------------------- |
| Implemented | The inspected source contains the behavior. Test execution was not reviewed.          |
| Partial     | The inspected source contains part of the behavior or has an identified boundary gap. |
| Open        | The required behavior is not present in the inspected source.                         |
| Unverified  | Required source or test evidence was not supplied.                                    |

### Persist and recover Session execution state

| Subtask                                                                                               | Status      | Evidence or required result                 |
| ----------------------------------------------------------------------------------------------------- | ----------- | ------------------------------------------- |
| Persist Session, Execution, ProgramFrame, WorkItem, Attempt, Wait, effect, channel, and trigger facts | Implemented | `internal/agentruntime-go/sessionv3/`       |
| Enforce checkpoint-before-dispatch behavior                                                           | Implemented | Runtime checkpoint and dispatch gates       |
| Persist and restore Session Store v3 documents and tombstones                                         | Unverified  | `internal/sessionstorev3/` was not supplied |
| Verify restore behavior through integration tests                                                     | Unverified  | Test evidence was not supplied              |

### Prepare Workflow execution from a complete static plan

| Subtask                                                             | Status      | Evidence or required result                                   |
| ------------------------------------------------------------------- | ----------- | ------------------------------------------------------------- |
| Separate callable targets from requirements                         | Implemented | `workflow.Plan.Inventory`                                     |
| Consume planning-owned callable targets and requirements separately | Implemented | `workflowruntime/admission.go`                                |
| Publish complete static Artifact closure with immutable pins        | Open        | Current public `Plan` does not visibly expose this closure    |
| Stop recursive nested Workflow plan resolution during setup         | Open        | Current admission recursively calls nested Workflow admission |
| Rename Workflow admission to execution preparation                  | Open        | Current public API still uses `WorkflowAdmission`             |
| Convert admitted typed source without repeated static validation    | Partial     | Compiler conversion exists but repeats validation guards      |

### Bind target domains without Workflow leakage

| Subtask                                                               | Status      | Evidence or required result                           |
| --------------------------------------------------------------------- | ----------- | ----------------------------------------------------- |
| Bind Model and Agent through existing setup APIs                      | Implemented | Model and Agent invocation adapter                    |
| Bind Go Tools, MCP Tools, and Skill Tools through native arguments    | Implemented | V3 Tool, MCP, and Skill setup executors               |
| Keep Provider SDK Tools provider-managed                              | Implemented | Provider Tool bindings remain Model-plan choices      |
| Bind Service and Activity through registered adapters                 | Implemented | Workflow binding registry                             |
| Validate Service and Activity port capability before execution starts | Partial     | Current check occurs during WorkItem binding          |
| Route direct capabilities through owning adapters                     | Open        | Current direct capability path fails                  |
| Evaluate callable interface capability requirements                   | Open        | Current code rejects declared capability requirements |
| Move concrete URL and diff behavior out of generic Workflow setup     | Open        | Built-in bindings remain in `workflowruntime`         |

### Execute Workflow reliability behavior

| Subtask                                                                   | Status      | Evidence or required result                 |
| ------------------------------------------------------------------------- | ----------- | ------------------------------------------- |
| Persist state owner identity and schema                                   | Implemented | Workflow state-owner runtime code           |
| Persist Parallel and Foreach parent-state merges                          | Implemented | Durable StateUpdate merge WorkItems         |
| Persist failed and uncertain step outcomes                                | Implemented | Workflow outcome runtime code               |
| Persist retry boundaries and backoff waits                                | Implemented | Retry ProgramFrame implementation           |
| Preserve effects, reconciliation, and compensation                        | Implemented | Session Runtime v3 effect and Workflow code |
| Verify Service and Activity retry adaptation for every registered binding | Unverified  | Binding-level tests are required            |
| Verify cancellation races for every mutating target family                | Unverified  | Cross-domain tests are required             |

### Complete normal conversation parity

| Subtask                                                               | Status      | Evidence or required result                                      |
| --------------------------------------------------------------------- | ----------- | ---------------------------------------------------------------- |
| Persist messages, Tool calls, waits, streaming, Apps, and projections | Implemented | Session Runtime v3 and V3 setup binder                           |
| Persist typed Model defaults and selected Artifact facts              | Implemented | Session defaults and dispatch plans                              |
| Support full Session v2 configuration and direct capability surface   | Partial     | V3 configuration surface remains smaller                         |
| Map every V2 command and read to V3                                   | Partial     | Wrapper has broad coverage, but no complete parity ledger exists |
| Migrate frontend traffic and stored Sessions                          | Open        | Migration and frontend evidence was not supplied                 |

### Correct runtime validation and duplicate logic

| Subtask                                                             | Status  | Evidence or required result                         |
| ------------------------------------------------------------------- | ------- | --------------------------------------------------- |
| Validate every Wait request and status                              | Open    | Correct `Wait.Validate`                             |
| Remove duplicate Model dispatch-plan validation                     | Open    | Consolidate plan validation                         |
| Normalize MCP Tool exposure before plan validation                  | Open    | Treat empty exposure as `all` consistently          |
| Give each target family one dispatch-time pin verification owner    | Partial | Direct Tool and MCP verification overlap            |
| Remove source-language validation duplication from setup conversion | Open    | Retain only conversion and runtime integrity checks |

### Verify deployment composition

| Subtask                                                 | Status     | Evidence or required result                     |
| ------------------------------------------------------- | ---------- | ----------------------------------------------- |
| Verify Session Store v3 checkpoint and restore behavior | Unverified | Store source and tests required                 |
| Verify application bootstrap ordering                   | Unverified | `cmd/agentgo/app.go` required                   |
| Verify runtime shutdown ordering                        | Unverified | Bootstrap and integration tests required        |
| Verify Wails command and projection compatibility       | Unverified | Frontend and wrapper integration tests required |
