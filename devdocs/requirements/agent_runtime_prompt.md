You are implementing the attached **Go Agent Runtime Proposal** in the FlexiGPT application.

Treat the proposal as the behavioral and architectural source of truth. The current frontend chat implementation, current conversation store, and current `internal/agentruntime-go` implementation are supplied as reference material and migration input.

This is an implementation task. Do not return a high-level design only.

## Primary outcome

Move chat execution ownership from the frontend into a Go Session/Run runtime while preserving the current user-facing chat loop:

- Send, Enter, Play, Fast-forward, Run, Retry, Skip, Submit error.
- Automatic tool execution and automatic continuation.
- The ability to stop upcoming automatic execution.
- Approval and intervention behavior.
- Streaming, partial output, cancellation, and error handling.
- MCP, skills, workspace, Agent, attachments, and App-context behavior.
- Existing chat presentation and authoring UX.

Breaking changes are acceptable. Compatibility layers are not required where they obscure the intended model.

The implementation must not weaken the behavior explicitly preserved in the attached proposal.

## Required architectural direction

### Session and Run are the only top-level execution state roots

Use these meanings consistently:

- **Session** is the runtime term for a conversation.
- **Run** is one accepted unit of work inside a Session.
- A model request is an operation owned by a Run, not a new top-level entity.
- A tool call, tool attempt, tool result, approval, or waiting request is owned by a Run.
- A tab is only a frontend view of a Session.

Do not create a separate top-level `ModelStep`, `ToolBatch`, `Approval`, or workflow-node entity merely to represent logical work.

Use IDs and records where required for execution, diagnostics, retries, cancellation, and future graph support, but keep ownership under the Run.

### Runtime location and dependency direction

Create the orchestration engine under:

```text
internal/agentruntime-go/session/
```

This package is the core Session/Run runtime.

It must define its own domain types and ports, including as appropriate:

- Session, SessionID, Session revision.
- Run, RunID, Run revision, lifecycle state.
- Command IDs and operation IDs.
- Model operation records.
- Tool call, tool attempt, and tool result records.
- Waiting requests and intervention resolution.
- Continuation input revisions.
- Automation controls, pause, cancellation, and limits.
- Runtime events and snapshots.
- Checkpoint and recovery contracts.
- Driver state and driver transition contracts.

The core `session` package must remain free of:

- Wails imports.
- `mapstore-go`.
- File and database storage.
- `internal/conversation`.
- `internal/artifactory-go`.
- `internal/llmartifactory-go`.
- Existing frontend DTOs.
- Concrete MCP, skill, model, workspace, or tool implementations.

The core may define ports for these capabilities. Concrete code must be supplied by adapters.

### Required target layout

Before writing implementation code, define a concrete domain-oriented target tree. Use this minimum boundary:

```text
internal/
  agentruntime-go/
    session/
      ... core Session/Run domain, coordinator, ports, events, drivers

    inference/
    attachment/
    skill/
    mcp/


  sessionstore/
    ... new file/MapStore/FTS-backed Session store implementation. this is optional wrt session knowing about it.

cmd/
  agentgo/
    ... Wails-facing wrappers and application composition
```

The final exact file names are yours to define, but the dependency direction is mandatory:

```text
session core
  <- domain specific runtimes like skill, mcp, etc
  <- app wrappers in cmd/agentgo composition
  <- Wails bindings
  <- frontend clients
```

`session` must not import domain runtimes or app wrappers or persistence/checkpoint providers.

### Existing wrapper migration

The current `internal/agentruntime-go/inference` contains artifact-backed and conversation-backed inference preparation for current things.

The new Sessions runtime must not import it directly.

Create a Session-facing inference adapter under:

```text
internal/agentruntime-go/inference/adapter.go
```

That adapter may initially reuse the current `inferencewrapper.ProviderSetAPI`, but it must translate between Session-owned domain types and current inference/artifact/conversation types.

Do not put artifact, conversation, storage, or Wails conversion code inside `internal/agentruntime-go/session`.

## Optional store and recovery requirements

