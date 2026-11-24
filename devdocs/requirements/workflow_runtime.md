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

The current audit inspected the pasted Runtime, setup, Store v3, bootstrap,
wrapper, and Workflow consumer-handoff sources. References to source evidence
mean inspected function bodies, not completion statements copied from this HLD.

The Workspace prompt producer was not supplied. The consumer injects both
Workspace prompt instructions and individual instruction Text contributions.
Payload overlap is provisional until the owning Workspace prompt builder is
inspected. The required single-injection ownership is recorded below.

Workflow declaration and static Artifact planning are accepted as complete for
this implementation cycle.

This HLD does not publish a completion percentage. Completion estimates are
reported outside this status record.

Session Runtime v3 and Workflow Runtime v3 remain work in progress (`WIP`). No
version identifier anywhere in declarations, schemas, Store documents,
interpreters, packages, or APIs may be bumped in this cycle. Most Runtime,
Store v3, and setup tasks are implemented in source, while source-verified
pending items remain tracked as `Partial` or `Not implemented` in `Functional
tasks` at the end of this document. An implementer must update those task
statuses and completion evidence to `Implemented` only when the complete
behavior exists in source.

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

Completed follow-up IDs remain historical evidence, not additional delivery
units. A later functional deficiency must identify the affected parent and the
specific missing behavior. It must not reopen a completed bounded change or
create a second implementation of it.

Each open subtask below has its own status. The implementer must replace its
pending completion evidence with exact owning-source symbols in the same
delivery. Parent status changes only after every listed open subtask is complete.

The `Implemented` status remains source-based. Test execution and application
execution were not reviewed. This HLD remains work in progress, and no version
identifier has changed while these tasks were implemented.

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
  when it creates an Activity for a normal-conversation Execution. Active
  handoff-owner consumption remains open under `V3-CONTINUE-007`.
- V3 consumes the selected Agent Model capability `IncludeSystemPrompt` during
  Model patch composition. Effective selected-Model precedence remains open
  under `WF-PREPARE-005` and `V3-CONFIG-007`.
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
source grammar, re-decode immutable `ProgramSnapshot` payloads on every
checkpoint for unchanged or terminal frames, or repeat `Session.Validate`
multiple times on the same Store write. Lowering of already contract-admitted
Workflow declarations validates the final lowered program once rather than
revalidating every intermediate node. Runtime, Store, and setup code must use
`const` declarations and typed enums instead of hardcoded strings, and all
error strings must be lowercase.

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

- Status: Partial.
- Completion evidence:
  - `internal/agentruntime-go/sessionv3/runtime.go:persistLocked` is the
    committed Session-wide ready-work launch boundary. It accepts committed
    derived-index errors and clears stale checkpoint barriers before readiness.
  - `runtime_lifecycle.go:finalizeExecutionSettlementLocked`,
    `commitTerminalMutationLocked`,
    `settlePendingFrameCancellationsLocked`, and
    `executionTerminalReadyLocked` complete attempt, effect, claim, wait,
    owner, frame cancellation, Saga, Execution, checkpoint, timer rearm, and
    post-commit dispatch lifecycle work in one pre-checkpoint sequence.
  - `runtime.go:dispatchOne` persists `WorkItemStateBinding` reverts to
    `WorkItemStateReady` (`checkpointReasonWorkBindingReverted`), `ResolveWait`
    schedules wait deadlines only after `persistLocked` commits
    `checkpointReasonWaitResolutionFailed`, and `startDispatchedLocked`
    persists once at `checkpointReasonWorkDispatched`.

##### `RT-SETTLE-003.a` Reacquire authoritative Session-owned objects

- Status: Partial.
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
- Completion evidence:
  - `R/runtime.go:persistLocked` is the sole normal committed
    checkpoint-to-ready-work boundary.
  - `R/runtime_lifecycle.go:finalizeExecutionSettlementLocked` and
    `commitTerminalMutationLocked` and `R/budget.go:waitForCleanupDeadline`
    converge every terminal WorkItem, frame-cancellation, child, wait,
    resource, effect, Saga, and Execution
    mutation before `persistLocked`.
  - `R/Runtime.Restore` retains the one explicit Session-wide post-restore
    readiness wave after `resumeRestoredWork` completes.

#### `RT-CLEANUP-003` Execute cleanup through every owning lifecycle

- Status: Partial.
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
  - `budget.go:waitForCleanupDeadline` and `expireExecutionCleanupLocked`
    settle remaining WorkItems, waits, frames, resource leases,
    `CompensationStatePending` (`CompensationStateSkipped`) and
    `CompensationStateDispatching` (`CompensationStateUncertain`)
    `EffectRecord`s, and `CancelWithParent` child Executions before committing
    via `commitTerminalMutationLocked`.

##### `RT-CLEANUP-003.a` Carry cleanup classification and effective deadline

- Status: Partial.
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

- Status: Partial.
- Completion evidence:
  - `ProgramFrame.CancellationRequested`,
    `settlePendingFrameCancellationsLocked`, and
    `executionTerminalReadyLocked` keep frame and Execution ownership alive
    until dispatched work, claims, reconciliation, compensation, cleanup, and
    completion delivery settle.
  - `budget.go:expireExecutionCleanupLocked` retires uncompleted Saga
    `EffectRecord`s and `CancelWithParent` child Executions on cleanup expiry
    so terminal Execution settlement completes.

#### `RT-BUDGET-003` Complete scoped accounting and resource-ready scheduling

- Status: Partial.
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

- Status: Partial.
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
- Completion evidence:
  - `R/ports_runtime.go:appendPublishedWorkflowPortInputLocked` charges each
    retained child input record through its receiving ProgramFrame and that
    frame's structured scope.
  - `R/ports.go:ChannelForward` persists receiving WorkItem, ProgramFrame,
    and budget-scope ownership facts.
  - `R/ports_runtime.go:appendForwardedChannelMessageLocked` uses receiving
    ownership only. It does not charge a parent Execution through foreign
    child WorkItem or ProgramFrame identities.
  - `R/budget_scope.go:reserveMessageBudgetForSemanticOwnerLocked` performs
    one count and byte reservation for each retained physical record.

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

- Status: Partial.
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

- Status: Partial.
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

- Status: Partial.
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
  - `R/PreparedContributionPlan` is the neutral source-pinned contract used
    by normal Session V3 setup and Workflow Model or Agent preparation.
  - It retains immutable Model, Agent, Workspace, Text, Tool, Skill, MCP,
    rendered instruction-mode Skill text, prepared MCP context, discovery
    identity, and provider Tool plans, Workspace prompt, user context, ordered
    contribution, and provenance facts.
  - `program.go:startAgentLLM`, retry capture, started calls, handoff,
    nested preparation, and recovery retain one immutable prepared invocation.
  - `compiler_invocation.go` lowers typed expression-backed message and
    instruction collections. `workflow_invocation.go:captureWorkflowLLMInvocation`
    evaluates and freezes those collections at the logical call boundary.
  - `S/mcpruntime/v3.go:collectMCPServerChoices` emits one choice per selected
    MCP tool; `S/contributionruntime/prepared.go` filters overrides before live
    capability preparation, skips unready optional Agent capabilities, and
    pins/prepares `context.defaults` selections; and `S/workflowruntime`
    populates per-call-site `PreparedTargets` for Model, Agent, and
    ProviderTool calls.

