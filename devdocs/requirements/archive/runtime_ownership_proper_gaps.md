# Runtime gaps

## Gap map

| ID  | Gap                                                                                                                        | Main impact                                                                              | Priority           |
| --- | -------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ------------------ |
| 1   | Frontend still compiles and merges full `Session` configuration intent.                                                    | Parallel clients must reproduce semantic configuration logic.                            | High               |
| 2   | Backend has no `Session`-level default intent or default `Agent` and `Model` resolution.                                   | Different UIs can produce different default behavior.                                    | Medium             |
| 3   | `Session` waits are tool-centric; generic configuration and authentication waits are not implemented.                      | Blocks `Session`-owned MCP readiness, App-context approval, and similar flows.           | High, foundational |
| 4   | Selected MCP readiness, connection, and authentication are frontend-managed, and the backend can silently omit selections. | Selected MCP behavior differs by UI; a CLI must orchestrate management runtime behavior. | High               |
| 5   | MCP App-context approval and provenance can be bypassed through normal user or continuation input.                         | Backend trusts frontend claims that App context was approved.                            | Critical           |
| 6   | Tool `userArguments` are ignored for non-web-search tools.                                                                 | UI validates options that do not affect execution.                                       | High               |
| 7   | A normal `Session` `Skill` cannot be selected as a backend-owned instruction source.                                       | Frontend flattens `Skill` output into text and loses provenance and pinning.             | Medium / High      |
| 8   | Continuation submission is a frontend read-merge-write transaction.                                                        | Every client must reproduce continuation merge semantics.                                | Medium             |
| 9   | Resolution summaries do not retain excluded or displaced inherited contributions.                                          | Backend cannot authoritatively explain removals or replacements.                         | Medium             |
| 10  | Input and resource budgeting are not centrally enforced by the backend.                                                    | UI limits can be bypassed; oversized attachments or MCP context can inflate requests.    | High               |

## Session configuration intent is still compiled and merged in the frontend

### Gap

The frontend does not merely serialize user choices into a command. It constructs the low-level `SessionConfiguration` override structure.

The primary logic is in:

- `frontend/app/chats/session/session_intent.ts`
- `mergeSessionConfiguration`
- `mergeCollectionOverrides`
- `mergeWorkspaceIntent`
- `workspaceSelectionFromPayload`
- `mcpOverridesFromPayload`
- Pin-preservation helpers
- `Agent` exclusion and release handling

The backend has the actual composition engine:

- `runtimesetup.normalizeConfigurationIntent`
- `resolveAgent`
- `resolveWorkspace`
- `applySessionOverrides`
- `applyInstructionOverrides`
- `applyToolOverrides`
- `applySkillOverrides`
- `applyMCPOverrides`

However, `Session` commands currently accept a complete replacement configuration rather than a higher-level configuration patch.

### Impact

A second UI must understand and reproduce:

- `inherit`, `add`, `remove`, and multi-record `replace` semantics.
- Canonical override keys.
- Which pins must survive a UI refresh.
- How to preserve a removal override when the visible UI no longer displays its source.
- How `Workspace` preview state becomes `Workspace` context, `Skill`, and MCP override records.
- How runtime MCP selections become durable `Artifact`-backed selections.
- Which prior values must survive a partial Composer edit.

The backend remains execution-authoritative, so this is not a duplicate execution-runtime problem.

It is a configuration-semantics portability problem.

The current backend normalizer also has some ambiguity. For example, `normalizeArtifactOverrides`, `normalizeToolOverrides`, and `normalizeMCPOverrides` validate that a key exists, but do not consistently enforce that an override key corresponds to the supplied value's canonical identity. Some application functions use `override.Key`, while others derive a key from the value again.

### Current behavior

- The frontend retains `sessionConfigurationBaseline`.
- The frontend merges the baseline with current Composer state before each submission.
- The backend validates the resulting shape in `PrepareSubmission`.
- The backend independently resolves actual sources only while preparing the operation.

This prevents execution from trusting the frontend, but it makes clients responsible for producing the exact intent representation.

### Backend enhancement or exposure needed

#### Add a backend-owned configuration patch operation

Prefer commands such as:

- `ApplySessionConfigurationPatch`
- `PreviewSessionConfiguration`

These should replace the requirement for every client to construct a complete `SessionConfiguration`.

