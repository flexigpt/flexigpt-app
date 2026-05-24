# Security and Privacy Model

FlexiGPT is local-first: conversations, workflow catalogs, settings metadata, and app data are stored on your machine.
Requests still go to whichever provider or compatible endpoint you choose for a chat turn.

This page is the trust checklist for what stays local, what can leave the device, and which features need deliberate handling.

## What stays local

Stored locally:

- conversations and local history search data
- model presets and provider metadata
- assistant presets
- prompt templates
- tool definitions
- skill definitions and hydrated built-in skill files when needed
- app settings metadata
- debug logs
- browser-local workspace state such as tabs and scroll position

Provider API key secrets are protected through the OS keyring.
The app can display key metadata, but it does not display the secret value.

## What can leave the machine

When you send a chat request to a remote provider, the provider may receive:

- the current user message
- selected prior turns based on the history setting
- system prompt content
- prompt template output
- attachments included in the request
- selected tool definitions or tool outputs
- web-search configuration when supported by the provider
- skill-session context when skills are enabled
- model preset parameters and provider-specific request options

Local-first does not mean every inference request is local.
If you choose a remote model provider, the assembled request is sent to that provider.

## Provider proxying and billing

FlexiGPT does not proxy normal LLM requests through a FlexiGPT-hosted service.
It uses the provider keys and endpoints you configure locally.

Billing, quotas, rate limits, retention behavior, and provider-side logging are controlled by the provider account and endpoint you use.

## Storage locations

FlexiGPT stores app data under the app's XDG data-home location in a `flexigpt` folder.

Typical packaged-build locations:

- macOS: `~/Library/Containers/io.github.flexigpt.client/Data/Library/Application Support/flexigpt/`
- Linux Flatpak: `~/.var/app/io.github.flexigpt.client/data/flexigpt/`
- Windows: `%LOCALAPPDATA%\flexigpt\`

Close the app before copying the folder for backup.
OS-keyring secrets may need separate OS-level backup or re-entry after restore.

## Attachments

Attachments are request context, not just UI labels.

Depending on attachment type and provider support, FlexiGPT may transform files, folders, images, PDFs, or URLs into model-ready content blocks.

Review attachments before sending, especially when:

- a folder expands into many files
- an old user turn with attachments is included again through the history setting
- a URL is fetched and converted into text context
- a binary or large file falls back to display text or partial extraction

## URL fetching

URL attachments can cause content to be fetched and included in the model request.

Treat URL attachments as potentially sending page-derived content to the provider.
If a URL points to private or authenticated content, make sure you understand what the backend can access and what will be included in the request.

## Tools and auto-execute

Tools can add execution capability to a chat.

Important safety points:

- tool definitions can describe local, HTTP, SDK, or provider-side capabilities
- HTTP tools can call network endpoints
- file-system tools can read or write local paths when exposed and selected
- shell or script execution tools should be treated as high risk
- auto-execute should only be used for trusted tools and low-risk workflows
- tool outputs can be sent back to the model in later turns

Prefer manual tool review for new, imported, or user-created tools until you trust the behavior.

## Debug logs

Debug settings can increase local logging.

Be careful with:

- raw LLM request and response logging
- disabling content stripping for per-message details
- sharing logs in issues or support requests

When raw request/response logging is enabled, logs can contain prompts, attachments, outputs, and other sensitive content.

## Deleting and exporting data

Current local data control surfaces include:

- conversation export from the chat workspace
- settings export from Settings
- model preset export from Model Presets
- local app-data folder backup by copying the `flexigpt` data directory while the app is closed

For a full reset, close FlexiGPT and remove the local `flexigpt` app-data folder.
This removes local conversations, catalogs, settings metadata, indexes, overlays, and logs.
Provider secrets stored in the OS keyring may need to be removed separately through OS credential management.

## Practical checklist before sending sensitive work

- Confirm the selected provider and model are intended.
- Keep **Previous user turns** no larger than necessary.
- Remove stale attachments.
- Review tool outputs before resending them.
- Avoid auto-execute for untrusted tools.
- Keep raw debug logging off unless actively diagnosing a problem.
- Back up local data by copying the app-data folder while FlexiGPT is closed.
