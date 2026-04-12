# Presets, Providers, and Settings

The pages outside **Chats** manage the reusable building blocks behind your day-to-day workflow.

## Assistant Presets

An assistant preset is a reusable starting setup.

It can define a starting combination of:

- model preset with patch overrides
- whether the model's own system prompt should be included
- instruction template references
- tool selections
- skill selections

### When to use an assistant preset

Create one when you keep rebuilding the same environment by hand.

Examples:

- a code review setup
- a documentation writing setup
- a tool-assisted workspace setup
- a planning workflow with specific prompts and skills

## Prompts

The **Prompts** page manages prompt bundles and template versions.

Use it to maintain reusable request structure instead of rewriting the same framing in every conversation.

## Tools

The **Tools** page manages the tool catalog.

This is where you maintain the tool definitions that conversations can later use. Actual execution still happens from the chat workflow.

## Skills

The **Skills** page manages the skill catalog.

This is where you maintain the reusable workflow modes that conversations can draw from.

## Model Presets

The **Model Presets** page manages the execution layer.

It is responsible for:

- the default provider for the app
- provider presets
- model presets under each provider
- enabled or disabled state for providers and models
- compatible custom provider endpoints

### A useful mental split

- If you are deciding **how the request runs**, use **Model Presets**.
- If you are deciding **what kind of workspace you want**, use **Assistant Presets**.

## Settings

The **Settings** page handles app-wide configuration.

It includes:

- **Theme**
- **Auth Keys**
- **Debug**

### Auth Keys

This is where provider credentials are added.

Key metadata is stored locally while secrets are protected through the OS keyring.

### Debug

Debug settings affect how much internal request and response detail is logged or preserved.

Use them carefully when working with sensitive data.

## Built-in content and your own content

Across assistant presets, prompts, tools, skills, and model presets, the same broad pattern appears:

- built-in items ship with the app
- your own items are stored locally
- built-in items can usually be enabled or disabled
- built-in items are generally treated as read-only definitions

## How to decide which page to use

| Goal                                     | Page                  |
| ---------------------------------------- | --------------------- |
| Reuse a whole working setup              | **Assistant Presets** |
| Reuse message structure                  | **Prompts**           |
| Add or maintain callable capability      | **Tools**             |
| Maintain reusable workflow modes         | **Skills**            |
| Change provider or model execution setup | **Model Presets**     |
| Manage keys, theme, and debug            | **Settings**          |
