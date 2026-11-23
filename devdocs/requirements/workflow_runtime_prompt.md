Complete every `Partial` and `Not implemented` item in the `Current task
record` in `devdocs/requirements/workflow_runtime.md`.

This is one substantial, connected, end-to-end functional delivery. Do not
split the work into phases, status-only edits, small helper changes, or later
milestones. Implement the complete behavior and then update the HLD task
statuses and completion evidence in the same delivery.

This is WIP. Do not bump or change any version identifier. This includes
Workflow apiVersion, ProgramSnapshot or ProgramFrame interpreter versions,
Artifact and Store schema versions, package or module versions, public API
versions, and release identifiers.

Explicit exclusions:

- Do not add or change tests.
- Do not add validation work, validation-only hardening, malformed-input
  handling work, or validation reports.
- Do not do frontend integration.
- Do not delete, retire, migrate, convert, or clean up SessionV2.
- Do not implement stored-session conversion.
- Do not do release work.
- Do not implement cost metering, pricing, currency, or cost settlement.
- Do not add direct Tool, Skill, Model, Agent, MCP, Service, Activity, or
  Workflow capability paths.

Implement the following connected functional results.

- Establish one Runtime-owned committed lifecycle for every terminal WorkItem
  path. Settle Attempt and effect evidence, release or transfer claims, wake
  completion and resource waiters, advance owners, settle Executions, commit
  the Session, then launch all dependency-ready work. Handle ordinary commits,
  committed derived-index errors, prior checkpoint-pending state, cancellation,
  deadlines, panic, binding failure, port failure, retry, resource waits,
  reconciliation, compensation, child completion, and recovery.

- Reacquire Session-owned objects by durable identity after any call that can
  append, replace, forward, resolve, restore, create child frames, create child
  Executions, or otherwise invalidate slice-backed pointers.

- Remove pre-commit work launch from port and channel helpers. Remove repeated
  direct ready-work launches where the common post-commit lifecycle can own the
  behavior. Preserve harmless dispatch guards during transition, but do not
  retain duplicated scheduling algorithms.

- Complete cancellation and bounded cleanup. Preserve cleanup classification
  and effective deadlines through Workflow, retry, Agent-loop, single-Model,
  Machine, merge, Fork, Join, waits, resources, streams, emits, settlement,
  and recovery. Support failure-driven cleanup as well as cancellation and
  ordinary deadline cleanup. Do not terminalize a frame or Execution while
  dispatched owned work, late effect evidence, reconciliation, compensation,
  claim release, or completion delivery remains active.

- Complete recovery as operation resumption. Reconstruct eligible
  pre-invocation domain waits from captured semantic requests without
  persisting tokens, clients, callbacks, or private bindings. Route recovery
  decisions through the original WorkItem owner, catch, finally, cleanup,
  reconciliation, compensation, and dependent completion lifecycle. Finish
  restore with one committed Session-wide readiness pass.

- Preserve one immutable prepared Workflow invocation through invoke, start,
  retry, handoff, nested Workflow preparation, Agent rounds, and recovery.
  Preserve planned source pins, selected contribution facts, Workspace prompt
  and context behavior, target-set facts, typed messages, instructions,
  defaults, ports, effect policy, and actual owner provenance. Do not resolve a
  different current Agent or Workspace graph after preparation.

- Add typed lowering and capture for expression-backed message and instruction
  collections. Evaluate them at the logical call boundary. Do not turn
  arbitrary Workflow JSON into fallback Model or Agent chat text.

- Capture complete logical call values before child materialization or dispatch,
  including actual port input values and cursors. Retry, started work,
  reconciliation, and recovery must use this captured operation and must not
  observe later channel content or later state.

- Make channel and subscription coordination authoritative. Preserve declared
  aliases separately from private nested channel names. Use the actual
  invocation Execution for every evaluation boundary. Re-read durable event
  consumption after each wait settlement. Permit only one consumption for the
  same originating subscription owner.

