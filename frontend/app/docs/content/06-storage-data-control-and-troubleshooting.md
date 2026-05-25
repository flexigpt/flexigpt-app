# Storage, Data Control, and Troubleshooting

FlexiGPT is local-first: conversations, workflow catalogs, provider configuration, and most app state live on your machine.
Requests still go to the provider you choose.

This page is the operational reference for local data, storage locations, backup/reset/export, and common troubleshooting.
For the trust boundary, provider-request behavior, attachment risks, tool risks, logs, and safe-send checklist, see **Security, Privacy, and Trust Model**.

## Table of contents <!-- omit from toc -->

- [Local data categories](#local-data-categories)
  - [Browser-local workspace state](#browser-local-workspace-state)
  - [Secrets and key metadata](#secrets-and-key-metadata)
  - [Conversation history and search](#conversation-history-and-search)
  - [Catalog and app configuration data](#catalog-and-app-configuration-data)
  - [Bundled app data](#bundled-app-data)
  - [Storage locations](#storage-locations)
- [Backup, restore, export, and reset](#backup-restore-export-and-reset)
- [Request context quick check](#request-context-quick-check)
- [Troubleshooting checklist](#troubleshooting-checklist)
  - [If you cannot send a request](#if-you-cannot-send-a-request)
  - [If the answer is weak](#if-the-answer-is-weak)
  - [If a tool flow seems stuck](#if-a-tool-flow-seems-stuck)
  - [If you need more visibility](#if-you-need-more-visibility)
- [Operational checklist](#operational-checklist)

## Local data categories

### Browser-local workspace state

The frontend may remember UI-only state in your browser profile, such as:

- chat tab state
- selected tab
- scroll position
- startup theme choice

This is separate from durable app data.

### Secrets and key metadata

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
- logs and local indexes

Most of this is local configuration and workflow catalog data.

### Bundled app data

Built-in providers, model presets, prompts, tools, skills, assistant presets, and docs ship with the app.
Your local changes and user-created entries are stored separately from those bundled defaults.

### Storage locations

FlexiGPT stores its local data under the app's XDG data-home location, using a `flexigpt` app-data folder.
These get created at first launch of FlexiGPT.

For current packaged builds, that typically means:

- macOS: `~/Library/Containers/io.github.flexigpt.client/Data/Library/Application Support/flexigpt/`
- Linux Flatpak: `~/.var/app/io.github.flexigpt.client/data/flexigpt/`
- Windows: `%LOCALAPPDATA%\flexigpt\`, usually `C:\Users\<username>\AppData\Local\flexigpt\`. Manually finding the folder:
  - Open **File Explorer** -> Go to `C:\Users\<username>\` -> Enable hidden folders as **View** -> **Show** -> **Hidden items** -> Open `AppData\Local\flexigpt`.
  - OR if `AppData` is not visible or you are unsure which local profile path Windows is using, open PowerShell and run `$env:LOCALAPPDATA`.

You can back up FlexiGPT local data by copying the `flexigpt` app-data folder while the app is closed. API key secret values are protected through the OS keyring and should be re-added or restored through the OS credential mechanism if needed.

## Backup, restore, export, and reset

Local data control surfaces include:

- conversation export from the chat workspace
- settings export from **Settings**
- model preset export from **Model Presets**
- manual backup by copying the `flexigpt` app-data folder while the app is closed
- full reset by closing FlexiGPT and removing the local `flexigpt` app-data folder

The app-data folder contains local conversations, catalogs, settings metadata, indexes, overlays, and logs.
Provider secrets stored in the OS keyring may need separate OS-level backup, restore, or removal.

## Request context quick check

This page focuses on local storage and troubleshooting, but one reminder matters during normal use:

- local-first does not mean every request stays local.

Before sending, quickly check:

- the selected provider and model
- **Previous user turns**
- current attachments
- selected tools, tool outputs, skills, and web-search options
- debug logging state if you are diagnosing a problem

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

## Operational checklist

Before backup, reset, troubleshooting, or support sharing:

- close FlexiGPT before copying or deleting the app-data folder
- remember that API key secrets live in the OS keyring, not only in app files
- export conversations or settings when you want portable JSON snapshots
- keep raw debug logging off unless you are actively diagnosing a problem
- avoid sharing logs until you have checked them for sensitive content