Conceptually:

```go
type SessionConfigurationPatch struct {
    Agent         *SelectionPatch
    Workspace     *WorkspaceSelectionPatch
    Model         *SelectionPatch
    ModelDefaults *ModelDefaultsPatch
    History       *HistoryPatch

    Instructions []InstructionChange
    Tools        []ToolChange
    Skills       []SkillChange
    ActiveSkills []SkillActivityChange
    MCP          []MCPChange
}
```

The patch should express user intent rather than frontend-computed merge artifacts. For example:

- Add tool `X`.
- Remove inherited `Agent` tool `Y`.
- Replace the `Workspace` context selection with `[A, B]`.
- Activate `Skill` `K`.
- Clear the model system prompt.

The backend should then:

1. Load the existing `Session` configuration.
2. Apply the patch.
3. Normalize keys and pins.
4. Reject ambiguous or inconsistent override records.
5. Return the normalized durable `Session` configuration.
6. Optionally return a safe resolution preview.

#### Add a generic Session configuration preview

The preview should reuse `runtimesetup` composition and return only safe data:

- Normalized configuration.
- Resolution summary.
- Effective owners.
- Diagnostics.
- Stale or unavailable state.
- Included and excluded items.

It must not return:

- Provider credentials.
- Provider-ready request data.
- Private tool bindings.
- Authorization tokens.
- `Skill` runtime handles.
- Raw attachment bytes.

#### Existing backend pieces to reuse

Most of the difficult implementation already exists:

- `normalizeConfigurationIntent`
- `PrepareModelOperation`
- `OperationResolutionSummary`
- `apply*Overrides`
- `Agent`, `Workspace`, `Tool`, `Skill`, and MCP resolution in `runtimesetup`

This is primarily an API and command-shape enhancement in `Session` and `runtimesetup`. It does not require moving execution logic out of the frontend.

#### Likely LLM Artifactory work

Generic patch support should require little or no LLM Artifactory work.

Potentially useful additions include:

- A stable public canonical capability-selection identity API.
- Stronger domain-level validation of `Artifact` and capability selectors.
- Stable default-selection APIs, covered separately below.

### Frontend behavior after filling the gap

The Composer keeps local draft state but no longer owns semantic merging.

It would:

1. Maintain visual choices and draft edits.
2. Send a patch or preview request.
3. Render backend-returned normalized configuration and diagnostics.
4. Start a `Run` using the normalized `Session` configuration or a run-only patch.

`session_intent.ts` becomes substantially smaller:

- DTO adaptation remains.
- UI-only formatting remains.
- Backend semantic merge, pin, and override handling disappears.

## Default Agent and Model selection are frontend policy today

### Gap

A `Session` with no effective model fails during backend preparation with:

```text
model_selection_missing
```

The frontend compensates by selecting defaults:

- `useComposerContextState` chooses the catalog-default `Model`.
- `useAgentManager` chooses a base or default `Agent`, including fallback behavior such as selecting the first selectable `Agent`.

There is no visible `Session`-level backend API equivalent to:

- `ResolveDefaultSessionIntent`
- `GetDefaultSessionConfiguration`
- `ResolveDefaultAgent`
- `ResolveDefaultModel`

### Impact

A CLI or another UI must reproduce frontend policy to get equivalent behavior:

- Which `Agent` should a new `Session` use?
- Is no `Agent` different from a default `Agent`?
- Which `Model` should be selected when no explicit `Model` is supplied?
- Does an `Agent`-selected `Model` override the platform default?
- What does clearing the `Agent` or `Model` selection mean?

Without a backend answer, different clients can create semantically different default `Session` configurations.

### Current behavior

The React Composer chooses an `Agent` and a `Model` before calling `CreateSession` or `StartRun`.

This works, but it means the UI defines product-default behavior.

### Backend enhancement or exposure needed

Add a runtime-facing default-intent resolver, such as `GetDefaultSessionConfiguration` or `ResolveDefaultSessionIntent`.

It should return explicit, durable selections:

- `Agent` `Artifact` reference, revision, or digest.
- `Model` `Artifact` reference, revision, or digest.
- Default model overrides and history policy, where applicable.

The contract must distinguish between:

- Unspecified selection, meaning use the backend default.
- Explicitly cleared selection, meaning no `Agent`.
- Explicitly selected `Agent`.

