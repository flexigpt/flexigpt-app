## Prompt 1: Complete target-domain and setup support

```text
Complete the full target-domain and setup support work in one coherent implementation change set.

Read and use:
- `devdocs/requirements/workflow_runtime.md`
- `devdocs/requirements/workflow_artifact.md`
- the Functional completion ledger
- the Architecture ownership and duplication register

This is work in progress. Do not bump any version identifier anywhere. Do not change Workflow apiVersion, ProgramFrame payload or interpreter versions, Artifact or Store schema versions, package or module versions, or public API versions.

Out of scope:
- Tests, test execution, and new validation work.
- Frontend or Wails integration.
- Session v2 deletion, cleanup, migration, or stored-session conversion.

Do not return a narrow patch, partial scaffold, or implementation plan. Complete every feasible target-domain item below in one large coherent source change. Update the applicable HLD status rows from Open or Partial to Implemented only after the complete behavior exists.

Required behavior:

1. Direct capability support
- Carry direct capability evidence from Workflow planning through execution preparation and runtime-neutral program data.
- Route direct Model, Agent, Tool, Service, and Activity targets through their owning setup-domain adapter.
- Do not fabricate an Artifact, Source, Definition, revision, or Artifact pin for a direct capability.
- Do not add a generic Workflow fallback that converts direct capability input into native request text or envelopes.

2. Callable capability requirements
- Make callable interface capability requirements operational during execution preparation.
- Keep root Workflow requirements separate from callable interface capability requirements.
- Resolve current domain capability facts through the owning adapter and persist the resulting runtime-neutral facts for dispatch, retry, recovery, and resumption.

3. Model and Agent Workflow target transport
- Consume `WorkflowInvocationTargetCapabilities`; do not leave them as unused declarations.
- Preserve only explicit Model and Agent invocation forms: messages, instructions, defaults, final output ports, streamed output ports, and Agent handoff.
- Propagate accepted final and streamed PortPlans into Model and Agent child ProgramFrames before child work begins.
- Do not accept Model or Agent input ports.
- Do not convert arbitrary Workflow JSON into fallback chat text.

4. Provider-managed SDK Tool support
- Lower a provider SDK Tool Artifact only through an owning typed Model child plan.
- Keep provider Tool execution inside the Model request plan.
- Do not create a local Tool WorkItem, local retry target, synthetic Tool result, or fabricated local success.
- Preserve provider-managed Tool output semantics for Workflow calls.

5. Go Tool, MCP Tool, Skill Tool, Service, Activity, Text, and attachment boundaries
- Make each domain own its native request grammar, current readiness, native invocation, and result conversion.
- Make setup preparation enforce the actual target transport behavior before WorkItem creation.
- Do not allow unsupported Tool context, input-port, or streamed-port behavior to be silently accepted or converted into a synthetic stream record.
- Keep confirmed final Tool output delivery in Runtime orchestration.
- Make Service and Activity transport capability facts available during preparation, not only after a WorkItem exists.
- Keep URL, text, attachment, Skill, MCP, and Tool behavior in their owning domain packages. Workflow only registers or invokes typed domain bindings.

6. Immutable origin facts for generated Tool WorkItems
- Persist or reference immutable effective LLM invocation configuration, capability facts, selected bindings, and origin facts for every Tool WorkItem emitted from LLM output.
- Ensure Tool binders use those retained facts rather than current Session defaults.
- Remove duplicate Model dispatch-plan processing and duplicate target preparation paths where one authoritative typed plan can be used.

Architecture constraints:
- Session Runtime v3 owns durable Session state, WorkItem lifecycle, attempts, waits, retries, cancellation, recovery, scheduling facts, and checkpointing.
- Workflow owns orchestration only.
- Domain packages own native domain behavior.
- Core `sessionv3` must not import Artifact Store, setup, provider-client, MCP-client, credential, or native-handle packages.
- Remove duplicate static declaration interpretation where it is redundant. Keep only the minimum durable integrity behavior in Runtime and current readiness behavior in setup.
- Do not add new broad validation layers.

Documentation deliverables:
- Update `workflow_runtime.md` Functional completion ledger and Architecture ownership and duplication register.
- Mark only fully completed items as Implemented.
- Keep incomplete items as Open or Partial with precise evidence.
- Keep the Artifact HLD Artifact-focused.

Produce the full implementation, affected source changes, and HLD status updates in one response. If a narrow set of things remain pending, concretely give the pending items at last and mark the HLD status as such.
```

## Prompt 2: Complete Workflow orchestration and durable runtime behavior

