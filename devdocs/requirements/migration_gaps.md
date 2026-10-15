## Current state

**The new version retains much of the visible chat UI, but it does not preserve all of the old end-to-end behavior.** There are both missing user actions and controls that remain visible but no longer represent, submit, or restore the same state.

I used the `/home/ppipada/...` version as the baseline and `/mnt/data/...` as the new version. This is a source-level comparison, with your assumptions applied:

- Wails bridging and the management façades work.
- Shared `components`, `lib`, and `hooks` are unchanged.
- Backend execution is not being evaluated.
- A move from frontend execution to backend execution is **not**, by itself, counted as a loss.
- API calls are discussed only where a user choice is omitted, changed into another action, or not reflected back into the UI.

### Summary

| Area                                                     | Current comparison                                                                                             |
| -------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| Ordinary drafting, attachments, model-parameter controls | Largely retained                                                                                               |
| Applying an Agent and adjusting its recipe               | Not equivalent: the Agent selection changes, but its recipe is no longer reflected in the composer             |
| System instructions and quick create/fork                | Core actions remain; some inspection and workflow behavior has changed                                         |
| Workspace selections                                     | Active-skill choice is not submitted; deselections do not restore correctly                                    |
| Editing an earlier message                               | Edit mode appears, but the original message is not loaded                                                      |
| Live response text and thinking                          | Snapshot content is masked by an empty stream source                                                           |
| Manual tool follow-up                                    | Run-only, batch controls, editable follow-up, and stop-next-auto-batch behavior are missing                    |
| Message context, usage, and details                      | Several existing UI elements receive no data from the new projection                                           |
| Reopening conversations and restoring tabs               | Multiple concrete restoration gaps                                                                             |
| MCP Apps                                                 | Rendering remains, but tool responses, context staging, and confirmation preferences are not equivalent        |
| Search and ordinary tab navigation                       | Mostly retained; saved old-tab layout, automatic title refinement, and close-during-generation behavior differ |

References below are relative to `frontend/app`.

---

# 1. Composer: parameters, context, selections, and starters

## 1.1 Controls that remain available

There is no general removal of the ordinary composer controls.

| Flow                    | Retained behavior                                                                                                                                  |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Drafting                | Plain-text input/paste, multiline editing, large-text handling, editor scrolling, Enter submission and soft breaks                                 |
| Model selection         | Searchable, provider-grouped model menu                                                                                                            |
| Temperature             | Default, presets, custom value                                                                                                                     |
| Reasoning               | Hybrid Default/Enabled/Disabled, reasoning levels, token presets and custom budget                                                                 |
| Output effort/verbosity | Existing provider-dependent control and choices                                                                                                    |
| Previous turns          | Current turn only, 1/2/3, all, and custom count                                                                                                    |
| Advanced parameters     | Streaming, prompt/output token limits, timeout, reasoning summary, output format, JSON Schema fields and strictness, cache options, stop sequences |
| Attachments             | Files, folders, URLs, drag-and-drop, mode selection, individual/group removal                                                                      |
| Folder attachments      | Text/glob/regex/inverted filtering, add/remove matches, restore excluded files, overflow notices                                                   |
| Tool selection          | Per-message and conversation tools, enable/disable, Auto, options JSON, examples, formatting, required-option checks                               |
| Provider web search     | Selection, disable, configuration, provider compatibility checks                                                                                   |
| MCP selection           | Connection/authentication controls, discovery, tool exposure, resources, resource templates, prompts, arguments and completions                    |
| Workspace selection     | Attach/detach, directory registration, context/skill selection, refresh, diagnostics, MCP setup                                                    |

The normal parameter submission path also carries the sparse request preferences into `SessionConfiguration.modelOverrides`; simply replacing the old completion request with that configuration is not a loss.

The problems are in particular interactions and round trips below.

---

## 1.2 Applying an Agent no longer populates the composer recipe

### Old behavior

Selecting or reapplying an Agent populated the visible composer with:

- Its model.
- Its model-default-instruction preference.
- Its tools and web-search selection.
- Its enabled and active installed skills.
- Its MCP selections.
- Individually selectable Agent instruction sources.
- Its starting/request text, **only when the draft was empty**.

Applying an Agent with fewer capabilities also cleared the previous recipe’s tool, skill, and MCP selections.

### New behavior

`useAgentManager.selectAgent()` now:

1. Resolves the Agent.
2. Checks unavailable required capabilities.
3. Sets `modelSelectionSource` to `'agent'`.
4. Tracks the selected Agent.

It does not apply the resolved recipe to any of those composer controls.

### User-visible effects

- The Agent badge changes, but the model dropdown can still show the previous model.
- Tool, Skill, MCP, and system-source controls do not show the Agent’s contributions.
- The user cannot inspect and adjust the Agent-populated selection in the same way.
- **Reapply** no longer restores the visible recipe.
- The Agent menu’s **Clear to base / Reset to default** no longer clears the previous composer selections as the old recipe application did.
- An Agent request template no longer prefills an empty draft.

The last point also affects workflow starters: an explicit URL `draft` still loads, but an Agent-only starter no longer gets its starting text from the Agent.

**Sources:**
Old `chats/composer/composer_box.tsx::applyAgentRuntime` and `composer/agents/use_agent_manager.ts::selectAgent`; new versions of those functions.

### The displayed model and the submitted model can disagree

This is not only a missing preview.

The new submission builder deliberately omits `configuration.model` when the selection source is `'agent'`. Meanwhile, the displayed model, provider SDK, reasoning controls, web-search compatibility, and model-default prompt still come from the previous `selectedModel`.

For example:

1. The composer displays model A.
2. The user selects an Agent whose model is B.
3. The composer continues displaying A and A’s provider-dependent controls.
4. The submitted configuration delegates model selection to the Agent.

A functioning backend can execute B correctly, but the user is still configuring the UI as though A were selected.

**Sources:**
`chats/composer/composer_box.tsx::handleSubmit`, `composer/contextarea/context_bar.tsx`, `chats/session/session_intent.ts::buildSessionSubmissionIntent`.

---

## 1.3 Saving advanced parameters can change model selection after choosing an Agent

The new `applyAdvancedModel()` sets `modelSelectionSource` to `'session'` unconditionally.

Combined with the Agent/model-display mismatch above:

1. Model A remains displayed after selecting an Agent that supplies B.
2. The user opens Advanced Parameters and changes a timeout—or simply saves.
3. The selection source changes to `'session'`.
4. The next submission explicitly selects displayed model A.

The old Advanced Parameters save did not separately change model ownership.

**Source:**
`chats/composer/contextarea/use_context_state.ts::applyAdvancedModel`.

---

## 1.4 Removing Agent-provided context is no longer represented by the same controls