Do not infer that a missing `Agent` means use the default `Agent` unless that behavior is formally part of the contract.

#### Existing backend pieces to reuse

- The `Model` aggregate already has default-provider preference logic.
- The model runtime adapter already resolves concrete models.
- `Agent` runtime setup already resolves explicit `Agent` selections.
- Built-in topology likely already knows which base or default `Agent` exists.

#### Likely LLM Artifactory work

This likely requires:

- A domain-level `ResolveDefaultModel` capability rather than a frontend catalog convention.
- A domain-level `ResolveDefaultAgent` capability or default-`Agent` policy.

The policy should not be based on the frontend finding an `Agent` named `base`.

### Frontend behavior after filling the gap

For a new chat:

1. The Composer requests defaults from the `Session` runtime, or calls `CreateSession(useDefaults=true)`.
2. The backend returns the selected and pinned default configuration.
3. The frontend renders that configuration.

The frontend may still display a label such as Base Agent, but it no longer decides what that label means.

## Generic Session waits are declared but not implemented

### Gap

The `Session` type system declares several non-tool wait kinds:

- `WaitKindConfiguration`
- `WaitKindAuthentication`
- `WaitKindUserInput`
- `WaitKindContinuationReview`

However, current `ResolveWait` handling is effectively tool-call-centric. Outside recovery waits, it assumes that a wait has an owning `Tool` call:

```go
findToolCall(run, wait.ToolCallID)
```

`dispatchWaitResolution` similarly requires a `Tool` call and routes to `Tools.ResolveToolWait`.

A configuration, authentication, or App-context wait therefore cannot currently be owned and resolved independently of a `Tool` call.

### Impact

This prevents the runtime from properly owning flows that should be `Run`-owned:

- MCP connection requirements.
- MCP OAuth requirements.
- MCP configuration requirements.
- MCP App-context approval.
- Future user-input waits.
- Future configuration confirmation or recovery flows unrelated to a `Tool`.

The frontend must therefore manage some of these flows outside the `Session` wait model.

### Current behavior

- Tool approval uses normal `Session` waits correctly.
- Recovery uses a special-case wait path correctly.
- MCP connection and OAuth are handled through management runtime calls and frontend polling.
- MCP App-context approval is frontend modal state.
- Other non-tool wait kinds are effectively placeholders.

### Backend enhancement or exposure needed

Generalize `Session` waits into a handler-based model.

```go
type WaitHandler interface {
    ResolveWait(
        ctx context.Context,
        request WaitResolutionRequest,
    ) (WaitResolutionResult, error)
}
```

Each wait should have an explicit owner or handler category:

- `tool`
- `recovery`
- `mcp-connection`
- `mcp-authentication`
- `app-context`
- `configuration`

The `Session` core should:

1. Persist safe wait state.
2. Keep private executor references out of snapshots and checkpoints.
3. Dispatch to the correct backend handler.
4. Revalidate `Run` and `Session` revisions.
5. Resume, fail, or continue only through the `Session` driver.

The dependency direction should remain:

```text
session -> generic wait port
runtimesetup -> MCP/App/configuration wait implementations
```

The `Session` package should not import MCP packages.

### Frontend behavior after filling the gap

The frontend has one generic wait flow:

1. Read the pending wait from the snapshot.
2. Render the wait.
3. Collect the user decision or input.
4. Call `ResolveWait`.
5. Refresh the snapshot.

No custom MCP polling or custom approval state is required for correctness.

## Selected MCP readiness, connection, and authentication are still frontend-gated

### Gap

`runtimesetup.prepareMCP` only checks whether the MCP runtime is already ready:

```go
status, statusErr := c.mcpRuntime.runtime.Status(ctx, serverID)
if statusErr != nil || status == nil || status.Status != Ready {
    // Record a diagnostic, then skip or fail only required Agent MCP.
}
```

It does not:

- Start a connection.
- Create a configuration wait.
- Create an authentication wait.
- Wait for OAuth completion.
- Retry preparation after connection.
- Treat an explicit user-selected MCP server as required by default.

For an explicit `Session` MCP selection, current behavior is generally:

- Detect that the server is not ready.
- Record a warning diagnostic.
- Omit the server, tools, resources, and prompts.
- Continue the model operation.

