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

## Current implementation cycle

This HLD and its implementation status are work in progress.

The current source-backed assessment covers the supplied:

- SessionV3 commands, dispatch, settlement, recovery, interpreters, budgets,
  resources, channels, ports, triggers, effects, and projections.
- V3 Session setup and Model, Tool, MCP, Skill, attachment, and
  registered-binding paths.
- Workflow execution preparation and source-to-program conversion.
- Session Store v3.
- Application bootstrap and backend wrappers.
- Workflow planning and Inventory where required to establish the preparation
  boundary.
- V2 behavior only as the backend functional replacement reference.

The conclusions are source-based. Application execution and test results were
not reviewed. Absence claims apply only to the supplied source set.

The source-based engineering estimate is:

- Workflow v1 declaration and Artifact planning: accepted as complete for this
  audit.
- Workflow v1, including execution preparation and Runtime behavior:
  approximately 75 percent complete.
- SessionV3 backend, including associated setup: approximately 80 percent
  complete.

These are estimates of functional coverage. They are not percentages of files,
lines, or ledger rows. Repeated descriptions of the same behavior must not
increase the completion estimate.

The main remaining problem is not missing operation types. Most operation
types exist. The remaining gaps are composed execution paths, especially
recovery, cancellation, continuation, capability configuration, channel
visibility, and settlement.

### Scope exclusions

This implementation cycle excludes:

- Tests and all validation work.
- Validation-only hardening and cleanup.
- Frontend integration.
- SessionV2 deletion, replacement cleanup, migration, and stored-session
  conversion.
- Version changes, release work, and release notes.
- Cost-budget metering.

No version identifier may be changed in this cycle. This includes Workflow
`apiVersion`, ProgramFrame payload and interpreter versions, Artifact and Store
schema versions, package and module versions, public API versions, and release
identifiers.

### Completion discipline

This is one substantial end-to-end functional delivery. It is not a sequence
of status-only edits, small phases, or isolated helper changes.

An item can change from `Not implemented` or `Partial` to `Implemented` only
when the full listed behavior exists in source. A status change alone is not
completion.

The single current Runtime and consumer-execution status authority is
`Functional tasks` at the end of this document. It replaces former Runtime
component ledgers, pending ledgers, task registers, subregisters, architecture
status tables, and repeated Runtime status claims.

This does not replace the separate source-planning completion record in
`workflow_artifact.md`. That record is limited to Artifact admission,
Inventory, closure, source identity, and read-only planning facts.

## Functional guardrails

This HLD tracks missing behavior and remaining functional work only.

- Artifact and attachment admission own source and input admission behavior.
- Runtime and setup must not repeat admission checks without a functional
  ownership reason.
- Do not add strictness, malformed-input handling, or validation-only work.
- Do not add tests or validation reports in this cycle.
- A Session is a reproducible semantic checkpoint. It is not a serialization
  of live Runtime state.
- Durable Session facts can include completed semantic messages, selected
  Artifact facts, completed Tool and Model outcomes, and approved next-message
  context.
- Durable Session facts must not include live clients, timers, cancellation
  functions, private bindings, credentials, OAuth tokens, Skill Runtime session
  IDs, callbacks, or partial in-flight loads.
- All executable Model, Agent, Tool, Skill, MCP, Service, Activity, and
  Workflow capabilities in scope are Artifact-backed or registered
  binding-backed.
- `session.DirectCapabilitySelection` is a legacy SessionV2 compatibility
  representation. It is not a SessionV3 replacement requirement.
- Current application registration provides no direct capability provider.
- Current Tool Store mapping and resolution accept only Artifact-backed Tool
  targets. Consequently, no fresh normal user, Agent, built-in Tool, provider
  SDK Tool, MCP Tool, or Skill Tool path can create a usable direct Tool
  target.
- Built-in Go Tools and provider SDK Tools are Tool Artifacts.
- MCP Tools are discovered operations of selected MCP Server Artifacts.
- Skill lifecycle Tools are supplied by selected Skill Runtime sessions.
- Do not add direct Tool, Skill, Model, Agent, MCP, Service, Activity, or
  Workflow capability paths.
- Inline Text remains content or declaration data. It is not a direct
  executable capability.

At eventual SessionV2 retirement, legacy direct Tool compatibility fields can
be removed without a direct-target migration or translation. No current task is
created for that future retirement because the supplied application has no
fresh-user producer for a direct Tool target.

## Execution architecture and ownership

`workflow_artifact.md` defines Artifact source admission, declaration grammar,
Inventory construction, read-only planning, and static closure construction.
This document records only the consumer-facing Artifact handoff contract.

### Separate Artifact planning from execution preparation

Workflow Artifact planning owns static source facts.

It returns:

- The admitted Workflow declaration.
- The root Workflow Artifact identity.
- A complete static Artifact closure.
- Static dependency edges.
- Callable target facts.
- Requirement facts.
- Unsupported direct-composition issue facts.
- Source inventory facts for calls, contexts, ports, retries, effects, blocks,
  and triggers.

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
- Typed Workflow source-to-runtime conversion.
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
- Recreate a trigger or retry scheduler.
- Create WorkItems.
- Start timers.
- Persist Session state.
- Run inside Session Runtime.

The Runtime accepts only a typed `ProgramSnapshot`.

The application setup boundary invokes execution preparation before it submits a
Workflow start or trigger-registration command. Session Runtime persists and
interprets the prepared snapshot only.

### Core Runtime, setup, and domain ownership

Core Runtime owns:

- Durable Session execution.
- Program interpretation.
- Logical evaluation boundaries.
- Scheduling.
- Settlement.
- Cancellation.
- Recovery.
- Lifecycle notification.
- Durable command semantics.
- Checkpointing.
- Projection state.

Core Runtime must not import Artifact Store or setup packages.

Setup can consume existing generic Artifact facts and read-only Workflow plans.
Artifact Store remains a passive source-fact provider. It must not gain
preparation, binding, dispatch, scheduling, recovery, or consumer-private APIs.

Domain setup owns:

- Native request grammar.
- Source-family contribution semantics.
- Current private resources.
- Native invocation.
- Native result conversion.
- Domain-specific authorization.

Session and Workflow setup compose neutral domain facts. They must not become
alternative implementations of domain behavior.

An immutable origin-plan reference must preserve invocation meaning. Replacing
a copied plan with a reference must not cause generated Tools to resolve current
Session defaults.

Shared Runtime helpers must replace equivalent algorithms without forming a
generic native-domain envelope. Lowered-tree traversal remains separate from
Artifact-source interpretation.

MCP connection, authorization, discovery, and MCP-owned timers remain
MCP-domain lifecycle behavior. They are separate from the Runtime orchestration
due-time owner.

V2 coexistence remains intact. This HLD does not authorize V2 retirement.

Ownership consolidation must not add repeated admission or a separate
validation project.

Architecture completion must not introduce a second scheduler, second durable
execution root, generic native-domain envelope, or Artifact Store execution API.

### Retained source-backed behavior

The following source-backed behavior is retained. These are component facts, not
independent completion claims for composed execution paths.

- Core Runtime has no Artifact Store or setup dependency.
  `agentruntime-go/workbinding` provides neutral execution contracts.
- Workflow direct composition for Model, Agent, Tool, MCP Tool, Skill Tool,
  Service, Activity, and Nested Workflow is rejected before a
  `ProgramSnapshot` exists.
- Generic callable capability facts and `CallableInterface.capabilities` are
  removed from Workflow v1 grammar and Runtime plumbing.
- Root requirements remain static Artifact-backed or registered-binding
  prerequisites. They are not callable target selections.
- Service and Activity remain registered typed `WorkBinding` targets.
- Explicit Model and Agent messages, instructions, defaults, and inline Text
  remain declaration data. They are not generic native request envelopes.
- Provider SDK Tools remain provider-managed typed Model-plan capabilities.
  They do not become local Tool WorkItems or synthetic local results.