##### `WF-PREPARE-003.a` Retain immutable pins and prepared contribution facts

- Status: Partial.
- Completion evidence:
  - `S/contributionruntime/prepared.go` captures source-pinned Agent and
    Workspace contributions before Workflow ProgramSnapshot creation, rendering
    instruction-mode Skills through `InstructionSkillRenderer` into
    `WorkflowPreparedSkill.InstructionContent` and capturing MCP prompt
    context, discovery digest, policy, and provider Tool plans through
    `MCPContributionPreparer` into `WorkflowPreparedMCP`.
  - `S/sessionv3runtime/workflow_invocation.go`, `llm_plan.go`, and
    `tools_choices.go` consume captured Text, instruction Skill, MCP context,
    and MCP Tool choices without late Skill rendering or live MCP discovery
    reselection.
  - `S/sessionv3runtime/llm_replay.go` reconstructs only private resources
    from captured dispatch facts and exact provider inputs.
  - `S/workflowruntime/compiler_invocation.go:prepareWorkflowLLMInvocations`
    and `model_agent_adapter.go` freeze call-site `context.defaults` overrides
    in `PreparedTargets`, and `S/contributionruntime/prepared.go` skips
    unavailable optional Agent capabilities (`occurrence.Required == false`).

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
    `ProgramFrameIntent.PortInputs`, `workflow_retry.go`,
    `workflow_program.go`, and
    `retry_program.go` capture actual port values and cursors before dispatch
    or child materialization.
  - `workflow_retry.go:captureWorkflowRetryCall` resolves call port plans and
    captures all input-port values and cursors at the original logical call
    boundary before retry child creation.
  - `ports_runtime.go:portInputsLocked` respects captured inputs and does not
    append later channel records on retry or recovery.

##### `WF-RETRY-003.a` Capture the original evaluation boundary

- Status: Implemented.
- Completion evidence:
  - Captured target, target-set membership, input, context, messages,
    instructions, port input values and cursors, key, policy, declaration
    path, and logical time are captured at the original logical call boundary
    and preserved through retry, delayed child creation, recovery, and
    resumption.

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
  - `R/ports_runtime.go:preparePublishedWorkflowPortBoundaryLocked` resolves
    the caller Execution and receiving frame/scope owner through
    `resolvePortChannelTarget` / `portChannelExecution` for independent started
    nested Workflow calls.

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

##### `WF-CHANNEL-003.d` Forward published nested Workflow ports across independent started calls

- Status: Implemented.
- Completion evidence:
  - `R/ports_runtime.go:preparePublishedWorkflowPortBoundaryLocked` and
    `resolvePortChannelTarget` (shared with `portChannelExecution`) resolve
    `TargetExecutionID` and receiving frame/budget-scope ownership across
    wrapper frames and parent Executions so `mode: start` with
    `lifetime: independent` delivers published nested Workflow output and
    streamed ports to the caller Execution channel.

### Workflow effects, recovery, structured control, and outcomes

#### `WF-EFFECT-003` Preserve declaration effects and actual business-effect facts

- Status: Partial.
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
  - `R/runtime.go:finishWorkLocked` handles read-only failure before mutating
    uncertainty. Go and Skill executors contain conversation error-output
    branches. MCP lacks the equivalent conversation path.
  - A completed error observation does not by itself prove business-effect
    non-application. Complete this distinction under `V3-TOOL-006`; complete
    keyed target representation under `WF-EFFECT-007`.

##### `WF-EFFECT-003.a` Translate `non-idempotent`

- Status: Implemented.
- Completion evidence:
  - `S/workflowruntime/compiler.go:compileWorkflowSourceEffectKind` explicitly
    maps source `non-idempotent` to Runtime `EffectKindNonIdempotent`.
  - Target-level keyed capability declarations lower to an idempotent base
    contract with separate keyed capability facts.

##### `WF-EFFECT-003.b` Settle MCP from actual domain evidence

- Status: Partial.
- Completion evidence:
  - `S/mcpruntime/v3.go:V3Executor.Execute` returns confirmed success without
    forcing `EffectStateUncertain`.
  - Successful MCP output can settle through Runtime without an explicit
    receipt. The current executor does not create the claimed MCP success receipt.
  - Mutating MCP invocation errors and `IsError` responses currently enter
    uncertainty without a conversation-source branch. `V3-TOOL-006` owns the
    missing consumable error observation and truthful effect classification.

##### `WF-EFFECT-003.c` Preserve compensation causality and operation facts

- Status: Implemented.
- Completion evidence:
  - Compensation WorkItems retain original WorkItem, Attempt, frame, target,
    receipt, effect, idempotency key, compensation WorkItem, and compensation
    ProgramFrame identities before child materialization.

##### `WF-EFFECT-003.d` Preserve target-owned effect and keying guarantees

- Status: Partial.
- Completion evidence:
  - Keyed replay requires a stable captured key and
    `WorkflowNativeIdempotencyKeying`. Go Tool, MCP Tool, Skill Tool, Model,
    and Agent grammars receive no generic Workflow metadata.

#### `WF-CONTROL-003` Complete structured-control value and lifetime behavior

- Status: Partial.
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

- Status: Partial.
- Completion evidence:
  - `settleSingleStructuredChild`, `settleRepeatChild`,
    `settleMachineChild`, `settleSagaBodySuccess`, catch, and finally merge
    state and causal frontiers before nonlocal control exits.

#### `WF-OUTCOME-003` Preserve complete Workflow and conversation terminal meaning

- Status: Partial.
- Completion evidence:
  - `workflow_outcomes.go`, child completion, reconciliation, and
    cross-Execution settlement preserve typed errors, categories, causes,
    receipts, effects, outputs, aggregates, and declaration paths.
  - `Attempt.LLMStream`, `PortStream.AttemptID`,
    `appendInterruptedAssistantActivityLocked`, and projection logic keep
    interrupted and replayed output separate by Attempt.

##### `WF-OUTCOME-003.a` Record complete settled step facts

- Status: Partial.
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

- Status: Partial.
- Completion evidence:
  - `SessionCollectionPolicies` preserves collection replacement, explicit
    empty clearing, removed keys, and override intent.
  - `applySpecificCollections`,
    `llmExecutionPlan.applySpecificInstructionPolicy`, `Instruction.Owner`,
    `Instruction.SourceKey`, and `InstructionDispatchPlan.Owner` preserve
    policy precedence, explicit removal, empty replacement, owner provenance,
    and ordering.

