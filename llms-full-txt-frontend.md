# FlexiGPT frontend map

> FlexiGPT is a local-first AI workspace built as a Wails desktop app with a React frontend.
>
> This document is organized around functional layout, runtime flow, and architectural ownership so an LLM can understand the frontend quickly.
> It focuses on what the frontend actually does, how the major surfaces fit together, and which files own which behavior.
> Generated Wails binding files and low-value visual trivia are intentionally de-emphasized.

## Frontend architecture in one view

The frontend is built around a few stable ideas:

- React Router owns route composition.
- `root.tsx` owns the app shell, hydration bootstrap, theme bootstrap, Wails script injection, and the global attachment-drop bridge.
- `Sidebar` and `TitleBar` form the desktop shell.
- `PageFrame` gives most routes a common full-height container.
- The `chats` workspace is the main application surface and owns conversation tabs, restoration, streaming, and the composer.
- Catalog pages own editing of reusable data:
  - assistant presets
  - model presets
  - prompts
  - tools
  - skills
  - settings
- The docs route renders bundled markdown content and handles internal doc navigation.
- The typed backend boundary lives in `app/apis`; UI code should not talk to backend implementation details directly.
- Shared domain shapes live in `app/spec`, and shared UI/runtime helpers live in `app/lib`, `app/hooks`, and `app/components`.

## How to read this map

If you want the fastest mental model, read in this order:

1. `app/root.tsx`
2. `app/components/sidebar.tsx`
3. `app/components/title_bar.tsx`
4. `app/chats/page.tsx`
5. `app/chats/tabs/use_chats_controller.ts`
6. `app/chats/conversation/conversation_area.tsx`
7. `app/chats/conversation/use_send_message.ts`
8. `app/chats/composer/composer_box.tsx`
9. `app/chats/composer/editor/editor_area.tsx`
10. `app/chats/composer/systemprompts/use_composer_system_prompt.ts`
11. `app/chats/composer/assistantpresets/use_assistant_preset_manager.ts`
12. `app/apis/interface.ts`
13. `app/spec/*.ts`

## Functional layout

### App shell and desktop bootstrap

Key files:

- `app/routes.ts`
- `app/root.tsx`
- `app/entry.client.tsx`
- `app/components/sidebar.tsx`
- `app/components/title_bar.tsx`
- `app/components/page_frame.tsx`
- `app/home/page.tsx`

Route map:

- `/` -> home
- `/chats`
- `/assistantpresets`
- `/skills`
- `/tools`
- `/prompts`
- `/modelpresets`
- `/docs`
- `/settings`

Important bootstrap behavior:

- `root.tsx` injects Wails scripts only when `VITE_PLATFORM === 'wails'`.
- `clientLoader()` waits for DOM readiness, starts the attachment-drop listener, then initializes:
  - built-in provider/preset data
  - startup theme
- `attachmentsDropAPI.setNoTargetHandler()` navigates to `/chats` when a file drop occurs outside an active chat drop target.
- `ensureWorker()` is scheduled on idle to initialize code highlighting.
- `CustomThemeProvider` reads the startup theme synchronously after bootstrap, with fallback to system theme if needed.
- `TitleBar` shows the app name/version and wraps window controls:
  - minimize
  - maximize/restore
  - quit
- The title bar is draggable except for `.app-no-drag` elements.

Page framing:

- Most pages use `PageFrame`.
- `chats` and `docs` pass `contentScrollable={false}` because they manage their own scroll containers.
- `PageFrame` is just layout; it does not own data.

### Home surface

File: `app/home/page.tsx`

What it does:

- sets title-bar center content to a home icon
- shows a primary action card for Chats
- shows docs shortcut cards
- acts as a landing page, not a data-heavy workspace

### Chats workspace

This is the core frontend system.

Main files:

- `app/chats/page.tsx`
- `app/chats/tabs/use_chats_controller.ts`
- `app/chats/tabs/chat_tabs_bar.tsx`
- `app/chats/tabs/tabs_model.ts`
- `app/chats/tabs/tabs_persistence.ts`
- `app/chats/conversation/conversation_area.tsx`
- `app/chats/conversation/conversation_input_pane.tsx`
- `app/chats/conversation/use_send_message.ts`
- `app/chats/conversation/use_streaming_runtime.ts`
- `app/chats/conversation/use_input_registry.ts`
- `app/chats/conversation/use_scroll_restore.ts`
- `app/chats/conversation/hydration_helper.ts`
- `app/chats/conversation/conversation_persistence_mapper.ts`
- `app/chats/search/*`
- `app/chats/messages/*`
- `app/chats/composer/*`

The chat page is not just a message list.
It coordinates:

- tab lifecycle
- conversation hydration
- conversation search and reopen
- scroll restoration
- streaming responses
- edit/replay flows
- the composer subsystem

#### Chat tabs and workspace state

Important behaviors from `use_chats_controller.ts` and `tabs_model.ts`:

- Workspace tabs are browser-local UI state.
- Real conversation content remains in backend storage.
- Tabs are restored from `localStorage`.
- There is always a scratch tab available so the user can start a new conversation.
- The workspace enforces a max tab count of `MAX_TABS = 16`.
- When the limit is reached, the least-recently-used non-scratch tab is evicted.
- The controller tracks:
  - selected tab
  - tab scroll position
  - last activation timestamps
  - which tabs are hydrated
  - which tabs are still mounted in the editor area

Persistence behavior:

- tab layout and scroll position are stored locally
- conversation content is persisted through `conversationStoreAPI`
- `saveUpdatedConversation()` chooses between:
  - `putConversation()` for new/full-save/title-change cases
  - `putMessagesToConversation()` for simple message updates

Important detail:

- tab titles can auto-generate early in the thread
- manual rename locks the title against later auto-renaming

#### Conversation hydration and restoration

`conversation_area.tsx`, `hydration_helper.ts`, and `conversation_persistence_mapper.ts` define the restore model.

Key behavior:

- stored conversation data is hydrated into UI-friendly conversation state
- derived UI fields include:
  - rendered markdown text
  - reasoning content
  - tool calls
  - tool outputs
  - citations
  - debug details
- restoring a conversation also restores composer context:
  - model preset ref
  - model params
  - tool choices
  - web search choices
  - enabled skills
  - active skills
- if the backend record is missing, the tab falls back to a blank conversation
- restoring a saved conversation clears any in-flight stream for that tab before syncing the composer

#### Search and reopening conversations

Search is part of the workspace.

Files:

- `app/chats/search/conversation_search.tsx`
- `app/chats/search/use_conversation_search.ts`
- related dropdown/row/util files

Behavior:

- search appears in the title bar on the chats page
- it can reopen saved conversations into tabs
- open conversation IDs are tracked so the UI can avoid duplicates
- restoring a selected result can reuse the scratch tab instead of opening an unlimited number of tabs

#### Timeline and streaming

The timeline is split between stored messages and transient stream state.

- `useStreamingRuntime()` keeps:
  - abort controllers per tab
  - request IDs per tab
  - streamed text/thinking buffers
  - per-tab listeners
- the active assistant message is rendered as a streaming target while the request is in flight
- scroll buttons appear when the active timeline is not at top/bottom
- the timeline supports editing earlier user messages

Important rendering details:

- `ChatMessage` receives both the stored message and any streamed text/thinking
- the last assistant message in a busy tab is the one that reflects live stream updates
- streamed state is transient and is cleared when the request resolves or the tab is disposed

#### End-to-end send flow

The main send path is in `useSendMessage.ts`.

The functional flow is:

1. user submits from the composer
2. the current draft is validated
3. pending tool calls may be run first, depending on the action chosen
4. a user message is appended locally
5. an assistant placeholder is appended locally
6. `HandleCompletion()` is called with:
   - provider
   - model preset
   - model params
   - current user message
   - previous history
   - tool choices
   - skill session ID
   - request ID
   - abort signal
   - stream callbacks
7. partial stream text/thinking is buffered in the frontend
8. final assistant content is persisted back into the conversation
9. any returned tool calls may be reloaded into the composer for manual execution

Important edge cases:

- if the backend errors after streaming partial text, the frontend preserves the streamed content and appends a terminal-style error line
- if the request is aborted before any tokens arrive, the assistant placeholder is removed
- if the request is aborted after partial tokens arrive, the partial response is preserved as a completed terminal-style message
- editing an earlier user message replaces the edited message and all later messages in that tab

#### Composer subsystem

The composer is the most complex frontend subsystem.

Top-level files:

- `app/chats/composer/composer_box.tsx`
- `app/chats/composer/editor/editor_area.tsx`
- `app/chats/composer/editor/editor_bottom_bar.tsx`
- `app/chats/composer/editor/editor_chips_bar.tsx`
- `app/chats/composer/contextarea/*`
- `app/chats/composer/attachments/*`
- `app/chats/composer/tools/*`
- `app/chats/composer/toolruntime/*`
- `app/chats/composer/skills/*`
- `app/chats/composer/systemprompts/*`
- `app/chats/composer/assistantpresets/*`
- `app/chats/composer/platedoc/*`

The composer owns:

- the rich-text editor document
- attachments
- conversation-level tools
- per-message attached tools
- tool-call runtime
- skills
- assistant preset selection
- system prompt composition
- previous-message editing
- web search selection
- send / stop / fast-forward behavior

##### Editor model

The editor uses Plate.

Important behavior:

- the composer is not plain text
- template nodes and tool nodes are embedded in the document model
- template variables can block submit when required values are missing
- the editor can be restored from a previous conversation message
- editing an old message is treated as a replacement workflow, not an append workflow

##### Bottom bar and chips bar

The bottom bar is the picker surface.
The chips bar is the live representation of what is already attached.

The chips bar shows, in order:

- conversation tools
- standalone attachments
- directory groups
- per-message attached tools
- tool calls and tool outputs

The bottom bar provides:

- attachment picker
- system prompt selection
- prompt template picker
- tool picker
- skills picker
- web search controls
- command tips

##### Attachments

Files, folders, and URLs are normalized through backend APIs.

Important behaviors:

- the frontend does not materialize raw filesystem content itself
- attachments are requested from the backend through `backendAPI`
- directories are grouped in the composer so the user can remove a whole folder selection later
- duplicate attachments are deduplicated by identity key
- individual files over the single-file limit become error attachments
- directory traversal may produce overflow directories when limits are reached

Concrete limits in the composer layer:

- max single attachment: 16 MiB
- max files per directory: 128

##### Tools and tool runtime

There are three distinct tool-related states:

- conversation-level tool choices
- per-message attached tools
- runtime tool calls and tool outputs

Important behavior:

- SDK tools are provider-family specific and are filtered by the current provider SDK type
- web search is treated as a special tool category
- tool args are validated before submit
- pending tool calls can be:
  - run one by one
  - fast-forwarded
  - discarded
  - retried if they fail
- tool outputs can be removed before submit
- tool calls with `autoExecute` can be drained automatically when the composer is not blocked

The runtime is split into:

- `useComposerToolConfig()`
- `useComposerToolRuntime()`
- `useToolAutoExecDrainer()`

##### Skills

Skills have their own catalog and session model.

Important behavior:

- the composer loads the skill catalog from the store
- enabled refs and active refs are tracked separately
- a skill session is created or refreshed when needed
- the active skill set is snapped just before submit
- the composer can clear, enable all, or disable all skills
- missing or stale skill refs are filtered out when the catalog is loaded
- skill sessions are closed on unmount

