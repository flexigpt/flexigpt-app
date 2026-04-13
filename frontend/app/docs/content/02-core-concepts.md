# Core Concepts

FlexiGPT becomes much easier to use once a few core terms are clear.

## Table of contents <!-- omit from toc -->

- [The main terms](#the-main-terms)
- [Think in four layers](#think-in-four-layers)
- [Assistant Presets and Model Presets solve different problems](#assistant-presets-and-model-presets-solve-different-problems)
  - [Assistant Presets answer](#assistant-presets-answer)
  - [Model Presets answer](#model-presets-answer)
- [Persistent conversation setup versus current-message input](#persistent-conversation-setup-versus-current-message-input)
  - [Usually persistent for the conversation](#usually-persistent-for-the-conversation)
  - [Usually specific to the current message](#usually-specific-to-the-current-message)
- [What happens on send](#what-happens-on-send)
- [Tool flows: human-in-loop and more agentic modes](#tool-flows-human-in-loop-and-more-agentic-modes)
  - [Human-in-loop](#human-in-loop)
  - [Auto-execute and agentic flow](#auto-execute-and-agentic-flow)
- [Skills are workflow modes, not just saved text](#skills-are-workflow-modes-not-just-saved-text)
- [Attachments belong to messages](#attachments-belong-to-messages)
- [Built-in content versus your local content](#built-in-content-versus-your-local-content)

## The main terms

| Term                    | What it means                                                                                                                                                       |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Provider**            | The API family or endpoint the app talks to, such as OpenAI, Anthropic, Google Gemini API, OpenRouter, `llama.cpp`, or a compatible custom endpoint.                |
| **Model Preset**        | A saved provider-and-model choice with request defaults such as streaming, timeout, token limits, temperature, reasoning, output settings, and advanced parameters. |
| **Assistant Preset**    | A reusable starting workspace that can preload a model preset, instruction templates, tools, and skills.                                                            |
| **System Prompt**       | Instructional context that shapes how the model should behave.                                                                                                      |
| **Prompt Template**     | Reusable prompt structure that helps you format a request consistently.                                                                                             |
| **Previous user turns** | The amount of earlier user context that should be resent with the next request.                                                                                     |
| **Attachment**          | Extra source material attached to a message, such as a file, folder-derived files, image, PDF, or URL.                                                              |
| **Tool**                | A callable capability the model can ask the app to run during a conversation.                                                                                       |
| **Skill**               | A reusable workflow mode that helps the model approach a task in a more structured way. [Specification](https://agentskills.io/specification)                       |

## Think in four layers

A useful mental model is to separate FlexiGPT into four layers.

1. **Provider and transport**
   - Which provider family or compatible endpoint is being used.
2. **Model and request settings**
   - Which model preset is active and what defaults it applies.
3. **Behavior setup**
   - Which assistant preset, system prompt sources, prompt templates, tools, and skills shape the conversation.
4. **Turn context**
   - The current message, selected earlier turns, attachments, and any tool outputs included in the next request.

If a result changes, it is usually because one of these layers changed.

## Assistant Presets and Model Presets solve different problems

### Assistant Presets answer

"What kind of workspace should I start from?"

Use an assistant preset when you want a reusable setup for a type of work. It can decide:

- which model preset to start from
- whether the model's own system prompt should be included
- which instruction templates should be selected
- which tools should already be available
- which skills should already be enabled

### Model Presets answer

"What model and request defaults should run this turn?"

A model preset controls execution details such as:

- provider identity
- model name
- streaming
- timeout
- token limits
- temperature or reasoning behavior
- output settings
- raw provider-specific parameters

## Persistent conversation setup versus current-message input

### Usually persistent for the conversation

These usually stay active until you change them:

- assistant preset selection
- model selection and model defaults
- history window through **Previous user turns**
- system prompt source selection
- conversation-level tool choices
- conversation skill state
- compatible web-search selection

### Usually specific to the current message

These belong to the turn you are preparing now:

- the text you type
- attachments you add now
- prompt template output inserted for this send
- tool outputs you choose to include back into the conversation

## What happens on send

When you send a message, FlexiGPT combines the active conversation setup with the current turn.

That request can include:

- the current user message
- selected model parameters
- system prompt content
- prompt template output
- earlier user turns allowed by the history setting
- attachments belonging to included messages
- selected tool choices and tool outputs
- skill session context and skill-related behavior when enabled

The exact final payload depends on the selected provider family and model capabilities.

## Tool flows: human-in-loop and more agentic modes

Tool use has a few separate stages:

1. a tool is made available to the conversation
2. the model may propose a tool call
3. the call may be reviewed or executed
4. the tool output may be sent back as part of the continuing conversation

FlexiGPT supports both manual review and more automated or agentic tool loops.

### Human-in-loop

Use this when execution should stay under closer control:

- the model proposes a tool call
- you review it
- you decide whether to run it
- the resulting output can then be submitted back into the conversation

### Auto-execute and agentic flow

In a more automated or agentic flow, you can mark a tool for **auto-execute**.

When a matching tool call is produced and it has the required arguments, FlexiGPT can:

1. run the tool automatically
2. capture the result
3. submit that result back to the model so the conversation can continue

This keeps tool-assisted workflows more automatic while still staying within the configured tools and the current chat flow.

## Skills are workflow modes, not just saved text

Skills are separate from prompts and tools.

From a user perspective, a skill is best understood as a reusable working mode for a conversation. It helps the model approach a task in a consistent way across turns.

## Attachments belong to messages

Attachments are attached to messages, not to a hidden global chat bucket.

This matters when you change **Previous user turns**:

- if an earlier user turn is included again, its attachments can be included again
- if that earlier user turn is left out, its attachments are left out too

That makes the history control one of the most important context-management controls in the app.

## Built-in content versus your local content

Across model presets, prompts, tools, skills, and assistant presets, the same broad pattern appears:

- built-in content ships with the app
- your own entries are stored locally
- built-in content can usually be enabled or disabled
- built-in content is generally treated as read-only

The fuller user-facing explanation is in **Presets, Providers, and Settings**, but the short version is that FlexiGPT gives you ready-to-use defaults without taking away local customization.