##### `V3-CONFIG-003.a` Preserve collection policy durably

- Status: Partial.
- Completion evidence:
  - `CollectionPolicy` stores mode and removed keys.
  - `SessionDefaults` persists one policy for each supported collection.
  - `SessionDefaultsOverride` preserves explicit empty replacement through
    pointer presence and stores explicit policy when supplied.

##### `V3-CONFIG-003.b` Apply effective contribution precedence

- Status: Partial.
- Completion evidence:
  - Specific Session and invocation policies filter inherited Agent and
    Workspace Text, instruction Skill, Tool, Skill, MCP, prompt, and context
    effects before final request construction.

##### `V3-CONFIG-003.c` Retain truthful ownership and ordering

- Status: Partial.
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

- Status: Partial.
- Completion evidence:
  - `UpdateContinuationInput`, `ContinueExecution`,
    `conversationMessageExecutionLocked`, and
    `requestAgentContinuationLocked` route public and staged continuation to
    one active semantic conversation owner, including handoff ownership.
  - `R/normal_session.go` routes manual Tool commands and `SetAutomationHold`
    through `resolveConversationControlExecutionIDLocked` and enforces
    terminal/cancelling, recovery-review, active-LLM, and current unconsumed
    batch guards, while `R/program.go:startQueuedAgentLLM` preserves
    `PendingToolIDs` into `startAgentLLM` to record `ConsumedByWorkItemID`.

##### `V3-CONTINUE-003.a` Resume the active conversation operation

- Status: Partial.
- Completion evidence:
  - Staged continuation transfers exact defaults, budget, attachments, Tool
    results, and approved App context to the active owner frame.
  - `R/program.go:startQueuedAgentLLM` passes settled `PendingToolIDs` into
    `startAgentLLM` so `advanceFrameLocked` records `ConsumedByWorkItemID` on
    consumed Tool WorkItems.

##### `V3-CONTINUE-003.b` Scope manual controls to the current operation

- Status: Implemented.
- Completion evidence:
  - `R/normal_session.go:resolveConversationControlExecutionIDLocked`,
    `validateManualToolControlExecution`, and `validateManualToolBatchItem`
    enforce active handoff conversation routing, terminal/cancelling rejection,
    pending recovery-review rejection, active (`Binding`/`Dispatched`) LLM
    WorkItem rejection, and current unconsumed Tool-batch ownership across
    `ExecuteToolCalls`,
    `RetryToolCall`, `SkipToolCall`, `CompleteToolCallAsError`, and
    `SetAutomationHold`.

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
  - `R/program.go` tracks `CompletedInputActivityIDs` across queued handoff
    turns and `S/sessionv3runtime/inputs.go:llmInputActivitySequence` preserves
    prior handoff user inputs, frame-owned assistant Activities, and completed
    tool outputs in chronological order without ambient Session history.

##### `WF-HANDOFF-003.a` Preserve queued message semantic facts

- Status: Implemented.
- Completion evidence:
  - `conversationQueuedActivity` carries immutable defaults and budget.
  - Agent-loop queue conversion preserves legacy queued IDs.
  - `startQueuedAgentLLM` applies queued facts to the next LLM WorkItem.
  - The preceding assistant Activity remains attached to the transition that
    starts queued handoff work.
  - `CompletedInputActivityIDs` and `llmInputActivitySequence` retain prior
    handoff user input Activities and frame-owned assistant/tool history across
    queued handoff turns.

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
  - `S/contributionruntime` owns neutral source-pinned Agent and Workspace
    contribution preparation. Workflow and normal Session setup consume the
    same contract.
  - Core Runtime remains Artifact Store independent.
  - `S/contributionruntime/prepared.go` applies collection overrides and
    `Clear` fields before live capability preparation, skips unready optional
    Agent capabilities, and pins/prepares Session and Workflow
    `context.defaults` selections.

##### `ARC-DOMAIN-003.a` Move contribution preparation to owning domains

- Status: Partial.
- Completion evidence:
  - `S/contributionruntime.Preparer` resolves Agent and Workspace domain
    capability facts, Workspace runtime-plan prompt facts, and immutable pins.
  - `S/sessionv3runtime/llm_plan.go` consumes that contract for normal Session
    LLM preparation.
  - `S/workflowruntime/model_agent_adapter.go` uses the same contract for
    Workflow Model and Agent preparation.
  - Both normal Session `llm_plan.go` and Workflow `context.defaults`
    preparation route overrides, exclusions, and optional Agent capability
    skips through `S/contributionruntime.Preparer`.

##### `ARC-DOMAIN-003.b` Move attachment and Model conversion to owning domains

- Status: Partial.
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

#### `V3-APP-003` Support Session-scoped MCP Apps in V3

- Status: Implemented.
- Completion evidence:
  - `R/normal_app.go`, `S/mcpruntime/v3.go`, `S/sessionv3runtime/binder.go`,
    `S/workflowruntime/binder.go`, and `cmd/agentgo/wrapper_sessionv3_runtime.go`
    implement `OpenAppView`, `ExecuteAppTool`, `StageAppContext`,
    `ReadAppResource`, and durable `AppInstance` projections.
  - `R/runtime.go:SubmitMessage` transfers unconsumed
    `execution.Continuation.AppContext` into the created user
    `MessageActivity` and clears consumed `AppContext`.

##### `V3-APP-003.a` Persist MCP App instances, views, resources, and App Tool calls

- Status: Implemented.
- Completion evidence:
  - `recordAppInstanceLocked`, `OpenAppView`, `ExecuteAppTool`,
    `ReadAppResource`, and `StageAppContext` persist semantic App facts without
    retaining live MCP sessions in checkpoints.

##### `V3-APP-003.b` Transfer staged App context on normal message submission

- Status: Implemented.
- Completion evidence:
  - `R/runtime.go:SubmitMessage` passes unconsumed
    `execution.Continuation.AppContext` into
    `submitNormalMessageWithDefaultsLocked`, clears consumed `AppContext`, and
    advances `ConsumedRevision` when no unconsumed draft text or attachments
    remain.

#### `ARC-DUP-003` Remove replicated effective state and duplicated execution algorithms

- Status: Partial.
- Completion evidence:
  - `persistLocked` is the sole committed ready-work launcher.
  - `runtime_lifecycle.go` is the shared Runtime terminal lifecycle.
  - `ModelDispatchPlan` is the canonical immutable LLM plan and generated
    Tool WorkItems retain canonical origin-plan references.
  - `attachmentruntime.SessionV3Admitter` is the canonical V3 attachment
    admission implementation.
  - `R/workflow_structured.go:WalkWorkflowBlock` is the single typed lowered
    Workflow tree traversal mechanism used by channel namespacing, resource
    namespacing, call lookup, and Inventory-driven declaration-path
    application.
  - `R/ArtifactPin.Selection()`, `S/contributionruntime` pinning helpers,
    `R/projection.go:BuildConversationProjection` and
    `BuildActivityProjection`, and the single `collectMCPServerChoices` loop
    exist.
  - `S/sessionv3runtime/attachments.go` still exists as a redundant delegate
    with an unused `maximumDirectoryFiles` field.
  - Store v3 still builds conversation and Activity projections separately.
    Remaining ownership and algorithm consolidation is `ARC-CLEAN-006`.

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

