# Assistant UX Requirements

## Terms

- Assistant:

  - Chat persona/preset that sets defaults (model preset, system prompt) and defines allowed catalogs (which model presets, prompt templates, tools, and knowledge/doc sets can be used in this chat). If a category is marked “All allowed,” there’s no restriction for that category.

- Prompt Template:

  - Reusable scaffold of role-tagged messages (System/Developer/User/Assistant), with/without variables, and/or preprocessors. Applied per turn.

- Model Preset:

  - Named configuration for inference: model ID plus parameters (e.g., temperature, reasoning).

- Tool:

  - LLM-callable capability (function/tool call) available to the model during inference.

- Knowledge/Document Set:
  - Configured retrieval source or index (e.g., RAG corpus) that can be included in a chat/turn.

## Core principles

- Always chat with an Assistant (Default Assistant provides full freedom).
- Defaults come from the Assistant. Any user change persists for the chat until changed again.
- Clear scoping: Assistant defaults → chat-level selections → per-turn additions.
- Deterministic composition order; minimal surprises.
- Inline, keyboard-first template UX.
- Happy path: type → (optional /template) → send.
- Backend prompts/tools/docs are out of scope here; this is UI/flow.

## Assistant Context Bar (above the composer)

- This is a Chat-level context. It is persistent until user changes something.

- Assistant dropdown: avatar/name; switch Assistant. Selections persist for the chat.

  - On switch, reset to the new Assistant’s defaults for model/system/tools/knowledge.
  - Attempt to preserve the current chat selections if they’re allowed by the new Assistant; otherwise: fallback to the new Assistant’s default.
    - System prompts and recent prompts should not be preserved (Assistant-scoped); options revert to the new Assistant defaults.
    - Available tools, docs are also reset to Assistant defaults.
  - Default Assistant allows all models/tools/docs/templates.

- Model params

  - Model dropdown: shows available models with a check on the Assistant default; user selection replaces the default and persists.
  - Temperature: dropdown presets (e.g., 0.0, 0.2, 0.5, 0.7, 1.0) + free entry (0–1). Hidden if model doesn’t support it.
  - Reasoning: dropdown with levels (Low/Med/High) or token-based entry (e.g., 0–8192). Hidden if not supported.

- History

  - Toggle button (Include prior messages). Current state applies to all turns and persists until changed.

- Tools and Knowledge: availability and chat defaults only. Selection is per turn, see Composer.

  - Managed via dropdowns (select-only, no freeform). Selections persist for the chat and are used by default on each turn.
  - Tools dropdown: multi-select of allowed tools (select-only). Selections persist for the chat (defaults). Turn-level additions via “+” don’t change this.
  - Knowledge dropdown: multi-select of allowed sources (select-only). Selections persist for the chat (defaults). Turn-level additions via “#” don’t change this.
  - Managed in a advanced params modal.

- System prompt:

  - Dropdown shows: Assistant default (checked), prior selections (recents, max ~5), and any template-injected options after send.
  - When a template with a System is sent, it is appended here and becomes the active selection; removal before send does not add it.
  - During editing and before send, an indicator can show “new system” (e.g., tinted preview) if a template is previewing a different System.

- Sliders button:
  - Opens a advanced params modal for Additional context (tools, knowledge, extra params).

## Turn details and Inspector (per message)

- Purpose: show exact request/response for a specific exchange without cluttering the transcript.

- Entry points

  - Under the User (request) message: “View request details”.
  - Under the Assistant (response) message: “View response details”.
  - Both open the same Inspector component, deep-linked to the relevant tab/anchor.

- Inspector content

  - Request tab (from user message):
    - Composed messages[] (System, examples, user), tool/source config, model params, history state.
    - Template details (name, variables used, preprocessors status).
    - Attachments summary (processed or raw).
    - Actions: Copy JSON, Copy cURL (truncated display with full copy), Download request.
  - Response tab (from assistant message):
    - Model metadata, latency, token counts.
    - Tool call trace (names, args, results), streaming timeline if applicable.
    - Headers/response metadata (redact as needed).
    - Actions: Copy JSON, Download response.

- Behavior
  - Single Inspector component; opens as a modal.
  - Keyboard: Esc closes; focus returns to the triggering message/action.
  - Privacy: hide/redact sensitive variable values by default with a “show values” toggle (sticky per session).

## Composer

- All actions inside composer are per-message (turn-level) actions only. None of it persists for future turns.
- Text input; Enter/Cmd+Enter to send.
- Neutral tip below composer: “Shortcuts: / templates, + tools, # knowledge. Type at start or after a space.”
- Send is enabled only when aggregate text is non-empty and all prerequisites are satisfied. Aggregate text:

  - Composer text
  - Compiled User content from a selected template
  - Processed file text (if “Process to text” is on)

