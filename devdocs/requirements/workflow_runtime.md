# Workflow Runtime HLD

## Durable root

A Session is the only durable root for one coherent activity.

```text
Session
├── Timeline[] Activity
├── Executions[] Execution
│   ├── ProgramFrames[]
│   ├── WorkItems[]
│   ├── Attempts[]
│   ├── Waits[]
│   ├── Controls
│   └── Outcome
└── Defaults and metadata
```

Chat, Agent loops, and Workflow execution are ProgramFrames inside the same
Session. They are not Session kinds, Execution kinds, or separate persistence
roots.

## Architecture boundary

Workflow execution has four separate ownership layers.

```text
Workflow declaration contract
        |
        v
Artifact admission and typed program compiler
        |
        v
Session v3 typed Workflow interpreter
        |
        v
Domain runtime binders and external execution
```

| Layer                                                  | Owner                                    | Must not own                                               |
| ------------------------------------------------------ | ---------------------------------------- | ---------------------------------------------------------- |
| Declaration syntax and static validation               | `llmartifactory-go/workflow/contract/v1` | Session state, Artifact mutation, external work            |
| Artifact admission, pins, and typed IR compilation     | `agentruntimesetup/workflowruntime`      | Session mutation, WorkItem dispatch, external execution    |
| ProgramFrame execution and durable Workflow state      | `agentruntime-go/sessionv3`              | Artifact imports, Artifact services, source locators       |
| Model, Tool, MCP, Skill, Service, and Activity binding | Domain runtime setup packages            | Session persistence roots and Workflow declaration parsing |

`agentruntime-go` must not import Artifact or LLM Artifactory packages.
`llmartifactory-go` must not import Agent Runtime packages. Runtime setup is
the explicit dependency bridge.

Workflow source is compiled into a typed runtime-neutral program. The runtime
program uses typed blocks, steps, targets, state, and value templates.
`CanonicalValue` is used only for generic JSON values and schemas. New runtime
semantic structures must not expose `json.RawMessage` or persist
`map[string]any` as their primary model.

## Typed boundary discipline

Session Runtime v3 uses concrete types whenever the runtime owns the
semantics. This includes:

- Session, Execution, ProgramFrame, WorkItem, Attempt, Wait, and Outcome.
- Artifact pins, selections, target kinds, binding IDs, binding versions, and
  semantic binding facts.
- Retry policy, deadline, effect contract, resource claim, budget, receipt,
  control-flow, and state-machine state.
- Typed Workflow blocks, steps, targets, values, state, outcomes, and
  WorkItem payload unions.

`CanonicalValue` is used only for arbitrary JSON that is intentionally
schema-defined or external:

- User input and arbitrary Workflow values.
- JSON Schema documents.
- CEL expression literals and results.
- Generic Tool arguments and external Tool output.
- Schema-defined state values.
- Event, channel, correlation, metadata, and human-response data.
- Generic ProgramSnapshot payload and ProgramState data envelopes.

`jsonutil` is a conversion utility, not a durable runtime model. A domain
binder decodes a generic canonical value once into a concrete request type at
its boundary. Session Runtime does not retain a decoded `map[string]any` as
durable orchestration state.

## Persistence

`internal/sessionstorev3` owns one Session document per Session ID. Index and
FTS records are derived documents and are rebuilt from the authoritative
Session document. Workflow runtime does not create a Workflow repository,
Workflow store, filesystem execution store, or standalone invocation record.

The durable model persists semantic facts only:

- Program snapshots and interpreter versions.
- Artifact identity, revision, and definition digest pins.
- Canonical semantic input and output values.
- WorkItem dependencies, state, retry history, idempotency keys, deadlines,
  waits, decisions, safe receipts, errors, and outcomes.
- Branch, loop, scope, compensation, and reconciliation state owned by the
  Workflow interpreter.

The durable model also records the interpreter version. A declaration language
version does not define a compiled ProgramFrame payload version. Shipped V2
payload compatibility remains unchanged. The current V3 Workflow payload is
development-only and can be replaced in place as `artifact.workflow.v1@1`.
Stable V3 payload compatibility requires a separate approved policy.

The durable model does not persist API keys, OAuth tokens, MCP approval
tokens, HTTP clients, MCP clients, function pointers, cancellation functions,
stream callbacks, debug configuration, native paths, compiled CEL programs,
or browser and Wails state.

## Runtime binding

A ready WorkItem follows this durable sequence:

1. Persist semantic WorkItem intent.
2. Resolve current runtime dependencies.
3. Verify current bindings against durable semantic pins.
4. Persist semantic binding facts.
5. Persist dispatched attempt state.
6. Dispatch external work.
7. Persist confirmed result, failure, receipt, or uncertainty.

