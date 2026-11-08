# Session Runtime v3 and Workflow Runtime v3 Implementation Handoff

## 1. Objective

Implement Session Runtime v3 and Workflow Runtime v3 as one typed durable backend.

The completed backend provides:

```text
Session Runtime v3
    +
Workflow Runtime v3
    =
normal Session v2 backend behavior
    +
full Workflow v1 execution behavior
```

The backend is ready for frontend replacement of Session v2 after the functional implementation phases complete.

All work remains inside the existing Session v3 document and Session Store v3 derived data.

## 2. Source ownership map

| Area                                   | Source directory or file                           |
| -------------------------------------- | -------------------------------------------------- |
| Workflow source contract               | `internal/llmartifactory-go/workflow/contract/v1/` |
| Workflow planning                      | `internal/llmartifactory-go/workflow/`             |
| Session Runtime v3                     | `internal/agentruntime-go/sessionv3/`              |
| Session Runtime v2 reference behavior  | `internal/agentruntime-go/session/`                |
| V3 setup binder                        | `internal/agentruntimesetup/sessionv3runtime/`     |
| V2 setup reference behavior            | `internal/agentruntimesetup/sessionruntime/`       |
| Workflow admission and compiler        | `internal/agentruntimesetup/workflowruntime/`      |
| Model setup                            | `internal/agentruntimesetup/modelruntime/`         |
| Tool setup                             | `internal/agentruntimesetup/toolruntime/`          |
| MCP setup                              | `internal/agentruntimesetup/mcpruntime/`           |
| Skill setup                            | `internal/agentruntimesetup/skillruntime/`         |
| Attachment domain                      | `internal/agentruntime-go/attachment/`             |
| MCP domain                             | `internal/agentruntime-go/mcp/`                    |
| Skill domain                           | `internal/agentruntime-go/skill/`                  |
| Session Store v3                       | `internal/sessionstorev3/`                         |
| Session Store v2 reference behavior    | `internal/sessionstore/`                           |
| Desktop composition and Wails wrappers | `cmd/agentgo/`                                     |
| Frontend adoption                      | `frontend/`                                        |
| Requirements documents                 | `devdocs/requirements/`                            |

`cmd/agentgo` is the application entry point and composition root.

The application uses Wails services as transport adapters. Runtime behavior remains in runtime and setup packages.

No module metadata file is required for this work.

## 3. Delivery form

Implementation output uses plain Markdown.

For source changes:

- Produce apply-ready unified diffs for existing files.
- Produce complete replacement-file contents for deliberate file replacement.
- Identify exact paths.
- Distinguish authored patch content from applied working-tree changes.
- State patch status accurately.
- Preserve repository state across continuation responses.
- Continue immediately when output space ends and substantive implementation remains.

A final implementation response uses this structure:

```text
implemented:
- concise source-level facts

files changed:
- exact paths

remaining:
- only unavailable input, authorization, external contract, environment, or output-limit blockers

hld status diffs
- truthful completed and pending items
```

End an implementation response with:

```text
## Implementation status
- [x] evidenced completed work
- [ ] pending work or exact blocker

## Next steps
- [x] completed scope, when applicable
- [ ] remaining implementation work

### Expected user actions
- patch application, required authorization, exact missing input, or environment-only verification
```

Implementation output contains a large substantive volume of production behavior. Count only additions and modifications that implement runtime behavior, compiler behavior, persistence behavior, recovery behavior, execution behavior, domain binding behavior, or caller wiring.

The implementation scope ends early only when:

- Every requested requirement is complete in the source tree.
- A hard platform output limit ends the response.
- A required source file, required authorization, repository write capability, or external domain contract is unavailable.

Complete all independent work before reporting an unavailable dependency.

## 4. Architecture invariants

### 4.1 Durable root

Session v3 is the only durable root.

No additional Workflow repository, invocation store, execution store, event store, trigger store, or task store is created.

Session Store v3 persists:

- Session documents.
- Session tombstones.
- Derived index data.
- Derived FTS data.

Derived data rebuilds from the authoritative Session document.

### 4.2 Layer boundaries

`internal/llmartifactory-go` owns:

- Declaration syntax.
- Source schema validity.
- Static CEL scope.
- Declaration target relationships.
- Portable declaration validity.

`internal/agentruntimesetup` owns:

- Declaration-to-runtime compilation.
- Artifact pinning.
- Dependency resolution.
- Provider plans.
- Tool binding.
- MCP discovery.
- Skill resolution.
- Workspace planning.
- Attachment materialization.
- Service binding.
- Activity binding.
- External domain execution adaptation.

`internal/agentruntime-go/sessionv3` owns:

- ProgramFrames.
- WorkItems.
- Attempts.
- Waits.
- Timers.
- Recovery.
- Command idempotency.
- Cancellation.
- Durable state.
- Persistence-safe semantics.
- Resource scheduling.
- Budget accounting.
- Channel and port delivery.

Session Runtime v3 remains independent of Artifact packages and setup packages.

### 4.3 Typed state

Use concrete types for:

- IDs.
- Pins.
- Kinds.
- Limits.
- Effects.
- Attempts.
- Waits.
- Receipts.
- Errors.
- Budgets.
- Trigger state.
- Port state.
- Control-flow state.
- Conversation ownership.
- App identity.

Use `CanonicalValue` only at arbitrary JSON boundaries.

### 4.4 Durable external-work sequence

Every external WorkItem follows this order:

1. Persist typed WorkItem intent.
2. Resolve dependencies through the owning setup domain.
3. Verify Artifact pins and current target bindings.
4. Persist semantic binding facts.
5. Persist Attempt facts.
6. Invoke external work.
7. Persist output, receipt, failure, cancellation, or uncertainty.
8. Apply effect-contract retry and recovery behavior.
9. Use Runtime-owned timers for durable timing.
10. Cancel all timers and owned goroutines during `Runtime.Close`.

### 4.5 Workflow target transport

Workflow owns:

- Ports.
- Channels.
- Cursors.
- Parent-child delivery.
- Ordering.
- Retention.
- Durable replay semantics.

Model, Agent, nested Workflow, and explicitly port-aware Service or Activity bindings support typed Workflow transport.

Go Tool, MCP Tool, and Skill Tool domains preserve native contracts.

Go Tool, MCP Tool, and Skill Tool declarations requiring Workflow context, input ports, or streamed output ports fail during Workflow admission before external work starts.

Runtime can route a confirmed final Tool result to a normal non-stream output port.

Workflow transport does not enter native target APIs through:

- Tool argument envelopes.
- Go context sidecars.
- MCP metadata.
- MCP extensions.
- Skill sidecars.
- Provider metadata.
- Domain imports.

Provider SDK Tool Artifacts remain Model-plan-only capabilities.

### 4.6 V2 and V3 isolation

Session v2 behavior remains unchanged.

Session v2 has no dependency on Session v3.

Session v3 becomes a proper backend superset through typed V3 implementation, not by making V2 code depend on V3 code.

## 5. Active functional work

### 5.1 Normal Session v3 parity

V3 needs the complete normal Session v2 configuration and command surface.

The work includes:

- Full typed Model request defaults and clear semantics.
- Full Agent, Workspace, Text, Tool, Skill, MCP, history, metadata, and attachment configuration semantics.
- Typed direct capability selections.
- Per-message configuration overrides.
- Per-continuation configuration overrides.
- Per-execution limits.
- Operation-level resolved configuration facts.
- User-message Skill template preparation.
- Instruction Skill preview.
- Command parity ledger.
- Projection parity ledger.
- Frontend-ready V3 snapshot reads.

Relevant source comparison:

- `internal/agentruntime-go/session/`
- `internal/agentruntimesetup/sessionruntime/`
- `internal/agentruntime-go/sessionv3/`
- `internal/agentruntimesetup/sessionv3runtime/`
- `cmd/agentgo/`

### 5.2 Workflow state and structured-control closure

The current Workflow runtime requires these functional completions:

- Carry nearest state-owner schema and identity through child scope context.
- Support inherited `set` behavior in all nested block forms.
- Add explicit parent-state merge declarations for Parallel and Foreach.
- Persist failed and uncertain step outcomes.
- Add structured retry boundaries for every accepted retry declaration.
- Complete published nested Workflow port schema compatibility.
- Complete target-specific typed Model and Agent invocation adapters.

Relevant source evidence:

- `WorkflowScopeContext` stores `OuterState` in `internal/agentruntime-go/sessionv3/workflow_structured.go`.
- `WorkflowInterpreter.set` uses `program.Root.State` in `internal/agentruntime-go/sessionv3/workflow_program.go`.
- Parallel and Foreach declarations have no parent merge field in `internal/llmartifactory-go/workflow/contract/v1/types.go` and `workflow-v1.schema.json`.
- `WorkflowStepOutcomeStatus` has no failed or uncertain state in `internal/agentruntime-go/sessionv3/workflow_program.go`.
- `workflowStepSupportsRetry` limits retry support in `internal/agentruntimesetup/workflowruntime/compiler_retry.go`.
- `validateWorkflowFramePortBoundary` checks access and channel existence without target channel-schema comparison in `internal/agentruntime-go/sessionv3/workflow_published_ports.go`.
- `workflowInvocationInputs` accepts strict `messages` and `instructions` forms in `internal/agentruntimesetup/sessionv3runtime/inputs.go`.

