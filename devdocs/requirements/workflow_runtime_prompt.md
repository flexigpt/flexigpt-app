Complete the Runtime and associated setup functionality specified by the
`Functional tasks` section at the end of
`devdocs/requirements/workflow_runtime.md`.

That heading-based section is the sole current completion ledger. It replaces
all former completion tables, pending ledgers, status registers, architecture
status tables, and repeated component claims.

Deliver one substantial end-to-end functional implementation. Do not split the
work into phases, status-only edits, isolated helpers, or a partial milestone.
Complete every dependency-ready parent and subtask in this delivery.

Scope exclusions are mandatory:

- Do not add, run, or report tests.
- Do not add validation-only hardening, validation reports, malformed-input
  cleanup, or a separate validation project.
- Do not implement frontend integration.
- Do not remove SessionV2, migrate SessionV2, convert stored Sessions, or do
  V2 retirement cleanup.
- Do not change any version identifier.
- Do not add release work, release notes, or cost metering.
- Do not add direct Model, Agent, Tool, MCP, Skill, Service, Activity, or
  Workflow capability paths.
- Do not reopen the prior V2 direct Tool analysis.

No version identifier may change. This includes Workflow apiVersion, interpreter
or payload versions, Artifact and Store schemas, package versions, module
versions, public APIs, and release identifiers.

Use the current HLD as the source of truth. A parent task is complete only when
every listed subtask is complete. Change a status to `Implemented` only when the
complete behavior exists in source. Replace the current evidence with exact
source evidence when completion is real.

Complete these parent tasks and every listed subtask:

- `RT-SETTLE-003`
- `RT-CLEANUP-003`
- `RT-BUDGET-003`
- `RT-TRIGGER-003`
- `RT-COMMAND-003`
- `RT-PERSIST-003`
- `WF-PREPARE-003`
- `WF-RETRY-003`
- `WF-CHANNEL-003`
- `WF-EFFECT-003`
- `WF-CONTROL-003`
- `WF-OUTCOME-003`
- `V3-REPLAY-003`
- `V3-CONFIG-003`
- `DOMAIN-SKILL-003`
- `V3-CONTINUE-003`
- `WF-HANDOFF-003`
- `ARC-DOMAIN-003`
- `V3-API-003`
- `DOMAIN-ATTACH-003`
- `ARC-DUP-003`

Implement the following functional outcomes.

Runtime settlement and lifecycle:

- Use durable identity reacquisition after any mutation-capable operation that
  can invalidate Session-owned slice-backed objects.
- Route binding error, port error, panic, invalid wait result, manual result,
  cancellation, reconciliation, child completion, and ordinary completion
  through one terminal Runtime lifecycle.
- Settle Attempt and effect facts, release or transfer claims, wake completion
  and resource waiters, advance owners, settle Executions, checkpoint, then
  launch all newly ready Session work.
- Do not leave catch, finally, parent completion waits, resource waiters,
  forwarded channel work, or cross-Execution work idle after committed
  settlement.

Cleanup, Saga, budget, and scheduling:

- Stop normal work after cancellation or normal deadline expiry.
- Dispatch eligible cleanup through its effective cleanup deadline across
  binding, execution, retry, resource acquisition, waits, emits, streams, and
  settlement.
- Preserve cleanup classification through synthetic frames, including retry
  wrappers.
- Keep cancellation as the primary outcome.
- Retain a Saga while late external effect confirmation can still require
  reconciliation or compensation. An empty initial compensation list is not
  proof that no later compensation is required.
- Wake concurrency waiters after every counted-slot release.
- Apply `step.Limits` to all structured constructs.
- Distinguish nested Workflow depth from internal Runtime frame depth.
- Persist outer child-call Attempt, policy, accounting, and ownership before a
  child can dispatch native work.
- Drain eligible trigger queues on registration reactivation.
- Use one event projection with `type`, source, identity, data, correlation,
  and time for trigger, wait, and Machine expressions.

Command and persistence semantics:

- Return accepted command receipts before repeated attachment admission or
  Workflow preparation is required.
- Preserve caller payload identity across prepared snapshots.
- Keep server-generated occurrence time outside caller command identity.
- Do not move Workflow preparation into Runtime.
- Treat a committed tombstone as committed if derived-index deletion fails.
- Schedule derived-index repair without restoring a deleted Session.
- Permit deletion of an idle long-lived normal conversation while protecting
  actual active Workflow, child, wait, cleanup, and external work.

Workflow preparation, calls, channels, and transport:

- Retain complete prepared target meaning through neutral immutable snapshots.
- Preserve transitive prepared Agent, Model, Text, Tool, Skill, MCP, Workspace,
  nested Workflow, and requirement facts without storing private clients,
  credentials, or Artifact Store execution handles.
- Resolve an owner Model only for provider-managed Tool lowering.
- Capture a complete logical call before dispatch or child materialization.
- Retain target, target-set membership, input, context, typed messages,
  instructions, port inputs, effect key, policy, and logical time.
- Reuse the capture for retry, started calls, recovery, and resumption.
- Preserve retry schema matching, effect-safe replay decisions, and native
  Model and Agent failure classification.
