# Privacy, Storage, and Troubleshooting

FlexiGPT is local-first, but requests still go to the provider you choose. This page explains what stays on your device, what can leave it, and what to check when something feels wrong.

## Table of contents <!-- omit from toc -->

- [What stays local](#what-stays-local)
  - [Browser-local workspace state](#browser-local-workspace-state)
  - [Provider secret handling](#provider-secret-handling)
  - [Conversation history and search](#conversation-history-and-search)
  - [Catalog and app configuration data](#catalog-and-app-configuration-data)
  - [Storage locations](#storage-locations)
- [What can be sent to a provider](#what-can-be-sent-to-a-provider)
- [Attachments need deliberate handling](#attachments-need-deliberate-handling)
  - [Local files and folders](#local-files-and-folders)
  - [URLs](#urls)
- [Tool outputs can be resent too](#tool-outputs-can-be-resent-too)
- [Debug settings can expose more than normal use](#debug-settings-can-expose-more-than-normal-use)
- [Troubleshooting checklist](#troubleshooting-checklist)
  - [If you cannot send a request](#if-you-cannot-send-a-request)
  - [If the answer is weak](#if-the-answer-is-weak)
  - [If a tool flow seems stuck](#if-a-tool-flow-seems-stuck)
  - [If you need more visibility](#if-you-need-more-visibility)
- [Practical privacy checklist](#practical-privacy-checklist)
- [Final reminder](#final-reminder)

## What stays local

### Browser-local workspace state

The frontend may remember UI-only state in your browser profile, such as:

- chat tab state
- selected tab
- scroll position
- startup theme choice

This is separate from durable app data.

### Provider secret handling

Provider secrets are protected through the OS keyring.

The app can show credential metadata such as key name or whether a value exists, but not the secret itself.

### Conversation history and search

Conversations are stored locally, and local search is available for conversation history.

That means these capabilities stay on-device:

- stored conversation history
- local history search
- reopening saved conversations into tabs

### Catalog and app configuration data

The app also stores local data for areas such as:

- settings metadata
- provider and model presets
- prompt bundles and templates
- tool bundles and tool definitions
- assistant presets
- skill bundles and skills
- bundled docs shipped inside the app

Most of this is local configuration and catalog data.

### Storage locations

FlexiGPT stores its local data under the app's XDG data-home location, using a `flexigpt` app-data folder.

For current packaged builds, that typically means:

- macOS: `~/Library/Containers/io.github.flexigpt.client/Data/Library/Application Support/flexigpt/`
- Linux Flatpak: `~/.var/app/io.github.flexigpt.client/data/flexigpt/`

## What can be sent to a provider

When you send a request, the selected provider may receive some or all of the following:

- the current user message
- system prompt content
- prompt template output
- earlier user turns included by the history setting
- attachments belonging to included messages
- model preset and advanced parameter values
- selected tool choices and tool outputs
- web-search configuration when supported by the provider
- skill session context and skill-related behavior when enabled

FlexiGPT does not proxy billing through its own service. Billing and limits come from the provider account behind the configured key.

## Attachments need deliberate handling

Attachments are often the biggest privacy multiplier in a request.

### Local files and folders

Depending on the attachment type, the app may turn them into:

- text content
- image content
- file content

Folder attachments can expand into multiple files, so treat them as a batch of source material rather than as a single symbolic pointer.

### URLs

A URL attachment is not the same as leaving a plain link in your message.

Depending on the attachment type and provider support, the app may fetch and transform URL content before it becomes request context.

## Tool outputs can be resent too

Tool-assisted workflows can add more context to later requests.

Before sending sensitive work, remember that tool outputs can also become part of the continuing conversation, whether they were added after manual review or after an auto-executed step.

## Debug settings can expose more than normal use

Debug settings can affect:

- backend log verbosity
- whether raw LLM request and response payloads are logged
- whether content stripping is disabled for debug details

If you enable raw request and response logging, treat your local logs as sensitive.

## Troubleshooting checklist

### If you cannot send a request

Check these first:

- a provider key was added in **Settings**
- the selected provider is enabled in **Model Presets**
- the selected model preset is enabled
- the current tool or web-search choice is not blocked by missing arguments or missing configuration
- the current request still has real content after removing stale attachments or tool outputs

### If the answer is weak

Adjust one layer at a time:

1. simplify the request
2. reduce stale history with **Previous user turns**
3. attach the exact source file or URL that matters
4. switch to a better model preset
5. refine system prompt sources or prompt templates
6. enable tools or skills only if the task truly needs them

### If a tool flow seems stuck

Look for these causes:

- the tool call is waiting for manual execution
- auto-execute is off
- required user arguments are missing
- the selected tool is not the right one for the current provider or task

### If you need more visibility

Use these inspection surfaces before changing too many things at once:

- message details
- token usage
- citations
- tool details and outputs
- debug settings, only when needed

## Practical privacy checklist

Before sending sensitive work, confirm:

- the selected provider key is the intended one
- the history window is not larger than necessary
- stale attachments are removed
- tool outputs do not include data you would not want resent
- debug logging is off unless you are actively diagnosing a problem

## Final reminder

Local-first does not mean provider requests stay local. The safest habit is to think about the full request payload, not only the text currently visible in the editor.
