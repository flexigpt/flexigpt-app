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

Workflow declaration and static Artifact planning are accepted as complete for
this implementation cycle.

This HLD does not publish a completion percentage. Completion estimates are
reported outside this status record.

The remaining work is composed Runtime behavior: preparation, dispatch,
settlement, cancellation, cleanup, recovery, exact retained operation meaning,
scoped accounting, continuation, and domain ownership.

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

Functional tasks in this HLD must not create test work, validation-only work,
frontend work, SessionV2 retirement work, stored-session conversion work,
migration cleanup work, cost-metering work, or version-change work. These
concerns remain excluded when a Runtime task changes adjacent code.

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

`Current task record` at the end of this ledger is part of this ledger. For a
task identifier repeated there, its status, source evidence, open behavior, and
completion result replace every earlier status claim for that identifier.

An item is `Implemented` only when every listed functional result exists in
source. Existing types, helper functions, partial paths, or status edits do not
complete an item.

Each open task has a `Completion evidence` field. The implementer must replace
that field with source paths, symbols, and the connected behavior that now
exists before changing the status. The implementer must then remove obsolete
gap text for the same task from earlier entries in this ledger.

This remains one substantial WIP delivery. It must not be split into
status-only updates, small implementation phases, or isolated helper changes.

`R` means `internal/agentruntime-go/sessionv3/`.

`S` means `internal/agentruntimesetup/`.

### Runtime settlement, cleanup, scheduling, commands, and persistence

#### `RT-SETTLE-003` Authoritative mutation and terminal propagation

- Status: Implemented.
- Implemented evidence:
  - `Runtime.persistLocked` wakes budget and resource waiters before it
    snapshots durable state.
  - `Runtime.persistLocked` launches all dependency-ready Session work only
    after the authoritative checkpoint commits.
  - Restore sets `Runtime.restoring`, which prevents checkpoint-time launch
    until the complete Recovery source has returned.
  - Wait resolution routes invalid resolver output through `finishWorkLocked`
    instead of bypassing owner advancement, effect settlement, and wakeup.
  - Earlier patches re-acquire Session-owned objects after durable wait,
    channel, emit, manual Tool, panic, and frame mutations.
- Current evidence:
  - Source review only. Test and application execution were not reviewed.
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

- Status: Implemented.
- Implemented evidence:
  - `R/runtime_durability.go:effectiveWorkDeadlineLocked` and
    `effectiveAttemptDeadlineLocked` select the cleanup deadline for eligible
    cleanup WorkItems.
  - `R/runtime.go:dispatchOne` uses the effective deadline during binding,
    post-binding dispatch, durable waits, and Attempt creation.
  - `R/coordination.go`, `retry.go`, `runtime_resources.go`, `stream.go`, and
    `ports_runtime.go` distinguish normal cancellation from eligible cleanup.
  - `R/runtime_children.go:childFrameDeadlineInputs` prevents cleanup child
    frames from inheriting expired ordinary deadlines.
  - `R/workflow_saga.go:workflowProgramRunsCleanup` recursively recognizes
    cleanup work inside retry-boundary ProgramSnapshots.

##### `RT-CLEANUP-003.b` Retain Saga ownership until late outcomes settle

- Status: Implemented.
- Implemented evidence:
  - `workflow_saga.go:beginSagaCompensation` keeps a Saga pending when its
    first compensation order is empty but associated normal work or child
    frames can still settle.
  - `workflowSagaHasOutstandingNormalWork` includes active Saga-scoped
    Workflow frames as well as WorkItems.
  - `finishWorkLocked` calls `resumeSagaCompensationFramesLocked` after both
    ordinary and cancellation settlement paths.
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

- Status: Implemented.
- Implemented evidence:
  - `wakeBudgetConcurrencyWaitersLocked` and
    `wakeResourceWaitersLocked` report durable state change.
  - `persistLocked` invokes the wake path for every Execution before the
    checkpoint candidate is built and touches affected Execution revisions.
  - The post-commit ready-work launch covers Binding-to-Waiting, binding
    failure, port failure, panic, manual settlement, retry, resource, and
    cross-Execution settlement paths.
- Required result:
  - Every counted-slot release runs eligible durable wakeup and ready-work
    launch.
  - A durable wait must not require an unrelated event to restart available
    work.

##### `RT-BUDGET-003.b` Apply structured limits and semantic Workflow depth

- Status: Implemented.
- Implemented evidence:
  - Scope, Choose, Parallel, Foreach, Repeat, Machine, and Saga child frame
    intents carry their controlling `step.Limits`.
  - `childProgramFrameDepth` increments only for
    `ProgramFrameIntent.WorkflowNesting`; retry, Fork, Join, and ordinary
    structured-block frames no longer consume Workflow nesting depth.
- Required result:
  - Apply construct-local narrowed limits to Scope, Choose, Parallel, Foreach,
    Repeat, Machine, and Saga work.
  - Preserve parent aggregate accounting.
  - Count nested Workflow depth separately from internal Fork, retry, and
    structured-block frames.

##### `RT-BUDGET-003.c` Reserve outer child-call work before child dispatch

- Status: Implemented.
- Implemented evidence:
  - The durable Fork WorkItem for a child call depends on the matching outer
    `ChildCall` WorkItem.
  - A Fork dependency accepts a ChildCall only after its binding has persisted
    the outer dispatched Attempt and durable completion wait.
  - Child ProgramFrames cannot materialize or dispatch native work before
    outer call budgeting, ownership, effect state, and Attempt facts exist.
- Required result:
  - Persist the logical outer Attempt, effective policy, accounting, and
    ownership before a child performs native work.
  - A child must not bypass the outer budget decision.

#### `RT-TRIGGER-003` Resume trigger lifecycle changes and expose one event meaning

- Status: Implemented.
- Corrected prior claims: `RT-TRIGGER-002`, `ARC-SCHEDULER-001`, and complete
  trigger queue resumption claims.
- Implemented evidence:
  - `R/runtime.go:SetTriggerRegistrationState` calls
    `recoverSessionTriggersLocked` and `drainTriggerQueuesLocked` while the
    registration-state command owns the Session mutation. The normal
    `changeSession` checkpoint and ready-work launch then commit and dispatch
    the resulting work.
  - `R/workflow_triggers.go:TriggerEventEnvelope` serializes `type`.
  - `R/runtime_triggers.go` maps accepted trigger command `Name` to `Type` and
    maps the same `Type` to `ExecutionSignal.Name`.
  - Event waits and Machine waits continue to use `ExecutionSignal.Name` as
    their durable event type.

##### `RT-TRIGGER-003.a` Drain eligible occurrences after reactivation

- Status: Implemented.
- Implemented evidence:
  - `R/runtime.go:SetTriggerRegistrationState` recovers missed schedule
    occurrences and drains trigger queues before the command checkpoint.
  - `R/runtime_durability.go:resumeRestoredWork` and
    `R/runtime.go:changeSession` use the same durable ready-work launch path.

##### `RT-TRIGGER-003.b` Use one declared event projection

- Status: Implemented.
- Implemented evidence:
  - `R/workflow_triggers.go` serializes trigger event `type`, source, identity,
    data, correlation, and time.
  - `R/coordination.go:workflowEventWaitOutput` retains the same event type
    through durable ordinary event waits and Machine transitions.

#### `RT-COMMAND-003` Replay accepted commands independently of regenerated preparation

- Status: Implemented.
- Implemented evidence:
  - `Runtime.FindCommandReceipt` finds accepted Session and tombstone receipts
    before setup-owned work begins.
  - Submit-message, continuation-input, and edit-and-resend paths check for a
    durable receipt before attachment admission.
  - The Wails Workflow start and trigger-registration wrappers check a stable
    caller identity before Session reads or Workflow preparation.
  - `StartWorkflowCommand` and `RegisterWorkflowTriggersCommand` persist
    caller identities rather than regenerated ProgramSnapshot payloads.
- Required result:
  - An accepted command returns its prior receipt without requiring former
    source files, attachments, selected Artifacts, or setup resources.
  - Stable caller payload identity is preserved across preparation.
  - Runtime does not own Workflow preparation.

##### `RT-COMMAND-003.a` Preserve public command identity before preparation

- Status: Implemented.
- Implemented evidence:
  - Workflow wrapper identities contain the original command metadata,
    Session, optional Execution, Workflow selection, and canonical input.
  - Runtime command fingerprinting uses those identities when supplied.
  - Attachment-backed commands query durable command records before external
    attachment materialization.
- Required result:
  - Use the original caller identity to find prior accepted receipts.
  - Preserve the caller identity across prepared snapshots.
  - Do not repeat attachment admission or source preparation for an accepted
    duplicate command.

##### `RT-COMMAND-003.b` Keep generated occurrence time outside caller identity

- Status: Implemented.
- Implemented evidence:
  - `R/runtime_triggers.go:acceptTriggerEvent` copies the caller command to
    `identity` before it assigns a Runtime occurrence time.
  - `changeSession` fingerprints and persists `identity`, while the closure
    uses the normalized command to persist the generated occurrence fact.

#### `RT-PERSIST-003` Complete deletion commitment and idle Session deletion

- Status: Implemented.
- Corrected prior claims: `RT-PERSIST-002` and complete normal V2 replacement
  claims.
- Implemented evidence:
  - `internal/sessionstorev3/store.go:HandleSessionTombstone` writes the
    tombstone first, schedules derived repair when index deletion fails, and
    returns `DerivedIndexError`, which implements `CheckpointCommitStatus`.
  - `R/runtime.go:DeleteSession` recognizes committed checkpoint errors and
    does not restore the live Session.
  - `R/runtime.go:idleNormalConversationExecution` permits deletion only for
    an idle conversation root with no active WorkItem, wait, cleanup, resource
    lease, pending Fork or Join, uncertain work, or live child frame.