- Action: Attach Templates (/)

  - Per-message intent; variables filled inline; preprocessors run pre-send with retry; manual override allowed.
  - Multi Role/Message block behavior
    - Last User block behavior (primary mental model):
      - If the template’s last message is a User block, its compiled content becomes the current message text for this turn.
      - By default, insert the compiled User content into the composer at the cursor (users can edit before sending).
      - Earlier User/Assistant blocks (if any) become hidden examples for this turn and do not appear in the transcript (shown in Inspector).
    - No last User block:
      - The template contributes only non-User blocks (System/Developer/Assistant/examples). Composer text remains the user’s message.
      - If composer is empty and there’s no processed attachment text, Send stays disabled.
    - System from template is preview-only until send; on send it is added to the System dropdown and becomes the current chat selection.
  - Slash (/) opens inline combobox above the composer.
    - Rows: name, short description, role icons (S/D/U/A), “needs preprocessors” icon when applicable, and a “helper” badge for preprocessor-only templates.
    - Enter to select → enter Template mode.
  - Template mode (inline)
    - Template Bar: [icon + name] • status (spinner/ready) • [Edit] [Cancel] • “Send now.”
    - Edit opens compact variables panel:
      - Required (expanded), Optional (collapsed).
      - Supported types: string, number, boolean, enum, date/time.
      - Defaults prefilled; required marked; large values as truncated chips; preprocessors auto-run with retry; manual override allowed.
    - Composer text mapping hint (when applicable): “Using your message as: [var] • Change” (or a selector if multiple).
    - Cancel/Esc exits template mode, clears mapping, and reverts any previewed System.
  - Preprocessor-only templates (helpers)
    - Identified by shape: zero message blocks, one or more preprocessors.
    - Use the same Template Bar (no System preview, no “Send now,” no mapping). Actions: Edit inputs, Run.
    - On success, show a Result chip with actions: Insert at cursor (default for short text), Attach as file (for large/non-text).

- Action: Attach Tools (+)

  - Inline “+” opens select-only autocomplete for allowed tools to mark them “eligible/selected” for this turn.
  - Selection ≠ invocation; the model decides, user may confirm/deny calls if required.
  - Inline additions do not change chat defaults unless the user also updates the Tools dropdown.

- Action: Attach Knowledge (#)

  - Inline “#” opens select-only autocomplete for allowed sources (retrievers).
  - Inclusion applies to this turn only.
  - Allowed sources here are configured doc sets.
  - File attachments are separate (see Attach files).

- Action: Attach Files Button
  - “Attach” button to add files (txt/pdf/html/images).
  - Attachments are per-turn only.
  - Shows chips with size and truncated name.
  - Option to process files (extract text); show truncated preview; allow remove before send.

## Request pipeline

- Disable Send until required fields are resolved and any failing preprocessors are addressed; show concise reason inline.
- Client builds UI state: Assistant, model params, System selection, History state, selected tools/sources, attachments, template id/values/mapping.
- Call BuildCompletionRequest (server) → returns messages[], tool/source config, params.
- Composition order (server):
  1. System (current chat-level, including any newly selected via template on send)
  2. Developer/Assistant blocks (from Assistant/Template)
  3. Prior chat history (if included)
  4. Template examples (all role blocks except the last User when present), in declared order, hidden in transcript
  5. Current User content:
     - If last User block exists: use its compiled content
     - Else: use composer text (plus processed attachment text if applicable)
- Send request; stream response.
- After send
  - Clear template mode state (chips, preprocessors, mapping).
  - If the template had a System, it’s now added and selected in the System dropdown (chat-level).
  - Tools/sources added via inline triggers apply only to that turn unless separately selected in dropdown.

## Acceptance criteria (MVP)

- Every chat starts with an Assistant (Default Assistant if none chosen).
- Assistant Context Bar includes: Assistant, Model, Temperature, Reasoning, History toggle, System prompt dropdown, Tools dropdown, Knowledge dropdown, Sliders (modal).
- On Assistant switch, incompatible chat selections auto-fallback.
  - System recents are Assistant-scoped; switching Assistant clears recents and reverts System to new Assistant default.
- User selections (model, temp, reasoning, history, system, tools, knowledge) persist at chat level until the user changes them.
- Inspector redacts sensitive values by default and cURL omits secrets.
- “/” opens templates; Template Bar supports variables, preprocessors, and mapping; Cancel/Esc reverts preview.
- Last User block in a template becomes the current message text (inserted into composer by default). If absent, template contributes only context and composer text remains the message.
- Template System shows as “Preview” before send; added and selected only after send.
- “+” adds tools for this turn; “#” adds knowledge sources for this turn; both are select-only from allowed sets.
- “Attach” supports files; optional text extraction; attachments are per-turn.
- Send is enabled if compiled content exists even when composer is empty (template-only or processed attachments).
- Triggers (+/#//) do not activate on paste; typing only.
- Server builds the completion request; transcript stays clean; Inspector shows message details and template examples used.
