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

Every LLM WorkItem retains its effective invocation configuration and
capability facts. Tool WorkItems created from that LLM result retain the
immutable origin facts required by their target binders. A binder must not
replace those facts with current Session defaults when it evaluates or executes
the generated Tool WorkItem.

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

This ledger defines implementation order for Workflow and Session Runtime
functional completion.

| Functional capability                                                                                   | Status      | Current evidence or required result                                                           |
| ------------------------------------------------------------------------------------------------------- | ----------- | --------------------------------------------------------------------------------------------- |
| Durable Session, Execution, ProgramFrame, WorkItem, Attempt, Wait, effect, and trigger state            | Implemented | `sessionv3/` and `sessionstorev3/` persist the durable execution model                        |
| Ordered work, state updates, scopes, branching, collection work, repeat, machines, and child boundaries | Implemented | `workflow_program.go`, `workflow_structured.go`, and fork/join runtime paths                  |
| Timers, events, human work, channel waits, and child-completion waits                                   | Implemented | coordination, wait, timer, and trigger runtime paths                                          |
| Planning-owned declaration path in runtime descriptors                                                  | Open        | compiled Program, WorkItems, and child frames retain only runtime path identities             |
| Channel permission and port authorization semantics                                                     | Partial     | runtime compares permissions to runtime paths instead of planning-owned declaration paths     |
| Model and Agent Workflow calls with final and streamed ports                                            | Partial     | typed invocation exists; accepted child port plans are not propagated                         |
| Direct capability targets                                                                               | Open        | planning retains direct evidence; execution preparation accepts only source-backed targets    |
| Callable capability requirements                                                                        | Open        | setup has no evaluator that converts declared requirements to current domain capability facts |
| Provider-managed Tool calls from Workflow                                                               | Open        | execution preparation rejects the call instead of lowering a typed Model child plan           |
| Service and Activity context, ports, and stream delivery                                                | Partial     | bindings exist; complete transport capability preparation is absent                           |
| `start` and handoff durable results                                                                     | Partial     | child ownership exists; `start` receipts are not retained in step outcomes                    |
| Trigger lifecycle separation                                                                            | Partial     | event and schedule handling exists; direct Workflow start also registers triggers             |
| Effective invocation configuration for Tool WorkItems emitted by LLM work                               | Open        | generated Tool WorkItems do not retain their origin LLM invocation configuration              |
| Effects, retries, reconciliation, compensation, and saga execution                                      | Implemented | durable effect, retry, reconciliation, and compensation runtime paths                         |

## Validate at the owning boundary

| Validation class                                                | Owner                                    |
| --------------------------------------------------------------- | ---------------------------------------- |
| Workflow source grammar and static semantic validity            | Workflow Artifact contract               |
| Static Artifact relationship closure                            | Workflow Artifact planning               |
| Workflow execution preparation and source-to-program conversion | Workflow setup                           |
| Invocation input validation                                     | Workflow setup                           |
| Current Artifact pin and enablement                             | Workflow setup or target-specific binder |
| Direct capability routing                                       | Owning setup adapter                     |
| Domain readiness and native request validity                    | Owning domain adapter                    |
| Durable Session and execution integrity                         | Session Runtime v3                       |
| Session document and tombstone persistence                      | Session Store v3                         |
| Wails request transport                                         | Wrapper and frontend transport           |

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

| Subtask                                                             | Status      | Evidence or required result                                                                                                        |
| ------------------------------------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Separate callable targets from requirements                         | Implemented | `workflow.Plan.Inventory`                                                                                                          |
| Consume planning-owned callable targets and requirements separately | Implemented | `workflowruntime/admission.go`                                                                                                     |
| Publish complete static Artifact closure with immutable pins        | Implemented | `workflow.Plan.Closure` and `workflow/closure.go`                                                                                  |
| Stop recursive nested Workflow plan resolution during setup         | Implemented | `workflowruntime/admission.go` consumes closure `WorkflowPlanNode` facts                                                           |
| Keep execution preparation outside Session Runtime commands         | Implemented | Wrapper invokes `workflowruntime.Admission.PrepareWorkflow` or `PrepareWorkflowTriggers` before Runtime receives `ProgramSnapshot` |
| Remove Workflow Artifact selections from core Workflow commands     | Implemented | Core `StartWorkflowCommand` and `RegisterWorkflowTriggersCommand` accept prepared `ProgramSnapshot` only                           |
| Rename Workflow admission to execution preparation                  | Partial     | `PreparationRequest` and preparation methods are authoritative; the compatibility type name remains `Admission`                    |
| Lower one admitted static plan with canonical declaration paths     | Implemented | `workflowruntime/declaration_paths.go` copies Inventory paths into lowered runtime descriptors                                     |

### Bind target domains without Workflow leakage

