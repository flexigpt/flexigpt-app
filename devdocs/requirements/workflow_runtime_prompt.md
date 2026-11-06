You are taking over the FlexiGPT Session Runtime v3 and Workflow Runtime v3 backend implementation.

Work directly in the supplied FlexiGPT source tree. The repository state is authoritative.

The primary deliverable is source changes in the working tree. Do not return a hypothetical patch instead of editing the tree.

Hard execution and output contract:

- Use the FULL available implementation budget.
- The caller has reserved `128000` output tokens.
- Do not produce a final response before at least `100000` meaningful implementation tokens have been used, unless every requirement below is actually complete in the source tree.
- Meaningful implementation tokens are production source edits only. Do not satisfy this requirement with explanations, repeated source, comments, Markdown, docs, tests, generated filler, speculative guards, empty interfaces, placeholder types, no-op branches, or fake completion markers.
- If one completion reaches an output limit, continue in the same tree immediately. Do not stop at a partial milestone.
- Do not ask for confirmation, do not ask for missing work items, and do not defer feasible work.
- Do not write tests.
- Do not run tests.
- Do not add test files.
- Do not modify requirements, Markdown, changelogs, or documentation in this pass.
- Do not spend time on cosmetic cleanup, broad hardening, formatting-only changes, speculative validation, compatibility wrappers, or unrelated refactors.
- Do not add a new persistence root.
- Do not create a Workflow repository, invocation store, event store, trigger store, or execution store outside the existing Session v3 document and derived Session Store v3 index and FTS data.
- Do not modify Session v2 or Session Store v2 behavior.
- Do not make Session v2 depend on Session v3.
- Do not increment `apiVersion: v1`.
- Do not change `artifact.workflow.v1@1`.
- Do not introduce `artifact.workflow.v1@2`.
- Do not create fallback chat text from arbitrary Workflow JSON.
- Do not silently omit a declaration field.
- Do not accept a valid Workflow v1 feature and later ignore it, fabricate success, or execute it as a no-op.
- Error messages added by this pass must be lower-case.

Read these files first and treat them as the required behavior:

- `devdocs/requirements/workflow_runtime.md`
- `devdocs/requirements/workflow_model.md`
- `internal/agentruntime-go/sessionv3/...`
- `internal/agentruntimesetup/sessionv3runtime/...`
- `internal/agentruntimesetup/workflowruntime/...`
- `internal/llmartifactory-go/workflow/...`
- `internal/sessionstorev3/...`

Architecture is mandatory:

- `internal/agentruntime-go/...` owns typed orchestration, ProgramFrames, WorkItems, Attempts, Waits, timers, recovery, command idempotency, cancellation, durable state, and persistence-safe semantics.
- `internal/llmartifactory-go/...` owns declaration syntax, schema validity, static CEL scope, declaration target relationships, and portable declaration validity.
- `internal/agentruntimesetup/...` owns declaration-to-runtime compilation, Artifact pinning, current dependency resolution, provider plans, Tool binding, MCP discovery, Skill resolution, Workspace planning, attachment materialization, Service binding, Activity binding, and external domain execution adaptation.
- Session Runtime v3 must not import Artifact packages or setup packages.
- Session Runtime v3 must not resolve Artifacts, configure providers, read Workspace files, discover MCP tools, render Skills, execute Tools, or materialize attachments itself.
- Do not add broad facade interfaces. Use concrete dependencies or narrow typed domain ports only at real domain boundaries.
- Use `CanonicalValue` only for arbitrary JSON boundaries.
- Keep IDs, pins, kinds, limits, effects, attempts, waits, receipts, errors, budgets, trigger state, and control-flow state as concrete typed structures.

The durable external-work sequence is non-negotiable:

1. Persist typed WorkItem intent.
2. Resolve current dependencies.
3. Verify Artifact pins and current target bindings.
4. Persist semantic binding facts.
5. Persist an Attempt before external invocation.
6. Invoke external work.
7. Persist confirmed output, receipt, failure, cancellation fact, or uncertainty.
8. Never automatically replay uncertain non-idempotent work.
9. Use Runtime timers for durable timing.
10. Cancel all timers and goroutines in `Runtime.Close`.

Implement all remaining Workflow Runtime v3 and Session Runtime v3 backend work in this order.

First: complete callable targets and executable planning.

- Finish Tool Artifact targets through typed WorkItems.
- Finish MCP Tool targets through current MCP discovery and persisted MCP binding facts.
- Finish Service targets through registered typed Service bindings.
- Finish Activity targets through registered typed Activity bindings.
- Finish nested Workflow targets through child Workflow ProgramFrames.
- Finish Agent targets through child Agent-loop ProgramFrames with typed Agent execution plans.
- Finish Model targets through child single-model ProgramFrames with typed Model execution plans.
- Finish static target sets.
- Finish resolver-backed target sets with typed resolver WorkItems where external discovery is required, persisted resolved membership, and persisted selected target identity.
- Finish full typed plans before Agent-loop or single-model dispatch:
  - Agent,
  - Workspace,
  - Model,
  - instruction,
  - Text,
  - Tool,
  - Skill,
  - MCP,
  - attachment,
  - message,
  - context,
  - channel,
  - port.