##### `RT-PERSIST-003.a` Treat committed tombstones as committed

- Status: Implemented.
- Implemented evidence:
  - `sessionstorev3.Store.HandleSessionTombstone` returns
    `DerivedIndexError` only after durable tombstone publication and schedules
    repair.
  - `R/runtime.go:DeleteSession` accepts errors whose
    `CheckpointCommitStatus` reports a committed tombstone.

##### `RT-PERSIST-003.b` Delete idle long-lived normal conversations

- Status: Implemented.
- Implemented evidence:
  - `R/runtime.go:idleNormalConversationExecution` distinguishes an idle
    long-lived conversation root from active external, cleanup, child, wait,
    or uncertain work.
  - `R/runtime.go:DeleteSession` cancels due-time callbacks, discards private
    bindings, and requests owning-domain execution-binding cleanup after the
    committed Session deletion.

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

- Status: Implemented.
- Implemented evidence:
  - Workflow Model and Agent targets retain
    `WorkflowPreparedInvocation` snapshots.
  - Snapshots retain exact Artifact pins for Model, Agent, Workspace, Text,
    Tool, Skill, and MCP contributions.
  - `sessionv3runtime.applyPreparedWorkflowInvocation` consumes those pinned
    facts and does not resolve a current Agent or Workspace contribution graph
    during Workflow child dispatch.
  - Text, Tool, Skill, and MCP domains reconstruct only their private current
    resources after exact source-pin verification.
- Required result:
  - Lower prepared transitive selections into neutral Runtime snapshots.
  - Preserve exact prepared Model, Agent, Text, Tool, Skill, MCP, Workspace,
    and nested Workflow meaning.
  - Reconstruct only private current resources at dispatch.

##### `WF-PREPARE-003.b` Resolve an owner Model only where required

- Status: Implemented.
- Implemented evidence:
  - `S/workflowruntime/admission.go:admitPlanned` no longer eagerly resolves
    an owner Model.
  - `bindTarget` calls `workflowOwnerModel` only after Tool setup identifies a
    provider-managed SDK Tool target.
  - Nested Workflow preparation receives inherited owner Model and invocation
    defaults only when provider-managed Tool lowering requires them.
  - Pure deterministic, registered-binding, ordinary Tool, MCP, nested
    Workflow, Model, and Agent Workflow programs no longer depend on unrelated
    Session conversation Model defaults.

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

- Status: Implemented.
- Implemented evidence:
  - `startWorkflowRetry` resolves a resolver-backed target set before retry
    child materialization.
  - `captureWorkflowRetryCall` freezes target, input, context, typed messages,
    instructions, keyed effect values, and the current captured message view.
  - Retry child programs inherit the resolved durable target-set membership.
- Required result:
  - Capture evaluated target, target-set membership, input, context, typed
    messages, instructions, relevant port inputs, key, effect policy, and
    logical time.
  - Capture before dispatch or child creation.
  - Do not observe later channel contents, state, or settlement time.

##### `WF-RETRY-003.b` Apply complete retry selection and effect replay semantics

- Status: Implemented.
- Implemented evidence:
  - `compiler_coordination.go:compileWorkflowRetry` retains
    `match.schema` in the typed Runtime retry matcher.
  - `retryMatches` validates declared retry schemas against durable typed
    errors.
  - `retryBoundaryReplaySafe` permits declared read, idempotent, keyed, and
    confirmed-not-applied replay paths before it rejects uncertain outcomes.
- Required result:
  - Retain `match.schema`.
  - Reuse declared retry matching.
  - Permit only policy-authorized read, idempotent, keyed, or
    confirmed-not-applied retries.
  - Consider declared replay-safe effects before rejecting an uncertain result.

##### `WF-RETRY-003.c` Preserve native Model and Agent failure classification

- Status: Implemented.
- Implemented evidence:
  - Agent-loop and single-model child settlement propagate the original
    `SanitizedError` category, effect state, details, and causes.
  - Runtime no longer replaces settled Model and Agent failure facts with a
    generic string-only child-frame error.
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
  - `WorkflowScopeContext.ChannelAliases` now retains declared-to-private
    channel mapping for one nested Workflow invocation.
  - `workflow_messages.go:captureWorkflowMessages` restores declared names for
    expression-map keys.
  - `WorkflowInterpreter.drive` recaptures messages when it selects a step or
    root output. Persisted call-snapshot reuse remains open.
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

- Status: Implemented.
- Implemented evidence:
  - `R/ports_runtime.go:namespaceWorkflowProgramChannels` persists
    `WorkflowScopeContext.ChannelAliases`.
  - `R/workflow_messages.go:workflowDeclaredChannelName` publishes declared
    names to CEL while `ChannelLog` keeps private invocation-local names.

##### `WF-CHANNEL-003.b` Capture each evaluation boundary

- Status: Implemented.
- Implemented evidence:
  - `EvaluationBoundary.Messages` retains the authorized declaration-visible
    channel snapshot for WorkItems and child ProgramFrames.
  - Workflow transitions capture messages when they create WorkItem and frame
    intents.
  - `advanceFrameLocked` refreshes the Workflow frame boundary snapshot after
    each interpreted event.
  - Retry capture freezes evaluated call templates before later channel
    messages can alter a replayed operation.
- Required result:
  - Capture a new authorized channel snapshot for the declaration boundary
    actually being evaluated.
  - Do not allow a later step or root output to inherit a prior boundary's
    authorization or snapshot.
  - Retain the original snapshot for a captured logical call.

##### `WF-CHANNEL-003.c` Preserve inherited subscription identity

- Status: Implemented.
- Implemented evidence:
  - `R/workflow_coordination.go:WorkflowSubscriptionState` persists
    `OwnerFrameID`.
  - `R/workflow_program.go:ensureWorkflowSubscriptionOwners` assigns the
    current ProgramFrame identity to locally activated subscriptions while
    preserving inherited owner identities.
  - `R/workflow_machine.go:startMachine` records machine listener ownership.
  - `R/subscription_runtime.go:subscriptionConsumptionForWait` uses the
    originating subscription owner instead of the transient child wait frame.

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

- Status: Implemented.
- Implemented evidence:
  - `S/workflowruntime/compiler.go:compileWorkflowSourceEffectKind` explicitly
    maps source `non-idempotent` to Runtime `EffectKindNonIdempotent`.
  - Target-level keyed capability declarations continue to lower to an
    idempotent base contract with separate keyed capability facts.

##### `WF-EFFECT-003.b` Settle MCP from actual domain evidence

- Status: Implemented.
- Implemented evidence:
  - `S/mcpruntime/v3.go:V3Executor.Execute` returns confirmed success without
    forcing `EffectStateUncertain`; Runtime derives read `notApplicable` and
    mutating `confirmedApplied` from the durable receipt.
  - MCP error responses remain uncertain for mutating WorkItems and preserve
    their Tool output and typed uncertainty error.
  - Read-only MCP error responses remain confirmed read failures and do not
    invent a mutating business effect.

##### `WF-EFFECT-003.c` Preserve compensation causality and operation facts

- Status: Implemented.
- Implemented evidence:
  - `WorkflowState` and child scope snapshots retain causal effect-frontier
    WorkItem identities.
  - Scope, Choose, Parallel, Foreach, Repeat, Machine, Saga, nested Workflow,
    and retry boundaries preserve or merge those frontiers.
  - Later mutating WorkItems depend on the complete causal frontier; parallel
    branches remain independent until a later dependent operation joins them.
  - `WorkflowCompletedOperationFacts` now retains original WorkItem, Attempt,
    ProgramFrame, target binding facts, effect contract and state, receipt,
    idempotency key, and compensation WorkItem and frame identities.
  - Existing compensation ordering continues to use persisted dependencies,
    so concurrent compensation occurs only when no reverse dependency blocks
    it.

#### `WF-CONTROL-003` Complete structured-control value and lifetime behavior

- Status: Partial.
- Corrected prior claims: `WF-CONTROL-001`, `WF-SCOPE-001`, and
  `WF-MACHINE-002`.
- Current evidence:
  - Empty Foreach with parent-state merge can create merge progress and return
    idle without emitting required merge work.
  - Empty runtime collections under non-`all` completion can remain pending.
  - Lowering loses the difference between omitted and explicit zero
    `maxItems` or `maxIterations`.
  - Repeat condition evaluation now receives the current Repeat index.
  - Machine immediate-transition accounting now resets at child-block and wait
    suspension boundaries.
- Required result:
  - Apply current Repeat index at every Repeat condition boundary.
  - Execute empty-collection merge behavior.
  - Settle empty collections truthfully.
  - Preserve explicit maximum values.
  - Limit consecutive immediate Machine transitions without limiting the
    lifetime count of accepted events.

##### `WF-CONTROL-003.a` Apply Repeat index to condition evaluation

- Status: Implemented.
- Implemented evidence:
  - `R/workflow_structured.go:repeatAllowsNext` supplies the next iteration
    index to `while`.
  - `R/workflow_structured.go:settleRepeatChild` supplies the completed body
    index to `until` with carried state and result facts.

##### `WF-CONTROL-003.b` Settle empty collections and preserve maxima

- Status: Implemented.
- Implemented evidence:
  - `R/types.go` retains `WorkflowForeach.MaxItems` and
    `WorkflowRepeat.MaxIterations` as pointers.
  - `S/workflowruntime/compiler_structured.go` preserves explicit zero values
    from the admitted declaration.
  - `R/workflow_structured.go:launchForeachChildren` re-enters
    `drivePending` after an empty collection completes so a declared parent
    state merge emits its durable StateUpdate WorkItem.
  - Empty non-`all` Foreach completion creates a typed no-selected-item or
    quorum-unreachable failure rather than waiting indefinitely.