- Complete scoped budget behavior. Preserve Execution aggregate accounting and
  controlling structured-scope accounting. Mark real nested Workflow calls as
  Workflow nesting. Account user Activities, trigger signals, channel records,
  forwarded ports, staged inputs, output ports, and supplied provider usage
  against their actual semantic owner. Keep cost metering excluded.

- Preserve target-owned effect guarantees. Distinguish unconditional
  idempotence from keyed-only deduplication. Treat keyed work as replay-safe
  only when it has a stable captured key and the target supports a
  target-native keying mechanism. Do not add generic Workflow metadata to Go
  Tool, MCP Tool, Skill Tool, Model, Agent, Service, or Activity request
  grammars.

- Preserve causal effect dependencies through ChildCall boundaries, structured
  exits, reconciliation, and compensation. Persist complete compensation facts,
  including identities, before compensation code consumes `completed`.

- Complete structured control behavior. Preserve child state, causal facts,
  claims, and outcomes through break, continue, block return, Workflow return,
  Saga exit, and cancellation. Persist actual completion order for
  first-completed behavior. Keep one selected winner while retaining all
  branch or item outcomes.

- Preserve truthful terminal meaning. Keep typed errors, categories, causes,
  receipt evidence, effect state, outputs, and declaration paths through
  WorkItems, ProgramFrames, Executions, completion waits, projections, retry,
  and catch. Separate interrupted and replayed LLM streams. Persist one
  interrupted assistant Activity with its actual terminal state.

- Reconstruct exact retained LLM inputs and final non-secret configuration.
  Use captured message ordering, assistant protocol facts, Tool results,
  attachments, App context, injected context, and final effective Model
  configuration. Reconstruct only private credentials, clients, and private
  Skill resources. Never substitute current Session defaults, current history,
  current Agent contributions, current Workspace contributions, or later
  operation results.

- Complete continuation routing. Route public and staged continuation to one
  active semantic conversation owner, including Workflow handoff ownership.
  Consume or durably transfer every accepted queued Activity before an Agent
  frame completes. Scope manual Tool controls and automation holds to the
  active operation and current Tool batch.

- Move domain behavior to the owning domain:
  - Agent setup owns Agent capability interpretation and Model selection.
  - Workspace setup owns Workspace contribution, override, prompt, and context
    interpretation.
  - Attachment setup owns V3 attachment admission and immutable snapshot
    creation.
  - Model setup owns final provider input, output, and Model-default
    adaptation.
  - Tool, MCP, Skill, Service, and Activity setup own native request
    construction, private resources, native invocation, and native result
    conversion.
  - Session and Workflow setup compose neutral domain facts and lower only
    orchestration intent.
  - Core Runtime owns durable orchestration only.

- Remove duplicated execution algorithms and replicated effective state. Keep
  one canonical immutable LLM dispatch plan and canonical origin-plan
  references for generated Tool WorkItems. Do not reintroduce current-default
  lookup or full per-Tool plan copies.

- Keep Artifact Store passive. Do not add Runtime preparation, execution,
  binding, dispatch, scheduler, persistence, recovery, or consumer-private
  APIs to Artifact Store. Do not move Runtime behavior into
  `workflow_artifact.md`.

- Reduce redundant validation only by retaining a single existing owner for
  admission or domain checks. Do not create a validation project and do not add
  repeated source admission checks.

- Preserve V2 coexistence. Do not remove or alter V2 behavior as part of this
  implementation cycle.

- Preserve backend setup ownership. If Runtime restore succeeds but final
  wrapper publication fails, close the restored Runtime and preserve the
  existing Runtime-before-Store shutdown order.

Update `devdocs/requirements/workflow_runtime.md` as part of the delivery:

- Replace each pending completion-evidence field with exact source paths,
  symbols, and connected behavior.
- Change a task to `Implemented` only after every listed result is present.
- Update parent status only after all subtasks are complete.
- Remove obsolete earlier gap text and obsolete false completion claims for
  corrected task IDs.
- Do not modify `workflow_artifact.md` unless a separate demonstrated
  Artifact-planning defect exists.

The delivery is complete only when no in-scope task in the current task record
remains `Partial` or `Not implemented`.