```text
After the target-domain and setup support work is complete, complete the remaining Workflow orchestration and durable Runtime behavior in one coherent implementation change set.

Read and use:
- `devdocs/requirements/workflow_runtime.md`
- the Functional completion ledger
- the Architecture ownership and duplication register
- all Workflow interpreter, Runtime v3, Store v3, and Workflow setup source

This is work in progress. Do not bump any version identifier anywhere. Do not change Workflow apiVersion, ProgramFrame payload or interpreter versions, Artifact or Store schema versions, package or module versions, or public API versions.

Out of scope:
- Tests, test execution, and new validation work.
- Frontend or Wails integration.
- Session v2 deletion, cleanup, migration, or stored-session conversion.
- Replacing domain-owned native behavior with Workflow-specific logic.

Do not return a narrow patch, partial scaffold, or implementation plan. Complete every feasible Workflow and Runtime item below in one large coherent source change. Update the applicable HLD status rows from Open or Partial to Implemented only after the complete behavior exists.

Required behavior:

1. Trigger lifecycle separation
- A direct Workflow start creates one Execution only.
- Only explicit trigger registration creates, activates, changes, or rearms trigger lifecycle state.
- Trigger registration persists an input-unbound Program template.
- Direct invocation input must never become trigger template input.

2. Durable declaration identity
- Preserve the planning-owned declaration path on every root and child ProgramFrame.
- Preserve it on WorkItems, PortPlans, target facts, and every WorkflowStepOutcome, including skipped, started, cancelled, structured-control, nested-child, retry, reconciliation, compensation, and saga outcomes.
- Keep runtime paths separate from declaration paths.

3. Runtime-owned logical time
- Capture initial Workflow logical time from the Runtime-owned clock before initial expression evaluation.
- Eliminate direct process-clock reads from Workflow semantic initialization and deterministic semantic receipts.
- Persist and reuse logical time across recovery and resumption.

4. Retry, timers, scheduler, and recovery
- Rearm every durable retry delay after restore.
- Make a valid nested Workflow retry policy with `maxAttempts: 1` execute once without rejection.
- Preserve correct retry semantics for nested Workflow calls, retry boundaries, effect safety, deadlines, cancellation, and recovery.
- Consolidate duplicated timer scheduling behavior into one Runtime-owned scheduler lifecycle for waits, retries, resource waits, execution deadlines, cleanup deadlines, and trigger schedules.
- Preserve durable due-time facts only. Do not persist timer objects, callbacks, or contexts.

5. `start`, handoff, cancellation, and child settlement
- Persist the `start` completion handle and start receipt in the parent step outcome.
- Preserve independent child ownership, budget, cancellation relationship, and completion semantics.
- Preserve Agent handoff owner, saved owner, target Agent pin, completion behavior, and durable result facts.
- Ensure cancellation uses durable ownership boundaries and does not erase uncertainty.

6. Effects, reconciliation, compensation, and saga behavior
- Complete and audit the entire saga, reconciliation, compensation, retry, and cleanup path as one behavior.
- Preserve primary failure or cancellation while attaching compensation causes.
- Reconcile uncertain effects before automatic compensation where required.
- Preserve confirmed application, confirmed non-application, uncertainty, reconciliation review, compensation ordering, compensation concurrency, and cleanup allowance through restore and cancellation.
- Do not create synthetic success, confirmed non-application, or compensation facts.

7. Remove orchestration duplication
- Remove duplicate state transitions, duplicate scheduler pathways, and duplicate declaration-shaped interpretation that are no longer needed after the durable Runtime owns the behavior.
- Retain Runtime integrity checks only where they protect durable state.
- Do not reintroduce Artifact resolution, provider configuration, credentials, native requests, Tool invocation grammar, MCP sessions, or other domain behavior into Session Runtime or Workflow interpreter code.

Architecture constraints:
- Session Runtime v3 owns durable orchestration and persistence-safe semantics.
- Workflow owns declaration-level orchestration and typed control flow.
- Domain binders own native execution.
- Core `sessionv3` must remain independent of Artifact Store and setup imports.
- Workflow must not become a second Model, Agent, Tool, MCP, Skill, Service, or Activity implementation.

Documentation deliverables:
- Update `workflow_runtime.md` Functional completion ledger and Architecture ownership and duplication register.
- Mark only fully completed items as Implemented.
- Keep incomplete items as Open or Partial with exact remaining behavior.
- Do not add implementation status to `workflow_artifact.md` beyond Artifact planning concerns.

Produce the full implementation, affected source changes, removal of redundant pathways, and HLD status updates in one response. If a narrow set of things remain pending, concretely give the pending items at last and mark the HLD status as such. Any additional files needed for next stage should be explicitly stated at end.
```

## Remaining items after both implementation prompts

Assuming both prompts fully implement their listed scope, these items still remain outside the implementation cycle:

- Test work, integration verification, and failure-injection verification.
- Frontend and Wails integration.
- Stored Session migration.
- Session v2 cleanup, deletion, and retirement.
- Release/versioning work. No version bump is permitted during this cycle.
- A formal non-frontend Session v2 to Session v3 parity inventory.

The parity inventory remains necessary because the current source proves one known gap, direct capability selection, but does not define every non-frontend Session v2 behavior that must remain available before Session v3 can be declared a full replacement.
