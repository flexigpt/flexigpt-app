# Workflow Runtime

## 1. Decision

A Workflow declaration describes a reusable process.

A Workflow invocation is one durable execution of that process.

A Session is a conversation and execution container for interactive reasoning.

A Session Run is one accepted reasoning interaction, including its Model
operations, Tool calls, approvals, and continuation.

These are separate identities:

- Workflow Artifact: reusable process definition.
- Workflow invocation: process execution.
- Session: conversation.
- Session Run: reasoning execution.
- Workflow operation: one logical operation within an invocation.
- Operation attempt: one dispatch attempt for that operation.

A Workflow invocation is not a provider completion.

An Agent call from a Workflow uses the existing Session runtime. The Workflow
runtime does not implement another Agent Tool loop.

## 2. Ownership

### Declaration layer

`internal/llmartifactory-go/workflow` owns read-only declaration planning.

It owns:

- Workflow Artifact selection.
- Canonical Workflow document reconstruction.
- Composition results.
- Binding requirement inventory.
- Definition revision and digest evidence.

It does not own:

- Invocation state.
- CEL evaluation.
- Target execution.
- Timers.
- Human decisions.
- Invocation persistence.
- Runtime recovery.

`CompositionComplete` is not runtime readiness.

### Workflow runtime

`internal/agentruntime-go/workflow` owns:

- Invocation admission.
- Immutable invocation definition snapshots.
- Runtime capability admission.
- Operation identities.
- Prepared binding values.
- Execution and retry state.
- Durable waits.
- Invocation controls.
- Typed operational errors.
- Recovery barriers.
- Invocation snapshots.

The runtime consumes narrow Source, Executor, Repository, and HumanAuthorizer
ports. It does not receive an Artifact Store aggregate.

### Runtime setup

`internal/agentruntime-go/runtimesetup` joins:

- Workflow declaration planning.
- Artifact revision and digest checks.
- Model runtime preparation.
- Session runtime Agent execution.
- Application-registered service and activity bindings.

Credentials remain in the existing inference and Model runtime boundaries.
They are not written into Workflow records.

### Transport

The Wails wrapper exposes explicit invocation commands and read-only
snapshots. It does not accept prepared bindings, credentials, actor roles, or
arbitrary Workflow documents from the frontend.

## 3. Invocation API

An explicit invocation supplies:

- Command ID.
- Workflow Artifact selection.
- Expected Artifact revision or Definition digest when selected by the UI.
- JSON input.
- Optional JSON context.
- Optional Session association for presentation.

Example:

    InvokeWorkflow({
      commandID: "...",
      workflow: {
        rootID: "...",
        artifactID: "...",
        expectedRevision: 7,
        expectedDigest: "sha256:..."
      },
      input: {
        requestId: "...",
        release: "..."
      },
      sessionID: "..."
    })

`sessionID` is an association only. It does not transfer conversation history,
Model selection, Tool authority, attachments, or instructions.

The response contains an invocation ID.

The UI attaches through `GetWorkflowInvocation`. It can poll while an
invocation is active. Existing Session APIs remain authoritative for Agent
child Runs and their Tool approvals.

Commands are:

- Invoke.
- Pause.
- Resume.
- Cancel.
- Respond to a human wait.
- Resolve recovery review for an uncertain call.

Command IDs are idempotency identities. Reuse with another payload is an
error.

## 4. Admission

Admission completes before business work starts.

The runtime:

1. Resolves the selected Workflow plan.
2. Checks selection pins.
3. Captures the canonical document and Definition evidence.
4. Requires complete declared composition.
5. Prepares every target and requirement binding.
6. Checks the local execution profile.
7. Validates invocation input.
8. Writes the accepted invocation.
9. Starts runtime-owned execution.

An unavailable required binding rejects invocation admission.

A declaration that uses an unsupported execution capability also rejects
admission. The runtime must not discover an unsupported operation after
earlier business work has executed.

A later target availability change is an operational failure. It does not
change the captured Workflow definition.

## 5. Durable Execution Model

The local runtime uses a persisted operation journal.

An operation key includes:

- Declared step path.
- Composite attempt path.
- Loop iteration or collection index.

A retry keeps the logical step identity. Its attempt counter changes.

Each step captures its entry environment once. This includes completed
enclosing outcomes, state, input, context, and logical evaluation time.

Replay reconstructs control flow from that captured environment and terminal
journal entries.

Replay does not repeat a confirmed successful call.

For an external call, the write order is:

1. Capture selected binding and mapped values.
2. Persist prepared call data.
3. Persist dispatching state.
4. Execute the external call.
5. Persist returned output or asynchronous receipt.
6. Continue.

A persistence failure before dispatch prevents dispatch.

