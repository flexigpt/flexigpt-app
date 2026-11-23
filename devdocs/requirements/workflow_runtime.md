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

The remaining work is limited to items marked `Partial` or `Not implemented`
in the task ledger. It concerns one post-checkpoint dispatch authority,
owner-scoped accounting for nested Workflow port transfer, complete
domain-owned prepared Model and Agent contributions, removal of duplicate V3
setup algorithms, and the remaining backend semantic-export surface.

This audit does not create Artifact planning work.
`workflow_artifact.md` remains the authority for source admission, retrieval,
closure, and planning. Runtime and setup must not move execution preparation,
binding, dispatch, scheduling, recovery, or consumer-private state into
Artifact Store.

The task statuses below are corrected from source inspection. This remains
work in progress. No version identifier may change while these tasks are
implemented.

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
  when it creates that Activity.
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

This is the single Runtime and consumer-execution status ledger.

The separate Artifact source-planning record in `workflow_artifact.md` is not
an execution-status ledger and does not create Runtime work.

Each functional group contains parent tasks and their subtasks. A parent is
complete only when every listed subtask is complete. Source evidence identifies
the implemented behavior.

`R` means `internal/agentruntime-go/sessionv3/`.

`S` means `internal/agentruntimesetup/`.

### Runtime settlement, cleanup, scheduling, commands, recovery, and persistence

#### `RT-SETTLE-003` Authoritative mutation and terminal propagation

- Status: Implemented.
- Completion evidence:
  - `internal/agentruntime-go/sessionv3/runtime.go:persistLocked` is the
    committed Session-wide ready-work launch boundary. It accepts committed
    derived-index errors and clears stale checkpoint barriers before readiness.
  - `runtime_lifecycle.go:commitTerminalMutationLocked`,
    `settlePendingFrameCancellationsLocked`, and
    `executionTerminalReadyLocked` complete attempt, effect, claim, wait,
    owner, Execution, checkpoint, and post-commit dispatch lifecycle work.
  - `runtime.go:dispatchOne`, `finishWorkLocked`, `finishPanickedWork`, and
    `runtime_durability.go:settleUndispatchedWorkLocked` route binding, port,
    panic, deadline, retry, resource, cancellation, reconciliation, and child
    terminal paths through the common lifecycle.

##### `RT-SETTLE-003.a` Reacquire authoritative Session-owned objects

- Status: Implemented.
- Completion evidence:
  - `coordination.go:wakeEventWaitsLocked`, `wakeChannelWaitsLocked`,
    `wakeCompletionWorkWaitsLocked`, and `wakeCompletionFrameWaitsLocked`
    re-acquire Session, Execution, Wait, signal, and channel records by
    durable identity after each nested settlement.
  - `runtime_children.go:wakeCompletionExecutionWaitsLocked`,
    `runtime.go:advanceFrameLocked`, and
    `runtime_durability.go:settleUndispatchedWorkLocked` re-acquire durable
    owners after child, frame, and wait mutation.

##### `RT-SETTLE-003.b` Use one terminal lifecycle and committed ready-work launch

- Status: Partial.
- Implemented behavior:
  - `persistLocked` provides a committed Session-wide ready-work launch path.
  - Dispatch guards prevent a second queued dispatch from normally reaching a
    second native invocation.
- Remaining functional gap:
  - `runtime.go:blockWorkForBudgetLocked`,
    `runtime_durability.go:settleTimerLocked`,
    `runtime_durability.go:settleUnstartedAttemptLocked`,
    `runtime_triggers.go:waitForTrigger`, `normal_app.go:ExecuteAppTool`, and
    `normal_session.go:EditAndResend` still launch ready work after
    `persistLocked`.
  - The duplicated launch waves violate the single committed launch authority.
    Only the completed post-restore readiness wave may remain separate.

#### `RT-CLEANUP-003` Execute cleanup through every owning lifecycle

- Status: Implemented.
- Completion evidence:
  - `budget.go:beginCleanupLocked` and `rebaseCleanupDeadlinesLocked` retain
    bounded cleanup deadlines independently from ordinary deadlines.
  - `runtime.go:advanceFrameLocked` propagates cleanup classification through
    child frames and synthetic WorkItems. `runtime_durability.go` and
    `runtime_children.go` use the effective cleanup deadline for bind, retry,
    waits, resources, streams, emits, and child frames.
  - `runtime_child_cancel.go:cancelFrameTreeLocked` records frame cancellation
    intent without terminalizing an owner that still has dispatched work.
    `runtime_lifecycle.go:settlePendingFrameCancellationsLocked` completes the
    frame only after late evidence, release, cleanup, and completion delivery.

##### `RT-CLEANUP-003.a` Carry cleanup classification and effective deadline

- Status: Implemented.
- Completion evidence:
  - `effectiveWorkDeadlineLocked`, `effectiveAttemptDeadlineLocked`,
    `rebaseCleanupDeadlinesLocked`, `workflowFrameIntentRunsCleanup`, and
    `childFrameDeadlineInputs` preserve cleanup classification and the bounded
    deadline through built-in Agent, Model, Fork, Join, merge, wait, retry,
    stream, emit, settlement, and recovery paths.

