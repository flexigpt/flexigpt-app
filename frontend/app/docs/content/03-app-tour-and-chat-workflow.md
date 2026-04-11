# App tour and chat workflow

This page shows where things live in the UI and what a normal FlexiGPT session looks like.

## Sidebar pages

| Page                  | What it is for                                                                                        |
| --------------------- | ----------------------------------------------------------------------------------------------------- |
| **Chats**             | The main workspace for conversations, search, tabs, attachments, prompts, tools, skills, and exports. |
| **Composer**          | The input area cockpit where you can control the content seen by the LLM and its behavior.            |
| **Assistant Presets** | Create or edit higher-level starting setups.                                                          |
| **Skills**            | Manage reusable workflow packs.                                                                       |
| **Tools**             | Manage tool bundles and tool definitions.                                                             |
| **Prompts**           | Manage prompt bundles and reusable templates.                                                         |
| **Model Presets**     | Manage providers, models, defaults, and compatible custom endpoints.                                  |
| **Settings**          | Manage auth keys, theme, and debug options.                                                           |
| **Docs**              | Read the bundled markdown docs shipped with the app.                                                  |

## A normal chat workflow

### 1. Start or reopen a conversation

On the **Chats** page you can:

- open a new chat
- switch between chat tabs
- search local conversation history
- reopen an older conversation into a tab
- export the current conversation as JSON

FlexiGPT keeps multiple chat tabs open so you can compare or continue different threads without losing your place.

### 2. Configure the next request

The composer controls how the next send will run.

| Control                     | What it changes                                                                   |
| --------------------------- | --------------------------------------------------------------------------------- |
| **Assistant Preset**        | Applies a starting setup when the preset defines one.                             |
| **Model Preset**            | Chooses the active provider and model.                                            |
| **Reasoning / Temperature** | Changes based on the selected model's capabilities.                               |
| **Output Verbosity**        | Appears only for models that support it.                                          |
| **Previous user turns**     | Controls how much earlier user context is resent.                                 |
| **Advanced parameters**     | Opens streaming, token, timeout, output format, stop sequences and cache control. |

### 3. Add context and helpers

The composer bottom bar lets you add the supporting pieces for the next request.

- **Attachments** for files, folders, images, PDFs, and URLs
- **System Prompt** sources for durable instructions
- **Prompts** for reusable templates
- **Tools** for callable capabilities, including tools you may keep manual or mark for auto-execution
- **Skills** for reusable workflow frames
- **Web Search** when compatible with the current provider family

Use only the pieces that help the current request. More context is not always better context.

### 4. Send and inspect the result

After you send:

- assistant text can stream into the message view
- responses can render as Markdown, syntax-highlighted code, Mermaid diagrams, and KaTeX math
- token usage is available after completion
- citations may appear when the provider returns them
- tool calls or tool outputs can appear in the thread
- message details can help with debugging

Per-message actions can include:

- copy
- message details
- token usage
- markdown toggle
- Mermaid zoom and export actions where applicable
- edit and resend for user messages

### 5. Continue, branch, or switch setup

You can change the setup between turns without starting over.

For example, you can:

- switch to a different model preset
- change the history window
- add or remove tools
- enable or disable skills
- change system instructions
- attach new files or URLs

If you edit and resend an earlier user message, later messages in that branch are removed so the conversation stays consistent.

## Tools, skills, and web search behavior

These three often work together, but they are not the same thing.

### Tools

Tools are explicit capabilities attached to the conversation.

Important behavior:

- tool choices persist per conversation until you change them
- individual tools can be left manual or marked for auto-execution
- some tools require user configuration before send
- when an auto-execute tool is called, FlexiGPT can run it and automatically submit the result back to the model
- tool outputs can become part of later request context

### Skills

Skills are reusable workflow frames. The app implements [agent-skills specification](https://agentskills.io/specification) for the workflow and users can write skills using the same specification and use it inside FlexiGPT.

They are useful when you want the model to approach work in a consistent way, such as review, refactoring, or structured implementation.

### Web search

Web search is exposed through provider-compatible tool flows.

That means:

- only compatible web-search options are shown for the current provider family
- switching providers can clear incompatible web-search choices
- web search is best used when recency actually matters

Note that most providers have separate billing for web-search tool calls than tokens.

## Rich message rendering

FlexiGPT can render assistant messages as more than plain text.

- Markdown
- syntax-highlighted code blocks
- Mermaid diagrams
- KaTeX math
- citations when returned by the provider
- token usage and request or response inspection details

This makes the chat view useful not only for reading answers, but also for inspecting diagrams, math, code, and message-level execution details in place.

## History, search, and export

Conversation history is stored locally.

In practice this means:

- conversations remain on the device unless you export or share them
- local search runs against local conversation data
- reopening an old conversation restores it into the current workspace
