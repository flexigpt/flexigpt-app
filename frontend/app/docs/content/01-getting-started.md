# Getting started

This page is the shortest path from install to your first useful conversation in FlexiGPT.

## Before you begin

- FlexiGPT is a desktop client for remote model providers.
- You bring your own provider API key.
- FlexiGPT already includes built-in providers and curated model presets for leading APIs, so in most cases you do not need to define custom endpoints or model defaults just to get started.
- Billing and rate limits come from that provider, not from FlexiGPT.

## 1. Add a provider key

1. Open **Settings** from the sidebar.
2. In **Auth Keys**, add at least one provider API key.
3. Save the key, then go back to **Chats**.

If you do not add a provider key first, you will not have a live model to send requests to.

## 2. Open the chat workspace

The **Chats** page is the main working area.

From here you can:

- start a new conversation
- search local conversation history
- work across multiple chat tabs
- export the current conversation as JSON

## 3. Pick a starting setup

Before sending your first message, choose the setup for the next request.

- **Assistant Preset** chooses a starting workspace. It can preload a model, instructions, tools, and skills.
- **Model Preset** chooses the provider and model that will handle the next request.
- **Previous user turns** decides how much earlier user context should be resent with the next request.

If you are unsure, start with the default assistant preset and a built-in model preset from the provider you just configured.

In most cases, getting started is simply:

1. add your API key
2. choose a built-in preset
3. send your first message

You can add custom providers or tune advanced settings later.

## 4. Write your message

Start simple.

A good first message is direct and specific, for example:

- explain a file
- summarize notes
- review a short code snippet
- rewrite a paragraph
- answer a focused question

## 5. Add optional context

The composer lets you add more than plain text.

You can optionally add:

- **Attachments** such as files, folders, images, PDFs, or URLs
- **Prompt templates** for reusable message structure
- **System prompt sources** for durable instructions
- **Tools** when the task needs callable capabilities
- **Skills** when you want a reusable workflow frame
- **Web search** when the selected provider family supports it

## 6. Send the message

When you click **Send**, FlexiGPT builds a request from your current chat setup and sends it to the selected provider.

After the response completes, you can usually:

- copy the answer
- inspect token usage
- open message details
- toggle markdown rendering
- edit and resend earlier user messages to branch the conversation