##### `RT-CLEANUP-003.b` Retain Saga ownership until late outcomes settle

- Status: Implemented.
- Completion evidence:
  - `workflow_saga.go:beginSagaCompensation` keeps a Saga pending when its
    first compensation order is empty but associated normal work or child
    frames can still settle.
  - `workflowSagaHasOutstandingNormalWork` includes active Saga-scoped
    Workflow frames as well as WorkItems.
  - `finishWorkLocked` calls `resumeSagaCompensationFramesLocked` after both
    ordinary and cancellation settlement paths.

##### `RT-CLEANUP-003.c` Preserve cancellation ownership until late settlement

- Status: Implemented.
- Completion evidence:
  - `ProgramFrame.CancellationRequested`,
    `settlePendingFrameCancellationsLocked`, and
    `executionTerminalReadyLocked` keep frame and Execution ownership alive
    until dispatched work, claims, reconciliation, compensation, cleanup, and
    completion delivery settle.

#### `RT-BUDGET-003` Complete scoped accounting and resource-ready scheduling

- Status: Implemented.
- Completion evidence:
  - `budget_scope.go:ExecutionBudgetScope`,
    `reserveScopedBudgetLocked`, and
    `runtime_fork_join.go:deferFrameForkLocked` retain one Execution aggregate
    and one controlling structured-scope aggregate.
  - `workflow_program.go:startWorkflowChildCall` and
    `workflow_call_lifecycle.go:startCall` set `WorkflowNesting` only for
    actual nested Workflow boundaries.
  - `reserveMessageBudgetForOwnerLocked`,
    `reserveEventBudgetForOwnerLocked`, `normal_session.go`,
    `runtime.go:SignalExecution`, `coordination.go`, and `ports_runtime.go`
    account Activities, signals, channel records, forwarded ports, staged
    inputs, and output ports by semantic owner. Cost metering remains absent.

##### `RT-BUDGET-003.a` Wake waiters when a counted slot becomes available

- Status: Implemented.
- Completion evidence:
  - `persistLocked`, `wakeBudgetConcurrencyWaitersLocked`, and
    `wakeResourceWaitersLocked` commit durable wakeups before the common
    post-commit ready-work launch.

##### `RT-BUDGET-003.b` Apply structured limits and semantic Workflow depth

- Status: Implemented.
- Completion evidence:
  - `ExecutionBudgetScope`, `ProgramFrame.BudgetScopeID`,
    `WorkItem.BudgetScopeID`, and `ProgramFrameIntent.BudgetScopeID` retain
    controlling structured aggregate accounting while child frames retain
    narrowed limits.

##### `RT-BUDGET-003.c` Reserve outer child-call work before child dispatch

- Status: Implemented.
- Completion evidence:
  - The durable Fork WorkItem for a child call depends on the matching outer
    `ChildCall` WorkItem.
  - A Fork dependency accepts a ChildCall only after its binding has persisted
    the outer dispatched Attempt and durable completion wait.
  - Child ProgramFrames cannot materialize or dispatch native work before
    outer call budgeting, ownership, effect state, and Attempt facts exist.

##### `RT-BUDGET-003.d` Account retained Runtime records through their semantic owner

- Status: Partial.
- Implemented behavior:
  - `Activity`, `ExecutionSignal`, and `ChannelMessage` retain semantic owner
    identities. Owner-aware budget reservation is used for retained Runtime
    records and provider usage is consumed even when a Model result has an
    execution error.
- Remaining functional gap:
  - `ports_runtime.go:appendPublishedWorkflowPortInputLocked` receives a
    child frame identity but reserves message budget without that frame or its
    structured budget scope.
  - `appendForwardedChannelMessageLocked` uses source WorkItem and frame
    identities while it charges the target Execution. Those identities do not
    exist in the parent Execution for an independent child Workflow.
  - `ChannelForward` has no receiving-side semantic owner. Nested input and
    forwarded port records can therefore bypass the intended owner-scoped
    message and byte budgets.

#### `RT-TRIGGER-003` Resume trigger lifecycle changes and expose one event meaning

- Status: Implemented.
- Completion evidence:
  - `R/runtime.go:SetTriggerRegistrationState` calls
    `recoverSessionTriggersLocked` and `drainTriggerQueuesLocked` while the
    registration-state command owns the Session mutation.
  - `R/workflow_triggers.go:TriggerEventEnvelope` serializes `type`.
  - `R/runtime_triggers.go` maps accepted trigger command `Name` to `Type` and
    maps the same `Type` to `ExecutionSignal.Name`.
  - Event waits and Machine waits use `ExecutionSignal.Name` as their durable
    event type.

##### `RT-TRIGGER-003.a` Drain eligible occurrences after reactivation

- Status: Implemented.
- Completion evidence:
  - `R/runtime.go:SetTriggerRegistrationState` recovers missed schedule
    occurrences and drains trigger queues before the command checkpoint.
  - `R/runtime_durability.go:resumeRestoredWork` and
    `R/runtime.go:changeSession` use the same durable ready-work launch path.

##### `RT-TRIGGER-003.b` Use one declared event projection