The old Agent recipe became ordinary editable composer state. Removing one of its tools, disabling an enabled skill, or deselecting one of its instruction sources changed what the next request contained.

In the new version:

- The Agent remains in the submitted configuration.
- Explicit tools, skills, instructions, and MCP selections are serialized as `Add` overrides.
- Explicit active skills are serialized as `Activate`.
- Those composer controls do not produce corresponding exclusions of Agent-provided items.

Consequently, removing something from the visible composer is not equivalent to excluding that contribution from the selected Agent.

This particularly affects:

- **Clear tools**.
- **Disable all skills**.
- Clearing system-instruction sources.
- Disabling an Agent-provided skill.
- Removing an individual Agent instruction source—which is no longer shown in the first place.

This finding is specifically about **inherited Agent contributions**. It is not a claim that removing an ordinary, explicitly added tool fails in every conversation.

**Source:**
`chats/session/session_intent.ts`, particularly `toolOverridesFromChoices`, `skillOverridesFromPayload`, and `instructionOverridesFromPayload`.

---

## 1.5 System-instruction inspection has lost information

### Selected instruction text is no longer inspectable from its source row

Old flattened/restored instruction-source rows exposed `source.text` in their hover title, including source tags.

The new rows show the name and tags, but no text preview and no replacement inspection action.

The source can still be selected or deselected, but the user cannot inspect the actual selected instructions from that row.

This concerns flattened/restored/Agent source rows; the old model-default row itself did not have the same text preview.

### Instruction-skill search no longer includes Plugin identity

Old instruction-skill rows showed the Plugin/skill identity, and search included `pluginSlug` and `pluginID`.

The new rows show the skill name and Root ID. Their search fields omit the available `plugins` metadata.

Searching instruction skills by a Plugin name therefore no longer follows the old behavior.

**Source:**
Old and new `chats/composer/skills/skills_bottom_bar_chip.tsx`.

---

## 1.6 Quick creation and forking remain, but the workflow is different

These actions have **not** disappeared.

| Aspect                                          | Old                                                                       | New                                                      |
| ----------------------------------------------- | ------------------------------------------------------------------------- | -------------------------------------------------------- |
| Create entry                                    | “Add new instruction skill”                                               | “Create”                                                 |
| Destination                                     | Selected within the creation form, with a preferred enabled custom Plugin | Separate destination chooser; initial selection is blank |
| Fork destination                                | Preferred the source’s custom Plugin where applicable                     | Explicit destination-selection step                      |
| Fork source                                     | Rendered instruction text                                                 | Copied source document                                   |
| Editor                                          | Simple instruction-skill form                                             | `ManagedSkillEditorModal`                                |
| Add newly saved simple instructions to composer | Rendered the created skill and selected it                                | Selects the saved document body directly                 |

The new save callback still adds an enabled, argument-free instruction document to the current system-instruction sources. So the basic create/fork-and-use outcome is represented.

However, it is no longer the same quick flow, and forking is now a **source-document copy**, not a **rendered-instruction copy**.

The body of `ManagedSkillEditorModal` is not in the supplied files, so I am not claiming any loss of its internal fields or validation.

There are also smaller visible changes:

- The old skill action-error alert had a **Dismiss** action; the new one does not.
- Enabled/active counts now count saved references directly rather than the old available-catalog intersection.
- A successful Agent selection does not clear an earlier `actionError`, so an old selection error can remain alongside the newly selected Agent.

**Sources:**
`chats/composer/skills/skills_bottom_bar_chip.tsx`; `apis/skill_management.ts`; `chats/composer/agents/use_agent_manager.ts`.

---

## 1.7 Workspace skill “Active” changes are not submitted

The Workspace selection modal still exposes an **Active** toggle for selected instruction skills.

Changing it updates the composer’s combined skill state. But submission includes only:

- Installed enabled skills.
- Installed active skills.
- Workspace context/skill membership selections.

The installed-skill getters explicitly exclude Workspace-owned skill references. The Workspace selection object has no active-skill field, and the Workspace intent builder does not add those active references to `configuration.activeSkills`.

**The resulting command is the same whether a Workspace skill’s Active toggle is on or off.**

In the old version, the combined active selection was applied to the skill session used for sending. The replacement path does not convey that choice.

**Sources:**
`chats/composer/workspaces/use_composer_workspace.ts::setSkillActive`;
`chats/composer/editor/editor_area.tsx` payload construction;
`chats/session/session_intent.ts::workspaceSelectionFromPayload` and `skillOverridesFromPayload`.

---

## 1.8 Templates retain insertion and resources, but some behavior changes

### Resource attachments are no longer selected by default

Old template configuration initialized the resource selection with all listed attachable resource paths.

The new modal initializes it as an empty list.

Therefore, the same sequence—choose a resource-backed template, then press Insert—now inserts text without the resources that were previously selected by default.

The capability remains through individual selection and **Select all listed**. This is a changed default, not removal of resource attachment support.

### A resource-attachment exception now prevents text insertion

Old order:

1. Insert rendered text.
2. Attach resources.
3. If attachment fails, retain the text and show a warning.

New order:

1. Prepare rendered text.
2. Attach resources.
3. Insert text.

If resource attachment throws, the new flow does not insert the template text. Returned per-resource warnings are handled separately and still allow insertion.

### Some template-list information is gone

The old list displayed:

- Exact argument counts.
- Exact resource counts.
- Per-Plugin counts.
- A separate “Templates with resource references” section.

The new list uses generic **Args** and **Resources** badges and does not retain those subdivisions/counts.

**Source:**
Old and new `chats/composer/skills/skill_template_bottom_bar_chip.tsx`.

---

# 2. Chat messages and edit/resend

## 2.1 Clicking Edit no longer loads the message

This is a direct break in the existing edit flow.

### Old

`beginEditMessageForTab()` built an `EditorExternalMessage` containing:

- Original text.
- Attachments.
- Tool choices and unavailable-selection information.
- MCP context and App context.
- Tool outputs.
- Enabled/active skills.
- Workspace selection.

It then called the composer’s `loadExternalMessage()` before setting edit mode.

### New

It only sets:

```ts
editingMessageId: id;
```

There is no message-loading call.

### What the user sees

- The old message receives its editing highlight.
- The composer displays the “Editing an earlier message” banner.
- The composer remains empty, or retains whatever unrelated draft was already there.
- Original attachments and context are not loaded.
- Sending can replace the old message with that unrelated draft and drop later messages.

The edit command itself exists and includes `replaceMessageID`. The missing part is loading the message that the user chose to edit.

Cancel also no longer restores the old pre-edit context behavior, because the snapshot normally created by `loadExternalMessage()` was never taken.