- Status: Partial.
- Completion evidence:
  - Shared lifecycle, captured evaluation, owner accounting, retry matching,
    effect replay, channel forwarding, and scoped budget algorithms remain
    Runtime-owned. Native request grammars remain in Model, Tool, MCP, Skill,
    Service, Activity, Attachment, Agent, and Workspace domains.
  - `R/ports_runtime.go:namespaceWorkflowBlockChannels`,
    `R/resource_namespace.go:namespaceWorkflowBlockResources`,
    `S/workflowruntime/declaration_paths.go:attachWorkflowDeclarationPaths`,
    and `S/workflowruntime/compiler_invocation.go:workflowCallAtDeclarationPath`
    all use `R/workflow_structured.go:WalkWorkflowBlock`.
  - Deduplicated `S/mcpruntime/v3.go:collectMCPServerChoices`, consolidated
    shared pinning/selection helpers in `S/contributionruntime`, added
    `BuildConversationProjection` and `BuildActivityProjection` in
    `R/projection.go`.
  - Remaining Runtime algorithms, native-domain conversion ownership, Store
    projection reuse, and literal consistency are `ARC-CLEAN-006`.

##### `ARC-DUP-003.d` Keep one V3 attachment admission implementation

- Status: Implemented.
- Completion evidence:
  - `S/attachmentruntime/sessionv3_admission.go` is the only public V3
    admission implementation for file, directory, and URL sources.
  - It preserves immutable content, digest, label, directory-relative label,
    and URL source facts in the durable Attachment snapshot.
  - Session setup uses that implementation directly. The unused compatibility
    delegate still exists and is tracked under `ARC-CLEAN-006`.

### Runtime setup lifecycle

#### `RT-SETUP-003` Publish and release the composed backend Runtime safely

- Status: Implemented.
- Completion evidence:
  - `cmd/agentgo/wrapper_sessionv3_bootstrap.go:InitComposedSessionV3Runtime`
    closes a restored Runtime after final wrapper publication fails.
  - `cmd/agentgo/wrapper_sessionv3_runtime.go:SessionV3Wrapper.close` retains
    Runtime-before-Store shutdown order.

### Audited completed follow-up tasks

The following tasks were previously listed as pending. The inspected source
contains their listed behavior, so their `Implemented` status is retained.
They remain as completion evidence and are not open implementation work.

The current pending functional tasks follow this completed record. An
implementation owner must update their statuses and affected `Partial` entries
in the same delivery. This work remains WIP and must not create tests,
validation-only work, frontend work, SessionV2 retirement work, migration
cleanup, cost metering, or version changes.

#### `RT-DISPATCH-004` Make committed persistence the one normal ready-work launch authority

- Status: Implemented.
- Completion evidence:
  - `R/runtime.go:persistLocked` launches normal ready work only after the
    durable checkpoint commits.
  - Normal budget, timer, trigger, settlement, App Tool, and edit-and-resend
    paths persist through that boundary without a second launch wave.
  - `R/Runtime.Restore` retains one explicit post-restore Session-wide wave.

##### `RT-DISPATCH-004.a` Remove duplicated normal ready-work launches

- Status: Implemented.
- Completion evidence:
  - `R/normal_app.go`, `R/normal_session.go`, `R/runtime_durability.go`, and
    `R/runtime_triggers.go` no longer launch ready work after `persistLocked`.
  - Dispatch guards remain stale-work and idempotency safety checks only.

##### `RT-DISPATCH-004.b` Preserve complete restore readiness behavior

- Status: Implemented.
- Completion evidence:
  - `R/runtime.go:Restore` clears `restoring` only after
    `resumeRestoredWork`, then launches one Session-wide ready-work wave.

#### `RT-PORT-BUDGET-004` Account nested Workflow port records at their semantic owner

- Status: Implemented.
- Completion evidence:
  - `R/ports_runtime.go:appendPublishedWorkflowPortInputLocked` reserves the
    receiving child frame and its effective scope.
  - `R/ports.go:ChannelForward` retains receiving WorkItem, frame, and scope
    facts for each durable forwarding edge.
  - `R/ports_runtime.go:appendForwardedChannelMessageLocked` charges the
    receiving parent owner for each retained forwarded record.

##### `RT-PORT-BUDGET-004.a` Account published child input records

- Status: Implemented.
- Completion evidence:
  - Published input staging uses `childFrame.ID` and
    `childFrame.BudgetScopeID` through
    `reserveMessageBudgetForSemanticOwnerLocked`.

##### `RT-PORT-BUDGET-004.b` Account cross-Execution forwarded records

- Status: Implemented.
- Completion evidence:
  - Forward creation persists receiving ownership from the parent child-call
    boundary.
  - Forward delivery rejects unavailable receiving owners and never charges a
    target Execution through source-side identities.

#### `WF-PREPARE-004` Preserve complete, domain-owned Workflow Model and Agent invocation meaning

- Status: Partial.
- Completion evidence:
  - `S/contributionruntime/prepared.go` produces the common neutral
    source-pinned contribution contract.
  - It captures Workspace runtime-plan instructions, user context, selected
    Text, Tool, Skill, MCP, Model prompt behavior, immutable pins, and typed
    contribution order before Workflow ProgramSnapshot creation.
  - `PreparedContributionPlan.WorkspacePromptPrepared` is set only with a
    complete immutable `WorkspacePrompt` payload.
  - `S/sessionv3runtime/workflow_invocation.go` consumes captured order and
    facts without graph traversal. Recovery uses the immutable dispatch plan.

##### `WF-PREPARE-004.a` Reuse owner-domain contribution preparation

- Status: Implemented.
- Completion evidence:
  - Normal `S/sessionv3runtime/llm_plan.go` and Workflow
    `S/workflowruntime/model_agent_adapter.go` both call
    `S/contributionruntime.Preparer`.

##### `WF-PREPARE-004.b` Preserve and consume complete Workspace invocation facts

- Status: Partial.
- Completion evidence:
  - Workspace runtime-plan prompt instructions and user context are retained
    in `PreparedContributionPlan.WorkspacePrompt`.
  - `applyPreparedContributionPlan` applies the persisted typed order when it
    composes Workflow Model and Agent child invocation facts.

#### `V3-STORE-004` Provide Store-owned semantic Session export