- Provider SDK Tool configuration and an owner Model are retained through
  `ToolSelection.UserArguments`, `ToolDispatchPlan.ProviderConfiguration`,
  `ProviderManagedToolChoice.Configuration`, and owner-Model resolution.
- Workflow MCP preparation retains selected server, operation, policy, and
  discovery facts before ProgramSnapshot creation.
- Static target sets and resolver-backed target sets have separate durable
  representations. Resolver settlement resumes the preserved call.
- Child-call WorkItem identities include the owning ProgramFrame and logical
  child path.
- Published nested Workflow port boundaries reject undeclared mappings.
- Forward-only started-call wrappers retain routing facts and suppress duplicate
  final delivery for covered paths.
- Repeated local schedule-slot identity is retained to prevent duplicate local
  calendar occurrences.
- Attachment content and digest are admitted before message persistence.
- Typed Model defaults and Store-owned semantic export exist.
- Direct Workflow start and trigger registration are separate operations.
- V3 MCP content adaptation has one MCP-domain conversion boundary.
- WorkItems and child ProgramFrames retain canonical declaration paths and
  `EvaluationBoundary` logical-time facts through normal, retry, structured,
  cleanup, and recovery execution.
- `resumeRestoredWork` rearms ordinary `RetryWaiting` WorkItems through the
  Runtime due-time lifecycle after restore.
- `consumeContinuationLocked` transfers staged
  `Execution.Continuation.AppContext` into the next persisted message Activity
  when it creates that Activity. `V3-CONTINUE-003` remains open because the
  active owning Agent operation is not always resumed.
- V3 consumes the selected Agent Model capability `IncludeSystemPrompt` during
  Model patch composition while retaining explicit Session Model prompt
  precedence.
- Bound native Tool calls persist before invocation. Neutral registered-binding
  results can request durable waits while their private resume state remains
  outside Session persistence.
- `cmd/agentgo/app.go` constructs dependencies before Store v3 and Runtime v3
  restore. `SessionV3Wrapper.close` closes Runtime v3 before Store v3.
- Runtime logical evaluation time is Runtime-owned.
- Cost metering remains intentionally absent.

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

`ConversationOwnership.ConversationID` is a Workflow handoff semantic key. It
does not create another Session root, normal conversation identity, Message
identity, Continuation identity, or Execution identity.

Normal conversation routing consults only the ownership record whose
conversation key is `string(Session.ID)`. A Workflow can retain other ownership
records for orchestration semantics, but they do not create frontend-facing
conversations in this cycle.

## Persist commands before external work

Every mutating command has a command identity.

The Runtime:

- Canonicalizes the caller payload.
- Computes a command fingerprint.
- Returns the prior receipt for an identical command identity and payload.
- Rejects reuse of a command identity with a different payload.
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
    Asynchronous Runtime work

The Runtime must not start new external work after a checkpoint failure.

Workflow preparation remains outside Runtime. Command replay must nevertheless
return a prior accepted receipt without requiring old local files, source
selection, or setup resources to remain available.

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

The Runtime performs these operations:

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
- Launch all newly ready Session work only after committed settlement.

The Runtime owns WorkItem state. A domain binder owns external execution.

A terminal result must use one complete Runtime lifecycle:

- Settle Attempts and effect facts.
- Release or transfer claims.
- Wake completion and resource waiters.
- Advance owners and settle Executions.
- Persist the authoritative Session state.
- Launch all newly ready work.

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

- `builtin.conversation@1`: routes visible conversation activity.
- `builtin.agent-loop@1`: runs Agent Model and Tool cycles.
- `builtin.single-model@1`: runs one direct Model request.
- `artifact.workflow.v1@1`: runs typed Workflow behavior.

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

The Runtime owns:

- Activity persistence.
- WorkItem lifecycle.
- Streaming state.
- Tool continuation.
- Wait state.
- Execution state.
- Checkpointing.
- Projection state.

Every LLM WorkItem retains its effective invocation configuration and
capability facts. Tool WorkItems created from that LLM result retain immutable
origin facts required by their target binders.

A binder must not replace selected typed bindings or invocation facts with
current Session defaults when it evaluates or executes a generated Tool
WorkItem.

### Canonical V3 Tool selection

`SessionDefaults.ToolSelections` is the canonical V3 Tool-selection
collection. It contains the Artifact selection plus Tool-only user arguments
and automatic-execution facts.

`ArtifactSelection` remains generic. V3 must not retain a second Artifact-only
`Tools` collection because duplicate collections produce ambiguous precedence
and duplicate Tool choices.

## Admit replay-safe attachments before persistence

Attachment admission is setup-owned.

The V3 attachment path:

- Validates command-only attachment sources.
- Reads local files and images.
- Expands directories once.
- Fetches or snapshots URL content according to the selected mode.
- Produces immutable semantic attachments.
- Persists attachment content and digest rather than a live native path.

Session Runtime v3 stores attachment snapshots. Historical replay does not
reopen local native files.

Provider conversion must preserve semantic attachment labels and stable
directory-relative identity while continuing to use the admitted snapshot.

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

Execution preparation must preserve the prepared target meaning. It must retain
the source-backed immutable pins and domain-prepared contribution facts required
to reproduce the prepared operation without retaining clients, credentials, or
Artifact Store handles.

Pure deterministic or registered-binding Workflows must not depend on unrelated
Session conversation defaults. An owner Model is resolved only where
provider-managed Tool lowering requires one.

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

The interpreter emits Runtime transitions. It does not call a domain directly.

A Runtime transition can contain:

- WorkItem intent.
- Child ProgramFrame intent.
- State update intent.
- Wait intent.
- Cancellation intent.
- Reconciliation settlement.
- Compensation settlement.
- Terminal ProgramFrame outcome.

### Persist callable operation facts

Execution preparation lowers each callable declaration to typed Runtime intent.

Before dispatch or child materialization, Runtime persists the evaluated logical
call boundary:

- Prepared input and explicit context.
- Selected target identity.
- Resolved target-set membership.
- Typed Model or Agent messages and instructions.
- Relevant port input values.
- Effect key and retry identity.
- Deadline and port plans.
- Logical evaluation time.
- Immutable Artifact pins and semantic binding facts.
- Captured retry, deadline, cleanup, effect, idempotency, and budget policy.

Retries, started calls, waits, recovery, and resumption reuse those persisted
facts. They must not evaluate a different request from later channel contents,
later state, or settlement time.

A resolver-backed target set starts with a typed resolver WorkItem. Runtime
persists normalized membership before target selection.

A successful `start` call persists its completion handle and start receipt in
the parent step outcome.

A handoff persists conversation ownership facts before its child ProgramFrame
starts.

### Preserve state ownership and Workflow outcomes

A state-owning scope provides:

- State owner identity.
- State schema.
- State snapshot.

A typed `StateUpdateWork` applies a state update atomically.

Parallel and Foreach child frames receive isolated state.

A parent state changes only through a declared parent-state merge. Runtime
persists that merge as a typed StateUpdate WorkItem.

Workflow outcomes retain:

- Actual step identity.
- Outcome state.
- Output.
- Selected target.
- Receipt.
- Error.
- Effect state.
- Aggregate facts.
- Declaration path.
- Native typed error facts where available.

Outcome states are:

- `succeeded`
- `failed`
- `skipped`
- `cancelled`
- `uncertain`

A skipped step has no output.

An uncertain step records that an external effect can have occurred without
durable confirmation.

Every actual terminal step boundary, including inline failures, return, break,
continue, reconciliation, cancellation, and child settlement, must record its
semantic outcome.

## Bind each target through its owning domain

Workflow target behavior is:

- Model: a child single-model ProgramFrame. Model and Session setup own
  preparation.
- Agent: a child Agent-loop ProgramFrame. Agent and Model setup own
  preparation.
- Tool Artifact: a Tool WorkItem. Tool Runtime owns native execution.
- MCP Tool: a Tool WorkItem. MCP Runtime owns native execution.
- Skill Artifact selection: a typed Model or Agent child plan with Skill
  lifecycle Tool calls. Skill Runtime owns lifecycle behavior.
