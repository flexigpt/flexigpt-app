# Privacy, Storage, and Troubleshooting

FlexiGPT is local-first, but it still sends requests to the provider you choose. This page explains what stays local, what can leave the device, and what to check when something feels wrong.

## What stays local

### Provider secret handling

Provider secrets are managed through the backend settings store using keyring-backed encryption.

The normal settings response sent to the frontend returns metadata such as key type, key name, and whether a key is non-empty, but not the secret itself. The secret remains strictly backend only.

### Conversation history and search

Conversations are stored locally. A local full-text index for conversation search is also provided.

That means these capabilities stay on-device:

- stored conversation history
- local history search
- reopening a saved conversation into a tab

### Catalog and app configuration data

The app also stores local configuration for areas such as:

- settings metadata
- provider and model presets
- prompt bundles and templates
- tool bundles and tool definitions
- assistant presets
- skill bundles and skills
- bundled docs shipped inside the app

Most of these stores are JSON-backed local data, with SQLite used where a local index or overrides for builtins are needed.

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
- skill session context and skill-derived prompt/tool behavior when enabled

FlexiGPT does not proxy billing through its own service. The provider behind the configured key handles billing and limits.

## Attachments need deliberate handling

Attachments are often the largest privacy multiplier in a request.

### Local files and folders

Depending on attachment type, the backend may turn them into:

- text content
- image content
- file content

Folder attachments can expand into multiple files, so treat them as a batch of context rather than as a single symbolic pointer.

### URLs

A URL attachment is not the same as leaving a plain link in your typed message. The attachment flow can fetch and transform URL content before it becomes request context.

The user can choose to fetch the page and send it as text. Or an image URL or PDF url can be fetched and sent as base64 files when supported by the provider. Or the base URL can be sent as explicit URL link, when the provider supports URL fetch and process.

## Debug settings can expose more than normal usage

The debug settings can affect:

- backend log verbosity
- whether raw LLM request and response payloads are logged
- whether content stripping is disabled for debug details

If you enable raw request and response logging, treat your local logs as sensitive because they may contain prompt text, attachment-derived content, and provider responses.

## Troubleshooting checklist

### If you cannot send a request

Check these first:

- you added a provider key in **Settings**
- the selected provider is enabled on **Model Presets**
- the selected model preset is enabled
- the current tool or web-search choice is not blocked by missing arguments
- the request still has real content after you removed stale attachments or tool outputs

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
- auto-execution is off
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

Local-first does not mean provider requests are local. The safest habit is to think about the full request payload, not only the text currently visible in the editor.