- Status: Implemented.
- Completion evidence:
  - `R/workflow_triggers.go` serializes trigger event `type`, source, identity,
    data, correlation, and time.
  - `R/coordination.go:workflowEventWaitOutput` retains the same event type
    through durable ordinary event waits and Machine transitions.

#### `RT-COMMAND-003` Replay accepted commands independently of regenerated preparation

- Status: Implemented.
- Completion evidence:
  - `runtime_triggers.go:acceptTriggerEvent` fingerprints a caller-owned
    identity before Runtime assigns `OccurredAt`.
  - `runtime.go:FindCommandReceipt` and wrapper command identities return
    prior receipts before attachment or Workflow preparation.

##### `RT-COMMAND-003.a` Preserve public command identity before preparation

- Status: Implemented.
- Completion evidence:
  - Workflow wrapper identities contain the original command metadata,
    Session, optional Execution, Workflow selection, and canonical input.
  - Runtime command fingerprinting uses those identities when supplied.
  - Attachment-backed commands query durable command records before external
    attachment materialization.

##### `RT-COMMAND-003.b` Keep generated occurrence time outside caller identity

- Status: Implemented.
- Completion evidence:
  - `R/runtime_triggers.go:acceptTriggerEvent` copies caller identity before
    Runtime assigns or normalizes `OccurredAt`.
  - `changeSession` fingerprints the unchanged identity while the mutation
    closure persists the normalized trigger occurrence time.

#### `RT-RECOVERY-003` Resume recovered operations through their actual owners

- Status: Implemented.
- Completion evidence:
  - `runtime.go:recoverLocked` recreates eligible pre-invocation waits from
    retained semantic requests and creates owner-specific recovery reviews.
  - `resolveRecoveryWaitLocked` and `settleRecoveredReviewWorkLocked` route
    review decisions through original WorkItem owners, catch, finally,
    reconciliation, compensation, and completion lifecycle.
  - `Runtime.Restore` and `resumeRestoredWork` finish restore with one
    committed Session-wide readiness pass.

##### `RT-RECOVERY-003.a` Reconstruct pre-invocation domain waits

- Status: Implemented.
- Completion evidence:
  - `R/runtime.go:recoverLocked` requeues a waiting Runtime-bound WorkItem
    from its retained semantic request.
  - The prior private wait record is cancelled and normal binding recreates
    current domain-private authorization state after restore.

##### `RT-RECOVERY-003.b` Resume the complete committed Session dependency wave

- Status: Implemented.
- Completion evidence:
  - Restore reacquires Executions after interpreter advancement and persists
    recovered child settlement, wait resolution, reconciliation, and trigger
    queue mutations before the Session-wide ready-work launch.

##### `RT-RECOVERY-003.c` Settle recovery review through operation lifecycle

- Status: Implemented.
- Completion evidence:
  - `R/runtime.go:resolveRecoveryWaitLocked` settles retained uncertain
    WorkItems through owner advancement.
  - Recovery cancellation uses Execution cancellation and the cleanup
    lifecycle rather than direct terminal replacement.

#### `RT-PERSIST-003` Complete deletion commitment and idle Session deletion

- Status: Implemented.
- Completion evidence:
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
- Completion evidence:
  - `sessionstorev3.Store.HandleSessionTombstone` returns
    `DerivedIndexError` only after durable tombstone publication and schedules
    repair.
  - `R/runtime.go:DeleteSession` accepts errors whose
    `CheckpointCommitStatus` reports a committed tombstone.

##### `RT-PERSIST-003.b` Delete idle long-lived normal conversations

- Status: Implemented.
- Completion evidence:
  - `R/runtime.go:idleNormalConversationExecution` distinguishes an idle
    long-lived conversation root from active external, cleanup, child, wait,
    or uncertain work.
  - `R/runtime.go:DeleteSession` cancels due-time callbacks, discards private
    bindings, and requests owning-domain execution-binding cleanup after the
    committed Session deletion.

### Workflow preparation, captured calls, ports, and channels

#### `WF-PREPARE-003` Preserve complete prepared target meaning

- Status: Partial.
- Completion evidence:
  - `WorkflowPreparedInvocation` retains immutable Model, Agent, Workspace,
    Text, Tool, Skill, MCP, Workspace prompt/context, contribution-order, and
    owner-provenance facts.
  - `program.go:startAgentLLM`, retry capture, started calls, handoff,
    nested preparation, and recovery retain one immutable prepared invocation.
  - `compiler_invocation.go` lowers typed expression-backed message and
    instruction collections. `workflow_invocation.go:captureWorkflowLLMInvocation`
    evaluates and freezes those collections at the logical call boundary.
- Remaining functional gap:
  - The data type contains Workspace prompt, context, and contribution-order
    fields, but the current Workflow Agent preparation path does not fully
    populate or consume them. The complete requirement is tracked by
    `WF-PREPARE-004`.

##### `WF-PREPARE-003.a` Retain immutable pins and prepared contribution facts

- Status: Partial.
- Implemented behavior:
  - Immutable target pins and typed message and instruction collections are
    retained through invoke, start, retry, handoff, nested Workflow
    preparation, Agent rounds, and recovery.