- Provider SDK Tool: a provider-managed Tool choice in a typed Model plan.
- Service: a Service WorkItem. A registered Service adapter owns native work.
- Activity: an Activity WorkItem. A registered Activity adapter owns native
  work.
- Nested Workflow: a child Workflow ProgramFrame. Workflow execution
  preparation owns child preparation.

A Skill Artifact is not a standalone Workflow callable target. A typed Model
or Agent invocation can select Artifact-backed Skills through typed defaults.
The resulting Skill lifecycle Tool calls belong to that Model or Agent plan.

Standalone Tool selection is limited to built-in Tool Artifacts. Go Tools and
provider SDK Tools remain Tool Artifacts. MCP Tools remain discovered MCP
Server operations. Skill lifecycle Tools remain Skill Runtime operations.

Workflow owns orchestration. It does not own native domain behavior.

A target adapter owns:

- Native request construction.
- Current domain readiness.
- Native result conversion.
- Domain-specific authorization.
- Domain-private state.

A target adapter must not use generic Workflow data as a replacement for native
domain request grammar.

## Convert Model and Agent Workflow calls before Runtime execution

Workflow Model and Agent calls use explicit typed invocation data.

Supported forms are:

- Explicit message lists.
- Explicit instructions.
- Existing Session default overrides.
- Final output ports.
- Streamed output ports.
- Agent handoff.

Runtime receives typed fields. Runtime must not parse generic Workflow JSON to
discover messages, instructions, Model defaults, Agent defaults, or provider
request fields.

Setup converts supported Workflow forms into typed Runtime invocation fields.
Runtime then creates child Agent-loop or single-model ProgramFrames.

When a Model or Agent call accepts a final-output or streamed-output port, the
lowered child ProgramFrame contains accepted port plans before child work starts.

Arbitrary Workflow JSON must never become fallback Model or Agent chat text.

Native Model and Agent failures must retain their typed category and cause
through child settlement so retry and catch can use actual domain meaning.

## Enforce ports, channels, and target transport

Runtime owns:

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
declaration path separately from its runtime path.

Channel permissions and port authorization use declaration paths.

Private channel storage names are invocation-local. Declaration-visible channel
names remain available to expressions. A declaration expression such as
`messages.review` must continue to use its declared alias even when the
underlying channel log has a private nested Workflow namespace.

Setup owns target transport capability.

A target transport check validates:

- Context support.
- Input port support.
- Final output port support.
- Streamed output port support.
- Port schema compatibility.
- Channel authorization.
- Current target capability.

Target transport behavior is:

- Model and Agent calls support only explicit `messages`, `instructions`, and
  `defaults` as Workflow context. They do not accept generic input ports. They
  support final and streamed output ports.
- Nested Workflows use child ProgramFrame scope and support input, final-output,
  and streamed-output ports.
- Port-aware Service and Activity targets support only the context and ports
  declared by their registered binding.
- Go Tool, MCP Tool, and Skill Tool targets do not accept Workflow context or
  input ports. They support final output only after confirmed settlement. They
  do not support streamed output ports.
- Provider SDK Tools exist only inside typed Model plans. Their input and output
  behavior is provider-managed.

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
contains ordered stream records only. Terminal stream signaling, if needed,
uses a distinct durable stream record.

A provider-managed Tool target is lowered only through a typed Model child
plan. It never becomes a local Tool WorkItem or synthetic local result.

## Persist waits, events, channels, and triggers

Runtime owns durable wait state.

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

Runtime persists timer facts, not timer objects.

Runtime recreates timers from persisted facts after restore.

Runtime owns:

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

Runtime deduplicates accepted external events by `(source, id)`.

A Runtime channel record contains message identity, sequence, channel identity,
kind, data, correlation, deduplication key, creation time, and consumption
facts. Channel cursors and consumption facts remain Runtime state.

An emit persists a durable receipt. A duplicate deduplication key for the same
destination returns the stored receipt and does not append another message.

Human-wait resolution is accepted only when the response actor satisfies
persisted assignment facts. A business rejection remains response data unless
the Workflow explicitly fails.

Runtime has one scheduler for:

- Wait expiry.
- Retry delays.
- Resource waits.
- Execution deadlines.
- Cleanup deadlines.
- Trigger schedules.

Workflow setup must not create another scheduler.

Trigger registration is explicit and independent from direct Workflow start. A
direct start creates one Execution only. It does not activate event or schedule
registrations.

Trigger registration persists an input-unbound program template and is the only
operation that activates or changes trigger lifecycle state.

Trigger, ordinary event wait, and Machine expressions must expose the same
semantic event projection:

- `type`
- source
- identity
- data
- correlation
- time

## Preserve effect safety during retry and recovery

A WorkItem has an effect contract and effect state.

Effect kinds are:

- `read`
- `idempotent`
- `keyed`
- `nonIdempotent`

Effect states are:

- `notApplicable`
- `notStarted`
- `dispatched`
- `confirmedApplied`
- `confirmedNotApplied`
- `uncertain`
- `compensating`
- `compensated`
- `compensationFailed`

Workflow source effect values must map explicitly to Runtime effect values. In
particular, source `non-idempotent` maps to Runtime `nonIdempotent`.

Retry behavior follows effect safety:

- Read, idempotent, and keyed work can replay according to declared policy.
- A keyed operation keeps one stable idempotency key.
- Non-idempotent uncertain work does not replay automatically.
- Confirmed non-application can enable a retry.
- Recovery creates review work when certainty is unavailable.
- Declared retry matching, including schema matching, remains available.
- Replay decisions consider declared effect safety before rejecting or allowing
  an uncertain result.

Workflow retry uses a durable retry ProgramFrame boundary.

The retry boundary retains:

- Attempt count.
- Child frame identity.
- Backoff wait identity.
- Retry deadline.
- Last error.
- Resolved target-set facts.
- Child settlement facts.
- Original logical-call snapshot.

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

MCP execution must report the strongest actual domain evidence. A confirmed
successful MCP response must not automatically become effect-uncertain. An MCP
error must not automatically prove non-application.

Normal Tool continuation, Workflow outcomes, reconciliation eligibility,
historical Tool reconstruction, and Saga recording consume the same settled
effect facts.

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

Runtime preserves primary failure or cancellation as the primary outcome.

Unknown effects do not enter automatic compensation until reconciliation confirms
application.

Compensation dependencies preserve causal order from ordinary ordered calls and
structured boundaries. Independent effects can compensate concurrently only when
their persisted dependencies permit it.

Compensation receives declared operation facts without private state:

- Original operation identity.
- Original target.
- Original Attempt.
- Original receipt.
- Original effect facts.
- Compensation identity.

A Saga that can receive a late external effect confirmation remains available
until eligible in-flight work settles or its cleanup allowance expires. An empty
initial compensation order is not proof that later compensation is unnecessary.

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
- Runs finalization.
- Starts eligible reconciliation and compensation.

Cancellation does not prove that a dispatched mutating target did not apply its
effect. Cancellation remains cancellation through catch and finally behavior.
A catch handler can return an explicit allowed result only after cancellation
cleanup completes.

Cleanup work uses its effective cleanup deadline through binding, execution,
retry, resource acquisition, waits, emits, streams, and settlement. Normal work
stops. Eligible cleanup continues through the bounded cleanup allowance.

Synthetic frames, including retry wrappers, retain the cleanup classification of
the work that they contain.

Runtime close:

- Marks the Runtime closed.
- Cancels lifecycle context.
- Cancels dispatch contexts.
- Cancels timers.
- Waits for owned asynchronous work.
- Closes process-private execution bindings.
- Releases subscribers.

Recovery:

- Restores Session documents.
- Removes process-private state.
- Retains immutable semantic dispatch facts for replay-safe interrupted work.
- Marks in-flight external work uncertain unless replay-safe.
- Requeues replay-safe work.
- Starts reconciliation when declared and applicable.
- Creates recovery review waits for unresolved uncertainty.
- Rearms timers only after complete restore.
- Resumes dispatchable work.

