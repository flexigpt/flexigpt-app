# Privacy, storage, and usage

FlexiGPT is local-first, but it is still a client for remote/local model providers. This page separates what stays on your device from what can leave it.

## What stays local

### Provider secrets

Provider API secrets are stored through the OS keyring integration rather than being written as plain text into exported settings data.

### Conversations and search

Conversation history is stored locally.

That includes:

- conversation data on disk
- local full-text search data
- local restore and reopen behavior

### Other local app data

The app also keeps local data for things such as:

- settings metadata
- chat tab state and UI state
- bundled markdown docs
- files you explicitly export

## What can be sent to a model provider

When you send a message, the selected provider may receive:

- current message text
- selected system prompt content
- selected prompt template output
- earlier user turns included by the history control
- attachments attached to included messages
- model preset values and advanced parameters
- tool choices and tool outputs
- web-search configuration, when applicable
- skill session context, when applicable

FlexiGPT does not proxy billing through its own service. Requests go to the provider behind the API key you configured.

## Attachments deserve extra care

Attachments can contain more than the visible text in the composer.

### Local files

- text-like files can become request context
- images are sent as base64 image attachments
- PDFs may be extracted as text or sent as file content depending on mode and provider support

### Folders

A folder attach can include many files at once, so review what was actually attached rather than assuming the entire directory tree was included.

### URLs

A URL attach may fetch readable page content, image data, PDF content, or file-like content depending on the target. Treat URL attach as sending fetched content, not just sharing a bare link.

## Debug settings can expose sensitive local data

The **Settings** page includes debug options that can make local data more visible.

### Log level

Changes backend log verbosity.

### Raw LLM request and response logging

If enabled, raw request and response payloads can be written to local logs. That may include prompt text, attachment-derived text, and provider responses.

### Disable content stripping

If enabled, per-message details can retain more content instead of hiding or trimming it.

If you turn these options on, treat local logs and debug views as sensitive.

## Practical privacy checklist

Before sending something sensitive, confirm:

- the selected provider key is the one you intended to use
- old history does not need to be resent
- unnecessary attachments are removed
- tool outputs do not contain data you do not want to resend
- raw request or response logging is disabled unless you truly need it

## Storage summary

| Data                      | Where it lives                |
| ------------------------- | ----------------------------- |
| Provider secrets          | OS keyring-backed storage     |
| Settings metadata         | Local app data                |
| Conversations             | Local files                   |
| Conversation search index | Local full-text index         |
| Bundled docs              | Inside the shipped app bundle |

## Final reminder

Local-first does not mean provider calls are local. Always think about the full request payload, not just the text currently visible in the editor.
