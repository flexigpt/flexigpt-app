Complete all pending backend functional, architectural, deduplication, and code-hygiene tasks for Session Runtime v3, Workflow Runtime v3, Session Store v3, and V3 setup packages in one large, comprehensive, single-shot end-to-end delivery.

Scope and strict guardrails:

- This work is WIP. Do NOT bump or change any version identifier anywhere (including Workflow apiVersion, schema versions, Store schema versions, interpreter versions, payload versions, module/package versions, or public API versions).
- Ignore and do NOT add or modify tests, validation reports, validation-only hardening, frontend integration, SessionV2 removal/migration/conversion/cleanup, or cost metering.
- Maintain strict architectural separation of concerns:
  - Artifact Store and `internal/llmartifactory-go/workflow` remain passive read-only source-fact providers and must not gain runtime, setup, binding, or execution concerns.
  - Core Runtime (`internal/agentruntime-go/sessionv3`) owns durable orchestration, scheduling, settlement, recovery, and projections only, and must never import Artifact Store or setup packages.
  - Owning domain setup packages (`attachmentruntime`, `modelruntime`, `toolruntime`, `mcpruntime`, `skillruntime`, `textruntime`, `contributionruntime`) own domain request construction, readiness, native execution, and result adaptation.
  - `sessionv3runtime` and `workflowruntime` remain thin orchestration setup layers composing neutral domain contracts.
- Coding conventions:
  - Do NOT use hardcoded string literals; declare and use `const` declarations.
  - Use proper typed enums everywhere instead of raw `string` fields for closed state/mode/reason/phase/policy sets.
  - All error strings (`errors.New`, `fmt.Errorf`, `SanitizedError.Message`) in touched V3/setup/store files MUST be lowercase.
- Delivery format:
  - No tools are attached. Produce only unified diff outputs strictly following the attached code delivery instructions.
  - Expect and produce a complete, large single-shot output covering every item below rather than partial or incremental patches.

Functional and architectural items to complete:

1. Fix MCP tool choice duplication (`WF-PREPARE-005.b`, `ARC-CLEAN-006`):
   - In `internal/agentruntimesetup/mcpruntime/v3.go:collectMCPServerChoices`, remove the duplicated nested sort-and-append loop so each selected MCP tool emits exactly one `toolruntime.V3PreparedChoice` with its `ServerArtifact` pin preserved.

2. Complete domain-owned contribution preparation for Workflow `context.defaults` and optional Agent capabilities (`WF-PREPARE-003`, `WF-PREPARE-003.a`, `WF-PREPARE-005`, `ARC-DOMAIN-003`, `ARC-DOMAIN-003.a`):
   - In `internal/agentruntimesetup/contributionruntime/prepared.go`:
     - When preparing Agent capabilities (`TypeText`, `TypeSkill` instruction rendering, `TypeMCP`), if preparation fails and `occurrence.Required` is false, skip that optional capability instead of failing the invocation; fail when `occurrence.Required` is true.
     - Accept optional specific `SessionDefaults` / `SessionDefaultsOverride` in `contributionruntime.Request` and apply collection policies/removals/replacements and `Clear` fields (including clearing or replacing an Agent-inherited `Workspace` or applying `WorkspaceOverrides`) BEFORE performing live Text, Skill, or MCP preparation, and pin/prepare explicit `Workspace`, `InstructionArtifacts`, `ToolSelections`, `Skills`, `ActiveSkills`, and `MCP` selections in the same `PreparedContributionPlan`.
   - In `internal/agentruntimesetup/workflowruntime` (`admission.go`, `model_agent_adapter.go`, `target_preparation.go`, `compiler_invocation.go`) and `internal/agentruntimesetup/sessionv3runtime/llm_plan.go`:
     - Prepare per-call-site `PreparedTargets` for Model, Agent, and ProviderTool calls with `call.LLMInvocation.Defaults` applied during Workflow execution preparation so `context.defaults` (`workspace`, `workspaceOverrides`, `clear`, `instructionArtifacts`, `toolSelections`, `skills`, `activeSkills`, `mcp`) is pinned and frozen before `ProgramSnapshot` creation and never fails with missing Workspace pins or falls back to live bind-time MCP discovery.
     - Route normal Session V3 `llm_plan.go` preparation through the same `contributionruntime.Preparer` override filtering so excluded Agent/Workspace capabilities are not materialized before removal.

