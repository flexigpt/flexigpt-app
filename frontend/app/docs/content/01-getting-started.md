# Getting Started

FlexiGPT is a local-first BYOK AI workspace for power users and teams who need repeatable prompts, tools, skills, model choices, assistants/agents, and private local history across multiple LLM providers.

This guide is the fastest path after installation: connect a provider, choose a starting assistant/model setup, add context, and send a useful first message.

## Table of contents <!-- omit from toc -->

- [First steps](#first-steps)
- [Choose a setup path](#choose-a-setup-path)
- [Chats workspace](#chats-workspace)
- [Pick a starting setup](#pick-a-starting-setup)
  - [Assistant Preset](#assistant-preset)
  - [Model Preset](#model-preset)
  - [Previous user turns](#previous-user-turns)
- [Writing the first message](#writing-the-first-message)
- [Adding optional context](#adding-optional-context)
- [Send and inspect the result](#send-and-inspect-the-result)
- [Good first workflows](#good-first-workflows)
- [If the first send feels off](#if-the-first-send-feels-off)

## First steps

Before your first successful send, you usually need:

- Get an API key for your favorite provider.
  - [OpenAI](https://platform.openai.com/settings/organization/api-keys), [Anthropic Claude](https://platform.claude.com/settings/keys), [Google Gemini](https://aistudio.google.com/api-keys), [xAI](https://console.x.ai/team/default/api-keys), [Mistral AI](https://console.mistral.ai/home?profile_dialog=api-keys)
  - [OpenRouter](https://openrouter.ai/workspaces/default/keys), [Hugging Face](https://huggingface.co/settings/tokens)

- Add the key in [Settings -> Auth Keys](/settings).
- Open the home screen and pick a starter workflow, or go directly to the [Chats](/chats) page.
- Try different assistants for spec-driven development, code review, architecture review, or refactoring.
- Attach files, folders, URLs, PDFs, or notes when the model needs source material.

FlexiGPT ships with built-in providers, model presets, tools, skills, prompts, and assistant presets. These building blocks are meant to be combined into reusable workflows, not configured from scratch every time.

- Using another provider or endpoint? You can create a custom provider compatible with OpenAI Responses, OpenAI Chat Completions, Anthropic Messages, or Google GenerateContent APIs.
- Want different defaults? Customize or create your own providers, model presets, prompts, tools, skills, and assistant presets.

## Choose a setup path

- **Hosted provider**: add the provider key in **Settings -> Auth Keys**, then choose an enabled model preset in **Chats**.
- **OpenRouter**: add the OpenRouter key, enable the OpenRouter provider/model preset, then test a small prompt.
- **Custom or local endpoint**: use [Provider and Local Model Setup](/docs?doc=provider-and-local-model-setup) before sending sensitive work.
- **Sensitive work**: review [Security, Privacy, and Trust Model](/docs?doc=security-privacy-and-trust-model) so you know what can leave the machine.

If you are unsure, start with a hosted provider or OpenRouter and send a tiny test prompt first.

## Chats workspace

The **Chats** page is the main place where you work.

From there you can:

- start a new conversation
- work across multiple chat tabs in parallel
- search local conversation history
- reopen earlier conversations and continue them
- export the current conversation as JSON

## Pick a starting setup

Before you send, decide what kind of workflow you want for this conversation.

If you just want the quick version: an assistant preset shapes the workflow, while a model preset chooses the provider, model, and request defaults.

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

## Writing the first message

Good first requests are simple and concrete.

Examples:

- explain this file
- summarize these notes
- compare these two approaches
- review this code snippet
- rewrite this paragraph more clearly

The home screen starter workflow cards provide useful entry points for common workflows like spec driven dev, reviews, etc.

## Adding optional context

The composer can send much more than plain text.

You can add:

- **System prompts** for durable instructions
- **Prompt templates** for reusable request structure
- **Attachments** such as files, folders, images, PDFs, and URLs
- **Tools** when the task needs execution capability
- **Skills** when you want a reusable workflow style
- **Web search** when recent information matters and the current provider supports it

Only add the context that helps the current request. More context is not always better.

## Send and inspect the result

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
- start from a built-in software assistant such as Reviewing Code, Spec Driven Development, Refactoring Code, or Designing System Architecture

## If the first send feels off

Check these first:

- the provider key was added in **Settings**
- the selected provider is enabled in **Model Presets**
- the selected model preset is enabled
- the history window through **Previous user turns** is not larger than needed
- the current tool or web-search choice is configured correctly
