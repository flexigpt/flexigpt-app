Complete the FlexiGPT Session Runtime v3 and Workflow Runtime v3 backend.

Complete the full backend scope in this assignment as much as possible: contracts, compiler, interpreter, lifecycle, domain binders,
transport-neutral commands/projections, persistence integration, and cleanup of V3 duplicate logic.

The current repository source is authoritative. Inspect the current source before changing it.

Use dependency order internally, but complete all dependency-ready work in this assignment. If an actual
external blocker exists, complete all independent work and report the exact blocking contract, source path, and smallest required input. Do not treat
ordinary implementation scope as a blocker.

Primary completion outcome:

1. Session Runtime v3 is a functional backend superset of normal Session v2
   behavior.

2. Workflow Runtime v3 fully executes every valid Workflow v1 declaration
   feature with typed durable semantics.

3. Workflow v1 does not retain permanent “unsupported because not yet
   implemented” behavior at completion. Admission rejects only:
   - Invalid declarations.
   - Unsatisfied required Artifact pins or bindings.
   - Current policy, credential, approval, or external availability failures.
   - Behavior outside Workflow v1.

4. Session v2 and Session Store v2 remain shipped and unchanged. Do not break,
   delete, migrate, or broadly rewrite them.

5. Do not increment any versions.
   - `apiVersion: v1` remains Workflow declaration language v1.
   - `artifact.workflow.v1@1` remains the current typed development payload.
   - Do not add `artifact.workflow.v1@2`.
   - Session Runtime v3 and Workflow Runtime v3 are development features, so
     the current v1 payload may be replaced in place.
   - This does not change shipped V2 payload compatibility.

Explicit non-goals:

- No frontend Session v3 adoption.
- No offline v2-to-v3 migration implementation.
- No deletion of V2 Session Runtime, V2 Store, or legacy interpreter paths.
- No new Workflow persistence root, Workflow store, trigger store, or event
  store outside the existing v3 Session document.
- No broad compatibility rewrite of existing V2 Session behavior.

Required source material to inspect or attach:

Repository-wide:

- `.golangci.yml`

Session Runtime v3:

- `internal/agentruntime-go/sessionv3/`
- `internal/sessionstorev3/`
- `internal/agentruntime-go/inference/`
- `internal/agentruntime-go/attachment/`
- `internal/agentruntime-go/session/`
- `internal/sessionstore/`

Workflow declaration and planning:

- `internal/llmartifactory-go/workflow/`
- `internal/llmartifactory-go/workflow/contract/v1/`
- `internal/llmartifactory-go/core/composition/`
- `internal/llmartifactory-go/core/declaration/`
- `internal/artifactsetup/registration/llm_declarations.go`
- `internal/artifactsetup/topology/contract.go`
- `internal/artifactsetup/topology/contract_topology.yaml`

Workflow and Session v3 setup:

- `internal/agentruntimesetup/workflowruntime/`
- `internal/agentruntimesetup/sessionv3runtime/`
- `internal/agentruntimesetup/modelruntime/`
- `internal/agentruntimesetup/toolruntime/`
- `internal/agentruntimesetup/mcpruntime/`
- `internal/agentruntimesetup/skillruntime/`
- `internal/agentruntimesetup/attachmentruntime/`

Bootstrap and transport:

- `cmd/agentgo/app.go`
- `cmd/agentgo/wrapper_sessionv3_bootstrap.go`
- `cmd/agentgo/wrapper_sessionv3_runtime.go`
- all current Session and command wrappers used by AgentGo

Behavioral reference only:

- `internal/agentruntime-go/session/`
- `internal/agentruntimesetup/sessionruntime/`
- `internal/sessionstore/`

Read these requirements:

- `devdocs/requirements/Workflow Model .md`
- `devdocs/requirements/Workflow Runtime.md`

Architecture and dependency direction:

`internal/agentruntime-go/...` must not import:

- `internal/artifactory-go/...`
- `internal/llmartifactory-go/...`
- `internal/agentruntimesetup/...`
- `cmd/...`

`internal/llmartifactory-go/...` must not import Agent Runtime.

Only `internal/agentruntimesetup/...` may depend on both Artifact-domain
contracts and Agent Runtime contracts.

Keep this valid under `.golangci.yml` depguard rules.

Required ownership:

- Workflow contract package:
  declaration syntax, schema validity, static expression scope, static target
  relationships, portable schema validity.

- Workflow compiler in setup:
  declaration-to-typed-IR conversion, supported-operation compilation,
  Artifact pins, Service/Activity binding availability, target plans, typed IR
  integrity.