The frontend masks this by calling management runtime APIs before submission:

- Connect the server.
- Poll readiness.
- Load discovery data.
- Reject or block Send until the server is ready.

### Impact

A CLI can submit a valid `Session` MCP selection and receive a `Run` that silently executes without the selected MCP contribution.

A second UI must reproduce the Composer's management workflow before calling `StartRun`.

This is not an MCP execution implementation gap. The MCP runtime itself is backend-owned. It is an admission and readiness ownership gap.

### Current behavior

The Composer uses:

- `ConnectMCPServer`
- Runtime-status polling
- Discovery APIs
- `prepareForSubmit`

The MCP runtime manager is backend-owned and already provides:

- `StartConnect(...)`
- `Status(...)`
- `Refresh(...)`

However, `runtimesetup.mcpRuntimeAdapter` intentionally exposes only read and discovery operations, not connection orchestration.

### Backend enhancement or exposure needed

Add a `Session`-facing MCP readiness adapter in `runtimesetup`.

It should be able to:

- Resolve an `Artifact` MCP server to a runtime server identity.
- Inspect setup and authentication status.
- Start or join a backend connection attempt.
- Determine whether user interaction is required.
- Return a safe wait or diagnostic outcome.

Explicit `Session` MCP selections need a policy such as:

- `required`
- `best-effort`

Suggested defaults:

- An explicitly user-selected `Session` MCP server is `required`.
- A required `Agent` MCP server is `required`.
- An optional `Agent` or `Workspace` MCP candidate is `best-effort`.

For a `required` selection, a server that is not ready should produce a configuration or authentication wait, or a typed blocked-admission result. It should not be silently omitted.

#### Preferred approach: accepted Run enters a backend-owned wait

1. `StartRun` creates a `Run` in `Waiting` state.
2. The backend starts or joins the MCP connection attempt.
3. If OAuth or configuration is required, the backend creates a `WaitKindAuthentication` or `WaitKindConfiguration` wait.
4. The UI resolves the wait and opens an authorization URL when instructed.
5. The backend resumes preparation and starts the model operation.

This requires the generic wait enhancement described in Gap 3.

#### Less ambitious approach: explicit MCP preparation command

Add a command such as `PrepareMCPSelectionsForRun`.

It can return typed readiness and authentication requirements before `StartRun`.

This is still substantially better than requiring every UI to call management APIs directly, but the `Run`-owned wait model is cleaner.

#### Likely LLM Artifactory work

Connection control itself should require little or no LLM Artifactory work.

The required implementation already exists in:

- `MCPRuntimeManager.StartConnect`
- MCP runtime adapter and `Artifact` server resolution
- OAuth broker and authentication manager

A useful addition could be a safe domain-level MCP selection-readiness result that exposes no secrets while identifying configuration and authentication requirements.

### Frontend behavior after filling the gap

The Composer may still show MCP status and allow a manual connection action for UX.

However, on Send it:

- Sends durable `Artifact`-backed MCP selection intent.
- Does not require `prepareForSubmit` to succeed for correctness.
- Does not poll connection readiness for correctness.
- Renders `Session` waits and diagnostics from snapshots.
- Opens an authorization URL only when the `Session` runtime instructs it to.

A CLI receives the same behavior through `Session` commands and waits.

## MCP App context can bypass backend authorization and approval

### Gap

There are two related bypasses.

#### Raw App context can be sent through normal user input

`UserInput` and `ContinuationInput` contain:

```go
MCPAppContext []AppContextUpdate
```

`PrepareSubmission` calls `normalizeAppContextUpdates(...)`.

That validates shape and canonical JSON, but it does not verify:

- The App instance exists.
- The App belongs to this `Session` or `Run`.
- The view belongs to the App.
- The server belongs to the originating App route.
- The context came from an App.
- The user approved it.
- App policy permits it.

`buildInputs` then injects the content with language stating that it was:

> Explicitly approved from a Session-scoped MCP App.

That provenance can be forged by a client.

#### `RequireApprovalForContextUpdates` is enforced only in the frontend

`ValidateAppContext` checks:

- App policy is enabled.
- App, view, and server scope are valid.

However, it does not enforce `RequireApprovalForContextUpdates`.

The React `MCPAppView` displays a confirmation dialog before calling `StageAppContext`, but another UI, or a direct caller, can call it without that approval.