**Source:**
Old and new `chats/conversation/use_send_message.ts::beginEditMessageForTab`;
`chats/composer/editor/editor_area.tsx::loadExternalMessage`.

### Tool-output resend has a second break

Even if a draft contains historical tool outputs, the new `buildSessionSubmissionIntent()` does not serialize `payload.toolOutputs`.

The old path explicitly converted them into tool-output inputs.

Thus, loading the original message alone would not restore the old tool-output edit/resend flow.

**Sources:**
Old `chats/conversation/hydration_helper.ts::buildUserConversationMessageFromEditor`;
new `chats/session/session_intent.ts::buildSessionSubmissionIntent`.

### Edit/resend also loses its explicit scroll-to-response behavior

The new edit branch does not call `scrollTabToBottomSoon()`.

For an edit that shortens the transcript while the user is scrolled up, the unchanged scroll logic does not necessarily move to the replacement response, because it mainly reacts to increasing message count and existing auto-follow state.

The old edit path entered the normal response-start flow, which explicitly scrolled to the bottom.

---

## 2.2 Several message bars and details are no longer populated

The corresponding components still exist. The new `projectTranscriptMessage()` simply does not populate the fields that make them appear.

| Existing user-visible information           | Fields not supplied by the new projection | Result                                                                                         |
| ------------------------------------------- | ----------------------------------------- | ---------------------------------------------------------------------------------------------- |
| Tools selected for that user turn           | `toolSelections`, `uiToolChoices`         | Selected-tool chips/details do not appear                                                      |
| Web-search configuration used for that turn | `uiToolChoices`                           | Configuration chip is absent, independently of actual search activity                          |
| Unavailable historical Tool selection       | `uiToolSelectionIssues`                   | Historical unavailable-selection warning/details are absent                                    |
| MCP selection                               | `mcpContext`                              | Server count, exposure, instruction flags, resource/prompt counts are absent                   |
| Enabled/active Skills for the turn          | `enabledSkillRefs`, `activeSkillRefs`     | Skills chip and counts are absent                                                              |
| Workspace selection and actual usage        | `workspaceSelection`, `workspaceUsage`    | Requested selection, included/truncated/excluded outcomes, revision and diagnostics are absent |
| Token usage                                 | `usage`                                   | Input/output arrows and cached/uncached/reasoning-token tooltip are absent                     |
| Request/response/provider debug details     | `debugDetails`, `uiDebugDetails`          | Normal successful messages no longer expose the old details content                            |

The projection also does not recover these from transcript metadata or operation preparation data.

This is not a finding that the backend cannot supply the information. **The current frontend projection does not carry it to the existing UI.**

Some related surfaces do remain populated:

- Ordinary attachment chips.
- Tool-call/result chips, with the limitations below.
- App-context history counts.
- Final text and reasoning from protocol outputs.
- URL citations from protocol outputs.
- Error details when `message.error` exists.

**Sources:**
`chats/session/session_projection.ts::projectTranscriptMessage`;
`chats/messages/message.tsx`, `message_attachments_bar.tsx`, `message_footer.tsx`.

---

## 2.3 Tool-result inspection is less complete

The new `toolOutputFromResult()` creates a single text output using:

```ts
result.text || result.error || structuredText;
```

That changes several existing inspection flows.

### Text and structured results are no longer both retained

If a result contains both textual content and structured data, the text wins. The structured data is not retained elsewhere on the projected ordinary `UIToolOutput`.

The old MCP result construction included both text content and structured content in the inspectable result.

### Typed file/image result items are not reconstructed

Old tool outputs could retain multiple text, file, and image items. The details modal displayed each item and its metadata.

The new projection always builds one text item. A structured-only payload may still appear as JSON text, but it is not the old typed file/image result representation.

This also affects **Apply Diff** path hints when a local path existed only in a file Tool result: the unchanged candidate-path collector looks for `ToolOutputKind.File`, which this projection no longer produces.

### Original result arguments and selection metadata are missing

The new result projection does not attach:

- Original call arguments.
- The ordinary Tool selection/choice metadata.
- Web-search call items.

The call’s own details may still contain arguments, but the result’s details no longer carry the same information.

### Web-search activity can lose detail through replacement

When operation Tool calls exist, the projection replaces protocol-derived calls with `uiToolCallFromSession()`, which does not preserve `webSearchToolCallItems`.

For those messages, the web-search chip can fall back from the actual query/find pattern to a generic tool name.

Likewise, if `message.toolResults` exists, those results replace rather than combine with protocol-derived web-search outputs.

This is conditional on which projection branch is used; the protocol-output-only fallback still preserves its web-search fields.

**Sources:**
`chats/session/session_projection.ts::toolOutputFromResult`, `uiToolCallFromSession`, `projectTranscriptMessage`;
`chats/composer/tools/tool_details_modal.tsx`;
`chats/conversation/conversation_area.tsx::getMessageCandidatePathSources`.

---

# 3. Inference, tool follow-up, and completion

## 3.1 Live text and thinking are hidden by the retained stream renderer

This is the clearest live-response display problem.

The new flow does receive streamed content through snapshots:

- `projectActiveOperation()` puts `operation.streamedText` into `message.uiContent`.
- It puts `operation.streamedThinking` into `uiReasoningContents`.

But the rendered busy assistant row still receives `activeStreamSource` from `useStreamingRuntime()`.

That stream buffer is no longer populated by the new send/snapshot flow.

The unchanged content renderer then selects:

```ts
isBusy && streamingText !== undefined ? streamingText : content;
```

Since `streamingText` is the defined empty string, it overrides the populated snapshot content.

Thinking has the same problem: while busy, `MessageThinkingSection` uses the empty streamed-thinking value rather than the snapshot’s reasoning.

### User-visible result

While the message is the busy last assistant row:

- Live answer text does not appear.
- Live thinking does not appear.
- The status stays **Waiting for response**, rather than switching to Thinking or Streaming.
- A blank body can appear even though the snapshot already contains text.
- Content becomes visible once the row is no longer treated as busy.

The initial waiting presentation also differs: the old UI created an immediate assistant placeholder. The new transient projection creates no row until text or thinking exists. For a queued/preparing snapshot without an assistant transcript entry, there is no message-level waiting footer.

**Sources:**
`chats/session/session_projection.ts::projectActiveOperation`;
`chats/conversation/conversation_area.tsx`;
`chats/conversation/use_streaming_runtime.ts`;
`chats/messages/message_content_card.tsx`;
`chats/messages/message_thinking_section.tsx`.

---

## 3.2 A Run waiting for user/tool action is treated as ongoing generation

`sessionRunIsActive()` includes Waiting and Paused states, and that result becomes the tab’s `isBusy`.