##### `WF-CONTROL-003.c` Limit immediate Machine transitions only

- Status: Implemented.
- Implemented evidence:
  - `R/workflow_machine.go:selectMachineTransition` increments the counter
    only for immediate transitions.
  - Event and timer transitions reset it.
  - `startMachineChild` and `armMachineWaits` reset it at genuine suspension
    boundaries while retaining the immediate-transition bound.

#### `WF-OUTCOME-003` Preserve complete Workflow and conversation terminal meaning

- Status: Partial.
- Current evidence:
  - Terminal outcome recording now prefers the post-transition active
    declaration path and includes the settled WorkItem receipt, semantic
    output, target, effect state, and typed error where available.
  - Inline `fail`, `return`, `break`, and `continue` record their actual
    source-step outcomes before they clear the pending declaration boundary.
  - Model and Agent child-frame failures now retain the settled LLM
    `SanitizedError` rather than replacing its category and causes with a
    generic string-only error.
  - Recovery and some cross-Execution terminal aggregation paths still need
    consolidation with the common settlement lifecycle.
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

- Status: Implemented.
- Implemented evidence:
  - `R/program.go:interruptedAssistantActivities` persists usable final or
    streamed assistant text with failed, cancelled, or uncertain status.
  - Agent-loop and single-model interpreters attach that Activity to their
    terminal failure transition.
  - The originating WorkItem retains the provider output, receipt, and actual
    error. The Activity does not create a successful Model or Tool result.

### Conversation replay, configuration, continuation, handoff, and Skill state

#### `V3-REPLAY-003` Reconstruct private resources from the exact retained operation

- Status: Implemented.
- Implemented evidence:
  - `ModelDispatchPlan.RuntimeConfiguration` retains a typed non-secret
    Provider, Model, capability-override, and credential-binding snapshot.
  - `modelruntime.PreparedRuntimeConfiguration` reconstructs only private
    credential values through the original Store-validated binding.
  - `bindRecoveredLLM` uses the durable configuration snapshot and captured
    message sequence. It does not resolve current Model overlays, Agent
    capabilities, Workspace capabilities, or Session defaults.
  - Missing legacy snapshots fail preparation rather than silently substituting
    current source configuration.
- Required result:
  - Retain immutable semantic dispatch facts while discarding private resources.
  - Preserve exact semantic message order.
  - Retain effective non-secret Model and Provider configuration.
  - Reconstruct credentials and clients privately.
  - Never substitute current defaults or current contribution assembly.

##### `V3-REPLAY-003.a` Retain semantic LLM bindings

- Status: Implemented.
- Implemented evidence:
  - Durable LLM bindings retain canonical dispatch plans, source pins, exact
    provider inputs, Tool choices, immutable contribution facts, and prepared
    non-secret Model runtime configuration.
- Required result:
  - Preserve immutable plans for dispatched replay-safe work and equivalent
    interrupted preparation paths.
  - Clear only private resources.

##### `V3-REPLAY-003.b` Reproduce captured message order and configuration

- Status: Implemented.
- Implemented evidence:
  - Replay reconstructs the exact durable message sequence.
  - Provider and Model request configuration comes from
    `PreparedRuntimeConfiguration`, not current source defaults.
  - Credential resolution remains private and validates the retained binding.
- Required result:
  - Rebuild from captured message identities and causal order.
  - Retain effective Provider, Model, capability, instruction, context, and
    non-secret overlay facts.
  - Resolve credentials and clients privately without current-default
    substitution.

#### `V3-CONFIG-003` Complete inherited and explicit invocation configuration

- Status: Implemented.
- Implemented evidence:
  - `SessionCollectionPolicies` preserves collection replacement, explicit
    empty clearing, and removed keys independently from JSON slice omission.
  - Invocation overrides set replacement policy whenever a collection pointer
    is supplied, including an explicitly empty collection.
  - `prepareLLMPlan` applies Agent and Workspace inherited contributions first,
    then applies specific Session and invocation collection policy.
  - Later specific values replace matching inherited selections without
    allowing inherited Agent or Workspace contributions to recreate removed
    entries.
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

- Status: Implemented.
- Implemented evidence:
  - `CollectionPolicy` stores mode and removed keys.
  - `SessionDefaults` persists one policy for each supported collection.
  - `SessionDefaultsOverride` preserves explicit empty replacement through
    pointer presence and stores explicit policy when supplied.
- Required result:
  - Carry collection policy separately from resulting collections.
  - Distinguish inherit, replace, explicit empty clearing, per-item removal,
    and supported override behavior.
  - Prevent Agent and Workspace inheritance from recreating explicitly removed
    Tools, Skills, MCP selections, or instructions.

##### `V3-CONFIG-003.b` Apply effective contribution precedence

- Status: Implemented.
- Implemented evidence:
  - Inherited Agent and Workspace contributions are assembled before specific
    Session and invocation collections are overlaid.
  - Per-key overlay replaces an inherited value only with its more-specific
    Session or invocation value.
  - Removed keys are filtered before specific values are applied.
- Required result:
  - Preserve specific explicit MCP selections over inherited selections.
  - Apply the effective selected Workspace.
  - Preserve selected Tools, resources, templates, prompts, instructions, and
    overrides in their intended order.

##### `V3-CONFIG-003.c` Retain truthful ownership and ordering

- Status: Implemented.
- Implemented evidence:
  - Canonical top-level `ModelDispatchPlan` collections retain effective
    invocation meaning.
  - Agent and Workspace dispatch records no longer copy complete effective
    Tools, Skills, MCP, Context, Text, or Instruction collections.
  - Prepared Workflow invocation snapshots retain only source contribution
    pins and declaration facts, not copied effective plans.
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

- Status: Implemented.
- Implemented evidence:
  - `S/skillruntime/v3_hydration.go` exposes
    `EffectiveActiveSelectionsForFrame`.
  - `S/sessionv3runtime/llm_plan.go` requests Skill state using the current LLM
    ProgramFrame identity.
  - The Skill Runtime filters terminal lifecycle Tool outcomes by owner frame
    and enabled Skill pins before it selects the latest applicable result.

##### `DOMAIN-SKILL-003.b` Release Skill sessions on natural completion

- Status: Implemented.
- Implemented evidence:
  - `R/runtime.go:advanceFrameLocked` invokes the optional
    `ProgramFrameBindingCloser` when a ProgramFrame reaches a terminal state.
  - `S/sessionv3runtime.Binder` and `S/workflowruntime.Binder` forward the
    frame lifecycle callback.
  - `S/skillruntime.V3Executor.CloseProgramFrame` removes and closes the exact
    Execution and ProgramFrame keyed private Skill session.

#### `V3-CONTINUE-003` Complete active-operation continuation and manual Tool ownership

- Status: Implemented.
- Corrected prior claims: `V3-CONTROL-002` and parts of `WF-HANDOFF-002`.
- Implemented evidence:
  - `R/normal_session.go:consumeContinuationLocked` routes staged input to the
    active conversation Agent ProgramFrame when one exists.
  - `R/program.go:agentLoopProgramState` retains queued Activity identity,
    invocation defaults, and invocation budget.
  - `R/runtime.go:advanceFrameLocked` persists the LLM WorkItem that consumes
    each generated Tool result.
  - `R/normal_session.go` rejects retry, skip, and manual error completion for
    Tool results already consumed by a later LLM WorkItem.
  - `requestAgentContinuationLocked` advances only the active conversation or
    handoff Agent frame.

##### `V3-CONTINUE-003.a` Resume the active conversation operation

- Status: Implemented.
- Implemented evidence:
  - `consumeContinuationLocked` uses `activeConversationAgentFrame` and
    `submitNormalMessageToFrameWithDefaultsLocked`.
  - ProgramEvent activity facts retain the exact defaults and budget required
    for the next Agent Model round.
  - Existing continuation consumption keeps approved App context and durable
    Tool-result activity facts in the resulting message Activity.

##### `V3-CONTINUE-003.b` Scope manual controls to the current operation

- Status: Implemented.
- Implemented evidence:
  - `WorkItem.ConsumedByWorkItemID` retains durable consumer identity.
  - Manual Tool retry and completion paths reject consumed results.
  - `requestAgentContinuationLocked` selects one active semantic owner instead
    of broadcasting ProgramEventContinue.
  - Existing effect settlement facts remain unchanged when manual controls are
    rejected.

#### `WF-HANDOFF-003` Complete durable multi-message handoff ownership

- Status: Implemented.
- Corrected prior claims: `WF-HANDOFF-002`.
- Implemented evidence:
  - `R/program.go` stores queued Activity ID, invocation defaults, and
    invocation budget in Agent-loop state.
  - A queued handoff round appends the completed assistant Activities before it
    starts the next queued LLM WorkItem.
  - `CompleteConversationHandoffCommand` emits
    `ProgramEventHandoffComplete`.
  - Agent-loop completion uses an ordinary completed ProgramFrame terminal
    result; `advanceFrameLocked` then invokes existing restore or retain
    ownership settlement.

##### `WF-HANDOFF-003.a` Preserve queued message semantic facts

- Status: Implemented.
- Implemented evidence:
  - `conversationQueuedActivity` carries immutable defaults and budget.
  - Agent-loop queue conversion preserves legacy queued IDs.
  - `startQueuedAgentLLM` applies queued facts to the next LLM WorkItem.
  - The preceding assistant Activity remains attached to the transition that
    starts queued handoff work.

##### `WF-HANDOFF-003.b` Complete or relinquish handoff successfully

- Status: Implemented.
- Implemented evidence:
  - `R/Runtime.CompleteConversationHandoff` targets one active handoff frame.
  - `ProgramEventHandoffComplete` completes only an idle handoff Agent frame.
  - Existing `settleConversationHandoffLocked` applies the durable restore or
    retain policy after normal ProgramFrame completion.