##### Assistant presets

Assistant preset behavior is split into selection, compatibility, and application.

Important reserved identities:

- base assistant preset bundle ID: `__conversation__`
- base assistant preset slug: `base`
- base assistant preset version: `v1.0.0`
- synthetic previous-conversation system prompt identity:
  - `__conversation__:previous-system-prompt`

What the preset manager does:

- loads assistant preset options
- checks whether the preset is currently selectable
- checks model, prompt, tool, and skill compatibility
- applies selected preset state to the composer
- tracks a runtime snapshot so it can show what changed
- can reset to the base preset
- can reapply the current preset
- can track the default preset without immediately applying it

Important detail:

- assistant preset application is capability-aware
- SDK tool selections are validated against the selected model’s provider SDK type
- a preset can be visible but not selectable if one of its required references is unavailable

##### System prompts

System prompt composition is its own controller.

Important behavior:

- the effective system prompt is built from:
  - model default prompt
  - user-selected prompt sources
  - the include-model-default toggle
- a restored conversation can create a synthetic “previous conversation prompt” source
- prompt templates are selected from the prompt catalog and can be added from the composer
- assistant preset application can also drive system prompt selection

##### Submit and abort behavior

The composer supports three primary actions:

- send only
- run tools only
- run tools and send

Important guardrails:

- submit is blocked when required template variables are missing
- submit is blocked when tool or web-search args are incomplete
- submit is blocked while the editor is locked or generating
- tool calls must finish before send if the submit path depends on them
- abort preserves partial output that has already arrived

##### Composer input restoration

When editing an earlier message, the composer snapshots the current context first so it can restore:

- conversation tools
- web search choices
- enabled skills
- active skills

If the edit is canceled or the edited message becomes empty, the prior context is restored.

### Assistant presets page

Files:

- `app/assistantpresets/page.tsx`
- `app/assistantpresets/*`

Behavior:

- loads assistant preset bundles and presets, including disabled entries
- shows bundles with nested preset versions
- allows:
  - adding bundles
  - enabling/disabling bundles
  - adding/editing preset versions
  - enabling/disabling preset versions
  - deleting user-created preset versions
  - deleting empty user-created bundles
- built-in bundles and built-in presets are read-only except for enable/disable overlay state
- bundle/preset mutations are refreshed bundle-by-bundle after save

### Model presets page

Files:

- `app/modelpresets/page.tsx`
- `app/modelpresets/*`

Behavior:

- loads current settings, default provider, and provider presets in parallel
- shows provider cards with their model presets
- lets the user:
  - add provider presets
  - edit provider presets
  - enable/disable providers
  - delete providers if they are empty and not default
  - add/edit/delete model presets per provider
  - set the default provider
  - set the default model per provider
- exports the provider/model catalog as JSON
- automatically repairs the default provider if the stored default is missing or disabled

Important guardrails:

- cannot disable the current default provider without choosing another one first
- cannot disable the last enabled provider
- cannot delete built-in providers
- cannot delete a provider that still has model presets
- cannot disable the default model of a provider without changing the default first
- built-in model presets are read-only

### Prompts page

Files:

- `app/prompts/page.tsx`
- `app/prompts/*`

Behavior:

- loads prompt bundles and prompt templates, including disabled entries
- each bundle contains versioned template files
- allows:
  - adding bundles
  - enabling/disabling bundles
  - adding/editing prompt templates
  - enabling/disabling prompt versions
  - deleting user-created template versions
  - deleting empty user-created bundles
- when saving a template, the page derives:
  - template kind
  - resolved status
  - validation-related fields
- built-in prompt bundles/templates are read-only except for enable/disable overlay state

### Skills page

Files:

- `app/skills/page.tsx`
- `app/skills/*`

Behavior:

- loads skill bundles and skill definitions
- includes missing entries so broken refs can be surfaced
- allows:
  - adding bundles
  - enabling/disabling bundles
  - adding/editing skills
  - enabling/disabling skills
  - deleting skills
  - deleting bundles
- uses request/version guards so stale async refreshes do not overwrite newer state
- shows action-denied alerts when backend rules reject a mutation

### Tools page

Files:

- `app/tools/page.tsx`
- `app/tools/*`

Behavior:

- loads tool bundles and tool definitions
- allows:
  - adding bundles
  - enabling/disabling bundles
  - adding/editing tools
  - enabling/disabling tools
  - deleting tools
  - deleting bundles
- new tools default to HTTP-style tool definitions in the UI
- the backend still enforces the actual implementation rules and allowed types

Important behavior:

- the page is a catalog editor, not an executor
- runtime tool execution happens in the chats composer, not here

### Settings page

Files:

- `app/settings/page.tsx`
- `app/settings/*`

Behavior:

- shows theme selection
- shows auth keys
- shows debug settings
- exports the current settings JSON
- opens modals to add or edit auth keys
- theme and debug changes go through `settingstoreAPI`

Important detail:

- startup theme is loaded on app bootstrap and then reflected in the frontend theme provider

### Docs surface

Files:

- `app/docs/page.tsx`
- `app/docs/manifest.ts`
- `app/docs/content/*.md`

Behavior:

- renders bundled markdown content in-app
- uses a route-aware query parameter:
  - `?doc=<section-id>`
- also supports legacy hash-only links and normalizes them
- intercepts internal markdown links so docs navigation stays inside the app
- shows previous/next section controls
- uses a responsive layout with its own scroll handling

Docs content is grouped into:

- User Guide
- Architecture

The docs renderer uses `EnhancedMarkdown`.

## Shared frontend infrastructure

### Typed backend boundary

Files:

- `app/apis/baseapi.ts`
- `app/apis/interface.ts`
- `app/apis/list_helper.ts`
- `app/apis/wailsapi/*`
- `app/apis/wailsjs/*`

Important behavior:

- `baseapi.ts` chooses Wails implementations only on the Wails platform
- `app/apis/interface.ts` is the canonical typed contract for frontend-to-backend calls
- `list_helper.ts` handles paginated list APIs for catalog domains
- `app/apis/wailsapi` is the hand-written typed wrapper layer
- `app/apis/wailsjs` is generated Wails binding output

If you add a backend-facing frontend feature, this is usually the first place to update.

### Spec layer

Files:

- `app/spec/*.ts`

These files are the frontend-side canonical shapes for:

- conversations
- attachments
- inference
- model presets
- prompts
- settings
- skills
- tools
- tool runtime
- assistant presets

Treat `spec` as the cross-boundary contract layer.
If a feature changes data shape, update `spec` first or alongside the backend contract.

### Shared hooks and utilities

Important hooks:

- `use_title_bar.tsx`
- `use_startup_theme.tsx`
- `use_theme_provider.tsx`
- `use_highlight.tsx`
- `use_builtin_provider.tsx`
- `use_enter_submit.tsx`
- `use_chat_shortcuts` in `lib/keyboard_shortcuts.ts`
- `use_event.ts`
- `use_debounce.tsx`
- `use_tool.ts`

Important shared behavior:

- title bar content is managed through an external store-like API
- startup theme is loaded once and then reused
- code highlighting runs through a dedicated worker
- global chat shortcuts are capture-phase listeners and avoid firing inside menus/dialogs
- the app uses `MOD_LABEL` to adapt shortcut display to Mac-like vs non-Mac platforms

### Shared components

Important shared UI pieces:

- `Sidebar`
- `TitleBar`
- `PageFrame`
- `Loader`
- `DownloadButton`
- `ActionDeniedAlertModal`
- `DeleteConfirmationModal`
- `HoverTip`
- markdown rendering components
- chips, dropdowns, and modal helpers

### Markdown rendering

Files:

- `app/components/markdown_enhanced.tsx`
- `app/components/markdown_code_block.tsx`
- `app/components/thinking_fence.tsx`
- `app/components/mermaid_diagram_card.tsx`
- `app/components/markdown_error_boundary.tsx`

Important behavior:

- supports GFM markdown
- supports math via KaTeX
- supports raw HTML with sanitization
- supports mermaid rendering
- supports code highlighting
- supports special thinking fences
- opens external links through the backend URL opener
- lets internal docs links be intercepted by the docs page

## Key end-to-end flows

### Open app

1. `root.tsx` boots the shell.
2. backend-built-in provider data and startup theme are initialized.
3. the Wails runtime is injected if the app is running on Wails.
4. the sidebar and title bar become active.
5. the chats workspace restores its local tab state.

### Start or restore a chat

1. the chats controller restores tab layout from local storage or creates a scratch tab.
2. the selected tab is hydrated if needed.
3. the composer is synchronized from stored conversation context.
4. the timeline is restored with derived UI fields and scroll position.

### Send a turn

1. the composer validates templates, attachments, tools, and skills.
2. tool calls may run first.
3. the request is submitted with the current model and prompt state.
4. stream data is buffered and rendered live.
5. the final result is written back to the conversation store.
6. the composer resets or retains state depending on the flow.

### Edit an older message

1. the composer snapshots current conversation-tool, web-search, and skill state.
2. the older message is loaded into the editor.
3. sending replaces that message and all later messages.
4. cancel restores the pre-edit context.

## Important frontend invariants

- `chats` is the primary workspace and owns conversation coordination.
- The composer is a real subsystem, not a simple text box.
- Backend data is the source of truth for conversations and catalogs.
- Browser local storage is only for workspace convenience state.
- Built-in catalog data is loaded first and then merged with user content through the backend contract.
- SDK tools are provider-family specific.
- Web search is a special tool path and is not treated exactly like ordinary per-message tools.
- Tool args, template variables, and skill selection can block submit.
- Streaming state is transient and should not be confused with persisted conversation state.
- The active assistant message during generation is a live stream target.
- Dropped files should route to chats if there is no active drop target.
- Docs navigation is route-aware and internal links should remain inside the app when possible.

## What to change where

- If you change shell or window behavior, start with `root.tsx`, `Sidebar`, `TitleBar`, and `use_title_bar.tsx`.
- If you change chat workflows, start with `use_chats_controller.ts`, `conversation_area.tsx`, and `use_send_message.ts`.
- If you change draft construction or submit rules, start with `ComposerBox`, `EditorArea`, `useComposerTools`, `useComposerSkills`, `useComposerAttachments`, and `useComposerSystemPrompt`.
- If you change provider/model/preset behavior, update the matching page plus the relevant `spec/*` and `apis/*` contracts.
- If you change markdown or docs rendering, update `docs/page.tsx` and `markdown_enhanced.tsx`.
- If you change the typed backend boundary, update `app/apis/interface.ts` and the Wails wrapper layer together.

## Frontend file ownership map

- `app/root.tsx`
  - bootstrap, hydration, theme, drop bridge, app shell
- `app/chats/`
  - workspace, composer, streaming, search, timeline
- `app/assistantpresets/`
  - assistant preset catalog editor
- `app/modelpresets/`
  - provider and model catalog editor
- `app/prompts/`
  - prompt bundle and template editor
- `app/skills/`
  - skill bundle and skill editor
- `app/tools/`
  - tool bundle and tool editor
- `app/settings/`
  - app settings, auth keys, debug, theme
- `app/docs/`
  - bundled docs viewer
- `app/apis/`
  - typed backend boundary
- `app/spec/`
  - cross-boundary data contracts
- `app/components/`
  - shared UI primitives
- `app/hooks/`
  - stateful cross-cutting hooks
- `app/lib/`
  - pure helper utilities