### Impact

This is a real backend authority gap.

A client can inject arbitrary content as approved MCP App context, bypassing:

- App scope validation.
- App policy.
- User-confirmation policy.
- Durable provenance guarantees.

This does not necessarily create an external side effect, but it breaks the intended trust model.

### Current behavior

The normal React MCP App path behaves correctly in practice:

1. An App asks to update model context.
2. The frontend displays a confirmation dialog.
3. The frontend calls `StageAppContext`.
4. The backend validates App-route scope.

However, public `Session` input paths bypass that route.

### Backend enhancement or exposure needed

#### Make App context backend-originated only

Choose one of the following approaches:

1. Remove raw `MCPAppContext` from public `StartRun`, `UpdateContinuationInput`, and `EditAndResend` input contracts.
2. Reject raw App context unless it references a backend-owned staged App-context record.
3. Replace raw values with `CarryForwardAppContextIDs` and resolve those IDs on the backend.

#### Make App-context approval a Session-owned wait

When App policy specifies:

```text
RequireApprovalForContextUpdates = true
```

`StageAppContext` should not immediately append content.

Instead:

1. `StageAppContext` records an App-context proposal.
2. The backend validates the App, view, and server.
3. The backend stores a pending App-context proposal.
4. The backend creates `WaitKindAppContextApproval`.
5. The user resolves the wait.
6. The backend appends approved context to the `Run` continuation.

This uses the generic non-tool wait mechanism described in Gap 3.

#### Existing backend pieces to reuse

- `StageAppContext`
- `ValidateAppContext`
- `AppInstance`
- `AppRoute`
- `WaitingRequest`
- `Run.Continuation`
- Existing MCP App policy fields

#### Likely LLM Artifactory work

No LLM Artifactory change should be necessary. The relevant policy field already exists:

```text
RequireApprovalForContextUpdates
```

The runtime needs to enforce it.

### Frontend behavior after filling the gap

- The App UI can still ask for user confirmation visually.
- Confirmation is no longer the authority.
- The frontend sends an App-context proposal command.
- If the backend requires approval, the frontend renders a `Session` wait.
- The Composer reads staged App context from the `Run` snapshot.
- The Composer no longer serializes arbitrary App-context objects into normal send or continuation payloads.

External URL approval remains frontend-owned because opening a browser is intentionally a UI action.

## Tool userArguments are not applied outside SDK web search

### Gap

The frontend stores and validates `Tool` user configuration:

- `ToolSelectionIntent.UserArguments`
- `ToolStoreChoice.userArgSchemaInstance`

The backend normalizes and persists it.

However, `runtimesetup.toolChoiceAndBinding` uses it only for SDK web search:

```go
case SDKToolTypeWebSearch:
    json.Unmarshal(contribution.selection.UserArguments, &config)
```

For Go tools:

- `goToolBinding` does not contain `UserArguments`.
- `goToolExecutor.ExecuteTool` receives only model-generated tool-call arguments.

SDK function and custom tools also do not apply user arguments.

### Impact

Assuming `userArgSchemaInstance` is intended to affect tool behavior, as its name, schema, UI, and validation strongly imply, this is a functional gap.

Today, the UI can:

1. Ask the user for tool options.
2. Validate them.
3. Block Send when they are invalid.
4. Persist them.

However, backend execution ignores them.

If the intended meaning is web-search-only configuration, the UI and contract are misleading and should reject or hide the setting for every other tool type.

### Current behavior

Only `WebSearch` has a concrete backend interpretation.

For other tool kinds, the value is durable but unused.

### Backend enhancement or exposure needed

First, define the `Tool` domain semantics.

The backend must answer whether user arguments are:

- Defaults merged into model tool arguments.
- Fixed values that the model cannot override.
- Executor configuration separate from model arguments.
- Provider tool-choice configuration.
- Unsupported for a given tool kind.

Do not implement a generic JSON map merge until these semantics are explicit.

#### Likely implementation

A better `Tool` domain API could provide:

- `PrepareToolChoice(selection)`
- `PrepareToolInvocation(selection, modelArguments)`

It can return:

- Provider tool choice.
- Prepared binding.
- Effective executor arguments.
- Validation diagnostics.

For Go tools, the prepared binding can include a canonical user-configuration payload.

