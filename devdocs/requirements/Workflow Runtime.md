# Workflow Runtime HLD

> A Session is a durable timeline of related activities plus the execution state needed to continue those activities safely.

A Session is not only a timeline. A timeline explains what happened. Execution state explains what is still running, waiting, retryable, cancelled, or uncertain.

A chat turn, an Agent Tool loop, and a Workflow invocation differ in orchestration only. They use the same Session, execution records, LLM calls, Tool calls, waits, outcomes, and persistence boundary.

## 1. Core model

The Session is the only durable activity root.

```text
Session
├── Timeline
│   ├── user message activities
│   ├── assistant message activities
│   ├── action summaries
│   ├── approval and wait activities
│   └── workflow progress activities
│
├── Executions
│   ├── orchestration frames
│   ├── work items
│   ├── attempts
│   ├── waits
│   ├── outputs
│   └── recovery state
│
└── Session defaults and presentation metadata
```

The Session answers two different questions:

| Question                                              | Owned data      |
| ----------------------------------------------------- | --------------- |
| What happened in this coherent activity?              | Timeline        |
| What is still active, and how can it safely continue? | Execution state |

A timeline item is an activity record. It can be a message, action summary, wait, workflow phase, or system outcome.

An execution work item is not necessarily visible in the timeline. For example, a retry backoff calculation, a branch guard evaluation, or a provider binding resolution is execution state but is usually not user-facing activity.

## 2. Activity versus work item

These are different concepts.

### 2.1 Activity

An Activity is a durable timeline record.

It is append-only from the consumer perspective and has:

- Activity ID.
- Timestamp.
- Visibility.
- Session association.
- Optional execution association.
- Optional parent activity.
- Typed payload.
- Human-readable summary.
- Optional structured data safe for consumers.

Activity kinds include:

| Activity kind | Example                                       |
| ------------- | --------------------------------------------- |
| `message`     | User asks a question.                         |
| `message`     | Assistant returns an answer.                  |
| `action`      | Tool changed a file.                          |
| `action`      | MCP Tool returned a result.                   |
| `workflow`    | Review branch completed.                      |
| `wait`        | Approval is required.                         |
| `wait`        | Approval was granted.                         |
| `state`       | Workflow reached a meaningful milestone.      |
| `outcome`     | Activity completed, failed, or was cancelled. |

A normal chat transcript becomes a filtered view of message Activities.

A Workflow UI becomes a filtered view of workflow, action, wait, and outcome Activities.

A CLI or VS Code extension can consume the same Activity timeline without Wails-specific behavior.

### 2.2 Work item

A WorkItem is a durable execution node.

It has:

- Work item ID.
- Parent frame ID.
- Program path.
- Dependency IDs.
- Logical operation key.
- Work type.
- State.
- Attempt history.
- Semantic input.
- Semantic output.
- Sanitized error.
- Safe external receipt, when available.
- Visibility policy.

Work types include:

| Work type      | Meaning                                                             |
| -------------- | ------------------------------------------------------------------- |
| `llm`          | One provider-neutral LLM request.                                   |
| `tool`         | Go Tool, MCP Tool, Skill Tool, or provider-managed Tool action.     |
| `service`      | Trusted application service call.                                   |
| `activity`     | Trusted local or remote activity call.                              |
| `transform`    | Deterministic value construction.                                   |
| `condition`    | Durable branch selection.                                           |
| `stateUpdate`  | Atomic scope-state update.                                          |
| `fork`         | Start parallel child scopes.                                        |
| `join`         | Evaluate child completion policy.                                   |
| `wait`         | Timer, approval, event, child completion, recovery, or human input. |
| `emit`         | Local channel append or external event publication.                 |
| `compensation` | Declared compensating action.                                       |

A WorkItem is the unit that the scheduler can dispatch or resume.

A WorkItem can emit zero, one, or many Activities.

An Activity can be associated with zero, one, or many WorkItems.

## 3. Timeline and execution relation

The new Session model does not replace the current persistence model with an unrelated event store.

It evolves the current shape.

Current conceptual shape:

```text
Session
├── TranscriptMessage[]
├── Run[]
│   ├── ModelOperation[]
│   ├── ToolCall[]
│   ├── ToolAttempt[]
│   ├── ToolResult[]
│   └── WaitingRequest[]
└── Session configuration
```