- Remaining functional gap:
  - `workflowruntime/model_agent_adapter.go` marks a Workspace prompt as
    prepared after it reads Workspace capabilities, but it does not capture
    `ResolveWorkspaceRuntimePlan` prompt instructions or user context.
  - `sessionv3runtime/workflow_invocation.go` accepts that marker and applies
    empty Workspace prompt and context fields. It also does not consume
    `ContributionOrder`.

##### `WF-PREPARE-003.b` Resolve an owner Model only where required

- Status: Implemented.
- Completion evidence:
  - `S/workflowruntime/admission.go:admitPlanned` does not eagerly resolve an
    owner Model.
  - `bindTarget` calls `workflowOwnerModel` only after Tool setup identifies a
    provider-managed SDK Tool target.
  - Pure deterministic, registered-binding, ordinary Tool, MCP, nested
    Workflow, Model, and Agent Workflow programs do not depend on unrelated
    Session conversation Model defaults.

##### `WF-PREPARE-003.c` Transfer expression-backed typed invocation collections

- Status: Implemented.
- Completion evidence:
  - `WorkflowLLMMessageSegment`, `WorkflowLLMInstructionSegment`,
    `compileWorkflowLLMMessageSegment`,
    `compileWorkflowLLMInstructionSegment`, and
    `captureWorkflowLLMInvocation` retain typed role, text, attachment,
    ordering, instruction, and defaults facts without generic JSON chat
    fallback.

##### `WF-PREPARE-003.d` Keep prepared target transport authoritative at bind time

- Status: Implemented.
- Completion evidence:
  - `workflowruntime/target_transport.go` and
    `workflowruntime/binder.go:requireExternalPortCapabilities` use the
    prepared target contract for Service, Activity, final-output, stream,
    context, and input-port behavior.

#### `WF-RETRY-003` Preserve one captured logical call through retry and started execution

- Status: Implemented.
- Completion evidence:
  - `workflow_ports.go:captureWorkflowCallPortInputs`,
    `WorkflowCall.PortInputs`, `WorkItemIntent.PortInputs`,
    `ProgramFrameIntent.PortInputs`, `workflow_retry.go`, and
    `retry_program.go` capture actual port values and cursors before dispatch
    or child materialization.
  - `ports_runtime.go:portInputsLocked` respects captured inputs and does not
    append later channel records on retry or recovery.

##### `WF-RETRY-003.a` Capture the original evaluation boundary

- Status: Implemented.
- Completion evidence:
  - Captured target, target-set membership, input, context, messages,
    instructions, ports, key, policy, declaration path, and logical time are
    retained in WorkItem and child ProgramFrame intent facts.

##### `WF-RETRY-003.b` Apply complete retry selection and effect replay semantics

- Status: Implemented.
- Completion evidence:
  - `compiler_coordination.go:compileWorkflowRetry` retains `match.schema`.
  - `retryMatches` validates declared retry schemas against durable typed
    errors.
  - `effect_state.go:effectReplaySafe`, `workflow_program.go:workflowCallEffect`,
    and retry boundaries allow replay only for read, unconditional idempotent,
    target-native keyed, or confirmed-not-applied work.

##### `WF-RETRY-003.c` Preserve native Model and Agent failure classification

- Status: Implemented.
- Completion evidence:
  - Model and Agent child settlement preserves `SanitizedError` category,
    details, causes, receipt, and effect state through retry, catch, outcome,
    projection, and completion boundaries.

#### `WF-CHANNEL-003` Preserve invocation-local coordination and declaration-visible meaning

- Status: Implemented.
- Completion evidence:
  - `namespaceWorkflowProgramChannels`,
    `workflowDeclaredChannelName`, `EvaluationBoundary.Messages`, and captured
    call snapshots preserve declared aliases and actual Execution visibility.
  - `wakeEventWaitsLocked` captures originating subscription identity before
    settlement and re-reads durable signal consumption before the next wait.

##### `WF-CHANNEL-003.a` Preserve declared channel aliases

- Status: Implemented.
- Completion evidence:
  - `R/ports_runtime.go:namespaceWorkflowProgramChannels` persists
    `WorkflowScopeContext.ChannelAliases`.
  - `R/workflow_messages.go:workflowDeclaredChannelName` publishes declared
    names to CEL while `ChannelLog` keeps private invocation-local names.

##### `WF-CHANNEL-003.b` Capture each evaluation boundary

- Status: Implemented.
- Completion evidence:
  - Workflow step, structured completion, root-output, retry, and started-call
    boundaries use the actual invocation Execution and a durable authorized
    channel snapshot.

##### `WF-CHANNEL-003.c` Preserve inherited subscription identity

- Status: Implemented.
- Completion evidence:
  - `WorkflowSubscriptionState.OwnerFrameID`,
    `subscriptionConsumptionForWait`, and `wakeEventWaitsLocked` preserve one
    consumption identity for each originating subscription owner.

### Workflow effects, recovery, structured control, and outcomes

#### `WF-EFFECT-003` Preserve declaration effects and actual business-effect facts