## 6. Implementation phases

### Phase 1: Complete normal Session Runtime v3 parity

Outcome:

- Session Runtime v3 supports the full normal Session v2 backend capability set through native V3 commands, projections, planning, persistence, recovery, and cancellation behavior.

Implementation sequence:

1. Define V3 typed configuration and override structures.
2. Map every V2 `SessionConfiguration` behavior to typed V3 state.
3. Extend `SessionDefaults` or introduce typed V3 normal invocation options where required.
4. Add per-message and per-continuation override behavior.
5. Add per-execution budget behavior for normal conversations.
6. Persist immutable operation-level configuration and plan facts.
7. Add V3 Skill template preparation and instruction preview operations.
8. Add V3 snapshot projections where frontend reads need consolidated state.
9. Wire all V3 commands and projections through `cmd/agentgo`.
10. Update V3 persistence validation and derived projections.

Required source coverage:

- `internal/agentruntime-go/session/`
- `internal/agentruntime-go/sessionv3/`
- `internal/agentruntimesetup/sessionruntime/`
- `internal/agentruntimesetup/sessionv3runtime/`
- `internal/agentruntimesetup/modelruntime/`
- `internal/agentruntimesetup/toolruntime/`
- `internal/agentruntimesetup/mcpruntime/`
- `internal/agentruntimesetup/skillruntime/`
- `internal/sessionstore/`
- `internal/sessionstorev3/`
- `cmd/agentgo/`

### Phase 2: Complete Workflow state and callable-model semantics

Outcome:

- Every Workflow state scope and callable Model or Agent target has typed, durable, recoverable behavior.

Implementation sequence:

1. Extend Workflow scope state with nearest state-owner identity and schema.
2. Update child ProgramFrame construction for Scope, Choose, Parallel, Foreach, Repeat, Machine, Saga, catch, finally, reconciliation, and compensation.
3. Update StateUpdate WorkItem construction and settlement.
4. Add typed parent-state merge syntax, validation, compiler output, WorkItem boundaries, and runtime settlement.
5. Add failed and uncertain Workflow step outcomes.
6. Define typed Model and Agent invocation-plan adapters.
7. Validate callable input and context transport at admission.
8. Persist complete Model and Agent plans before dispatch.
9. Preserve all behavior through cancellation, recovery, nested frames, and projections.

Required source coverage:

- `internal/llmartifactory-go/workflow/`
- `internal/agentruntimesetup/workflowruntime/`
- `internal/agentruntime-go/sessionv3/`
- `internal/agentruntimesetup/sessionv3runtime/`
- `internal/sessionstorev3/`

### Phase 3: Complete structured retry and boundary reliability

Outcome:

- Every accepted Workflow retry, port, child, effect, and cleanup path has one durable runtime behavior.

Implementation sequence:

1. Add typed retry boundaries for supported structured operations.
2. Preserve WorkItem identity, effect state, attempt identity, deadlines, claims, budgets, and child facts through retry.
3. Align Workflow declaration validation with retry runtime behavior.
4. Complete nested Workflow published-port source-to-target schema checks.
5. Review output-port and stream-port ordering for attached and independent children.
6. Review cancellation of child frames, independent child Executions, waits, retries, resources, cleanup work, and handoffs.
7. Review reconciliation, compensation, and finalizer behavior through restart.
8. Remove reachable permanent unsupported branches from valid Workflow declarations.

Required source coverage:

- `internal/llmartifactory-go/workflow/`
- `internal/agentruntimesetup/workflowruntime/`
- `internal/agentruntime-go/sessionv3/`
- `internal/agentruntimesetup/sessionv3runtime/`
- `internal/sessionstorev3/`

### Phase 4: Complete the source-to-runtime audit

Outcome:

- Every valid Workflow v1 field and every normal Session v2 backend capability has a typed V3 result.

Implementation sequence:

1. Inventory every valid Workflow field from:
   - `internal/llmartifactory-go/workflow/contract/v1/types.go`
   - `internal/llmartifactory-go/workflow/contract/v1/validation.go`
   - `internal/llmartifactory-go/workflow/contract/v1/workflow-v1.schema.json`
2. Inventory every normal Session v2 command, projection, runtime behavior, and domain route.
3. Map every inventory entry to:
   - Compiler behavior.
   - Runtime state.
   - Persistence.
   - Recovery.
   - Cancellation.
   - Binding.
   - Command.
   - Projection.