New conceptual shape:

```text
Session
├── Activity[]
├── Execution[]
│   ├── ProgramFrame[]
│   ├── WorkItem[]
│   ├── Attempt[]
│   ├── Wait[]
│   └── Outcome[]
└── Session defaults and presentation metadata
```

The important preservation is:

| Current persisted concept   | New persisted concept                     |
| --------------------------- | ----------------------------------------- |
| `TranscriptMessage`         | `Activity` with `kind=message`            |
| `Run`                       | `Execution`                               |
| `ModelOperation`            | `WorkItem` with `type=llm`                |
| `ToolCall`                  | `WorkItem` with `type=tool`               |
| `ToolAttempt`               | Work item attempt                         |
| `ToolResult`                | Work item outcome                         |
| `WaitingRequest`            | `WorkItem` with `type=wait`               |
| `Run.Driver`                | Program frame interpreter state           |
| `ContinuationInput`         | Session input Activity plus program event |
| Assistant transcript output | Message Activity plus LLM outcome         |

This is a breaking structural migration, but it preserves the meaning of existing records.

## 4. What a Session saves

A Session saves durable semantic facts.

### 4.1 Persisted Session facts

The Session persists:

- Activity timeline.
- Program snapshots.
- Program frame state.
- Work item graph.
- Branch selections.
- Loop iteration state.
- Scope state.
- LLM semantic requests.
- Tool semantic requests.
- Selected Artifact references.
- Selected Artifact revisions and definition digests.
- Service and activity registration identities and revisions.
- Inputs and outputs.
- Wait requests and resolutions.
- Absolute logical deadlines.
- Retry attempt history.
- Idempotency keys.
- Safe external receipts.
- Sanitized errors.
- Cancellation and recovery state.
- Visible and audit-safe summaries.

These are required to answer:

- What happened?
- What is still active?
- What may safely run next?
- What must not run again?
- What needs user or operator review?

### 4.2 Not persisted

The Session must not persist process-local runtime binding data.

It must not persist:

- API keys.
- OAuth tokens.
- MCP approval tokens.
- Provider headers containing secrets.
- Provider client objects.
- MCP client objects.
- Go function pointers.
- Service client instances.
- Open streams.
- HTTP request contexts.
- Cancellation functions.
- Stream callbacks.
- Debug configuration.
- Logging configuration.
- Worker pool state.
- Native paths.
- Compiled CEL programs.
- Cached provider capability resolver instances.
- Browser or Wails state.

These values are runtime bindings. They are recreated when a durable WorkItem becomes eligible for dispatch.

## 5. Caller-time input versus semantic input

Not every caller-supplied value has the same persistence rule.

| Value                                          | Persist? | Reason                             |
| ---------------------------------------------- | -------: | ---------------------------------- |
| User prompt                                    |      Yes | It is Session activity input.      |
| Selected Agent                                 |      Yes | It changes semantic execution.     |
| Selected Model                                 |      Yes | It changes semantic execution.     |
| Temperature override accepted for one LLM call |      Yes | It changes output semantics.       |
| Tool policy override                           |      Yes | It changes safety semantics.       |
| Workflow input                                 |      Yes | It changes execution semantics.    |
| Branch selection                               |      Yes | It changes continuation semantics. |
| Approval decision                              |      Yes | It changes execution semantics.    |
| Provider request timeout                       |       No | Transport policy only.             |
| Stream flush settings                          |       No | Transport policy only.             |
| Wails event subscription                       |       No | Consumer transport only.           |
| Debug logging                                  |       No | Diagnostic behavior only.          |
| API key                                        |       No | Credential.                        |
| Open MCP connection                            |       No | Process-local resource.            |

The rule is:

> If a value changes what the Session means or what it may safely do next, persist it. If it only changes how the current process dispatches, streams, observes, or transports work, do not persist it.

## 6. Program frames

A ProgramFrame is a durable scope that interprets orchestration.

It has:

- Frame ID.
- Parent frame ID.
- Program snapshot.
- Local state.
- Visible scope path.
- Current continuation state.
- Child frame references.
- Child work-item references.
- Completion status.
- Error and output.

There is no `Session.Kind`.

