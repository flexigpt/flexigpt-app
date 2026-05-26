# FlexiGPT backend map

> FlexiGPT is a local-first AI workspace built as a Wails desktop app with a Go backend.
>
> This document is organized around backend architecture, data flow, and store/runtime behavior so an LLM can understand the system quickly.
> HTTP handler files are intentionally omitted because they are not useful for architecture-first backend context.

## Backend architecture in one view

FlexiGPT’s backend is built around a few stable ideas:

- The frontend talks to a local Go backend through Wails bindings.
- Durable application state lives on the user’s machine under the XDG data directory.
- Built-in catalog data ships in the binary through embedded filesystem trees.
- Built-in content is mutable only through overlay flags stored in SQLite, not by editing embedded assets.
- File-backed stores use `mapstore-go` for persistence and directory traversal.
- `inference-go` handles provider orchestration and completion flows.
- `llmtools-go` provides built-in Go tool execution and registry plumbing.
- `agentskills-go` provides the runtime skill model, sessions, and skill tool execution support.
- `attachment` normalizes external context into model-ready content blocks before inference.
- Wrapper files are intentionally thin except for the aggregate wrapper, which coordinates provider state, auth keys, debug settings, and completion orchestration.

## Cross-cutting patterns

These patterns show up in most subsystems:

- Built-in data is embedded, read-only, and toggled through overlay state.
- User-created data is stored on disk and validated on write.
- Many list/read paths validate stored data again so stale or corrupted records can be skipped safely.
- Pagination is usually based on opaque base64-encoded JSON tokens.
- The skill subsystem is the most stateful one because it keeps disk state and runtime state synchronized.
- Store identity and runtime identity are different:
  - stores use bundle/slug/version or `SkillRef`
  - runtime uses provider/tool/skill internals
- Attachment normalization is part of inference setup, not UI only.
- Most subsystems favor best-effort skipping of malformed records over hard crashes.

## How to read this map

Read in this order if you want the fastest mental model of the backend:

1. `cmd/agentgo/main.go`
2. `cmd/agentgo/app.go`
3. `cmd/agentgo/wrapper_aggregate.go`
4. `internal/setting/store/store.go`
5. `internal/modelpreset/store/store.go`
6. `internal/prompt/store/store.go`
7. `internal/tool/store/store.go`
8. `internal/skill/store/store.go`
9. `internal/assistantpreset/store/store.go`
10. `internal/inferencewrapper/provider_set.go`
11. `internal/toolruntime/toolruntime.go`
12. `internal/attachment/attachment.go`
13. `internal/llmtoolsutil/registry.go`

## Functional layout

### Desktop bootstrap and Wails boundary

Package: `cmd/agentgo`

This package is the app shell and binding layer for the backend.

Files:

- `main.go`
  - Wails entrypoint
  - logging setup
  - log rotation setup
  - asset server setup
  - backend binding registration
  - application startup/shutdown hooks
- `app.go`
  - app struct
  - local data directory setup
  - manager/store initialization order
  - lifecycle hooks
- `apputils.go`
  - embedded asset URL rewriting
  - stack trace logging helper
  - embedded FS debugging helper
- `logger_adapter.go`
  - adapts `slog.Logger` to the Wails logger interface
- `wrapper_aggregate.go`
  - provider orchestration
  - auth key synchronization
  - completion streaming and cancellation coordination
  - live debug setting application
- `wrapper_setting.go`
- `wrapper_conversation.go`
- `wrapper_modelpreset.go`
- `wrapper_prompt.go`
- `wrapper_tool.go`
- `wrapper_toolruntime.go`
- `wrapper_skill.go`
- `wrapper_assistantpreset.go`
- `wrapper_attachment.go`
  - thin Wails-facing adapters for the relevant stores and helper APIs
- `wails.json`
  - shell configuration, not core backend logic

Important startup behavior:

- `main.go` imports `internal/llmtoolsutil` for registry initialization side effects.
- `NewApp()` creates the local data layout under `xdg.DataHome/flexigpt`.
- Logging is routed through a rotating file writer under `logs/` in the app data directory.
- `initManagers()` order matters because later stores depend on earlier stores:
  - conversation
  - prompts
  - tools
  - tool runtime
  - skills
  - model presets
  - assistant presets
  - settings
  - aggregate provider wrapper
- `startup()` stores the app context and shows the frontend window.
- `SetWrappedProviderAppContext()` is called during startup so the aggregate wrapper can emit Wails events and manage cancellation.

