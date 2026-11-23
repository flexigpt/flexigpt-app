Complete the remaining Runtime and setup functional work in one substantial delivery.

Scope:

- Implement `RT-SETTLE-005`, `WF-PREPARE-005`, `WF-RETRY-005`, and `ARC-DUP-005` from `devdocs/requirements/workflow_runtime.md`.
- Update the affected HLD statuses and source evidence in the same delivery. Mark a task `Implemented` only when its complete listed behavior exists in source.
- Keep `workflow_artifact.md` focused only on Artifact admission, retrieval, identity, closure, Inventory, and read-only planning. Do not add Runtime execution, setup, binding, dispatch, scheduling, recovery, or consumer-private concerns to that HLD.

Required functional outcomes:

- Make terminal WorkItem, frame-cancellation, child, wait, resource, effect, Saga, and Execution settlement one pre-checkpoint mutation sequence. Remove the post-checkpoint `settlePendingFrameCancellationsLocked` mutation path from normal execution. `persistLocked` must remain the only normal committed ready-work launch authority.
- Make Workflow preparation retain complete immutable Model and Agent contribution meaning before Runtime receives the Workflow Program snapshot. Render instruction-mode Skills through the Skill domain during preparation and persist the rendered text in `WorkflowPreparedSkill.InstructionContent`. Capture MCP contribution facts that affect the final Workflow Model or Agent prompt and provider Tool plan. Runtime binding and recovery can reconstruct private clients, connections, credentials, and sessions only from durable semantic facts. They must not reselect changed contribution meaning.
- Keep domain ownership strict:
  - Artifact Store owns Artifact facts only.
  - Workflow setup owns Workflow plan lowering and execution preparation.
  - Runtime owns Session, ProgramFrame, WorkItem, scheduler, settlement, recovery, and persistence.
  - Model, Tool, MCP, Skill, Attachment, Service, and Activity domains own native request grammar, native resources, invocation, and result conversion.
  - Workflow remains orchestration only.
- Freeze all retry input-port values and cursors at the original logical call boundary. Preserve them through retry, delayed child creation, recovery, and resumption. Never add channel records that arrived after the original boundary.
- Remove duplicated lowered Workflow tree traversal. Use one typed lowered-tree traversal mechanism for channel namespacing, resource namespacing, and Runtime declaration-path application. Do not move Workflow source parsing, Artifact planning, or Artifact Store access into Runtime.
- Remove redundant lifecycle and preparation algorithms rather than adding parallel code paths.
- Do not add validation work. Do not add repeated admission or validation checks outside their owning domain. Reduce redundant checks only when that does not change valid functional behavior.
- Use named `const` declarations for stable string values. Use existing typed enums or add typed enums for domain states and modes. Do not add scattered hardcoded protocol, identifier, or error strings. Error strings must be lowercase.
- Preserve current version identifiers. This is WIP. Do not change Workflow `apiVersion`, interpreter versions, payload versions, Store schema versions, public API versions, package versions, module versions, or release identifiers.

Explicit exclusions:

- Do not add or modify tests, test fixtures, test reports, validation reports, validation-only hardening, or test execution work.
- Do not implement frontend integration, Wails transport migration, or frontend behavior.
- Do not implement SessionV2 deletion, SessionV2 cleanup, stored-session conversion, SessionV2 migration cleanup, or release work.
- Do not implement cost metering.
- Do not add direct capability paths.

Delivery:

- No tools will be attached.
- Follow the separately supplied code-delivery instructions.
- Return one complete implementation response only.
- Return only contextual unified diffs in the required format. Do not return prose, summaries, phased plans, or partial patches.