### Domain setup ownership, binder composition, attachment meaning, and duplication

#### `ARC-DOMAIN-003` Complete owning-domain setup boundaries

- Status: Implemented.
- Implemented evidence:
  - Model setup owns durable non-secret runtime configuration preparation and
    private credential reconstruction through `modelruntime`.
  - Workflow invocation preparation uses Agent and Workspace APIs only to
    create neutral pinned contribution snapshots.
  - Text, Tool, Skill, MCP, Attachment, and Model setup continue to own
    materialization, native request conversion, and private resource creation.
  - Session and Workflow setup now compose prepared neutral facts instead of
    rebuilding source contribution graphs at dispatch or recovery.
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

- Status: Implemented.
- Implemented evidence:
  - `WorkflowInvocationTargetAdapter` produces neutral source-pinned
    contribution snapshots through existing Agent, Workspace, and Model APIs.
  - Workflow Runtime consumes the snapshot and does not perform Agent Model
    selection or Workspace capability traversal.
- Required result:
  - Remove duplicated Agent Model selection.
  - Provide neutral Agent, Workspace, Text, Skill, and Model contribution facts
    from their owning setup domains.
  - Keep Session and Workflow setup as compositional consumers.

##### `ARC-DOMAIN-003.b` Move attachment and Model conversion to owning domains

- Status: Implemented.
- Implemented evidence:
  - V3 Attachment admission remains in `attachmentruntime`.
  - `modelruntime.PreparedRuntimeConfiguration` owns durable Model-default,
    Provider, capability override, and credential-binding adaptation.
  - `sessionv3runtime` only consumes typed Model runtime and attachment facts.
- Required result:
  - Move V3 attachment materialization to Attachment setup.
  - Keep semantic history selection in Session orchestration.
  - Move native provider request and output conversion, including Model-default
    adaptation, to Model setup.
  - Preserve MCP, Skill, Tool, Text, Service, and Activity native semantics.

#### `V3-API-003` Preserve optional backend capabilities through final binder composition

- Status: Implemented.
- Corrected prior claims: `V3-SETUP-002`.
- Implemented evidence:
  - `S/workflowruntime/binder.go` forwards
    `PrepareUserMessageSkillTemplate` and `PreviewInstructionSkill` to Base.
  - Base remains the owner of Skill rendering and preview behavior.

##### `V3-API-003.a` Forward Skill template preparation and preview

- Status: Implemented.
- Implemented evidence:
  - `S/workflowruntime.Binder` implements both optional SessionV3 capability
    interfaces and delegates without creating another rendering path.

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

- Status: Implemented.
- Implemented evidence:
  - `S/sessionv3runtime/attachments.go:admitDirectory` records a stable
    slash-normalized path relative to the admitted directory root in the
    child attachment label.
  - `S/sessionv3runtime/inputs.go:attachmentTextForProvider` emits text
    attachment labels with the immutable admitted content snapshot.
  - Images and files retain their existing admitted Label field through native
    provider content conversion without reopening paths.

#### `ARC-DUP-003` Remove replicated effective state and duplicated execution algorithms

- Status: Implemented.
- Implemented evidence:
  - `ModelDispatchPlan` is the canonical immutable LLM operation plan.
  - Generated Tool WorkItems retain `LLMToolOrigin.PlanReference` rather than
    a copied Model plan.
  - Durable message dispatch plans are the sole attachment snapshot owner;
    top-level duplicate attachment-plan population is removed.
  - Committed checkpoint handling wakes waiters and launches ready work through
    one Runtime-owned post-commit path.
- Required result:
  - Retain one canonical immutable effective plan with truthful provenance.
  - Reference immutable origin plans and authoritative attachment content rather
    than deep-copying them for every consumer.
  - Consolidate shared Runtime semantic algorithms.
  - Retain one authoritative settlement and ready-work launch lifecycle.
  - Do not create a generic native-domain envelope.

##### `ARC-DUP-003.a` Retain one canonical effective plan

- Status: Implemented.
- Implemented evidence:
  - Top-level `ModelDispatchPlan` is canonical.
  - Nested Agent and Workspace dispatch structures no longer duplicate
    complete effective collections.
- Required result:
  - Retain owner references and actual contribution facts instead of complete
    copied effective collections in Workspace, Agent, and Agent.Workspace
    records.
  - Preserve exact immutable invocation meaning.

##### `ARC-DUP-003.b` Reference canonical origin plans and attachment content

- Status: Implemented.
- Implemented evidence:
  - `LLMToolOrigin` resolves its immutable plan from the original LLM
    WorkItem through `PlanReference`.
  - New dispatch-plan construction retains attachments only in captured
    message facts.
- Required result:
  - Replace per-Tool complete plan copies with immutable canonical origin-plan
    references plus required Tool-specific facts.
  - Retain one authoritative content snapshot for messages and attachments.
  - Do not reintroduce current-default lookup for generated Tools.

##### `ARC-DUP-003.c` Consolidate Runtime algorithms and lowered-tree traversal

- Status: Implemented.
- Implemented evidence:
  - `persistLocked` is the common committed wake and ready-work launch point.
  - Workflow EvaluationBoundary snapshots provide one durable source for
    captured channel visibility at each interpreted boundary.
- Required result:
  - Consolidate recursive evaluation, retry and catch matching, error cloning,
    operation-input extraction, channel and resource namespacing, call lookup,
    and equivalent retention calculations.
  - Keep shared algorithms Runtime-owned where they are semantic orchestration
    behavior.
  - Retain domain-native request grammar in owning domains.

### Runtime setup lifecycle

#### `RT-SETUP-003` Release a restored Runtime when wrapper publication fails

- Status: Implemented.
- Source evidence:
  - `cmd/agentgo/app.go` constructs Runtime dependencies before Store v3 and
    Runtime v3 restore.
  - `cmd/agentgo/wrapper_sessionv3_runtime.go:close` closes Runtime before
    Store.
  - `cmd/agentgo/wrapper_sessionv3_bootstrap.go:InitComposedSessionV3Runtime`
    closes the restored Runtime if final `InitSessionV3Wrapper` publication
    fails.
- Completed behavior:
  - Runtime ownership transfers to the wrapper only after wrapper publication
    succeeds.
  - A failed publication does not leave restored asynchronous Runtime work
    alive against caller-owned Store resources.
  - Existing Runtime-before-Store shutdown order remains unchanged.

### Current task record

This subsection is the current functional status authority for every task ID
listed below. Earlier evidence remains useful only as historical context.

All items below remain in the current WIP implementation cycle. No item below
authorizes tests, validation-only work, frontend work, SessionV2 retirement,
stored-session conversion, migration cleanup, cost metering, or version
changes.

#### `RT-SETTLE-003` Authoritative mutation and terminal propagation

- Status: Partial.
- Source evidence:
  - `R/runtime.go:persistLocked` wakes resource and budget waiters, checkpoints
    the Session, and launches ready work after an ordinary successful commit.
  - `R/runtime.go:finishWorkLocked` contains the most complete WorkItem
    settlement path.
  - `R/coordination.go`, `R/runtime_reconciliation.go`, `R/ports_runtime.go`,
    `R/runtime_resources.go`, `R/retry.go`, and `R/runtime_child_cancel.go`
    retain additional terminal paths.
- Missing functional behavior:
  - `persistLocked` returns before ready-work launch when a Store error reports
    that the authoritative checkpoint committed but a derived update failed.
  - Ready-work selection can still see `checkpointPending` until deferred
    cleanup runs after a successful checkpoint.
  - `appendPortMessageLocked` launches work before the enclosing Session
    mutation is committed.
  - Binding failure, port-adapter failure, resource deadline, retry
    cancellation, direct child cancellation, and deadline cancellation do not
    all use the same effect, attempt, claim, completion-wait, owner-advance,
    Execution-settlement, checkpoint, and post-commit launch behavior.
  - Several paths retain Session-owned pointers after calls that can append
    frames, waits, WorkItems, child Executions, channels, or timeline records.
  - Late cancellation settlement does not consistently retain Attempt receipt
    evidence through the same path as ordinary settlement.
- Completion result:
  - Use one Runtime-owned terminal lifecycle for every WorkItem result,
    cancellation, deadline, panic, binding failure, port failure, wait result,
    reconciliation result, and child completion.
  - Reacquire Session, Execution, ProgramFrame, WorkItem, Attempt, and Wait by
    durable identity after mutation-capable Runtime calls.
  - Commit all durable consequences before any ready work launches.
  - Treat a committed authoritative checkpoint with a derived-index error as a
    committed checkpoint for post-commit launch purposes.
  - Clear an earlier checkpoint-pending barrier before selecting ready work
    after the later authoritative checkpoint succeeds.
  - Keep one Session-wide post-commit dispatch path. Remove pre-commit launch
    behavior from port and channel helpers.
- Completion evidence: Pending implementation.

##### `RT-SETTLE-003.a` Reacquire authoritative Session-owned objects

- Status: Partial.
- Source evidence:
  - `R/coordination.go` reacquires several waits and Executions by identity.
  - `R/runtime.go:advanceFrameLocked` includes a local reacquisition helper.
- Missing functional behavior:
  - Completion-wait loops can continue over earlier slices after resolving a
    wait.
  - Reconciliation can continue with earlier WorkItem and Execution values
    after port delivery or wait wakeup.
  - Stream and channel forwarding paths can update an earlier object after
    nested settlement.
  - Restore and cancellation can continue with an earlier Execution after a
    child Execution or ProgramFrame changes the Session.
- Completion result:
  - Use durable identity reacquisition at every mutation boundary that can
    invalidate slice-backed objects.
  - Do not use stale object pointers for later state mutation, readiness,
    output delivery, owner advancement, or terminal aggregation.