Persistence is optional.

The Session runtime must run correctly with no persistence provider configured.

Implement optional core ports equivalent to:

```go
type CheckpointSink interface {
    HandleCheckpoint(context.Context, RuntimeCheckpointView) error
}

type RecoverySource interface {
    Restore(context.Context, RuntimeRestorer) error
}
```

The exact names and signatures may differ, but these rules are mandatory:

- The Session core owns live in-memory state.
- A store is optional.
- A store receives semantic checkpoint views or runtime events.
- A store controls its own file, database, event, snapshot, encryption, retention, and indexing format.
- The runtime must not impose a database schema or serialized storage format.
- Store failures are observational by default and must not stop an active in-memory Run.
- Recovery is opt-in and supplied by the configured recovery source.
- Recovery must not blindly replay in-flight external tool operations.
- No persistence provider means no restart recovery, which is acceptable.
- An in-memory provider must be possible.
- A file or database provider must be possible without changing the Session core.

The Session runtime must record state transitions in its live Run state before considering continuation. It must not rely on frontend persistence or frontend state to know whether a tool result exists.

## New Session store

Create a new breaking Session store. Do not make the current `internal/conversation` store the canonical runtime store.

The current conversation store is reference material for:

- `mapstore-go` directory layout.
- UUID-v7 file naming.
- Month partitioning.
- JSON encoding.
- Full-text search integration.
- List, search, and read behavior.

Create the new store outside the Session core:

```text
internal/sessionstore/
```

The store shoudl implmeent the runtime needed ports and then bootstrap shoudl init it nad inject to sessions as needed.
The Session store must support the Session-oriented read model needed by chat:

- Read Session.
- Read Session conversation projection.
- List Sessions.
- Search Sessions.
- Read active/recent Runs where needed.
- Read Run snapshots and waiting requests where needed.
- Export Session data where needed.

The frontend must not directly call Session-store write APIs.

All Session transcript and execution writes must be orchestrator-driven through the Session runtime command path.

The Session store may be configured as the application’s default checkpoint/projection provider, but the Session core must remain functional without it.

The current store is battle tested for multiple scenarios. Do not modify or do things on your own, us this as proper reference.

### Attachment storage

The new Session store must store attachment information as references.

It must not persist hydrated file payloads, generated Base64 content, or copied attachment content inside transcript messages merely because those values were used for inference.

Support durable references for at least:

- Files.
- Directories or folder selections.
- Images.
- URLs.
- Generic attachment handles where currently supported.

Use the existing attachment domain types as reference material. If the existing attachment type cannot faithfully represent a directory reference required by Session storage, define a Session-store attachment-reference type and provide adapters.

Keep attachment materialization behind a Session port and wrapper.
Again, Sessions doesnt really understand if attachments are done in single file or as refs or whatever, it takes its own input and works with it.

## Existing service integration

Use proper narrow interfaces and constructor injection. Do not hardwire concrete services into the Session core.

The Session runtime must use existing systems through adapters:

| Capability                            | Existing implementation to integrate through wrappers                                     |
| ------------------------------------- | ----------------------------------------------------------------------------------------- |
| Model completion                      | `internal/agentruntime-go/inferencewrapper` and the Go service behind `CompletionWrapper` |
| Go tools                              | Tool runtime and `InvokeGoToolTarget` behavior                                            |
| Skill runtime                         | `internal/agentruntime-go/skill`                                                          |
| MCP invocation and policy             | `internal/agentruntime-go/mcp/invocation`, `mcp/connection`, and `mcp/server`             |
| Workspace materialization             | Existing workspace runtime APIs                                                           |
| Agent preparation                     | Existing Agent store/service resolution APIs                                              |
| Attachments                           | `internal/attachment`                                                                     |
| Legacy conversation import/projection | `internal/conversation` only through wrappers                                             |

The core runtime should expose ports such as:

- Model executor.
- Tool executor.
- Agent resolver.
- Skill-session controller.
- MCP executor.
- Workspace resolver.
- Attachment materializer.
- Event publisher.
- Clock and ID generator where useful.
- Optional checkpoint sink and recovery source.