Previously, `isBusy` ended when the model response finished. Pending manual tools could then be handled with an editable composer.

Now, during a waiting Run:

- The text editor remains read-only.
- Bottom-bar attachments, Tools, Skills, MCP, Templates, and Workspace controls remain locked.
- The primary button remains Stop rather than the old manual follow-up controls.
- Earlier user-message Edit buttons can remain visible, but their handler returns because the tab is busy.

For the latest assistant row, generation gating also continues to suppress:

- Copy.
- Disable Markdown.
- Message-details access.
- Citations.
- MCP App rendering.

The streaming defect above makes its body disappear as well.

There is an additional App-specific effect: the message list defers rich rendering for the last message whenever the tab is busy, even if that last message is a Tool result. Since App rendering is disabled while rich rendering is deferred, an App on the latest result cannot mount during that waiting state.

**Sources:**
`chats/session/session_projection.ts::sessionRunIsActive`;
`chats/tabs/use_chats_controller.ts::applySessionSnapshot`;
`chats/composer/composer_box.tsx`;
`chats/messages/message.tsx`;
`chats/conversation/conversation_area.tsx::ConversationMessageList`.

---

## 3.3 The old manual Tool workflow is not preserved

Tool selection and backend Tool execution are not generally absent. The new UI has individual message-level Run, Retry, and Skip command handlers.

The differences are in the user’s control over that execution and the next model request.

| Old user flow                                                                   | New behavior                                                                                |
| ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| Run one Tool and inspect its result before sending                              | Message-level Run calls `executeCalls` with `continue: true`                                |
| Run all pending Tools only                                                      | Composer Play button is unreachable                                                         |
| Run all pending Tools and send                                                  | Composer Fast-forward button is unreachable                                                 |
| Press Enter with only pending Tool calls                                        | There is no composer pending-call queue for this flow                                       |
| Review outputs, add text/attachments, then send                                 | Waiting Run keeps composer locked; no continuation-draft submission path                    |
| Change context/parameters for a Tool follow-up                                  | Some top model controls remain editable, but Run/Continue commands do not carry those edits |
| Stop automatic execution for the next batch without aborting the model response | The separate stop-auto-exec control is gone                                                 |
| Retry a failed call                                                             | Present for calls projected as `Failed`                                                     |
| Submit a failed call as an error result                                         | The button is labelled “Submit error” but sends Skip                                        |
| Run a previously skipped/error result again                                     | No equivalent result-level action in the replacement flow                                   |

### Why Play and Fast-forward are not merely relocated

The new `useComposerTools()` always returns:

- An empty call list.
- `hasPendingToolCalls: false`.
- `hasRunningToolCalls: false`.

Its local load/run/retry handlers are no-ops.

Consequently, the old composer buttons are still present in JSX but cannot become visible through this state.

The replacement message-level Run action always requests continuation. There is no supplied UI action that issues the old **run-only versus run-and-send** choice.

### Stop-next-auto-batch is a separate lost action

The old `FiZapOff` action armed a stop for the next automatic Tool-call batch while allowing the current response to finish.

The new Stop action cancels the Run. It is not the same user operation.

`setAutomationHold` exists on the façade, but no equivalent composer action calls it.

### “Submit error” currently submits a different command

The failed-call button uses the same `onSkipToolCall` path as a pending-call Skip button.

That path invokes `skipCall()`. It does not invoke `completeCallAsError()` or provide the failure text.

The old flow distinguished:

- Pending call → intentionally skipped output.
- Failed call → error output retaining the failure information.

### Error-result retries are not covered by call-state retries

A call marked `Resolved` is projected as `Succeeded` regardless of whether its selected result is an Error or Skipped result.

For that state:

- The output can be marked as an error.
- The call does not get Retry.
- The result chip has no replacement retry/run-again handler.

The old composer exposed Retry/Run on eligible error and skipped outputs.

**Sources:**
`chats/composer/toolruntime/use_composer_tools.ts`;
old and new `chats/composer/editor/editor_area.tsx`;
`chats/conversation/conversation_area.tsx::executeSessionToolCall` and `skipSessionToolCall`;
`chats/messages/message_attachments_bar.tsx::MessageToolCallChip`;
`chats/session/session_projection.ts::toolCallStatus`.

---

## 3.4 The replacement message actions do not target the call’s owning Run

Every message row passes only `call.id` to its Run/Retry/Skip handler.

`runCommand()` then uses the tab’s selected/current Run—not the Run that owns that historical call.

For an actionable call belonging to an earlier Run, the command combines:

- The earlier call ID.
- The current Run ID.

The UI does not preserve the owning Run in the action target or restrict those buttons to calls from the selected Run.

This is a misbehavior in the newly added history actions; the old history chips were read-only.

**Source:**
`chats/conversation/conversation_area.tsx::runCommand` and `itemContent`;
`chats/session/session_projection.ts::uiToolCallFromSession`.

---

## 3.5 Active Skill indicators no longer follow lifecycle results

The old Tool flow refreshed active Skill references after a Skill lifecycle operation and updated the composer’s Active state.

The new version retains active state only as local composer selection state. Session snapshots update the tab and messages, but do not feed:

- `run.skillSession.active`.
- `run.skillSession.enabled`.
- Operation effective/active Skill state.

back into the composer.

After a runtime Skill load/unload:

- Active counts can remain at their pre-run values.
- The Skill row can show the wrong Activate/Deactivate action.
- Workspace Active toggles can remain stale.
- A later submission is built from that unchanged local selection state.

This is a missing state-to-UI update, not a requirement that Skill lifecycle execution return to the frontend.

**Sources:**
Old `chats/composer/toolruntime/use_tool_runtime.ts::refreshActiveSkillRefs`;
new `chats/composer/skills/use_composer_skills.ts`;
`chats/tabs/use_chats_controller.ts::applySessionSnapshot`.

---

## 3.6 Completion and failure information is not presented consistently

The new UI does handle some errors visibly:

- Rejected Send/Edit submission → composer error, with the draft retained.
- Wait-resolution command failure → error in the wait modal.
- App opening/resource failure → App warning.
- Transcript error with no body text → message error fallback.

Other error paths have no equivalent presentation.

### Tab-level runtime errors are stored but never rendered

`runtimeError` is assigned for submission/snapshot/loading/deletion failures, but it is not passed into a visible alert, message, tab indicator, or retry surface.

### Tool-action command failures are console-only

Run, Retry, Skip, Continue, and cancellation command failures are caught with `console.error`.

A rejected action can therefore leave the visible state unchanged without telling the user why.

### Run-level terminal outcomes are not displayed

The projection does not render `run.outcome` or operation-level errors.

It also maps statuses other than completed/failed to `Status.None`, including cancellation statuses.

