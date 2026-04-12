# Chats, Composer, and Everyday Workflow

This page focuses on the day-to-day workspace inside FlexiGPT.

## App surfaces at a glance

| Page                  | What it is for                                                              |
| --------------------- | --------------------------------------------------------------------------- |
| **Home**              | Lightweight entry point into chats and bundled docs.                        |
| **Chats**             | Main workspace for conversations, tabs, search, messages, and the composer. |
| **Assistant Presets** | Manage reusable starting setups.                                            |
| **Prompts**           | Manage prompt bundles and templates.                                        |
| **Tools**             | Manage tool bundles and tool definitions.                                   |
| **Skills**            | Manage skill bundles and skills.                                            |
| **Model Presets**     | Manage providers, model presets, and the default provider.                  |
| **Settings**          | Manage theme, provider keys, and debug settings.                            |
| **Docs**              | Read the bundled setup, user, and architecture docs.                        |

## What the Chats page brings together

The **Chats** page combines four responsibilities in one place:

- conversation search
- chat tabs
- the active conversation timeline
- the composer for the next request

That is why most everyday work happens there.

## A normal workflow

### 1. Open or create a conversation

On **Chats**, you can:

- start a fresh conversation
- switch between open tabs
- search local history
- reopen a saved conversation
- export the current conversation as JSON

Multiple tabs make it easier to compare or continue different threads without losing your place.

### 2. Configure the next request in the context bar

At the top of the composer, the context bar controls the setup for the next request.

Use it when you want to change how the next request should run.

Examples:

- switch to a stronger model
- reduce temperature for a stricter answer
- shrink the history window because the thread drifted
- open advanced parameters for more control

| Control                      | What it influences                                                                                           |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------ |
| **Assistant Preset**         | Reusable starting workspace for model, instructions, tools, and skills.                                      |
| **Model**                    | Active provider and model choice for the request.                                                            |
| **Temperature or Reasoning** | Quality and style controls exposed by the selected model.                                                    |
| **Output Verbosity**         | Output verbosity when the current model supports it.                                                         |
| **Previous user turns**      | How much earlier user context is resent.                                                                     |
| **Advanced parameters**      | Streaming, token limits, timeout, output format, stop sequences, cache control, and similar request options. |

### 3. Prepare the current turn in the editor area

Below the context bar, the editor area is where you build the current request.

That can include:

- the message text
- attachments
- prompt template insertion
- system prompt selection
- conversation tool choices
- web-search selection when supported
- skill selection
- pending tool calls and tool outputs in tool-assisted flows

This is where human input, reusable configuration, and execution helpers come together.

### 4. Add only the context that helps

The composer lets you add supporting context for the current request.

Common examples:

- **Attachments** for files, folders, images, PDFs, and URLs
- **System Prompt** sources for durable instructions
- **Prompts** for reusable request structure
- **Tools** for callable capability
- **Skills** for reusable workflow modes
- **Web Search** when the current provider supports it and fresh information matters

A focused request usually works better than an overloaded one.

## What happens after send

After you send:

- assistant text can stream into the message view
- responses can render as Markdown, syntax-highlighted code, Mermaid diagrams, and math
- token usage becomes available after completion
- citations may appear when the provider returns them
- tool calls can appear in the thread
- tool outputs appear once you run them manually or they auto-execute
- message details help with inspection and debugging

## Tool-assisted conversations inside the chat flow

When tools are available to a conversation, the workflow still stays in the chat.

### Manual review

In a manual flow:

- the model proposes a tool call
- you inspect the call
- you decide whether to run it
- the output is then available for the next step in the conversation

### Auto-execute

In an auto-execute flow:

- the model proposes a tool call
- FlexiGPT runs it automatically when the tool is configured for auto-execution and the call has the required arguments
- the result is submitted back into the conversation so the model can continue

That is the app's more agentic mode: faster tool loops, but still bounded by the tools you selected and how you configured them.

## Reading results in the message timeline

The message area is more than a plain transcript.

It can show:

- rendered Markdown
- code blocks
- Mermaid diagrams with zoom and export
- math rendering
- citations when present
- attachments, tool calls, and tool outputs under messages
- token usage and message details

Per-message actions can include:

- copy
- message details
- token usage
- Markdown toggle
- Mermaid actions where applicable
- edit and resend for user messages

## Editing and branching

User messages can be edited and resent.

When you resend an earlier user message:

- that message is loaded back into the composer
- later messages in that branch are dropped
- the updated message becomes the new continuation point

That makes resend a branching workflow rather than a hidden patch on the old transcript.

## Search, tabs, and continuity

The chat workspace also preserves local working context for you.

That includes:

- multi-tab navigation
- reopening saved conversations
- restoring scroll position
- handling attachment drops into the active tab
- keeping the active tab's composer and runtime state aligned with the selected conversation

## When to leave the Chats page

Stay on **Chats** for most work.

Leave it when you need to change the reusable building blocks behind the chat, such as:

- creating or editing an assistant preset
- updating a tool definition
- adding a prompt template
- changing provider or model setup
- changing auth keys or debug settings