There is no `Run.Kind = chat|workflow`.

There is only a ProgramFrame whose program source may be:

| Program source                   | Meaning                        |
| -------------------------------- | ------------------------------ |
| Built-in conversation program    | Normal chat behavior.          |
| Built-in Agent-loop program      | LLM, Tool, continuation loop.  |
| Built-in single-model program    | One direct Model request.      |
| Workflow Artifact program        | Authored Workflow declaration. |
| Nested Workflow Artifact program | Child Workflow frame.          |

A simple chat has a ProgramFrame. It is just small.

An Agent loop has a ProgramFrame. It loops over LLM and Tool work items.

A Workflow has a ProgramFrame. It can create scopes, loops, branches, waits, and child frames.

## 7. Simple chat under the unified model

A simple user message creates a message Activity.

The built-in conversation program receives that Activity and creates an LLM WorkItem.

```text
User message Activity
  -> conversation program frame
  -> LLM WorkItem
  -> assistant message Activity
```

If the LLM returns no local Tool calls, the frame reaches a stable waiting state for another user message.

This preserves the current chat behavior.

`FetchCompletion` remains unchanged:

```text
Session scheduler
  -> ready LLM WorkItem
  -> runtime setup prepares provider request
  -> inference.FetchCompletion
  -> durable LLM WorkItem outcome
  -> Activity projection
```

The provider layer does not need to know whether the LLM WorkItem came from chat, an Agent loop, or a Workflow.

## 8. Agent loop under the unified model

An Agent is a reusable LLM interaction configuration.

An Agent execution is not a different Session type.

The built-in Agent-loop program does this:

```text
LLM WorkItem
  -> model output
  -> zero or more Tool WorkItems
  -> optional approval Wait WorkItems
  -> continuation LLM WorkItem
  -> final output
```

The existing Tool, MCP, Skill, approval, App, and continuation behavior becomes reusable orchestration behavior.

A Workflow calling an Agent creates a child Agent-loop frame inside the same Session execution graph.

It does not create:

- A second Session.
- A separate Workflow persistence record.
- A hidden unrelated execution root.
- A separate Tool approval mechanism.

## 9. Workflow under the unified model

A Workflow Artifact is an authored orchestration program.

At admission:

1. Resolve the Workflow Artifact.
2. Verify selection pins.
3. Capture the canonical declaration or canonical semantic IR.
4. Capture the definition digest and interpreter version.
5. Resolve static capability requirements.
6. Reject unsupported runtime capabilities before business work starts.
7. Create a ProgramFrame in the Session execution graph.
8. Create a visible Workflow-started Activity.

The Workflow interpreter creates WorkItems and child frames.

Examples:

| Workflow declaration | Unified runtime behavior             |
| -------------------- | ------------------------------------ |
| `call: agent`        | Child Agent-loop frame               |
| `call: model`        | Child single-model frame             |
| `call: tool`         | Tool WorkItem                        |
| `call: service`      | Service WorkItem                     |
| `call: activity`     | Activity WorkItem                    |
| `call: workflow`     | Child Workflow frame                 |
| `parallel`           | Fork plus sibling scopes plus join   |
| `foreach`            | Captured collection plus item scopes |
| `repeat`             | Loop state and repeated child scope  |
| `wait: human`        | Human Wait WorkItem                  |
| `wait: timer`        | Timer Wait WorkItem                  |
| `wait: event`        | Event Wait WorkItem                  |
| `machine`            | State-machine ProgramFrame           |
| `saga`               | Compensation scope and WorkItems     |

Workflow execution does not bypass the Session scheduler.

## 10. LLM WorkItems

An LLM WorkItem persists provider-neutral semantic input.

It contains:

- Selected Agent or Model identity.
- Selected revision and definition digest.
- System instruction contributions.
- Message and content input references or canonical values.
- Tool-choice semantic identities.
- Output schema.
- Semantic Model overrides.
- Explicit context transfer.
- Declared logical deadline.
- Stable operation key.

It does not contain:

- Provider endpoint.
- API key.
- HTTP headers.
- Provider client state.
- Debug configuration.
- Stream callback.

At dispatch, runtime setup resolves the current provider runtime configuration and verifies that it is compatible with the persisted semantic target.

