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

| Interpreter | Purpose |
| --- | --- |
| `builtin.conversation@1` | Ordinary Session chat and visible message history. |
| `builtin.agent-loop@1` | LLM, Tool, approval, Tool result, and continuation loop. |
| `builtin.single-model@1` | One direct Model call. |
| `artifact.workflow.v1@1` | Artifact-backed Workflow execution. |

The Workflow interpreter owns its bounded canonical ProgramFrame payload and
state. It maps Workflow calls as follows:

| Workflow target or operation | Session runtime mapping |
| --- | --- |
| Agent call | Child built-in Agent-loop ProgramFrame. |
| Model call | Child built-in single-model ProgramFrame. |
| Workflow call | Child Artifact-backed Workflow ProgramFrame. |
| Tool call | Tool WorkItem. |
| Service call | Service WorkItem bound by registered Service binding. |
| Activity call | Activity WorkItem bound by registered Activity binding. |
| Transform | Transform WorkItem. |
| Choice and loop guard | Condition WorkItem. |
| Set | StateUpdate WorkItem. |
| Parallel and foreach fan-out | Fork WorkItem plus child scopes. |
| Parallel and foreach completion | Join WorkItem. |
| Human, approval, timer, event, channel, child wait | Wait WorkItem. |
| Emit | Emit WorkItem. |
| Saga recovery | Compensation WorkItem. |

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

Implemented backend runtime components:

- Session Runtime v3 durable model.
- Session Store v3 one-document persistence, restore, index, FTS, search,
  export, tombstones, and derived-data rebuild.
- Conversation, Activity, and Workflow projections.
- Built-in conversation, Agent-loop, and single-model ProgramFrames.
- Workflow Artifact admission through `workflow.Service.ResolvePlan`.
- Artifact pin capture and nested Workflow ProgramFrame snapshots.
- Workflow interpreter and typed WorkItem mappings.
- Service and Activity binding registry and runtime binder composition.
- Tool approval waits, cancellation, recovery, uncertain effect handling, and
  Session command idempotency.
- Wails-independent backend command service and Wails delegation.
- Runtime setup that preserves `FetchCompletion` as provider execution.

The offline migration is separate tooling. It translates v2 transcript and
Run data to v3 Timeline and Executions, removes private runtime data, changes
in-flight external work to uncertain, and rebuilds derived Session Store data.