A crash after dispatching state but before a confirmed result creates an
uncertainty barrier. It does not cause automatic redispatch.

This is not an exactly-once guarantee for external systems.

## 6. Definition and Binding Consistency

The invocation stores an immutable Workflow document and prepared binding
descriptors.

Artifact-backed binding descriptors contain identities and expected
revision/digest evidence, not provider credentials.

Service and activity bindings contain application registration identities
and revisions. They do not create synthetic Artifacts.

Before use, an application registration must still have the captured
revision.

A target's authored interface constrains the binding. It cannot grant
capabilities that the binding does not support.

The binding's effect contract is authoritative for retry safety.

## 7. Values and State

CEL evaluation uses the standard pure CEL environment.

Runtime evaluation has a cost limit and observes context cancellation.

JSON values are copied at binding boundaries.

Integral JSON numbers outside the interoperable exact-integer range are
rejected instead of silently changing value during persistence or CEL
adaptation.

A single-key `$literal` object suppresses directive interpretation for its
complete value.

All fields of one constructed value use the same environment snapshot.

A `set` operation:

1. Evaluates every assignment against the previous state.
2. Applies shallow assignments and removals.
3. Validates the complete candidate state.
4. Publishes the new state atomically.

Parallel branches and Foreach items receive independent state copies.
Their state is not implicitly merged into the parent.

Step outcomes expose status, output, error, and metadata. Skipped steps do
not expose output.

## 8. Calls

### Agent

An Agent call creates a dedicated child Session and Run.

Only mapped input and mapped context enter that Session.

The child Session configuration selects the captured Agent. It does not
inherit the active chat configuration.

Agent execution uses the existing Session runtime for:

- Model preparation.
- Tool execution.
- MCP approvals.
- Skill sessions.
- Streaming.
- Cancellation.
- Recovery.

The Workflow call retains the child Session and Run IDs as its receipt.

A waiting child Run keeps the Workflow call waiting. The UI uses existing
Session commands to resolve child Tool approvals or request continuation.

Agent output is parsed as JSON when possible. Otherwise it is a string.
Declared output schemas determine whether that result is acceptable.

### Model

A Model call performs one prepared Model operation.

It uses the existing runtime setup and inference boundary.

It does not start an Agent Tool loop.

Unexpected local Tool-call proposals are an execution error.

### Service and activity

These calls require explicit application registrations.

A registration supplies:

- Stable binding identity.
- Registration revision.
- Callable interface.
- Effect contract.
- Execute function.

The registry is application-owned. The frontend cannot register executable
functions.

### Nested Workflow

Nested Workflow calls require a separately prepared child invocation and
parent-child lifecycle integration. This is outside the initial local
execution profile.

They must not be simulated by recursively calling an unjournaled function.

## 9. Waiting

Timer waits persist their absolute deadline. Replay does not reset a timer.

Human waits persist:

- Prepared request.
- Assignees.
- Response schema.
- Step identity.
- Deadline inherited from the step.

The frontend supplies response data only.

A backend HumanAuthorizer identifies and authorizes the responder.
Caller-supplied roles are not accepted.

An invalid response leaves the existing wait pending.

Session Tool approvals are not converted into Workflow human responses.
They remain owned by the child Session Run.

## 10. Retry and Recovery

Automatic call retry requires:

- Remaining declared attempts.
- A matching retry policy.
- A binding-owned safe effect contract.
- No unresolved uncertain effect.

Read and idempotent bindings can be retried.

Keyed retry additionally requires a stable prepared key and binding support
for keyed invocation.

Unknown or non-idempotent effects do not receive automatic retry.

On recovery, a call left in dispatching state becomes uncertain.

Recovery review can establish:

- Applied, with validated output.
- Not applied, allowing an explicit safe redispatch.
- Cancel invocation.

The review command is recorded durably.

The local profile does not automatically reconcile unknown effects or
perform saga compensation.

## 11. Controls and Shutdown

Pause prevents new normal dispatch. It does not claim to undo active work.

Resume makes pending work eligible again.

Cancel:

- Cancels the active runtime context.
- Requests cancellation of an active Agent child Run.
- Prevents new normal work.
- Preserves uncertainty when external execution may have occurred.

Shutdown cancels runtime-owned workers and waits for them.

Accepted work does not depend on the lifetime of a Wails request.

## 12. Persistence

The local Repository stores complete invocation records.

The supplied filesystem implementation uses:

- Private directory permissions.
- Bounded record reads and writes.
- Temporary-file writes.
- File synchronization.
- Atomic rename.
- Directory synchronization.

Workflow persistence is separate from Session persistence.

Agent child Sessions continue to use the existing Session checkpoint store.