- Status: Implemented.
- Completion evidence:
  - `compileWorkflowEffect`, `workflowCallEffect`, `EffectContract.KeyedNative`,
    `KeyedEffectCapability`, and `effectReplaySafe` distinguish unconditional
    idempotence from target-native keyed deduplication.
  - `workflowChildCallBoundary`,
    `appendWorkflowCausalDependencies`, structured settlement, machine,
    catch, finally, reconciliation, and Saga paths preserve causal facts.
  - `workflow_saga.go` and `compensation.go:markCompensationIntentLocked`
    persist complete compensation identities before compensation code consumes
    completed-operation facts.

##### `WF-EFFECT-003.a` Translate `non-idempotent`

- Status: Implemented.
- Completion evidence:
  - `S/workflowruntime/compiler.go:compileWorkflowSourceEffectKind` explicitly
    maps source `non-idempotent` to Runtime `EffectKindNonIdempotent`.
  - Target-level keyed capability declarations lower to an idempotent base
    contract with separate keyed capability facts.

##### `WF-EFFECT-003.b` Settle MCP from actual domain evidence

- Status: Implemented.
- Completion evidence:
  - `S/mcpruntime/v3.go:V3Executor.Execute` returns confirmed success without
    forcing `EffectStateUncertain`.
  - Runtime derives read `notApplicable` and mutating `confirmedApplied` from
    the durable receipt.
  - MCP error responses preserve typed uncertainty for mutating WorkItems and
    remain confirmed read failures for read-only WorkItems.

##### `WF-EFFECT-003.c` Preserve compensation causality and operation facts

- Status: Implemented.
- Completion evidence:
  - Compensation WorkItems retain original WorkItem, Attempt, frame, target,
    receipt, effect, idempotency key, compensation WorkItem, and compensation
    ProgramFrame identities before child materialization.

##### `WF-EFFECT-003.d` Preserve target-owned effect and keying guarantees

- Status: Implemented.
- Completion evidence:
  - Keyed replay requires a stable captured key and
    `WorkflowNativeIdempotencyKeying`. Go Tool, MCP Tool, Skill Tool, Model,
    and Agent grammars receive no generic Workflow metadata.

#### `WF-CONTROL-003` Complete structured-control value and lifetime behavior

- Status: Implemented.
- Completion evidence:
  - `WorkflowParallelProgress.CompletionSequence`,
    `WorkflowForeachProgress.CompletionSequence`, and child completion order
    retain actual settlement order for first-completed and first-success.
  - Structured exits merge child state and causal facts before return, break,
    continue, Workflow return, and Saga exit while preserving all outcomes.

##### `WF-CONTROL-003.a` Apply Repeat index to condition evaluation

- Status: Implemented.
- Completion evidence:
  - `R/workflow_structured.go:repeatAllowsNext` supplies the next iteration
    index to `while`.
  - `R/workflow_structured.go:settleRepeatChild` supplies the completed body
    index to `until` with carried state and result facts.

##### `WF-CONTROL-003.b` Settle empty collections and preserve maxima

- Status: Implemented.
- Completion evidence:
  - `R/types.go` retains `WorkflowForeach.MaxItems` and
    `WorkflowRepeat.MaxIterations` as pointers.
  - `S/workflowruntime/compiler_structured.go` preserves explicit zero values
    from the admitted declaration.
  - `R/workflow_structured.go:launchForeachChildren` re-enters
    `drivePending` after an empty collection completes so a declared parent
    state merge emits its durable StateUpdate WorkItem.

##### `WF-CONTROL-003.c` Limit immediate Machine transitions only

- Status: Implemented.
- Completion evidence:
  - `R/workflow_machine.go:selectMachineTransition` increments the counter
    only for immediate transitions.
  - Event and timer transitions reset it.
  - `startMachineChild` and `armMachineWaits` reset it at genuine suspension
    boundaries while retaining the immediate-transition bound.

##### `WF-CONTROL-003.d` Preserve state and causality through structured exits

- Status: Implemented.
- Completion evidence:
  - `settleSingleStructuredChild`, `settleRepeatChild`,
    `settleMachineChild`, `settleSagaBodySuccess`, catch, and finally merge
    state and causal frontiers before nonlocal control exits.

#### `WF-OUTCOME-003` Preserve complete Workflow and conversation terminal meaning

- Status: Implemented.
- Completion evidence:
  - `workflow_outcomes.go`, child completion, reconciliation, and
    cross-Execution settlement preserve typed errors, categories, causes,
    receipts, effects, outputs, aggregates, and declaration paths.
  - `Attempt.LLMStream`, `PortStream.AttemptID`,
    `appendInterruptedAssistantActivityLocked`, and projection logic keep
    interrupted and replayed output separate by Attempt.

##### `WF-OUTCOME-003.a` Record complete settled step facts

- Status: Implemented.
- Completion evidence:
  - Ordinary, inline, child, reconciliation, cancellation, return, break,
    continue, and recovery paths record the actual step and clear output for
    terminal states that cannot carry output.

##### `WF-OUTCOME-003.b` Persist interrupted assistant output truthfully

- Status: Implemented.
- Completion evidence:
  - Failed, cancelled, and uncertain LLM Attempts append one Attempt-scoped
    assistant Activity with actual terminal meaning. Replayed Attempts use
    separate streams and port-record identities.