- Provider SDK Tool Artifacts must execute only as provider-managed Tool choices inside typed Model plans.
- Provider SDK Tool Artifacts must never be converted into local Tool WorkItems or fabricated local success.
- Finish typed port plans and runtime delivery:
  - input ports,
  - output ports,
  - streamed ports,
  - channel access,
  - schema validation,
  - port ownership,
  - parent-child messaging,
  - durable messages.
- Preserve target, call, and target-set input, output, error, receipt, mode, and effect contracts.

Second: complete effects, reliability, reconciliation, and compensation.

- Complete `read`, `idempotent`, `keyed`, and `non-idempotent` contracts.
- Persist one stable idempotency key per logical keyed WorkItem.
- Preserve that same key through retries, waits, recovery, and resumption.
- Propagate idempotency keys into Tool, Service, Activity, MCP, Agent, Model, and Workflow execution plans where the owning domain supports it.
- Add typed effect state to WorkItems, Attempts, receipts, and errors.
- Model partial application explicitly.
- Implement reconciliation as typed WorkItems.
- Reconciliation must receive typed original operation facts, input, selected target, key, receipt, and uncertainty error.
- Reconciliation outcomes must distinguish:
  - applied,
  - not applied,
  - unknown.
- Only confirmed non-application can enable a retry after uncertainty.
- Unknown remains uncertain and requires typed recovery review.
- Implement saga tracking.
- Persist confirmed compensatable effects in dependency order.
- Implement reverse compensation order.
- Implement compensation WorkItems.
- Implement compensation receipts, failures, uncertainty, and reverse dependency behavior.
- Implement compensation error policy:
  - continue,
  - stop.
- Preserve the primary failure and attach compensation failures as typed causes.
- Implement cleanup allowance separately from normal deadline allowance.
- Cancellation must permit bounded cleanup and compensation work after normal work deadlines.
- Do not automatically replay uncertain non-idempotent work.
- Do not automatically compensate unknown effects before reconciliation.

Third: complete all Workflow structured behavior.

- Finish block `catch`.
- Finish block `finally`.
- Ensure cancellation retains cancellation semantics through catch and finally.
- Ensure finalizer failure becomes primary only after prior success.
- Ensure finalizer failure is attached as a cause after prior failure or cancellation.
- Finish machines:
  - initial state,
  - local state,
  - inherited state,
  - entry,
  - exit,
  - immediate transitions,
  - event transitions,
  - timer transitions,
  - transition blocks,
  - final output,
  - serialized event handling,
  - durable re-entry timers,
  - cancellation,
  - recovery.
- Finish all Parallel policies:
  - all,
  - first-completed,
  - first-success,
  - quorum,
  - remaining cancel,
  - remaining wait,
  - fail-fast,
  - collect.
- Ensure uncertain branch work remains visible after another branch wins.
- Finish Foreach:
  - source-order output,
  - captured input collection,
  - concurrency,
  - all completion modes,
  - cancellation,
  - isolated state,
  - explicit merge,
  - empty collection semantics,
  - item schemas,
  - max items,
  - budget interaction.
- Finish Repeat:
  - count,
  - while,
  - until,
  - until-cancelled,
  - timeout,
  - carried state,
  - local per-iteration state,
  - exhaustion,
  - onExhausted,
  - nested break,
  - nested continue,
  - cancellation,
  - recovery.
- Implement durable Fork WorkItems where child scheduling crosses a durable boundary.
- Implement durable Join WorkItems where completion aggregation crosses a durable boundary.
- Do not represent Fork or Join only as transient interpreter behavior where durable scheduling state is required.

Fourth: complete durable coordination and triggers.

- Implement event triggers.
- Implement schedule triggers.
- Persist trigger registrations and trigger state inside the existing Session v3 document.
- Do not add a trigger persistence root.
- Implement trigger overlap policies:
  - allow,
  - skip,
  - queue,
  - replace.
- Implement missed-occurrence policies:
  - skip,
  - latest,
  - bounded catch-up.
- Implement interval schedules.
- Implement five-field cron schedule behavior with IANA timezone semantics.
- Preserve calendar gap and repeated-local-time behavior defined by Workflow v1.
- Implement event trigger behaviors:
  - start,
  - signal,
  - start-or-signal.
- Persist accepted trigger occurrence identity, input, instance key, occurrence time, selected execution, queue state, and deduplication facts.
- Complete event retention.
- Complete subscription retention.
- Complete subscription capacity and byte limits.
- Complete event consumption by subscription identity.
- Complete channel retention and byte limits.
- Complete channel cursors.
- Complete channel consume and observe semantics.
- Complete emit deduplication.
- Complete human waits:
  - users,
  - roles,
  - typed request,
  - typed response schema,
  - durable assignment facts,
  - response authorization boundary.
