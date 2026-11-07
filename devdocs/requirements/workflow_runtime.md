# Workflow Runtime v3 and Session Runtime v3 HLD

## 1. Purpose

Session Runtime v3 is the durable backend for conversation, Agent loops, direct Model calls, Workflow execution, coordination, effects, and long-lived process behavior.

Workflow Runtime v3 is the typed Workflow compiler and ProgramFrame interpreter path inside Session Runtime v3.

The functional goal is:

```text
Session Runtime v3
    +
Workflow Runtime v3
    =
functional typed durable backend superset of normal Session v2 behavior
    +
full Workflow v1 execution behavior
```

Functional completion is backend completion.

Frontend adoption, migration, deprecation, and deletion are later aggregate-stage work.

## 2. Durable root

A Session is the only authoritative durable root for one coherent activity.

```text
Session
├── Timeline[] Activity
├── Executions[] Execution
│   ├── ProgramFrames[]
│   ├── WorkItems[]
│   ├── Attempts[]
│   ├── Waits[]
│   ├── Effects[]
│   ├── Cleanup
│   ├── Signals[]
│   ├── Channels[]
│   └── Outcome
├── TriggerRegistrations[]
├── ConversationOwnership[]
├── CommandRecords[]
└── Defaults and metadata
```

A Session document can contain:

- Conversation ProgramFrames.
- Agent-loop ProgramFrames.
- Single-model ProgramFrames.
- Workflow ProgramFrames.
- Independent child Executions.
- Trigger registrations.
- Durable event and channel records.
- Effect and compensation records.

A Session document must not contain:

- Provider clients.
- HTTP clients.
- MCP clients.
- Artifact service handles.
- Native file paths used as live handles.
- Browser state.
- Wails state.
- Credentials.
- OAuth tokens.
- Approval tokens.
- Callbacks.
- Cancellation functions.
- Timers.
- Compiled CEL programs.
- Compiled JSON Schemas.

## 3. Ownership boundaries

```text
Workflow declaration contract
        |
        v
Artifact admission and typed program compilation
        |
        v
Session Runtime v3 ProgramFrame interpreter
        |
        v
Setup-owned domain binders and domain runtimes
```

| Layer                     | Owner                                    | Responsibilities                                                                                               | Must not own                                              |
| ------------------------- | ---------------------------------------- | -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| Declaration syntax        | `llmartifactory-go/workflow/contract/v1` | Source grammar, static validation, CEL scope, target relationships, portable validity                          | Session state, Artifact mutation, external work           |
| Admission and compilation | `agentruntimesetup/workflowruntime`      | Artifact pins, typed program compilation, registered Service and Activity binding selection                    | Session mutation, WorkItem dispatch, provider invocation  |
| Durable orchestration     | `agentruntime-go/sessionv3`              | ProgramFrames, WorkItems, Attempts, Waits, timers, effects, recovery, cancellation, persistence-safe semantics | Artifact services, source parsing, provider configuration |
| Domain binding            | `agentruntimesetup` and domain runtimes  | Model, Tool, MCP, Skill, Workspace, attachment, Service, Activity, provider adaptation                         | Session persistence roots, Workflow source parsing        |

Session Runtime v3 must not import Artifact packages or setup packages.

Session Runtime v3 must not:

- Resolve Artifacts.
- Read Workspace files.
- Discover MCP tools.
- Render Skills.
- Configure providers.
- Invoke Tool implementations directly.
- Materialize local attachments.
- Resolve credentials.
- Persist domain-private values.

## 4. Typed runtime discipline

Runtime-owned semantics use concrete types.

This includes:

- Session identity.
- Execution identity.
- ProgramFrame identity.
- WorkItem identity.
- Attempt identity.
- Wait identity.
- Trigger identity.
- Artifact pin.
- Target selection.
- Binding version.
- Retry policy.
- Deadline.
- Effect contract.
- Effect state.
- Effect ledger record.
- Compensation state.
- Receipt.
- Error.
- Budget.
- Claim.
- Channel cursor.
- Port plan.
- Trigger occurrence.
- Machine state.
- Structured control-flow state.

