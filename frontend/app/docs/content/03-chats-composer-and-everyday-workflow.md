# Chats, Composer, and Everyday Workflow

This page focuses on the day-to-day working surface inside FlexiGPT.

## The app surfaces at a glance

| Page                  | What it does                                                                                |
| --------------------- | ------------------------------------------------------------------------------------------- |
| **Home**              | Lightweight landing page with entry points into chats and bundled docs.                     |
| **Chats**             | The primary working area for conversations, tabs, local search, messages, and the composer. |
| **Assistant Presets** | Manage reusable starting setups.                                                            |
| **Prompts**           | Manage prompt bundles and templates.                                                        |
| **Tools**             | Manage tool bundles and tool definitions.                                                   |
| **Skills**            | Manage skill bundles and skills.                                                            |
| **Model Presets**     | Manage providers, model presets, and the default provider.                                  |
| **Settings**          | Manage theme, provider keys, and debug settings.                                            |
| **Docs**              | Read the bundled user and architecture docs.                                                |

## What the Chats page is made of

The `Chats` route stitches together four major responsibilities:

- **conversation search** in the title bar
- **chat tabs** across open conversations
- **conversation area** for the selected thread
- **composer** for the next request

That structure matters because most of the app's actual day-to-day work happens here.

## A normal workflow

### 1. Open or create a conversation

On **Chats**, you can:

- start from a fresh tab
- switch between existing tabs
- search local history
- reopen a stored conversation into the current workspace
- export the current conversation as JSON

The conversation store keeps these threads locally, and the search surface uses the local conversation index. Also, FlexiGPT keeps multiple chat tabs open so you can compare or continue different threads without losing your place.

### 2. Configure the next request in the context bar

At the top of the composer, the context bar controls the setup for the next request. This bar is where you decide the shape of the next request before you add turn-specific context. Use this bar when you are changing how the chat should behave.

Examples:

- switch to a stronger model
- reduce temperature for a stricter answer
- shrink the history window because the thread drifted
- open advanced parameters for output constraints

| Control                      | What it influences                                                              |
| ---------------------------- | ------------------------------------------------------------------------------- |
| **Assistant Preset**         | Reusable starting workspace for model, instructions, tools, and skills.         |
| **Model**                    | Active provider and model choice for the request.                               |
| **Temperature or Reasoning** | The quality and style controls exposed by the selected model capabilities.      |
| **Output Verbosity**         | Output verbosity when the current model supports it.                            |
| **Previous user turns**      | How much earlier user context is resent.                                        |
| **Advanced parameters**      | Streaming, token limits, timeout, output format, stop sequences, cache control. |

### 3. Prepare the message in the editor area

Below the context bar, the editor area is where you build the current turn.

That includes:

- the message text itself
- current attachments
- prompt template insertion
- system prompt selection
- conversation tool choices
- web-search selection when supported
- skill selection
- pending tool calls and tool outputs when the conversation is in a tool-assisted flow

In practice, this is the place where human input, reusable configuration, and execution helpers come together.

### 4. Add context and helpers using the composer bottom bar

The composer bottom bar lets you add the supporting pieces for the next request. Use it when you are changing what the model should see or what it may use.

Examples:

- **Attachments** for files, folders, images, PDFs, and URLs
- **System Prompt** sources for durable instructions
- **Prompts** for reusable templates
- **Tools** for callable capabilities, including tools you may keep manual or mark for auto-execution
- **Skills** for reusable workflow frames
- **Web Search** when compatible with the current provider family

Use only the pieces that help the current request. More context is not always better context.

## What happens after send

After you send:

- assistant text can stream into the message view
- responses can render as Markdown, syntax-highlighted code, Mermaid diagrams, and KaTeX math
- token usage is available after completion
- citations may appear when the provider returns them
- tool calls can appear in the thread
- tool outputs will be visible once executed manually or automatically.
- message details can help with debugging

## Reading results inside the message timeline

The message area is more than a plain transcript.

From the current message components, the UI can show:

- markdown-rendered content
- syntax-highlighted code blocks
- Mermaid diagrams with zoom and export
- KaTeX math rendering
- citations when present
- attachments, tool choices, tool calls, and tool outputs under the message
- token usage and message details

Per-message actions can include:

- copy
- message details
- token usage
- markdown toggle
- Mermaid zoom and export actions where applicable
- edit and resend for user messages

The assistant and user messages are also treated differently in the UI, which makes it easier to inspect the structure of the conversation.

## Editing and branching

User messages can be edited and resent.

The current send flow is important here:

- the selected user message is loaded back into the composer
- when you resend it, later messages in that branch are dropped
- the updated message becomes the new continuation point

This is a deliberate branching model, not a hidden patch on the old transcript.

## Search, tabs, and scroll state

The chat workspace also handles a lot of local working-state responsibilities for you.

That includes:

- multi-tab navigation
- reopening saved conversations
- restoring scroll position
- handling attachment drops into the active tab
- keeping the selected tab's composer and runtime state aligned with the selected conversation

## When to leave the Chats page

You stay on **Chats** for most work.

You usually leave it only when you need to change the reusable building blocks behind the chat, such as:

- creating or editing an assistant preset
- updating a tool definition
- adding a new prompt template
- changing provider or model setup
- changing auth keys or debug settings
