Complete the remaining domain-owned Session Runtime v3 and Workflow setup behavior in one coherent implementation. This is a large single-shot completion task. Do not split it into phases or leave partial adapters.

Scope:

- Keep Artifact Store passive. Do not add Artifact Store imports or source resolution to `internal/agentruntime-go/sessionv3/`.
- Keep Session Runtime v3 as durable orchestration only.
- Keep Model, Agent, Tool, MCP, Skill, Text, attachment, Service, and Activity behavior in their owning setup domains.
- Keep Workflow as orchestration. Do not introduce generic native-domain envelopes.

Required functional completion:

1. Move Workflow Model, Agent, and provider-managed Tool invocation lowering out of `internal/agentruntime-go/sessionv3/workflow_invocation.go`.
   - Workflow execution preparation in `internal/agentruntimesetup/workflowruntime/` must lower supported explicit message, instruction, defaults, Text, Skill, MCP, and provider-managed Tool forms into typed runtime descriptors.
   - Runtime may evaluate typed templates and create durable Activities and child ProgramFrames.
   - Runtime must not decode a generic Workflow JSON envelope to discover Model or Agent message structure, instructions, defaults, or provider request fields.
   - Preserve the rule that arbitrary Workflow JSON never becomes fallback chat text.

2. Complete target transport preparation.
   - Use planning-owned `Inventory.ContextUses` and prepared `WorkflowTargetTransport.ContextForms`.
   - Enforce the declared target context matrix before a `ProgramSnapshot` is accepted.
   - Preserve the existing mode, input-port, final-output-port, and stream-port rules.
   - Do not add another source grammar pass.

3. Complete artifact-backed domain support through typed plans.
   - Model and Agent targets must remain Artifact-backed and use existing setup-domain resolution.
   - Go Tool and MCP Tool calls must retain native argument grammars and domain-owned binding.
   - Provider SDK Tools must remain provider-managed inside a typed Model plan. They must never become local Tool WorkItems or synthetic local Tool results.
   - Skill behavior must remain Artifact-backed and available through typed Model or Agent plans and Skill lifecycle Tools. Do not add a direct Workflow Skill target or a direct capability path.
   - Text must remain a typed Artifact contribution to Model or Agent plans, not a Workflow-native provider request field.
   - Service and Activity remain registered binding targets.

4. Remove ownership duplication created by the old generic invocation decoder.
   - There must be one typed lowering boundary for Workflow Model, Agent, and provider-managed Tool invocation facts.
   - Domain binders must not fall back to current Session defaults when an LLM-origin Tool WorkItem has durable origin facts.
   - Remove redundant declaration-shaped checks only where the ownership move makes them redundant. Do not add validation-only behavior.

Documentation:

- Update `devdocs/requirements/workflow_runtime.md`.
- Change `ARC-TRANSPORT-001` and `ARC-INVOCATION-002` from `Partial` to `Implemented` only after the complete source behavior exists.
- Update the authoritative functional ledger evidence with exact source paths.
- Do not add runtime behavior to `workflow_artifact.md`.

Hard exclusions:

- Do not add tests or test reports.
- Do not implement frontend integration.
- Do not remove, migrate, or clean up SessionV2.
- Do not add or restore direct Tool, Skill, Model, Agent, MCP, Service, Activity, or Workflow capability paths.
- Do not bump any version, payload version, API version, module version, package version, or schema version.
- Do not implement cost metering.

Deliver one complete implementation, not small isolated edits.

---

Complete the remaining Workflow v1 runtime behavior in one coherent implementation. This is a large single-shot functional completion task. Do not split it into phases.

Required functional completion:

1. Repair structured Workflow progression.
   - After successful Scope, Choose, Parallel, Foreach, Repeat, parent-state merge, and successful Saga-body settlement, clear the active parent step boundary.
   - Clear `PendingPath`, `PendingDeclarationPath`, `PendingTarget`, and completed step deadline state at the correct settlement point.
   - Record the parent outcome once, advance the index once, and continue to the next Workflow step.
   - Preserve failure, cancellation, Fork, Join, retry, reconciliation, compensation, and state-merge behavior.
   - Do not make successful structured controls require an external manual continue command.

2. Unify call execution-policy lowering.
   - Apply step claims, step limits, and applicable effect-plan facts to every callable target form.
   - Cover local Tool, MCP Tool, Service, Activity, nested Workflow, Model, Agent, provider-managed Tool, `start`, and `handoff`.
   - Preserve current timeout and retry semantics.
   - Do not make provider-managed Tools local WorkItems.
   - Do not create direct capability targets.

3. Complete planning-owned declaration-path persistence.
   - Preserve the canonical declaration path on every child ProgramFrame.
   - Preserve it on Fork and Join-related work where needed.
   - Persist it on all Workflow outcomes, including structured success, uncertain outcomes, cancellation, machine, Saga, reconciliation, and compensation paths.
   - Keep runtime frame IDs separate from declaration paths.
   - Do not reparse Workflow source to reconstruct paths.

4. Complete scheduled-trigger due-time arming.
   - After a scheduled trigger creates an Execution and its Session mutation is durable, arm execution deadline, cleanup deadline, retry, resource-wait, and wait timers through the existing Runtime-owned due-time lifecycle.
   - Use the existing scheduler registry. Do not create a trigger-specific scheduler.
   - Preserve the separation between direct Workflow start and trigger registration.

5. Remove implementation duplication where it causes divergent behavior.
   - Use shared helpers for child-call policy construction and structured-step finalization.
   - Preserve domain ownership: runtime owns orchestration and durable state; setup owns Artifact and domain binding; Artifact Store remains passive.
   - Do not add new validation-only work. Reduce duplicated checks only when the functional consolidation removes the need for them.

Documentation:

- Update `devdocs/requirements/workflow_runtime.md` after implementation.
- Change `ARC-CALL-POLICY-001`, `ARC-PATH-001`, and `ARC-SCHEDULER-001` to `Implemented` only after all listed behavior exists.
- Change the related functional-ledger rows from `Partial` to `Implemented` only after the complete behavior exists.
- Keep the document marked WIP and retain the existing exclusion statements.

Hard exclusions:

- Do not add tests or test reports.
- Do not implement frontend integration.
- Do not remove, migrate, or clean up SessionV2.
- Do not add direct capability routes.
- Do not bump Workflow, ProgramFrame, Store, API, package, module, or schema versions.
- Do not implement cost metering.

Deliver one complete implementation, not small isolated edits.
