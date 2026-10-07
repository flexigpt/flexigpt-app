# Concepts and Ownership

FlexiGPT is not only a chat box. It is a local-first workspace where a request is assembled from a model, instructions, current message, selected history, source material, and optional execution capabilities.

This page gives the vocabulary and explains which page owns each part. Detailed workflows live on later pages.

## Table of contents <!-- omit from toc -->

- [Mental model](#mental-model)
- [Main terms](#main-terms)
- [Agent versus model](#agent-versus-model)
- [Context and execution](#context-and-execution)
- [History](#history)
- [Built-in content and your content](#built-in-content-and-your-content)
- [Page ownership map](#page-ownership-map)
- [Decision guide](#decision-guide)

## Mental model

A chat turn is assembled in layers:

1. **Provider**
   - The API family or endpoint that receives the request.
2. **Model**
   - The provider/model choice and request defaults.
3. **Agent setup**
   - Optional starter workflow: model, instructions, opening text, tools, Skills, and connected services.
4. **Conversation history**
   - Earlier turns included by **Previous user turns**.
5. **Current message**
   - Draft text, attachments, tool outputs, and active composer context.

When a result changes, compare these layers one at a time.

## Main terms

| Term                         | Meaning                                                                                                                                                                                            |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Provider**                 | The API family or endpoint FlexiGPT talks to, such as OpenAI, Anthropic, Google Gemini API, xAI, Mistral, Hugging Face, OpenRouter, a local runtime preset, or a compatible custom endpoint.       |
| **Model**                    | A saved provider/model configuration with defaults such as model name, streaming, timeout, prompt/output limits, temperature, reasoning, output format, and provider-specific parameters.          |
| **Agent**                    | A reusable starting setup that can apply a model choice, instructions, opening text, tools, Skills, and connected services.                                                                        |
| **Instruction context**      | Durable system instruction context built from the selected model default and instruction-only skill sources. The model default comes first; selected sources are then combined in selection order. |
| **Attachment**               | Message-scoped source material such as files, folders, images, PDFs, or URLs.                                                                                                                      |
| **Tool**                     | A built-in Go capability the model can request during a conversation.                                                                                                                              |
| **Skill**                    | A reusable workflow mode, including template-style skills and instruction-only skills that can seed drafts or shape behavior.                                                                      |
| **MCP server**               | One configured MCP (Model context protocol) endpoint inside a plugin, including transport, auth, trust, setup, discovery, and runtime state.                                                       |
| **MCP conversation context** | The selected MCP servers, tools, resources, resource templates, prompts, and arguments attached to the next request.                                                                               |
| **Previous user turns**      | The history window for the next request.                                                                                                                                                           |

## Agent versus model

Use a **model** when the question is:

> Which provider, model, and request parameters should run this turn?

Use an **Agent** when the question is:

> What kind of workflow should I start from?

An Agent is a starter recipe, not a locked mode. After loading one, you can still change the model, instructions, tools, Skills, attachments, web search, and history setting. Treat opening text as an editable first draft rather than a locked prompt.

The detailed rules for Agent contents, importing, exporting, inspection, and management live in [Agents](/docs?doc=agents#what-an-agent-can-set-up).

## Context and execution

Use the right mechanism for the job:

| Need                                             | Use                            |
| ------------------------------------------------ | ------------------------------ |
| Exact source material                            | Attachment                     |
| Durable behavior rules                           | Instruction-only skill         |
| Repeatable current-message structure             | Template-style skill           |
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
- models
- tools
- skills
- MCP server catalogs
- Agents
- docs

Built-in content is generally read-only. You can usually enable or disable it, but not edit its definition directly.
For local LLM providers, treat built-ins as starting points: copy/fork the provider first, adjust the endpoint and compatibility settings, then copy or add models under that provider.

Your local content is stored locally and can be created, edited, deleted, and versioned depending on the page.

To customize an Agent, export it, edit the file in your editor, and import it under a new name, or replace the managed copy.
See [Agents](/docs?doc=agents#manage-agents) for the management flow.

## Page ownership map

| Goal                                                                   | Page                            |
| ---------------------------------------------------------------------- | ------------------------------- |
| Do active work with a model                                            | **Chats**                       |
| Attach files, folders, URLs, tools, skills, or web search to a message | **Chats -> Composer**           |
| Select MCP server context for a turn                                   | **Chats -> MCP**                |
| Start from a reusable Agent workflow                                   | **Chats -> Agent menu**         |
| Import, export, enable, or manage Agents                               | **Agents**                      |
| Browse available built-in Go tools                                     | **Tools**                       |
| Enable a tool for a conversation                                       | **Chats -> Tools** or an Agent  |
| Create or maintain skill definitions                                   | **Skills**                      |
| Enable skills for a conversation                                       | **Chats -> Skills** or an Agent |
| Create or maintain MCP server catalogs                                 | **MCP Servers**                 |
| Configure providers, models, and auth keys                             | **Models**                      |
| Change theme or logging/debug options                                  | Title-bar **Settings** menu     |
| Search and reopen old conversations                                    | **Chats**                       |

## Decision guide

| If you want to...                   | Change...                          |
| ----------------------------------- | ---------------------------------- |
| Compare model quality               | only the selected model            |
| Make answers follow a durable style | instruction-only skill source      |
| Bring exact source material         | attachments                        |
| Give the model execution ability    | tools                              |
| Use a structured workflow mode      | skills                             |
| Rebuild the same setup often        | Agent                              |
| Avoid stale context                 | Previous user turns                |
| Keep work local-only                | provider endpoint and tool choices |