- Keep private nested channel storage names separate from declaration-visible
  aliases.
- Capture an authorized channel view at every actual step and block-output
  evaluation boundary.
- Preserve inherited subscription ownership and consumption identity.

Effects, recovery, control flow, and outcomes:

- Explicitly map source `non-idempotent` to Runtime `nonIdempotent`.
- Settle MCP results from actual domain evidence. Do not make every confirmed
  MCP success uncertain.
- Use the same settled effect facts for Tool continuation, Workflow outcome,
  history, reconciliation, retry, and Saga recording.
- Preserve causal effect dependencies across ordered and structured work.
- Provide complete original and compensation operation facts.
- Apply Repeat index to `while` and `until`.
- Settle empty Foreach and other empty collections truthfully.
- Preserve omitted versus explicit zero maxima.
- Limit consecutive immediate Machine transitions without limiting long-lived
  event-driven Machines.
- Record actual terminal step boundaries for inline failure, return, break,
  continue, cancellation, reconciliation, and child settlement.
- Preserve partial assistant output once without inventing success.

Conversation, replay, configuration, Skill, continuation, and handoff:

- Preserve semantic LLM dispatch plans before clearing private bindings during
  recovery.
- Rebuild recovered requests from captured message identities, causal order,
  effective non-secret Model and Provider facts, instructions, capabilities,
  and context.
- Recreate credentials and clients privately. Do not substitute current
  defaults, overlays, or contribution assembly.
- Add durable collection policy for inherit, replace, explicit empty clearing,
  supported removal, and override semantics.
- Preserve explicit MCP selection and apply the effective Workspace.
- Preserve truthful Agent, Workspace, Session, and invocation contribution
  provenance.
- Reconstruct Skill active state for the exact owning ProgramFrame and enabled
  set, then release private Skill sessions at natural frame and Execution
  completion.
- Resume staged continuation into the active owning Agent operation instead of
  queueing it behind that Agent.
- Scope manual Tool controls and continuation to the current unconsumed
  operation.
- Preserve queued handoff message defaults, budget, completed assistant rounds,
  and causal history.
- Add typed successful handoff completion or relinquish behavior with restore or
  retain. Do not use cancellation as success.

Domain ownership, binder composition, attachments, and duplication:

- Move Agent Model selection and Agent contribution semantics into Agent setup.
- Move Workspace contribution semantics into Workspace setup.
- Move Text contribution preparation into Text setup.
- Move attachment admission and materialization into Attachment setup.
- Move native Model request, output, and Model-default adaptation into Model
  setup.
- Keep Session and Workflow setup as compositional consumers of neutral domain
  facts.
- Forward `PrepareUserMessageSkillTemplate` and `PreviewInstructionSkill`
  through the final composed Workflow binder. Reuse the Base implementation.
- Preserve attachment labels and root-relative directory identity in provider
  content while replaying admitted snapshots only.
- Retain one canonical immutable effective plan with truthful provenance.
- Replace full per-Tool plan copies with immutable canonical origin-plan
  references plus necessary Tool-specific facts.
- Retain one authoritative attachment content snapshot.
- Consolidate duplicate Runtime evaluation, failure matching, error cloning,
  operation-input extraction, lowered-tree traversal, namespacing, retention,
  terminal settlement, and ready-work launch behavior.

Architecture constraints are mandatory:

- Core Runtime owns durable orchestration and must not import Artifact Store or
  setup.
- Artifact Store remains a passive source-fact provider.
- Setup owns preparation and composition, not Runtime scheduling or recovery.
- Domain setup owns native grammar, private resources, invocation, and result
  conversion.
- Workflow remains an orchestration language. It must not become a generic
  native-domain envelope.
- Provider SDK Tools remain typed Model-plan capabilities.
- Native Tool arguments remain native domain arguments.
- Keep one Runtime due-time owner. MCP connection and discovery timers remain
  MCP-domain lifecycle behavior.
- Do not add another durable execution root or another orchestration path.
- Do not add repeated admission or a separate validation project.

Preserve already working behavior:

- Artifact-independent core Runtime.
- Direct Workflow composition rejection.
- Typed prepared Workflow commands.
- Provider-managed SDK Tool lowering.
- Registered Service and Activity binding behavior.
- Published-port validation.
- Forward-only started-call wrappers.
- Repeated calendar local-slot identity.
- Immutable attachment content snapshots.
- Typed Model defaults and Store-owned semantic export.
- Shared V3 MCP content conversion.

Update `devdocs/requirements/workflow_runtime.md` in the same delivery:

- Keep `Functional tasks` as the only current status ledger.
- Update only the completed parent and subtask headings.
- Change status to `Implemented` only after complete source behavior exists.
- Replace current evidence with exact source evidence.
- Remove obsolete contradictory claims if implementation makes them inaccurate.
- Retain the WIP statement, scope exclusions, no-version-change rule, ownership
  constraints, and V2 coexistence rule.

Return:

- Completed source changes.
- Updated HLD task statuses and exact source evidence.
- Any genuine remaining in-scope blocker, including the exact unavailable
  input, authorization, or environment limitation.
- Do not stop at a milestone while a dependency-ready task remains unfinished.