The old frontend explicitly displayed stopped/error terminal lines. In the new view:

- A failure recorded only at Run/operation level has no corresponding visible message.
- An error accompanying nonempty partial text is available through Details, but does not itself produce a visible terminal error line.
- Cancellation does not itself produce a stopped notice.

This does **not** establish that the backend loses partial output. It establishes that the supplied frontend no longer provides the same terminal-state presentation.

**Sources:**
`chats/session/session_projection.ts`;
`chats/tabs/tabs_model.ts`;
`chats/tabs/use_chats_controller.ts`;
`chats/messages/message.tsx`;
old `chats/conversation/completion_finalizer.ts`.

---

## 3.7 Approval handling remains, but the interaction is different

The old MCP approval dialog showed separately labelled:

- Server.
- Tool.
- Risk.
- Reason.
- Arguments.

The new wait modal renders a generic wait-kind title and a JSON/text summary.

The information may still exist inside that summary, but the old structured presentation is gone.

The dismissal behavior also changed:

- Old Close/Escape resolved the approval as Deny once.
- New modal has cancellation blocked, a disabled close control, and a no-op close handler.

For manual execution and other wait kinds, this also introduces a blocking modal where the old manual Tool flow used the composer.

### New wait states requiring input have no input UI

`SessionWaitingRequest` supports `inputSchema`, and `resolveWait` supports `input`.

The new modal only renders resolution buttons. It neither renders the schema nor collects or submits input.

That is a **new-flow completeness gap**, rather than a claim that the old application had a generic wait-schema form.

**Sources:**
Old `chats/composer/mcp/mcp_approval_modal.tsx`;
new `chats/session/session_wait_modal.tsx`;
`chats/conversation/conversation_area.tsx::resolveWait`.

---

# 4. Reopening conversations and restoring state

Several composer-specific losses occur only after reopening, so a fresh-conversation check would miss them.

## 4.1 Restored background tabs can show their messages with a default composer

The new controller observes every persisted Session tab, including tabs whose composer has not mounted.

An observed snapshot sets:

```ts
isLoaded: true;
sessionSnapshot: snapshot;
```

but does not restore the composer.

Later, selecting that tab calls `ensureTabLoaded()`, which returns immediately when `isLoaded` and `sessionSnapshot` are already present.

Therefore, for a restored background tab whose snapshot arrived before first activation:

- Messages and title load.
- The composer can initialize with its default model/Agent/context.
- Saved Tools, Skills, MCP, Workspace, and instruction selections are never applied.

The old selected-tab load path explicitly synchronized the composer after conversation hydration.

**Source:**
`chats/tabs/use_chats_controller.ts::ensureSessionObserver`, `applySessionSnapshot`, `ensureTabLoaded`.

---

## 4.2 Model restoration is dropped if the model catalog finishes later

Old `restoreConversationContext()` queued its pending restoration until model options loaded.

New `restoreSessionConfiguration()` clears the pending restoration and returns `null` when model options are not loaded.

There is no pending Session-configuration restore for the catalog-load callback to apply.

When Session loading finishes before the model catalog:

- Saved model selection is not applied.
- Saved request preferences are not applied.
- Saved history selection is not applied.
- The later catalog result installs defaults instead.

**Source:**
`chats/composer/contextarea/use_context_state.ts::restoreSessionConfiguration` and the model-catalog effect.

---

## 4.3 A saved explicit model override is lost when an Agent is also selected

A concrete round trip:

1. Select Agent A.
2. Explicitly choose model B.
3. Send. The configuration includes Agent A and model B.
4. Reopen the Session.
5. The composer displays B.
6. Send again.

Both context restoration and Agent restoration set the model selection source to `'agent'` merely because an Agent exists.

The next submission therefore omits model B and delegates back to the Agent, while the dropdown still shows B.

The old restored model selection was not replaced by this Agent-ownership rule.

**Sources:**
`chats/composer/contextarea/use_context_state.ts::restoreSessionConfiguration`;
`chats/composer/agents/use_agent_manager.ts::restoreSelectedAgent`;
`chats/session/session_intent.ts::buildSessionSubmissionIntent`.

---

## 4.4 Restored model-default instructions cannot reliably be re-enabled

Another concrete round trip:

1. Turn off Model default instructions.
2. Send and reopen.
3. Turn the checkbox back on.
4. Send again.

Restoration places `SystemPrompt` in `selectedModel.requestPatch.clear`.

The checkbox changes only the system-prompt controller’s include preference. Submission copies the existing clear set and adds `SystemPrompt` when inclusion is false, but does not remove it when inclusion becomes true.

Thus, the checkbox can be checked while the request still explicitly clears the model-default prompt.

**Sources:**
`chats/composer/skills/use_agent_system_prompt.ts::restoreSessionConfiguration`;
`chats/composer/contextarea/use_context_state.ts::requestPatchFromSessionOverrides`;
`chats/session/session_intent.ts::modelOverridesFromComposer`.

### Restored hybrid-off state also has contradictory UI

On a hybrid model, restoring a reasoning clear still initializes `isHybridReasoningEnabled` from the base model’s reasoning type.

The result can be:

- Disabled selected in the Hybrid menu, from the saved clear.
- “On” on the trigger.
- Effort-token controls shown instead of temperature.

That is a presentation inconsistency in the new restoration path, separate from whether the backend correctly honors the clear.

---

## 4.5 Workspace context and Skill deselections reappear after reopening

The submission builder writes Workspace removal keys as:

```text
artifact:<rootID>:<artifactID>
```

The Workspace restoration helper stores its baseline map under:

```text
<rootID>:<artifactID>
```

It then tries to delete the removal key directly from that map.

The keys do not match.

### User-visible round trip

1. Attach a Workspace.
2. Uncheck a context item or instruction Skill.
3. Send.
4. Reopen.

The deselected baseline item becomes checked again.

With unchanged Workspace contents, the next send is then built from that reselected UI state, rather than preserving the removal.

**Sources:**
`chats/session/session_intent.ts::artifactOverridesForWorkspace`;
`chats/composer/workspaces/use_composer_workspace.ts::applyArtifactOverrides`.

### Exact previous selection information is also reduced

The new Workspace intent records differences from the current baseline rather than the old selected-reference snapshot.

On reopening, it rebuilds the baseline from current contents. Consequently:

- Newly discovered baseline items can appear selected without the old explicit “Use Current Contents” step.
- Previously selected per-item revisions/digests are not retained for unchanged baseline items.
- The old changed-item comparison loses that previous selection evidence.

### The newly added Workspace MCP checkbox has a related round-trip gap

This is a new control, not a removed old capability.

Its submission contains `Add` entries for checked servers but no removal entries for unchecked baseline servers. Restoration starts from all baseline servers and adds the checked subset.

