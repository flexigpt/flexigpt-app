# Consolidated UX degradation, completion, and cross-verification checklist

This consolidates the original comparison, the rolling notes, the post-patch notes, and the latest frontend patch set. It intentionally excludes legacy-data migration concerns such as old tab-storage restoration, old conversation-store search, and export-schema continuity.

Status meanings:

- ✓ Patched — verify: addressed by the described frontend patches; listed verification checks confirm behavior in the assembled branch.
- ◐ Partially covered: some user-facing behavior is now present, but the full old flow or target behavior is not yet complete.
- ○ Frontend follow-up: can be completed with current frontend/facade contracts.
- △ Backend/display-contract dependency: current frontend contracts do not carry enough information for full parity.

## Implementation status

- ◐ Agent selection is now visibly represented, but Agent recipe application is still incomplete.
  - Original degraded behavior: selecting an Agent changed only the Agent label while Model, Tools, Skills, MCP, instruction sources, and draft state continued showing prior Composer state.
  - Required UX: selecting an Agent must make it clear what Agent intent has been selected, what it contributes, and what remains user-controlled.
  - Patch status: the latest patch adds `selectedAgentResolution` in `use_agent_manager.ts` and an Agent selected badge plus capability summary in `agent_dropdown.tsx`. The dropdown now shows available/unavailable Agent capability records grouped as Model, Tools, Skills, MCP, and Instructions.
  - Cross-verify: select an Agent, reopen the Agent menu, verify:
    - the trigger includes Agent selected;
    - the selected Agent row says Agent selected, not locally “Applied”;
    - capability counts and individual resolution rows appear;
    - unavailable required capabilities show their status;
    - restored Session Agent selection reloads a capability preview after catalog availability.
  - Remaining: the preview is descriptive only. It does not populate ordinary Composer controls with the actual Agent recipe.

- △ Agent recipe materialization into Composer controls remains open.
  - Original degraded behavior: old Agent selection immediately applied:
    - Model;
    - Tool and web-search selections;
    - enabled/active Skills;
    - MCP selections;
    - Agent instruction sources;
    - model-default prompt preference;
    - empty-draft starter text.
  - Required UX: an Agent-selected Composer must visibly show its effective baseline without making frontend preview authoritative for execution. Users need to understand what will be used before Run admission.
  - Current state: the Agent dropdown now shows capability occurrences, but `resolveAgent()` only provides capability-resolution records. It does not provide the complete display-safe effective starter recipe that old `prepareAgentStarter()` provided.
  - Blocked by: a management-facade or Session-preparation display projection containing:
    - resolved Model identity/display data;
    - resolved Tool choices and metadata;
    - Skill mode/use state;
    - MCP server/resource/prompt selections;
    - materialized instruction sources;
    - starter-text metadata;
    - model-default prompt preference.
  - Relevant boundary: `agentManagementAPI`, Session operation preparation summary, and the Composer’s Agent/Context Bar presentation layer.

- ○ Agent model ownership still has a visible-model versus submitted-model mismatch.
  - Original degraded behavior: a user could select Agent A, continue seeing Model A in the dropdown, while the submitted configuration delegated actual model selection to the Agent, potentially using Model B.
  - Required UX: the displayed Model and provider-dependent controls must describe either:
    - the effective Agent Model; or
    - an explicit “Agent chooses model” state.
  - Current state: the latest patch fixes a related restoration case: a saved explicit Session Model is no longer silently switched back to Agent ownership after reopen. The ordinary “select a new Agent” path still leaves the existing displayed model in place.
  - Cross-verify: select an Agent with a different Model than the currently visible one, inspect:
    - Model dropdown;
    - reasoning controls;
    - web-search compatibility;
    - advanced parameters;
    - submitted `modelSelectionSource`.
  - Completion needed: frontend-only ownership handling in `use_context_state.ts` and `use_agent_manager.ts`, informed by an effective Agent Model preview.