- Completion evidence: Pending implementation.

##### `RT-SETTLE-003.b` Use one terminal lifecycle and committed ready-work launch

- Status: Partial.
- Source evidence:
  - Ordinary `finishWorkLocked` settles attempts, effects, claims, owner
    advancement, completion waits, Saga progress, and Execution state.
  - `R/runtime.go:finishPanickedWork` now invokes a ready-work launch after
    persistence.
- Missing functional behavior:
  - `R/runtime_resources.go:waitForResource` can fail an item without
    advancing its owner.
  - `R/retry.go:waitForRetry` cancellation and deadline paths do not use the
    complete terminal lifecycle.
  - Deadline cancellation can bypass effect facts, Attempt facts, claim
    release, completion waits, and dependent owner progression.
  - Binding and transport failures duplicate only a subset of settlement.
- Completion result:
  - Route every terminal condition through the common lifecycle.
  - Preserve actual typed error, receipt, and effect evidence.
  - Wake completion and resource waiters after settlement.
  - Advance catch, finally, parent joins, independent-child waits, forwarded
    channels, and cross-Execution dependents after the committed checkpoint.
- Completion evidence: Pending implementation.

#### `RT-CLEANUP-003` Execute cleanup through every owning lifecycle

- Status: Partial.
- Source evidence:
  - `R/runtime_durability.go` provides effective cleanup work and Attempt
    deadline helpers.
  - `R/workflow_saga.go:workflowProgramRunsCleanup` recognizes Workflow and
    retry-wrapper cleanup programs.
  - `R/runtime_child_cancel.go` preserves dispatched WorkItems for later
    domain settlement instead of claiming confirmed non-application.
- Missing functional behavior:
  - Built-in Agent-loop and single-Model child programs created during cleanup
    do not retain cleanup classification through their Fork and child work.
  - Machine waits, parent-state merge work, and some synthetic Fork and Join
    work do not consistently inherit cleanup classification.
  - Wait creation can first narrow a deadline with the expired ordinary
    WorkItem deadline before it considers the cleanup deadline.
  - `R/retry.go:prepareRetryLocked` compares backoff time to the ordinary
    WorkItem deadline instead of the effective cleanup deadline.
  - Cleanup dispatch can omit an earlier per-Attempt timeout.
  - Failure-driven finally, reconciliation, and compensation can run without
    starting the bounded Execution cleanup lifecycle.
  - `Execution.Validate` rejects a cleanup deadline before the ordinary
    Execution deadline. This prevents a failure that occurs before the normal
    deadline from persisting its own bounded cleanup allowance.
  - `R/runtime.go:frameOutcomeLocked` can make an Execution terminal after a
    frame becomes terminal even though dispatched owned work still needs late
    effect settlement.
  - `R/budget.go:waitForExecutionDeadline` cancels WorkItems through an
    abbreviated path that can leave resource leases, completion waits, and
    owner transitions incomplete.
- Completion result:
  - Preserve cleanup classification and the effective bounded deadline through
    bind, dispatch, waits, retries, resource acquisition, streams, emits,
    Forks, Joins, Agent and Model child frames, state merges, settlement, and
    recovery.
  - Use the earliest applicable cleanup and per-Attempt deadline without
    reintroducing an expired ordinary deadline.
  - Start bounded cleanup for failure-driven finalization, reconciliation, and
    compensation as well as cancellation and ordinary deadline expiry.
  - Keep an owner available until dispatched work, late effect evidence,
    reconciliation, compensation, release, and dependent wakeup settle.
  - Preserve cancellation as the primary terminal meaning.
- Completion evidence: Pending implementation.

##### `RT-CLEANUP-003.a` Carry cleanup classification and effective deadline

- Status: Partial.
- Existing behavior:
  - Eligible Workflow cleanup work uses `effectiveWorkDeadlineLocked`.
  - Retry-wrapper cleanup recognition exists.
- Completion result:
  - Complete the missing built-in child, synthetic-work, wait, retry, and
    failure-driven cleanup paths listed under the parent task.
- Completion evidence: Pending implementation.

##### `RT-CLEANUP-003.c` Preserve cancellation ownership until late settlement

- Status: Partial.
- Source evidence:
  - `R/runtime_child_cancel.go:cancelFrameTreeLocked` can terminalize a frame
    while a dispatched child WorkItem remains active.
  - `R/runtime.go:frameOutcomeLocked` primarily evaluates root-frame status.
- Completion result:
  - Keep frame cancellation distinct from global Execution cancellation.
  - Keep late dispatched WorkItem settlement attached to the cancelled owner.
  - Wake direct WorkItem completion waits and release or transfer resources
    through the common lifecycle.
  - Do not finalize an Execution merely because its root frame is terminal
    while owned dispatched work, cleanup work, reconciliation, compensation,
    resource ownership, or completion delivery remains active.
- Completion evidence: Pending implementation.

#### `RT-BUDGET-003` Complete scoped accounting and resource-ready scheduling

- Status: Partial.
- Source evidence:
  - `R/budget_scope.go` applies Execution, frame, and WorkItem counters.
  - `R/runtime.go:childCallForkReservationReady` preserves an outer ChildCall
    Attempt before child native work dispatches.
  - `R/runtime_children.go:childProgramFrameDepth` distinguishes internal
    frames from Workflow nesting when `WorkflowNesting` is set.
- Missing functional behavior:
  - Nested Workflow child intents do not consistently set
    `WorkflowNesting`, so Workflow depth can remain unchanged.
  - Parallel branches, Foreach items, Repeat iterations, and Machine child
    blocks receive separate narrowed counters without one controlling-step
    aggregate.
  - User Activities, forwarded channel messages, port input and output
    messages, and trigger-delivered signals do not consistently reserve the
    metric of their semantic owner.
  - Model usage can be omitted when a provider result contains usage and an
    error.
  - Ready work can remain unavailable after a committed derived-index error or
    while the prior checkpoint-pending barrier still affects readiness.
- Completion result:
  - Preserve one Execution aggregate and one controlling structured-scope
    aggregate while child frames retain their own narrowed limits.
  - Mark actual nested Workflow calls as Workflow nesting and leave retry,
    Fork, Join, and ordinary structured frames outside that depth metric.
  - Account every retained Activity, event, channel record, and port record
    against its actual durable owner.
  - Account provider-supplied usage when it exists, independent of whether the
    provider returned an execution error.
  - Keep cost metering excluded.
- Completion evidence: Pending implementation.

##### `RT-BUDGET-003.a` Wake waiters when a counted slot becomes available

- Status: Partial.
- Existing behavior:
  - `persistLocked` wakes budget-concurrency waiters before ordinary
    checkpoint construction.
- Completion result:
  - Every slot release must reach the common committed ready-work launch,
    including releases caused by binding failure, wait transition, resource
    deadline, retry cancellation, panic, cancellation, and cross-Execution
    settlement.
- Completion evidence: Pending implementation.

##### `RT-BUDGET-003.b` Apply structured limits and semantic Workflow depth

- Status: Partial.
- Completion result:
  - Apply declared narrowed limits to Scope, Choose, Parallel, Foreach,
    Repeat, Machine, Saga, and nested Workflow work.
  - Preserve parent aggregate accounting while each child retains only its
    own narrower view.
  - Set and consume Workflow nesting only for actual nested Workflow
    boundaries.
- Completion evidence: Pending implementation.

##### `RT-BUDGET-003.d` Account retained Runtime records through their semantic owner

- Status: Partial.
- Source evidence:
  - `reserveChannelMessageBudgetLocked`, `reserveMessageBudgetLocked`, and
    `reserveEventBudgetLocked` can receive no WorkItem or frame owner.
  - `submitNormalMessageWithDefaultsLocked` and
    `signalTriggerExecutionLocked` bypass equivalent owner-specific paths.
- Completion result:
  - Associate user Activities, trigger signals, channel records, forwarded
    ports, and staged port inputs with the owning WorkItem or ProgramFrame when
    one exists.
  - Retain only one accounting algorithm for each metric.
  - Do not create a second budget scheduler or a separate durable budget root.
- Completion evidence: Pending implementation.

#### `RT-COMMAND-003` Replay accepted commands independently of regenerated preparation

- Status: Partial.
- Existing behavior:
  - `FindCommandReceipt` checks durable Session and tombstone command records
    before Workflow preparation and attachment admission.
  - Workflow wrappers retain caller-owned identity instead of regenerated
    ProgramSnapshot payload.
- Missing functional behavior:
  - `R/runtime_triggers.go:acceptTriggerEvent` assigns or normalizes
    `OccurredAt` before it copies the command into the command identity.
  - A duplicate caller command with omitted occurrence time can therefore
    fingerprint differently and fail instead of returning the prior receipt.
- Completion result:
  - Fingerprint the unchanged caller command before Runtime creates an
    occurrence time.
  - Persist generated occurrence time only as a trigger occurrence fact.
  - Preserve command replay without source preparation, Artifact reads,
    attachment admission, or regenerated occurrence time.
- Completion evidence: Pending implementation.

##### `RT-COMMAND-003.b` Keep generated occurrence time outside caller identity

- Status: Partial.
- Completion result:
  - Copy caller identity before assigning a generated occurrence time.
  - Use that unchanged identity for durable command fingerprinting.
  - Keep the normalized generated time only in the durable occurrence record.
- Completion evidence: Pending implementation.

#### `RT-RECOVERY-003` Resume recovered operations through their actual owners

- Status: Partial.
- Source evidence:
  - `R/runtime.go:recoverLocked` removes private binding state, marks
    dispatched work uncertain or replay-ready, and invalidates Runtime-bound
    waits.
  - `R/runtime.go:resolveRecoveryWaitLocked` completes the whole Execution for
    a general recovery review.
  - `R/runtime_durability.go:resumeRestoredWork` resumes work one Execution at
    a time after restore.