### Conversation replay, configuration, continuation, handoff, and Skill state

#### `V3-REPLAY-003` Reconstruct private resources from the exact retained operation

- Status: Implemented.
- Completion evidence:
  - `ModelDispatchPlan.ProviderInputs`, `FinalSystemPrompt`, and
    `RuntimeConfiguration` retain exact provider inputs and final non-secret
    configuration.
  - `modelruntime.FinalizePreparedRuntimeConfiguration` retains final Model
    configuration with a Store-validated credential binding only.
  - `bindRecoveredLLM` reconstructs credentials, clients, and private Skill
    state from immutable plan facts without current defaults, history,
    contribution graphs, or later outcomes.

##### `V3-REPLAY-003.a` Retain semantic LLM bindings

- Status: Implemented.
- Completion evidence:
  - One canonical immutable dispatch plan retains provider inputs, Tool
    choices, pins, prompt, configuration, protocol facts, attachments, App
    context, and canonical generated-Tool origin references.

##### `V3-REPLAY-003.b` Reproduce captured message order and configuration

- Status: Implemented.
- Completion evidence:
  - Recovery decodes `ProviderInputs` directly and verifies the final prompt
    from `RuntimeConfiguration`; it does not recompose current Session data.

##### `V3-REPLAY-003.c` Separate interrupted and replayed streams

- Status: Implemented.
- Completion evidence:
  - `Attempt.LLMStream`, `LLMStreamState.AttemptID`, and
    Attempt-qualified port stream identities prevent replay from extending an
    earlier interrupted Attempt stream.

#### `V3-CONFIG-003` Complete inherited and explicit invocation configuration

- Status: Implemented.
- Completion evidence:
  - `SessionCollectionPolicies` preserves collection replacement, explicit
    empty clearing, removed keys, and override intent.
  - `applySpecificCollections`,
    `llmExecutionPlan.applySpecificInstructionPolicy`, `Instruction.Owner`,
    `Instruction.SourceKey`, and `InstructionDispatchPlan.Owner` preserve
    policy precedence, explicit removal, empty replacement, owner provenance,
    and ordering.

##### `V3-CONFIG-003.a` Preserve collection policy durably

- Status: Implemented.
- Completion evidence:
  - `CollectionPolicy` stores mode and removed keys.
  - `SessionDefaults` persists one policy for each supported collection.
  - `SessionDefaultsOverride` preserves explicit empty replacement through
    pointer presence and stores explicit policy when supplied.

##### `V3-CONFIG-003.b` Apply effective contribution precedence

- Status: Implemented.
- Completion evidence:
  - Specific Session and invocation policies filter inherited Agent and
    Workspace Text, instruction Skill, Tool, Skill, MCP, prompt, and context
    effects before final request construction.

##### `V3-CONFIG-003.c` Retain truthful ownership and ordering

- Status: Implemented.
- Completion evidence:
  - The canonical Model dispatch plan retains effective owner and order facts
    without copying complete effective collections into Agent or Workspace
    records.

#### `DOMAIN-SKILL-003` Reconstruct and release Skill resources for the exact owner

- Status: Implemented.
- Completion evidence:
  - `V3Executor.Choices` and `EffectiveActiveSelectionsForFrame` use the exact
    Execution and ProgramFrame owner key.
  - `runtime_lifecycle.go:scheduleCommittedFrameClosuresLocked` starts private
    frame cleanup only after the terminal ownership checkpoint commits.

##### `DOMAIN-SKILL-003.a` Select applicable active Skill state

- Status: Implemented.
- Completion evidence:
  - Active Skill lookup filters terminal lifecycle Tool results by exact frame,
    enabled pins, and durable completion order.

##### `DOMAIN-SKILL-003.b` Release Skill sessions on natural completion

- Status: Implemented.
- Completion evidence:
  - Committed terminal frame lifecycle cleanup covers normal completion,
    cancellation, deadline, panic, Fork, Join, cleanup expiry, and parent
    terminal propagation.

#### `V3-CONTINUE-003` Complete active-operation continuation and manual Tool ownership

- Status: Implemented.
- Completion evidence:
  - `UpdateContinuationInput`, `ContinueExecution`,
    `conversationMessageExecutionLocked`, and
    `requestAgentContinuationLocked` route public and staged continuation to
    one active semantic conversation owner, including handoff ownership.
  - Existing consumed-result and current Tool-batch protections scope manual
    Tool control and automation holds to the active operation.

##### `V3-CONTINUE-003.a` Resume the active conversation operation

- Status: Implemented.
- Completion evidence:
  - Staged continuation transfers exact defaults, budget, attachments, Tool
    results, and approved App context to the active owner frame.

##### `V3-CONTINUE-003.b` Scope manual controls to the current operation

- Status: Implemented.
- Completion evidence:
  - Manual Tool mutation protects consumed results, current Tool-batch
    ownership, and active conversation ownership.

#### `WF-HANDOFF-003` Complete durable multi-message handoff ownership