- Status: Implemented.
- Completion evidence:
  - `internal/sessionstorev3.Store.ExportSemanticSession` reads only the
    durable Session document through `ReadSession`.
  - `cmd/agentgo/SessionV3Wrapper.ExportSession` exposes that Store-owned
    semantic export.
  - The export has no dependency on live Runtime maps, private bindings,
    clients, callbacks, Artifact Store, or frontend state.
  - The capability adds no V2 conversion, retirement, deletion, migration, or
    version-change work.

##### `V3-STORE-004.a` Maintain FTS search and derived index repair

- Status: Partial.
- Completion evidence:
  - `internal/sessionstorev3/store.go` and `store_fts.go` maintain derived
    Session index files, SQLite FTS search, `DerivedIndexError`
    (`CheckpointCommitStatus`), startup `RebuildDerived`, and background repair.
  - That repair currently rebuilds index files only. Failed FTS updates are
    logged without FTS reconciliation. Remaining search repair is `V3-STORE-007`.

### Reconciled follow-ups and remaining functional work

Tasks marked `Partial` or `Not implemented` in this section and in their parent
entries above are the known remaining backend Runtime and setup work found in
the supplied source. Completed follow-ups below are retained as bounded source
evidence, not instructions to implement them again. Each new task identifies
its affected entries. This remains one substantial WIP delivery with the
existing exclusions and no version changes.

#### `RT-SETTLE-005` Commit terminal mutations before the sole terminal checkpoint

- Status: Implemented.
- Completion evidence:
  - `R/runtime_lifecycle.go:finalizeExecutionSettlementLocked` converges Saga
    compensation resumption, failure-driven cleanup, pending frame
    cancellations, Execution cancellation completion, Execution state refresh,
    and terminal Execution settlement before any checkpoint.
  - `R/runtime_lifecycle.go:commitTerminalMutationLocked` runs
    `finalizeExecutionSettlementLocked`, `touchExecutionLocked`,
    `persistLocked`, `armSessionExecutionTimersLocked`, and `publishLocked` in
    one sequence.
  - `R/runtime.go:executeOne` and `R/runtime.go:ResolveWait` no longer perform
    post-checkpoint `settlePendingFrameCancellationsLocked` mutations.
  - `R/runtime_lifecycle.go:scheduleCommittedFrameClosuresLocked` schedules
    both ProgramFrame and Execution private binding cleanup only after
    `persistLocked` commits.
  - `dispatchOne` commits the binding-abort revert through
    `checkpointReasonWorkBindingReverted`.
  - Failed `ResolveWait` rearms its deadline after its checkpoint commits.
  - `startDispatchedLocked` persists through `checkpointReasonWorkDispatched`
    without a separate `workBound` checkpoint.

##### `RT-SETTLE-005.a` Remove post-checkpoint terminal mutation

- Status: Implemented.
- Completion evidence:
  - Removed post-checkpoint `settlePendingFrameCancellationsLocked` in
    `R/runtime.go:executeOne` and removed the second post-checkpoint
    `waitResolvedLifecycle` mutation in `R/runtime.go:ResolveWait`.
  - `R/budget.go:waitForExecutionDeadline`, `R/coordination.go:dispatchEmit`,
    `R/retry.go:waitForRetry`, `R/runtime_resources.go:waitForResource`,
    `R/runtime_durability.go`, and `R/runtime.go` all converge before their
    sole checkpoint.

##### `RT-SETTLE-005.b` Use one terminal commit sequence

- Status: Implemented.
- Completion evidence:
  - `R/runtime.go:finishWorkLocked` and
    `R/runtime_durability.go:settleUndispatchedWorkLocked` settle attempts,
    effects, resources, waits, owner advancement, and completion waits, then
    converge through `finalizeExecutionSettlementLocked`.
  - `R/runtime_lifecycle.go:commitTerminalMutationLocked` executes the final
    pre-checkpoint convergence, checkpoint commit (`persistLocked`), post-commit
    ready-work launch, due-time arming, and event publication in order.
  - The listed binding-abort, failed-resolution timer, dispatch, and
    cleanup-expiry changes exist. Other authoritative-owner and terminal
    obligation deficiencies are `RT-OWNER-007` and `RT-CLEANUP-007`.

#### `WF-PREPARE-005` Freeze complete Workflow Model and Agent contribution meaning

- Status: Partial.
- Completion evidence:
  - `S/contributionruntime/prepared.go:Preparer` coordinates Skill-domain
    instruction rendering (`InstructionSkillRenderer`) and MCP-domain
    contribution capture (`MCPContributionPreparer`) before Workflow
    `ProgramSnapshot` creation.
  - `cmd/agentgo/wrapper_sessionv3_bootstrap.go` and
    `S/sessionv3runtime/binder.go` wire `skillruntime.V3Executor` and
    `mcpruntime` preparers into contribution preparation.
  - `S/sessionv3runtime/workflow_invocation.go`, `llm_plan.go`, and
    `tools_choices.go` consume captured semantic facts without late Skill
    rendering or MCP discovery reselection.
  - The single MCP choice loop and optional MCP skip branch exist.
  - Remaining effective invocation preparation is specified by subtasks
    `WF-PREPARE-005.c` through `WF-PREPARE-005.f` below.
  - Shared preparation is not proof of owning-domain separation. Complete
    native contribution ownership under `ARC-CLEAN-006`.

##### `WF-PREPARE-005.a` Capture instruction Skill output

- Status: Implemented.
- Completion evidence:
  - `S/contributionruntime/prepared.go:prepareAgent` renders instruction-mode
    Skills through `InstructionSkillRenderer.Instruction` and stores the
    immutable text in `WorkflowPreparedSkill.InstructionContent`.
  - `S/sessionv3runtime/workflow_invocation.go:applyPreparedContributionPlan`
    consumes `WorkflowPreparedSkill.InstructionContent` directly and rejects
    missing captured instruction text without calling `skillTools.Instruction`.

##### `WF-PREPARE-005.b` Capture Workflow MCP contribution meaning

- Status: Implemented.
- Completion evidence:
  - `R/workflow_target_facts.go:WorkflowPreparedMCP` and
    `R/plans.go:MCPDispatchPlan` retain `Prepared`, `DiscoveryDigest`,
    `ServerDisplay`, `Context`, and `PreparedTools` (`[]ToolDispatchPlan`).
  - `S/mcpruntime/workflow_preparation.go:prepareMCPContribution` captures
    MCP server instructions, resources, templates, prompts, discovery digest,
    policy, and provider Tool choices through the shared MCP-domain helpers in
    `S/mcpruntime/v3.go`.
  - `S/sessionv3runtime/llm_plan.go` and `tools_choices.go` consume captured
    MCP context and `PreparedTools` directly without reselecting changed live
    MCP server state.
  - `collectMCPServerChoices` emits one choice per selected Tool. The broader
    call-default preparation gap is owned by `WF-PREPARE-005.c`; this captured
    MCP contribution mechanism must not be implemented again.

#### `WF-RETRY-005` Freeze retry call port inputs at the original boundary