Concrete wrappers may use existing MCP and Skill runtimes directly. The Session core itself must depend only on its own ports.

### MCP approval integration

Do not duplicate MCP policy or approval logic.

Use the existing MCP tool bridge:

```text
internal/agentruntime-go/mcp/invocation.ToolBridge
```

The Session runtime must represent approval as a generic Run waiting request.

The MCP wrapper must map:

- `ToolBridge.Evaluate`
- `ToolBridge.ResolveApproval`
- `ToolBridge.Invoke` or `InvokeMapped`

into the generic Session wait and execution model.

The Session core must not know MCP approval-token details.

Approving a call must not automatically bypass a currently active automation hold.

### Skill-session integration

The frontend currently owns skill-session lifecycle for chat. Move that ownership to the backend Session runtime for Session/Run execution.

The Session runtime must own the lifecycle needed for a Run:

- Create or replace skill sessions.
- Close them when appropriate.
- Track enabled and active skill state.
- Preserve installed versus workspace-owned skill membership.
- Update active skills after skill lifecycle tools execute.
- Avoid stale session-creation results overwriting newer Run state.

The frontend may still edit selections, but it must not be the owner of the active skill-session state for an executing Run.

## Current chat-loop implementation requirements

Implement the initial sequential chat driver.

It must preserve:

- Current history selection semantics, including pure-user-turn behavior and pinned system/developer messages.
- Per-operation effective model and configuration snapshots.
- Provider-managed tools remaining provider-managed.
- Go tools, MCP tools, and skill tools using the appropriate executor.
- Tool-call identity bound to the originating model operation.
- Explicit tool errors versus failures without a usable result.
- Retry, skip, and explicit error-result handling.
- Sequential automatic execution.
- Existing automatic continuation rules.
- Manual execution and Fast-forward semantics.
- “Run tools only” semantics, including existing automatic-batch behavior.
- Automation holds that stop upcoming automatic execution.
- Pause and cancellation as separate controls.
- Partial streaming output.
- Empty final response handling.
- Interrupted model request handling.
- MCP App context and tool behavior.
- Current Agent configuration semantics.
- Workspace requested selection versus actual send-time usage.

Do not implement graph workflows, subagents, or agent-to-agent transfer now.

Do implement the driver abstraction and serialized driver state required to add them later.

## Backend implementation

Do not begin frontend migration first. Do below things together for the backend coherently first.

### First: Domain layout, core contracts, session runtime and backend composition

- Define the target folder tree.
- Define Session/Run domain types.
- Define core ports.
- Define command, snapshot, event, checkpoint, and restore contracts.
- Define the chat-driver state and transition model.
- Define explicit dependency direction.

Do not stop after this design step. Continue into implementation as:

- Implement the in-memory Session/Run coordinator.
- Implement the sequential chat driver.
- Implement optional checkpoint and recovery ports.
- Wire model, tools, skills, MCP, workspace, Agent, and attachment capabilities.
- Add Wails-facing Session runtime command/query APIs.
- Initialize the runtime only after its model, tool, skill, MCP, workspace, Agent, and attachment dependencies are ready.
- Shut it down before those dependencies are closed.

### Second: Session store and Wails-facing read API

- Implement the new `internal/sessionstore` package.
- Reuse the current conversation store’s MapStore, directory partitioning, UUID-v7 filename, and FTS approach where appropriate.
- Implement the Session-store wrapper/adapter.
- Add a Wails-facing Session-store wrapper exposing read/list/search/export APIs.
- Do not expose direct frontend transcript write APIs.

### Third: Wails event bridge

- Add backend event publication for Session and Run changes.
- Include Session ID, Run ID, event sequence, command ID, and relevant operation IDs.
- Support frontend reattachment through subscription plus snapshot retrieval.
- Do not rely on a Wails event as the only source of truth.
- Do not make event persistence mandatory.

## Frontend migration

Move the chat execution path from frontend orchestration to backend Session runtime.