Recovery reconstructs private resources from retained semantic operation facts.
It does not substitute current Session defaults, current Model overlays, current
contribution selection, or current message activity selection for a retained
operation.

## Persist projections and expose wrapper reads

Runtime provides durable projections for:

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

## Validation ownership

Validation ownership remains:

- Workflow source grammar and static semantic validity: Workflow Artifact
  contract.
- Static Artifact relationship closure: Workflow Artifact planning.
- Workflow execution preparation and source-to-program conversion: Workflow
  setup.
- Invocation input validation: Workflow setup.
- Current Artifact pin and enablement: Workflow setup or target-specific binder.
- Direct composition rejection for Workflow targets: Workflow Artifact planning
  and Workflow setup.
- Domain readiness and native request validity: owning domain adapter.
- Durable Session and execution integrity: Session Runtime v3.
- Session document and tombstone persistence: Session Store v3.
- Wails request transport: wrapper and frontend transport.

Runtime validates durable Runtime objects. It does not revalidate Workflow
source grammar.

## Completion and replacement readiness

Status meanings are:

- `Implemented`: the inspected source contains the complete listed behavior.
  Test execution was not reviewed.
- `Partial`: the inspected source contains part of the behavior or has an
  identified functional boundary gap.
- `Not implemented`: the required behavior is absent from the inspected source.
- `Not applicable`: the behavior is explicitly excluded from this cycle or is
  not a Runtime task.

Every open parent and subtask must be completed in one substantial end-to-end
functional delivery.

When every current task is implemented, V3 is intended to replace the supported
Artifact-backed and registered-binding V2 backend behavior while adding Workflow
orchestration.

This is a backend functional replacement decision. It does not claim that:

- Frontend traffic has migrated.
- Stored Sessions have migrated or been converted.
- SessionV2 has been retired.
- Release or migration work has completed.

Frontend adaptation, stored-session conversion or archival policy, application
traffic switching, and eventual V2 retirement remain separate migration work.
They are not architectural blockers to beginning migration after the functional
tasks are complete.

## Functional tasks

This is the single current Runtime and consumer-execution status ledger.

It replaces former Runtime completion tables, pending-work tables, status
registers, subregisters, Runtime architecture status tables, and repeated
Runtime component claims.

The separate Artifact source-planning record in `workflow_artifact.md` is not
an execution-status ledger and does not create Runtime work.

Each functional group contains parent tasks and their subtasks. A parent is
complete only when every listed subtask is complete. Source evidence identifies
the current observed gap. Required result identifies the complete behavior
required before the status can change.

`R` means `internal/agentruntime-go/sessionv3/`.

`S` means `internal/agentruntimesetup/`.

### Runtime settlement, cleanup, scheduling, commands, and persistence

#### `RT-SETTLE-003` Authoritative mutation and terminal propagation

- Status: Partial.
- Corrected prior claims: `RT-DISPATCH-001`, `RT-DISPATCH-001.a`,
  `RT-DISPATCH-001.b`, and `ARC-CALL-POLICY-001`.
- Current evidence:
  - `advanceFrameLocked` contains identity-based reacquisition.
  - `wakeEventWaitsLocked` can retain a `Wait` pointer across
    `resolveDurableWaitLocked`, which can call `finishWorkLocked` and advance
    orchestration.
  - `dispatchEmit` calls `finishWorkLocked` and can continue with an earlier
    Execution pointer before later reacquisition.
  - `completeNormalToolCall` can advance the owning frame and continue with
    earlier WorkItem and Execution pointers.
  - Reconciliation delivery can wake work and mutate Session execution state
    before later mutation of the original operation.
  - The `bindErr` branch in `dispatchOne` can create catch or finally work
    without launching it.
  - Port-failure branches can advance a frame and persist without the common
    ready-work launch.
  - `finishPanickedWork` persists settlement without launching next ready work.
  - Invalid wait-resolution failure can bypass normal settlement.
  - Early cancellation settlement in `finishWorkLocked` can return before
    ordinary completion-wait wakeup.
- Required result:
  - Every mutation after an operation that can append, replace, restore, or
    otherwise invalidate slice-backed Session objects reacquires the
    authoritative Session, Execution, ProgramFrame, WorkItem, and Wait by
    durable identity.
  - Every terminal path uses one complete committed lifecycle: settle Attempt
    and effect facts, release or transfer claims, wake completion waits, advance
    owners, settle Executions, checkpoint, then launch all newly ready Session
    work.
  - Catch, finally, parent completion waits, resource waiters, forwarded
    channels, independent children, and cross-Execution work do not remain idle
    after a committed dependency settles.

##### `RT-SETTLE-003.a` Reacquire authoritative Session-owned objects

- Status: Partial.
- Current evidence: `R/coordination.go` retains wait pointers across durable
  resolution. Emit, manual result, and reconciliation paths can continue with
  earlier objects.
- Required result:
  - Mutate through durable Session, Execution, ProgramFrame, WorkItem, and Wait
    identities after append, rollback, child creation, forwarding, orchestration
    advancement, or durable resolution.
  - Do not use a stale pointer after a mutation-capable Runtime call.

##### `RT-SETTLE-003.b` Use one terminal lifecycle and committed ready-work launch

- Status: Partial.
- Current evidence: binding error, port error, panic, invalid wait-resolution
  result, manual result, cancellation, reconciliation, and child completion use
  different terminal consequences.
- Required result:
  - Route each terminal condition through the common lifecycle.
  - Persist before launching dependency-ready work.
  - Preserve the actual effect and error facts through settlement.
  - Ensure a failure cannot leave catch, finally, completion, or resource work
    dormant.

#### `RT-CLEANUP-003` Execute cleanup through every owning lifecycle

- Status: Partial.
- Corrected prior claims: `RT-CLEANUP-002` and cleanup-complete scheduler
  claims.
- Current evidence:
  - `dispatchOne` builds binding context from `item.Deadline`, including an
    expired ordinary deadline.
  - Its post-binding deadline branch does not exclude cleanup work.
  - `dispatchEmit` rejects cancellation-requested work, including cleanup
    emits.
  - Retry and resource timer callbacks cancel work on Execution cancellation
    without cleanup ownership.
  - Stream paths reject cancellation without distinguishing permitted cleanup
    output.
  - Cleanup detection examines a Workflow program directly. A retry wrapper
    around cleanup work has a different representation.
- Required result:
  - Normal work stops after cancellation or ordinary deadline expiry.
  - Eligible cleanup uses its effective cleanup deadline and continues through
    the bounded cleanup allowance.
  - Cleanup classification survives binding, execution, waits, retry, resource
    acquisition, emits, streams, settlement, and synthetic frames.
  - Cancellation remains the primary terminal meaning.

##### `RT-CLEANUP-003.a` Carry cleanup classification and effective deadline

- Status: Partial.
- Current evidence: `R/runtime.go`, `coordination.go`, `retry.go`,
  `runtime_resources.go`, and stream paths retain ordinary-work decisions.
- Required result:
  - Apply the effective cleanup deadline from binding through external
    invocation, retry, resource waiting, emits, streams, and final settlement.
  - Ensure retry wrappers and other synthetic frames inherit cleanup
    classification.
  - Do not reject eligible cleanup only because ordinary work is cancelled or
    expired.

##### `RT-CLEANUP-003.b` Retain Saga ownership until late outcomes settle

- Status: Partial.
- Current evidence:
  - `workflow_saga.go:beginSagaCompensation` can finish a Saga immediately when
    its initial compensation order is empty.
  - Dispatched normal work can confirm an effect after that point.
  - `resumeCancellationCleanupFramesLocked` resumes only nonterminal frames
    with pending Saga progress.
- Required result:
  - Keep a Saga available while late effect confirmation, reconciliation,
    compensation, completion wakeup, or final resource release remains
    possible.
  - Do not treat an empty initial compensation list as proof that no later
    compensation is required.
  - Preserve cancellation as the primary outcome.