- Status: Implemented.
- Completion evidence:
  - `R/workflow_retry.go:captureWorkflowRetryCall` resolves the target and
    call `PortPlan` values via `workflowCallPortPlans`, then freezes all input
    port values and cursors via `capturedWorkflowCallPortInputs` before retry
    child creation.
  - `R/workflow_program.go:call` propagates captured `PortInputs` onto the
    step copy passed to `startWorkflowChildCall`.
  - `R/retry_program.go:retryCaptureChildCall` preserves already-captured
    `PortInputs` and falls back to child frame `PortInputs` for `ChildCall`
    work items.

#### `ARC-DUP-005` Give lowered Workflow tree traversal one owner

- Status: Implemented.
- Completion evidence:
  - `R/workflow_structured.go:WalkWorkflowBlock` provides the single typed
    lowered Workflow tree traversal contract with `WorkflowBlockRole`,
    `WorkflowBlockNode`, `WorkflowStepNode`, `WorkflowTreeVisitor`, and
    `ErrStopWorkflowWalk`, with no Artifact Store or source-parsing dependency.
  - `R/ports_runtime.go:namespaceWorkflowBlockChannels`,
    `R/resource_namespace.go:namespaceWorkflowBlockResources`,
    `S/workflowruntime/declaration_paths.go:attachWorkflowDeclarationPaths`,
    and `S/workflowruntime/compiler_invocation.go:workflowCallAtDeclarationPath`
    all use `WalkWorkflowBlock`.
  - `S/workflowruntime/declaration_paths.go` consumes planning-owned
    `inventory.Blocks` directly instead of reconstructing nested block paths.

#### `RT-CLEANUP-006` Settle Saga effects and child Executions on cleanup deadline expiry

- Status: Implemented.
- Completion evidence:
  - `R/budget.go:expireExecutionCleanupLocked` retires pending and dispatching
    compensation records, expires owned child cleanup, and retains expiry causes.
  - `waitForCleanupDeadline` commits through `commitTerminalMutationLocked`.

#### `WF-PORT-006` Forward published nested Workflow output ports across independent started calls

- Status: Implemented.
- Completion evidence:
  - `R/ports_runtime.go:preparePublishedWorkflowPortBoundaryLocked` resolves
    the target and receiving owner through `resolvePortChannelTarget`.
  - `ChannelForward` retains receiving ownership for cross-Execution delivery.

#### `WF-HANDOFF-006` Retain multi-turn handoff conversation history and continuation consumption

- Status: Implemented.
- Completion evidence:
  - `R/program.go:startQueuedAgentLLM` retains completed input IDs and passes
    pending Tool IDs through `startAgentLLM` for durable consumption marking.
  - `S/sessionv3runtime/inputs.go:llmInputActivitySequence` selects isolated
    handoff input and frame-owned assistant history. Staged active-owner input
    is a different missing behavior owned by `V3-CONTINUE-007`.

#### `V3-TOOL-006` Complete normal conversation Tool error settlement and manual Tool guards

- Status: Partial.
- Completion evidence:
  - Runtime read-only failure settlement, normal Go and Skill error-output
    branches, manual current-batch controls, and handoff Tool routing exist.
- Functional subtasks:
  - `V3-TOOL-006.a`, Status: Not implemented. Complete consumable MCP conversation error output for returned error responses and terminated invocation failures. Preserve available rich output and provider call identity.
  - `V3-TOOL-006.b`, Status: Partial. Separate completed Tool observation from business-effect certainty across Go, MCP, and Skill. An error alone must not manufacture confirmed non-application. Preserve mutating Workflow uncertainty, reconciliation, and retry eligibility from actual domain evidence.
  - `V3-TOOL-006.c`, Status: Partial. Make confirmed conversation error observations available to explicit continuation without automatically retrying uncertain effects or automatically continuing unsuccessful batches.
  - `V3-TOOL-006.d`, Status: Not implemented. Preserve supported sequential automatic Tool-batch execution and explicit requested-subset behavior. Runtime owns ordering; native domains must not add orchestration locks or schedulers.
- Completion evidence after implementation: Pending for each open subtask.

#### `V3-APP-006` Consume staged MCP App context on `SubmitMessage`

- Status: Implemented.
- Completion evidence:
  - `R/runtime.go:SubmitMessage` transfers staged App context into the created
    message Activity, clears it, and advances consumption when no draft remains.

#### `ARC-CLEAN-006` Consolidate owning-domain code and Runtime algorithms

- Status: Partial.
- Completion evidence: Shared pinning helpers, `ArtifactPin.Selection()`, projection builders, and lowered-tree traversal exist. The supplied compatibility wrapper and Store projection duplication remain.
- Functional subtasks:
  - `ARC-CLEAN-006.a`, Status: Not implemented. Remove the redundant Session attachment delegate and unused directory-limit field after callers use Attachment setup directly. Put immutable attachment-to-provider conversion and text-envelope construction in Attachment setup, not Session or Workflow orchestration.
  - `ARC-CLEAN-006.b`, Status: Partial. Agent and Workspace setup must own their contribution interpretation. `contributionruntime` composes neutral prepared facts and cross-domain precedence; it must not remain a second implementation of Agent or Workspace semantics.
  - `ARC-CLEAN-006.c`, Status: Partial. Model setup owns native Model-default adaptation and native completion interpretation. Session setup selects semantic messages and orchestrates domain results without owning another native Model grammar.
  - `ARC-CLEAN-006.d`, Status: Partial. Reuse Runtime conversation and Activity builders in Store v3. Consolidate repeated recursive error clones, retry/catch matching, expression evaluation, operation-input extraction, outcome-to-CEL projection, retention calculations, and equivalent snapshot-copy algorithms within their owning boundaries.
  - `ARC-CLEAN-006.e`, Status: Not implemented. Remove the consecutive duplicate provider-header clone in `S/sessionv3runtime/llm.go:bindLLM` and equivalent repeated execution preparation. Preserve distinct transport DTOs and exact provider-input capture where they serve different contracts.
  - `ARC-CLEAN-006.f`, Status: Partial. Establish one Workspace instruction injection route. Retain per-Text pins and provenance without injecting content again when the owning Workspace prompt already contains it. Inspect the Workspace prompt producer to settle the provisional payload-overlap finding.
  - `ARC-CLEAN-006.g`, Status: Partial. Use existing constants and typed enums for closed vocabularies, including remaining MCP policy and action-kind fields. Use lowercase implementation-owned error strings. Preserve user data, protocol identifiers, and domain payload meaning.
- Validation work remains excluded. Ownership consolidation must remove equivalent repeated checks in touched paths where ownership is already established, without creating new checking, strictness, reports, or a separate validation task.
- Completion evidence after implementation: Pending for each open subtask.

#### `WF-PREPARE-005.c` Prepare complete Model and provider-managed Tool call defaults