`CanonicalValue` is used only at arbitrary JSON boundaries:

- Invocation input.
- Invocation output.
- JSON Schema.
- Literal expression values.
- Expression results.
- Schema-defined Workflow state.
- Generic Tool input and output.
- Generic metadata.
- Event data.
- Event correlation.
- Channel message data.
- Human response.
- Generic protocol projections.
- Generic debugger projections.

No durable semantic model uses `map[string]any` or `json.RawMessage` as its primary representation.

## 5. Persistence and Session Store v3

`internal/sessionstorev3` is the only authoritative V3 persistence store.

The authoritative document is one Session document per Session ID.

Derived data includes:

- Session list index.
- Execution list index.
- Full-text search records.
- Projection records.

Derived data can be rebuilt from the authoritative Session document.

Tombstones are durable deletion barriers.

A tombstone prevents a stale checkpoint from resurrecting a deleted Session.

A Session restore must preserve:

- Durable command records.
- Trigger registrations.
- Trigger occurrences.
- Execution hierarchy.
- ProgramFrame state.
- WorkItems.
- Attempts.
- Waits.
- Signals.
- Channels.
- Emit receipts.
- Effect records.
- Compensation records.
- Outcomes.

## 6. Durable external-work sequence

Every external WorkItem follows this sequence:

1. Persist typed WorkItem intent.
2. Resolve current dependencies through the owning setup domain.
3. Verify Artifact pins and current binding facts.
4. Persist semantic binding facts.
5. Persist an Attempt before external invocation.
6. Invoke external work.
7. Persist confirmed output, receipt, failure, cancellation fact, or uncertainty.
8. Never automatically replay uncertain non-idempotent work.
9. Use Runtime timers for durable time behavior.
10. Cancel timers and owned goroutines during `Runtime.Close`.

The sequence applies to:

- Tool WorkItems.
- MCP Tool WorkItems.
- Service WorkItems.
- Activity WorkItems.
- LLM WorkItems.
- Resolver WorkItems when a resolver performs external discovery.
- Reconciliation WorkItems.
- Compensation WorkItems.

A process stop after external dispatch but before a confirmed result produces uncertainty unless the durable effect contract proves safe replay behavior.

## 7. ProgramFrame interpreters

Built-in interpreters are versioned.

| Interpreter              | Purpose                                                             |
| ------------------------ | ------------------------------------------------------------------- |
| `builtin.conversation@1` | Conversation activity routing and visible message history.          |
| `builtin.agent-loop@1`   | Agent reasoning, Model calls, Tool calls, waits, Tool continuation. |
| `builtin.single-model@1` | One direct Model call.                                              |
| `artifact.workflow.v1@1` | Typed Workflow program execution.                                   |

A ProgramFrame contains:

- Interpreter identity.
- Interpreter version.
- Typed program snapshot.
- Typed program state.
- Parent relationship.
- Child frame identities.
- Child execution identities.
- WorkItem identities.
- Output.
- Error.
- Deadline.
- Fork boundary.
- Join boundaries.
- Effect-frame linkage where applicable.

A ProgramFrame interpreter must not persist:

- Compiled CEL.
- Compiled schemas.
- Clients.
- Callbacks.
- Credentials.
- Timers.
- Context cancellation functions.

## 8. Workflow compiler contract

Workflow source is not a ProgramFrame payload.

```text
Workflow declaration v1
        |
        v
Artifact pinning and target binding preparation
        |
        v
Typed runtime-neutral WorkflowProgram
        |
        v
Workflow ProgramFrame
```

The typed program contains:

- Typed blocks.
- Typed steps.
- Typed target descriptions.
- Typed target sets.
- Typed state specifications.
- Typed values.
- Typed retry policies.
- Typed effect plans.
- Typed saga plans.
- Typed waits.
- Typed channels.
- Typed subscriptions.
- Typed triggers.
- Typed ports.
- Typed machine state.
- Typed structured-control state.
- Stable Artifact pins where required.
- Stable Service and Activity binding IDs and versions.

