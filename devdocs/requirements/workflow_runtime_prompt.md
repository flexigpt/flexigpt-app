## Corrected remaining functional work

| Functional work                                                                | Why it is required                                                                                                                   | Owner                                          | Handoff  |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------- | -------- |
| Reconstruct completed normal model and Tool rounds on later turns              | Normal local Tool continuation currently sends Tool output without the preceding assistant Tool-call representation.                 | Session orchestration and provider integration | Prompt 2 |
| Reuse prior semantic context where it belongs in later conversation history    | Workspace, MCP, and other prepared semantic context can affect the original turn but is not reconstructed for later turns.           | Session orchestration                          | Prompt 2 |
| Consume staged App context into the next continuation message                  | The stage path and input-rendering path exist but are disconnected.                                                                  | Session orchestration and MCP integration      | Prompt 2 |
| Preserve or derive active Skill semantics across later turns and recovery      | Active Skills affect future prompt and Tool behavior. The semantic selection must survive without persisting a Skill Runtime handle. | Skill domain and Session orchestration         | Prompt 1 |
| Hydrate Skill rules and prompt before model execution                          | V2 does this. V3 exposes Skill Tools but omits the Skill prompt behavior.                                                            | Skill domain integration                       | Prompt 1 |
| Honor Agent `IncludeSystemPrompt` behavior                                     | Agent configuration currently cannot control selected Model prompt inclusion in V3.                                                  | Agent and Model integration                    | Prompt 1 |
| Record Workflow `start` receipts                                               | The runtime already creates the receipt but does not retain it in the parent outcome.                                                | Workflow Runtime                               | Prompt 2 |
| Keep direct Workflow start separate from trigger registration                  | Direct execution currently activates triggers despite the HLD contract.                                                              | Workflow and Session orchestration             | Prompt 2 |
| Preserve trigger preparation defaults where target preparation needs them      | Trigger preparation drops supplied invocation defaults.                                                                              | Workflow preparation                           | Prompt 2 |
| Rearm retry delays and correct nested retry behavior                           | Persisted retry work can remain stranded after restore; valid one-attempt nested retry declarations are rejected.                    | Runtime and Workflow orchestration             | Prompt 2 |
| Make Workflow logical time Runtime-owned and unify due-time lifecycle behavior | Runtime-owned timing is required for durable continuation and consistent restore behavior.                                           | Runtime and Workflow orchestration             | Prompt 2 |

No direct Tool, direct Skill, direct Model, direct Agent, direct MCP, direct Service, direct Activity, direct Workflow, cost meter, or RunLimit compatibility work belongs in the current handoffs.

## Corrected implementation handoff prompt 1

Implement the V3 domain-support completion work in one large pass.

Work in:

- `internal/agentruntimesetup/sessionv3runtime/`
- `internal/agentruntimesetup/{skillruntime,toolruntime,mcpruntime}/`
- `internal/agentruntime-go/sessionv3/types.go`
- `internal/agentruntime-go/skill/`
- `devdocs/requirements/workflow_runtime.md`

Required outcomes:

- Complete V3 Skill behavior.
  - Persist or deterministically reconstruct semantic enabled and active Skill selections that affect future conversation behavior.
  - Keep Skill Runtime session IDs, registries, handles, clients, and callbacks process-private.
  - Recreate private Skill Runtime sessions from semantic Skill selections after recovery.
  - Hydrate Skill rules and `SkillsPrompt` before each effective V3 model invocation.
  - Preserve immutable LLM-origin Skill selections for generated Skill Tool calls.
  - Do not substitute current Session defaults for retained origin facts.

- Honor Agent `IncludeSystemPrompt`.
  - Consume the Agent capability fact in V3 LLM plan composition.
  - Preserve current Session override precedence.
  - Keep Agent and Model behavior in their owning adapters.

- Preserve completed Tool semantics for later-turn reconstruction.
  - Reuse existing durable provider-neutral structured output and completed Tool facts where possible.
  - Preserve Tool call identity, provider call identity, choice identity, text, structured result, image, and file semantics needed by Prompt 2.
  - Do not add a separate direct-capability path.
  - Do not persist Tool clients, private bindings, callbacks, active runtime handles, or partial execution state.

- Consolidate only the shared domain behavior touched by this work.
  - Reuse common Tool-output, Skill-selection, and MCP-output adaptation where it prevents V2 and V3 functional drift.
  - Do not remove, migrate, or clean up SessionV2.
  - Do not create a third parallel adapter path.