- Session v3 core:
  ProgramSnapshot and ProgramState integrity, ProgramFrame advancement,
  WorkItem lifecycle, attempts, waits, command revisions, durable recovery,
  typed input/output/schema boundaries.

- Model runtime:
  provider configuration, credential resolution, typed model request plans,
  provider execution preparation.

- Tool runtime:
  Go Tool binding, pin verification, invocation, output conversion, Tool
  schema conversion.

- MCP runtime:
  MCP discovery, current identity verification, policy, approvals,
  invocation, resources, prompts, Apps, App context, uncertainty behavior.

- Skill runtime:
  Skill resolution, allow-lists, runtime sessions, Tool invocation, resource
  handling.

- Attachment runtime:
  attachment admission, semantic snapshotting, replay-safe materialization,
  provider adaptation.

- Workflow runtime setup:
  Artifact Workflow admission, pinning, target binding preparation, typed IR
  compilation.

- Session v3:
  orchestration only. It must not reimplement Artifact resolution, provider
  configuration, MCP discovery, Skill resolution, or attachment file access.

Typed runtime model requirements:

Use concrete types for all runtime-owned semantics:

- WorkflowProgram
- WorkflowBlock
- WorkflowStep
- WorkflowTarget
- WorkflowCallableInterface
- WorkflowState
- WorkflowStepOutcome
- Workflow scope, branch, loop, machine, saga, and compensation state
- WorkItem payload unions
- Attempt, retry, deadline, wait, receipt, error, effect, budget, resource
  claim, target-selection, and signal state
- Artifact pins, binding IDs, versions, and fingerprints

Use `sessionv3.CanonicalValue` only at arbitrary JSON boundaries:

- User input and arbitrary Workflow output.
- JSON Schema documents.
- CEL expression literal values and results.
- Schema-defined state values.
- Generic Tool arguments and external output.
- Event, channel, correlation, metadata, and human-response values.
- Generic ProgramSnapshot payload and ProgramState data envelopes.

Do not use `CanonicalValue` for IDs, enums, kinds, target selections, pins,
effect contracts, retry policies, wait state, receipt fields, errors, limits,
budgets, binding facts, or known request/response types.

Do not add `json.RawMessage` fields to new runtime models.
Do not persist `map[string]any` as a semantic runtime model.
Do not persist compiled CEL programs, compiled JSON Schemas, clients,
callbacks, tokens, function pointers, cancellation functions, browser state,
or native paths.

Use `jsonutil` in one conversion helper per boundary. Decode a generic
CanonicalValue into a concrete domain type exactly at the domain boundary.
Do not scatter direct JSON marshal/unmarshal through interpreter logic.

Completion requirements:

A. Repair and verify the current baseline first.

- Fix all compile, race, lint, depguard, and malformed partial-patch defects if any.
- Audit every currently applied Workflow compiler path for fields that are accepted but ignored.
- Preserve all correct current work while repairing it.

B. Complete Session Runtime v3 normal backend behavior.

Implement typed V3 equivalents for all normal V2 backend behavior:

- Direct Model chat.
- Agent-backed chat.
- Workspace contribution and instruction planning.
- Model default layering and typed request patches.
- Conversation history and continuation behavior.
- Provider streaming, protocol outputs, usage, and debugger projection.
- Attachment admission, snapshots, replay, and provider content adaptation.
- Go Tool, provider SDK Tool, MCP Tool, Skill Tool, Service, and Activity
  behavior.
- MCP resources, prompts, Apps, approvals, App Tool calls, resource reads, and
  approved App context.
- Skill sessions and active-skill lifecycle.
- Cancellation, waits, recovery, uncertainty, retries, and command
  idempotency.

A functional V3 superset does not require V3 to read V2 durable documents or
share V2 Go types. It requires no loss of normal V2 backend capability.

C. Complete the Workflow v1 compiler and all callable targets.

Implement every valid Workflow v1 target and call mode:

- Tool Artifact target.
- MCP Tool target.
- Service target.
- Activity target.
- Nested Workflow target.
- Agent target.
- Model target.
- Static dynamic target sets.
- Resolver-backed dynamic target sets through typed resolver WorkItems and
  persisted selected target identity.
- `invoke`, `start`, and `handoff`.
- Attached and independent child lifetime.
- Explicit call context transfer.
- Typed ports and streams.

Do not serialize arbitrary Workflow JSON into a chat message for Agent or
Model calls. First implement typed Agent, Workspace, Model, instruction,
Tool, Skill, MCP, attachment, message, and port execution plans. Then create
the appropriate child Agent-loop or single-model ProgramFrame.

Provider-managed SDK Tool Artifacts must not fabricate local success. They must
either execute through their typed model plan or fail before external work
begins.

D. Complete all Workflow v1 operations.