The compiler must preserve every valid Workflow v1 field.

The compiler must not:

- Invent chat text from generic Workflow input.
- Omit valid fields.
- Convert a provider-managed Tool into local work.
- Fabricate target success.
- Defer a known unsupported source feature until after unrelated external work starts.

## 9. Callable target runtime mapping

| Workflow target | Runtime mapping                                           |
| --------------- | --------------------------------------------------------- |
| Agent           | Child Agent-loop ProgramFrame with typed Agent plan.      |
| Model           | Child single-model ProgramFrame with typed Model plan.    |
| Nested Workflow | Child Workflow ProgramFrame.                              |
| Tool Artifact   | Tool WorkItem.                                            |
| MCP Tool        | Tool WorkItem with persisted MCP binding facts.           |
| Service         | Service WorkItem with registered binding ID and version.  |
| Activity        | Activity WorkItem with registered binding ID and version. |

A target-set resolver maps to a typed resolver WorkItem when membership requires durable resolution.

A target-set selection persists:

- Target-set identity.
- Normalized member identities.
- Resolver time.
- Selected target identity.
- Input.
- Context.
- Effect key.
- Retry identity.

## 10. Model and Agent execution plans

A typed Model or Agent plan is complete before provider dispatch.

The plan can contain:

- Selected Agent.
- Selected Workspace.
- Selected Model.
- Model request patch.
- Instruction contributions.
- Text contributions.
- Tool choices.
- Provider-managed Tool choices.
- Skill contributions.
- MCP server contributions.
- MCP Tool choices.
- MCP resource contributions.
- MCP resource-template contributions.
- MCP prompt contributions.
- Attachment contributions.
- Message contributions.
- Channel contributions.
- Port contributions.
- Conversation ownership facts.
- Artifact pins.
- Semantic binding facts.

Session Runtime persists only durable semantic facts.

Runtime setup keeps private provider-ready state in memory until dispatch completes.

## 11. Tool behavior

### 11.1 Go Tools

Go Tools use:

- Typed Tool binding.
- Artifact pin verification.
- Function identity.
- Canonical arguments.
- Typed Tool output.
- Receipt or error facts.

### 11.2 Provider SDK Tools

Provider SDK Tools are provider-managed.

They remain Tool choices in the typed Model plan.

They do not produce local Tool WorkItems.

They do not produce fabricated local success.

### 11.3 MCP Tools

MCP Tool binding performs:

1. Current MCP server verification.
2. Current discovery lookup.
3. Provider Tool identity verification.
4. Digest verification.
5. Current policy evaluation.
6. Approval behavior.
7. Invocation.
8. Typed result adaptation.

MCP binding facts persist before invocation.

MCP approval tokens remain process-private.

### 11.4 Skill Tools

Skill Tools use an Execution-owned Skill session.

The durable model persists selected Skill Artifact identities and active state facts.

The durable model does not persist the Skill runtime session handle.

### 11.5 Service and Activity bindings

Service and Activity bindings are registered typed domain bindings.

A binding provides:

- Stable binding identity.
- Stable binding version.
- Effect contract.
- Typed bind behavior.
- Typed execution behavior.
- Typed wait behavior when applicable.

The Session core does not implement Service or Activity domain behavior.

## 12. Ports, channels, and messaging

A typed port plan contains:

- Port name.
- Port direction.
- Stream capability.
- Schema.
- Channel identity.
- Reader or writer access.
- Parent-child ownership.
- Input cursor where applicable.
- Output delivery semantics.

The runtime validates ports before external dispatch.

A port failure is a typed binding or validation failure.

A channel stores:

- Ordered messages.
- Message identities.
- Cursors.
- Retention facts.
- Capacity facts.
- Byte accounting.
- Consumption facts.
- Observe facts.
- Deduplication receipts.

Port delivery must preserve:

- Schema validity.
- Channel authorization.
- Parent-child ownership.
- Message ordering.
- Stream ordering.
- Durable replay semantics.

## 13. Structured execution

### 13.1 Sequential execution

The Workflow interpreter advances ordered steps only after the controlling predecessor settles.

### 13.2 Fork

A Fork WorkItem persists child ProgramFrame intents before child scheduling.

A Fork WorkItem is required whenever child scheduling crosses a durable boundary.

### 13.3 Join

A Join WorkItem persists child terminal aggregation before a parent interpreter receives child settlement.

A Join WorkItem is required whenever completion aggregation crosses a durable boundary.

### 13.4 Parallel and Foreach

Parallel and Foreach retain:

- Selected branch or item identities.
- Branch or item state.
- Child frame identities.
- Winner identities.
- Cancellation requests.
- Ordered outcomes.
- Uncertain outcomes.
- Merge behavior.

### 13.5 Repeat

Repeat retains:

- Current iteration.
- Carried state.
- Local iteration state.
- Last result.
- Exhaustion state.
- Deadline.
- Nested break and continue behavior.

### 13.6 Machine

A machine retains:

- Current state.
- Entry time.
- Transition generation.
- Carried state.
- Active subscriptions.
- Timer WorkItems.
- Event WorkItems.
- Selected transition.
- Current event.
- Child block identities.

## 14. Waits and timers

A Wait WorkItem persists:

- Wait identity.
- Owner WorkItem identity.
- Wait kind.
- Typed request.
- Deadline.
- Timer state.
- Subscription identity.
- Channel cursor.
- Completion handle.
- Assignee facts.
- Response schema.
- Resolution.
- Response.
- Error.

Runtime timers exist only in process memory.

A durable timer is reconstructed from its persisted semantic timer state after restore completes.

Retry timers and trigger timers follow the same rule.

## 15. Event and channel coordination

An event signal persists:

- Source.
- Identity.
- Event name.
- Data.
- Correlation.
- Time.
- Consumption facts.

Duplicate event identity is `(source, id)`.

An event can be consumed once per subscription identity.

A channel message persists:

- Channel.
- Sequence.
- Identity.
- Kind.
- Data.
- Correlation.
- Deduplication key.
- Consumption facts.

A channel observe operation does not consume a message.

A channel consume operation records the consuming wait identity.

## 16. Trigger runtime

A trigger registration persists:

- Trigger identity.
- Workflow Artifact pin.
- Typed Workflow Program snapshot.
- Trigger declaration identity.
- Trigger kind.
- Registration state.
- Next schedule occurrence.
- Last occurrence.
- Last calendar local occurrence.
- Accepted occurrences.
- Queue state.

A trigger occurrence persists:

- Trigger identity.
- External source.
- External source identity.
- Occurrence time.
- Acceptance time.
- Input.
- Instance key.
- Selected Execution.
- Queue position.
- State.
- Typed error.

The Runtime owns trigger timer rearming.

Trigger timers start only after restore completes.

A trigger does not create another persistence root.

## 17. Effects, reconciliation, compensation, and cleanup

### 17.1 Effect facts

A WorkItem retains:

- Effect contract.
- Effect state.
- Stable idempotency key.
- Effect plan.
- Reconciliation state.
- Cleanup marker.
- Receipt.
- Error.

An Attempt retains:

- Attempt identity.
- Stable idempotency key.
- Effect state.
- Receipt.
- Error.
- Deadline.

An EffectRecord retains:

- Original WorkItem identity.
- Original ProgramFrame identity.
- Saga identity.
- Confirmation order.
- Dependencies.
- Typed original operation facts.
- Compensation block.
- Compensation policy.
- Compensation WorkItem identity.
- Compensation ProgramFrame identity.
- Compensation receipt.
- Compensation error.

### 17.2 Reconciliation flow