Shutdown behavior to remember:

- `App.shutdown()` currently closes only:
  - assistant presets
  - tools
  - prompts
  - skills
- `conversation`, `modelpreset`, and `setting` stores expose `Close()` methods, but `App.shutdown()` does not currently call them.
- If you add more background workers or runtime resources, this shutdown path should be revisited.

### Shared utilities and guardrails

Package: `internal/middleware`

Files:

- `recover.go`

Functionality:

- panic recovery for backend calls
- stack trace logging
- consistent error wrapping for Wails-exposed methods

Critical behavior:

- wrapper files use it to prevent panics from escaping into the frontend boundary.

Package: `internal/logrotate`

Files:

- `writer.go`
- `rand.go`

Functionality:

- rotating log writer for backend logs

Critical behavior:

- concurrency-safe logging
- rotation by size and lifetime
- used by the application-wide logger in `main.go`

Package: `internal/jsonutil`

Files:

- `base64.go`
- `raw.go`

Functionality:

- compact opaque token encoding and decoding
- stricter JSON handling helpers

Critical behavior:

- used heavily for page tokens and store-side payload handling

Package: `internal/fsutil`

Files:

- `fsutil.go`

Functionality:

- resolve a subdirectory from an `fs.FS`

Critical behavior:

- used by embedded content loaders and built-in filesystem traversal

Package: `internal/bundleitemutils`

Files:

- `type_const.go`
- `validate.go`
- `bundlepartition.go`
- `filename.go`
- `dirname.go`

Functionality:

- shared bundle/item naming and validation rules
- filename and directory derivation for versioned catalog items

Critical behavior:

- ensures all bundle-based domains use consistent slug/version and directory conventions
- guards against bad filenames, versions, and bundle directory names

Package: `internal/overlay`

Files:

- `store.go`
- `valgroup.go`
- `type_const.go`

Functionality:

- tiny SQLite-backed overlay store
- typed overlay flag access

Critical behavior:

- stores small mutable flags keyed by group and key
- used to toggle built-in catalog content on/off
- also used by some built-in stores for default selection overlays

### Built-in content and embedded catalogs

Package: `internal/builtin`

Files:

- `builtin.go`
- `async_rebuild_helper.go`

Embedded content roots:

- `tools/`
- `prompts/`
- `skills/`
- `assistantpresets/`

Manifest files:

- `tools.bundles.json`
- `prompts.bundles.json`
- `skills.json`
- `assistantpresets.bundles.json`

Functionality:

- embedded built-in catalogs
- stable bundle IDs and root constants
- lazy rebuild helper for snapshot refreshes

Critical behavior:

- built-in data is shipped inside the binary
- overlay state controls whether built-ins appear enabled or disabled
- snapshot rebuild is asynchronous and guarded against concurrent rebuilds
- built-in stores keep an immutable base snapshot plus an overlay-applied view snapshot

### Local settings and conversation state

Package: `internal/setting`

Files:

- `spec/type_const.go`
- `spec/req_resp.go`
- `store/builtin_data.go`
- `store/validate.go`
- `store/store.go`

Functionality:

- application settings
- theme and debug settings
- auth key storage and retrieval

Critical behavior:

- auth secrets are encrypted using an OS keyring-backed encoder/decoder
- built-in auth keys are preserved and read-only
- migration fills in missing built-in auth keys and normalizes debug settings
- runtime debug settings are applied live through an injected applier
- `GetSettings()` strips secrets and returns only metadata for auth keys
- `SetAuthKey()` also updates the SHA and non-empty metadata fields

Package: `internal/conversation`

Files:

- `spec/type_const.go`
- `spec/req_resp.go`
- `store/store.go`
- `store/ftslistner.go`

Functionality:

- conversation transcript persistence
- conversation listing and search

Critical behavior:

- conversations are file-backed, one JSON file per conversation
- filenames are derived from UUID-v7 plus title
- `PutConversation()` replaces the whole transcript file
- `PutMessagesToConversation()` updates the existing conversation content
- list/search paths skip malformed or stale files rather than crashing
- FTS search is optional and only enabled when the store is constructed with `WithFTS(true)`
- the FTS index tracks title, system, user, and assistant text

### Catalog domains

These are the content systems that merge built-in content with user-created content.

#### Model presets

Package: `internal/modelpreset`

Files:

- `spec/type_const.go`
- `spec/req_resp.go`
- `store/builtin_data.go`
- `store/builtin_overlay.go`
- `store/modelpreset_patch.go`
- `store/provider_patch.go`
- `store/validate.go`
- `store/clone.go`
- `store/store.go`

Functionality:

- provider presets
- model presets
- default provider selection
- built-in preset exposure

Critical behavior:

- built-ins are derived from `inference-go`’s default model catalog
- built-ins are read-only except for:
  - `isEnabled`
  - `defaultModelPresetID`
- user-defined provider/model presets live in a local JSON-backed map store
- list APIs merge built-ins and user presets
- page tokens are opaque and preserve filters/cursors
- validations reject malformed or built-in-mutation attempts
- `GetModelPreset()` returns the provider and model pair for the requested preset and intentionally omits the full provider model map from the response
- `PatchDefaultProvider()` only checks existence, not enabled state
- `DeleteProviderPreset()` refuses to delete non-empty providers and the selected default provider

Useful detail:

- provider patches and model patches are intentionally split into separate helper files.

#### Prompt templates

Package: `internal/prompt`

Files:

- `spec/type_const.go`
- `spec/req_resp.go`
- `store/builtin_data.go`
- `store/clone.go`
- `store/sluglock.go`
- `store/validate_template.go`
- `store/store.go`

Functionality:

- prompt bundles
- versioned prompt templates
- reusable instruction and template catalog

Critical behavior:

- the base bundle is reserved and hydrated on startup
- built-in bundles/templates are overlaid with local enable flags
- bundle deletion is soft-delete plus cleanup after a grace period
- template versions are stored as individual JSON files under a bundle directory
- list APIs emit built-ins first, then user templates
- template validation is strict:
  - block roles must match the template kind
  - block IDs must be unique
  - placeholders must be declared as variables
  - non-static variables must be used
  - `isResolved` is computed and enforced
  - enum defaults must exist in the enum list

Important restrictions:

- reserved bundle ID and slug are protected
- built-in bundles are read-only except for overlay enable/disable state

#### Tools

Package: `internal/tool`

Files:

- `spec/type_const.go`
- `spec/req_resp.go`
- `spec/choice.go`
- `storehelper/validate_tool.go`
- `storehelper/sluglock.go`
- `store/builtin_data.go`
- `store/builtin_gotool.go`
- `store/store.go`

Functionality:

- tool bundles
- versioned tool definitions
- tool choice and selection helpers
- built-in tool catalog loading

Critical behavior:

- only HTTP tools are user-creatable
- Go tools and SDK tools are treated differently
- built-ins are read-only except for overlay enable/disable state
- `llmtools-go` built-ins can be injected into the app’s tool catalog
- list APIs emit built-ins first, then user tools
- tool bundles can be soft-deleted if empty
- validation is centralized in `storehelper.ValidateTool`
- SDK tools are provider-side tools and are not executed through `ToolRuntime`

Validation highlights:

- `schemaVersion`, slug, version, and `argSchema` must be valid
- `LLMToolType` must be `function`, `custom`, or `webSearch`
- HTTP tools require `HTTPImpl`
- Go tools require `GoImpl`
- SDK tools require `SDKImpl`
- HTTP tools validate URL template, method, success codes, and error mode
- user-created HTTP tools must be callable by at least one of:
  - `UserCallable`
  - `LLMCallable`

Useful detail:

- `ToolStoreChoice` and `ToolSelection` are the persisted selection shapes used by assistant presets and completion hydration.

#### Skills

Package: `internal/skill`

Files:

- `spec/type_const.go`
- `spec/req_resp.go`
- `spec/runtime_req_resp.go`
- `store/builtin_data.go`
- `store/validate.go`
- `store/hydrate_embeddedfs.go`
- `store/runtime.go`
- `store/invoke.go`
- `store/list.go`
- `store/store_saga.go`
- `store/store_util.go`
- `store/sluglock.go`
- `store/store.go`

Functionality:

- skill bundles
- skill runtime catalog
- session/prompt/runtime coordination
- built-in skill hydration from embedded filesystem data

Critical behavior:

- this is the most stateful subsystem
- built-in skills are `embeddedfs`
- user skills can only be `fs`
- runtime is required; if no runtime is provided, the store creates the default `fsskillprovider`-backed runtime
- built-in skills can be hydrated to a real directory when required
- runtime state is synchronized with disk state using strict write-saga logic
- runtime/store mutation failures can trigger rollback
- list APIs merge built-ins and user skills, but use a two-phase cursor model
- session and skill prompt APIs are part of the store, not just CRUD
- `InvokeSkillTool()` executes built-in skill tools via the runtime registry bridge

Important runtime behavior:

- enabling a built-in skill bundle or skill may trigger embedded FS hydration
- the store uses `agentskills-go` runtime primitives to create sessions, list runtime skills, and build prompts
- allowlists are best-effort and stale-safe:
  - unresolved refs are skipped
  - empty allowlists behave like “no skills allowed” instead of hard error in some flows
- built-in skill tools exposed to the model include:
  - `skills-load`
  - `skills-unload`
  - `skills-readresource`
  - `skills-runscript`
- whether `skills-runscript` is advertised depends on the store configuration

Skill identity details:

- `SkillRef` is the persisted store identity
- runtime-facing identity is derived from the skill record
- `SkillID` is required in runtime-facing refs to avoid stale-ref ambiguity

### Assistant presets

Package: `internal/assistantpreset`

Files:

- `spec/type_const.go`
- `spec/req_resp.go`
- `store/builtin_data.go`
- `store/clone.go`
- `store/lookups.go`
- `store/validate_preset.go`
- `store/sluglock.go`
- `store/store.go`
- `lookupimpl/lookupimpl.go`

Functionality:

- assistant presets
- cross-store reference resolution
- reusable starter configurations that combine model, prompt, tool, and skill choices

Critical behavior:

- presets are validated against other stores before they are accepted
- built-in assistant presets are read-only except for overlay enable/disable state
- versioned presets are stored as individual JSON files inside a bundle directory
- bundle deletion is soft-delete plus cleanup after a grace period
- list APIs emit built-ins first, then user presets
- lookups are injected through interfaces so validation is decoupled from concrete stores

Validation highlights:

- `StartingModelPresetRef` must resolve and be enabled
- `StartingInstructionTemplateRefs` must resolve, be enabled, be `instructionsOnly`, and be `isResolved == true`
- `StartingToolSelections` must resolve to enabled tools
- `StartingSkillSelections` must resolve to enabled skills
- `StartingModelPresetPatch` must not set `systemPrompt`
- `StartingModelPresetPatch` must not set `capabilitiesOverride`

Useful detail:

- assistant presets are the cross-store composition layer and are where model/prompt/tool/skill choices come together.

### Execution and request orchestration

Package: `internal/inferencewrapper`

Files:

- `provider_set.go`
- `spec/req_resp.go`

Functionality:

- provider orchestration
- completion request assembly
- streaming response delivery
- provider API key management
- attachment and skill hydration for inference requests

Critical behavior:

- converts stored app state into provider-ready requests
- hydrates tools, skills, and attachment context into completion requests
- handles streaming text and thinking callbacks
- manages request cancellation and reuse safety
- applies debug settings live to the provider set
- rejects pre-populated tool choices in completion requests
- `FetchCompletion()` requires:
  - non-empty provider
  - non-empty model preset ID
  - non-empty request ID
  - app context set during startup
- current turn must have role `user`
- model params are taken from the request body or the most recent historical turn
- if `MaxPromptLength` is missing, it defaults to 8000
- attachments are normalized into content blocks and merged into the current user message when possible
- the response includes hydrated current inputs for replay/persistence

Skill-specific completion behavior:

- if a skill session is active, the wrapper may inject:
  - skill prompt text
  - skill-specific tool choices
- the skill tool set depends on whether any skills are currently active
- streaming events are bridged into Wails events using the callback IDs supplied by the frontend

Cancellation behavior:

- request IDs are tracked so a cancel can arrive before the fetch is registered
- pre-cancelled request IDs are remembered briefly and then pruned
- duplicate request IDs while a completion is in flight are rejected

Note:

- `httphandler.go` exists in the repo but is intentionally omitted from this map.

### Tool runtime and request execution

Package: `internal/toolruntime`

Files:

- `toolruntime.go`
- `httprunner/runner.go`
- `spec/req_resp.go`

Functionality:

- tool invocation runtime
- execution of Go and HTTP tools

Critical behavior:

- validates the request and loads the tool definition from the tool store
- dispatches HTTP tools to the HTTP runner
- dispatches Go tools through `llmtoolsutil`
- returns outputs plus metadata
- marks whether the tool was built-in
- marks whether the tool execution failed and includes an error message
- SDK tools are not invoked through this runtime