- Status: Not implemented.
- Source evidence: `S/workflowruntime/model_agent_adapter.go:PrepareWorkflowModelCallInvocation` receives but ignores `defaults`; Runtime subsequently replaces contribution selections from that incomplete prepared plan.
- Functional work:
  - Prepare Workspace, Workspace overrides, explicit clears, instruction Artifacts, Tool selections and arguments, Skills, active Skills, and MCP selections for Model calls through the common owning-domain preparation path.
  - Apply the same complete call-default preparation to the owner Model of a provider-managed Tool target.
  - Retain the selected target and complete effective contribution plan in `PreparedTargets`; consume it without replacing omitted facts with current Session defaults.
- Completion evidence after implementation: Pending.

#### `WF-PREPARE-005.d` Resolve effective exclusions, clears, and optional contributions before live preparation

- Status: Partial.
- Source evidence: `S/contributionruntime/prepared.go:Prepare` restores `inheritedWorkspace` after explicit clearing; `prepareAgent` pins occurrences before collection exclusions and does not retain optional Tool or runtime Skill preparation semantics through final domain preparation.
- Functional work:
  - Preserve explicit Workspace clearing without fallback to the Agent Workspace.
  - Apply collection replacement, removals, and selected contribution precedence before preparing capabilities that the effective invocation excludes.
  - Make `InstructionArtifacts` replacement suppress the applicable inherited instruction Text contributions.
  - Retain required versus optional contribution meaning until owning-domain preparation finishes. Skip unavailable optional capabilities without losing required or explicitly selected capability failures.
- Completion evidence after implementation: Pending.

#### `WF-PREPARE-005.e` Preserve selected Model prompt policy and effective contribution provenance

- Status: Partial.
- Source evidence: `prepareAgent` assigns `IncludeModelSystemPrompt` for Model occurrences independently from final Model selection; replacement helpers can change contribution values without updating their effective owner or order.
- Functional work:
  - Apply Agent Model prompt inclusion only to the Model capability actually selected from that Agent.
  - Preserve explicit Session or invocation Model precedence and explicit system-prompt precedence.
  - Update owner, source key, and deterministic contribution order when a more-specific selection replaces an inherited contribution.
  - Preserve one authoritative Workspace prompt contribution as specified by `ARC-CLEAN-006.f`.
- Completion evidence after implementation: Pending.

#### `WF-PREPARE-005.f` Prepare effective call targets without repeated baseline preparation

- Status: Partial.
- Source evidence: Agent admission prepares baseline contributions, `prepareBoundTarget` prepares them again, and call-site preparation can prepare them again with different effective exclusions.
- Functional work:
  - Separate target identity and callable transport facts from complete effective call contribution preparation.
  - Prepare each applicable effective call-target plan once after its defaults and exclusions are known.
  - Prevent unused baseline contributions from failing a call that explicitly excludes them. Keep explicit root requirements independent.
- Completion evidence after implementation: Pending.

#### `RT-OWNER-007` Retain authoritative Session-owned mutation targets

- Status: Partial.
- Affected entries: `RT-SETTLE-003.a`, `RT-SETTLE-003.b`, and `RT-RECOVERY-003.b`.
- Source evidence: `SignalExecution`, `signalTriggerExecutionLocked`, restore wait resolution, Fork materialization, and reconciliation delivery retain objects across nested settlement that can drain triggers and append Session Executions.
- Functional subtasks:
  - `RT-OWNER-007.a`, Status: Partial. Capture durable identities before nested transitions and reacquire Execution, frame, WorkItem, Wait, and signal owners before subsequent mutations in every listed path.
  - `RT-OWNER-007.b`, Status: Partial. Iterate dependency-ready frame, Execution, and Wait identities rather than stale slice-backed objects during restore, signal delivery, and child materialization.
  - `RT-OWNER-007.c`, Status: Not implemented. Commit the dispatch-start failure transition back to `Ready` in `dispatchOne`; the existing binding-abort revert is already implemented and is not this task.
- Completion evidence after implementation: Pending for each open subtask.

#### `RT-CLEANUP-007` Complete cleanup classification and terminal obligations

- Status: Partial.
- Affected entries: `RT-CLEANUP-003.a`, `RT-CLEANUP-003.c`, and `RT-SETTLE-003.b`.
- Source evidence: `advanceFrameLocked` propagates cleanup to WorkItems only inside its child-frame branch; retained frame leases can prevent cancellation delivery; frame-derived and explicit Execution completion use different outstanding-obligation rules.
- Functional subtasks:
  - `RT-CLEANUP-007.a`, Status: Partial. Propagate owning cleanup classification to all emitted WorkItems and frames independently, including Agent, Model, retry, target-resolver, Machine, and other synthetic work.
  - `RT-CLEANUP-007.b`, Status: Partial. Permit eligible cleanup retries and retain narrowed cleanup operation or Attempt timeouts within the enclosing cleanup allowance, without inheriting an expired ordinary deadline.
  - `RT-CLEANUP-007.c`, Status: Not implemented. Distinguish resumable operation reconciliation from whole-Execution shutdown cleanup. A confirmed reconciliation or recovered inner scope must permit its authorized subsequent normal work when the Execution is not stopping.
  - `RT-CLEANUP-007.d`, Status: Partial. Release retained frame claims at the correct cancellation boundary so a frame does not wait on its own lease before it can receive cancellation and finalization.
  - `RT-CLEANUP-007.e`, Status: Partial. Use one semantic terminal-obligation decision for frame-derived and explicit Execution settlement, preserving required cleanup, compensation, completion delivery, and owned-child cancellation obligations.
  - `RT-CLEANUP-007.f`, Status: Not implemented. Retire compensation records that are no longer eligible after successful Saga completion, excluded compensation outcomes, or stop-policy completion. Retain confirmed original effects and primary failure or cancellation.
- Completion evidence after implementation: Pending for each open subtask. `RT-CLEANUP-006` remains implemented.

#### `RT-BUDGET-007` Complete stable structured aggregates and retained-record accounting

- Status: Partial.
- Affected entries: `RT-BUDGET-003.b` and `RT-BUDGET-003.d`.
- Source evidence: `deferFrameForkLocked` allocates a new scope per Fork; trigger signals and several direct Timeline append paths bypass semantic-owner reservations; `startCall` classifies its synthetic wrapper as Workflow nesting.
- Functional subtasks:
  - `RT-BUDGET-007.a`, Status: Partial. Retain one controlling aggregate for the complete structured operation across Foreach launch waves, Repeat iterations, retry children, and descendant Fork or Join work.
  - `RT-BUDGET-007.b`, Status: Partial. Preserve enclosing structured aggregate ownership when nested operations create narrower scopes. Charge each applicable aggregate once, without replacing or losing the enclosing scope.
  - `RT-BUDGET-007.c`, Status: Partial. Account retained trigger signals and message, action, wait, recovery, interrupted-assistant, Workflow, and trigger Activities through their durable semantic owners.
  - `RT-BUDGET-007.d`, Status: Not implemented. Count Workflow depth only at actual nested Workflow invocation boundaries, not started-call wrappers or other synthetic frames.
