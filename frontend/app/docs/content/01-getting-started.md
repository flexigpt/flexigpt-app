# Getting Started

FlexiGPT is a local-first BYOK AI workspace for repeatable LLM work. You bring provider keys or local endpoints, then combine models, assistant presets, prompts, attachments, tools, and skills inside a local desktop app.

This page is the shortest path to a useful first request.

## Table of contents <!-- omit from toc -->

- [First successful send](#first-successful-send)
- [Choose your first path](#choose-your-first-path)
- [What to look at before sending](#what-to-look-at-before-sending)
- [A good first request](#a-good-first-request)
- [After the response](#after-the-response)
- [Where to go next](#where-to-go-next)
- [If the first send fails](#if-the-first-send-fails)

## First successful send

1. Get an API key for a provider you want to use.
   - OpenAI
   - Anthropic Claude
   - Google Gemini API
   - xAI
   - Mistral AI
   - Hugging Face
   - OpenRouter
   - Any compatible local LLM server
2. Open **Settings -> Auth Keys**. Add the provider key.
3. Open **Model Presets**. Confirm the provider and at least one model preset are enabled.
4. Open **Chats**. Select a model preset. Type a small test request. Send.

Good first test: `Reply with one sentence confirming the model is working.`

Once that works, attach source material or use a starter assistant preset.

## Choose your first path

| If you want to...                       | Start here                                                                                                                          |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Use a normal hosted model               | Add the key in **Settings**, enable the model in **Model Presets**, then use **Chats**.                                             |
| Try many hosted models through one key  | Use OpenRouter. See [Providers and Models](/docs?doc=providers-and-models#openrouter).                                              |
| Use a local model server                | Configure a custom compatible provider. See [Providers and Models](/docs?doc=providers-and-models#local-openai-compatible-servers). |
| Start from a known workflow             | Choose a home screen workflow card or an assistant preset in **Chats**.                                                             |
| Work with private or sensitive material | Read [Privacy, Data, and Troubleshooting](/docs?doc=privacy-data-and-troubleshooting) before sending.                               |

## What to look at before sending

Before each important request, check these layers:

1. **Provider/model**
   - Which endpoint receives the request?
2. **Assistant preset**
   - Did you apply a starter workflow?
3. **Instructions**
   - Are model default prompts or saved system prompts active?
4. **Current message**
   - Is the request clear and specific?
5. **Context**
   - Are the right attachments, prompt templates, tools, skills, and web search selections active?
6. **History**
   - Is **Previous user turns** small enough to avoid stale context?

If the output changes unexpectedly, usually one of these layers changed.

## A good first request

Start with one focused task and one focused piece of context.

Examples:

- Explain this attached file.
- Summarize these notes.
- Compare these two approaches.
- Develop this feature in the attached repo. First inspect relevant files, write a short spec, then implement after confirmation.
- Review this code snippet for correctness and test gaps.
- Rewrite this paragraph for a technical README.
- Turn this stack trace and log excerpt into likely causes and next checks.

Avoid beginning with a huge folder, many tools, and long chat history. Add complexity after the basic path works.
If a starter workflow prefills the composer, replace its placeholder with your concrete task before sending.

## After the response

In the message timeline you can usually:

- read rendered Markdown, code, diagrams, and math
- inspect token usage and message details
- review citations when the provider returns them
- copy output
- inspect tool calls and tool outputs
- edit an earlier user message and resend to branch the conversation
- export the current conversation as JSON

## Where to go next

- Learn the vocabulary and page ownership: [Concepts and Ownership](/docs?doc=concepts-and-ownership)
- Learn the main work surface: [Chat Workspace](/docs?doc=chat-workspace)
- Learn how to add files, prompts, tools, skills, and web search: [Composer Context](/docs?doc=composer-context)
- Learn how to maintain reusable assistant presets and catalogs: [Reusable Catalogs](/docs?doc=reusable-catalogs)
- Configure providers and local models: [Providers and Models](/docs?doc=providers-and-models)
- Try outcome-based tasks: [Everyday Recipes](/docs?doc=everyday-recipes)

## If the first send fails

Check:

- provider key exists in **Settings -> Auth Keys**
- provider is enabled in **Model Presets**
- model preset is enabled
- model preset uses the correct provider/API type
- custom endpoint origin includes `http://` or `https://`
- local server is running, if using a local endpoint
- current tool or web-search option is not blocked by missing required arguments
- request still has real user content after removing stale attachments or failed tool outputs