- Status: Implemented.
- Completion evidence:
  - `R/program.go` stores queued Activity ID, invocation defaults, and
    invocation budget in Agent-loop state.
  - A queued handoff round appends the completed assistant Activities before it
    starts the next queued LLM WorkItem.
  - `CompleteConversationHandoffCommand` emits
    `ProgramEventHandoffComplete`.
  - Agent-loop completion uses an ordinary completed ProgramFrame terminal
    result; `advanceFrameLocked` invokes existing restore or retain ownership
    settlement.

##### `WF-HANDOFF-003.a` Preserve queued message semantic facts

- Status: Implemented.
- Completion evidence:
  - `conversationQueuedActivity` carries immutable defaults and budget.
  - Agent-loop queue conversion preserves legacy queued IDs.
  - `startQueuedAgentLLM` applies queued facts to the next LLM WorkItem.
  - The preceding assistant Activity remains attached to the transition that
    starts queued handoff work.

##### `WF-HANDOFF-003.b` Complete or relinquish handoff successfully

- Status: Implemented.
- Completion evidence:
  - `R/Runtime.CompleteConversationHandoff` targets one active handoff frame.
  - `ProgramEventHandoffComplete` completes only an idle handoff Agent frame.
  - `settleConversationHandoffLocked` applies the durable restore or retain
    policy after normal ProgramFrame completion.

### Domain setup ownership, binder composition, attachment meaning, and duplication

#### `ARC-DOMAIN-003` Complete owning-domain setup boundaries

- Status: Partial.
- Completion evidence:
  - `attachmentruntime.SessionV3Admitter` owns V3 attachment admission and
    immutable snapshot construction.
  - `modelruntime.FinalizePreparedRuntimeConfiguration` owns final effective
    non-secret Model configuration capture.
  - Workflow and Session setup compose neutral domain facts. Core Runtime
    remains Artifact Store independent.
- Remaining functional gap:
  - Workflow Model and Agent preparation still contains a second Agent and
    Workspace contribution interpreter. The required domain-owned preparation
    consolidation is tracked by `WF-PREPARE-004`.

##### `ARC-DOMAIN-003.a` Move contribution preparation to owning domains

- Status: Partial.
- Remaining functional gap:
  - `sessionv3runtime/llm_plan.go:applyAgentPlan` and
    `workflowruntime/model_agent_adapter.go` independently traverse Agent and
    Workspace capability facts. The Workflow path also omits Workspace runtime
    prompt and context materialization.
  - Agent, Workspace, Model, Text, Tool, Skill, and MCP domains must expose
    one neutral prepared contribution result that normal Session setup and
    Workflow setup both consume.

##### `ARC-DOMAIN-003.b` Move attachment and Model conversion to owning domains

- Status: Implemented.
- Completion evidence:
  - Attachment setup owns snapshot admission. Model setup owns final
    configuration materialization and private credential reconstruction.

#### `V3-API-003` Preserve optional backend capabilities through final binder composition

- Status: Implemented.
- Completion evidence:
  - `S/workflowruntime/binder.go` forwards
    `PrepareUserMessageSkillTemplate` and `PreviewInstructionSkill` to Base.
  - Base remains the owner of Skill rendering and preview behavior.

##### `V3-API-003.a` Forward Skill template preparation and preview

- Status: Implemented.
- Completion evidence:
  - `S/workflowruntime.Binder` implements both optional SessionV3 capability
    interfaces and delegates without creating another rendering path.

#### `DOMAIN-ATTACH-003` Preserve semantic attachment identity in native Model input

- Status: Implemented.
- Completion evidence:
  - `attachmentruntime.SessionV3Admitter` preserves admitted labels and stable
    directory-relative paths. `inputs.go:attachmentToContentItem` retains
    text, image, and file labels without reopening source paths.

##### `DOMAIN-ATTACH-003.a` Carry labels and directory-relative names

- Status: Implemented.
- Completion evidence:
  - Text uses `attachmentTextForProvider`, images set `ImageName`, and files
    set `FileName` from the immutable admitted label.

#### `ARC-DUP-003` Remove replicated effective state and duplicated execution algorithms

- Status: Partial.
- Completion evidence:
  - `persistLocked` is the sole committed ready-work launcher.
  - `runtime_lifecycle.go` is the shared Runtime terminal lifecycle.
  - `ModelDispatchPlan` is the canonical immutable LLM plan and generated
    Tool WorkItems retain canonical origin-plan references.

##### `ARC-DUP-003.a` Retain one canonical effective plan

- Status: Implemented.
- Completion evidence:
  - Top-level `ModelDispatchPlan` is canonical.
  - Nested Agent and Workspace dispatch structures do not duplicate complete
    effective collections.

##### `ARC-DUP-003.b` Reference canonical origin plans and attachment content

- Status: Implemented.
- Completion evidence:
  - `LLMToolOrigin` resolves its immutable plan from the original LLM
    WorkItem through `PlanReference`.
  - New dispatch-plan construction retains attachments only in captured
    message facts.

##### `ARC-DUP-003.c` Consolidate Runtime algorithms and lowered-tree traversal