- ○ Saving Advanced Parameters after Agent selection can still switch ownership to Session Model selection.
  - Original degraded behavior: saving a timeout or output option after selecting an Agent changed `modelSelectionSource` to `'session'`, even when the user had not manually changed Model.
  - Required UX: editing request parameters should preserve Agent Model ownership unless the user explicitly selects a Model.
  - Current state: this remains open. `applyAdvancedModel()` still sets Session model ownership unconditionally.
  - Cross-verify: select an Agent, open Advanced Parameters, change only timeout or streaming, save, then inspect the next submitted configuration.
  - Completion needed: a frontend-only ownership rule distinguishing:
    - explicit Model selection;
    - Agent-selected Model;
    - request-parameter override.

- △ Agent-derived remove/override semantics remain incomplete.
  - Original degraded behavior: old Agent recipe contents became ordinary visible Composer state. A user could remove an Agent Tool, disable an Agent Skill, deselect Agent MCP, or remove Agent instruction text.
  - Required UX: inherited Agent contributions need visible source badges and explicit override actions:
    - inherited;
    - user-added;
    - user-removed;
    - unavailable;
    - changed since selected.
  - Current state: the latest patch gives an Agent summary but does not add per-item Agent-origin badges or persistent Remove overrides. Existing Session intent types support collection `Add`/`Remove` semantics, but current Composer payload construction largely emits additions.
  - Cross-verify: select an Agent, remove a Tool/Skill/MCP item from visible local Composer state, submit, and inspect whether the backend still receives the Agent baseline.
  - Completion needed: a frontend Composer baseline/override store plus serialization of explicit removal overrides through existing `SessionConfiguration` override types.

- △ Agent instruction sources and Agent starter text remain unavailable as editable Composer content.
  - Original degraded behavior: old Agent application added individually selectable Agent instruction sources and inserted Agent starter text only into an empty draft.
  - Required UX: users need to see:
    - each Agent instruction source;
    - source text/metadata;
    - selected/unselected state;
    - Agent starter text availability;
    - whether a workflow URL draft takes precedence.
  - Current state: Agent capability summary can show a generic `text` or `skill` occurrence, but not materialized instruction text or starter content.
  - Blocked by: display-safe Agent starter recipe/materialization data from the management façade or Session preparation projection.

- ○ System-instruction source inspection and Skill-plugin search remain minor frontend parity gaps.
  - Original degraded behavior: old instruction-source rows exposed source text and tags through hover/title content; old Skill search included Plugin identity.
  - Required UX: a selected instruction source should be inspectable, and instruction Skills should be searchable by Plugin identity where that metadata already exists.
  - Current state: names/tags remain visible, but source text preview and Plugin-oriented search are reduced.
  - Completion needed: frontend-only changes in `skills_bottom_bar_chip.tsx` using existing:
    - `SystemInstructionSource.text`;
    - `SystemInstructionSource.sourceTags`;
    - Skill catalog `plugins` metadata.

- ◐ Quick Create/Fork Skill flows remain available, but exact old defaults differ.
  - Original degraded behavior: the old Composer used a compact creation/fork flow, often preferred the source Plugin, and forked rendered instruction text.
  - Required UX: users must still be able to create/fork an instruction Skill and immediately use it as instructions.
  - Current state: capability remains:
    - Create exists;
    - Fork exists;
    - destination selection exists;
    - `ManagedSkillEditorModal` exists;
    - saved simple instruction Skills are added to Composer instruction sources.
  - Difference to verify: Fork copies source document rather than rendered instruction text; destination selection is explicit rather than preselected.
  - Completion status: no blocking capability loss. Exact old convenience/default behavior remains a frontend/product choice.