- Missing functional behavior:
  - A pre-invocation approval wait cannot recreate its private domain wait
    state after recovery.
  - General recovery review bypasses the original WorkItem owner, Workflow
    catch and finally behavior, cleanup, reconciliation, compensation, and
    dependent work.
  - An Execution awakened by a later Execution can be missed after the earlier
    Execution has already been traversed during restore.
- Completion result:
  - Recover an operation through its original semantic owner.
  - Recreate private bindings, authorization waits, credentials, clients, and
    domain resources only from captured semantic facts.
  - Keep private tokens, callbacks, handles, and clients outside Session
    persistence.
  - Finish restore with one committed Session-wide readiness pass.
- Completion evidence: Pending implementation.

##### `RT-RECOVERY-003.a` Reconstruct pre-invocation domain waits

- Status: Not implemented.
- Completion result:
  - Distinguish a pre-invocation authorization wait from dispatched uncertain
    business work.
  - Rebind eligible pre-invocation work from the captured request.
  - Obtain a new domain-owned private wait state when required.
  - Preserve logical operation identity, captured input, effect policy, and
    idempotency key.
- Completion evidence: Pending implementation.

##### `RT-RECOVERY-003.b` Resume the complete committed Session dependency wave

- Status: Partial.
- Completion result:
  - Reacquire Executions after restore-time interpreter advancement.
  - Persist recovered child settlement, durable wait resolution,
    reconciliation, and trigger queue mutations.
  - Launch all dependency-ready work across the Session after recovery
    mutations commit.
  - Use Runtime due-time ownership only. Setup must not create a scheduler.
- Completion evidence: Pending implementation.

##### `RT-RECOVERY-003.c` Settle recovery review through operation lifecycle

- Status: Not implemented.
- Completion result:
  - Preserve uncertainty when no domain evidence proves application or
    non-application.
  - Apply an allowed review decision to the original WorkItem owner.
  - Route failure through declared catch and finally behavior.
  - Route cancellation through bounded cleanup, child propagation,
    reconciliation, and compensation.
  - Do not force the whole Execution terminal before its owned settlement
    completes.
- Completion evidence: Pending implementation.

#### `RT-SETUP-003` Publish and release the composed backend Runtime safely

- Status: Partial.
- Source evidence:
  - `cmd/agentgo/app.go` constructs dependencies before Store v3 and Runtime
    v3 restore.
  - `cmd/agentgo/wrapper_sessionv3_runtime.go:close` closes Runtime before
    Store.
  - `cmd/agentgo/wrapper_sessionv3_bootstrap.go` restores Runtime before final
    wrapper publication.
- Missing functional behavior:
  - If final wrapper publication fails after restore,
    `InitComposedSessionV3Runtime` returns without closing the restored
    Runtime.
- Completion result:
  - Transfer Runtime ownership only after final backend publication succeeds.
  - Close a restored Runtime on publication failure.
  - Preserve Store ownership and shutdown order.
  - Do not add frontend integration work.
- Completion evidence: Pending implementation.

#### `WF-PREPARE-003` Preserve complete prepared target meaning

- Status: Partial.
- Source evidence:
  - Workflow preparation uses a static Workflow closure and creates typed
    `WorkflowPreparedInvocation` snapshots.
  - `S/workflowruntime/admission.go` resolves owner Model lazily for
    provider-managed Tool lowering.
- Missing functional behavior:
  - `R/program.go:startAgentLLM` drops `PreparedInvocation` when it creates
    the next Agent LLM WorkItem.
  - `R/retry_program.go:retryCaptureLLMInvocation` drops
    `PreparedTargets`.
  - `S/workflowruntime/model_agent_adapter.go` independently resolves Agent
    and Workspace contribution graphs after planning instead of consuming only
    prepared planned facts.
  - Prepared Workflow Agent and Workspace facts are not consistently restored
    into effective invocation ownership fields before Model dispatch.
  - Prepared Workspace use can bypass Workspace prompt decisions,
    user-message context, and effective override semantics.
- Completion result:
  - Preserve one immutable prepared invocation meaning through invoke, start,
    retry, handoff, nested Workflow preparation, recovery, and all Agent
    rounds.
  - Use exact prepared pins and domain-prepared contribution facts. Do not
    silently substitute a current Agent or Workspace graph.
  - Retain only private-resource reconstruction for dispatch.
  - Keep pure deterministic and registered-binding Workflows independent from
    unrelated Session conversation defaults.
- Completion evidence: Pending implementation.

##### `WF-PREPARE-003.a` Retain immutable pins and prepared contribution facts

- Status: Partial.
- Completion result:
  - Carry `PreparedInvocation` through every Agent LLM WorkItem.
  - Retain prepared targets through retry capture and started-call capture.
  - Retain selected Agent, Model, Workspace, Text, Tool, Skill, MCP, and
    nested Workflow facts with truthful owner provenance.
  - Use immutable references where the same prepared target appears in more
    than one lowered call.
- Completion evidence: Pending implementation.

##### `WF-PREPARE-003.c` Transfer expression-backed typed invocation collections

- Status: Not implemented.
- Source evidence:
  - `S/workflowruntime/compiler_invocation.go` accepts static message and
    instruction arrays with expression-backed fields.
  - A whole expression-backed message or instruction collection has no typed
    lowering path.
- Completion result:
  - Lower expression-backed typed message and instruction collection intent.
  - Evaluate and capture the collection at the logical call boundary.
  - Preserve roles, text, attachments, ordering, and defaults through invoke,
    start, retry, handoff, and recovery.
  - Do not turn arbitrary Workflow JSON into fallback Model or Agent chat text.
- Completion evidence: Pending implementation.

##### `WF-PREPARE-003.d` Keep prepared target transport authoritative at bind time

- Status: Partial.
- Source evidence:
  - `S/workflowruntime/target_preparation.go` prepares target transport facts.
  - `S/workflowruntime/binder.go:requireExternalPortCapabilities` checks
    context, input ports, and output streams, but does not check declared
    final-output-port support.
- Completion result:
  - Use one prepared transport capability contract at admission and bind time.
  - Enforce explicit Service and Activity context, input-port, final-output,
    and stream capability without duplicating source admission.
  - Keep Model, Agent, Tool, MCP, Skill, Service, Activity, and nested
    Workflow transport in their owning domains.
- Completion evidence: Pending implementation.

#### `WF-RETRY-003` Preserve one captured logical call through retry and started execution

- Status: Partial.
- Source evidence:
  - `captureWorkflowRetryCall` freezes target, input, context, typed messages,
    instructions, and keyed-effect values.
  - Runtime retry matching retains declared schema matching.
- Missing functional behavior:
  - Port input values and cursors are staged later by
    `R/ports_runtime.go:portInputsLocked`; a later Attempt can consume later
    channel messages.
  - Prepared target facts can be lost during retry and started-call capture.
  - Recovery-safe replay and retry boundaries do not consistently receive the
    same captured operation after uncertainty.
  - Model and Agent error adaptation still reduces several available domain
    error facts to code and message text.
- Completion result:
  - Capture target, resolved target-set membership, input, context, typed
    messages, instructions, port input values and cursors, effect key, retry
    policy, deadline, declaration path, and logical time before child
    materialization or dispatch.
  - Reuse that one snapshot for retry, start, recovery, reconciliation, and
    settlement.
  - Preserve typed domain error category, cause, receipt, and effect evidence.
- Completion evidence: Pending implementation.

##### `WF-RETRY-003.a` Capture the original evaluation boundary

- Status: Partial.
- Completion result:
  - Freeze actual port input values with the logical call.
  - Do not append later channel records on a retry Attempt.
  - Preserve prepared invocation target facts and target-set membership in the
    captured child snapshot.
- Completion evidence: Pending implementation.

##### `WF-RETRY-003.b` Apply complete retry selection and effect replay semantics

- Status: Partial.
- Completion result:
  - Retain existing retry schema matching.
  - Permit replay only for the actual target effect guarantee and captured
    operation facts.
  - Permit a replay-safe uncertain operation to reach its declared retry path
    without abandoning its owner through an unrelated Execution review.
- Completion evidence: Pending implementation.

##### `WF-RETRY-003.c` Preserve native Model and Agent failure classification

- Status: Partial.
- Completion result:
  - Preserve native category, typed details, causes, receipt, and effect state
    through Model and Agent child settlement.
  - Make those facts available to retry, catch, outcome, and projection logic.
- Completion evidence: Pending implementation.

#### `WF-CHANNEL-003` Preserve invocation-local coordination and declaration-visible meaning

- Status: Partial.
- Source evidence:
  - `R/ports_runtime.go:namespaceWorkflowProgramChannels` persists declared to
    private channel aliases.
  - `R/workflow_messages.go:captureWorkflowMessages` restores declaration
    names for expression scope.
  - `R/subscription_runtime.go` retains an originating subscription owner.
- Missing functional behavior:
  - `R/workflow_structured.go` and `R/workflow_saga.go` call `drive` with
    `Execution{}` on some structured completions.
  - That empty Execution replaces the actual authorized channel snapshot before
    a later step or block output evaluates.
  - `R/coordination.go:wakeEventWaitsLocked` checks its earlier by-value signal
    after stored subscription consumption changes.
  - Two inherited child waits can consume one event for the same originating
    subscription in one wake pass.
- Completion result:
  - Preserve declaration-visible aliases separately from invocation-private
    storage names.
  - Evaluate every step and output boundary using the actual invocation
    Execution and its authorized channel view.
  - Re-read stored signal consumption after each wait settlement.
  - Keep acceptance, retention, and consumption authoritative at the
    originating subscription owner while inherited children retain references.