An unchecked baseline server therefore reappears selected on restore.

---

## 4.6 Restored MCP selections use inconsistent server identities

`hydrateSessionMCPOverrides()` resolves the current runtime server ID and uses it for Tools, resources, templates, and prompts.

But the top-level server entry uses:

```ts
server: selection.server.artifactID;
```

The composer indexes its server selections by runtime server ID.

Where the Artifact ID and runtime ID differ, reopening can produce:

- A selected count without a matching checked server row.
- Separate selection entries for one server’s resources and its top-level selection.
- Discovery not loading for the top-level entry.
- Send failing the connected-server check despite the real server being connected.

**Sources:**
`chats/session/session_composer_restore.ts::hydrateSessionMCPOverrides`;
`chats/composer/mcp/mcp_composer_types.ts::mcpContextToSelectionMap`;
`chats/composer/mcp/use_composer_mcp.ts::prepareForSubmit`.

---

## 4.7 Restored MCP prompt/template values lose their editable fields

The restoration helper preserves saved argument **values**, but does not restore argument **definitions**.

`MCPArgumentFields` renders its inputs from the selected item’s `arguments`, not from the newly discovered catalog item.

Discovery does not merge those definitions back into the restored selected prompts/templates.

As a result, after reopening:

- Saved values may still exist internally.
- Their argument inputs are not rendered.
- The user cannot inspect/change those values through the original fields.
- Required-argument indicators no longer reflect the missing definitions.

This is independent of the server-identity problem above.

**Sources:**
`chats/session/session_composer_restore.ts::hydrateSessionMCPOverrides`;
`chats/composer/mcp/mcp_bottom_bar_chip.tsx::MCPArgumentFields`;
`chats/composer/mcp/use_composer_mcp.ts`.

### One unavailable MCP selection can prevent unrelated restoration

Editor restoration clears transient state, then awaits Tool hydration and MCP hydration together.

If MCP hydration fails—for example, a saved server is no longer available—the function never applies the successfully hydrated Tools or reaches Workspace/Skill restoration.

The failure therefore affects more than the unavailable MCP selection, and the controller’s stored error is not rendered.

**Source:**
`chats/composer/editor/editor_area.tsx::restoreSessionConfiguration`.

---

## 4.8 Hydration and generation indicators can disagree with the loaded state

### Opening an active Session through search finishes by marking it idle

`loadConversationIntoTab()` initially derives `isBusy` from the loaded Run.

After composer restoration, its final update unconditionally sets `isBusy: false`.

Until a later snapshot corrects it:

- Busy indication disappears.
- Stop can disappear.
- The composer becomes editable.
- Sending encounters “This Session already has an active Run.”

### Snapshot loading can unlock input before composer restoration completes

`applySessionSnapshot()` sets `isHydrating: false` independently of the asynchronous composer restoration.

That exposes the composer while saved selections are still being restored. An earlier observer snapshot can also unlock it before the explicit load path reaches its editor reset.

### A failed initial load can remain visually stuck loading

The error path clears `isHydrating` but can leave `isLoaded: false`.

The conversation area computes hydration as:

```ts
isHydrating || !isLoaded;
```

so the body remains in its loading state and the composer stays locked. There is no rendered load error or Retry action for that state.

**Source:**
`chats/tabs/use_chats_controller.ts::applySessionSnapshot`, `ensureTabLoaded`, `loadConversationIntoTab`;
`chats/conversation/conversation_area.tsx`.

---

# 5. MCP Apps

## 5.1 App-initiated Tool calls no longer return their result to the original request

Old `tools/call` returned the Tool result in its JSON-RPC response:

- `content`.
- `structuredContent`.
- `isError`.

The new router returns:

```ts
{
  accepted: true;
}
```

after command acceptance.

A later observation sends a `ui/notifications/tool-result` notification, but that does not preserve the original request/response contract for an unchanged App awaiting `tools/call`.

App actions that read the result of that awaited call no longer receive it there.

The later notification also reduces content to one text item rather than preserving the old full MCP content list.

**Sources:**
Old and new `chats/mcpapps/mcp_app_rpc_router.ts::handleToolCall`;
new `chats/mcpapps/mcp_app_view.tsx` Run-observation effect.

---

## 5.2 App context no longer enters the existing visible composer queue

### Old flow

1. App proposes model context.
2. User approves where required.
3. Context is added to the composer.
4. MCP chip shows the queued update count.
5. User can clear it.
6. Normal Send includes it.

### New flow

The App stages context directly into the Run. It no longer dispatches the composer-context event.

The old composer queue/count/Clear UI remains, but is not connected to that staging path.

Consequently:

- An approved App update does not appear in the old queued-count surface.
- Clearing the composer’s App-context list does not clear the staged Run context.
- The user no longer has the same review/clear-before-send interaction.
- The replacement Continue action sends Run identifiers, not the current composer draft or configuration.

The new **Continue with staged App context** button appears only when the selected Run is Waiting, has an unconsumed continuation revision, and has no pending wait request. That is not the old ordinary next-message flow.

**Sources:**
`chats/mcpapps/mcp_app_rpc_router.ts::handleUpdateModelContext`;
`chats/mcpapps/mcp_app_view.tsx::stageModelContext`;
`chats/mcpapps/mcp_app_events.ts`;
`chats/conversation/conversation_area.tsx::hasStagedContinuation`.

---

## 5.3 App context loses data on submission and restoration

### Non-text content is reduced

Staging maps content down to:

- Type.
- Text.
- URI.
- Name.
- Description.
- MIME type.

Image/audio data and embedded resource blobs are not forwarded. The old context path carried the full `MCPContent`.

An App proposal can therefore be shown in the approval preview but have its binary content omitted from the staged command.

### Restored App context fails the new submission filter

`appContextUpdatesFromPayload()` requires a `viewID`.

But:

- `MCPAppModelContextUpdate` does not declare it.
- `appContextUpdatesFromSession()` does not preserve it.

Restored App updates therefore fail that filter if loaded back into a composer submission.

**Sources:**
`chats/mcpapps/mcp_app_view.tsx`;
`chats/session/session_intent.ts::appContextUpdatesFromPayload`;
`chats/session/session_projection.ts::appContextUpdatesFromSession`.

---

## 5.4 App confirmation preferences are no longer reflected in the UI

The old App host honored:

- `requireApprovalForOpenLink`.
- `requireApprovalForContextUpdates`.

When those were false, it did not show the corresponding local confirmation dialog.

The new host always asks locally.

The new host also always advertises Tool-call support to the App, whereas the old host reflected `allowAppInitiatedToolCalls`.

