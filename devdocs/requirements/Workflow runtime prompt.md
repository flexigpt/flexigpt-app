Complete Session Runtime v3 Workflow support in the FlexiGPT repository.

This is a backend/runtime task. Do not implement frontend adoption, v2-to-v3 migration tooling, or deletion of v2/legacy paths unless an existing source change requires a compatibility adapter. Preserve existing durable payload compatibility.
Treat the source as authoritative.

Primary goal:

Implement the complete Workflow Runtime v3 backend through a typed,
runtime-neutral compiled Workflow program. Artifact declaration parsing and
Artifact pinning must stay outside Session Runtime. Session Runtime must execute
only a typed compiled IR and must not import Artifact or LLM Artifactory
packages.

Non-goals:

- No frontend v3 adoption.
- No offline v2-to-v3 migration implementation.
- No deletion of v2 Session Runtime, v2 Store, or legacy interpreter support.
- No new Workflow persistence root, Workflow store, trigger store, or event
  store outside the existing v3 Session document.
- No broad compatibility rewrite of existing v2 Session behavior.

Read these sources first:

Dependency and style constraints:

- `.golangci.yml`

Session v3 runtime and persistence:

- `internal/agentruntime-go/sessionv3/`
- `internal/sessionstorev3/`

Workflow Artifact declaration, planning, and validation:

- `internal/llmartifactory-go/workflow/`
- `internal/llmartifactory-go/workflow/contract/v1/`
- `internal/artifactsetup/registration/llm_declarations.go`
- `internal/artifactsetup/topology/contract.go`
- `internal/artifactsetup/topology/contract_topology.yaml`

Workflow setup boundary:

- `internal/agentruntimesetup/workflowruntime/`
- `internal/agentruntimesetup/sessionv3runtime/`

Domain runtime adapters:

- `internal/agentruntimesetup/modelruntime/`
- `internal/agentruntimesetup/toolruntime/`
- `internal/agentruntimesetup/mcpruntime/`
- `internal/agentruntimesetup/skillruntime/`
- `internal/agentruntimesetup/attachmentruntime/`

Bootstrap and transport:

- `cmd/agentgo/app.go`
- `cmd/agentgo/wrapper_sessionv3_bootstrap.go`
- `cmd/agentgo/wrapper_sessionv3_runtime.go`

Behavioral reference only, not a dependency target:

- `internal/agentruntime-go/session/`
- `internal/agentruntimesetup/sessionruntime/`

Required architecture:

1. Dependency direction

   `internal/agentruntime-go/...` must not import:
   - `internal/artifactory-go/...`
   - `internal/llmartifactory-go/...`
   - `internal/agentruntimesetup/...`
   - `cmd/...`

   `internal/llmartifactory-go/...` must not import Agent Runtime.

   The only layer allowed to depend on both Artifact-domain contracts and
   Agent Runtime contracts is `internal/agentruntimesetup/...`.

   Keep this direction valid under the `depguard` rules in `.golangci.yml`.

2. Typed compiler boundary

   Workflow source and Artifact concerns stay here:
   - `internal/llmartifactory-go/workflow/contract/v1`
   - `internal/llmartifactory-go/workflow`
   - `internal/agentruntimesetup/workflowruntime`

   Session execution stays here:
   - `internal/agentruntime-go/sessionv3`

   `agentruntimesetup/workflowruntime` must compile an admitted
   `workflowv1.WorkflowDocument` plus resolved Artifact pins into a typed,
   runtime-neutral `sessionv3.WorkflowProgram` or equivalent typed IR.

   Do not place `workflowv1.WorkflowDocument`, `workflowv1.Value`,
   `workflowv1.Target`, Artifact refs, Artifact services, or declaration
   locators in `internal/agentruntime-go/sessionv3`.

   Do not place Session mutation, ProgramFrame advancement, WorkItem state,
   retry state, Wait state, or CEL evaluation execution in
   `agentruntimesetup/workflowruntime`.