A process stop after dispatch and before a confirmed outcome changes the
WorkItem to uncertain. Non-idempotent uncertain work is not automatically
repeated. Recovery creates a durable review wait.

## ProgramFrame interpreters

Built-in interpreters are versioned:

| Interpreter              | Purpose                                                  |
| ------------------------ | -------------------------------------------------------- |
| `builtin.conversation@1` | Ordinary Session chat and visible message history.       |
| `builtin.agent-loop@1`   | LLM, Tool, approval, Tool result, and continuation loop. |
| `builtin.single-model@1` | One direct Model call.                                   |
| `artifact.workflow.v1@1` | Current typed runtime-neutral Workflow program.          |

`artifact.workflow.v1@1` is development-only and is replaced in place by the
typed runtime-neutral Workflow payload. It is not an offline migration target.
The current Workflow interpreter owns its bounded typed ProgramFrame payload
and state. Artifact setup compiles Workflow source into this payload before
Session creates external work. It maps Workflow calls as follows:

| Workflow target or operation                       | Session runtime mapping                                 |
| -------------------------------------------------- | ------------------------------------------------------- |
| Agent call                                         | Child built-in Agent-loop ProgramFrame.                 |
| Model call                                         | Child built-in single-model ProgramFrame.               |
| Workflow call                                      | Child Artifact-backed Workflow ProgramFrame.            |
| Tool call                                          | Tool WorkItem.                                          |
| Service call                                       | Service WorkItem bound by registered Service binding.   |
| Activity call                                      | Activity WorkItem bound by registered Activity binding. |
| Transform                                          | Transform WorkItem.                                     |
| Choice and loop guard                              | Condition WorkItem.                                     |
| Set                                                | StateUpdate WorkItem.                                   |
| Parallel and foreach fan-out                       | Fork WorkItem plus child scopes.                        |
| Parallel and foreach completion                    | Join WorkItem.                                          |
| Human, approval, timer, event, channel, child wait | Wait WorkItem.                                          |
| Emit                                               | Emit WorkItem.                                          |
| Saga recovery                                      | Compensation WorkItem.                                  |

At completion, every mapping in this table is implemented with typed durable
semantics. A valid Workflow v1 operation cannot remain a permanent
unsupported-operation admission failure because an implementation is missing.

Agent and Model calls use typed Agent-loop and single-model execution plans.
They do not serialize arbitrary Workflow JSON into a chat message. Workspace,
instruction, text, Tool, Skill, MCP, attachment, message, and port
contributions are compiled by their domain runtime setup packages before a
Model WorkItem dispatches.

## Completion rule

Workflow Runtime does not accept partially executable behavior.

For each Workflow operation, one interpreter version must do exactly one of
the following during admission:

1. Compile it to typed ProgramFrame and WorkItem behavior with durable
   recovery semantics.
2. Reject it as unsupported before any external WorkItem is created.

The runtime must never accept an operation and later treat it as a no-op,
silently omit it, or change its declared effect behavior.

During development, typed admission rejection protects users from partial
execution. It is not the target completion state. Completion requires all
valid Workflow v1 operations to compile and execute. Admission rejection then
applies only to invalid declarations, unavailable required bindings, current
policy or credential failures, and behavior outside Workflow v1.

No Workflow operation may fabricate a successful result. Provider, Tool, MCP,
Skill, Service, Activity, Model, Agent, or child Workflow failure must produce
a typed failure, cancellation, or uncertainty outcome.

The required durable sequence for external work is:

1. Persist WorkItem intent.
2. Resolve and verify runtime dependencies against durable pins.
3. Persist semantic binding facts.
4. Persist dispatched Attempt state.
5. Dispatch external work.
6. Persist confirmed result, failure, receipt, or uncertainty.

After a process stop, a dispatched non-idempotent WorkItem without a confirmed
outcome is uncertain. It is not automatically replayed. Unsupported Workflow
operations are rejected during admission with a typed unsupported-operation
error. They are never silently omitted and never fail only after unrelated
external work begins.

Compiled CEL programs, compiled JSON Schemas, authorization references,
clients, callbacks, timers, and cancellation functions remain process-private.
They must be bounded by Runtime or Execution lifetime and removed during close
or execution settlement.

### Current development implementation boundary

The implementation is not complete merely because it can persist an Execution
or run a subset of ProgramFrame interpreters. Completion remains the state
defined below: a functional Session v2 backend superset plus full Workflow v1
compiler and interpreter support.

The current V3 implementation boundary is:

- V2 Session Runtime and Session Store remain supported and unchanged.
- Frontend V3 adoption and offline migration are separate work.
- The current `artifact.workflow.v1@1` development payload remains in place;
  no payload or declaration-language version is incremented.
- `CanonicalValue` is used only for arbitrary JSON boundaries. Durable
  orchestration semantics remain concrete typed values.
- Model, Tool, MCP, Skill, attachment, Service, Activity, and Workflow
  behavior remains in its owning domain runtime package.
- Session V3 setup is an orchestration router, not a duplicate domain
  fallback runtime.
- Provider-managed SDK Tool Artifacts cannot produce local fabricated success.
  They require a typed Model execution plan or fail before external dispatch.

## Provider execution

`FetchCompletion` remains the provider-facing single-call boundary. Runtime
setup resolves provider configuration and credentials only when an LLM
WorkItem dispatches. The provider layer does not know whether the WorkItem
originated from chat, an Agent loop, or a Workflow.

## Implementation status

This document defines the required completed V3 backend state. Foundation work
does not satisfy completion by itself.

The V3 backend is complete only when all of the following are true:

- Session Runtime v3 is a functional backend superset of normal Session v2
  behavior.
- Every valid Workflow v1 operation, target mode, control-flow construct,
  coordination primitive, effect contract, and limit has typed compiler and
  interpreter support.
- All V3 external work follows durable intent, current dependency resolution,
  pin verification, semantic binding persistence, dispatch persistence, and
  confirmed-result-or-uncertainty persistence.
- Agent, Model, Workspace, Tool, MCP, Skill, attachment, Service, Activity,
  and Workflow behavior remains in its owning domain runtime package.
- V3 WorkItems preserve cancellation, retry, deadline, panic, recovery,
  uncertainty, reconciliation, and compensation semantics across restart.
- V3 Session commands and projections are available through the
  transport-neutral backend command service. Wails remains an adapter.
- V3 Store persistence, restore, tombstones, derived index rebuild, FTS
  rebuild, command idempotency, and recovery are verified.
- No valid Workflow v1 feature is silently omitted or left as a permanent
  unsupported runtime feature.

### Session Runtime v3 functional superset requirement

Session Runtime v3 must provide typed backend equivalents for all normal
Session v2 behavior:

- Direct Model chat and Agent-backed chat.
- Workspace and instruction contributions.
- Attachment preparation, snapshotting, replay, and provider adaptation.
- Model streaming, protocol outputs, usage, debugger data, and continuation.
- Go Tool, provider SDK Tool, MCP Tool, Skill Tool, Service, and Activity
  behavior.
- MCP resources, prompts, Apps, approvals, and approved App context.
- Waiting, cancellation, recovery, uncertainty, retry, and command
  idempotency behavior.

This functional-superset requirement does not require:

- Frontend Session v3 adoption.
- Offline v2-to-v3 migration.
- V2 Session Runtime or Session Store deletion.
- Automatic reading of V2 durable documents by V3.

V2 remains supported until separate migration, adoption, and deletion
decisions are approved.

### Required completion scope

Completion includes:

1. Typed compiler and interpreter integrity.
2. Sequential Workflow execution and all callable targets.
3. WorkItem lifecycle correctness.
4. Agent, Model, Workspace, Skill, MCP, attachment, text, and instruction
   execution plans.
5. Structured control flow.
6. Durable coordination.
7. Effects, reconciliation, compensation, claims, budgets, and limits.
8. Transport-neutral Session v3 command and projection service support.
9. Removal of duplicate V3 fallback logic from Session setup when domain
   runtimes own the behavior.

The offline migration remains separate tooling. It translates V2 transcript
and Run data to V3 Timeline and Executions, removes private runtime data,
changes in-flight external work to uncertain, and rebuilds derived Session
Store data. Frontend adoption and deletion of legacy runtime paths occur only
after V3 backend completion and migration approval.

### Implemented lifecycle boundary and remaining work

Timer waits persist their preparation time and firing time. An enclosing
operation deadline remains separate. Recovery rearms pending durable timers
only after the recovery source completes. A timer cannot be resolved manually.
Response schemas are checked before a wait response reaches its domain binder.

A failed checkpoint prevents new dispatch and event publication for that
Session. The live confirmed result remains available for a later checkpoint.
Restart uses the last committed Session document. Dispatched work without a
committed result remains uncertain and requires recovery review.

Cancellation before external invocation settles the unstarted attempt.
Cancellation after invocation does not convert an unconfirmed mutating effect
into proof of cancellation. Its WorkItem and Attempt retain uncertainty.