## 11. Tool and action WorkItems

A Tool proposal from an LLM and a direct Workflow Tool call both become Tool WorkItems.

The origin differs. The persistence model does not.

| Origin                 | WorkItem source             |
| ---------------------- | --------------------------- |
| Model output           | LLM WorkItem outcome        |
| Workflow direct call   | Workflow ProgramFrame       |
| MCP App action         | Session input Activity      |
| Skill lifecycle action | Program frame or LLM output |
| Service call           | Workflow ProgramFrame       |
| Activity call          | Workflow ProgramFrame       |

A Tool WorkItem stores:

- Semantic Tool binding identity.
- Input arguments.
- Effect contract.
- Approval requirement.
- Execution mode.
- Attempt history.
- Output or error.
- Safe external receipt.
- Optional App projection.

MCP approval is a generic wait attached to a Tool WorkItem. MCP-specific tokens remain runtime-only.

## 12. Parallelism

Parallelism is a scheduler property inside one Session execution graph.

A `parallel` Workflow construct creates sibling scopes.

Each sibling scope gets:

- A local state snapshot.
- Local WorkItems.
- Local outputs.
- Local failures.
- Local waits.

The parent scope owns:

- Completion policy.
- Join behavior.
- Merge behavior.
- Aggregate outcome.

Parallel branches do not write shared parent state directly.

A normal chat session usually has one active LLM WorkItem. A Workflow can have many ready LLM WorkItems. The LLM execution path remains the same.

## 13. Waits, approvals, and events

A wait is a generic WorkItem.

Wait kinds include:

- Approval.
- Human response.
- Timer.
- Event.
- Channel message.
- Child completion.
- Authentication.
- Configuration.
- Recovery review.

A wait persists:

- Semantic request.
- Input schema.
- Allowed resolutions.
- Audience constraints.
- Correlation facts.
- Absolute expiry.
- Final resolution.

A wait does not persist:

- Approval token.
- OAuth state.
- Browser form state.
- Wails window state.
- Runtime client handle.

A human response, CLI command, VS Code interaction, or Wails action all enter the backend as a Session input Activity. The active ProgramFrame validates whether that Activity resolves a pending wait.

## 14. Runtime binding

A persisted WorkItem is semantic intent. Runtime binding is current process work.

```text
Persisted WorkItem
  -> bind current runtime dependencies
  -> verify compatibility
  -> construct ephemeral executor request
  -> dispatch
  -> persist outcome
```

### 14.1 Ready WorkItem

For a ready WorkItem:

- Resolve current Artifact/runtime bindings.
- Verify persisted Artifact revision and digest.
- Verify service or activity registration version.
- Resolve credentials only in trusted runtime code.
- Create an ephemeral executor request.
- Dispatch.

If binding compatibility fails, do not silently substitute another Model, Tool, Agent, or Workflow.

The WorkItem becomes blocked, fails safely, or requires an explicit rebind policy.

### 14.2 Dispatched WorkItem

Before external work:

1. Persist semantic input.
2. Persist selected semantic target.
3. Persist effect key when applicable.
4. Persist `dispatched`.
5. Execute external work.

After external work:

1. Persist output, error, or safe receipt.
2. Transition WorkItem state.
3. Advance the owning frame.

### 14.3 Uncertain WorkItem

If the process stops after dispatch and before a confirmed outcome:

- Mark the WorkItem uncertain.
- Do not automatically repeat non-idempotent work.
- Create a recovery wait or reconciliation action.
- Preserve the original semantic target and input.

## 15. Session Store

Session Store remains the only logical persistence model.

The current `storedSession` envelope already persists one Session and associated execution records. Keep that ownership model.

The new breaking schema should persist:

```text
StoredSession
├── Session
│   ├── metadata
│   ├── Timeline[] Activity
│   └── session defaults
│
└── Executions[] Execution
    ├── ProgramFrames[]
    ├── WorkItems[]
    ├── Attempts[]
    ├── Waits[]
    ├── Controls
    └── Outcomes
```

The implementation can keep a session document plus derived index and FTS files. That remains one Session Store model.

The FTS index remains derived data.

The execution graph is authoritative.

### 15.1 Timeline projection

A conversation projection returns only visible message Activities.

A workflow projection returns relevant workflow, action, wait, and outcome Activities.

