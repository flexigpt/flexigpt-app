# Presets, Providers, and Settings

The pages outside **Chats** exist to manage the reusable building blocks behind your day-to-day workflow.

## Assistant Presets page

The **Assistant Presets** page manages reusable starting setups.

An assistant preset can define a starting combination of:

- model preset with patch overrides
- whether the model's own system prompt should be included
- instruction template references
- tool selections
- skill selections

### When to use assistant presets

Create one when you keep rebuilding the same environment by hand.

Examples:

- a code-review setup
- a documentation-writing setup
- a tool-assisted local workspace setup
- a planning workflow with specific instruction templates and skills

## Prompts page

The **Prompts** page manages prompt bundles and prompt template versions.

The prompt store supports:

- bundles that can be enabled or disabled together
- individual template versions
- built-in prompt content plus user-created content

This is the place to maintain reusable request structure rather than repeating the same prompt framing in every conversation.

## Tools page

The **Tools** page manages tool bundles and tool versions.

The main tool categories are:

- built-in tools shipped with the app
- user-created HTTP tools

The page is for managing definitions and availability. Actual tool use still happens through the composer and the conversation workflow.

## Skills page

The **Skills** page manages skill bundles and individual skills.

The skills supported are:

- built-in skills shipped with the app
- user-defined skills that are stored in the local filesystem

This page is where you maintain the reusable workflow catalog that conversations can draw from.

## Model Presets page

The **Model Presets** page manages the execution layer.

It is responsible for:

- the default provider for the app
- provider presets
- model presets under each provider
- enabled or disabled state for providers and models
- compatible custom provider endpoints

This is also where the app keeps the distinction between provider configuration and assistant behavior setup.

### A useful mental split

- If you are deciding **how the request runs**, use **Model Presets**.
- If you are deciding **what kind of workspace you want**, use **Assistant Presets**.

## Settings page

The **Settings** page handles app-wide configuration.

It exposes:

- **Theme**
- **Auth Keys**
- **Debug**

### Auth Keys

This is where provider credentials are added.

The settings store persists key metadata locally while protecting secrets through keyring-backed encryption.

### Debug

Debug settings affect how much internal request and response detail is logged or preserved.

Debug options influence:

- backend log level
- raw LLM request and response logging
- content stripping behavior in per message debug details.

Use these carefully when working with sensitive data.

## Built-in content versus user-created content

Across presets, prompts, tools, skills, and assistant presets, the app follows a stable pattern:

- built-in items are shipped with the app
- user-created items live in local app storage
- built-in items are generally treated as read-only definitions
- built-in items can still be enabled or disabled

## How to decide which admin page to use

| Goal                                     | Page                  |
| ---------------------------------------- | --------------------- |
| Reuse a whole working setup              | **Assistant Presets** |
| Reuse message structure                  | **Prompts**           |
| Add or maintain callable capabilities    | **Tools**             |
| Maintain reusable workflow frames        | **Skills**            |
| Change provider or model execution setup | **Model Presets**     |
| Manage keys, theme, and debug            | **Settings**          |