3. Durable versioning

   Workflow declaration language version and runtime interpreter version are
   different things.
   - `apiVersion: v1` remains the Workflow declaration language.
   - Existing durable snapshots using `artifact.workflow.v1@1` must remain
     readable while migration is not authorized.
   - Do not change the meaning or payload shape of an existing interpreter
     version in place.
   - Introduce a new typed interpreter payload version, for example
     `artifact.workflow.v1@2`, for the new compiler output.
   - Keep legacy interpreter support as a compatibility path only.
   - New Workflow admission must emit only the current typed interpreter
     version after it is implemented.

4. Typed values and schemas

   Use proper typed runtime structures.
   - Use typed `WorkflowProgram`, `WorkflowBlock`, `WorkflowStep`,
     `WorkflowTarget`, `WorkflowValue`, `WorkflowState`, and typed WorkItem
     payloads.
   - `sessionv3.CanonicalValue` is allowed only at generic JSON boundaries:
     user values, JSON Schema documents, expression literal values, generic
     Tool payloads, generic external outputs, and persisted arbitrary values.
   - Do not add `json.RawMessage` fields to new runtime models.
   - Do not persist `map[string]any` as a semantic runtime model.
   - Decode CanonicalValue to a concrete type at each typed boundary.
   - Use `jsonutil` in one conversion helper per boundary. Do not scatter
     direct JSON marshal/unmarshal calls through interpreter logic.
   - Schema compilation belongs in runtime only after Artifact setup has
     expanded declaration-local `$defs` into a runtime-neutral schema value.
   - Do not persist compiled CEL programs, compiled JSON Schemas, clients,
     callbacks, function pointers, cancellation functions, authorization
     tokens, or native paths.

5. Validation discipline

   Do not add redundant validation layers.

   Validation ownership is:
   - Workflow contract package:
     source syntax, declaration shape, static expression scope, static target
     relationships, portable schema validity.
   - Workflow compiler in setup:
     conversion from declaration to typed IR, supported-operation check,
     durable Artifact pins, Service/Activity binding availability, typed IR
     integrity.
   - Session v3 core:
     ProgramSnapshot and ProgramState integrity, state/input/output boundary
     schemas, command revision checks, WorkItem lifecycle invariants.
   - Domain executors:
     current executable pin, current MCP discovery identity, credentials,
     policy, approval, and external invocation behavior.

   Do not re-run full declaration validation before every WorkItem.
   Do not add blanket string trimming, duplicate identity checks, arbitrary
   caps, or generic validation helpers unless they protect a real boundary or
   a durable invariant.

   Unsupported Workflow operations must fail at Workflow admission with a
   typed, explicit unsupported-operation error. Never accept an operation and
   silently ignore it, convert it to a no-op, or fail only after external work
   begins.

6. Runtime binding and privacy

   Follow this durable ordering:
   1. Persist typed WorkItem intent.
   2. Resolve current runtime dependencies.
   3. Verify semantic Artifact or runtime pins.
   4. Persist durable semantic binding facts.
   5. Persist dispatched Attempt state.
   6. Dispatch external work.
   7. Persist confirmed output, failure, receipt, or uncertainty.

   In-memory private state belongs only in BoundWork/private runtime maps:
   - OAuth and MCP approval tokens.
   - MCP approval IDs.
   - clients and transports.
   - Skill Runtime session IDs.
   - function pointers.
   - compiled CEL and schema objects.
   - cancellation functions.

   It must never enter ProgramSnapshot, ProgramState, WorkItem, Attempt,
   Wait, SessionStore v3, events, or Wails projections.