- Update only the relevant HLD status entries after implementation.

Constraints:

- Implement source changes, but do not add or modify tests.
- Do not perform validation work or validation-only changes.
- Do not integrate frontend work.
- Do not remove, migrate, or clean up SessionV2.
- Do not add direct Tool, Skill, Model, Agent, MCP, Service, Activity, or Workflow capabilities.
- Keep executable capabilities Artifact-backed or registered-binding-backed.
- Do not implement cost metering.
- Do not bump versions or perform release work.
- Keep the work WIP.
- Keep domain concerns in domain, Runtime concerns in Runtime, workflow and session concerns in orchestration, and artifact responsibilities in artifact storage and retrieval.
- Do not move Runtime execution behavior into the Artifact layer.
- Treat Session as a reproducible semantic checkpoint, not a live Runtime snapshot.
- Complete all feasible items in one large pass.

## Corrected implementation handoff prompt 2

Implement the V3 Runtime and Workflow orchestration completion work in one large pass.

Work in:

- `internal/agentruntime-go/sessionv3/`
- `internal/agentruntimesetup/sessionv3runtime/`
- `internal/agentruntimesetup/workflowruntime/`
- `internal/sessionstorev3/` only where a completed semantic checkpoint fact must be represented
- `cmd/agentgo/wrapper_sessionv3_*.go`
- `devdocs/requirements/workflow_runtime.md`

Required outcomes:

- Complete later-turn conversation reproduction.
  - Reconstruct provider input from completed semantic model and Tool facts.
  - Preserve the relation between an assistant Tool call and its Tool result.
  - Reuse `LLMOutput.ProtocolOutputs`, completed Tool outputs, Activity identity, Tool call identity, and semantic dispatch facts.
  - Reconstruct only completed semantic conversation facts.
  - Do not persist raw live provider requests, provider clients, credentials, callbacks, active streams, timers, private bindings, or partial in-flight state.

- Complete App-context continuation behavior.
  - Transfer staged `Execution.Continuation.AppContext` into the next consumed semantic message exactly once.
  - Ensure provider input construction reads that message context.
  - Preserve approved semantic context only. Do not persist browser state, App runtime handles, or MCP clients.

- Complete Workflow result and trigger behavior.
  - Store the already-created `start` receipt with the parent step outcome.
  - Remove trigger registration from direct `StartWorkflow`.
  - Keep explicit trigger registration as the only trigger lifecycle operation.
  - Preserve invocation defaults during trigger-template preparation where target preparation requires them.

- Complete retry and due-time behavior.
  - Rearm ordinary `RetryWaiting` work after restore.
  - Permit valid nested Workflow calls with one-attempt retry declarations.
  - Make Workflow logical time Runtime-owned.
  - Make wait, retry, resource, execution, cleanup, and trigger due-time behavior restore and cancel through one Runtime-owned lifecycle.
  - A single lifecycle owner does not require serializing timer objects or cancellation functions.

- Retain planning-owned declaration paths only where existing Runtime behavior requires durable authorization or outcome trace identity.
  - Do not turn this into a validation pass.
  - Reuse the existing planning-owned declaration-path model.

- Update relevant HLD status entries truthfully after implementation.

Constraints:

- Implement source changes, but do not add or modify tests.
- Do not perform validation work or validation-only changes.
- Do not integrate frontend work.
- Do not remove, migrate, or clean up SessionV2.
- Do not add direct Tool, Skill, Model, Agent, MCP, Service, Activity, or Workflow capabilities.
- Do not implement cost metering.
- Do not bump versions or perform release work.
- Keep the work WIP.
- Remove duplication rather than creating parallel paths.
- Keep domain concerns in domain, Runtime concerns in Runtime, workflow and session concerns in orchestration, and artifact responsibilities in artifact storage and retrieval.
- Do not move Runtime execution behavior into the Artifact layer.
- Treat Session as a reproducible semantic checkpoint, not a live Runtime snapshot.
- Complete all feasible items in one large pass.

## Deferred and excluded items

- Cost budgeting is `Not applicable — Deferred` for this implementation cycle.
- Tests and validation work are excluded.
- Frontend integration is excluded.
- SessionV2 removal, cleanup, migration, and stored-session conversion are excluded.
- Version, release, and release-note work are excluded.