- Completion evidence: Pending implementation.

##### `WF-CHANNEL-003.b` Capture each evaluation boundary

- Status: Partial.
- Completion result:
  - Keep one captured authorized channel snapshot for each actual logical
    boundary.
  - Never replace it with an empty Execution view during structured control
    completion.
  - Reuse the original captured view for a captured logical call.
- Completion evidence: Pending implementation.

##### `WF-CHANNEL-003.c` Preserve inherited subscription identity

- Status: Partial.
- Completion result:
  - Apply consumed-event state to the current durable signal before evaluating
    the next wait.
  - Permit one consuming wait per originating subscription owner.
  - Preserve this ownership through nested scopes, restore, and event replay.
- Completion evidence: Pending implementation.

#### `WF-EFFECT-003` Preserve declaration effects and actual business-effect facts

- Status: Partial.
- Source evidence:
  - `S/workflowruntime/compiler.go` explicitly maps source
    `non-idempotent` to Runtime `nonIdempotent`.
  - `S/mcpruntime/v3.go` now treats confirmed MCP success as confirmed
    settlement.
  - `WorkflowState` retains causal effect frontiers and compensation records.
- Missing functional behavior:
  - A keyed target capability is lowered to an idempotent target effect.
    `workflowCallEffect` can therefore treat key-dependent behavior as
    unconditional idempotence.
  - A declared keyed call does not require a target-specific keying capability.
  - Go Tool and MCP Tool execution have no complete target-native
    idempotency-key delivery contract.
  - `workflowChildCallBoundary` does not add the causal dependencies applied
    by ordinary effect-bearing Workflow work.
  - Nonlocal structured exits can bypass state and causal-frontier merge.
  - Compensation input can be built before its compensation ProgramFrame
    identity is durable.
- Completion result:
  - Distinguish read, unconditional idempotence, key-dependent deduplication,
    and non-idempotent behavior.
  - Treat keyed work as replay-safe only when a stable captured key and a
    target-owned keying mechanism exist.
  - Deliver a key only through a supported native target mechanism. Do not add
    generic Workflow metadata to Tool, MCP, Skill, Model, or Agent requests.
  - Preserve causal dependencies through child calls, structured exits,
    reconciliation, and compensation.
  - Persist the complete original operation and compensation identities before
    compensation code can consume `completed` facts.
- Completion evidence: Pending implementation.

##### `WF-EFFECT-003.c` Preserve compensation causality and operation facts

- Status: Partial.
- Completion result:
  - Apply the same causal frontier to ChildCall boundaries as to ordinary
    effect-bearing WorkItems.
  - Merge child causal facts before nonlocal control exits complete.
  - Persist original WorkItem, Attempt, frame, target, receipt, effect,
    idempotency key, compensation WorkItem, and compensation frame facts in
    the order visible to compensation code.
- Completion evidence: Pending implementation.

##### `WF-EFFECT-003.d` Preserve target-owned effect and keying guarantees

- Status: Partial.
- Completion result:
  - Obtain actual effect capability facts from the owning target domain.
  - Prevent a keyed-only target from becoming replay-safe without a key.
  - Preserve one key through attempts, waits, retry, reconciliation, and
    recovery.
  - Use identical settled facts for continuation, outcome, history,
    reconciliation, compensation, and completion waits.
- Completion evidence: Pending implementation.

#### `WF-CONTROL-003` Complete structured-control value and lifetime behavior

- Status: Partial.
- Existing behavior:
  - Repeat index handling, empty Foreach settlement, pointer preservation for
    collection maxima, and Machine immediate-transition reset behavior exist.
- Missing functional behavior:
  - Early Workflow return, block return, break, and continue paths do not all
    merge applicable child state and causal effect facts before control exits.
  - Parallel and Foreach first-completed selection scans declaration or item
    order instead of retaining durable actual completion order.
  - Parallel first-success with `remaining: wait` can retain more than one
    winner instead of one winning branch and all branch outcomes.
- Completion result:
  - Preserve child state, causal effect facts, claims, and semantic outcomes
    before any control exit crosses its structured owner.
  - Persist an actual completion-order fact where first-completed behavior
    requires it.
  - Preserve one selected winner while retaining all settled child outcomes,
    including late failure, cancellation, and uncertainty.
- Completion evidence: Pending implementation.

##### `WF-CONTROL-003.d` Preserve state and causality through structured exits

- Status: Partial.
- Completion result:
  - Complete state merge and causal merge before break, continue, block return,
    Workflow return, and Saga body exit.
  - Keep the actual invocation Execution available while selecting follow-on
    work and output.
  - Release claims through their actual owner lifecycle.
- Completion evidence: Pending implementation.

#### `WF-OUTCOME-003` Preserve complete Workflow and conversation terminal meaning

- Status: Partial.
- Source evidence:
  - `R/workflow_outcomes.go` records ordinary, inline, return, break, and
    continue outcomes.
  - `R/program.go:interruptedAssistantActivities` retains some interrupted
    assistant output.
- Missing functional behavior:
  - `R/runtime_children.go:completionExecutionResult` replaces a retained
    Execution error with a generic error and loses category, effect, details,
    and causes.
  - Recovery can mark original Workflow work uncertain without recording the
    equivalent Workflow step outcome.
  - Cancelled outcome construction can retain ordinary output and then fail
    Workflow outcome invariants.
  - A late cancelled LLM result can reach an already-terminal frame and omit
    its interrupted assistant Activity.
  - Replay-safe re-dispatch can merge an old interrupted stream with a new
    Attempt stream.
- Completion result:
  - Preserve actual declaration path, target, receipt, effect, aggregate
    facts, typed errors, and actual terminal meaning for every step path.
  - Preserve typed errors through WorkItem, ProgramFrame, Execution, child
    completion wait, projection, catch, and retry boundaries.
  - Persist partial assistant output once with failed, cancelled, or uncertain
    meaning. Never fabricate a successful operation.
- Completion evidence: Pending implementation.

##### `WF-OUTCOME-003.a` Record complete settled step facts

- Status: Partial.
- Completion result:
  - Record actual ordinary, inline, child, reconciliation, cancellation,
    return, break, continue, and recovery step outcomes.
  - Clear output where the terminal outcome does not permit output.
  - Preserve outcome facts through cross-Execution completion.
- Completion evidence: Pending implementation.

##### `WF-OUTCOME-003.b` Persist interrupted assistant output truthfully

- Status: Partial.
- Completion result:
  - Retain one Attempt-scoped interrupted stream.
  - Append one interrupted assistant Activity when the owning LLM operation
    actually settles.
  - Keep replay output separate from the earlier interrupted output.
- Completion evidence: Pending implementation.

#### `V3-REPLAY-003` Reconstruct private resources from the exact retained operation

- Status: Partial.
- Source evidence:
  - `ModelDispatchPlan.RuntimeConfiguration` stores a non-secret Model and
    Provider configuration snapshot.
  - `S/modelruntime.PreparedRuntimeConfiguration` reconstructs private
    credentials from a retained binding.
  - `S/sessionv3runtime/llm_replay.go` uses the retained dispatch plan instead
    of resolving current Model overlays.
- Missing functional behavior:
  - `recoveredDispatchActivities` does not retain original Activity,
    WorkItem, frame, and Execution associations.
  - Replay helpers can select current Session protocol, Tool, and context facts
    instead of only captured operation facts.
  - The stored runtime configuration is built before final instructions,
    Skill prompt, explicit prompt, and other effective request contributions
    are complete.
  - Recovered replay-safe LLM work retains an earlier stream without an
    Attempt boundary.
- Completion result:
  - Retain exact ordered input meaning, assistant protocol output, Tool calls,
    confirmed Tool results, attachments, App context, injected context, and
    effective non-secret Model configuration.
  - Reconstruct only private credentials, clients, and domain sessions.
  - Do not use current defaults, current history, current contribution
    selection, or later Session outcomes as replacement facts.
  - Preserve a separate stream boundary for each Attempt.
- Completion evidence: Pending implementation.

##### `V3-REPLAY-003.a` Retain semantic LLM bindings

- Status: Partial.
- Completion result:
  - Persist one canonical final dispatch plan after Model-owned effective
    configuration composition.
  - Preserve exact causal references required to reconstruct provider inputs.
  - Keep generated Tool WorkItems on canonical origin-plan references.
- Completion evidence: Pending implementation.

##### `V3-REPLAY-003.b` Reproduce captured message order and configuration

- Status: Partial.
- Completion result:
  - Reconstruct provider inputs from captured operation facts only.
  - Preserve assistant protocol output and Tool-result ordering.
  - Reuse final captured prompt and configuration instead of recomposing
    mutable contribution content during replay.
- Completion evidence: Pending implementation.

##### `V3-REPLAY-003.c` Separate interrupted and replayed streams

- Status: Not implemented.
- Completion result:
  - Associate streamed Model output with the Attempt that produced it.
  - Prevent a replayed Attempt from appending to earlier interrupted content.
  - Keep `LLMOutput`, `WorkItemOutput.Value`, port records, activities, and
    projections mutually consistent.
- Completion evidence: Pending implementation.

#### `V3-CONFIG-003` Complete inherited and explicit invocation configuration

- Status: Partial.
- Source evidence:
  - `SessionCollectionPolicies` retains collection replacement, explicit empty
    replacement, removals, and override intent.
  - `applySpecificCollections` applies Session and invocation collections after
    inherited collections.
- Missing functional behavior:
  - Agent Text, Agent instruction Skills, and Workspace instruction
    contributions can become effective before specific collection policy can
    remove or replace them.
  - Prepared Workflow Workspace consumption can bypass effective Workspace
    prompt, context, and override behavior.
  - Effective collections do not retain complete contribution owner and order
    provenance after flattening.