- Completion evidence after implementation: Pending for each open subtask. Cost metering remains excluded.

#### `RT-COMMAND-007` Persist and resume wait-resolution command intent

- Status: Partial.
- Affected entries: `RT-COMMAND-003` and `RT-RECOVERY-003`.
- Source evidence: `R/runtime.go:ResolveWait` persists a resolving Wait before recording the command receipt and final resolution data, then calls the domain resolver outside Runtime-owned asynchronous execution.
- Functional subtasks:
  - `RT-COMMAND-007.a`, Status: Not implemented. Persist the accepted resolution intent, actor, response, original command identity, and replay receipt before domain resolution starts, using the existing Session durable root.
  - `RT-COMMAND-007.b`, Status: Not implemented. Return the same accepted command meaning for duplicate resolution commands while work is pending and after restart, without inserting another command record or losing the requested resolution.
  - `RT-COMMAND-007.c`, Status: Partial. Resume domain resolution through the owning binding and settle the original Wait and WorkItem through the common committed lifecycle.
  - `RT-COMMAND-007.d`, Status: Not implemented. Make resolution execution subject to Runtime lifecycle cancellation and tracked shutdown so close cannot finish while an owned resolver can still mutate or checkpoint Session state.
- Completion evidence after implementation: Pending for each open subtask.

#### `WF-CONTROL-007` Preserve complete structured exits and captured started-call meaning

- Status: Partial.
- Affected entries: `WF-CONTROL-003.d`, `WF-OUTCOME-003.a`, and `WF-PREPARE-003`.
- Source evidence: Repeat, Machine, and Saga nonlocal exits bypass parts of normal state transfer; `finishWorkflowTerminal` omits causal IDs from rebuilt block results; `startCall` reuses frame deadline and start-output schema for internal invocation.
- Functional subtasks:
  - `WF-CONTROL-007.a`, Status: Partial. Transfer inherited or carried state and causal effect frontiers before Repeat break or Workflow return, Machine nonlocal exit, and Saga Workflow return. Preserve local-state isolation.
  - `WF-CONTROL-007.b`, Status: Partial. Retain causal effect IDs through catch, finally, and reconstructed child block results, including effects completed by handler and finalization blocks.
  - `WF-CONTROL-007.c`, Status: Partial. Record the enclosing structured step outcome before pending state is cleared by return, break, continue, cancellation, or Saga terminal handling.
  - `WF-CONTROL-007.d`, Status: Partial. Apply immediate, accepted-event, and due-timer transition priority from durable candidate facts before Machine selection. A buffered accepted event must not lose solely because its wait binds after a due timer.
  - `WF-CONTROL-007.e`, Status: Not implemented. Retain the captured call deadline through started-call wrappers and delayed child creation. Keep the start handle schema separate from the internal invoke output schema.
- Completion evidence after implementation: Pending for each open subtask.

#### `V3-CONTINUE-007` Consume staged input through the active conversation owner

- Status: Partial.
- Affected entry: `V3-CONTINUE-003.a`.
- Source evidence: `consumeContinuationLocked` excludes non-normal-conversation Executions even when `UpdateContinuationInput` routed input to their active handoff Agent.
- Functional subtasks:
  - `V3-CONTINUE-007.a`, Status: Not implemented. Consume staged text, attachments, defaults, budget, metadata, and App context through the active conversation frame in either a normal or handoff Execution.
  - `V3-CONTINUE-007.b`, Status: Partial. Preserve one next-message App-context consumption route when App provenance and active conversation ownership belong to different retained Executions. Keep original App provenance and prevent duplicate injection.
- Completion evidence after implementation: Pending for each open subtask.

#### `V3-CONFIG-007` Preserve explicit collection and nested Model patch intent

- Status: Partial.
- Affected entries: `V3-CONFIG-003.a`, `V3-CONFIG-003.b`, and `V3-CONFIG-003.c`.
- Source evidence: `resolveSessionInvocationDefaults` overwrites supplied collection modes; `MergeModelDefaultsOverride` replaces nested patches; Model setup ignores an empty stop-sequence array.
- Functional subtasks:
  - `V3-CONFIG-007.a`, Status: Not implemented. Preserve explicit collection policy when a collection is supplied. Use default replacement only when no explicit policy establishes another supported meaning.
  - `V3-CONFIG-007.b`, Status: Partial. Merge nested typed Model patches according to field presence, retain unspecified inherited fields, and apply explicit clears and empty collection replacement consistently.
  - `V3-CONFIG-007.c`, Status: Partial. Reuse the effective contribution precedence, selected Model prompt policy, and provenance behavior from `WF-PREPARE-005` for normal conversation and Workflow calls.
- Completion evidence after implementation: Pending for each open subtask.

#### `V3-ERROR-007` Retain owning-domain failure classification

- Status: Partial.
- Affected entries: `WF-RETRY-003.c` and `WF-OUTCOME-003`.
- Source evidence: common Model execution and Runtime bind or execute error paths retain only code and message even when an owning domain can supply typed error facts.
- Functional subtasks:
  - `V3-ERROR-007.a`, Status: Partial. Preserve supplied category, retryability, effect, details, and causes through Model, Agent, registered-binding, bind, execute, child settlement, retry, catch, and projection paths.
  - `V3-ERROR-007.b`, Status: Not implemented. Let the owning domain classify available native cancellation, timeout, transient, and authorization failures. Runtime must not infer domain meaning by matching human-readable error strings.
- Completion evidence after implementation: Pending for each open subtask.

#### `WF-EFFECT-007` Complete keyed-native registered-target preparation

- Status: Partial.
- Affected entry: `WF-EFFECT-003.d`.
- Source evidence: `compileWorkflowEffect` lowers a keyed target to a non-idempotent base while the Runtime keyed-capability representation expects a different base classification.
- Functional work:
  - Use one consistent typed representation for a keyed-only target, separate from unconditional idempotence.
  - Prepare admitted keyed-native Service and Activity targets, capture one key per logical operation, and deliver it only through `workbinding.Request.IdempotencyKey`.
  - Preserve that key through retry, wait, recovery, and reconciliation without adding metadata to another domain's native argument grammar.
- Completion evidence after implementation: Pending.

#### `V3-STORE-007` Repair FTS from authoritative Session documents

- Status: Partial.
- Affected entry: `V3-STORE-004.a`.
- Source evidence: `store_fts.go:ftsListener` logs failed FTS mutations; `store.go:rebuildDerivedLocked` repairs index files only.
- Functional work:
  - Route failed FTS upsert and deletion into the existing Store maintenance owner.
  - Reconcile live Session documents and tombstones into FTS so failed notifications do not leave search stale until restart.
  - Keep derived search repair separate from semantic checkpoint commitment, with no Runtime scheduler or Artifact Store dependency.
- Completion evidence after implementation: Pending.