The backend can still enforce its policy correctly; the user-visible confirmation behavior and App capability indication have nonetheless changed.

The App header also changes from the resolved Artifact identity—with a full identity tooltip—to the runtime server ID without that tooltip.

**Source:**
Old and new `chats/mcpapps/mcp_app_view.tsx`.

---

## 5.5 App input association has an ID mismatch

New projected calls use the provider call ID as `UIToolCall.callID` when it exists.

New projected outputs use the runtime call ID as `UIToolOutput.callID`.

`ChatMessage` joins App outputs to calls by `callID`.

When those IDs differ, the join does not find the originating call, and `toolInput={call?.arguments}` is undefined.

This is another completeness problem in the new App presentation path.

**Sources:**
`chats/session/session_projection.ts::uiToolCallFromSession` and `toolOutputFromResult`;
`chats/messages/message.tsx` App-view construction.

---

# 6. Search and tab management

## 6.1 Ordinary search and tab controls are largely retained

The following remain represented:

- Recent conversations.
- Short-query local title filtering.
- Enter to search messages.
- Debounced full search.
- Pagination.
- Arrow-key selection and Escape.
- Search error Retry.
- Delete confirmation.
- Hiding Delete for open conversations.
- Selecting an already-open conversation.
- Scratch-tab reuse.
- The 16-tab limit and LRU policy.
- Tab cycling and focus shortcuts.
- Rename UI.
- Per-tab scrolling and message-jump controls.
- JSON export through the façade.

The new search refresh path also explicitly refreshes an active full-text query rather than always switching through recent-conversation loading.

There is no basis here for calling the search feature generally missing.

---

## 6.2 The old saved open-tab layout is discarded

Both versions use the same local-storage key:

```text
app.chats.tabs.v1
```

Old data has `v: 1`. The new reader accepts only `v: 2`.

Opening the new build with the old saved state therefore starts from a new scratch tab rather than restoring:

- Open tabs.
- Selected tab.
- Scroll positions.
- Tab activation history.
- Associated saved tab flags.

This is specifically a loss of the **saved open-tab workspace**. It is not a claim that the underlying conversations are deleted or cannot be found through a properly migrated backend.

**Source:**
Old and new `chats/tabs/tabs_persistence.ts`.

---

## 6.3 Automatic title refinement is no longer part of the frontend flow

Old behavior considered the first two user messages, while the conversation was short, and selected the better-scoring generated title. Early edits could also pass through that title logic.

The new frontend calculates a title only when creating the Session. Subsequent title changes shown in the UI flow are explicit user renames.

The old second-message/early-edit refinement rule is no longer represented in these calls.

This is especially noticeable for conversations beginning with a short greeting or an attachment-only turn before the meaningful request.

**Sources:**
Old `chats/tabs/use_chats_controller.ts::saveUpdatedConversation`;
new `chats/conversation/use_send_message.ts` Session-creation branch.

---

## 6.4 Closing a generating tab no longer stops its model request

Old tab disposal aborted the active completion controller.

The new send path does not install that controller. Closing the tab disposes local state and observations, but does not call `cancelRun()`.

Therefore:

- Explicit Stop requests cancellation.
- Closing the tab does not.
- Busy-tab eviction follows the same changed behavior.
- Leaving the Chats page no longer performs the old completion-abort action.

A Run can consequently continue after its tab/view is closed.

That may be compatible with backend-owned execution, but it is a user-visible behavioral change from the supplied old implementation.

**Sources:**
`chats/tabs/use_chats_controller.ts::closeTab` and disposal;
old/new `chats/conversation/use_streaming_runtime.ts`;
new `chats/conversation/use_send_message.ts`.

---

# 7. Things I have not counted as new losses

A few apparent omissions already existed in the old attached version:

- **Raw model-parameter JSON editor:** the tooltip mentions it, but neither attached Advanced Parameters modal renders that field.
- **Search-result message snippets:** both implementations assign `snippet: ''`.
- **Restored folder groups:** the old attachment loader already flattened restored attachments rather than reconstructing directory groups.
- **Unsent draft persistence across restart:** neither tab-persistence format saves draft text or attachment state.
- **A Tool-output remove button:** the old output-chip comment mentions one, but that chip does not actually render it.
- **Resource-package copying through composer Fork:** the old quick Fork was restricted to simple skills without resources.
- **Client-side history slicing, execution, and persistence helpers:** their removal alone is not a UX finding when those responsibilities move to the backend.
- **Export format internals:** the export action remains connected; the string-returning Session export contract alone does not establish a user-facing export loss.

## Overall

The ordinary authoring controls, attachment selection, search, and basic tab navigation remain substantially present.

The main end-to-end losses are in:

1. **Applying and modifying Agent-provided context.**
2. **Editing and resending existing messages.**
3. **Displaying live responses.**
4. **Manual Tool execution and follow-up control.**
5. **Showing per-turn context, usage, and result details.**
6. **Restoring the composer from saved state.**
7. **Preserving MCP App interactions.**

Those differences arise in what the frontend displays, forwards, or restores, even with a correctly functioning Session backend.

---

# Post-patch UX gap analysis

This section excludes migration-only concerns such as old local tab-storage schema compatibility.

## Resolved or substantially covered by the patch set