Useful detail:

- the Wails wrapper for tool runtime is thin and just forwards into this package.

### Attachment normalization

Package: `internal/attachment`

Files:

- `attachment.go`
- `file.go`
- `image.go`
- `url.go`
- `generic.go`
- `format_text.go`
- `aggregate_helper.go`
- `dir_walk.go`
- `extension.go`
- `file_filter.go`
- `type_const.go`
- `type_const_content_block.go`
- `url_extract.go`

Functionality:

- attachment normalization
- file/image/url conversion into model-ready content blocks
- formatting attachment data into readable prompt text

Critical behavior:

- resolves files, images, directories, URLs, and generic attachment refs into content blocks where supported
- falls back to display-only text when content cannot be safely materialized
- detects changed or stale attachments
- respects snapshot modification checks unless the caller explicitly overrides them
- serves as the input normalization layer before request execution

Important detail:

- `BuildContentBlock()` can return a readable display-text fallback for binary or unreadable files.

### LLM tool registry bridge

Package: `internal/llmtoolsutil`

Files:

- `registry.go`
- `wrapper.go`
- `caller.go`

Functionality:

- centralized access to `llmtools-go` built-ins
- tool registry bootstrap and call normalization

Critical behavior:

- creates the default Go registry during package initialization
- maps stable function IDs to app bundle IDs
- normalizes tool outputs from the registry
- imported in `main.go` for registry initialization side effects

Important detail:

- this is the bridge that lets `ToolStore` and `ToolRuntime` execute Go tool functions through `llmtools-go`.

## Data layout summary

Main local data roots created under the app data directory:

- `settings/`
- `conversationsv1/`
- `modelpresetsv1/`
- `prompttemplatesv1/`
- `toolsv1/`
- `skills/`
- `assistantpresetsv1/`
- `logs/`

Primary storage shapes:

- `settings.json`
  - app settings, theme, debug settings, auth key metadata
- `conversationsv1/`
  - one JSON file per conversation
- `modelpresets.json`
  - all provider and model preset data
- `prompttemplatesv1/` plus `prompts.bundles.json`
  - prompt bundles and versioned templates
- `toolsv1/` plus `tools.bundles.json`
  - tool bundles and versioned tools
- `skills.bundles.json`
  - skill bundles and skill definitions in one map-backed file
- `assistantpresetsv1/` plus `assistantpresetbundles.json`
  - assistant preset bundles and versioned presets

Additional persistent files:

- overlay SQLite files for built-in enable/disable flags and default-selection overlays
- optional FTS SQLite file for conversations
- `skills-embeddedfs-hydrated/` for the on-disk copy of built-in embedded skill files when hydration is needed

Typical storage patterns:

- one JSON meta file per bundle collection, or one JSON file for the full store
- one JSON file per versioned item where the collection is directory-backed
- overlay SQLite files for built-in state
- optional FTS SQLite files for searchable collections

## Startup and shutdown summary

Startup flow:

- resolve data directories
- initialize logging
- initialize stores in dependency order
- initialize provider orchestration
- bind wrappers to Wails
- show the window after startup and bind the app context for completion callbacks

Shutdown flow:

- close assistant presets
- close tools
- close prompts
- close skills
- conversation/modelpreset/settings stores are not currently closed here, even though some expose `Close()` methods

## What to remember as a maintainer

- `spec` packages define data shapes and API contracts.
- `store` packages define persistence, validation, and mutation rules.
- `builtin` packages define embedded read-only content plus overlay state.
- `wrapper_*.go` files are mostly Wails adapters.
- `wrapper_aggregate.go` is the one wrapper that contains meaningful orchestration logic.
- `inferencewrapper` is the bridge between stored app state and provider requests.
- `toolruntime` actually executes Go and HTTP tools.
- `skill` is the most stateful subsystem because it synchronizes disk state with a live runtime catalog and session model.
- `attachment` is the input normalization layer that turns files, URLs, and images into model-ready content.
- `ToolSelection`/`ToolStoreChoice` and `SkillRef` are store identities, not runtime identities.
- If you add new provider, prompt, tool, skill, or assistant-preset behavior, update the relevant lookup/adapters and runtime hydration paths together.
- Malformed disk records are often skipped in list/read paths, so explicit validation is important when adding new storage rules.