For SDK tools, the backend can either:

- Convert the configuration into provider-specific tool-choice configuration.
- Reject it as unsupported.

#### Likely LLM Artifactory work

This likely requires explicit `Tool` artifact-contract semantics, such as:

- `userArgumentPolicy`
- Merge strategy
- Immutable or default keys
- Provider tool-choice mapping

The LLM Artifactory `Tool` service should own schema validation and merge semantics.

### Frontend behavior after filling the gap

- The Composer remains a form and UI for `Tool` user options.
- The frontend may perform early validation for UX.
- The backend is the final validator.
- Backend preview and operation summaries state whether the configuration is valid and how it is applied.
- No frontend-required `Tool` setting can be silently ignored during execution.

## Ordinary Session Skills cannot be backend-owned instruction sources

### Gap

The backend supports backend-rendered instruction `Skill` contributions for `Agent` contributions:

```go
skillContribution.instruction == true
```

However, `Session`-added `Skill` overrides always become ordinary `Skill`-session contributions. `applySkillOverrides(...)` sets `instruction` to `false`.

The frontend works around this by:

1. Calling instruction `Skill` preview or rendering.
2. Receiving rendered text.
3. Inserting the text into frontend system-prompt state.
4. Submitting inline instruction text.
5. Removing the matching `Skill` from ordinary `Skill`-session participation.

### Impact

The backend loses:

- `Skill` `Artifact` identity.
- Revision and digest pin semantics.
- Backend operation-time rendering.
- Source provenance.
- Stale or unavailable diagnostics.
- A clear distinction between a `Skill` contribution and user-authored inline text.

This is distinct from the `Agent` `Skill` issue, which is out of scope here.

### Current behavior

The frontend uses the following flow:

1. Call `PreviewInstructionSkill`.
2. Receive rendered text.
3. Submit the text through an inline `SessionInstructionOverride`.

The backend correctly renders `Agent` instruction skills, but not an explicitly selected `Session` `Skill` used in instruction mode.

### Backend enhancement or exposure needed

Add an explicit `Skill` contribution mode to `Session` configuration.

```go
type SkillSelectionIntent struct {
    Artifact  ArtifactSelection
    Use       SkillUseMode // session | instructions
    Arguments map[string]string
}
```

Alternatively, extend the current `Skill` override value with a `use` field containing either `skillSession` or `instructions`.

Update:

- `applySkillOverrides`
- `prepareSkills`
- Resolution-summary generation
- Active-skill validation

The backend already has most of the required execution capability:

- `Artifact` `Skill` resolution.
- `Skill` runtime support.
- `RenderSkill`.
- The instruction-rendering path in `prepareSkills`.

The backend needs to make that path available for explicit `Session` `Skill` selections, not only `Agent` baseline contributions.

Do not overload `InstructionOverride.Artifact` unless the backend can safely distinguish a `Text` `Artifact` from a `Skill` `Artifact`. That path currently uses the `Text` materializer.

#### Likely LLM Artifactory work

This may require small additions:

- Expose or standardize `Skill` insertion and use metadata.
- Expose argument and resource capability metadata needed to validate instruction rendering.
- Define typed render arguments if arguments are supported.

### Frontend behavior after filling the gap

Use as instructions becomes:

- Send a `Skill` `Artifact` selection.
- Set `use` to `instructions`.
- Include optional render arguments.

The frontend may show preview text, but does not submit that text as authority.

Keep a separate explicit action such as `Copy rendered Skill text into editable prompt`. That action is genuinely user-authored text and can remain frontend-authored.

## Continuation submission is still frontend read-merge-write orchestration

### Gap

The frontend currently has to:

1. Read the `Run` snapshot.
2. Read the existing `run.Continuation`.
3. Merge unconsumed attachments.
4. Merge staged MCP App context.
5. Decide whether to retain existing text.
6. Call `UpdateContinuationInput`.
7. Call `ContinueRun`.
8. Special-case the situation where input staging succeeds but continuation fails.

This logic is implemented in `stageContinuationForTab`.

The backend correctly validates and prepares continuation input, but the merge transaction itself is client-owned.

### Impact

Every client must reproduce the same behavior or risk:

- Dropping staged App context.
- Duplicating attachments.
- Replacing staged text accidentally.
- Handling partial success differently.
- Racing another accepted continuation update.

