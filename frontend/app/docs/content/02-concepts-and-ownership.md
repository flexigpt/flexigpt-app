# Concepts and Ownership

FlexiGPT is not only a chat box. It is a local-first workspace where a request is assembled from a model, instructions, current message, selected history, source material, and optional execution capabilities.

This page gives the vocabulary and explains which page owns each part. Detailed workflows live on later pages.

## Table of contents <!-- omit from toc -->

- [Mental model](#mental-model)
- [Main terms](#main-terms)
- [Assistant preset versus model preset](#assistant-preset-versus-model-preset)
- [Prompt types](#prompt-types)
- [Context and execution](#context-and-execution)
- [History](#history)
- [Built-in content and your content](#built-in-content-and-your-content)
- [Page ownership map](#page-ownership-map)
- [Decision guide](#decision-guide)

## Mental model

A chat turn is assembled in layers:

1. **Provider**
   - The API family or endpoint that receives the request.
2. **Model preset**
   - The provider/model choice and request defaults.
3. **Assistant setup**
   - Optional starter workflow: starting text, model, instructions, tools, and skills.
4. **Conversation history**
   - Earlier turns included by **Previous user turns**.
5. **Current message**
   - Draft text, prompt template values, attachments, tool outputs, and active composer context.

When a result changes, compare these layers one at a time.

## Main terms

| Term                         | Meaning                                                                                                                                                                            |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Provider**                 | The API family or endpoint FlexiGPT talks to, such as OpenAI, Anthropic, Google Gemini API, xAI, Mistral, Hugging Face, OpenRouter, `llama.cpp`, or a compatible custom endpoint.  |
| **Model preset**             | A saved provider/model choice with defaults such as model name, streaming, timeout, prompt/output limits, temperature, reasoning, output format, and provider-specific parameters. |
| **Assistant preset**         | A reusable starter setup that can apply starting text, a model preset, instruction templates, tool selections, web search choices, and skill selections.                           |
| **System prompt**            | Durable instruction context that shapes assistant behavior.                                                                                                                        |
| **Prompt template**          | Reusable request structure inserted into the current draft. It may contain variables and multiple role blocks.                                                                     |
| **Attachment**               | Message-scoped source material such as files, folders, images, PDFs, or URLs.                                                                                                      |
| **Tool**                     | A callable capability the model can request during a conversation.                                                                                                                 |
| **Skill**                    | A reusable workflow mode backed by skill runtime/session behavior.                                                                                                                 |
| **MCP server**               | One configured MCP (Model context protocol) endpoint inside a bundle, including transport, auth, trust, setup, discovery, and runtime state.                                       |
| **MCP conversation context** | The selected MCP servers, tools, resources, resource templates, prompts, and arguments attached to the next request.                                                               |
| **Previous user turns**      | The history window for the next request.                                                                                                                                           |

## Assistant preset versus model preset

Use a **model preset** when the question is:

> Which provider, model, and request parameters should run this turn?

Use an **assistant preset** when the question is:

> What kind of workflow should I start from?

An assistant preset is a starter recipe, not a locked mode. After applying one, you can still change the model, instructions, tools, skills, attachments, web search, prompt templates, and history setting. If the preset defines starting text, treat it as a replaceable first draft rather than a locked prompt.

The detailed rules for assistant preset contents, empty sections, modified state, inspection, and versioning live in [Reusable Catalogs](/docs?doc=reusable-catalogs#assistant-presets).

## Prompt types

FlexiGPT uses two related prompt ideas:

| Feature             | Use it when                                            |
| ------------------- | ------------------------------------------------------ |
| **System prompt**   | You want durable behavior instructions across turns.   |
| **Prompt template** | You want a reusable structure for the current message. |

Prompt templates have a derived kind:

- **Instructions Only**
  - all blocks are `system` or `developer`
  - can be selected as saved system prompt sources
  - can be referenced by assistant presets
- **Generic**
  - contains at least one `user` block
  - appears in the composer prompt picker
  - inserts reusable request text into the current draft

## Context and execution

Use the right mechanism for the job:

| Need                                             | Use                            |
| ------------------------------------------------ | ------------------------------ |
| Exact source material                            | Attachment                     |
| Durable behavior rules                           | System prompt                  |
| Repeatable current-message structure             | Prompt template                |
| Let the model ask the app to run something       | Tool                           |
| Use context from a model context protocol server | MCP                            |
| Keep a workflow mode active across turns         | Skill                          |
| Recent web information                           | Provider-compatible web search |

See [Composer Context](/docs?doc=composer-context) for how these are used while composing a message.

## History

**Previous user turns** controls how much earlier user context is resent.

Important behavior:

- `all` sends all previous messages.
- numeric `N` includes the current user turn plus `N` previous pure user turns.
- a pure user turn is a user message without tool outputs.
- system/developer instruction messages before the selected window are preserved.
- attachments on included older user turns may be included again.

If a conversation drifts, reduce **Previous user turns** before changing everything else.

## Built-in content and your content

FlexiGPT ships with built-in:

- providers
- model presets
- prompt templates
- tools
- skills
- MCP server catalogs
- assistant presets
- docs

Built-in content is generally read-only. You can usually enable or disable it, but not edit its definition directly.

Your local content is stored locally and can be created, edited, deleted, and versioned depending on the page.

For versioned domains such as prompts and assistant presets:

- editing a custom item creates a new version
- slugs are stable within a version series
- built-in items cannot be edited into new custom versions directly unless you create or fork local content where the UI supports it

## Page ownership map

| Goal                                                                                     | Page                                       |
| ---------------------------------------------------------------------------------------- | ------------------------------------------ |
| Do active work with a model                                                              | **Chats**                                  |
| Attach files, folders, URLs, tools, skills, prompt templates, or web search to a message | **Chats -> Composer**                      |
| Select MCP server context for a turn                                                     | **Chats -> MCP**                           |
| Start from a reusable assistant workflow                                                 | **Chats -> Assistant dropdown**            |
| Create or version assistant presets                                                      | **Assistant Presets**                      |
| Create or version prompt templates and saved system prompts                              | **Prompts**                                |
| Create or maintain tool definitions                                                      | **Tools**                                  |
| Enable a tool for a conversation                                                         | **Chats -> Tools** or an assistant preset  |
| Create or maintain skill definitions                                                     | **Skills**                                 |
| Enable skills for a conversation                                                         | **Chats -> Skills** or an assistant preset |
| Create or maintain MCP server catalogs                                                   | **MCP Servers**                            |
| Configure providers and model presets                                                    | **Model Presets**                          |
| Add provider keys, theme, and debug settings                                             | **Settings**                               |
| Search and reopen old conversations                                                      | **Chats**                                  |

## Decision guide

| If you want to...                   | Change...                          |
| ----------------------------------- | ---------------------------------- |
| Compare model quality               | only the model preset              |
| Make answers follow a durable style | system prompt source               |
| Reuse a task format                 | prompt template                    |
| Bring exact source material         | attachments                        |
| Give the model execution ability    | tools                              |
| Use a structured workflow mode      | skills                             |
| Rebuild the same setup often        | assistant preset                   |
| Avoid stale context                 | Previous user turns                |
| Keep work local-only                | provider endpoint and tool choices |