- ○ Workspace active-Skill toggles still need end-to-end submission verification.
  - Original degraded behavior: Workspace modal exposed an Active toggle, but Workspace-owned active Skills were excluded from submitted active-Skill selections.
  - Required UX: toggling a selected Workspace instruction Skill to Active must affect the Run’s effective active Skill set.
  - Current state: Workspace context/Skill deselection restoration was patched, and general Skill lifecycle synchronization was patched. The separate question is whether the Workspace Active toggle reaches `configuration.activeSkills`.
  - Cross-verify: attach a Workspace, select one Workspace instruction Skill, toggle Active, submit, then inspect the Run’s effective/active Skills. Repeat with Active off.
  - Completion needed: likely frontend/session-intent work using existing `SessionConfiguration.activeSkills`; backend semantics should confirm Workspace Skill Artifact refs are valid active overrides.

- ✓ Patched — Workspace Context/Skill deselection persistence.
  - Original degraded behavior: unchecking a baseline Workspace Context or Skill reappeared after reopening because removal keys used incompatible formats.
  - Required UX: deselected baseline Workspace contributions remain deselected after send, reopen, refresh, and later submission.
  - Patch status: rolling notes identify normalization of Workspace override keys and Workspace MCP removal handling as fixed.
  - Cross-verify: for Context, Skills, and Workspace MCP:
    1. attach Workspace;
    2. uncheck a baseline item;
    3. send;
    4. close/reopen Session;
    5. verify item remains unchecked;
    6. submit again and inspect effective backend selection.

- ○ Template resource defaults and attachment-error ordering remain changed.
  - Original degraded behavior: old resource-backed templates selected attachable resources by default and inserted text before attempting attachments.
  - Required UX: if preserving old behavior is desired:
    - all verified resource paths start selected;
    - template text survives a resource-attachment exception;
    - user gets a warning rather than losing insertion.
  - Current state: new flow uses empty initial resource selection and can block text insertion when attachment preparation throws.
  - Completion needed: frontend-only adjustments in `skill_template_bottom_bar_chip.tsx`.
  - Cross-verify: insert a resource-backed template without touching checkboxes, then repeat with an intentionally failing attachment path.

- ✓ Patched — editing an earlier user message loads its original Composer state.
  - Original degraded behavior: edit mode highlighted the message but left the Composer empty or containing unrelated draft text.
  - Required UX: Edit must load:
    - message text;
    - attachments;
    - Tool choices/issues;
    - MCP/App context;
    - displayed Tool outputs;
    - Skills;
    - Workspace selection;
    - then restore pre-edit state on Cancel.
  - Patch status: rolling notes list this as fixed.
  - Cross-verify: edit user messages containing each combination of:
    - text only;
    - attachments;
    - Tools;
    - MCP selection;
    - Skills;
    - Workspace;
    - Tool outputs;
    - App context.
      Then cancel and verify the pre-edit Composer state returns.

- △ Editing/resending Tool-output follow-ups remains blocked by Session input shape.
  - Original degraded behavior: old editor could serialize Tool outputs into the next inference input.
  - Required UX: if Tool-output chips are loaded into edit mode and make Send available, those Tool results must reach the next Run semantically.
  - Current state: visual loading is patched, but `SessionUserInput` has no Tool-output field.
  - Blocked by: a Session-native representation for:
    - carried Tool outputs;
    - Tool-result references;
    - edit/resend reuse semantics; or
    - a command-specific Tool-result continuation payload.
  - Cross-verify: edit a user turn containing Tool outputs, send, and inspect whether the next operation receives those outputs. Current contract indicates it cannot.

- ✓ Patched — edit-and-resend bottom-follow behavior.
  - Original degraded behavior: edit/resend could leave the user at the earlier message while replacement output started elsewhere.
  - Patch status: latest patch adds `scrollTabToBottomSoon(tabId)` after accepted `editAndResend`.
  - Cross-verify: scroll to an earlier user message, edit/resend it, verify viewport follows the replacement operation/output.