Workflow v3 coordination now uses the existing Session document as its only
durable root. The compiler produces typed event, channel, subscription, wait,
emit, and retry facts. The interpreter persists correlation, event source,
filters, channel cursors, consumed records, emit receipts, human request data,
retry policy, retry schedule, and attempt deadlines.

Pending event, channel, and completion waits inspect retained durable facts
when they are created. This prevents an already accepted signal, channel
message, or child completion from being lost because it arrived before the
Wait WorkItem bound. Runtime recovery rearms only durable timers and retry
schedules.

The V3 LLM binder now builds an execution plan through the owning Agent,
Workspace, Text, Skill, Tool, MCP, and Model domains before provider dispatch.
The Session core remains an orchestration owner. It persists only semantic
Artifact pins, selected identities, WorkItem state, attempts, waits, outputs,
and receipts. It does not resolve Artifacts, read Workspace files, discover
MCP capabilities, render Skills, or configure providers.

LLM WorkItems persist typed streamed text and thinking state, final
provider-neutral usage, canonical protocol-output projections, and canonical
debugger projections. Stream callbacks, clients, credentials, provider
configuration, and native attachment paths remain process-private. Selected
MCP resources, resource templates, prompts, and server instructions are
prepared by MCP Runtime before provider input construction.

Workflow `start` calls now use attached child ProgramFrames. The interpreter
persists the child identity before dispatch, returns a typed
`CompletionHandle`, and continues the parent frame. The existing completion
wait primitive resolves from the child terminal result. Attached child frames
also keep an Execution active after the parent root frame settles.

Workflow `cancel` now resolves a typed child-frame handle and requests
cancellation through the common ProgramFrame lifecycle. It does not mutate a
domain binder directly. Cancellation produces a typed Workflow step outcome.
Independent child lifetime creates a separate child Execution inside the same
Session document. The child persists parent and root Execution identities,
evaluated ownership, child budget, cancellation linkage, and a typed
Execution completion handle before it dispatches WorkItems.

Workflow `handoff` creates durable typed conversation ownership state before
the target Agent starts. It is not compiled as invoke. Child completion either
restores the prior owner or retains the handoff target according to the
declared completion behavior.

Parallel and Foreach support `all`, `first-completed`, `first-success`, and
`quorum` completion. They retain typed winner and branch or item outcome state,
support `remaining: cancel` and `remaining: wait`, and support `fail-fast` and
`collect` behavior. Foreach preserves source ordering and bounded concurrency.

Repeat supports count, while, until, timeout, carried state, local state,
exhaustion, break, continue, and ongoing `until-cancelled` execution.

Workflow blocks now compile ordered typed catch handlers and one typed finally
block. A terminal success, failure, timeout, or cancellation enters durable
terminal handling before the ProgramFrame becomes terminal. Matching catch
blocks receive typed `error` context. Finalizers receive typed `outcome`
context. Catch recovery output is schema-checked. A finalizer failure replaces
a success and is attached as a cause to a preceding failure or cancellation.

Workflow machines now compile into typed durable ProgramFrame state. The
runtime persists current machine state, carried state, entry time, active
subscription identities, active wait WorkItem identities, selected transition,
and transition generation. Entry, exit, transition blocks, immediate
conditions, event transitions, timer transitions, and final output use normal
typed child ProgramFrames and Wait WorkItems. Machine transition handling is
serialized by the owning ProgramFrame.

Resolver-backed target sets now resolve once at their first call boundary,
persist their accepted target names, and preserve the selected concrete target
identity in `WorkflowState` before external dispatch.

Full retries, complete structured-control policies, child ownership, event and
channel coordination, trigger scheduling, reconciliation, compensation,
resource accounting, and complete normal V2 backend capability remain
required. This partial coverage is not backend completion.

V3 backend completion still means a functional V2 backend superset plus full
Workflow v1 support. Session v2 and Session Store v2 remain preserved.
Frontend adoption and migration remain separate. CanonicalValue remains
limited to arbitrary JSON boundaries. Domain binders retain domain ownership,
and no Workflow, Trigger, Event, or Invocation persistence root is added.

## Implementation task list

- This list is the completion tracker for Session Runtime v3 and Workflow Runtime v3.

- Admission rejection during development is safety behavior, not completion.