Implement typed compiler IR and Session v3 interpreter semantics for:

- `call`
- `transform`
- `set`
- `scope`
- `choose`
- `parallel`
- `foreach`
- `repeat`
- `machine`
- `wait`
- `emit`
- `saga`
- `cancel`
- `return`
- `fail`
- `break`
- `continue`

Implement all associated declaration fields:

- `when`
- `timeout`
- `retry`
- `effect`
- `claims`
- `limits`
- step output schemas
- block state, output, listen, catch, finally
- target interfaces
- target sets
- events
- channels
- ports
- triggers
- resources
- root limits

For each operation, persist typed semantic state that permits deterministic
resume, recovery, cancellation, and inspection.

E. Complete WorkItem lifecycle before or together with fan-out.

Implement:

- Persisted WorkItem intent before dependency resolution.
- Current dependency resolution.
- Current pin and semantic binding verification.
- Persisted semantic binding facts.
- Persisted dispatched attempt before external dispatch.
- Confirmed output, receipt, failure, or uncertainty persistence.
- Deadlines covering retries and waits.
- Retry matching, backoff, jitter, scheduling, and persisted retry state.
- Effect-safe automatic retry.
- Cancellation settlement.
- Panic recovery.
- Recovery review waits.
- No automatic replay of non-idempotent uncertain work.
- Reconciliation before retry or compensation where required.
- Typed output and receipt propagation.
- Runtime.Close cancellation of every timer and goroutine.
- No `time.Sleep` for durable timing.

F. Complete structured control flow.

Implement durable typed semantics for:

- Nested scopes and block outputs.
- Catch and finally behavior.
- Choice routing.
- Parallel branch selection, isolation, completion policies, cancellation, and
  collected errors.
- Foreach collection capture, item schemas, ordering, concurrency, completion
  policies, and isolated item state.
- Repeat count, while, until, ongoing lifetime, carried state, exhaustion, and
  iteration identity.
- Fork and join WorkItems.
- Break and continue.
- State merge rules after isolated branch and item execution.

G. Complete durable coordination.

Implement:

- Human waits.
- Completion waits.
- Timer waits.
- Event subscriptions.
- Event retention and consumption semantics.
- Local channel logs and cursors.
- Emit receipts and deduplication.
- Execution signals.
- Machines and serialized transitions.
- Event triggers.
- Schedule triggers.
- Overlap and missed-occurrence policy.
- Trigger lifecycle without a new persistence root.

H. Complete effects and reliability.

Implement:

- Read, idempotent, keyed, and non-idempotent effect contracts.
- Stable idempotency keys.
- Reconciliation.
- Saga compensation.
- Compensation ordering and error policy.
- Resource claims.
- Concurrency and rate limits.
- Invocation and child budgets.
- Cleanup allowance.
- Typed uncertainty outcomes and recovery review.

I. Remove V3 duplicate fallback logic after domain paths are proven.

The V3 Session setup package must not reimplement domain behavior. Remove or
replace duplicate fallback paths from:

- `internal/agentruntimesetup/sessionv3runtime/go_exec.go`
- `internal/agentruntimesetup/sessionv3runtime/go_tools.go`
- `internal/agentruntimesetup/sessionv3runtime/mcp_exec.go`
- `internal/agentruntimesetup/sessionv3runtime/mcp_tools.go`
- `internal/agentruntimesetup/sessionv3runtime/skill_exec.go`
- `internal/agentruntimesetup/sessionv3runtime/skill_tools.go`

Keep V2 legacy behavior intact. Do not make V2 depend on V3.

J. Do not create redundant wrappers or persistence roots.

- Do not create another Session v3 Store wrapper.
- `SessionV3Wrapper` already owns Runtime and Store lifecycle for Wails.
- Do not create Workflow, Trigger, Event, or Invocation persistence roots
  outside the V3 Session document.

Documentation work:

Update these documents truthfully as part of the implementation:

- `devdocs/requirements/Workflow Model .md`
- `devdocs/requirements/Workflow Runtime.md`

The completed documentation must state:

- V3 backend completion means a functional V2 backend superset plus full
  Workflow v1 support.
- V2 remains preserved.
- Frontend adoption and migration remain separate work.
- CanonicalValue is limited to arbitrary JSON boundaries.
- Typed domain ownership and persistence constraints remain mandatory.

Delivery requirements:

- Complete contracts, implementation, callers, wiring.
- Do not leave a new partial milestone as the final result.
- Do not defer ordinary implementation work to another handoff.
- Provide unified diffs for modified files.
- Provide full contents for new files.
- Track authored, applied, and verified state separately.
- State exact external blockers only if they are real and evidenced.
- End with implementation status and next steps.