- ✓ Patched — snapshot-backed live text and thinking rendering.
  - Original degraded behavior: Session snapshot `streamedText`/`streamedThinking` existed but empty legacy stream buffers masked them, leaving blank busy assistant bubbles and “Waiting for response.”
  - Required UX: queued/preparing/running operations visibly progress from Waiting to Thinking/Streaming using Session snapshot content.
  - Patch status: rolling notes mark live text/thinking and active-operation placeholder behavior fixed.
  - Cross-verify: test:
    - queued operation with no text;
    - thinking-only operation;
    - streamed text;
    - streamed text plus thinking;
    - background tab then foreground tab;
    - final transcript replacement.
      Verify no blank busy body and no disappearance at terminal transition.

- ◐ Send acknowledgement before the first authoritative snapshot remains weaker than old optimistic UI.
  - Original degraded behavior: old send immediately inserted user message and assistant “Waiting for response” placeholder.
  - Required UX: after accepted `StartRun`, user sees immediate acknowledgement before the next snapshot arrives.
  - Current state: active-operation placeholder behavior is improved once a snapshot contains an operation. There may still be a short accepted-command interval showing only disabled Composer/tab busy state.
  - Completion needed: frontend-only ephemeral accepted-Run status reconciled against snapshot state.
  - Cross-verify: delay Session snapshot delivery after `StartRun` acceptance and inspect visible feedback.

- ◐ Waiting-Run continuation authoring is now partially restored.
  - Original degraded behavior: Composer was locked for all Waiting Runs; users could not add normal continuation text, attachments, or configuration.
  - Patch status: latest patch adds a continuation mode:
    - `ConversationArea` identifies a waiting Run with no pending wait and no executable pending Tool calls;
    - `TabInputPane` routes submission to `stageContinuationForTab`;
    - `use_send_message.ts` calls existing `updateContinuationInput` then `continueRun`;
    - Composer displays “This input will be staged for the waiting Run and then continued.”
  - Cross-verify: create a waiting Run with continuation available:
    1. enter text;
    2. attach a file;
    3. alter supported Composer context;
    4. send;
    5. verify `updateContinuationInput` accepts staged content and `continueRun` starts the next operation.
  - Remaining: this mode intentionally rejects local `payload.toolOutputs`, because Session input has no Tool-output field. It also does not provide a generic Composer for waits that still have pending Tool calls or an unresolved approval/input wait.

- ✓ Patched — manual Tool execution controls are substantially covered by Session controls.
  - Original degraded behavior: old local Tool queue supported individual run, batch run, fast-forward, skip/error behavior, and manual follow-up.
  - Patch status: rolling notes identify these as fixed or substantially covered:
    - individual Run action performs run-only;
    - waiting Run exposes Run pending tools;
    - waiting Run exposes Run tools and continue;
    - Continue run appears when ready;
    - failed “Submit error” uses `completeCallAsError`;
    - actions locate the owning Run;
    - command errors surface in chat;
    - one-batch automation hold is exposed.
  - Cross-verify: test one Run with:
    - one pending Tool;
    - multiple pending Tools;
    - failed Tool;
    - skipped Tool;
    - automatic Tool batch;
    - Tool in an older Run;
    - command revision conflict/error.
      Verify Run IDs, call IDs, status transitions, and visible errors.

- ○ Result-level retry/re-run remains incomplete.
  - Original degraded behavior: old Composer could retry eligible errored/skipped Tool outputs.
  - Required UX: a displayed skipped/error result can offer Retry/Run only when it maps to a valid owning Run/Tool-call action.
  - Current state: call-level Retry is covered. A resolved call with an Error/Skipped selected result may project as succeeded, leaving no retry action.
  - Completion needed: frontend ownership mapping from `UIToolOutput` to owning Session Run/Tool call, plus a validity check before exposing a result-level retry action. Existing `retryCall` can be used only when the original call remains retryable.

