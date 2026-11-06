You are taking over the FlexiGPT Session Runtime v3 and Workflow Runtime v3 backend implementation.

This is a Markdown-to-Markdown implementation handoff.

The supplied repository file bodies and the supplied HLD Markdown files are authoritative source context. This is not a tool-driven session unless a filesystem or editing tool is explicitly provided by the execution environment.

When no filesystem tool is available:

- Produce apply-ready unified diffs for existing files.
- Produce complete replacement-file contents for deliberate file replacement.
- Identify exact paths.
- Distinguish authored patch content from applied working-tree changes.
- Do not claim that a patch was applied when no write tool exists.
- Do not stop merely because the environment has no tool. Produce the requested Markdown patch artifacts.

When filesystem tools are available:

- Edit the working tree directly.
- Use repository state as authoritative.
- Do not substitute a hypothetical patch for direct edits.
- Still provide a concise change record that distinguishes authored, applied, and verified work.

Treat this work as financially consequential engineering work. Incomplete, fabricated, repetitive, or misleading output can waste significant user time and money. Preserve trust through exact scope tracking, substantive implementation, and truthful status reporting.

Do not use loopholes to avoid the required implementation volume or scope.

Do not satisfy the token requirement with:

- Explanations.
- Repeated code.
- Comments.
- Documentation.
- Tests.
- Generated filler.
- Placeholder types.
- Empty interfaces.
- No-op branches.
- Compatibility wrappers.
- Fake status markers.
- Repeated unchanged source.
- Speculative guards.
- Broad refactors unrelated to the requested feature slice.

The minimum output requirement is authoritative:

- Do not send a final response before at least `100000` meaningful production-source implementation tokens have been emitted.
- Count only substantive production source additions or modifications that implement runtime behavior, compiler behavior, persistence behavior, recovery behavior, execution behavior, domain binding behavior, or caller wiring.
- Do not count Markdown prose, status text, comments, tests, documentation, repeated unchanged code, or generated filler.
- If one response reaches an output limit before the threshold or requested implementation scope is complete, continue immediately in the same repository state in the next response.
- A preferred response length is not a completion boundary.
- The only valid reasons to stop before the threshold are:
  - Every requested requirement is actually complete in the source tree.
  - A hard platform output limit prevents continuation.
  - A required input, authorization, source file, repository write capability, or external domain contract is unavailable.
- If blocked, complete all independent unblocked work first. State the exact blocker, evidence, and smallest resumption input.

Use plain Markdown.

Markdown output rules are attached as instructions. Follow ALL instructions properly including language and output formats.

Do not write tests.
Do not add test files.
Do not make tests, documentation, changelogs, or release notes part of the functional completion gate.
Do not modify Session v2 behavior.
Do not make Session v2 depend on Session v3.
Do not add a new persistence root.
Do not create a Workflow repository, invocation store, execution store, event store, trigger store, or task store outside the existing Session v3 document and Session Store v3 derived data.
Do not increment `apiVersion: v1`.
Do not change `artifact.workflow.v1@1`.
Do not introduce `artifact.workflow.v1@2`.
Do not convert arbitrary Workflow JSON into fallback chat text.
Do not silently omit a valid declaration field.
Do not accept a valid Workflow v1 feature and later ignore it, fabricate success, or execute it as a no-op.

All newly added error messages must be lower-case.
Proper enums and consts are to be used end to end. no hardcoded strings/numbers anywhere.

Read these authoritative files first:

- `devdocs/requirements/workflow_model.md`
- `devdocs/requirements/workflow_runtime.md`
- `internal/agentruntime-go/sessionv3/...`
- `internal/agentruntimesetup/sessionv3runtime/...`
- `internal/agentruntimesetup/workflowruntime/...`
- `internal/llmartifactory-go/workflow/...`
- `internal/sessionstorev3/...`

Use the architecture boundaries exactly:

- `internal/agentruntime-go/...` owns typed orchestration, ProgramFrames, WorkItems, Attempts, Waits, timers, recovery, command idempotency, cancellation, durable state, and persistence-safe semantics.
- `internal/llmartifactory-go/...` owns declaration syntax, schema validity, static CEL scope, declaration target relationships, and portable declaration validity.
- `internal/agentruntimesetup/...` owns declaration-to-runtime compilation, Artifact pinning, dependency resolution, provider plans, Tool binding, MCP discovery, Skill resolution, Workspace planning, attachment materialization, Service binding, Activity binding, and external domain execution adaptation.
- Session Runtime v3 must not import Artifact packages or setup packages.
- Session Runtime v3 must not resolve Artifacts, configure providers, read Workspace files, discover MCP tools, render Skills, execute Tools, or materialize attachments itself.
- Do not add broad facade interfaces. Use concrete dependencies or narrow typed domain ports only at real domain boundaries.
- Use `CanonicalValue` only for arbitrary JSON boundaries.
- Keep IDs, pins, kinds, limits, effects, attempts, waits, receipts, errors, budgets, trigger state, and control-flow state as concrete typed structures.

Use the durable external-work sequence exactly:

1. Persist typed WorkItem intent.
2. Resolve current dependencies through the owning setup domain.
3. Verify Artifact pins and current target bindings.
4. Persist semantic binding facts.
5. Persist an Attempt before external invocation.
6. Invoke external work.
7. Persist confirmed output, receipt, failure, cancellation fact, or uncertainty.
8. Never automatically replay uncertain non-idempotent work.
9. Use Runtime timers for durable timing.
10. Cancel all timers and owned goroutines in `Runtime.Close`.

Implement in this order.

### Phase A: callable targets and typed plans

Complete:

- Tool Artifact targets.
- MCP Tool targets.
- Service targets.
- Activity targets.
- Nested Workflow targets.
- Agent targets.
- Model targets.
- Static target sets.
- Resolver-backed target sets.
- Typed Agent plans.
- Typed Workspace plans.
- Typed Model plans.
- Typed instruction plans.
- Typed Text plans.
- Typed Tool plans.
- Typed Skill plans.
- Typed MCP plans.
- Typed attachment plans.
- Typed message plans.
- Typed context plans.
- Typed channel plans.
- Typed port plans.
- Input ports.
- Output ports.
- Streamed ports.
- Parent-child durable messaging.
- Port ownership.
- Port schema validation.
- Port channel authorization.

Provider SDK Tool Artifacts must execute only through typed Model plans. They must never become local Tool WorkItems or fabricated local success.

### Phase B: effects, reliability, reconciliation, compensation, and cleanup

Complete:

- `read`, `idempotent`, `keyed`, and `non-idempotent` contracts.
- Stable keyed idempotency through retry, waits, recovery, and resumption.
- Typed effect state in WorkItems, Attempts, receipts, and errors.
- Partial application facts.
- Reconciliation WorkItems.
- Reconciliation operation facts.
- Reconciliation outcomes: `applied`, `notApplied`, `unknown`.
- Retry only after confirmed non-application.
- Recovery review for unknown results.
- Saga effect tracking.
- Reverse dependency-aware compensation.
- Compensation WorkItems.
- Compensation receipts, failures, and uncertainty.
- Compensation policies `continue` and `stop`.
- Primary failure preservation with typed compensation causes.
- Cleanup allowance separate from normal deadlines.
- Cancellation-safe cleanup.
- No automatic compensation of unknown effects before reconciliation.

### Phase C: structured behavior

Complete:

- Block catch.
- Block finally.
- Cancellation semantics through catch and finally.
- Finalizer primary failure and cause behavior.
- Machines.
- Parallel policies.
- Foreach behavior.
- Repeat behavior.
- Durable Fork WorkItems.
- Durable Join WorkItems.

### Phase D: durable coordination and triggers

Complete:

- Event triggers.
- Schedule triggers.
- Trigger overlap policies.
- Missed-occurrence policies.
- Interval schedules.
- Five-field cron schedules with IANA time zones.
- Calendar gaps.
- Repeated local-time behavior.
- Trigger occurrence facts.
- Subscription retention.
- Subscription capacity and byte limits.
- Event consumption by subscription identity.
- Channel retention and byte limits.
- Channel cursors.
- Emit deduplication.
- Human waits with actor authorization.
- Completion waits.
- Signals with `(source, id)` identity.
- Machine serialization and recovery.
- Timer rearming only after complete restore.

### Phase E: normal Session Runtime v3 capability

Complete:

- Direct Model chat.
- Agent-backed chat.
- Workspace contribution.
- Instruction planning.
- Attachment Runtime admission.
- Local file snapshotting.
- Image snapshotting.
- Directory expansion.
- URL normalization.
- Replay-safe attachment materialization.
- Historical attachment modification handling.
- Provider attachment adaptation.
- Go Tools.
- Provider SDK Tools.
- MCP Tools.
- Skill Tools.
- Service bindings.
- Activity bindings.
- MCP resources.
- MCP prompts.
- MCP Apps.
- MCP approvals.
- App Tool calls.
- App resource reads.
- Approved App context staging.
- Skill sessions.
- Active Skill lifecycle.
- Skill Tool invocation.
- Skill resource handling.
- Provider streaming.
- Protocol outputs.
- Usage.
- Debugger projections.
- Tool result continuation.
- Cancellation.
- Retries.
- Waits.
- Recovery.
- Uncertainty.
- Durable command idempotency.
- Transport-neutral commands and projections.

### Phase F: functional completion audit

Audit every valid Workflow v1 source field in:

- `internal/llmartifactory-go/workflow/contract/v1/types.go`
- `internal/llmartifactory-go/workflow/contract/v1/validation.go`
- `internal/llmartifactory-go/workflow/contract/v1/workflow-v1.schema.json`

For every valid field, prove exactly one result in source:

1. The field compiles into typed runtime behavior.
2. The field is invalid by Workflow language semantics.
3. The field requires an unavailable required Artifact pin, binding, credential, policy, or external domain capability and fails before unrelated external work begins.

Do not leave a valid field as a permanent unsupported runtime feature.

At the end of the implementation pass:

- Update only the phase-wise implementation status sections in the two HLD files.
- Mark completed source behavior with `- [x]`.
- Mark pending source behavior with `- [ ]`.
- Do not mark a task complete from type definitions alone.
- Do not mark a task complete from compiler support alone.
- Do not mark a task complete from one happy-path dispatch route alone.
- A completed task requires compiler, interpreter, persistence, recovery, execution, cancellation, and relevant caller wiring.
- Do not include tests or documentation tasks in the functional completion matrix.
- Do not claim functional completion until every Phase A through Phase F functional task is complete.

Final response format after source changes:

```text
implemented:
- concise source-level facts

files changed:
- exact paths

remaining:
- only actual unavailable input, authorization, external contract, environment, or output-limit blockers

hld status diffs
- mark proper items in hlds complete truthfully.
```

For an implementation or patch response, then end with:

```text
## Implementation status
- [x] evidenced completed work
- [ ] pending work or exact blocker

## Next steps
- [x] completed scope, when applicable
- [ ] remaining implementation work

### Expected user actions
- only patch application, required authorization, exact missing input, or environment-only verification
```

Do not use hostile language. Do not make excuses. Do not use superficial progress claims. Be precise, complete independent work, and preserve the user’s time and financial investment.
