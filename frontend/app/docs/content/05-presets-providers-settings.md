# Presets, Providers, and Settings

The pages outside **Chats** manage reusable building blocks: assistant presets, prompts, tools, skills, model presets, and app settings.

This page explains what each page owns, how assistant presets behave, and how to inspect or modify the setup behind a chat workflow.

## Table of contents <!-- omit from toc -->

- [Page ownership](#page-ownership)
- [Assistant Presets](#assistant-presets)
  - [Assistant presets are starters](#assistant-presets-are-starters)
  - [What an assistant preset can contain](#what-an-assistant-preset-can-contain)
  - [What empty preset sections mean](#what-empty-preset-sections-mean)
  - [How to inspect what a preset supplies](#how-to-inspect-what-a-preset-supplies)
    - [In Chats](#in-chats)
    - [On the Assistant Presets page](#on-the-assistant-presets-page)
  - [How to modify preset-supplied prompts](#how-to-modify-preset-supplied-prompts)
    - [Modify for this conversation only](#modify-for-this-conversation-only)
    - [Modify a reusable prompt source](#modify-a-reusable-prompt-source)
    - [Modify the assistant preset itself](#modify-the-assistant-preset-itself)
  - [Create your first assistant preset](#create-your-first-assistant-preset)
  - [Assistant preset versioning and built-ins](#assistant-preset-versioning-and-built-ins)
- [Prompts](#prompts)
- [Tools](#tools)
- [Skills](#skills)
- [Model Presets](#model-presets)
- [Settings](#settings)
- [Built-in content and custom content](#built-in-content-and-custom-content)
- [How to decide which page to use](#how-to-decide-which-page-to-use)

## Page ownership

| Goal                                                | Page              |
| --------------------------------------------------- | ----------------- |
| Reuse a whole workflow setup                        | Assistant Presets |
| Create reusable prompt structures or system prompts | Prompts           |
| Maintain callable capabilities                      | Tools             |
| Maintain reusable workflow modes                    | Skills            |
| Configure providers and models                      | Model Presets     |
| Add auth keys, change theme, or debug settings      | Settings          |

## Assistant Presets

An assistant preset is a reusable starter setup for a type of work.

Examples:

- code reviewer
- local reader
- documentation writer
- architecture reviewer
- bug investigator
- tool-assisted developer
- research brief writer

### Assistant presets are starters

Assistant presets are not locked modes.

Applying an assistant preset can seed the current chat, but you can still change:

- model
- model parameters
- system prompt sources
- prompt templates
- tools
- web search
- skills
- attachments
- previous user turns

In Chats, the assistant dropdown can show whether the active preset is:

- **In sync**
  - current preset-managed sections still match what was applied
- **Modified**
  - at least one preset-managed section has changed

Modified sections can include:

- `Model`
- `Instructions`
- `Tools`
- `Skills`

Use **Reapply** or **Reset** to restore preset-managed sections.
Use **Clear to base** to return to the base assistant preset or fallback selectable preset.

### What an assistant preset can contain

An assistant preset can define:

| Preset field                | User-facing effect                                                                                                                               |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Starting model preset       | Selects a provider/model preset in Chats.                                                                                                        |
| Starting model patch        | Overrides runtime knobs such as stream, token limits, temperature, reasoning, output settings, timeout, stop sequences, and raw JSON parameters. |
| Include model system prompt | Controls whether the selected model’s default system prompt is included.                                                                         |
| Instruction template refs   | Selects saved instructions-only system/developer prompt templates.                                                                               |
| Tool selections             | Adds conversation tools and compatible web-search choices.                                                                                       |
| Skill selections            | Enables skills and optionally preloads some as active.                                                                                           |

Assistant preset model patches intentionally cannot set:

- `systemPrompt`
- `capabilitiesOverride`

Those belong to model/prompt/provider configuration, not assistant preset runtime patching.

### What empty preset sections mean

Assistant presets are partial starter recipes.

An empty section usually means:

> This preset has no opinion about this section.

Examples:

- no tool selections means applying the preset does not necessarily clear current tools
- no skill selections means applying the preset does not necessarily clear current skills
- no instruction template refs means applying the preset does not replace prompt sources
- no starting model means applying the preset does not force a model

This behavior lets presets be lightweight.
A preset can be only a prompt setup, only a model setup, only a tool setup, or a full workflow.

### How to inspect what a preset supplies

There are two useful inspection places.

#### In Chats

1. Open **Chats**.
2. Open the **Assistant** dropdown.
3. Find the preset.
4. Click the eye/view action.

The view can show:

- **Model and advanced params**
  - model ref and runtime patch values
- **Instruction templates**
  - selected system/developer instruction templates
- **Tools and web search**
  - tool selections, web-search selections, auto-execute state, and saved args marker
- **Enabled skills**
  - skill selections and preload-as-active state

If viewing the active preset, the modal can also compare preset-applied values with current values.
This is the best way to understand why a preset shows **Modified**.

#### On the Assistant Presets page

1. Open **Assistant Presets**.
2. Expand a bundle.
3. Review counts for model, instructions, tools, and skills.
4. Use **View** on a preset.
5. Check metadata and selected refs.

The page is useful for catalog maintenance.
The Chats view is better for understanding the current conversation state.

### How to modify preset-supplied prompts

There are several ways to modify prompts, depending on what you want.

#### Modify for this conversation only

In Chats:

1. Open the **System prompt** menu.
2. Toggle the model default prompt.
3. Select or clear saved system prompts.
4. Add or fork a system prompt if needed.

This changes the current conversation setup.

#### Modify a reusable prompt source

Use **Prompts**:

1. Open **Prompts**.
2. Expand the relevant prompt bundle.
3. View the template used by the assistant preset.
4. For custom templates, create a new version.
5. Update the assistant preset to point to the new template version if needed.

Built-in prompt templates are read-only except for enable/disable state.

#### Modify the assistant preset itself

Use **Assistant Presets**:

1. Open **Assistant Presets**.
2. Create or select a custom bundle.
3. Add a preset or create a new version of an existing custom preset.
4. Select eligible instruction templates.
5. Save.
6. Apply the preset in Chats.

Assistant presets can only select enabled, resolved, instructions-only prompt templates as instruction refs.

### Create your first assistant preset

A simple first assistant preset should be small.

Recommended first example: a documentation reviewer.

1. Open **Assistant Presets**.
2. If needed, click **Add Bundle**.
3. Use a slug like `my-assistants`.
4. Expand the custom bundle.
5. Click **Add Assistant Preset**.
6. Fill:
   - display name: `Docs Reviewer`
   - slug: `docs-reviewer`
   - version: `v1.0.0`
   - enabled: on
7. Choose a starting model preset.
8. Decide whether to include the model system prompt.
9. Add one or two resolved instruction templates if available.
10. Leave tools and skills empty for the first version.
11. Save.
12. Open **Chats** and apply the preset.
13. Use **View** in the assistant dropdown to confirm what it supplied.

Then test with:

```text
Review the attached documentation for clarity, missing steps, unsupported claims, and reader expectations.
Return prioritized fixes.
```

Once the basic preset works, create a new version with tools or skills.

### Assistant preset versioning and built-ins

Assistant presets live inside bundles.

Important rules:

- built-in bundles are read-only except enable/disable state
- built-in presets cannot be edited or deleted
- custom bundles can contain custom presets
- editing a custom preset creates a new version
- a preset slug identifies a version series
- the version must be unique within that slug
- custom presets can be deleted
- empty custom bundles can be deleted

When creating a new version, the UI suggests the next minor version.

## Prompts

The **Prompts** page manages prompt bundles and template versions.

Use it to create:

- generic prompt templates
- instructions-only system/developer prompts
- reusable variables
- reusable prompt versions

Important behavior:

- template kind is derived from block roles
- resolved state is derived from placeholders and variable defaults/static values
- existing versions are immutable in practice; editing creates a new version
- built-in templates are read-only except enable/disable state
- custom prompt bundles can be added
- empty custom prompt bundles can be deleted

Use the Prompts page when you need full variable support.
Use Add/Fork in the Chats system prompt menu only for simple resolved system prompt text.

## Tools

The **Tools** page manages tool definitions.
Runtime execution happens in Chats.

From a user perspective:

- the Tools page defines what can be selected
- the Chats composer decides which tools are available to a conversation
- the model can then request tool calls
- tool calls can be run manually or auto-executed if configured

Tool-related safety expectations:

- start with manual review
- inspect tool arguments before running
- keep auto-execute off for risky tools
- inspect outputs before sending them back
- remove tools not needed for the current workflow

Some tool types are built-in or provider-bound.
User-created tools are not necessarily allowed to cover every implementation type.
For exact current create/edit fields, use the Tools page UI.

## Skills

The **Skills** page manages skill bundles and skills.

Important behavior:

- built-in embedded skills are generally read-only
- user-created skills are filesystem skills
- skill names must be unique within a bundle
- skill refs include a skill ID to avoid stale identity confusion
- skills can be enabled or disabled
- skills can show presence status such as present, missing, error, or unknown

Use skills when you want a reusable workflow mode that can participate across turns.

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

Use Model Presets when changing how requests run.

Use Assistant Presets when changing the kind of workflow you want to start from.

## Settings

The **Settings** page owns app-wide settings:

- theme
- auth keys
- debug settings
- settings export

Provider secrets are stored through the OS keyring.
Settings export does not expose raw provider secret values.

Be careful with debug options.
Raw request/response logging can include prompts, attachments, tool outputs, and sensitive provider responses.

## Built-in content and custom content

Across model presets, prompts, tools, skills, and assistant presets:

- built-in content ships with the app
- custom content is stored locally
- built-in definitions are generally read-only
- built-in items can usually be enabled or disabled
- custom entries can usually be edited/deleted according to that page’s rules

A practical workflow is:

1. start from built-in content
2. inspect what it does
3. create custom versions when you need durable changes
4. keep built-ins enabled only if useful

## How to decide which page to use

| If you want to...                         | Go to...                              |
| ----------------------------------------- | ------------------------------------- |
| Start a reusable workflow                 | Chats assistant dropdown              |
| Inspect the active assistant preset       | Chats assistant dropdown -> View      |
| Create or version an assistant preset     | Assistant Presets                     |
| Create reusable prompt text               | Prompts                               |
| Quickly add/fork a resolved system prompt | Chats -> System prompt menu           |
| Create or maintain tool definitions       | Tools                                 |
| Enable a tool in a chat                   | Chats -> Tools or assistant preset    |
| Create or maintain skill definitions      | Skills                                |
| Enable skills in a chat                   | Chats -> Skills or assistant preset   |
| Add an API key                            | Settings                              |
| Add a local/custom provider               | Model Presets                         |
| Compare models                            | Chats, changing only the model preset |