4. Resolve every compiler, interpreter, binder, and runtime fallback reachable from valid input.
5. Keep unavailable Artifact, binding, credential, policy, and external domain capability failures at admission or binding boundaries.
6. Mark functional backend completion after the ledgers are complete.

Required source coverage:

- `internal/llmartifactory-go/workflow/`
- `internal/agentruntime-go/session/`
- `internal/agentruntime-go/sessionv3/`
- `internal/agentruntimesetup/sessionruntime/`
- `internal/agentruntimesetup/sessionv3runtime/`
- `internal/agentruntimesetup/workflowruntime/`
- `internal/sessionstore/`
- `internal/sessionstorev3/`
- `cmd/agentgo/`

### Phase 5: Adoption, verification, frontend migration, migration tooling, and cleanup

Outcome:

- Frontend traffic uses V3 commands and projections, Session v2 data has a migration path, and retired paths are cleaned up.

Implementation sequence:

1. Add verification coverage for normal V3 parity and Workflow v1 behavior.
2. Update requirements and operational documentation.
3. Move frontend transport to `SessionV3Wrapper`.
4. Add offline Session v2 to Session v3 migration tooling.
5. Decide V2 and Session Store v2 deprecation.
6. Remove retired code after migration completion.
7. Complete aggregate cleanup.

Required source coverage:

- `frontend/`
- `cmd/agentgo/`
- `internal/agentruntime-go/session/`
- `internal/agentruntime-go/sessionv3/`
- `internal/agentruntimesetup/sessionruntime/`
- `internal/agentruntimesetup/sessionv3runtime/`
- `internal/sessionstore/`
- `internal/sessionstorev3/`
- `devdocs/requirements/`

## 8. Engineering rules

- New production behavior uses concrete types and constants end to end.
- error messages use lower-case text.
- No hardcoded strings should be used. proper enums everywhere.
- New domain behavior belongs in the owning domain package.
- Session Runtime v3 remains orchestration-only.
- Setup remains the source of Artifact, credential, provider, native-path, MCP, Skill, Workspace, and binding behavior.
- Wails remains an adapter and composition layer.
- Workflow v1 remains `apiVersion: v1`.
- Workflow ProgramFrame payload remains `artifact.workflow.v1@1`.
- Session v3 remains the only durable root.
- Functional phases change production behavior and caller wiring.
- Adoption phase carries tests, documentation, frontend migration, migration tooling, deprecation, and cleanup. This is NOT part of this implementation phase.

At the end of the implementation pass:

- Update only the phase-wise implementation status sections in the two HLD files.
- Mark completed source behavior with `- [x]`.
- Mark pending source behavior with `- [ ]`.
- Do not mark a task complete from type definitions alone.
- Do not mark a task complete from compiler support alone.
- Do not mark a task complete from one happy-path dispatch route alone.
- A completed task requires compiler, interpreter, persistence, recovery, execution, cancellation, and relevant caller wiring.
- Do not include tests or documentation tasks in the functional completion matrix.

Do not use hostile language. Do not make excuses. Do not use superficial progress claims. Be precise, complete independent work, and preserve the user’s time and financial investment.

Above phases are indicative of full work. current completion state is in HLDs.
Make sure domain related things are in domains and not aggregated as a dump in session folders.

What all is already there:

- the wrappers are FULLY there with you. this is wails app, there are no other entry points other than this. you need to implement things as a wails entry point nad composition only like others. inspect that and then implement.
- the agentruntime-go folder, agentruntimesetup and llmartifactory-go and sessionstorev3 folder are fully provided. there is nothing else other than attached things. you need to complete things in the provided things properly. if anything doesnt exist and you think you need it, that jsut means you need to implement it yourself and then integrate. schema jsons are not attached but assume go represents the schema truthfully. if anything needs to change in jsons mention at last.
- remove all unnecessary validations and duplicate work done making paths hot and diluting ownership. no new validations are to be added if someone down the line already does it. Validations and nil checks are not tasks. only functional work is to be done and nothing else anywhere.
- at last when you update hlds, take a look at phases carefully, if new phases or new work items are discovered you need to update that and then mark them incomplete or complete accordingly.

Final reminder: code delivery in md out. a substantial output is expected. this is financially sensitive endeavor. go top down phase wise. verify each thing, if not complete do it. if already done or implemented by you, go to next. there are lots of minor things you can do if your token quota is incomplete. again very large single shot output is needed. Do not block unnecessarily. out now properly. hlds capture the current state. keep it udpated at last as a diff.
if next step would require files that are not given to you, you need to tell which files exactly. no speculations. all existing code is already attached. so tell accordingly or say what needs to be written where. Follow INSTRUCTIONS Cleanly and FULLY.