#### `RT-BUDGET-003` Complete scoped accounting and resource-ready scheduling

- Status: Partial.
- Corrected prior claims: `RT-LIMIT-002` and child-call policy completion
  claims.
- Current evidence:
  - Budget concurrency counts `Binding` and `Dispatched` work.
  - A WorkItem changing from `Binding` to `Waiting` frees a slot but does not
    consistently wake budget-concurrency waiters.
  - Binding failure can release resources before changing the WorkItem out of
    `Binding`.
  - Scope, Choose, Parallel, Foreach, Repeat, Machine, and Saga child-frame
    construction do not consistently retain `step.Limits`.
  - `childProgramFrameDepth` increments depth for synthetic orchestration
    frames as well as nested Workflow frames.
  - Child-call and Fork work can become separately ready before a durable outer
    operation Attempt and budget reservation exist.
- Required result:
  - Slot release wakes eligible work regardless of whether release comes from
    binding wait, success, failure, cancellation, or immediate settlement.
  - Structured constructs apply declared narrowed budgets to owned work.
  - Attached descendants consume intended aggregate counters.
  - Workflow depth means Workflow nesting, not incidental Runtime frame depth.
  - The durable outer child-call operation is reserved before child native work
    can dispatch.

##### `RT-BUDGET-003.a` Wake waiters when a counted slot becomes available

- Status: Partial.
- Current evidence: `budget.go:activeBudgetWork`,
  `wakeBudgetConcurrencyWaitersLocked`, and `runtime.go:dispatchOne` do not
  consistently wake waiters after `Binding` to `Waiting`, binding failure, or
  similar state transitions.
- Required result:
  - Every counted-slot release runs eligible durable wakeup and ready-work
    launch.
  - A durable wait must not require an unrelated event to restart available
    work.

##### `RT-BUDGET-003.b` Apply structured limits and semantic Workflow depth

- Status: Partial.
- Current evidence: structured child-frame intents do not consistently carry
  `step.Limits`; `childProgramFrameDepth` counts synthetic frames.
- Required result:
  - Apply construct-local narrowed limits to Scope, Choose, Parallel, Foreach,
    Repeat, Machine, and Saga work.
  - Preserve parent aggregate accounting.
  - Count nested Workflow depth separately from internal Fork, retry, and
    structured-block frames.

##### `RT-BUDGET-003.c` Reserve outer child-call work before child dispatch

- Status: Partial.
- Current evidence: child-call and Fork work can be created as separately ready
  work.
- Required result:
  - Persist the logical outer Attempt, effective policy, accounting, and
    ownership before a child performs native work.
  - A child must not bypass the outer budget decision.

#### `RT-TRIGGER-003` Resume trigger lifecycle changes and expose one event meaning

- Status: Partial.
- Corrected prior claims: `RT-TRIGGER-002`, `ARC-SCHEDULER-001`, and complete
  trigger queue resumption claims.
- Current evidence:
  - `SetTriggerRegistrationState` rearms schedule timers without draining
    already eligible queued occurrences.
  - Trigger expressions expose `event.name`.
  - Ordinary event waits expose `type`.
  - Machine expression behavior depends on the event projection it receives.
- Required result:
  - Registration reactivation drains eligible occurrences through the same
    committed ready-work launch as other Runtime work.
  - Missed-occurrence behavior follows the declared registration policy.
  - Trigger, wait, and Machine expressions receive one semantic event
    projection.

##### `RT-TRIGGER-003.a` Drain eligible occurrences after reactivation

- Status: Partial.
- Current evidence: registration state changes rearm timers but do not drain
  eligible queues.
- Required result:
  - Persist registration and queue changes.
  - Apply missed-occurrence behavior.
  - Launch eligible work through the shared Runtime path.

##### `RT-TRIGGER-003.b` Use one declared event projection

- Status: Partial.
- Current evidence: `TriggerEventEnvelope` exposes `name`, while event-wait
  output exposes `type`.
- Required result:
  - Preserve `type`, source, identity, data, correlation, and time through one
    semantic event projection.

#### `RT-COMMAND-003` Replay accepted commands independently of regenerated preparation

- Status: Partial.
- Current evidence:
  - Message attachment admission can run before command receipt lookup.
  - Continuation attachment admission and edit paths can require old local
    sources before replay lookup.
  - Workflow wrappers can prepare a ProgramSnapshot before Runtime replay
    returns the prior receipt.
  - `AcceptTriggerEvent` fills missing `OccurredAt` before command
    fingerprinting.
- Required result:
  - An accepted command returns its prior receipt without requiring former
    source files, attachments, selected Artifacts, or setup resources.
  - Stable caller payload identity is preserved across preparation.
  - Runtime does not own Workflow preparation.

##### `RT-COMMAND-003.a` Preserve public command identity before preparation

- Status: Partial.
- Required result:
  - Use the original caller identity to find prior accepted receipts.
  - Preserve the caller identity across prepared snapshots.
  - Do not repeat attachment admission or source preparation for an accepted
    duplicate command.

##### `RT-COMMAND-003.b` Keep generated occurrence time outside caller identity

- Status: Partial.
- Current evidence: `AcceptTriggerEvent` fills `OccurredAt` before
  fingerprinting.
- Required result:
  - Fingerprint the original caller payload.
  - Persist and reuse generated occurrence facts after acceptance.

#### `RT-PERSIST-003` Complete deletion commitment and idle Session deletion

- Status: Partial.
- Corrected prior claims: `RT-PERSIST-002` and complete normal V2 replacement
  claims.
- Current evidence:
  - `sessionstorev3/store.go:HandleSessionTombstone` can commit a tombstone and
    then return an ordinary derived-index deletion error.
  - `sessionv3/runtime.go:DeleteSession` can restore the live in-memory Session
    after that committed tombstone.
  - An idle normal conversation Execution is intentionally long-lived.
  - `DeleteSession` rejects every nonterminal Execution.
- Required result:
  - Runtime and Store agree whether tombstone commitment occurred.
  - Derived-index repair never resurrects a deleted Session.
  - An idle long-lived normal conversation can be deleted while actual active
    Workflow, child, wait, cleanup, or external work remains protected.

##### `RT-PERSIST-003.a` Treat committed tombstones as committed

- Status: Partial.
- Required result:
  - Report semantic tombstone commitment distinctly from a repairable
    derived-index failure.
  - Preserve deletion receipt and deletion barrier.
  - Schedule derived-index repair without restoring the Session.

##### `RT-PERSIST-003.b` Delete idle long-lived normal conversations

- Status: Partial.
- Required result:
  - Retire an idle conversation root during normal Session deletion.
  - Retain ownership of active Workflow, child, wait, cleanup, and external
    work until it settles.
  - Do not treat this as SessionV2 retirement cleanup.

### Workflow preparation, captured calls, ports, and channels

#### `WF-PREPARE-003` Preserve complete prepared target meaning

- Status: Partial.
- Current evidence:
  - Workflow preparation receives a complete static closure.
  - `S/workflowruntime/admission.go:admitPlanned` primarily retains pins through
    targets and requirements.
  - An Agent target retains the Agent pin.
  - Dispatch later resolves Agent contributions through
    `sessionv3runtime.applyAgentPlan`.
  - There is no complete immutable transitive contribution snapshot on the
    prepared Workflow program.
  - `admitPlanned` calls `workflowOwnerModel` before it knows whether
    provider-managed Tool lowering needs an owner Model.
- Required result:
  - Preserve prepared source-backed meaning for Agent, Model, Text, Tool,
    Skill, MCP, Workspace, nested Workflow, and declared requirements.
  - Do not retain private clients, credentials, or Artifact Store execution
    handles.
  - Do not silently resolve different transitive contributions at dispatch.
  - Pure deterministic or registered-binding Workflows must not fail because
    unrelated Session defaults are unavailable.

##### `WF-PREPARE-003.a` Retain immutable pins and prepared contribution facts

