# Getting Started

This guide is the fastest path from install to a useful first conversation in FlexiGPT.

## What FlexiGPT expects from you

FlexiGPT is a desktop client for working with remote or local model providers from one workspace.

Before your first successful send, you usually need:

- the installed desktop app
- at least one enabled provider with an API key in **Settings**
- a normal question, task, or file to work with

FlexiGPT ships with built-in providers, model presets, tools, skills, prompts, and assistant presets, so the first run is mostly about connecting your key and picking a starting setup.

## First-run checklist

1. Open **Settings**.
2. Add a provider key in **Auth Keys**.
3. Open **Chats**.
4. Choose an **Assistant Preset** and **Model Preset**, or keep the defaults.
5. Write a message.
6. Optionally attach files, prompts, tools, or skills.
7. Send the request and inspect the result.

## 1. Add a provider key

The app can only send a live request after you configure credentials for at least one provider.

In **Settings**:

- use **Auth Keys** to add a provider key
- return to **Chats** after saving it

From the backend code, provider secrets are stored through the OS keyring-backed settings and not in plain text.

## 2. Open the Chats workspace

The **Chats** page is the main working surface.

From here you can::

- start a new conversation
- work parallelly across multiple chat tabs
- export the current conversation as JSON
- search conversation history; if you already have older conversations, you can reopen them into tabs and continue from there.

## 3. Pick a starting setup

Before you send, decide what kind of workspace you want for the next request.

### Assistant Preset

Use an assistant preset when you want a reusable starting setup.

An assistant preset can preload:

- a starting model preset
- instruction templates
- tool selections
- skill selections

### Model Preset

Use a model preset to choose the provider, model, and inference defaults.

That includes things like:

- provider family
- model name
- temperature or reasoning controls
- streaming and timeout behavior
- output options and advanced parameters

### Previous user turns

Use this control to decide how much earlier user context should be resent with the next request.

If you are unsure, start small. You can always increase the history window later.

## 4. Write the first message

Good first requests are simple and concrete.

Examples:

- explain this file
- summarize these notes
- compare these two approaches
- review this code snippet
- rewrite this paragraph more clearly

## 5. Add optional context

The composer can send much more than plain text.

You can add:

- **System prompts** for durable instructions
- **Prompt templates** for reusable request structure
- **Attachments** such as files, folders, images, PDFs, and URLs
- **Tools** when the task needs callable capabilities (file/text read/writes etc)
- **Skills** when you want a reusable workflow frame (structured reviews, spec driven development, etc)
- **Web search** for working with recent information from the interne (needs LLM provider support)

## 6. Send and inspect the result

When you send a message, FlexiGPT builds a request from the current chat state and forwards it through the selected provider.

After the response completes, you can usually:

- read the answer as rendered markdown
- inspect token usage and message details
- review citations when the provider returns them
- copy the content
- edit and resend earlier user messages to branch the conversation

If the assistant emits tool calls, the chat workflow can continue through the composer and tool runtime instead of forcing you into a separate interface.

## Good first workflows

Once the first send works, try one of these:

- attach a local file and ask for an explanation
- compare two model presets on the same question
- use a prompt template to standardize a repeated task
- enable a tool-assisted assistant preset for a workflow that needs execution help

## If the first send fails

Check these first:

- Did you add a provider key in **Settings**?
- Is the selected provider enabled on **Model Presets**?
- Is the chosen model preset enabled?
- Is the request being sent with too much stale history?
- Did you attach a tool or web-search option that still needs configuration?