| Subtask                                                                      | Status      | Evidence or required result                                                                              |
| ---------------------------------------------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------- |
| Bind Model and Agent through existing setup APIs                             | Implemented | Model and Agent invocation adapter                                                                       |
| Bind Go Tools, MCP Tools, and Skill Tools through native arguments           | Implemented | V3 Tool, MCP, and Skill setup executors                                                                  |
| Keep Provider SDK Tools provider-managed                                     | Implemented | Provider Tool bindings remain Model-plan choices                                                         |
| Bind Service and Activity through registered adapters                        | Implemented | Workflow binding registry                                                                                |
| Prepare Service and Activity context and port transport before work creation | Partial     | binding capability is discovered only after a WorkItem exists                                            |
| Preserve Model and Agent output and stream port plans                        | Open        | `invokeWorkflowLLMTarget` omits validated ports from its child-frame intent                              |
| Route direct capabilities through owning adapters                            | Open, held  | Separate functional work. Cleanup does not fabricate Artifacts or add generic Workflow fallback routing. |
| Evaluate callable interface capability requirements                          | Open, held  | Separate functional work. Cleanup removes generic rechecks but does not invent an evaluator.             |
| Lower provider-managed Workflow Tools through typed Model child plans        | Open        | admission currently rejects the target before Model-plan lowering                                        |
| Move concrete URL and diff behavior out of generic Workflow setup            | Implemented | `attachmentruntime` and `textruntime` own behavior; Workflow composition only registers bindings         |

### Execute Workflow reliability behavior

| Subtask                                                                   | Status      | Evidence or required result                                       |
| ------------------------------------------------------------------------- | ----------- | ----------------------------------------------------------------- |
| Persist state owner identity and schema                                   | Implemented | Workflow state-owner runtime code                                 |
| Persist Parallel and Foreach parent-state merges                          | Implemented | Durable StateUpdate merge WorkItems                               |
| Persist failed and uncertain step outcomes                                | Implemented | Workflow outcome runtime code                                     |
| Make Workflow logical evaluation time Runtime-owned                       | Open        | Workflow initial state currently reads the process clock directly |
| Persist retry boundaries and backoff waits                                | Implemented | Retry ProgramFrame implementation                                 |
| Preserve effects, reconciliation, and compensation                        | Implemented | Session Runtime v3 effect and Workflow code                       |
| Verify Service and Activity retry adaptation for every registered binding | Unverified  | Binding-level tests are required                                  |
| Verify cancellation races for every mutating target family                | Unverified  | Cross-domain tests are required                                   |

### Complete normal conversation parity

| Subtask                                                               | Status      | Evidence or required result                                      |
| --------------------------------------------------------------------- | ----------- | ---------------------------------------------------------------- |
| Persist messages, Tool calls, waits, streaming, Apps, and projections | Implemented | Session Runtime v3 and V3 setup binder                           |
| Persist typed Model defaults and selected Artifact facts              | Implemented | Session defaults and dispatch plans                              |
| Support full Session v2 configuration and direct capability surface   | Partial     | V3 configuration surface remains smaller                         |
| Map every V2 command and read to V3                                   | Partial     | Wrapper has broad coverage, but no complete parity ledger exists |
| Migrate frontend traffic and stored Sessions                          | Open        | Migration and frontend evidence was not supplied                 |

### Remove repeated admission and setup layers

| Removal item                                                | Status      | Required result                                                                                                                        |
| ----------------------------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Remove static declaration rechecks after Workflow admission | Implemented | Setup no longer calls generic invocation or Tool transport recheck passes; Runtime snapshot validation is durable integrity only       |
| Remove repeated source-tree walks during lowering           | Implemented | Compiler lowers admitted source once; Inventory decorates the lowered runtime tree without another source traversal                    |
| Remove the runtime-owned Workflow preparation port          | Implemented | Setup submits prepared snapshots; Session Runtime persists and interprets them                                                         |
| Collapse duplicate Model dispatch-plan processing           | Partial     | One setup entry point now owns normalized LLM planning; full origin-configuration propagation remains held as separate functional work |
| Normalize MCP Tool exposure once                            | Implemented | `normalizeMCPSelections` persists only `all`, `none`, or `selected` before dispatch-plan construction                                  |
| Assign one dispatch-time pin owner per target family        | Implemented | Go Tool freshness remains in Tool Runtime; MCP freshness is verified by MCP Runtime; generic Workflow wrappers no longer read pins     |
| Remove generic Workflow concrete bindings                   | Implemented | URL normalization moved to Attachment Runtime and unified diff moved to Text Runtime; Workflow composition only registers them         |
| Remove runtime reconstruction of declaration permissions    | Implemented | Canonical declaration paths are persisted on ProgramFrames, WorkItems, PortPlans, targets, and step outcomes                           |

### Verify deployment composition

| Subtask                                                 | Status     | Evidence or required result                     |
| ------------------------------------------------------- | ---------- | ----------------------------------------------- |
| Verify Session Store v3 checkpoint and restore behavior | Unverified | Store source and tests required                 |
| Verify application bootstrap ordering                   | Unverified | `cmd/agentgo/app.go` required                   |
| Verify runtime shutdown ordering                        | Unverified | Bootstrap and integration tests required        |
| Verify Wails command and projection compatibility       | Unverified | Frontend and wrapper integration tests required |