Revision checks reduce corruption risk, but they do not eliminate duplicated client protocol logic.

### Current behavior

The frontend performs a snapshot read, a full continuation replacement, and then a separate continue command.

### Backend enhancement or exposure needed

Add an atomic command such as `SubmitContinuation`.

```go
type SubmitContinuationCommand struct {
    Meta                  CommandMeta
    SessionID             SessionID
    RunID                 RunID
    ExpectedInputRevision Revision

    InputDelta         UserInput
    ConfigurationPatch *SessionConfigurationPatch
    ContinueImmediately bool
}
```

The backend should own merge semantics:

- Non-empty text replaces staged text.
- Attachments append and deduplicate.
- Backend-staged App context remains unless explicitly removed.
- Configuration applies only to the next operation.
- Continuation can be staged and started atomically.

Keep `UpdateContinuationInput` as a lower-level full-replacement API for advanced editors if needed.

Also add narrow backend commands such as:

- `ClearStagedAppContext`
- `RemoveStagedAppContext(ids)`

Clients should not need to overwrite the full continuation structure to remove staged App context.

#### Existing backend pieces to reuse

- `prepareSubmission`
- `beginContinuationModelLocked`
- `StageAppContext`
- Continuation revision checks
- Command idempotency and retry handling

No LLM Artifactory change should be required.

### Frontend behavior after filling the gap

The frontend sends one semantic command:

```text
Submit this continuation input and continue.
```

It no longer:

- Fetches `Run` state only to merge data.
- Reconstructs continuation input.
- Performs a two-command transaction.
- Handles partial staging and continuation failure as a special Composer case.

## Backend summaries lose excluded and replaced inherited contributions

### Gap

The backend composition code applies remove and replace overrides by deleting entries from effective maps and slices.

Examples include:

- `applyInstructionOverrides`
- `applyToolOverrides`
- `applySkillOverrides`
- `applyMCPOverrides`

When an inherited `Agent` or `Workspace` contribution is removed, it is usually removed before the final `OperationResolutionSummary` is built.

As a result, the summary can show what was included, but cannot reliably show:

- Inherited from `Agent`.
- Removed by `Session` override.
- Replaced by `Session` override.
- Excluded because another selection displaced it.

### Impact

This does not break execution, but it weakens transparent composition.

The frontend cannot authoritatively render the desired explanation model:

- Inherited.
- Added.
- Removed.
- Replaced.
- Unavailable.
- Changed.
- Blocked.
- Used.

The frontend may infer draft exclusions from local Composer state, but it cannot reliably explain historical operations or source changes using backend truth.

### Current behavior

The frontend retains some local exclusion state, such as `Agent` capability exclusions, and can show it while editing.

After operation preparation, the backend snapshot usually contains only effective values and some unavailable diagnostics.

### Backend enhancement or exposure needed

Treat operation preparation as building both:

- Effective contributions.
- A resolution ledger.

Before removing or replacing an inherited contribution, append a resolution item such as:

```go
ResolutionItem{
    Key:    ...,
    Kind:   ...,
    Owner:  ConfigurationOwnerAgent,
    State:  ResolutionStateExcluded,
    Reason: "removed_by_session_override",
}
```

For replacement, use machine-readable reasons such as:

- `replaced_by_session_override`
- `replaced_by_run_override`

For source-policy displacement, use reasons such as:

- `blocked_by_policy`
- `unavailable`
- `stale`

The existing `ResolutionItem` shape is close to sufficient. An explicit reason enum would be preferable to a free-form string.

No LLM Artifactory change should be required because the composition resolver already knows source ownership and capability origins.

### Frontend behavior after filling the gap

- The Composer can still show local draft changes before submission.
- Historical message and configuration panels render backend-produced resolution records.
- A CLI can explain exactly why a `Tool`, `Skill`, MCP server, or instruction was missing.
- No frontend reconstruction of removed inherited state is needed.

## Attachment and external-context limits are not centrally enforced by the Session runtime

### Gap

The backend correctly performs canonical attachment admission and snapshots references, but input limits are incomplete and inconsistent with frontend limits.

Examples include:

- The frontend has `MAX_SINGLE_ATTACHMENT_BYTES = 16 MiB`.
- The frontend limits active directory attachments to roughly `1,028`.
- Backend directory walking allows up to `4,096` or `8,192` files depending on the path.
- `BuildAttachmentForFile` does not enforce a file-size cap.
- Binary local attachment reading can read entire files into memory.
- URL text fetch has a `100k` text limit, but visible binary and image-size bounds are unclear.
- `prepareMCPResources` reads MCP resources and prompts, then concatenates text without a clear per-item or total context budget.
- No central combined budget is visible across attachments, `Workspace` context, MCP resources, MCP prompts, App context, and generated instruction text.

### Impact

A CLI or alternate UI can bypass frontend-only limits.

Potential outcomes include:

- Large file or base64 memory pressure.
- Provider request-payload growth.
- Context-window overflow.
- Different behavior between the React UI and a CLI.
- Silent provider failure instead of deterministic `Session` diagnostics.
- No durable explanation of truncation or omission.

### Current behavior

There are partial protections:

- Directory traversal has a bounded file count.
- URL text fetch has a maximum length.
- The frontend rejects or marks oversized attachments.
- The `Workspace` runtime appears to have its own contribution and truncation data.

However, there is no clearly centralized `Session`-owned budget policy.

### Backend enhancement or exposure needed

Introduce a runtime-owned context-budget policy with two phases.

#### Admission limits

At `Session` command admission, enforce limits such as:

- Maximum attachment count.
- Maximum per-file bytes.
- Maximum aggregate attachment source bytes.
- Maximum directory file count.
- Maximum directory aggregate bytes.
- Maximum URL fetch bytes.
- Maximum App-context bytes.

Reject input or record bounded overflow deterministically.

#### Operation-preparation budget

During operation preparation, enforce limits such as:

- Maximum total injected context bytes or tokens.
- Maximum MCP resource bytes.
- Maximum MCP prompt bytes.
- Maximum `Workspace` context bytes.
- Maximum generated instruction bytes.
- Provider or model prompt budget.

Use deterministic prioritization and diagnostics:

- `included`
- `truncated`
- `omitted`
- `rejected`

Persist the result in operation-preparation data.

The current summary may need richer states than `included`, `excluded`, `unavailable`, and `stale`, such as:

- `truncated`
- `budgetOmitted`

#### Likely dependency work

This is primarily `Session`, `runtimesetup`, attachment, and MCP work.

Possible additional work includes:

- `llmtools-go` may need bounded local-file, binary, and image-read APIs if it currently returns full data.
- Model runtime or LLM Artifactory may need to expose usable context-window and token-budget capability data.
- The `Workspace` runtime already has useful byte and truncation information that can align with the shared budget policy.

### Frontend behavior after filling the gap

The frontend can still show early warnings, but it is not the authority.

After submission, it renders backend truth:

- File accepted.
- File rejected as too large.
- Directory partially admitted.
- MCP resource truncated.
- `Workspace` context omitted because of budget.

A CLI receives the same limits and diagnostics automatically.

## Recommended implementation order

### Foundational runtime correctness

- Generic non-tool waits, Gap 3.
- MCP App-context provenance and approval, Gap 5.
- MCP `Session` readiness and authentication flow, Gap 4.

These changes remove the largest remaining places where frontend behavior acts as authority.

### Functional correctness

- `Tool` user-argument semantics, Gap 6.
- Backend-owned `Session` `Skill` instruction mode, Gap 7.
- Input and context budget policy, Gap 10.

### Client neutrality and explainability

- Configuration patch and preview, Gap 1.
- Default `Session` intent, Gap 2.
- Excluded and displaced resolution ledger, Gap 9.
- Atomic continuation command, Gap 8.

## Net result after these changes

After these gaps are closed, a frontend or CLI reduces to:

1. Query catalogs, defaults, and previews.
2. Maintain local draft and UI state.
3. Send typed intent and patch commands.
4. Render `Session` and `Run` snapshots.
5. Resolve backend-owned waits.

It no longer needs to own:

- Configuration merge semantics.
- MCP readiness orchestration.
- MCP context approval state.
- `Tool` configuration execution semantics.
- `Skill` instruction-rendering semantics.
- Continuation merge transactions.
- Context safety limits.
- Historical exclusion reconstruction.

At that point, the backend is not only an execution runtime. It is also a genuinely UI-neutral agent and Composer runtime.
