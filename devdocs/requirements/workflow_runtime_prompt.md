Complete the Runtime and associated setup functionality defined by the source-audited current completion register in devdocs/requirements/workflow_runtime.md.

This is a WIP implementation. No version identifier may be bumped anywhere. This includes Workflow apiVersion, interpreter and payload versions, Artifact and Store schemas, packages, modules, public APIs, application versions, and release identifiers.

Deliver one substantial end-to-end functional implementation. A sequence of small helper edits, status-only changes, isolated fixes, or a phase plan is not the requested outcome. Complete every dependency-ready task in this delivery.

Scope exclusions are explicit:

- Ignore tests. Do not add or run tests as part of this task.
- Ignore all validation work, validation reports, validation-only hardening, and separate validation cleanup.
- Ignore frontend integration.
- Ignore SessionV2 removal, replacement cleanup, migration, and stored-session conversion.
- Ignore version bumps, release work, release notes, and cost metering.
- Do not reopen the previously documented V2 direct Tool capability analysis or add direct executable capability paths.

Implement every open parent and subtask in the current register:
RT-SETTLE-003, RT-CLEANUP-003, RT-BUDGET-003, WF-PREPARE-003, WF-EFFECT-003, WF-RETRY-003, V3-REPLAY-003, V3-CONFIG-003, DOMAIN-SKILL-003, WF-CHANNEL-003, WF-CONTROL-003, WF-HANDOFF-003, V3-CONTINUE-003, WF-OUTCOME-003, RT-TRIGGER-003, RT-COMMAND-003, RT-PERSIST-003, V3-API-003, DOMAIN-ATTACH-003, ARC-DOMAIN-003, and ARC-DUP-003.

The required outcome is functional completeness, not more defensive layers.

Architecture ownership is mandatory:

- Runtime owns durable orchestration, interpretation, scheduling, logical evaluation boundaries, settlement, recovery, cancellation, and lifecycle notification.
- Core Runtime must not depend on Artifact Store or setup.
- Artifact Store remains a passive source-fact provider. No Runtime preparation, binding, dispatch, scheduling, recovery, or consumer-private API belongs there.
- Owning domains own their native request grammar, contribution semantics, private resources, invocation, and result conversion.
- Agent owns Agent contribution preparation and Agent Model selection.
- Workspace owns Workspace contribution and selection semantics.
- Attachment owns admission and content materialization.
- Model setup owns native Model request and result adaptation.
- MCP, Skill, Tool, Text, Service, and Activity own their respective native behavior.
- Session and Workflow remain composition and orchestration, not alternative implementations of domain behavior.

Complete the shared Runtime lifecycle, not only its most visible call sites. All terminal paths must retain authoritative Session-owned mutations, settle Attempts and effects, release or transfer claims, wake completion and resource waiters, advance owners, persist, and launch newly ready work. Include binding failures, port failures, panic settlement, manual results, invalid domain wait results, cancellation, reconciliation, independent children, and cross-Execution channel delivery.

Make cleanup work executable through its effective cleanup deadline from binding through settlement. Retain cleanup classification through synthetic and retry frames. Stop normal work. Preserve primary cancellation. Keep Sagas available for late confirmed effects, including when the initial compensation list is empty.

Correct effect translation and MCP settlement. Translate source non-idempotent explicitly. Preserve actual domain certainty instead of marking every MCP execution uncertain. Complete compensation dependencies and original-operation facts.

Capture one logical call before dispatch or child materialization. Retain evaluated target, membership, input, context, typed messages, instructions, port inputs, key, time, and policy. Retry and started work must reuse these facts rather than evaluate them again after settlement.

Complete actual LLM recovery. Preserve the semantic dispatch plan before clearing private state. Reconstruct the exact captured semantic request and effective non-secret Model and Provider facts without current-default or current-contribution substitution. Credentials and live resources remain private.

Complete durable configuration policy, including inheritance, explicit empty replacement, removal, and specific-over-inherited precedence. Preserve MCP selections and apply the effective selected Workspace. Retain truthful contribution provenance.

Complete invocation-owned Skill state reconstruction and natural owner cleanup. Complete channel source aliases, per-boundary authorized snapshots, inherited subscription identity, Repeat condition index, empty collection settlement and merge, explicit maximum values, and long-lived Machine behavior.

Complete staged continuation into the active owning operation, queued handoff message facts and history, successful handoff completion with restore or retain, current-operation manual Tool controls, complete step outcomes, interrupted assistant output, trigger queue reactivation, consistent event type, stable command replay, deletion commitment, idle conversation deletion, optional backend capability forwarding, and semantic attachment labels.

Remove duplication as part of the functional implementation:

- One canonical immutable effective plan with accurate owner provenance.
- Immutable origin-plan references rather than full per-Tool plan copies.
- One authoritative attachment content snapshot.
- Shared Runtime expression evaluation, failure matching, error cloning, operation-input extraction, lowered-tree traversal, and equivalent retention behavior.
- One authoritative Runtime settlement and ready-work launch lifecycle.
- One Agent Model-selection implementation.
- No new orchestration path and no second Workflow scheduler.

Keep admission authoritative. Ownership consolidation must reduce repeated admission and redundant downstream validation passes where the consolidation makes them unnecessary. Do not add checks, change validity policy, disable essential behavior indiscriminately, or turn this into a validation project.

Use the supplied source evidence and functional results in the HLD. Existing helper presence is not completion of its parent behavior. Preserve already working provider-managed Tool configuration, prepared Workflow command separation, published-port bridging, immutable attachment content, and domain-native argument paths.

Update the HLD in the same delivery:

- Mark each parent and subtask Implemented only after its complete behavior exists.
- Replace required-result text with exact source evidence.
- Synchronize the mapped older statuses.
- Remove obsolete contradictory completion statements and redundant status authority.
- Keep workflow_artifact.md focused on declaration, source identity, static relationships, storage, retrieval, and read-only planning. Runtime usage mechanics belong only in workflow_runtime.md.
- Retain the WIP statement, all scope exclusions, and the prohibition on version bumps.

Return the completed functional changes, the updated task statuses, exact source evidence, and any genuine remaining in-scope blocker. Do not stop at a milestone or leave an available independent task unfinished.