- Status: Partial.
- Required result:
  - Lower prepared transitive selections into neutral Runtime snapshots.
  - Preserve exact prepared Model, Agent, Text, Tool, Skill, MCP, Workspace,
    and nested Workflow meaning.
  - Reconstruct only private current resources at dispatch.

##### `WF-PREPARE-003.b` Resolve an owner Model only where required

- Status: Partial.
- Current evidence: `workflowOwnerModel` runs eagerly from `admitPlanned`.
- Required result:
  - Resolve an owner Model only for provider-managed Tool lowering.
  - Do not require unrelated conversation Model or Agent defaults for pure
    Workflow execution.

#### `WF-RETRY-003` Preserve one captured logical call through retry and started execution

- Status: Partial.
- Corrected prior claims: `WF-RETRY-002` and complete logical-call capture
  claims.
- Current evidence:
  - `retryCaptureChildCall` runs when a child settles.
  - Typed message and instruction capture can evaluate templates against later
    child state and later logical time.
  - Started calls freeze generic input and context but retain typed LLM
    templates for later wrapper-child evaluation.
  - `compileWorkflowRetry` drops `match.schema`.
  - Retry routing rejects uncertainty before considering declared replay-safe
    effects.
  - Generic Agent and single-model failure replacement can discard native typed
    category and cause facts.
- Required result:
  - Capture one complete call snapshot before dispatch or child materialization.
  - Reuse that snapshot through retry, started-child materialization, recovery,
    and settlement.
  - Preserve declared matching and effect-safe replay semantics.
  - Preserve native typed Model and Agent failure classification.

##### `WF-RETRY-003.a` Capture the original evaluation boundary

- Status: Partial.
- Required result:
  - Capture evaluated target, target-set membership, input, context, typed
    messages, instructions, relevant port inputs, key, effect policy, and
    logical time.
  - Capture before dispatch or child creation.
  - Do not observe later channel contents, state, or settlement time.

##### `WF-RETRY-003.b` Apply complete retry selection and effect replay semantics

- Status: Partial.
- Required result:
  - Retain `match.schema`.
  - Reuse declared retry matching.
  - Permit only policy-authorized read, idempotent, keyed, or
    confirmed-not-applied retries.
  - Consider declared replay-safe effects before rejecting an uncertain result.

##### `WF-RETRY-003.c` Preserve native Model and Agent failure classification

- Status: Partial.
- Required result:
  - Retain native category, cause, and domain meaning through child ProgramFrame
    settlement.
  - Make those facts available to retry, catch, outcome, and projection logic.

#### `WF-CHANNEL-003` Preserve invocation-local coordination and declaration-visible meaning

- Status: Partial.
- Corrected prior claims: `WF-PORT-003`, `ARC-TRANSPORT-001`,
  `WF-SCOPE-001`, and parts of `WF-MACHINE-002`.
- Current evidence:
  - `ports_runtime.go:namespaceWorkflowProgramChannels` changes declaration
    names to private names.
  - CEL expressions are not renamed.
  - `workflow_messages.go:captureWorkflowMessages` publishes namespaced names
    as expression-map keys.
  - `Advance` captures messages before settlement and before `drive` can move
    to another step or root output.
  - Inherited subscriptions are copied into child frames and consumption is
    keyed to the waiting child frame.
- Required result:
  - Separate invocation-private storage identity from declaration-visible
    expression identity.
  - Capture an authorized channel view for each actual step and block-output
    boundary.
  - Preserve original subscription ownership through inherited child scopes.
  - Reuse a captured call snapshot for retry and started work.

##### `WF-CHANNEL-003.a` Preserve declared channel aliases

- Status: Partial.
- Required result:
  - Keep declared aliases such as `messages.review` visible to expressions.
  - Keep private nested Workflow channel logs invocation-local.
  - Do not expose private storage namespaces as declaration vocabulary.

##### `WF-CHANNEL-003.b` Capture each evaluation boundary

- Status: Partial.
- Required result:
  - Capture a new authorized channel snapshot for the declaration boundary
    actually being evaluated.
  - Do not allow a later step or root output to inherit a prior boundary's
    authorization or snapshot.
  - Retain the original snapshot for a captured logical call.

##### `WF-CHANNEL-003.c` Preserve inherited subscription identity

- Status: Partial.
- Required result:
  - Retain the originating subscription owner when a child inherits a
    subscription.
  - Do not make an already consumed event available again because execution
    entered a child scope.

### Workflow effects, recovery, structured control, and outcomes

#### `WF-EFFECT-003` Preserve declaration effects and actual business-effect facts

- Status: Partial.
- Corrected prior claims: `WF-EFFECT-002`, `WF-RECOVERY-001`, and portions of
  `RT-CLEANUP-002`.
- Current evidence:
  - Source declarations use `non-idempotent`.
  - Runtime uses `nonIdempotent`.
  - `compileWorkflowEffect` and `compileWorkflowStepEffect` compare source
    strings with Runtime enum strings.
  - `S/mcpruntime/v3.go:Execute` returns `EffectStateUncertain` for every MCP
    response, including confirmed success.
  - `completedSemanticToolOutput` excludes effect-uncertain Tool results from
    historical reconstruction.
  - Ordinary sequential Workflow calls do not establish equivalent causal effect
    dependencies.
  - `WorkflowCompletedOperationFacts` lacks promised original-operation and
    compensation identity facts.
- Required result:
  - Explicitly translate declaration effects.
  - Preserve effect contracts through dispatch, retry, reconciliation, and
    compensation.
  - Use strongest actual domain certainty for MCP settlement.
  - Preserve causal dependencies and full compensation operation facts.
  - Use the same settled facts for continuation, outcome, recovery,
    reconciliation, history, and Saga recording.

##### `WF-EFFECT-003.a` Translate `non-idempotent`

- Status: Partial.
- Required result:
  - Map admitted source `non-idempotent` explicitly to Runtime
    `nonIdempotent`.
  - Do not treat this as a validation-only issue.

##### `WF-EFFECT-003.b` Settle MCP from actual domain evidence

- Status: Partial.
- Required result:
  - Mark confirmed successful MCP responses as confirmed application where
    domain evidence supports that result.
  - Preserve uncertainty for errors that do not prove non-application.
  - Preserve receipts and settled Tool results for normal continuation, history,
    reconciliation, and Saga recording.

##### `WF-EFFECT-003.c` Preserve compensation causality and operation facts

- Status: Partial.
- Required result:
  - Establish effect dependencies across ordered work and structured boundaries.
  - Compensate independent effects concurrently only when dependencies permit
    it.
  - Supply original operation, target, Attempt, receipt, effect, and
    compensation identities without private state.

#### `WF-CONTROL-003` Complete structured-control value and lifetime behavior

- Status: Partial.
- Corrected prior claims: `WF-CONTROL-001`, `WF-SCOPE-001`, and
  `WF-MACHINE-002`.
- Current evidence:
  - Repeat body frames receive an iteration index, but `while` and `until`
    evaluation do not explicitly install the current Repeat index.
  - Empty Foreach with parent-state merge can create merge progress and return
    idle without emitting required merge work.
  - Empty runtime collections under non-`all` completion can remain pending.
  - Lowering loses the difference between omitted and explicit zero
    `maxItems` or `maxIterations`.
  - Machine `Microsteps` increments for event-driven transitions and is not
    reset at a genuine suspension boundary.
- Required result:
  - Apply current Repeat index at every Repeat condition boundary.
  - Execute empty-collection merge behavior.
  - Settle empty collections truthfully.
  - Preserve explicit maximum values.
  - Limit consecutive immediate Machine transitions without limiting the
    lifetime count of accepted events.

##### `WF-CONTROL-003.a` Apply Repeat index to condition evaluation

- Status: Partial.
- Required result:
  - Install the current Repeat index for `while` and `until`.
  - Preserve carried state and result semantics at the same boundary.

##### `WF-CONTROL-003.b` Settle empty collections and preserve maxima