| Flow                                         | State after patch                                                                                                                                                      |
| -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Edit a prior user message                    | The composer now loads the message text, attachments, Tool choices, MCP context, App context, Skills, Workspace, and displayed Tool outputs before entering edit mode. |
| Live assistant text                          | Session snapshot text no longer disappears behind an empty legacy stream buffer.                                                                                       |
| Live thinking                                | Session snapshot thinking remains visible while the Run is active.                                                                                                     |
| Initial response waiting state               | An active model operation can project an assistant placeholder even before text arrives.                                                                               |
| Terminal operation errors                    | Operation-level errors and runtime metadata can now reach the existing message details/error UI.                                                                       |
| Manual Tool execution                        | Individual Tool Run now means run-only rather than forced continuation.                                                                                                |
| Batch Tool controls                          | Waiting Runs can expose Run pending tools and Run tools and continue actions.                                                                                          |
| Continue after Tool/App state                | A generic Continue run action is available when the Runtime indicates a continuation is ready.                                                                         |
| Failed Tool action                           | “Submit error” now uses `completeCallAsError` instead of skipping the call.                                                                                            |
| Tool command targeting                       | Message Tool actions now locate the Tool call’s owning Run instead of blindly using the currently selected Run.                                                        |
| Tool command errors                          | Session command failures are shown in the chat area instead of only reaching `console.error`.                                                                          |
| Automatic Tool hold                          | The UI now uses Session Runtime automation holds for a one-batch Tool/continuation hold.                                                                               |
| Wait states with structured input            | Pending waits can accept optional JSON response input and send it through `resolveWait`.                                                                               |
| Staged MCP App context                       | The UI shows staged App-context count, can discard it through `updateContinuationInput`, and exposes continuation.                                                     |
| MCP App `tools/call` result                  | App calls can receive the eventual Session Tool result as their JSON-RPC response, while retaining result notifications.                                               |
| App-context restoration identity             | `viewID` is retained through transcript projection and Session submission filtering.                                                                                   |
| Background Session-tab composer restoration  | A tab whose Session snapshot arrived before activation now still restores its composer configuration when opened.                                                      |
| Model-catalog timing                         | Session model configuration is retained pending catalog completion rather than being dropped when the catalog loads later.                                             |
| Agent plus explicit model                    | A saved explicit model remains Session-owned after restoration instead of being silently switched back to Agent-owned selection.                                       |
| Model default instructions                   | Re-enabling model defaults removes the stale `SystemPrompt` clear override.                                                                                            |
| Restored hybrid reasoning UI                 | A saved reasoning clear no longer leaves the Hybrid UI displaying “On” while the saved state is disabled.                                                              |
| Workspace Context/Skill deselection          | Workspace override key normalization now correctly applies saved removals.                                                                                             |
| Workspace MCP deselection                    | Workspace MCP intent now emits removal overrides for deselected baseline servers.                                                                                      |
| Restored MCP server identity                 | The top-level restored MCP selection now uses Runtime server IDs consistently.                                                                                         |
| Restored MCP prompt/template argument fields | Discovery refresh merges definitions back into restored selections while preserving saved values.                                                                      |
| Partial composer restore                     | A failed MCP restore no longer prevents Tool, Workspace, or Skill restoration from proceeding independently.                                                           |
| Skill lifecycle state                        | Session `skillSession.enabled` and `skillSession.active` values are synchronized back into mounted and pending composers.                                              |
| Active Session loading                       | Reopening an active Session no longer unconditionally marks it idle after composer restoration.                                                                        |
| Failed session load visibility               | Session load errors no longer leave a permanently hidden loading state; they become visible in the conversation area.                                                  |
| Closing a generating tab                     | Disposing a tab now best-effort cancels its active Runtime Run, matching the old visible close/abort behavior more closely.                                            |

---

## Still pending because the attached frontend/session contracts do not expose enough information for full parity

### 1. Agent recipe application remains the largest unresolved composer gap

The current Agent façade validates and tracks an Agent selection, but it does not expose a complete effective composer recipe that can safely be applied as editable UI state.

The remaining missing visible behavior includes:

- Applying Agent model/tool/Skill/MCP/instruction/text recipe into composer controls.
- Showing Agent-provided instruction sources individually.
- Reapply restoring visible Agent selections.
- Agent request starter text filling an otherwise empty draft.
- Reset-to-default clearing the previously applied Agent recipe.
- Correctly expressing user removal of an inherited Agent contribution as an explicit Session override rather than merely removing a locally displayed item.

The Session configuration model has `Add`, `Remove`, and `Replace` override modes, so this is representable. The missing piece is a complete Agent-effective baseline supplied through the management façade and carried as Composer baseline state.

### 2. Editing Tool-output follow-up messages still cannot fully reproduce the old wire behavior

The editor can now load a historical message’s displayed Tool outputs.

However, `SessionUserInput` has no Tool-output field. The old frontend could serialize Tool outputs into inference input unions. The current Session command shape does not expose an equivalent client-supplied Tool-output payload.

As a result:

- Editing text/context around a Tool follow-up is restored visually.
- Editing, removing, retrying, or resubmitting those Tool outputs cannot be guaranteed to alter the next Runtime operation through the attached Session API.

### 3. Historical message provenance remains incomplete

The Session projection does not currently carry all old per-message fields:

- Model/request details used for each specific historical turn.
- Token usage.
- Complete inference debug request/response/provider details.
- Hydrated Tool choices and unavailable Tool-selection diagnostics.
- Full MCP server/resource/prompt selection as used for that historical message.
- Full Workspace selection and send-time Workspace usage outcomes.
- Per-turn enabled/active Skill lists in historical message bars.

The UI components remain available. The Session transcript/projection lacks enough per-operation render data to fill all of them without extending the Session snapshot contract or adding a management façade projection endpoint.

### 4. Tool result richness is limited by `SessionToolResult`

The Session projection currently reduces Tool results to text/structured JSON.

The old UI could inspect richer Tool output items, including files and images. The current Session result DTO does not expose equivalent typed output item lists, so these remain unavailable in history:

- File/image Tool-result cards.
- Some file-derived diff-target hints.
- Exact original Tool-output item metadata.
- Some web-search query/output details when Session Tool calls replace protocol-output-derived fields.

### 5. MCP App context content still has a transport-shape limitation

The App-context staging command currently maps App content into text/URI/name/description/MIME fields.

That leaves no Session-side representation for all possible original `MCPContent` forms, such as arbitrary binary/image/audio/resource-blob payloads.

The patch restores view identity and visible staged-context handling, but cannot preserve those omitted content forms without a broader Session App-context DTO.

### 6. MCP App policy-driven confirmation behavior is still not fully represented

The old App host could reflect policy-level choices such as:

- Whether opening a link required local confirmation.
- Whether context updates required local confirmation.
- Whether App-initiated Tool calls were advertised as available.

The Session-scoped App view has no attached management-facade method that returns the App policy for its specific Session App instance. The backend can enforce policy, but the UI cannot yet reproduce all prior policy-dependent confirmation and capability-advertising behavior.

### 7. Automatic title refinement after later user turns remains absent

The old frontend could refine a generated title based on early subsequent user messages.

The new implementation creates the Session title on Session creation and exposes explicit rename. It does not re-run the old first/second-user-turn title scoring behavior.

### 8. Session-wide Runtime outcomes are still less explicit than old frontend terminal messages

The patch surfaces transcript/operation errors where available. It does not synthesize old-style terminal markdown messages for every Run-level cancellation, interruption, or failed outcome.

The Session snapshot includes Run outcomes, but the projection currently does not create a dedicated visible terminal message from every such outcome.

---

## Net result after the patch

The main remaining UX completeness work is concentrated in:

1. **Agent recipe preview/application and inherited override handling.**
2. **Richer historical Session projection data.**
3. **Tool-output follow-up editing through a Session-native input contract.**
4. **Full MCP App binary context and policy metadata.**
5. **Optional title refinement and richer Run-outcome presentation.**

The normal composer, model controls, attachments, Workspace selection, Skill state synchronization, Session hydration, Tool action controls, live output rendering, chat search, tabs, and Session-scoped MCP App Tool result flow are materially closer to the prior UX after these changes.