- ○ Tool metadata, automatic badges, and rich Tool output remain incomplete.
  - Original degraded behavior: old message chips/details had Tool choice/plugin/version metadata, Auto markers, original arguments, typed file/image/text output items, and web-search detail.
  - Required UX: message-level Tool rendering should preserve safe display metadata and typed outputs.
  - Current state: Session projection still reduces many regular Tool calls/results to minimal fields and text/structured content.
  - Frontend-only portion: `SessionToolCall.automatic` already exists and can be projected into `UIToolCall` for Auto badges. Provider-call/result-ID matching can also be corrected in projection where existing IDs are available.
  - Backend/display-contract portion: typed file/image result items, complete Tool binding metadata, and full selection provenance need Session result/projection data.
  - Cross-verify: compare Go Tool, Skill Tool, MCP Tool, Web Search, text result, file result, image result, error result, and skipped result.

- ✓ Patched — active Skill lifecycle state is synchronized back into Composer.
  - Original degraded behavior: Skill lifecycle changes during a Run did not update Composer active-state indicators.
  - Patch status: rolling notes mark Session `skillSession.enabled`/`active` synchronization into mounted and pending Composer instances as fixed.
  - Cross-verify: invoke a Skill lifecycle Tool that loads/unloads/activates/deactivates a Skill, then verify:
    - Skills menu enabled/active badges;
    - Workspace Skill Active toggle;
    - next submission’s Skill state.

- ◐ Terminal error/cancellation presentation is partially restored.
  - Original degraded behavior: old frontend appended explicit terminal lines for stopped, interrupted, and failed partial output.
  - Patch status: latest patch adds visible terminal banners for `Failed`, `Cancelled`, and `Incomplete` projected message statuses, and Session operation runtime metadata can reach message details.
  - Cross-verify: test:
    - cancellation before output;
    - cancellation after partial text;
    - incomplete operation;
    - failure with text;
    - failure without text;
    - failure with thinking.
  - Remaining: full Run-level outcomes are not always synthesized into a terminal assistant message. If the backend records only `run.outcome` without an associated transcript/operation message status, presentation remains incomplete.

- ◐ Session wait and approval behavior is functionally stronger but still differs from old structured presentation.
  - Original degraded behavior: old MCP approval modal showed named Server, Tool, Risk, Reason, and Arguments fields. Generic waits lacked structured JSON input.
  - Patch status: rolling notes mark optional JSON wait input and `resolveWait(input)` as fixed.
  - Cross-verify: test a wait with:
    - no input schema;
    - input schema and valid JSON;
    - invalid JSON;
    - approve/deny/cancel resolutions;
    - resolution conflict/error.
  - Remaining: MCP approval is still rendered as generic wait summary rather than the former structured approval card. This is frontend-only if the wait summary contains the relevant metadata.

- △ Historical message provenance remains the largest display-projection gap.
  - Original degraded behavior: old message bars/details showed selected Tools, web-search config, unavailable Tool selections, MCP context, Skills, Workspace selection/usage, model context, token usage, and debug details.
  - Required UX: each historical turn should expose safe, per-operation provenance:
    - selected and used Model;
    - Tools and bindings;
    - MCP selections;
    - Skills;
    - Workspace selection and actual usage;
    - usage counters;
    - safe diagnostics/debug data.
  - Patch status: latest patch improves `modelRef` and debug fallback from operation runtime metadata. Full provenance remains absent.
  - Blocked by: Session transcript/operation display projection data. Existing `ConversationMessage` UI fields and chips can render it once supplied.
  - Cross-verify: inspect a completed Session message containing Model overrides, Tools, web search, MCP resources/prompts, Skills, Workspace Context, and usage. Current frontend still lacks most message chips.

- ✓ Patched — background restored tabs now restore Composer configuration after later activation.
  - Original degraded behavior: background persisted tabs could receive snapshots before their Composer mounted, then later show messages with default Composer controls.
  - Patch status: rolling notes mark delayed/background Composer configuration restoration fixed.
  - Cross-verify: persist multiple Session tabs, restart, leave one background, activate it later, verify restored:
    - Agent;
    - Model;
    - Tools;
    - Skills;
    - MCP;
    - Workspace;
    - instructions;
    - history setting.