```text
uncertain mutating WorkItem
        |
        v
persist Reconciliation WorkItem
        |
        v
start reconciliation child ProgramFrame
        |
        +-- applied ------> confirm original WorkItem
        |
        +-- not-applied --> retry only when policy permits
        |
        +-- unknown ------> typed recovery review wait
```

### 17.3 Compensation flow

```text
failed or cancelled Saga boundary
        |
        v
select confirmed compensatable effects
        |
        v
reverse dependency-aware order
        |
        v
persist Compensation WorkItem
        |
        v
start compensation child ProgramFrame
        |
        +-- success -----> record compensated
        |
        +-- failure -----> record typed cause
```

Unknown effects do not enter automatic compensation before reconciliation confirms application.

### 17.4 Cleanup

Cleanup is active after normal failure, timeout, or cancellation when finalization, reconciliation, compensation, or release work remains.

Cleanup work can dispatch after normal cancellation only when it is marked as cleanup work and remains inside cleanup allowance.

## 18. Cancellation and close

Cancellation requests:

- Stop queued normal WorkItems.
- Cancel active normal dispatch contexts.
- Cancel ordinary waits.
- Cancel attached children.
- Cancel eligible independent children.
- Preserve uncertain dispatched mutations.
- Permit bounded cleanup work.
- Preserve cancellation as the primary terminal meaning.

`Runtime.Close`:

1. Marks the Runtime closed.
2. Cancels lifecycle context.
3. Cancels WorkItem dispatch contexts.
4. Cancels wait timers.
5. Cancels retry timers.
6. Cancels trigger timers.
7. Waits for owned asynchronous work.
8. Closes process-private domain bindings.
9. Closes process-private Skill sessions.
10. Releases subscribers.

## 19. Recovery

Restore behavior:

1. Restore all Session documents.
2. Restore tombstones.
3. Restore durable command records.
4. Normalize private runtime handles out of durable state.
5. Mark dispatched work uncertain when no confirmed terminal result exists.
6. Rebind only replay-safe WorkItems through current dependency verification.
7. Preserve stable keyed idempotency keys.
8. Create recovery review waits for unresolved uncertainty.
9. Rearm waits, retries, and triggers after complete restore.
10. Resume only dispatchable WorkItems.

Recovery never automatically replays uncertain non-idempotent work.

Recovery never automatically compensates unknown effects.

## 20. Commands and projections

Commands are transport-neutral.

Transport adapters can expose commands but must not own runtime semantics.

Core command categories include:

- Session creation, rename, defaults update, and deletion.
- Message submission.
- Workflow start.
- Trigger event acceptance.
- Execution signal.
- WorkItem execution request.
- Execution continue.
- Wait resolution.
- Execution pause.
- Execution resume.
- Execution cancellation.

Core projections include:

- Conversation projection.
- Activity projection.
- Workflow projection.
- Execution projection.
- Session list.
- Execution list.
- Search projection.

Wails is an adapter.

## 21. Functional completion

Functional backend completion requires:

- Full typed Workflow v1 support.
- Durable external-work sequence for every external domain.
- Typed planning before Model and Agent dispatch.
- Typed callable target support.
- Typed port support.
- Typed coordination support.
- Typed trigger support.
- Typed effect, reconciliation, saga, compensation, and cleanup support.
- Typed resource and budget support.
- Session Runtime v3 backend parity for normal Session v2 behavior.
- Durable persistence, recovery, cancellation, uncertainty, and command idempotency.

Functional backend completion does not require:

- Frontend migration.
- Frontend V3 adoption.
- Session v2 deletion.
- Session Store v2 deletion.
- Offline migration.
- Legacy cleanup.

## 22. Implementation status

### Phase 1: architecture and durable core

- [x] Session v3 is the only V3 durable root.
- [x] Session Store v3 stores authoritative Session documents.
- [x] Index and FTS records are derived.
- [x] Tombstones are durable deletion barriers.
- [x] Durable command idempotency exists.
- [x] Runtime-owned timers have explicit close ownership.
- [x] Restore runs before timer rearming.
- [x] Private clients, credentials, callbacks, timers, and native handles remain non-durable.

