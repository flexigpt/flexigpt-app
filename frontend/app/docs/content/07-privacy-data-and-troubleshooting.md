# Privacy, Data, and Troubleshooting

FlexiGPT is local-first: conversations, workflow catalogs, settings metadata, and most app state live on your machine. Requests still go to whichever provider or compatible endpoint you choose for a chat turn.

This page combines the trust boundary, local storage model, backup/reset behavior, logs, and common troubleshooting checks.

## Table of contents <!-- omit from toc -->

- [Trust boundaries](#trust-boundaries)
- [What stays local by default](#what-stays-local-by-default)
- [What can leave the machine](#what-can-leave-the-machine)
- [Provider proxying and billing](#provider-proxying-and-billing)
- [Local data categories](#local-data-categories)
  - [Local UI workspace state](#local-ui-workspace-state)
  - [Secrets and key metadata](#secrets-and-key-metadata)
  - [Conversation history and search](#conversation-history-and-search)
  - [Catalog and app configuration data](#catalog-and-app-configuration-data)
  - [Bundled app data](#bundled-app-data)
- [Storage locations](#storage-locations)
- [Backup, restore, export, and reset](#backup-restore-export-and-reset)
- [Attachments, URLs, tools, and skills](#attachments-urls-tools-and-skills)
- [Debug logs](#debug-logs)
- [Troubleshooting checklist](#troubleshooting-checklist)
  - [If you cannot send a request](#if-you-cannot-send-a-request)
  - [If the answer is weak](#if-the-answer-is-weak)
  - [If a tool flow seems stuck](#if-a-tool-flow-seems-stuck)
  - [If you need more visibility](#if-you-need-more-visibility)
- [Sensitive-work checklist](#sensitive-work-checklist)

## Trust boundaries

FlexiGPT is not a hosted chat service. The app runs locally, stores its main data locally, and uses the provider keys and endpoints you configure.

The main trust decision happens when you choose a provider, model, compatible endpoint, tool, skill, attachment, or debug setting for a workflow.

## What stays local by default

Stored locally by default:

- conversations and local history search data
- models and provider metadata
- Agent Plugins and Agent files
- Workspace setup
- tool definitions
- skill definitions and hydrated built-in skill files when needed
- app settings metadata
- debug logs
- local UI workspace state such as tabs and scroll position

Provider API key secrets are protected through the OS keyring rather than normal exported settings. The app can display key metadata, but it does not display the secret value.

## What can leave the machine

When you send a chat request to a remote provider, the provider may receive:

- the current user message
- selected prior turns based on **Previous user turns**
- selected model defaults and instruction-source content
- enabled Skill context or user-message template output
- attachments included in the request
- selected tool definitions or tool outputs
- selected Workspace context
- selected MCP server tools, resources, prompts, server instructions, or staged MCP App context
- web-search configuration when supported by the provider
- model preset parameters and provider-specific request options

Local-first does not mean every inference request is local. If you choose a remote model provider, the assembled request is sent to that provider.

If you choose a local endpoint, review the endpoint origin and confirm it is actually local before treating the request as local-only.
For local LLMs, also confirm that the local runtime is not configured to proxy requests to a remote provider and that the selected model preset belongs to the local provider you intended to use.

## Provider proxying and billing

FlexiGPT does not proxy normal LLM requests through a FlexiGPT-hosted service. It uses the provider keys and endpoints you configure locally.

Billing, quotas, rate limits, retention behavior, and provider-side logging are controlled by the provider account and endpoint you use.

## Local data categories

### Local UI workspace state

The frontend may remember UI-only state in your UI profile, such as:

- chat tab state
- selected tab
- scroll position
- startup theme choice

This is separate from durable app data.

### Secrets and key metadata

Provider secrets are protected through the OS keyring. The app can show credential metadata such as key name or whether a value exists, but not the secret itself.

### Conversation history and search

Conversations are stored locally, and local search is available for conversation history.

On-device capabilities include:

- stored conversation history
- local history search
- reopening saved conversations into tabs

### Catalog and app configuration data

The app also stores local data for:

- settings metadata
- providers and models
- tool, Skill, Agent, MCP server, and Workspace configuration
- Agent Plugins and Agent files
- Workspace setup
- skill plugins and skills
- bundled docs shipped inside the app
- logs and local indexes

### Bundled app data

Built-in providers, models, tools, Skills, Agents, and docs ship with the app. Your local changes and your own entries are stored separately from bundled defaults.

## Storage locations

The exact app-data location depends on the platform and packaging method.
Use the platform's application-data facilities or package documentation when
you need to locate local data.

Do not treat UI-only state, such as restored tabs or scroll positions, as a
backup format for durable conversation or catalog data.

For installation-specific package notes, use the release documentation that
matches your platform.

## Backup, restore, export, and reset

Local data control surfaces include:

- conversation export from the chat workspace
- Agent YAML export from **Agents**
- manual backup by copying the `flexigpt` app-data folder while the app is closed
- local source-file backup for Skills and Workspaces that you manage outside the app

Close FlexiGPT before taking a file-level backup or following support guidance
for a reset. Provider credentials may need separate OS-level backup, restore,
or removal.

## Attachments, URLs, tools, and skills

Attachments are request context, not just UI labels. Depending on type and provider support, FlexiGPT may transform files, folders, images, PDFs, or URLs into model-ready content blocks.

Review attachments before sending, especially when:

- a folder expands into many files
- an old user turn with attachments is included again through **Previous user turns**
- a URL is fetched and converted into text context
- a binary or large file falls back to display text or partial extraction

URL attachments can cause content to be fetched and included in the model request. Treat URL attachments as potentially sending page-derived content to the provider.

Tools and skills can add execution capability or runtime context.

FlexiGPT can expose native tools and provider-compatible tools. MCP server
tools are selected independently through MCP context. It does not support
user-defined HTTP or arbitrary custom tool definitions.

Safety points:

- inspect each selected tool's description, arguments, and expected side effects
- use manual review for tools that can modify local data or perform external actions
- skill-aware workflows can add skill session context and skill tool choices to the request
- auto-execute should only be used for trusted tools and low-risk workflows
- tool outputs can be sent back to the model in later turns

Prefer manual tool review for unfamiliar built-in tools.

## Debug logs

The title-bar **Settings** menu can increase local logging.

Be careful with:

- raw LLM request and response logging
- disabling content stripping for per-message details
- sharing logs in issues or support requests

When raw request/response logging is enabled, logs can contain prompts, attachments, outputs, and other sensitive content.

## Troubleshooting checklist

### If you cannot send a request

Check:

- provider key was added in **Models**
- selected provider is enabled in **Models**
- selected model is enabled
- custom endpoint origin/path are correct
- local server is running, if applicable
- current tool or web-search choice is not blocked by missing arguments or configuration
- current request still has real content after removing stale attachments or tool outputs

### If the answer is weak

Adjust one layer at a time:

1. simplify the request
2. reduce stale history with **Previous user turns**
3. attach the exact source file or URL that matters
4. switch to a better model preset
5. refine the relevant skill selections or instruction-only skill sources
6. enable tools or skills only if the task truly needs them

### If a tool flow seems stuck

Look for:

- the tool call is waiting for manual execution
- auto-execute is off
- required user arguments are missing
- the selected tool is not right for the current provider or task
- a failed runnable call must be retried or discarded before send

### If you need more visibility

Use:

- message details
- token usage
- citations
- tool details and outputs
- title-bar logging/debug controls only when needed

## Sensitive-work checklist

Before sending sensitive work:

- confirm the selected provider and model are intended
- keep **Previous user turns** no larger than necessary
- remove stale attachments
- review attachment modes
- review selected tools, skills, web search, and tool outputs
- avoid auto-execute for untrusted tools
- keep raw debug logging off unless actively diagnosing a problem
- for local-only work, confirm the provider origin points to a local endpoint you control
- for local-only work, prefer a forked local provider with a clear provider ID and harmless placeholder key
- confirm local model servers, routers, or gateways are not proxying the request to a remote service
- close FlexiGPT before copying or deleting the app-data folder
- remember API key secrets live in the OS keyring, not only in app files