- Status: Implemented.
- Completion evidence:
  - Shared lifecycle, captured evaluation, owner accounting, retry matching,
    effect replay, channel forwarding, and scoped budget algorithms remain
    Runtime-owned. Native request grammars remain in Model, Tool, MCP, Skill,
    Service, Activity, Attachment, Agent, and Workspace domains.

##### `ARC-DUP-003.d` Keep one V3 attachment admission implementation

- Status: Not implemented.
- Source evidence:
  - `attachmentruntime/sessionv3_admission.go` and
    `sessionv3runtime/attachments.go` both implement V3 file, directory, and
    URL admission, snapshot, mode, and digest behavior.
  - `sessionv3runtime.Binder` uses
    `attachmentruntime.NewSessionV3Admitter`, leaving the second implementation
    as parallel behavior that can drift.
- Required behavior:
  - Keep one Attachment-domain implementation and reduce any remaining setup
    symbol to a non-behavioral forwarding compatibility layer or remove it.

### Runtime setup lifecycle

#### `RT-SETUP-003` Publish and release the composed backend Runtime safely

- Status: Implemented.
- Completion evidence:
  - `cmd/agentgo/wrapper_sessionv3_bootstrap.go:InitComposedSessionV3Runtime`
    closes a restored Runtime after final wrapper publication fails.
  - `cmd/agentgo/wrapper_sessionv3_runtime.go:SessionV3Wrapper.close` retains
    Runtime-before-Store shutdown order.

### Pending functional tasks from this audit

The following tasks are the current Runtime and setup completion units.
An implementation owner must update their statuses and the affected `Partial`
entries in this ledger in the same delivery. This work remains WIP. It must
not create tests, validation-only work, frontend work, SessionV2 retirement
work, migration cleanup, cost metering, or version changes.

#### `RT-DISPATCH-004` Make committed persistence the one normal ready-work launch authority

- Status: Not implemented.
- Required behavior:
  - `persistLocked` is the one normal committed checkpoint-to-ready-work
    boundary.
  - Remove manual ready-work launches after successful persistence from
    command, trigger, timer, budget, and settlement paths.
  - Retain one explicit Session-wide ready-work wave only after complete
    restore finishes.

##### `RT-DISPATCH-004.a` Remove duplicated normal ready-work launches

- Status: Not implemented.
- Required behavior:
  - Route every normal ready-work launch through `persistLocked`.
  - Keep dispatch guards as safety checks, not as the mechanism that hides
    duplicated launch waves.

##### `RT-DISPATCH-004.b` Preserve complete restore readiness behavior

- Status: Not implemented.
- Required behavior:
  - Keep the post-restore Session-wide readiness wave after all restored
    Sessions, timers, trigger recovery, waits, and reconciliation state are
    stable.

#### `RT-PORT-BUDGET-004` Account nested Workflow port records at their semantic owner

- Status: Not implemented.
- Required behavior:
  - Charge published input-port records to the receiving child frame and its
    effective structured budget scope.
  - Persist receiving-side ownership in forwarded-port facts so a parent-side
    forwarded record does not use a foreign child WorkItem or frame identity.
  - Charge every retained source and forwarded record exactly at the boundary
    that owns that retained record.

##### `RT-PORT-BUDGET-004.a` Account published child input records

- Status: Not implemented.
- Required behavior:
  - Use the child frame identity already available during published-port input
    staging when reserving message count and message byte budget.

##### `RT-PORT-BUDGET-004.b` Account cross-Execution forwarded records

- Status: Not implemented.
- Required behavior:
  - Extend `ChannelForward` with receiving-side owner facts and use them for
    parent Execution budget attribution.

#### `WF-PREPARE-004` Preserve complete, domain-owned Workflow Model and Agent invocation meaning

- Status: Not implemented.
- Required behavior:
  - Replace Workflow-specific Agent and Workspace capability walkers with one
    neutral, source-pinned domain-prepared contribution result.
  - Capture the actual Workspace runtime-plan instructions, user context,
    selected pins, Tool selections, Skills, MCP selections, Model prompt
    behavior, and contribution order before `ProgramSnapshot` creation.
  - Do not mark Workspace prompt facts prepared when the prompt and context
    payload is absent.
  - Consume the captured order and facts during Workflow LLM binding. Private
    resource reconstruction may use captured pins but must not rebuild current
    Agent or Workspace contribution graphs.

##### `WF-PREPARE-004.a` Reuse owner-domain contribution preparation

- Status: Not implemented.
- Required behavior:
  - Make normal V3 setup and Workflow setup consume the same Agent and
    Workspace prepared contribution semantics.

##### `WF-PREPARE-004.b` Preserve and consume complete Workspace invocation facts

- Status: Not implemented.
- Required behavior:
  - Persist and apply Workspace prompt instructions, user context, and
    deterministic contribution order for Workflow Model and Agent calls.

#### `V3-STORE-004` Provide Store-owned semantic Session export

- Status: Not implemented.
- Required behavior:
  - Add a V3 Store export that reads the durable Session document and returns
    semantic Session data only.
  - Expose the export through `SessionV3Wrapper`.
  - Do not read live private bindings, clients, callbacks, Artifact Store, or
    frontend state while producing the export.
  - This is V3 backend functionality. It does not create V2 conversion,
    retirement, or deletion work.
