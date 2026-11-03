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
version does not define a compiled ProgramFrame payload version. Existing
interpreter payloads remain readable until migration is explicitly authorized.

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
| `artifact.workflow.v1@1` | Legacy early Workflow payload compatibility only.        |
| `artifact.workflow.v1@2` | Current typed runtime-neutral Workflow program.          |

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

## Completion rule

Workflow Runtime does not accept partially executable behavior.

For each Workflow operation, one interpreter version must do exactly one of
the following during admission:

1. Compile it to typed ProgramFrame and WorkItem behavior with durable
   recovery semantics.
2. Reject it as unsupported before any external WorkItem is created.

The runtime must never accept an operation and later treat it as a no-op,
silently omit it, or change its declared effect behavior.

The required durable sequence for external work is:

1. Persist WorkItem intent.
2. Resolve and verify runtime dependencies against durable pins.
3. Persist semantic binding facts.
4. Persist dispatched Attempt state.
5. Dispatch external work.
6. Persist confirmed result, failure, receipt, or uncertainty.

After a process stop, a dispatched non-idempotent WorkItem without a confirmed
outcome is uncertain. It is not automatically replayed.

Compiled CEL programs, compiled JSON Schemas, authorization references,
clients, callbacks, timers, and cancellation functions remain process-private.
They must be bounded by Runtime or Execution lifetime and removed during close
or execution settlement.

## Provider execution

`FetchCompletion` remains the provider-facing single-call boundary. Runtime
setup resolves provider configuration and credentials only when an LLM
WorkItem dispatches. The provider layer does not know whether the WorkItem
originated from chat, an Agent loop, or a Workflow.

## Backend consumers

`internal/sessioncommand.Service` is the transport-neutral backend command and
projection API. Wails is an adapter only. CLI, VS Code, HTTP, automation, and
test clients use the same command and projection service.

## Implementation status

Implemented foundation:

- Session Runtime v3 durable model.
- Session Store v3 one-document persistence, restore, index, FTS, search,
  export, tombstones, and derived-data rebuild.
- Conversation, Activity, and Workflow projections.
- Built-in conversation, Agent-loop, and single-model ProgramFrames.
- Workflow declaration validation and Artifact planning through
  `workflow.Service.ResolvePlan`.
- Workflow Artifact pinning, nested Workflow admission, and Service/Activity
  binding registry foundation.
- Service and Activity binding registry and runtime binder composition.
- Tool approval waits, cancellation, recovery, uncertain effect handling, and
  Session command idempotency.
- Wails-independent backend command service and Wails delegation.
- Runtime setup that preserves `FetchCompletion` as provider execution.

Workflow Runtime is not complete until all Workflow v1 operations have typed
compiler and interpreter support. Current completion work is ordered as:

1. Typed runtime-neutral Workflow program compiler and interpreter versioning.
2. Sequential operations, typed Tool/MCP Tool targets, Service, Activity, and
   nested Workflow execution.
3. Runtime lifecycle correctness: retry, timeout, cancellation, panic,
   uncertainty, and recovery.
4. Agent, Model, Workspace, Skill, and MCP preparation plans.
5. Structured control flow: scope, choose, repeat, parallel, foreach, fork,
   join, break, and continue.
6. Durable coordination: waits, events, channels, signals, machines, and
   triggers.
7. Effects, reconciliation, compensation, claims, and limits.

The offline migration remains separate tooling. It translates v2 transcript
and Run data to v3 Timeline and Executions, removes private runtime data,
changes in-flight external work to uncertain, and rebuilds derived Session
Store data. Frontend adoption and deletion of legacy runtime paths occur only
after runtime completion and migration.