- Status: Partial.
- Required result:
  - Emit required merge work for empty Foreach results.
  - Settle non-`all` empty collections according to declared result behavior.
  - Preserve omitted versus explicit zero `maxItems` and `maxIterations`.

##### `WF-CONTROL-003.c` Limit immediate Machine transitions only

- Status: Partial.
- Required result:
  - Reset the immediate-transition counter at genuine suspension.
  - Permit long-lived event-driven Machines.
  - Retain the immediate-transition safety bound.

#### `WF-OUTCOME-003` Preserve complete Workflow and conversation terminal meaning

- Status: Partial.
- Current evidence:
  - `recordTerminalWorkflowStepOutcome` can use an earlier `PendingPath`.
  - Inline failure reached while `drive` selects a new step can lack the actual
    failing-step outcome.
  - Successful `return`, `break`, and `continue` can bypass ordinary
    step-outcome recording.
  - Failed outcomes do not consistently retain receipt and effect facts from
    the settled WorkItem.
  - Failed or cancelled Agent LLM work can retain stream text only on the
    WorkItem.
  - Current stream merge behavior requires a final LLM payload.
- Required result:
  - Record the actual active step boundary.
  - Retain target, receipt, effect, aggregate, error, and declaration facts.
  - Persist interrupted assistant output once without fabricating successful
    business execution.
  - Preserve actual typed failures through child ProgramFrames.

##### `WF-OUTCOME-003.a` Record complete settled step facts

- Status: Partial.
- Required result:
  - Record actual step identity for ordinary, inline, child, reconciliation,
    failure, cancellation, return, break, and continue paths.
  - Retain target, receipt, effect, aggregate, native typed error, and
    declaration facts.

##### `WF-OUTCOME-003.b` Persist interrupted assistant output truthfully

- Status: Partial.
- Required result:
  - Persist a semantic assistant Activity for usable partial output.
  - Preserve actual provider observations and actual failure or cancellation.
  - Do not create a fabricated successful Model or Tool result.

### Conversation replay, configuration, continuation, handoff, and Skill state

#### `V3-REPLAY-003` Reconstruct private resources from the exact retained operation

- Status: Partial.
- Corrected prior claims: `V3-HISTORY-002` and `ARC-REPLAY-001`.
- Current evidence:
  - `R/runtime.go:recoverLocked` sets `item.Binding = nil` before the later
    plan-preservation condition.
  - `bindRecoveredLLM` calls `buildInputs` without retained activity sequence.
  - Recovered activity selection is recomputed.
  - Model resolution uses current source and overlay-derived configuration.
  - Retaining only a Model Artifact pin does not retain all effective Provider,
    overlay, capability, instruction, and prompt facts.
  - Private execution fingerprint can differ from retained semantic binding
    fingerprint.
- Required result:
  - Retain immutable semantic dispatch facts while discarding private resources.
  - Preserve exact semantic message order.
  - Retain effective non-secret Model and Provider configuration.
  - Reconstruct credentials and clients privately.
  - Never substitute current defaults or current contribution assembly.

##### `V3-REPLAY-003.a` Retain semantic LLM bindings

- Status: Partial.
- Required result:
  - Preserve immutable plans for dispatched replay-safe work and equivalent
    interrupted preparation paths.
  - Clear only private resources.

##### `V3-REPLAY-003.b` Reproduce captured message order and configuration

- Status: Partial.
- Required result:
  - Rebuild from captured message identities and causal order.
  - Retain effective Provider, Model, capability, instruction, context, and
    non-secret overlay facts.
  - Resolve credentials and clients privately without current-default
    substitution.

#### `V3-CONFIG-003` Complete inherited and explicit invocation configuration

- Status: Partial.
- Corrected prior claims: `V3-CONFIG-002`, `ARC-V3-001`, and
  `ARC-OWNERSHIP-002`.
- Current evidence:
  - `SessionDefaults` uses `omitempty` slices.
  - JSON cloning and collection-copy helpers can convert explicit empty
    collections to `nil`.
  - Agent and Workspace contributions can append after initial Session
    collections.
  - `appendUniqueMCPSelection` can replace a more-specific explicit selection
    with a later inherited selection.
  - An Agent-contributed Workspace can be assigned to
    `plan.defaults.Workspace` while Workspace application checks an earlier
    local `defaults.Workspace`.
  - `sessionv3runtime/llm_plan.go` centrally interprets Agent capability
    occurrences, Skill use modes, Text insertion behavior, Workspace overrides,
    and Workspace contribution assembly.
- Required result:
  - Preserve inheritance, replacement, explicit empty clearing, and supported
    per-item removals and overrides.
  - Apply inherited contributions before more-specific Session and invocation
    decisions.
  - Preserve explicit MCP Tools, resources, templates, prompts, and overrides.
  - Apply the effective selected Workspace.
  - Preserve accurate Agent, Workspace, Session, and invocation provenance.
  - Preserve selected Model system-prompt behavior.

##### `V3-CONFIG-003.a` Preserve collection policy durably

- Status: Not implemented.
- Required result:
  - Carry collection policy separately from resulting collections.
  - Distinguish inherit, replace, explicit empty clearing, per-item removal,
    and supported override behavior.
  - Prevent Agent and Workspace inheritance from recreating explicitly removed
    Tools, Skills, MCP selections, or instructions.

##### `V3-CONFIG-003.b` Apply effective contribution precedence

- Status: Partial.
- Required result:
  - Preserve specific explicit MCP selections over inherited selections.
  - Apply the effective selected Workspace.
  - Preserve selected Tools, resources, templates, prompts, instructions, and
    overrides in their intended order.

##### `V3-CONFIG-003.c` Retain truthful ownership and ordering

- Status: Partial.
- Required result:
  - Preserve actual contribution provenance instead of copying complete
    effective collections into unrelated owner records.
  - Keep Agent, Workspace, Session, and invocation ownership explicit.
  - Move source-family contribution interpretation to owning domains.

#### `DOMAIN-SKILL-003` Reconstruct and release Skill resources for the exact owner

- Status: Partial.
- Corrected prior claims: `DOMAIN-SKILL-002` and `ARC-SKILL-001`.
- Current evidence:
  - Private Skill sessions are keyed by Execution and ProgramFrame.
  - `prepareLLMPlan` calls unscoped `EffectiveActiveSelections`.
  - Active-state lookup searches across the Execution.
  - Lookup selects a latest global result before checking enabled-set
    compatibility.
  - Long-lived conversation Executions retain sessions for completed Agent
    frames because natural terminal paths do not consistently notify cleanup.
- Required result:
  - Reconstruct active Skill state for the exact invocation owner.
  - Select the latest applicable terminal semantic result for the owner and
    enabled pins.
  - Keep parallel Agent invocations independent.
  - Release private Skill sessions at natural ProgramFrame and Execution
    completion.

##### `DOMAIN-SKILL-003.a` Select applicable active Skill state

- Status: Partial.
- Required result:
  - Filter by semantic owner and enabled pin set before selecting the latest
    result.
  - Do not allow another frame or incompatible enabled set to hide an
    applicable earlier state.

##### `DOMAIN-SKILL-003.b` Release Skill sessions on natural completion

- Status: Partial.
- Required result:
  - Runtime sends lifecycle notification at natural frame and Execution
    termination.
  - Skill Runtime releases the exact corresponding private session.

#### `V3-CONTINUE-003` Complete active-operation continuation and manual Tool ownership

- Status: Partial.
- Corrected prior claims: `V3-CONTROL-002` and parts of `WF-HANDOFF-002`.
- Current evidence:
  - `normal_session.go:consumeContinuationLocked` submits staged input through
    the conversation frame.
  - A live Agent child waiting for explicit continuation can leave that input
    queued behind the Agent that must consume it.
  - `ContinueExecution` can return after consuming staged input without
    resuming the active Agent round.
  - Manual Tool controls can address a conversation queue or whole Execution.
  - Normal Tool mutation can rewrite consumed historical results.
  - Continuation can broadcast across Agent frames.