3. Complete normal conversation Tool error settlement and read-only uncertainty guard (`V3-TOOL-006`, `WF-EFFECT-003`, `WF-EFFECT-003.b`):
   - In `internal/agentruntime-go/sessionv3/runtime.go:finishWorkLocked`, ensure `item.Effect.Kind == EffectKindRead` never transitions to `WorkItemStateUncertain` (settle as `WorkItemStateFailed` with `EffectStateNotApplicable`).
   - In `internal/agentruntimesetup/toolruntime/v3.go:ExecuteGo`, `internal/agentruntimesetup/mcpruntime/v3.go:Execute`, and `internal/agentruntimesetup/skillruntime/v3.go:Execute`:
     - When `payload.Call.Source` is `ToolCallSourceModel` or `ToolCallSourceApp` and `ctx.Err() == nil`, settle tool execution errors (`invokeErr != nil`, `response.IsError == true`, `response.Body.IsError == true`) as confirmed `WorkExecutionResult{Output: ..., Confirmed: true, Error: nil}` with `ToolOutputKindError` and `EffectStateConfirmedNotApplied` (do not attach an applied `ExternalReceipt` on error, while preserving `SkillState` on Skill outputs), so Agent loops pause cleanly for retry/skip/continue and pass tool error outputs back to the Model instead of entering `WaitKindRecovery`.
     - Preserve `EffectStateUncertain` (`Confirmed: false`) when `ctx.Err() != nil` on mutating work and for mutating Workflow Tool calls (`Effect.Kind != EffectKindRead` when not Model/App-sourced).

4. Preserve multi-turn handoff history and continuation tool consumption (`WF-HANDOFF-006`, `WF-HANDOFF-003`, `WF-HANDOFF-003.a`, `V3-CONTINUE-003.a`):
   - In `internal/agentruntime-go/sessionv3/program.go` and `internal/agentruntimesetup/sessionv3runtime/inputs.go`:
     - Track completed handoff input Activity IDs in `agentLoopProgramState` across `startQueuedAgentLLM` transitions and include prior handoff user input Activities plus frame-owned assistant Activities and completed tool outputs in chronological order inside `llmInputActivitySequence` when `IsolatedInput` is true, without including ambient Session timeline messages from outside the handoff.
     - Pass settled `state.PendingToolIDs` into `startAgentLLM` when `startQueuedAgentLLM` advances after a tool batch so `advanceFrameLocked` records `ConsumedByWorkItemID` on those Tool WorkItems.

5. Scope manual Tool controls and automation holds (`V3-TOOL-006`, `V3-CONTINUE-003`, `V3-CONTINUE-003.b`):
   - In `internal/agentruntime-go/sessionv3/normal_session.go` and `runtime.go`:
     - Route conversation-scoped `ExecuteToolCalls`, `RetryToolCall`, `SkipToolCall`, `CompleteToolCallAsError`, and `SetAutomationHold` to the active conversation owner Execution (including active handoff Execution when `ExecutionID` refers to the conversation root or is omitted).
     - Reject manual Tool mutations (`ExecuteToolCalls`, `RetryToolCall`, `SkipToolCall`, `CompleteToolCallAsError`) when the Execution is terminal or cancelling, has a pending recovery review wait, has an active (`Binding` or `Dispatched`) LLM WorkItem on the owning frame, or targets a Tool WorkItem that does not belong to the owning frame's current unconsumed Tool batch.

6. Complete cleanup expiry settlement and pre-checkpoint consistency (`RT-CLEANUP-006`, `RT-SETTLE-003`, `RT-SETTLE-003.b`, `RT-CLEANUP-003`, `RT-CLEANUP-003.c`, `RT-SETTLE-005`, `RT-SETTLE-005.b`):
   - In `internal/agentruntime-go/sessionv3/budget.go:waitForCleanupDeadline`:
     - Mark remaining `CompensationStatePending` `EffectRecord`s as `CompensationStateSkipped` and `CompensationStateDispatching` `EffectRecord`s as `CompensationStateUncertain` (attaching the cleanup-deadline error), cancel active `CancelWithParent` child Executions, and commit through `commitTerminalMutationLocked` (after `completeExecutionLocked`) so `executionTerminalReadyLocked` succeeds and the Execution reaches its terminal state.
   - In `internal/agentruntime-go/sessionv3/runtime.go:dispatchOne`, persist the revert from `WorkItemStateBinding` to `WorkItemStateReady` if `!r.dispatchAllowedForWorkLocked` after `BindWork` returns.
   - In `internal/agentruntime-go/sessionv3/runtime.go:ResolveWait`, call `scheduleWaitDeadlineLocked` only after `persistLocked` commits the failed wait resolution.
   - In `internal/agentruntime-go/sessionv3/runtime.go:startDispatchedLocked`, remove the redundant intermediate `"workBound"` checkpoint so binding + attempt creation persist once at `checkpointReasonWorkDispatched`.