This avoids changing Session-store serialization without inspecting its
implementation. A future shared deployment can implement Repository using
another storage engine without changing the interpreter.

Public snapshots exclude:

- Prepared binding data.
- Captured operation environments.
- Executor-private prepared call data.

Business input, output, human requests, and sanitized errors remain durable
application data.

## 13. Initial Local Execution Profile

The initial implementation supports:

- Explicit invocation.
- Sequence.
- Transform.
- Set.
- Scope.
- Choose.
- Parallel with all completion.
- Foreach with all completion.
- Counted, while, and until Repeat.
- Block and Workflow return.
- Fail.
- Break and continue.
- Timer waits.
- Human waits.
- Agent calls.
- Model calls.
- Registered service and activity calls.
- Explicit-member dynamic target selection.
- Call retry.
- Pause, resume, cancellation, and recovery review.

The following declarations reject admission in this profile:

- Event and schedule triggers.
- Event subscriptions.
- Event and channel waits.
- Emit.
- Published ports and streaming ports.
- State machines.
- Sagas and compensation.
- Reconciliation blocks.
- Async start and completion handles.
- Independent child lifetime.
- Conversation handoff.
- Cancel-by-handle.
- Nested Workflow calls.
- Resolver-produced target sets.
- Resource claims and rate limits.
- Token and monetary budgets.
- Non-all parallel completion.
- Composite retry.

These are runtime capability exclusions, not changes to the portable v1
declaration language.

Complete v1 support requires additional runtime implementation. An invocation
using these capabilities must not be accepted by this implementation.

## 14. Verification Plan

The implementation requires tests for:

- Selection pin mismatch and incomplete composition.
- Rejected unsupported capability before target execution.
- Input, state, call-output, and human-response schema failures.
- CEL lexical scope and literal directives.
- Replay without repeated confirmed effects.
- Persistence failure before dispatch.
- Recovery from dispatching state.
- Applied and not-applied recovery decisions.
- Retry input, target, and key stability.
- Isolated parallel and collection state.
- Timer deadline preservation.
- Child Session approval and continuation.
- Cancellation while preparing, invoking, polling, and waiting.
- Shutdown with active calls.
- Command ID conflict and duplicate handling.
- Filesystem record limits and atomic replacement.

No existing declaration, decoder, hydration, or module version changes are
required by this local profile.

## Important review findings

The artifacts above are not sufficient to label the requested complete Workflow runtime finished.

In addition to the explicit profile exclusions, source review identifies these remaining correctness requirements:

- A separate persisted retry key is needed. Reading it from executor-private prepared JSON is not a sound general Executor contract.
- Applied recovery must validate the call-level and step-level output schemas, not only binding constraints.
- Invalid human responses should reject the response command synchronously. The current implementation validates them on the next drive.
- Paused-branch suspension must not be confused with invocation failure.
- Child limits need subtree-owned counters. The current root counters do not fully implement inherited step budgets.
- Root duration expiry and cancellation must preserve uncertainty and settle all active children before publishing a terminal invocation.
- Parallel Workflow returns need a durable winner and cancellation of remaining attached work.
- `finally` needs bounded cleanup execution and preserved original outcomes.
- Agent binding admission must prepare the effective Agent configuration before accepting the invocation, not only validate its Artifact identity.
- The filesystem implementation needs platform-specific atomic replacement and directory synchronization verification, particularly on Windows.
- Restored records need structural journal validation before execution.
- Registered callable failures need an explicit result/effect-state contract. An ordinary Go error cannot establish whether a side effect occurred.

These are assistant-owned implementation gaps, not work that should be assigned to your tests or entry-point hooking. I have listed them explicitly rather than presenting an unverified partial implementation as complete end-to-end `v1` support.

## Implementation status

- [x] Authored the runtime HLD, invocation contracts, persistence adapter, local interpreter, binding adapter, and Wails wrapper.
- [x] Supplied declaration-evidence changes against the attached `plan.go`.
- [x] Kept declaration planning separate from runtime execution.

## Next steps

- [ ] Review the invocation ownership decision and the explicit distinction between the local profile and complete `v1` support. Patch application should wait until the pending correctness work is complete.
- [ ] After completion, hook construction and shutdown in your application entry point: construct `workflow.Service`, a Repository, the trusted callable registry, and a backend human authorizer; initialize `WorkflowRuntimeWrapper`; close Workflow runtime before Session runtime.
- [ ] Assistant-owned: resolve the listed correctness gaps before integration.
- [ ] Assistant-owned: implement the remaining `v1` capabilities or obtain an explicit agreement to ship only the local profile.
- [ ] Assistant-owned: compile and verify the complete patch when repository access is available.