An audit projection returns authorized technical history.

The same Session supports all three consumers.

### 15.2 Checkpoint behavior

Checkpointing remains Session-owned.

A checkpoint persists:

- The current Session timeline.
- The changed Execution.
- Associated WorkItems and waits.
- Session revision.
- Execution revision.
- Durable outcomes.

No separate Workflow checkpoint sink exists.

## 16. Cancellation and recovery

Pause, resume, cancel, recovery, and deletion are Session execution controls.

They are not chat-specific controls.

### Pause

Pause prevents new ready WorkItems from dispatching.

It does not claim to stop already-running external work.

### Resume

Resume asks the scheduler to inspect ready WorkItems again.

### Cancel

Cancel:

1. Marks the relevant execution scope cancelling.
2. Prevents new ordinary WorkItems.
3. Requests cancellation for attached child scopes.
4. Cancels eligible waits.
5. Runs supported finalization and compensation.
6. Preserves uncertainty when external outcomes are unknown.
7. Produces a durable outcome Activity.

### Recovery

On restart:

1. Restore the Session document.
2. Reconstruct program interpreters from ProgramSnapshots.
3. Restore pending waits and deadlines.
4. Mark in-flight unconfirmed work uncertain.
5. Rebind runtime dependencies only when a WorkItem becomes ready.
6. Continue deterministic orchestration from durable state.

## 17. Breaking migration

This is a breaking Session schema change.

Use explicit offline migration scripts.

Do not maintain a permanent dual runtime model.

### 17.1 Migration mapping

| Existing state         | New state                                        |
| ---------------------- | ------------------------------------------------ |
| `TranscriptMessage`    | Message Activity                                 |
| `Run`                  | Execution                                        |
| `SequentialChatDriver` | Built-in conversation or Agent-loop ProgramFrame |
| `ModelOperation`       | LLM WorkItem                                     |
| `ToolCall`             | Tool WorkItem                                    |
| `ToolAttempt`          | WorkItem attempt                                 |
| `ToolResult`           | WorkItem outcome                                 |
| `WaitingRequest`       | Wait WorkItem                                    |
| `ContinuationInput`    | Session input Activity                           |
| Assistant output       | Message Activity plus LLM outcome                |

### 17.2 In-flight old state

Migration must not guess that external work completed.

It must:

- Preserve completed outcomes.
- Convert safe queued work to ready WorkItems.
- Convert in-flight Model or Tool operations to uncertain WorkItems.
- Convert durable waits where enough data exists.
- Create recovery review waits where information is insufficient.
- Drop private authorization tokens and runtime handles.
- Rebuild Session indexes and FTS.

## 18. Backend ownership

The runtime is backend-owned.

Wails is one consumer.

Other consumers can include:

- CLI.
- VS Code extension.
- HTTP API.
- Background service.
- Test harness.
- Automation client.

Consumers do not:

- Execute orchestration logic.
- Create runtime bindings.
- Resolve credentials.
- Decide whether a WorkItem is safe to retry.
- Mutate execution state directly.
- Supply approval authority.
- Interpret private runtime state.

Consumers submit typed Session commands and read Session snapshots or projections.

```text
Wails / CLI / VS Code / HTTP
           |
           v
       Session API
           |
           v
 Session orchestration runtime
           |
           v
 Runtime setup and executors
           |
           v
 Inference / Tools / MCP / Skills / Services
```

## 19. Implementation order

1. Preserve the current Session Store ownership model.
2. Introduce Timeline Activities without losing current message data.
3. Introduce generic Executions, ProgramFrames, WorkItems, Attempts, and Waits.
4. Map the current sequential chat driver into a built-in conversation program.
5. Map the current Agent Tool loop into a built-in agent-loop program.
6. Keep `FetchCompletion` and provider execution unchanged.
7. Add Workflow Artifact compilation into ProgramFrames.
8. Add Workflow Agent and Model calls.
9. Add direct Tool and MCP WorkItems.
10. Add service and activity registry bindings.
11. Add parallel, foreach, repeat, and scope-state handling.
12. Add waits, timers, events, channels, and triggers.
13. Add state machines, saga compensation, and reconciliation.
14. Run the offline migration and rebuild derived indexes.