### Phase 2: ProgramFrames and WorkItem lifecycle

- [x] Conversation ProgramFrames exist.
- [x] Agent-loop ProgramFrames exist.
- [x] Single-model ProgramFrames exist.
- [x] Workflow ProgramFrames exist.
- [x] Typed WorkItems, Attempts, Waits, receipts, errors, and outcomes exist.
- [x] Typed external-work bind, dispatch, and settlement sequence exists.
- [x] Typed retry schedule state exists.
- [x] Typed timer wait state exists.
- [x] Typed recovery review waits exist.
- [x] WorkItem-level resource acquisition and release exists.
- [x] WorkItem-level aggregate budget accounting exists, including durable step reservation and typed pre-dispatch cost-meter failure.

### Phase 3: callable targets

- [x] Tool Artifact targets map to Tool WorkItems.
- [x] MCP Tool targets map to Tool WorkItems with discovery binding.
- [x] Service targets map to Service WorkItems.
- [x] Activity targets map to Activity WorkItems.
- [x] Nested Workflow targets map to child ProgramFrames.
- [x] Agent targets map to child Agent-loop ProgramFrames.
- [x] Model targets map to child single-model ProgramFrames.
- [x] Static target sets persist member selection.
- [x] Resolver-backed target sets persist resolver WorkItems and membership.
- [x] Provider SDK Tools remain Model-plan-only capabilities.
- [x] Callable interface limits propagate to direct WorkItems and child ProgramFrame boundaries.
- [x] Callable interface context declarations are enforced for every target kind. Unsupported Tool context transport fails during admission.
- [x] Callable interface port declarations are enforced for every target kind. Unsupported Tool input and streamed-port transport fails during admission.
- [x] Full receipt schema enforcement exists for direct WorkItems, child Agent/Model/Workflow ProgramFrames, and start-call scheduling.

### Phase 4: planning and domain integration

- [x] Model resolution is setup-owned.
- [x] Provider configuration is setup-owned.
- [x] Agent capability planning is setup-owned.
- [x] Workspace planning is setup-owned.
- [x] Text contribution planning is setup-owned.
- [x] Go Tool binding is setup-owned.
- [x] MCP discovery and policy binding are setup-owned.
- [x] Skill resolution and Skill session setup are setup-owned.
- [x] Service and Activity binding registration has a typed runtime boundary.
- [x] Typed attachment planning is setup-owned for V3.
- [x] Typed port planning is setup-owned for V3 Model, Agent, nested Workflow, Service, and Activity paths.
- [x] Tool-family port unavailability is resolved by setup admission, not by Tool, MCP, or Skill domain APIs.
- [x] Full plan persistence facts exist for every Model and Agent contribution kind.

### Phase 5: structured execution

- [x] Sequence, transform, set, scope, choose, return, fail, break, and continue exist.
- [x] Parallel completion modes exist.
- [x] Parallel error policies exist.
- [x] Foreach source ordering and concurrency state exist.
- [x] Repeat controls and carried state exist.
- [x] Machine lifecycle state exists.
- [x] Catch and finally behavior exists.
- [x] Fork WorkItems persist child scheduling boundaries.
- [x] Join WorkItems persist child aggregation boundaries.
- [x] Resource and budget behavior is enforced inside Parallel, Foreach, Repeat, Scope, Machine, and Saga through frame-held claims.
- [x] Cleanup behavior is enforced inside structured cancellation paths, including finalizers, reconciliation, and compensation.

### Phase 6: coordination and triggers

- [x] Timer waits exist.
- [x] Event waits exist.
- [x] Channel waits exist.
- [x] Completion waits exist.
- [x] Human waits exist.
- [x] Signals use durable source and identity facts.
- [x] Channel cursors and emit receipts exist.
- [x] Trigger registrations persist in Session documents.
- [x] Event trigger occurrences persist.
- [x] Interval schedules exist.
- [x] Five-field cron schedules exist.
- [x] Trigger overlap policies exist.
- [x] Missed-occurrence policies exist.
- [x] Subscription retention enforcement exists.
- [x] Subscription capacity enforcement exists.
- [x] Subscription byte-limit enforcement exists.
- [x] Full human response authorization exists.
- [x] Trigger administration commands exist.

