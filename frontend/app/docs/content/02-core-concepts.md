# Core Concepts

FlexiGPT is a local-first BYOK AI workspace, not just a chat box.
It helps you build repeatable workflows by combining a model, assistant preset, prompt sources, attachments, tools, skills, and the current message.

This page explains the main concepts from a user perspective: what each part does, when to use it, and what to expect when it changes.

## Table of contents <!-- omit from toc -->

- [The main terms](#the-main-terms)
- [Think in layers](#think-in-layers)
- [Assistant preset versus model preset](#assistant-preset-versus-model-preset)
  - [Assistant presets are starter recipes](#assistant-presets-are-starter-recipes)
- [System prompts and prompt templates](#system-prompts-and-prompt-templates)
  - [Prompt template kinds and variables](#prompt-template-kinds-and-variables)
- [Attachments](#attachments)
- [Tools](#tools)
- [Skills](#skills)
- [Previous user turns controls history](#previous-user-turns-controls-history)
- [Built-in content and your local content](#built-in-content-and-your-local-content)
- [Quick decision guide](#quick-decision-guide)

## The main terms

| Term                    | Meaning                                                                                                                                                                                |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Provider**            | The API family or endpoint FlexiGPT talks to, such as OpenAI, Anthropic, Google Gemini API, xAI, Mistral, Hugging Face, OpenRouter, `llama.cpp`, or a compatible custom endpoint.      |
| **Model Preset**        | A saved provider/model choice with defaults such as streaming, timeout, prompt/output limits, temperature, reasoning, output format, cache controls, and provider-specific parameters. |
| **Assistant Preset**    | A reusable starter setup that can seed the chat with a model preset, instruction templates, tool selections, and skill selections.                                                     |
| **System Prompt**       | Durable instruction context that shapes how the assistant behaves.                                                                                                                     |
| **Prompt Template**     | A reusable request structure inserted into the composer. It can contain variables and multiple role blocks.                                                                            |
| **Previous user turns** | The history window for the next request. It controls how many earlier pure user turns are resent.                                                                                      |
| **Attachment**          | Message-scoped source material such as files, folders, images, PDFs, and URLs.                                                                                                         |
| **Tool**                | A callable capability the model can request during a conversation.                                                                                                                     |
| **Skill**               | A reusable workflow mode backed by a skill runtime/session model.                                                                                                                      |

## Think in layers

A useful FlexiGPT workflow has these layers:

1. **Provider and transport**
   - Which provider or compatible endpoint receives the request.
2. **Model and request settings**
   - Which model preset is active and which runtime parameters apply.
3. **Assistant setup**
   - Which assistant preset, system prompt sources, tools, and skills shape the conversation.
4. **Current turn**
   - The draft text, prompt template variables, attachments, tool outputs, and selected history for this send.

If a result changes, usually one of these layers changed.

## Assistant preset versus model preset

Use a **model preset** when the question is:

> Which model and request settings should run this turn?

A model preset controls things like:

- provider
- model name
- streaming
- timeout
- prompt/output limits
- temperature or reasoning settings
- output format
- stop sequences
- cache controls
- provider-specific raw JSON parameters

Use an **assistant preset** when the question is:

> What kind of workspace should I start from?

An assistant preset can seed:

- a starting model preset
- selected instruction templates
- whether to include the model default system prompt
- tool selections
- web-search selection when compatible
- skill selections

### Assistant presets are starter recipes

Assistant presets are not locked modes.

They are starter recipes that can prefill parts of the composer and context bar.
After applying one, you can still change:

- model
- temperature, reasoning, verbosity, and advanced parameters
- system prompt sources
- prompt templates
- attachments
- tools
- web search
- skills
- previous user turns

A preset may manage only some sections.

For example:

- a preset with only a model choice changes the model, but leaves tools and skills alone
- a preset with instruction templates changes selected instruction sources
- a preset with tool selections sets conversation tools and web search choices
- a preset with skill selections enables those skills and may preload some as active
- an empty section generally means “no opinion”, not “clear the current state”

In Chats, the assistant preset dropdown shows whether the active preset is **In sync** or **Modified**.
If you change a preset-managed section after applying the preset, the dropdown can show which sections changed, such as `Model`, `Instructions`, `Tools`, or `Skills`.

## System prompts and prompt templates

System prompts and prompt templates solve different problems.

| Feature                                    | Use it when                                                             |
| ------------------------------------------ | ----------------------------------------------------------------------- |
| **System prompt**                          | You want durable behavior instructions across turns.                    |
| **Prompt template**                        | You want a reusable request shape for the current message.              |
| **Assistant preset instruction templates** | You want a preset to select reusable instruction prompts automatically. |

The effective system prompt can be assembled from:

- the selected model preset’s default system prompt, when included
- selected saved system/developer prompts
- system/developer blocks from inserted prompt templates
- a synthetic “previous conversation prompt” when restoring older conversations

Saved system prompt sources are concatenated in order:
first the model default if enabled, then selected saved prompts.

### Prompt template kinds and variables

Prompt templates have a derived kind:

- **Instructions Only**
  - all blocks are `system` or `developer`
  - can be used as reusable system prompt sources
  - assistant presets can reference only enabled, resolved instructions-only templates
- **Generic**
  - contains at least one `user` block
  - appears in the composer prompt picker
  - inserts reusable request text into the current draft

Prompt templates can contain variables like `{{topic}}`.

Variables can be:

- string
- number
- boolean
- enum
- date

Variables can come from:

- **User** input
- **Static** values

A template is **resolved** when every placeholder can be filled locally by a static value or default.
If required variables are missing in the composer, sending is blocked until they are filled.

## Attachments

Attachments belong to messages.

That means:

- an attachment added to the current draft is attached to the message you send
- older attachments return only when their older user turn is included by the history setting
- deleting an attachment from the current draft does not rewrite older messages
- a restored older message may show attachments as flat chips rather than reconstructing the original folder grouping

Attachments can have different content modes, such as:

- text extracted into the message
- binary file attachment if provider-supported
- image attachment if provider-supported
- page content extracted from a URL
- link as text
- URL as image/file reference
- not readable

Review attachment mode before sending sensitive or large context.

## Tools

Tool use has three separate states:

1. **Tool choice**
   - You make a tool available to the conversation or current message.
2. **Tool call**
   - The model proposes a concrete call with arguments.
3. **Tool output**
   - FlexiGPT runs the call and captures the result.

These are not the same thing.

A tool choice can exist before the model calls it.
A tool call can be pending, running, failed, discarded, or converted into an output.
A tool output can be inspected, removed, retried when possible, or sent back into the conversation.

Tool choices can be:

- **conversation-level tools**
  - persist across turns until changed
- **per-message attached tools**
  - attached to the current draft
- **web search**
  - provider/SDK-bound and handled separately

## Skills

Skills are reusable workflow modes.
They are not just saved text.

In the composer:

- **enabled skills** are allowed in the current workflow
- **active skills** are currently active inside the skill session
- a skill session may be created or refreshed when needed
- enabled/active skill refs can be saved with messages and restored later

Assistant presets can enable skills and mark some as **preload as active**.

## Previous user turns controls history

The **Previous user turns** setting controls how much earlier user context is resent.

Important behavior:

- `all` sends all previous messages
- numeric `N` includes the current user turn plus `N` previous pure user turns
- a pure user turn is a user message without tool outputs
- system/developer instruction messages before the selected window are preserved
- attachments on included older user turns may be included again

If a conversation drifts, reduce this setting before changing everything else.

## Built-in content and your local content

FlexiGPT ships with built-in:

- providers
- model presets
- prompt templates
- tools
- skills
- assistant presets

Built-in content is generally read-only.
You can usually enable or disable it, but not edit its definition directly.

Your local content is stored locally and can be created, edited, deleted, and versioned depending on the page.

For versioned domains such as prompts and assistant presets:

- editing an existing custom item creates a new version
- slugs are stable within a version series
- built-in items cannot be edited into new versions directly

## Quick decision guide

| Goal                                      | Use                                           |
| ----------------------------------------- | --------------------------------------------- |
| Choose the provider/model                 | Model preset                                  |
| Start a known workflow                    | Assistant preset                              |
| Keep behavior rules active                | System prompt                                 |
| Reuse a current-message request structure | Prompt template                               |
| Add source material                       | Attachment                                    |
| Let the model request execution           | Tool                                          |
| Keep a workflow mode active across turns  | Skill                                         |
| Control stale context                     | Previous user turns                           |
| Compare model quality                     | Keep everything fixed except the model preset |