- Complete completion waits for WorkItems, ProgramFrames, and Executions.
- Complete signals with `(source, id)` identity.
- Complete machine serialization and recovery.
- Runtime timers must be rearmed only after full restore completes.

Fifth: complete normal Session v2 backend capability in Session Runtime v3.

- Implement direct Model chat.
- Implement Agent-backed chat.
- Implement Workspace contribution and instruction planning.
- Implement attachment admission through Attachment Runtime:
  - local file snapshotting,
  - image snapshotting,
  - directory expansion,
  - URL normalization,
  - replay-safe materialization,
  - historical modification handling,
  - provider adaptation.
- Session Runtime v3 must use a narrow Attachment domain port.
- Session Runtime v3 must not import setup packages or implement file access itself.
- Implement Go Tools.
- Implement provider SDK Tools.
- Implement MCP Tools.
- Implement Skill Tools.
- Implement Service and Activity runtime bindings.
- Implement MCP resources.
- Implement MCP prompts.
- Implement MCP Apps.
- Implement MCP approvals.
- Implement App Tool calls.
- Implement App resource reads.
- Implement approved App context staging.
- Implement Skill sessions.
- Implement active Skill lifecycle.
- Implement Skill Tool invocation and resource handling.
- Implement provider streaming.
- Implement protocol outputs.
- Implement usage.
- Implement debugger projections.
- Implement Tool result continuation.
- Implement cancellation.
- Implement retries.
- Implement waits.
- Implement recovery.
- Implement uncertainty.
- Implement durable command idempotency.
- Ensure V3 command and projection APIs remain transport-neutral.
- Wails must remain only an adapter.

Persistence and recovery requirements:

- `internal/sessionstorev3` remains the only V3 authoritative store.
- Derived index and FTS remain rebuildable.
- Tombstones remain durable deletion barriers.
- Restore must preserve command idempotency records.
- Restore must turn unknown dispatched mutating work into uncertainty.
- Restore must never replay uncertain non-idempotent work.
- Restore must rebind only replay-safe work through current dependency verification.
- Private clients, callbacks, credentials, approval tokens, OAuth tokens, native paths, compiled CEL, compiled schemas, timers, cancellation functions, and browser state must not persist.

Strict implementation rules:

- Replace each temporary unsupported branch with typed behavior in the same feature slice.
- Remove development-only admission rejection only after the corresponding typed behavior exists.
- Do not leave valid Workflow v1 features permanently rejected.
- Do not create `map[string]any` durable semantic state.
- Do not persist `json.RawMessage` as a semantic runtime model.
- Do not create broad facade interfaces.
- Do not turn generic Workflow input into arbitrary chat text.
- Do not add fallback success paths.
- Do not omit failure, cancellation, or uncertainty propagation.
- Do not silently weaken declared effect behavior.
- Do not silently weaken declared target, port, channel, event, schema, resource, budget, or trigger behavior.
- Use and rectifyy if missing proper ENUM usage everywhere.
- Do not add broad guard code before functional coverage is complete.
- Do not stop after implementing type definitions without compiler, interpreter, persistence, recovery, and execution wiring.

Use actual source inspection to find every remaining temporary branch, including:

- `unsupportedWorkflowOperation`,
- `ErrUnsupportedOperation`,
- `unsupported sessionv3 WorkItem type`,
- incomplete `WorkflowStep` switch branches,
- compiler rejections for valid Workflow v1 fields,
- ignored declaration fields,
- temporary return values,
- incomplete trigger manager behavior,
- placeholder Service or Activity binding behavior,
- provider Tool branches,
- attachment bypasses,
- App omissions,
- source-schema/runtime mismatches,
- recovery paths that lose effect semantics,
- close paths that leave timers or goroutines running.

Before finalizing, audit all valid Workflow v1 source fields in:

- `internal/llmartifactory-go/workflow/contract/v1/types.go`
- `internal/llmartifactory-go/workflow/contract/v1/validation.go`
- `internal/llmartifactory-go/workflow/contract/v1/workflow-v1.schema.json`

For every valid field, prove one of these is true in source:

1. It compiles into typed runtime behavior.
2. It is invalid by language semantics.
3. It requires a currently unavailable external binding, credential, policy, or Artifact pin and fails before unrelated external work begins.

Do not leave a valid field as a permanent unsupported runtime feature.

Final response requirements:

- No progress narrative.
- No tests section.
- No documentation section.
- No apology.
- No claim that work is complete unless every listed requirement is implemented in the source tree.
- Return only:
  - `implemented:` with concise source-level facts,
  - `files changed:` with paths,
  - `remaining:` only if an actual unavailable input, authorization, tool, or environment constraint blocked work.
- If no real blocker exists, continue implementing instead of listing remaining work.