- Required result:
  - Consume staged text, attachments, defaults, budget, approved App context,
    and settled Tool results in the next round of the active owning operation.
  - Scope manual retry, skip, completion-as-error, and continuation to the
    current unconsumed operation.
  - Preserve actual effect certainty.
  - Keep App-originated and unrelated Workflow work outside normal Tool-result
    continuation.

##### `V3-CONTINUE-003.a` Resume the active conversation operation

- Status: Partial.
- Required result:
  - Do not queue continuation behind the Agent round that needs it.
  - Consume staged input exactly once into the correct next Model round.
  - Include settled Tool results and approved App context.

##### `V3-CONTINUE-003.b` Scope manual controls to the current operation

- Status: Partial.
- Required result:
  - Preserve consumed-result ownership.
  - Do not mutate historical Tool results already consumed by a later Model
    round.
  - Do not broadcast continuation into unrelated Agent frames.
  - Do not replace confirmed-applied effect facts with synthetic
    confirmed-not-applied facts.

#### `WF-HANDOFF-003` Complete durable multi-message handoff ownership

- Status: Partial.
- Corrected prior claims: `WF-HANDOFF-002`.
- Current evidence:
  - Agent-loop queues retain activity IDs but not each message's effective
    defaults and budget.
  - When a no-Tool handoff Agent has queued work, it can return
    `startQueuedAgentLLM` without retaining assistant Activities already
    constructed for the completed round.
  - A successful no-Tool handoff round remains waiting.
  - No typed successful completion or relinquish operation exists.
- Required result:
  - Preserve each queued message's assistant round, immutable configuration,
    budget, and causal isolated history.
  - Complete or relinquish handoff successfully.
  - Apply `restore` or `retain` without using cancellation as success.

##### `WF-HANDOFF-003.a` Preserve queued message semantic facts

- Status: Partial.
- Required result:
  - Retain each completed assistant Activity exactly once.
  - Retain immutable defaults and budget for each accepted message.
  - Extend isolated history causally across messages handled by the handoff
    owner.

##### `WF-HANDOFF-003.b` Complete or relinquish handoff successfully

- Status: Not implemented.
- Required result:
  - Add a typed backend lifecycle operation that settles successful handoff.
  - Apply restore or retain behavior.
  - Do not use cancellation as a successful completion substitute.

### Domain setup ownership, binder composition, attachment meaning, and duplication

#### `ARC-DOMAIN-003` Complete owning-domain setup boundaries

- Status: Partial.
- Corrected prior claims: `ARC-OWNERSHIP-002`.
- Current evidence:
  - `S/sessionv3runtime/llm_plan.go` interprets capability occurrences,
    insertion modes, Workspace overrides, and contribution selection.
  - `S/workflowruntime/model_agent_adapter.go` independently selects Agent
    Models.
  - V3 attachment materialization remains in Session setup.
  - Native Model request conversion and Model-default adaptation are not fully
    owned by Model setup.
- Required result:
  - Agent setup owns Agent contribution semantics and Model selection.
  - Workspace setup owns Workspace selection and contribution semantics.
  - Text setup owns Text contribution preparation.
  - Attachment setup owns admission and content materialization.
  - Model setup owns provider input, output, and Model-default adaptation.
  - Session setup composes domain facts.
  - Workflow setup lowers orchestration intent.
  - Core Runtime receives only neutral semantic facts.

##### `ARC-DOMAIN-003.a` Move contribution preparation to owning domains

- Status: Not implemented.
- Required result:
  - Remove duplicated Agent Model selection.
  - Provide neutral Agent, Workspace, Text, Skill, and Model contribution facts
    from their owning setup domains.
  - Keep Session and Workflow setup as compositional consumers.

##### `ARC-DOMAIN-003.b` Move attachment and Model conversion to owning domains

- Status: Partial.
- Required result:
  - Move V3 attachment materialization to Attachment setup.
  - Keep semantic history selection in Session orchestration.
  - Move native provider request and output conversion, including Model-default
    adaptation, to Model setup.
  - Preserve MCP, Skill, Tool, Text, Service, and Activity native semantics.

#### `V3-API-003` Preserve optional backend capabilities through final binder composition

- Status: Partial.
- Corrected prior claims: `V3-SETUP-002`.
- Current evidence:
  - The base V3 binder implements `PrepareUserMessageSkillTemplate` and
    `PreviewInstructionSkill`.
  - `S/workflowruntime/binder.go` forwards App operations but does not supply
    those Skill forwarding methods.
  - Runtime discovers optional capability interfaces from the composed binder.
- Required result:
  - The composed Workflow binder delegates both optional Skill capabilities to
    Base.
  - Do not create a second rendering implementation.

##### `V3-API-003.a` Forward Skill template preparation and preview

- Status: Not implemented.
- Required result:
  - Forward `PrepareUserMessageSkillTemplate`.
  - Forward `PreviewInstructionSkill`.
  - Preserve existing base implementation ownership.

#### `DOMAIN-ATTACH-003` Preserve semantic attachment identity in native Model input

- Status: Partial.
- Current evidence:
  - `sessionv3runtime/attachments.go` snapshots content and digests.
  - `v3AttachmentFromContentBlock` stores raw text.
  - `inputs.go:attachmentToContentItem` can send text without its label.
  - Directory children can use only basenames.
- Required result:
  - Preserve meaningful file labels in provider content.
  - Preserve stable root-relative directory identity.
  - Distinguish same-named files from different directories.
  - Continue replaying admitted snapshots rather than reopening native paths.

##### `DOMAIN-ATTACH-003.a` Carry labels and directory-relative names

- Status: Partial.
- Required result:
  - Retain semantic display labels during provider conversion.
  - Retain root-relative identity for directory children.
  - Keep one authoritative admitted content snapshot.

#### `ARC-DUP-003` Remove replicated effective state and duplicated execution algorithms

- Status: Partial.
- Corrected prior claims: `ARC-DUP-002`.
- Current evidence:
  - `enrichDurableLLMDispatchPlan` copies effective collections into Workspace,
    Agent, and Agent.Workspace records.
  - `LLMToolOriginForWorkItem` deep-copies a complete Model plan into each
    generated Tool.
  - Message and attachment plans repeat attachment content.
  - `evaluate` and `evaluateWithActivation` duplicate recursive Workflow value
    evaluation.
  - Retry and catch matching duplicate failure matching behavior.
  - Error cloning and operation-input extraction have equivalent
    implementations.
  - Channel namespacing, resource namespacing, and call lookup repeat
    lowered-tree traversal.
  - Several failure paths independently release claims, advance frames, and
    launch work.
- Required result:
  - Retain one canonical immutable effective plan with truthful provenance.
  - Reference immutable origin plans and authoritative attachment content rather
    than deep-copying them for every consumer.
  - Consolidate shared Runtime semantic algorithms.
  - Retain one authoritative settlement and ready-work launch lifecycle.
  - Do not create a generic native-domain envelope.

##### `ARC-DUP-003.a` Retain one canonical effective plan

- Status: Partial.
- Required result:
  - Retain owner references and actual contribution facts instead of complete
    copied effective collections in Workspace, Agent, and Agent.Workspace
    records.
  - Preserve exact immutable invocation meaning.

##### `ARC-DUP-003.b` Reference canonical origin plans and attachment content

- Status: Partial.
- Required result:
  - Replace per-Tool complete plan copies with immutable canonical origin-plan
    references plus required Tool-specific facts.
  - Retain one authoritative content snapshot for messages and attachments.
  - Do not reintroduce current-default lookup for generated Tools.

##### `ARC-DUP-003.c` Consolidate Runtime algorithms and lowered-tree traversal

- Status: Partial.
- Required result:
  - Consolidate recursive evaluation, retry and catch matching, error cloning,
    operation-input extraction, channel and resource namespacing, call lookup,
    and equivalent retention calculations.
  - Keep shared algorithms Runtime-owned where they are semantic orchestration
    behavior.
  - Retain domain-native request grammar in owning domains.