- [x] `call.mode: invoke` uses direct typed execution.
- [x] `call.mode: start` with attached lifetime creates a child `ProgramFrame`.
- [x] `call.mode: start` with independent lifetime creates a child `Execution` in the existing Session document.
- [x] Independent child persists parent `ExecutionID`, root `ExecutionID`, ownership, budget, and cancel relationship.
- [x] Typed `CompletionHandle` supports `WorkItemID`, `FrameID`, and `ExecutionID`.
- [x] Completion waits resolve from child `WorkItem`, child frame, and child `Execution` terminal state.
- [x] `cancel` accepts child frame handles and child `Execution` handles.
- [x] Parent cancellation propagates only to independent children with `cancelWithParent`.
- [x] Agent handoff has typed conversation ownership state in the Session document.
- [x] `call.mode: handoff` creates distinct Agent ownership behavior and is not mapped to invoke.
- [x] Independent lifetime is not mapped to attached behavior.
- [x] No separate invocation store was added.
- [x] Tool Artifact target executes through a typed WorkItem.
- [x] MCP Tool target executes through a typed WorkItem and current MCP discovery.
- [x] Service target executes through a registered Service binding.
- [x] Activity target executes through a registered Activity binding.
- [x] Nested Workflow target executes as a child Workflow `ProgramFrame`.
- [x] Agent target executes as a child Agent-loop `ProgramFrame` with a typed plan.
- [x] Model target executes as a child single-model `ProgramFrame` with a typed plan.
- [x] Static target sets resolve and enforce allowed members.
- [x] Resolver-backed target sets resolve once at the call boundary and persist selected target identity.
- [ ] Agent, Workspace, Model, instruction, Tool, Skill, MCP, attachment, message, and port plans compile before Agent-loop or single-model dispatch.
- [x] Provider SDK Tool Artifacts execute only inside a typed Model plan and never produce local success.
- [x] `read` effect contract is implemented.
- [x] `idempotent` effect contract is implemented.
- [x] `keyed` effect contract is implemented with a stable idempotency key.
- [x] `non-idempotent` effect contract is implemented without automatic replay after uncertainty.
- [x] Stable idempotency keys persist in WorkItems and Attempts.
- [x] Idempotency keys propagate through the typed WorkItem dispatch request.
- [ ] Reconciliation WorkItems and recovery review are implemented.
- [ ] Saga effect tracking uses reverse compensation order.
- [ ] Compensation error policy and cleanup allowance are implemented.
- [x] Typed effect state persists in WorkItems, Attempts, errors, and receipts.
- [x] Block `catch` handlers are implemented.
- [x] Block `finally` behavior is implemented.
- [x] Machine initial state, entry, exit, transitions, timer transitions, event transitions, and final output are implemented.
- [x] Parallel `all` completion is implemented.
- [x] Parallel `first-completed` completion is implemented.
- [x] Parallel `first-success` completion is implemented.
- [x] Parallel `quorum` completion is implemented.
- [x] Parallel `remaining: cancel` and `remaining: wait` are implemented.
- [x] Parallel `fail-fast` and `collect` error behavior are implemented.
- [x] Foreach ordering, concurrency, all completion policies, cancellation, isolated state, and merge behavior are implemented.
- [x] Repeat `count`, `while`, `until`, timeout, carried state, local state, exhaustion, and `until-cancelled` lifecycle are implemented.
- [ ] Durable Fork and Join WorkItems exist where scheduling boundaries require them.
- [x] `break` and `continue` work across nested repeat scopes.
- [x] Event triggers are implemented through durable Session trigger registrations and transport-neutral event ingress.
- [x] Schedule triggers are implemented through Runtime-owned durable interval and five-field cron timers.
- [x] Trigger overlap and missed-occurrence policies are implemented for durable schedule registrations.
- [x] Trigger state remains in the existing Session v3 document.
- [ ] Event retention, event consumption, local channels, channel cursors, emit deduplication, human waits, completion waits, signals, and machine serialization are complete.
- [ ] Attachment admission, snapshots, replay-safe materialization, and provider adaptation use Attachment Runtime.
- [x] Agent and Workspace planning use their domain services.
- [x] Model defaults layering and typed request patches use Model Runtime.
- [ ] Go Tool, provider SDK Tool, MCP Tool, Skill Tool, Service, and Activity execution use their domain binders.
- [ ] MCP resources, prompts, Apps, approvals, App Tool calls, resource reads, and approved App context use MCP Runtime.
- [ ] Skill sessions, active Skill lifecycle, Tool invocation, and resource handling use Skill Runtime.
- [x] Provider streaming, protocol outputs, usage, debugger projections, Tool result continuation, cancellation, retries, waits, recovery, uncertainty, and command idempotency are implemented for current typed V3 paths.
- [ ] Session V3 setup retains only typed routing, semantic binding adaptation, and lifecycle ownership.
- [x] Session v2 behavior is unchanged and Session v2 does not depend on Session v3.
- [x] Workflow HLDs are updated truthfully when functional behavior changes.
