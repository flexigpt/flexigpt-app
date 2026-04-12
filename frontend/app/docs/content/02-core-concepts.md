# Core Concepts

The app becomes much easier to use once a few core terms and concepts are clear.

## The main terms

| Term                    | Role in FlexiGPT                                                                                                                                                                                                                                                                |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Provider**            | The API family or endpoint the app talks to, such as OpenAI, Anthropic, Google Gemini, OpenRouter, `llama.cpp`, or a compatible custom endpoint.                                                                                                                                |
| **Model Preset**        | A saved provider-plus-model choice with request defaults such as streaming, timeout, token limits, temperature, reasoning, output settings, and advanced parameters.                                                                                                            |
| **Assistant Preset**    | A higher-level starting setup that can preload a model preset, instruction templates, tools, and skills.                                                                                                                                                                        |
| **System Prompts**      | The instructions that shape how the model should behave for the request.                                                                                                                                                                                                        |
| **Prompt Template**     | Reusable prompt content that can be inserted into the current message flow.                                                                                                                                                                                                     |
| **Previous user turns** | The number of earlier user turns that should be resent with the next request.                                                                                                                                                                                                   |
| **Attachment**          | Extra context attached to a message, such as a file, folder-derived files, image, PDF, or URL.                                                                                                                                                                                  |
| **Tool**                | A callable capability that can be offered to the model. Some run through local tool runtime, while some provider-side capabilities, such as web search, depend on the current provider family. They can be set to autoexecute and submit, or manual review, execute and submit. |
| **Skill**               | A reusable workflow frame managed through the skills runtime and attached to a conversation.                                                                                                                                                                                    |

## Think in layers

A useful mental model is to separate FlexiGPT into four stable layers.

1. **Provider and transport**
   - Which provider family or compatible endpoint is being used.
2. **Model and inference settings**
   - Which model preset is active and what request defaults it applies.
3. **Behavior setup**
   - Which assistant preset, system prompt sources, prompt templates, tools, and skills shape the request.
4. **Turn context**
   - The current message, selected earlier turns, attachments, and any tool outputs that are being resent.

If the result changes, it is usually because one of these layers changed.

## Assistant Presets and Model Presets solve different problems

### Assistant Presets answer

"What workspace should I start from for this kind of job?"

An assistant preset is best for repeated workflows. It can decide:

- which model preset to start from
- whether the model's system prompt should be included
- which instruction templates should be selected
- which tools should already be available
- which skills should already be enabled

### Model Presets answer

"What model and request defaults should actually run this turn?"

A model preset is closer to execution details. It controls things like:

- provider identity
- model name
- streaming
- timeout
- token limits
- temperature or reasoning behavior
- output settings
- raw provider-specific parameters

## Persistent chat setup versus per-message inputs

### Usually persistent for the current conversation

These controls are part of the chat setup until you change them again:

- assistant preset selection
- model selection and model defaults
- history window through **Previous user turns**
- system prompt source selection
- conversation-level tool choices
- conversation skill state
- compatible web-search selection

### Usually specific to the current message

These belong to the turn you are currently preparing:

- the text you type now
- attachments you add now
- prompt template output inserted for this send
- tool outputs you choose to attach back into the conversation

## What happens on send

At send time, the frontend collects the active chat state and the current message, then the backend normalizes that into a provider request.

The request can include:

- the current user message
- selected model parameters
- combined system prompt content
- selected prompt template output
- earlier user turns allowed by the history control
- attachments belonging to included messages
- selected tool choices and tool outputs
- skill session context and skill-provided prompt/tool behavior when enabled

The exact final payload depends on the selected provider family and model capabilities.

## Tools, tool outputs and human-in-loop/agentic flows

There are several stages when it comes to tool use:

1. a tool is made available to the conversation as a "choice"
2. the model may decide to "call" it
3. the tool runtime may execute it locally or through HTTP, depending on the tool definition
4. the resulting "output" can become part of the next request context

This distinction matters because tool choice, tool call review, tool execution, and tool output reuse are related but not identical steps.

FlexiGPT supports both manual and more agentic tool workflows.

- In a **human-in-loop** setup, the model can propose tool calls and you review or trigger execution yourself and submit.
- In an **agentic** setup, you can mark a tool for **auto-execute**.

When an auto-execute tool is called, FlexiGPT can:

1. run the tool automatically
2. capture the tool result
3. submit that result back to the model without requiring a separate manual send

This lets you build more automated tool-driven flows inside the normal chat workspace, while still keeping manual control available when execution should be reviewed first.

## Skills are workflow frames, not just saved text

Skills are managed separately from prompts and tools.

The skill runtime can:

- create skill sessions
- expose a skills prompt to the model
- list runtime skills
- allow skill tools such as load, unload, and read-resource behavior

From a user perspective, that means skills are better thought of as reusable working modes for a conversation than as simple snippets.

## Attachments belong to messages

Attachments are attached to messages, not to a hidden global chat bucket.

This matters when you change **Previous user turns**:

- if an earlier user turn is included again, its attachments can be included again
- if that earlier user turn is left out, its attachments are left out too

That makes the history control one of the most important context-management knobs in the app.

## Built-in content versus user-defined content

Several admin surfaces use the same pattern:

- built-in bundles and versions are shipped with the app
- user-created entries are stored locally
- built-in content can generally be enabled or disabled
- built-in content is not meant to be freely edited in place

You will see this pattern across model presets, prompts, tools, skills, and assistant presets.