7. Forward published nested Workflow output ports across independent started calls (`WF-PORT-006`, `WF-CHANNEL-003`, `WF-CHANNEL-003.d`):
   - In `internal/agentruntime-go/sessionv3/ports_runtime.go:preparePublishedWorkflowPortBoundaryLocked`, resolve the target Execution and receiving frame/budget-scope owner for each output `PortPlan` via `portChannelExecution(session, childExecution, plan)` so `mode: start` with `lifetime: independent` forwards published nested Workflow output and streamed port records to the caller Execution channel.

8. Consume staged MCP App context on `SubmitMessage` (`V3-APP-006`, `V3-APP-003`, `V3-APP-003.b`):
   - In `internal/agentruntime-go/sessionv3/runtime.go:SubmitMessage`, pass unconsumed `execution.Continuation.AppContext` into `submitNormalMessageWithDefaultsLocked` and clear consumed `AppContext` (updating `ConsumedRevision` when no unconsumed continuation text/attachments remain).

9. Remove duplications, redundant validations, and hardcoded/untyped strings (`ARC-CLEAN-006`, `ARC-DUP-003`, `ARC-DUP-003.c`, `ARC-DUP-003.d`):
   - Remove the unused `V3AttachmentAdmitter` wrapper in `internal/agentruntimesetup/sessionv3runtime/attachments.go`.
   - Add `Selection()` on `sessionv3.ArtifactPin` and consolidate duplicated Artifact pinning/selection helpers across `contributionruntime`, `sessionv3runtime`, and `workflowruntime` into one shared setup helper.
   - Add `BuildConversationProjection` and `BuildActivityProjection` in `internal/agentruntime-go/sessionv3/projection.go` and reuse them from `sessionv3.Runtime`, `BuildSessionSnapshotProjection`, and `internal/sessionstorev3.Store`.
   - Eliminate redundant validations:
     - Decode and validate `command.Program` once in `Runtime.StartWorkflow` and `Runtime.RegisterWorkflowTriggers`.
     - In `Runtime.persistLocked`, validate only non-terminal frame states (`ValidateState`) without re-decoding/re-validating immutable `ProgramSnapshot` payloads or `TriggerRegistration` programs on every checkpoint.
     - In `internal/sessionstorev3`, validate the incoming `Session` once per checkpoint write rather than repeating `Session.Validate()` in both `HandleCheckpoint` and `writeRawDocument`.
     - Remove redundant per-node `.Validate()` calls during `workflowruntime` compilation where the final lowered `WorkflowProgram.Validate()` already validates the tree.
   - Replace hardcoded checkpoint reason strings and untyped closed string fields in `sessionv3` (`CheckpointView.Reason`, `MessageActivity.Status`, `WorkflowActivity.Phase`, `WorkflowCompletion.Mode`, `WorkflowParallel.Errors`/`Remaining`, `WorkflowForeach.Errors`/`Remaining`, `WorkflowRepeat.Lifetime`/`OnExhausted`, `WorkflowCall.OnCompletion`) with typed enums and `const` declarations, and normalize error strings to lowercase.

10. Update HLD ledger statuses:
    - Update `devdocs/requirements/workflow_runtime.md` in the same diff delivery so every completed `Partial` and `Not implemented` task and subtask (`RT-SETTLE-003`, `RT-SETTLE-003.b`, `RT-CLEANUP-003`, `RT-CLEANUP-003.c`, `WF-PREPARE-003`, `WF-PREPARE-003.a`, `WF-CHANNEL-003`, `WF-CHANNEL-003.d`, `WF-EFFECT-003`, `WF-EFFECT-003.b`, `V3-CONTINUE-003`, `V3-CONTINUE-003.a`, `V3-CONTINUE-003.b`, `WF-HANDOFF-003`, `WF-HANDOFF-003.a`, `ARC-DOMAIN-003`, `ARC-DOMAIN-003.a`, `V3-APP-003`, `V3-APP-003.b`, `ARC-DUP-003`, `ARC-DUP-003.c`, `ARC-DUP-003.d`, `RT-SETTLE-005`, `RT-SETTLE-005.b`, `WF-PREPARE-005`, `WF-PREPARE-005.b`, `RT-CLEANUP-006`, `WF-PORT-006`, `WF-HANDOFF-006`, `V3-TOOL-006`, `V3-APP-006`, `ARC-CLEAN-006`) is marked `Implemented.` with exact source completion evidence, while keeping the HLD marked `WIP` with no version changes.
