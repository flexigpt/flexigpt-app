# Backend HLD

This page captures the backend's high-level design decisions.
It focuses on storage shapes, module boundaries, and the external libraries that
supply the heavy lifting for persistence, provider access, tools, and skills.

## Storage model

The backend uses storage shapes that match the kind of data being stored.

### Map-backed files for catalogs

Catalog-style data such as settings, presets, prompts, tools, skills, and assistant presets is stored locally using mapstore-backed files and directories.
That lets the backend treat each domain as structured local content rather than as opaque blobs.

### Directory stores for versioned items

Versioned content such as prompts, tools, and assistant presets naturally fits a directory layout.
The backend uses directory-oriented stores for those domains so each bundle can hold multiple versions and related items.

### Built-in overlay

Bundled defaults are stored separately from user-created content and then merged into the working view.
That keeps the shipped data effectively read-only while still letting the user manage local entries alongside it.

### SQLite overlay

Some local state is small, keyed, and mutable in a way that fits a lightweight database better than a flat file.
The overlay module exists for that kind of data.
It is a supporting persistence primitive rather than the main catalog store.

### Conversation FTS

Conversation search uses a dedicated FTS index rather than relying on the main conversation files alone.
That keeps text search and thread storage as separate concerns.

## Role of the main backend modules

| Module                     | High-level decision                                 | Why it exists                                                   |
| -------------------------- | --------------------------------------------------- | --------------------------------------------------------------- |
| **Settings store**         | Local settings with secure auth-key handling.       | Configuration should stay on the machine.                       |
| **Model preset store**     | Local model/provider catalog with built-in overlay. | Provider behavior should be configurable without hardcoding it. |
| **Conversation store**     | File-backed threads plus search index support.      | Chats need persistence and search.                              |
| **Prompt store**           | Reusable prompt bundles and template versions.      | Prompt structure should be shareable across chats.              |
| **Tool store**             | Reusable tool bundles and tool versions.            | Tool capability should be cataloged and selectable.             |
| **Skill store**            | Local skill catalog plus runtime synchronization.   | Skills need both storage and runtime presence.                  |
| **Assistant preset store** | Bundled starting workspaces.                        | A user should be able to start from a curated setup.            |
| **Request orchestration**  | Compose model requests from active context.         | This is where the app becomes agentic.                          |
| **Tool runtime**           | Execute HTTP and Go tool implementations.           | Tool execution needs a runtime boundary.                        |
| **Attachment layer**       | Convert files and URLs into content blocks.         | Provider requests need normalized inputs.                       |

## External libraries as architectural dependencies

### `mapstore-go`

This repository is the persistence substrate for the local catalog model.
It gives the backend the primitives for map files, directory stores, partitioning, listeners, and paged file access.
The local domains build on that substrate rather than inventing their own storage engine.

### `inference-go`

This repository is the provider execution layer.
It handles provider registration, capability lookup, request execution, and streaming responses.
The backend orchestration layer uses it to avoid owning provider transport logic itself.

### `llmtools-go`

This repository defines the tool model and the concrete tool implementations that the app exposes.
The backend uses it both to execute local tools and to present tool shapes to provider-side requests.

### `agentskills-go`

This repository supplies the skill runtime and the builtin skill tool shapes.
The backend skill store uses it to keep skill-aware workflows aligned with the runtime model that skills need.

## Design decisions that shape evolution

- Local data stays local, so the backend does not depend on a separate database server.
- Different data shapes use different local store shapes instead of forcing one storage pattern everywhere.
- Built-ins are shipped as catalog content, not as hardcoded UI defaults.
- Conversation search is indexed separately from the conversation thread store.
- Provider execution is delegated to the provider library layer rather than embedded into UI code.
- Tool execution and skill execution are separated from provider execution, even when they appear in the same conversation flow.

## How to read the backend from an architecture perspective

If a change affects persisted user data, look at the relevant store.
If it affects model requests or streaming, look at request orchestration.
If it affects tool behavior, look at the tool runtime and tool definitions.
If it affects skill-aware behavior, look at the skill store and the skill runtime bridge.
If it affects built-in content, look at the overlay model and the bundled catalogs.