7. Domain executors

   Keep runtime behavior in its domain runtime package.
   - Model runtime owns provider configuration and credential resolution.
   - Tool runtime owns Go Tool target binding and Tool output conversion.
   - MCP runtime owns MCP Tool discovery, policy, approval, invocation,
     resources, prompts, and Apps.
   - Skill runtime owns Skill resolution, allow-lists, Skill sessions, and
     Skill Tool invocation.
   - Workflow runtime setup owns Artifact Workflow admission and typed IR
     compilation.
   - Session v3 owns only orchestration and durable state transitions.

   Do not create interface-on-interface wrappers around concrete domain
   runtimes. Existing `ProgramInterpreter`, `WorkBinder`, and external
   Service/Activity binding interfaces are sufficient boundaries.

8. Workflow target behavior

   Complete target support in this order:
   - Tool Artifact target:
     compile an immutable Tool Artifact pin; bind it through Tool Runtime at
     dispatch; verify current identity and definition digest.
   - MCP Tool target:
     compile an immutable MCP server Artifact pin plus declared operation
     name; discover and bind the current provider Tool at dispatch; preserve
     policy, digest, approval, and uncertainty behavior.
   - Service and Activity:
     use the existing BindingRegistry; bind stable binding ID and version at
     admission; execute private binding state through workflowruntime.Binder.
   - Nested Workflow:
     admit a pinned child ProgramSnapshot; preserve cycle detection;
     propagate typed input and child outcome.
   - Model and Agent:
     do not silently serialize arbitrary JSON into a chat message. Implement
     them only after the typed Model and Agent execution-plan compiler exists.
   - Dynamic target sets:
     support only when the runtime has a typed resolver WorkItem and a
     persisted selected target identity. Do not use strings alone as a
     durable dynamic target binding.

9. Workflow implementation order

   Implement in dependency order.

   A. Stabilize the typed compiler boundary.
   - Preserve legacy interpreter compatibility.
   - Add current typed interpreter version.
   - Compile only the operations actually implemented.
   - Verify no Artifact import enters Session Runtime.

   B. Complete the sequential typed core.
   - `when`
   - `transform`
   - `set`
   - `call`
   - `return`
   - `fail`
   - root input/output/state schema boundaries
   - nested Workflow child settlement
   - Tool, MCP Tool, Service, Activity target binding

   C. Complete core WorkItem lifecycle before fan-out.
   - deadline handling
   - retry scheduling
   - cancellation settlement
   - panic recovery
   - uncertain external effects
   - recovery review waits
   - typed output and receipt propagation

   D. Add Agent, Model, Workspace, Skill, and MCP preparation plans.
   - Agent contribution plan.
   - Workspace contribution plan.
   - Model request patch plan.
   - instruction and text contribution plan.
   - Tool/Skill/MCP merged plan.
   - immutable source pins and operation fingerprint.
   - no Session coordinator logic that reimplements domain resolution.

   E. Add structured Workflow control flow.
   - `scope`
   - `choose`
   - `repeat`
   - `parallel`
   - `foreach`
   - `fork`
   - `join`
   - `break`
   - `continue`
   - child block output and scoped state semantics

   F. Add durable coordination.
   - human waits
   - completion waits
   - timers
   - event subscriptions
   - channel logs and cursors
   - `emit`
   - execution signals
   - state machines
   - trigger scheduling and overlap policy

   G. Add effects and reliability.
   - effect kinds
   - stable idempotency keys
   - reconciliation
   - saga compensation
   - resource claims
   - budgets and limits
   - cleanup allowance

10. Runtime correctness rules

- Do not use `time.Sleep` for durable timer waits.
- All timers and goroutines must be cancelable by Runtime.Close.
- Use the existing runtime lifecycle context and background panic boundary.
- Do not create an unbounded global cache of compiled CEL or schemas.
- If caching is necessary, scope it to runtime lifecycle or execution
  lifecycle and explicitly evict it.
- Preserve original external-effect uncertainty after restart.
- Never replay a non-idempotent dispatched WorkItem automatically.
- A WorkItem effect contract controls retry and recovery safety.
- A Provider, MCP, Tool, Skill, or Service failure must not fabricate a
  successful output.
- A selected Tool result must be represented once and consumed once.
- App and approval private references must be invalid after recovery unless
  the runtime can reconstruct them safely.