### Phase 7: effects and cleanup

- [x] Effect kinds exist.
- [x] Stable keyed idempotency keys exist.
- [x] Effect state exists in WorkItems, Attempts, receipts, and errors.
- [x] Effect operation facts are typed.
- [x] Reconciliation WorkItem intent exists.
- [x] Reconciliation decision union exists.
- [x] Saga scope identity exists.
- [x] Confirmed effect ledger records exist.
- [x] Compensation WorkItem intent exists.
- [x] Cleanup state exists.
- [x] Reconciliation execution is complete for all effect-bearing target kinds.
- [x] Reconciliation result validation is complete.
- [x] Confirmed non-application retry is complete.
- [x] Unknown reconciliation recovery review is complete in every reconciliation path.
- [x] Reverse dependency-aware compensation is complete.
- [x] Compensation concurrency is complete.
- [x] Compensation `continue` behavior is complete.
- [x] Compensation `stop` behavior is complete.
- [x] Cleanup deadline expiry is complete.
- [x] Public projection of primary failure with compensation causes is complete.

### Phase 8: normal Session capability

- [x] Direct Model chat exists.
- [x] Agent-backed chat exists.
- [x] Provider streaming exists.
- [x] Protocol output projections exist.
- [x] Usage projections exist.
- [x] Debugger projections exist.
- [x] Tool result continuation exists.
- [x] Go Tool execution path exists.
- [x] Provider SDK Tool planning exists.
- [x] MCP Tool execution path exists.
- [x] Skill Tool execution path exists.
- [x] MCP resources, prompts, Apps, approvals, App Tool calls, and App resource reads have domain routes.
- [x] Skill session lifecycle has domain routes.
- [x] V3 attachment admission exists.
- [x] V3 replay-safe attachment materialization exists.
- [x] V3 historical attachment protection exists.
- [x] V3 provider attachment adaptation exists.
- [x] V3 Service binding composition is complete.
- [x] V3 Activity binding composition is complete.
- [x] V3 Tool streamed output ports emit durable typed output records for Go Tool, MCP Tool, and Skill Tool routes.
- [ ] Go Tool, MCP Tool, and Skill Tool input-port adaptation is unavailable because their native invocation contracts have no Workflow input transport.
- [ ] Go Tool, MCP Tool, and Skill Tool streamed-port adaptation is unavailable because their native invocation contracts expose no typed progress stream.
- [x] Go Tool, MCP Tool, and Skill Tool calls with unsupported Workflow context, input-port, or streamed-port declarations fail before external work begins.
- [x] V2 Tool contracts remain unchanged and do not depend on Session Runtime v3 or Workflow Runtime v3.
- [ ] V3 command and projection coverage is complete for all normal Session v2 behavior.

### Phase 9: functional completion gate

- [ ] Every valid Workflow v1 field has typed behavior, language-invalidity rejection, or pre-dispatch unavailable-binding failure.
- [ ] No valid Workflow v1 field is silently omitted.
- [ ] No valid Workflow v1 feature remains permanently unsupported because implementation is absent.
- [ ] All external mutating work preserves uncertainty semantics through restart.
- [ ] All typed WorkItems preserve cancellation, retry, deadline, panic, recovery, reconciliation, and compensation semantics.
- [ ] V3 is a functional backend superset of normal Session v2 behavior.
- [ ] Functional backend completion is reached.

### Phase 10: aggregate stage after functional completion

- [ ] Frontend V3 adoption.
- [ ] Frontend command migration.
- [ ] Offline Session v2 to Session v3 migration.
- [ ] Session v2 deprecation.
- [ ] Session Store v2 deprecation.
- [ ] Legacy path deletion.
- [ ] Aggregate cleanup.