- ✓ Patched — model catalog timing during Session restore.
  - Original degraded behavior: restoring Session configuration before model catalog load dropped saved Model/request/history state and later installed defaults.
  - Patch status: rolling notes mark pending Session configuration restoration until catalog completion as fixed.
  - Cross-verify: delay model catalog, open a Session with explicit Model/overrides/history, then verify final Composer state after catalog completion.

- ✓ Patched — saved explicit Model plus Agent restoration.
  - Original degraded behavior: a Session with Agent A plus explicit Model B reopened with B displayed but Agent ownership, causing later send to omit B and delegate back to Agent.
  - Patch status: rolling notes mark this restored-model ownership case fixed.
  - Cross-verify: create Agent A + explicit Model B Session, reopen, alter a parameter, send, verify B remains explicit.

- ✓ Patched — model-default instruction clear and restored hybrid reasoning UI.
  - Original degraded behavior: re-enabling model-default instructions could retain `SystemPrompt` clear; restored hybrid-off state could show contradictory “On” UI.
  - Patch status: rolling notes mark both fixed.
  - Cross-verify: disable/re-enable model default instructions and restore Hybrid Off state across a Session reopen.

- ✓ Patched — MCP restoration identity, argument definitions, and partial recovery.
  - Original degraded behavior: restored MCP top-level server selection could use Artifact ID instead of runtime ID; restored prompt/template arguments lacked editable definitions; one unavailable MCP selection could block unrelated restoration.
  - Patch status: rolling notes mark all three fixed.
  - Cross-verify: restore:
    - MCP server with distinct Artifact/runtime IDs;
    - selected Tool exposure;
    - resource template arguments;
    - prompt arguments;
    - one unavailable server plus valid Tool/Workspace/Skill state.

- ✓ Patched — active Session load/busy/error/close behavior.
  - Original degraded behavior: loading an active Session could mark it idle; load failure could leave permanent spinner; closing busy tab could leave Run continuing.
  - Patch status: rolling notes mark active-run state retention, visible load errors, and best-effort Run cancellation on tab close as fixed.
  - Cross-verify: open active Session through search, force a load failure, and close a busy tab. Inspect busy state, Composer lock, error alert, and Run cancellation receipt/snapshot.

- △ Reopen behavior still depends on Session configuration versus latest Run configuration contract.
  - Original degraded behavior: old conversation restoration derived current Composer state from latest user-message context. New UI restores `snapshot.session.configuration`.
  - Required UX: reopening a Session reproduces the configuration the user expects to continue from.
  - Current state: this remains contract-dependent. Later Run configuration may differ from Session configuration.
  - Blocked by: either:
    - backend guarantee that accepted Run configuration updates Session configuration; or
    - an explicit latest-effective-Run configuration projection for frontend restoration.
  - Cross-verify: change Composer configuration on a later Run, close/reopen, compare restored controls with actual latest Run configuration.

- ✓ Patched — MCP App draft messages route to owning Session.
  - Original degraded behavior: an App in Session A could add a draft into currently selected unrelated tab B.
  - Patch status: latest patch adds Session-to-tab routing:
    - existing owner tab receives draft;
    - absent owner Session opens in a scratch tab;
    - draft is queued until target Composer mounts/restores configuration.
  - Cross-verify: test:
    1. App in tab A;
    2. select tab B;
    3. request Add draft from App;
    4. verify tab B draft is unchanged;
    5. verify A or newly opened owner tab receives the draft.

- ✓ Patched — MCP App Tool calls can await Session Tool results.
  - Original degraded behavior: App `tools/call` returned only acceptance, while old Apps received Tool result content/structured data in the JSON-RPC response.
  - Patch status: rolling notes mark eventual Session Tool result response behavior fixed while retaining notifications.
  - Cross-verify: App invokes Tool, waits for JSON-RPC response, verify:
    - result content;
    - structured content;
    - error state;
    - timeout/error behavior;
    - additional `tool-result` notification behavior.