- Completion result:
  - Apply specific collection policy before inherited contributions become
    effective request content.
  - Ensure explicit removal and explicit empty replacement remove all effects
    of the inherited contribution.
  - Preserve owner, selection, ordering, override, and effective-result facts
    without duplicating complete effective plans in multiple owner records.
- Completion evidence: Pending implementation.

##### `V3-CONFIG-003.b` Apply effective contribution precedence

- Status: Partial.
- Completion result:
  - Preserve explicit Session and invocation policy over Agent and Workspace
    inherited contributions, including Text and instruction Skills.
  - Preserve effective Workspace prompt and context contributions for normal
    and prepared Workflow invocations.
- Completion evidence: Pending implementation.

##### `V3-CONFIG-003.c` Retain truthful ownership and ordering

- Status: Partial.
- Completion result:
  - Retain source-family owner provenance and actual contribution ordering.
  - Keep one canonical effective LLM plan.
  - Do not reconstruct ownership by copying complete effective collections into
    Agent, Workspace, or call-site records.
- Completion evidence: Pending implementation.

#### `DOMAIN-SKILL-003` Reconstruct and release Skill resources for the exact owner

- Status: Partial.
- Source evidence:
  - `S/skillruntime/v3_hydration.go` implements
    `EffectiveActiveSelectionsForFrame`.
  - `R/runtime.go:advanceFrameLocked` invokes optional frame cleanup.
- Missing functional behavior:
  - `V3Executor.Choices` still uses unscoped active-Skill reconstruction.
  - Frame cleanup is launched before the enclosing terminal mutation commits.
  - Fork, Join, cleanup-expiry, cancellation, and interpreter-failure paths do
    not all issue the same frame lifecycle cleanup notification.
  - A cancelled frame can close its Skill session while owned external work
    still needs late settlement.
- Completion result:
  - Use one exact owner key for all active-Skill selection and private Skill
    session lookup.
  - Close a frame-owned Skill session only after committed terminal ownership
    permits no eligible in-flight use.
  - Route every terminal frame path through one lifecycle callback.
- Completion evidence: Pending implementation.

##### `DOMAIN-SKILL-003.a` Select applicable active Skill state

- Status: Partial.
- Completion result:
  - Remove or route unscoped active-Skill lookup through the frame-scoped
    owner path.
  - Keep parallel Agent invocations independent.
  - Select only the latest applicable terminal lifecycle result for the same
    owner and enabled pins.
- Completion evidence: Pending implementation.

##### `DOMAIN-SKILL-003.b` Release Skill sessions on natural completion

- Status: Partial.
- Completion result:
  - Commit terminal frame facts before asynchronous private Skill cleanup.
  - Cover normal completion, cancellation, deadline, panic, Fork, Join,
    cleanup expiry, and parent terminal propagation.
- Completion evidence: Pending implementation.

#### `V3-CONTINUE-003` Complete active-operation continuation and manual Tool ownership

- Status: Partial.
- Source evidence:
  - `consumeContinuationLocked` can route staged input to an active Agent
    ProgramFrame.
  - `requestAgentContinuationLocked` selects an active conversation or
    handoff Agent frame.
- Missing functional behavior:
  - `R/runtime.go:ContinueExecution` broadcasts Continue to all live frames
    when it does not consume staged input.
  - Staged continuation is restricted to a normal conversation Execution even
    when a Workflow handoff owns the Session conversation.
  - An Agent can queue an Activity during active LLM work and then complete
    without consuming or transferring that Activity.
  - Manual Tool operations and automation holds do not consistently enforce
    active conversation ownership and current Tool-batch scope.
- Completion result:
  - Route public continuation to one semantic conversation owner.
  - Permit staged continuation for the active handoff owner.
  - Consume or durably transfer every accepted queued Activity before an Agent
    frame completes.
  - Scope manual Tool actions and automation holds to the active operation and
    applicable Tool batch.
- Completion evidence: Pending implementation.

##### `V3-CONTINUE-003.a` Resume the active conversation operation

- Status: Partial.
- Completion result:
  - Use the active Session conversation ownership record for public and staged
    continuation routing.
  - Retain the exact Activity defaults, budget, attachments, and approved App
    context with the operation that consumes it.
- Completion evidence: Pending implementation.

##### `V3-CONTINUE-003.b` Scope manual controls to the current operation

- Status: Partial.
- Completion result:
  - Retain consumed-result protection.
  - Also reject a manual Tool mutation outside the active owner and current
    Tool batch.
  - Select automation-hold scope from the active semantic owner rather than
    the latest unrelated LLM WorkItem in the Execution.
- Completion evidence: Pending implementation.

#### `ARC-DOMAIN-003` Complete owning-domain setup boundaries

- Status: Partial.
- Source evidence:
  - Core Runtime remains independent from Artifact Store and setup imports.
  - Model setup owns durable non-secret provider configuration and private
    credential reconstruction.
  - Tool, MCP, Skill, and registered-binding domains expose typed execution
    boundaries.
- Missing architectural behavior:
  - `S/workflowruntime/model_agent_adapter.go` interprets Agent contribution
    modes, selects Models, and traverses Workspace capability graphs.
  - `S/sessionv3runtime/llm_plan.go` independently implements Agent and
    Workspace contribution interpretation.
  - `S/sessionv3runtime/attachments.go` owns V3 attachment admission behavior
    instead of Attachment setup.
  - `S/sessionv3runtime/inputs.go` and `llm.go` retain substantial provider
    input/output and Model-default adaptation.
- Completion result:
  - Agent setup owns Agent contribution interpretation and Model selection.
  - Workspace setup owns Workspace selection, overrides, prompt, and context
    interpretation.
  - Attachment setup owns V3 admission and immutable semantic snapshot
    materialization.
  - Model setup owns final provider input, output, and effective Model request
    adaptation.
  - Session and Workflow setup compose neutral domain facts and lower
    orchestration intent only.
  - Core Runtime remains Artifact-independent.
  - Artifact Store remains passive and gains no execution API.
- Completion evidence: Pending implementation.

##### `ARC-DOMAIN-003.a` Move contribution preparation to owning domains

- Status: Partial.
- Completion result:
  - Replace duplicated Agent and Workspace graph traversal with domain-owned
    neutral preparation results.
  - Let Workflow setup consume prepared contribution facts from those domains.
  - Keep Workflow source interpretation in Workflow setup and native domain
    semantics in their owning setup packages.
- Completion evidence: Pending implementation.

##### `ARC-DOMAIN-003.b` Move attachment and Model conversion to owning domains

- Status: Partial.
- Completion result:
  - Move V3 attachment snapshot admission into Attachment setup.
  - Move final provider input and output conversion, including effective
    Model-default adaptation, into Model setup.
  - Keep Session history selection and Runtime lifecycle orchestration outside
    native provider conversion.
- Completion evidence: Pending implementation.

#### `DOMAIN-ATTACH-003` Preserve semantic attachment identity in native Model input

- Status: Partial.
- Source evidence:
  - `S/sessionv3runtime/attachments.go` creates stable directory-relative
    labels for text and file snapshots.
  - `S/sessionv3runtime/inputs.go:attachmentTextForProvider` retains text
    labels.
  - File conversion assigns `ContentItemFile.FileName`.
- Missing functional behavior:
  - Image conversion does not assign the admitted label to
    `ContentItemImage.ImageName`.
- Completion result:
  - Preserve admitted labels for text, image, and file provider content.
  - Preserve stable directory-relative names for same-named files.
  - Continue using admitted immutable snapshots without reopening native paths.
  - Move admission ownership as required by `ARC-DOMAIN-003.b`.
- Completion evidence: Pending implementation.

##### `DOMAIN-ATTACH-003.a` Carry labels and directory-relative names

- Status: Partial.
- Completion result:
  - Populate the native image name from the admitted attachment label.
  - Retain the existing text and file label behavior.
- Completion evidence: Pending implementation.

#### `ARC-DUP-003` Remove replicated effective state and duplicated execution algorithms

- Status: Partial.
- Source evidence:
  - `ModelDispatchPlan` is the canonical durable LLM operation plan.
  - Generated Tool WorkItems retain `LLMToolOrigin.PlanReference`.
  - `persistLocked` is the intended common post-commit dispatch boundary.
- Missing duplication:
  - `persistLocked`, `changeSession`, `changeExecution`, terminal handlers,
    port helpers, retry handlers, and restore paths all request overlapping
    ready-work launch behavior.
  - Workflow value evaluation, retry and catch matching, recursive error
    cloning, operation-input extraction, and retention calculations have
    equivalent implementations in more than one Runtime path.
  - Channel and resource namespacing have parallel lowered-tree walks.
  - Workflow setup repeats lowered call-tree traversal.
  - Session and Workflow setup both interpret Agent and Workspace
    contributions.
  - Prepared target facts are copied into both target and call-site structures.
- Completion result:
  - Keep one canonical immutable effective plan and reference it from
    generated work.
  - Consolidate Runtime semantic algorithms under Runtime ownership.
  - Consolidate domain contribution interpretation under its owning domain.
  - Consolidate lowered-tree traversal where the traversal is Runtime
    orchestration behavior.
  - Do not create a generic native-domain envelope.
  - Preserve V2 coexistence. This task does not remove V2 code.
- Completion evidence: Pending implementation.

##### `ARC-DUP-003.c` Consolidate Runtime algorithms and lowered-tree traversal

- Status: Partial.
- Completion result:
  - Retain one committed ready-work launch lifecycle.
  - Consolidate equivalent evaluation, matching, error cloning, operation
    extraction, channel and resource namespacing, call lookup, and retention
    logic.
  - Keep domain-native request grammar in Model, Tool, MCP, Skill, Service,
    Activity, Attachment, Agent, and Workspace domains.
- Completion evidence: Pending implementation.