The chat UI may issue commands and read Session/Run snapshots. It must stop directly owning execution.

For the chat path, remove or replace direct frontend ownership of:

- `FetchCompletion`.
- `CancelCompletion`.
- `PutConversation`.
- `PutMessagesToConversation`.
- Frontend tool execution scheduling.
- Frontend auto-exec draining.
- Frontend auto-submit scheduling.
- Frontend skill-session creation/replacement.
- Frontend MCP approval promise ownership.
- Assistant-turn delivery as an execution mechanism.

The frontend may retain:

- Draft editing.
- Attachment picking.
- UI validation.
- Menus.
- Focus.
- Rendering.
- Stream display batching.
- Local tab layout persistence.
- Management-page APIs.
- Direct read APIs for Session conversation data.

The frontend writes Session state only through runtime commands. It does not write the Session transcript directly to the store.

## MCP App routing requirement

Current MCP App events are routed through the selected tab. Replace that behavior.

MCP App actions must be explicitly associated with their owning Session and, where relevant, originating Run or tool result.

A delayed App request must not be routed to whichever tab is currently selected.

Maintain browser sandboxing and frontend iframe host responsibilities.

## Wails integration requirement

Add separate Wails wrappers rather than overloading unrelated wrappers.

Expected responsibilities:

| Wrapper                 | Purpose                                                                                                       |
| ----------------------- | ------------------------------------------------------------------------------------------------------------- |
| `SessionRuntimeWrapper` | Commands, Run snapshots, Session runtime queries, cancellation, pause, continuation, waits, and event bridge. |
| `SessionStoreWrapper`   | Read/list/search/export Session projections. No direct frontend transcript writes.                            |

Update `cmd/agentgo/app.go` composition and shutdown ordering accordingly.

Use the current Wails wrapper patterns and generated frontend binding patterns. Do not hand-edit generated bindings if the repository generation workflow owns them.

## Frontend migration target

The frontend should converge on this model:

```text
Composer authoring state
  -> Session runtime command
  -> accepted command receipt
  -> Session/Run event stream and read snapshots
  -> chat presentation
```

The frontend should consume Session conversation read APIs going forward.

All transcript and execution writes should be orchestrator-driven.

The existing current conversation store may remain available only for legacy import, migration, or compatibility projection while the new Session store becomes the active chat storage path.

## Do not spend work on these items

Do not spend output or implementation time on:

- Tests.
- Test fixtures.
- Documentation files.
- README updates.
- Migration guides.
- Demo code.
- Compatibility shims that keep the old frontend runtime alive.
- A graph workflow engine.
- Subagent execution.
- Agent transfer execution.
- Broad management-page redesign.

Implement the actual runtime and migration path first.

## Output requirements

Your response must contain:

### Target layout first

Start with:

```md
## Target domain layout
```

Show the final intended file and folder tree and explain dependency direction briefly.

### Code delivery format

- For every **new file**, provide the full file contents.
- For every **replacement file**, provide the full file contents.
- For every **existing modified file**, provide a unified diff against the exact repository path.
- Do not use placeholders such as `...`, “existing code unchanged”, or omitted method bodies for required implementation.
- Do not invent APIs not supported by the attached code. Inspect actual types and interfaces first.
- If an essential dependency is not attached, identify that exact missing path before relying on it.

### End-of-response task list

End with exactly these sections:

```md
## Implementation status

- [x] ...
- [x] ...
- [ ] ...

## Next steps

- [ ] ...
- [ ] ...
```

The task list must be concise, copy-pastable, and suitable for maintaining in a file.

Do not place prose after `## Next steps`.

I have already done some amount of work. build on that.

Work as much as possible with available files. after that in next steps if your next items require some more files, ask for them too.
complete the first and second step properly now.
the file layout i have definied myself a bit.

now, for sessionstore we will not have anything inside runtime. it is a pluggable thing. we should define that sessionstore properly based on conversation store and have files in internal/sessionstore. any other corrections in current things must be rectified.

now give output properly.
