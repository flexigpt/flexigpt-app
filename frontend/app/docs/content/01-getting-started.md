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

FlexiGPT can only send a live request after you configure credentials for at least one provider.

In **Settings**:

- open **Auth Keys**
- add a key for the provider you want to use
- return to **Chats** after saving it

Provider secrets are stored through the OS keyring rather than kept in plain-text settings.

## 2. Open the Chats workspace

The **Chats** page is the main place where you work.

From there you can:

- start a new conversation
- work parallelly across multiple chat tabs
- search local conversation history
- reopen earlier conversations and continue them
- export the current conversation as JSON

## 3. Pick a starting setup

Before you send, decide what kind of workspace you want for this conversation.

### Assistant Preset

Use an assistant preset when you want a reusable starting setup for a type of work.

An assistant preset can preload:

- a model preset
- instruction templates
- tool selections
- skill selections

### Model Preset

Use a model preset to choose the provider, model, and request defaults.

That can include:

- provider family
- model name
- temperature or reasoning controls
- timeout and streaming behavior
- output or advanced request parameters

### Previous user turns

Use this control to decide how much earlier user context should be resent with the next request.

If you are unsure, start small. You can always include more history later.

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
- **Tools** when the task needs callable capability
- **Skills** when you want a reusable workflow style
- **Web search** when recent information matters and the current provider supports it

Only add the context that helps the current request. More context is not always better.

## 6. Send and inspect the result

After you send, you can usually:

- read the answer as rendered Markdown
- inspect token usage and message details
- review citations when the provider returns them
- copy the content
- edit and resend earlier user messages to branch the conversation

If the assistant proposes a tool call, the workflow stays inside the chat. You can review and run the call yourself, or allow configured tools to auto-execute when that fits the task.

## Good first workflows

Once the first send works, try one of these:

- attach a local file and ask for an explanation
- compare two model presets on the same question
- use a prompt template for a repeated task
- enable a tool-assisted assistant preset for a workflow that needs execution help

## If the first send feels off

Check these first:

- the provider key was added in **Settings**
- the selected provider is enabled in **Model Presets**
- the selected model preset is enabled
- the history window is not larger than needed
- the current tool or web-search choice is configured correctly
