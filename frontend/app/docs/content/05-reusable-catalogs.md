# Reusable Catalogs

The pages outside **Chats** maintain reusable building blocks: assistant presets, tools, skills, MCP server catalogs, model presets, and app settings.

Use Chats to apply these things to a conversation. Use the catalog pages to create, inspect, version, enable, disable, or delete reusable definitions.

## Table of contents <!-- omit from toc -->

- [Catalog ownership](#catalog-ownership)
- [Assistant Presets](#assistant-presets)
  - [What an assistant preset can contain](#what-an-assistant-preset-can-contain)
  - [Empty preset sections](#empty-preset-sections)
  - [Inspecting a preset](#inspecting-a-preset)
  - [Modified state in Chats](#modified-state-in-chats)
  - [Versioning and built-ins](#versioning-and-built-ins)
- [Tools](#tools)
- [Skills](#skills)
- [Model Presets](#model-presets)
- [Settings](#settings)
- [Built-in and custom content](#built-in-and-custom-content)
- [Choosing the right page](#choosing-the-right-page)

## Catalog ownership

| Goal                                                 | Page              |
| ---------------------------------------------------- | ----------------- |
| Reuse a whole workflow setup                         | Assistant Presets |
| Create reusable workflow modes or skill-based drafts | Skills            |
| Maintain callable capabilities                       | Tools             |
| Create or maintain MCP server catalogs               | MCP Servers       |
| Configure providers and models                       | Model Presets     |
| Add auth keys, change theme, or debug settings       | Settings          |

## Assistant Presets

An assistant preset is a reusable starter setup for a type of work.

Examples:

- feature developer using spec driven development
- code reviewer
- local reader
- documentation writer
- architecture reviewer
- bug investigator
- tool-assisted developer
- research brief writer

Assistant presets are starters, not locked modes. Applying one can seed the current chat, but you can still change:

- current draft text
- model
- model parameters
- tools
- web search
- skills
- attachments
- previous user turns

### What an assistant preset can contain

| Preset field         | User-facing effect                                                                                                                                  |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Starting text        | Seeds the composer/editor with an initial draft for the workflow.                                                                                   |
| Starting model       | Selects a provider/model preset in Chats.                                                                                                           |
| Starting model patch | Overrides runtime knobs such as streaming, token limits, temperature, reasoning, output settings, timeout, stop sequences, and raw JSON parameters. |
| Tool selections      | Adds conversation tools and compatible web-search choices.                                                                                          |
| Skill selections     | Enables skills and optionally preloads some as active. These can include template-style skills or instruction-only skills.                          |

Assistant preset model patches intentionally cannot set skill definitions directly. Skill behavior belongs to the Skills page.

### Empty preset sections

Assistant presets are partial recipes.

An empty section usually means:

> This preset has no opinion about this section.

Examples:

- no tool selections means applying the preset does not necessarily clear current tools
- no skill selections means applying the preset does not necessarily clear current skills
- no starting model means applying the preset does not force a model
- no starting text means applying the preset does not seed a composer draft

A preset can be only a skill setup, only a model setup, only a tool setup, or a full workflow.

### Inspecting a preset

In **Chats**:

1. Open the **Assistant** dropdown.
2. Find the preset.
3. Use **View**.

This is best for understanding the current conversation state and modified sections.

On **Assistant Presets**:

1. Open **Assistant Presets**.
2. Expand a bundle.
3. Review counts for model, skills, tools, and starting text.
4. Use **View** on a preset.

This is best for catalog maintenance.

### Modified state in Chats

The assistant dropdown can show:

- **In sync**
  - current preset-managed sections still match what was applied
- **Modified**
  - at least one preset-managed section changed

Modified sections can include:

- `Model`
- `Skills`
- `Tools`
- `Starting text`

Use **Reapply** or **Reset** to restore preset-managed sections. Use **Clear to base** to return to the base assistant preset or fallback selectable preset.

### Versioning and built-ins

Assistant presets live inside bundles.

Rules:

- built-in bundles are read-only except enable/disable state
- built-in presets cannot be edited or deleted
- custom bundles can contain custom presets
- editing a custom preset creates a new version
- a preset slug identifies a version series
- the version must be unique within that slug
- custom presets can be deleted
- empty custom bundles can be deleted

## Tools

The **Tools** page manages tool definitions. Runtime execution happens in Chats.

From a user perspective:

- the Tools page defines what can be selected
- the Chats composer decides which tools are available to a conversation or message
- the model can then request tool calls
- tool calls can be run manually or auto-executed if configured

Safety expectations:

- start with manual review
- inspect tool arguments before running
- keep auto-execute off for risky tools
- inspect outputs before sending them back
- remove tools not needed for the current workflow

Some tool types are built-in or provider-bound. User-created tools may not cover every implementation type. Use the Tools page UI as the source of truth for current create/edit fields.

## Skills

The **Skills** page manages skill bundles and skills.

Important behavior:

- built-in embedded skills are generally read-only
- user-created skills are filesystem skills
- skill names must be unique within a bundle
- skill refs include a skill ID to avoid stale identity confusion
- skills can be enabled or disabled
- skills can show presence status such as present, missing, error, or unknown

Use skills when you want a reusable workflow mode, a template-style draft starter, or instruction-only behavior that shapes the request context.

Use assistant presets to preload skills for common workflows.

## Model Presets

The **Model Presets** page owns provider and model setup.

It controls:

- default provider
- provider enablement
- model preset enablement
- provider SDK/API compatibility type
- provider origin and path
- API key header name
- default headers
- model parameters and capability overrides
- default model per provider

Use Model Presets when changing how requests run. Use Assistant Presets when changing the kind of workflow you want to start from.

For local and self-hosted LLMs, the recommended customization order is:

1. copy/fork an existing provider preset
2. adjust provider-level settings
3. then copy, add, or edit model presets under that provider

Do provider first because the provider owns the shared endpoint contract:

- SDK/API compatibility type
- origin URL
- chat path
- API-key header name
- default headers
- provider-wide capability assumptions

Built-in local providers such as LocalAI, LM Studio, `llama.cpp`, Ollama, SGLang, and vLLM are useful defaults, but local server ports, paths, headers, model names, and feature support vary. Use **Add Provider -> Prefill from Existing -> Copy Existing Provider** to create your local fork instead of trying to edit a read-only built-in. Then use **Add Model Preset -> Copy Existing Preset** if an existing model preset is close to the model your server exposes.

See [Local LLM Setup](/docs?doc=local-llm-setup) for a full local runtime setup flow.

## Settings

The **Settings** page owns app-wide settings:

- theme
- auth keys
- debug settings
- settings export

Provider secrets are stored through the OS keyring. Settings export does not expose raw provider secret values.

Be careful with debug options. Raw request/response logging can include prompts, attachments, tool outputs, and sensitive provider responses.

## Built-in and custom content

Across model presets, tools, skills, and assistant presets:

- built-in content ships with the app
- custom content is stored locally
- built-in definitions are generally read-only
- built-in items can usually be enabled or disabled
- custom entries can usually be edited/deleted according to that page’s rules

Practical workflow:

1. start from built-in content
2. inspect what it does
3. create custom versions when you need durable changes
4. keep built-ins enabled only if useful

## Choosing the right page

| If you want to...                                    | Go to...                              |
| ---------------------------------------------------- | ------------------------------------- |
| Start a reusable workflow                            | Chats assistant dropdown              |
| Inspect the active assistant preset                  | Chats assistant dropdown -> View      |
| Create or version an assistant preset                | Assistant Presets                     |
| Create reusable skill-based drafts or behavior rules | Skills                                |
| Create or maintain tool definitions                  | Tools                                 |
| Enable a tool in a chat                              | Chats -> Tools or assistant preset    |
| Create or maintain skill definitions                 | Skills                                |
| Create or maintain MCP server catalogs               | MCP Servers                           |
| Select MCP server context for a turn                 | Chats -> Composer -> MCP              |
| Enable skills in a chat                              | Chats -> Skills or assistant preset   |
| Add an API key                                       | Settings                              |
| Add a local/custom provider                          | Model Presets                         |
| Compare models                                       | Chats, changing only the model preset |