- ✓ Patched — MCP App context view identity is retained through staging/restoration.
  - Original degraded behavior: restored App context could lose `viewID`, then fail Session submission filtering.
  - Patch status: rolling notes mark `viewID` retention fixed.
  - Cross-verify: stage App context, persist/reopen, inspect staged/restored update, then submit/continue and verify it is accepted.

- △ MCP App binary content and policy-aware host behavior remain open.
  - Original degraded behavior: App context can contain richer MCP content than current staged text/URI/name/description/MIME projection; old host also reflected per-server App policy for Tool advertisement and confirmation.
  - Required UX: App context must preserve intended content types, and host capability/confirmation behavior must match Session App policy.
  - Current state: view identity and staged context UX are fixed; binary/image/audio/resource-blob payloads and policy-aware host capability declarations remain limited.
  - Blocked by: Session App-context DTO and Session App policy display/projection contract.
  - Cross-verify: App requests:
    - text context;
    - structured context;
    - image/audio/resource content;
    - link open under no-confirm policy;
    - context update under no-confirm policy;
    - Tool call with App Tool calls disabled.

- ✓ Patched — MCP App readable server label.
  - Original degraded behavior: App header showed opaque runtime server ID.
  - Patch status: latest patch resolves the runtime ID through `mcpManagementAPI.findComposerServerByRuntimeID` and displays server name/logical identity with runtime-ID fallback.
  - Cross-verify: open an App for a known MCP server and verify header/dialog text shows readable server identity; disable/remove catalog visibility and verify runtime ID fallback remains safe.

- ○ MCP App call/input association still needs verification and likely a projection adjustment.
  - Original degraded behavior: projected Tool calls can use provider call ID while projected Tool outputs use runtime call ID, preventing `ChatMessage` from finding the originating call arguments for an App.
  - Required UX: App view receives the Tool input arguments that produced the App result.
  - Current state: not listed as fixed in rolling notes.
  - Completion needed: frontend projection should normalize call/output join identity using existing `providerCallID`/`callID` fields consistently.
  - Cross-verify: invoke an App-capable MCP Tool with arguments and verify `ui/notifications/tool-input` receives those arguments.

- ✓ Patched — second-turn automatic title refinement, if latest patch is applied.
  - Original degraded behavior: new Session creation generated only an initial title; old UI refined title using early first/second user-message scoring.
  - Patch status: latest patch adds `maybeAutomaticallyRenameSession()` in `use_chats_controller.ts`, using existing `renameSession`. This supersedes the older rolling-note statement that title refinement remained absent.
  - Cross-verify: start with a weak first prompt, send a stronger second user prompt, verify title refinement occurs unless manual title lock is set.

- ✓ Patched — close-during-generation cancellation, if rolling patch is applied.
  - Original degraded behavior: old tab disposal aborted active completion; new Session tab disposal originally removed UI state without cancelling Run.
  - Patch status: rolling notes mark best-effort `cancelRun` on disposed busy tab fixed.
  - Cross-verify: start a streaming Run, close its tab, observe Session Run transitions toward cancellation and confirm no orphaned active Run remains.

## Next steps

- [ ] Treat Agent effective recipe/baseline/override UX as the next largest frontend-plus-display-contract work item.
- [ ] Define a Session-native semantic Tool-result carry/reuse contract for edit/resend and continuation workflows.
- [ ] Add safe per-message/per-operation display projection for Model, Tool, MCP, Skill, Workspace, usage, and diagnostics.
- [ ] Verify Workspace Active Skill submission and add active override serialization if it remains absent.
- [ ] Complete result-level Tool retry behavior using owning Run/Tool-call identity.
- [ ] Add frontend-only parity refinements for instruction-source inspection, Plugin search, template resource defaults, and structured approval display.
- [ ] Verify every ✓ item above against actual assembled code and Session snapshots before treating it as closed.
