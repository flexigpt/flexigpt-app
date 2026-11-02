You are implementing a breaking Session Runtime v3 migration for FlexiGPT.

The architecture is non-negotiable:

1. A Session is the single durable root for one coherent activity.
2. Chat, Agent loops, and Workflow execution are orchestration programs inside
   the same Session. They are not Session kinds and they are not separate
   persistence roots.
3. Do not introduce a standalone Workflow repository, filesystem store,
   Workflow runtime persistence model, Workflow session type, or
   Run.Kind=chat|workflow discriminator.
4. Keep the existing Session Store as the only logical persistence owner.
   Internal Store files, indexes, and FTS may remain implementation details.
5. FetchCompletion and provider execution remain provider-facing execution
   APIs. Do not redesign them merely because Workflows are added.
6. The change is breaking. Use an explicit offline migration. Do not add a
   permanent legacy dual-read or dual-runtime compatibility layer.
7. Wails is one backend consumer only. CLI, VS Code, HTTP, and test clients
   must use the same backend Session commands and projections.

The target model is:

Session

- Timeline[] Activity
- Executions[] Execution
- Session defaults and metadata

Execution

- ProgramFrames[]
- WorkItems[]
- Attempts[]
- Waits[]
- Controls
- Outcome

Activity is user-facing durable history.
WorkItem is resumable execution state.
An Activity and WorkItem are related but are not the same object.

A message is an Activity with kind=message.
A Tool summary is an Activity with kind=action.
A Workflow milestone is an Activity with kind=workflow.
A wait or approval is an Activity with kind=wait.
Not every WorkItem transition must become an Activity.

ProgramFrame is the universal orchestration scope.
A built-in conversation program implements ordinary chat.
A built-in agent-loop program implements the current LLM -> Tool -> LLM loop.
A Workflow Artifact compiles to an Artifact-backed program.
A nested Workflow is a child ProgramFrame.
An Agent call from a Workflow is a child built-in agent-loop ProgramFrame.
A Model call from a Workflow is a child single-model ProgramFrame.

WorkItem types must be a closed typed union, not unvalidated arbitrary maps.
At minimum support:

- llm
- tool
- service
- activity
- transform
- condition
- stateUpdate
- fork
- join
- wait
- emit
- compensation

Persist semantic execution facts:

- Program snapshot and interpreter version.
- Artifact identities, revisions, and definition digests.
- Canonical semantic inputs and outputs.
- Branch decisions.
- Loop and scope state.
- WorkItem dependencies and state.
- Retry history.
- Idempotency keys.
- Absolute deadlines.
- Wait requests and resolutions.
- Safe external receipts.
- Sanitized errors and outcomes.

Do not persist runtime bindings:

- API keys.
- OAuth tokens.
- MCP approval tokens.
- Provider headers with secrets.
- HTTP clients.
- MCP clients.
- Go function pointers.
- cancellation functions.
- stream callbacks.
- debug settings.
- native paths.
- compiled CEL programs.
- browser or Wails state.

Runtime binding occurs only when a ready WorkItem dispatches:

1. Resolve current runtime dependencies.
2. Verify compatibility with the persisted semantic target.
3. Build ephemeral executor data.
4. Dispatch.
5. Persist the outcome.

A semantic value that affects resumable behavior must persist.
A process-local transport or observation value must not persist.

Examples:

- Persist accepted model temperature override.
- Persist approval decision.
- Persist selected target.
- Persist absolute deadline.
- Do not persist provider HTTP timeout, debug setting, or API key.

External-effect safety is mandatory:

- Persist semantic intent before dispatch.
- Persist dispatched state before external work.
- Persist confirmed outcome after external work.
- Treat crash-after-dispatch-before-outcome as uncertain.
- Do not automatically repeat uncertain non-idempotent work.
- Implement recovery review or reconciliation paths.

Keep the current Session persistence essence:

- One stored Session document remains the durable root.
- Session Store checkpoint and restore remain the persistence path.
- Existing index and FTS stay derived data.
- Existing Session command idempotency, revision checks, events, checkpointing,
  cancellation, and recovery behavior should be reused where valid.

Breaking migration:

- Create explicit offline migration tooling.
- Translate TranscriptMessage to Activity(kind=message).
- Translate Run to Execution.
- Translate ModelOperation to WorkItem(type=llm).
- Translate ToolCall and ToolAttempt to WorkItem(type=tool) and Attempt.
- Translate WaitingRequest to WorkItem(type=wait).
- Translate current sequential chat driver state to a built-in conversation or
  agent-loop ProgramFrame.
- Convert in-flight external work to uncertain state unless a durable outcome
  proves completion.
- Drop private authorization tokens, process handles, and runtime-only data.
- Rebuild Session indexes and FTS after migration.
- Do not attempt live in-place migration while the old runtime is active.

Do not start implementation until you inspect the relevant current source.

Required directories to inspect first:

- internal/agentruntime-go/session/
- internal/agentruntime-go/runtimesetup/
- internal/agentruntime-go/inference/
- internal/sessionstore/
- internal/llmartifactory-go/workflow/
- internal/llmartifactory-go/core/composition/
- internal/llmartifactory-go/core/declaration/
- internal/artifactsetup/
- cmd/agentgo/

Implementation requirements:

1. Preserve FetchCompletion as the provider single call execution boundary.
2. Replace hardcoded sequential-chat-only orchestration with a versioned ProgramFrame interpreter model.
3. Do not encode program kind as Session kind or Run kind.
4. Create a closed typed WorkItem model. Do not use arbitrary map-shaped execution state as the primary durable model.
5. Keep opaque JSON only where a versioned program interpreter explicitly owns a bounded canonical payload. Validate it on persistence and restoration. The json also should be only transport with its type existing with the interpreter owning the type and common model usign narrow ports to supprot different such types. these should be minimal types.
6. Preserve current user-visible conversation behavior through a conversation projection over Timeline Activities.
7. Add Workflow and activity projections without making Wails authoritative.
8. Ensure backend commands work without Wails.
9. Migration will be done as a offline script. you need not give anything related to it. the code dirs related to session persistence will be moved to new version by the user so that nothing breaks.
10. Test addition is done later via a separate implementor. You jsut need to mention in the next steps as someone needs to add tests for belwo and anything you think more:
    - normal chat migration and replay;
    - Agent Tool loop migration and replay;
    - Workflow admission;
    - parallel LLM WorkItems;
    - Tool approval waits;
    - pause, resume, and cancel;
    - crash after dispatch;
    - uncertain external effects;
    - Session Store restore;
    - stale Artifact binding;
    - Wails-independent CLI-style command use;
    - FTS rebuild after migration.
11. Do not change dependency versions unless an actual missing dependency is demonstrated by inspected source.
12. Do not leave a second Workflow persistence mechanism in the repository.
13. Do not claim a Workflow declaration feature is supported until it has a complete ProgramFrame, WorkItem, binding, persistence, cancellation, and recovery mapping.

Deliver:

- Updated HLD if any changes needed.
- Full and proper implementation patch following the instructions attached.
- Exact removed files with proper rm commands.
- A concise final status that separates authored, applied, and verified work.
