# Chats, Composer, and Everyday Workflow

The **Chats** page is the main place where FlexiGPT turns reusable setup into actual work.
It combines tabs, local conversation search, the active conversation timeline, model controls, assistant presets, prompt sources, attachments, tools, skills, and the editor for the next message.

## Table of contents <!-- omit from toc -->

- [What the Chats page brings together](#what-the-chats-page-brings-together)
- [A normal workflow](#a-normal-workflow)
- [Composer control bar](#composer-control-bar)
  - [Assistant preset dropdown](#assistant-preset-dropdown)
- [Composer context bar](#composer-context-bar)
- [Composer active chips bar](#composer-active-chips-bar)
  - [Attachments in the composer](#attachments-in-the-composer)
  - [Tools in the composer](#tools-in-the-composer)
  - [Skills in the composer](#skills-in-the-composer)
  - [Sending, running tools, and stopping](#sending-running-tools-and-stopping)
- [Prompt templates in the composer](#prompt-templates-in-the-composer)
- [Reading results](#reading-results)
- [Editing and branching](#editing-and-branching)
- [Search, tabs, and continuity](#search-tabs-and-continuity)
- [When to leave Chats](#when-to-leave-chats)

## What the Chats page brings together

The Chats page coordinates:

- chat tabs
- local conversation search
- conversation restoration
- the message timeline
- streaming responses
- the composer
- assistant preset application
- model and runtime parameter controls
- attachments
- prompt templates
- system prompt sources
- tools and tool outputs
- skills and skill sessions

Most everyday work should start here.

## A normal workflow

A typical workflow is:

1. Open **Chats** or choose a starter card from the home screen.
2. Pick an assistant preset if you want a known workflow shape.
3. Confirm the model preset and provider.
4. Set **Previous user turns** intentionally.
5. Add source context such as files, folders, URLs, or notes.
6. Add prompt templates, tools, web search, or skills only if they help.
7. Send.
8. Inspect the answer, tool calls, citations, token usage, and message details.
9. Adjust one layer at a time.

## Composer control bar

The request control bar sits above the editor and controls how the next turn runs.

It includes:

| Control                      | What it affects                                                                                             |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------- |
| **Assistant**                | Applies a starter recipe for model, instructions, tools, and skills.                                        |
| **Model**                    | Chooses the provider/model preset.                                                                          |
| **Temperature or reasoning** | Controls model style or reasoning behavior where supported.                                                 |
| **Effort/verbosity**         | Controls output verbosity when supported by the model/provider.                                             |
| **Previous user turns**      | Controls how much earlier user context is resent.                                                           |
| **Advanced parameters**      | Streaming, token limits, timeout, cache control, output format, stop sequences, and other model parameters. |

Use the bar when you want to change the workflow or request params, not just the text of the current message.

### Assistant preset dropdown

Assistant presets are starter recipes.
They seed the composer, but they do not lock the conversation.

The assistant preset dropdown can show:

- the selected preset
- whether it is **In sync** or **Modified**
- which sections changed after applying it
- actions to view, reset, reapply, or clear to base

Use **View** to inspect what a preset contributes.

Depending on the preset, the view can show:

- model and advanced parameter values
- instruction template selections
- tool and web-search selections
- enabled skills
- current values when the active preset has been modified

Use **Reapply** or **Reset** when you want preset-managed sections to return to the preset values.

Use **Clear to base** when you want to return to the base assistant preset or fallback selectable preset.

Important expectation:

- if a preset does not define a section, applying it usually leaves that section alone
- if a preset defines instruction templates, those selected prompt sources are replaced by the preset selection
- if a preset defines tools, conversation tools and web-search choices are set from the preset
- if a preset defines skills, enabled/active skill state is set from the preset

## Composer context bar

The composer context bar is the picker area under the editor.

It provides:

- **Attachments**
  - files
  - folders
  - URLs
- **System prompt**
  - model default prompt toggle
  - saved system/developer prompt sources
  - add/fork system prompt
- **Prompts**
  - reusable prompt templates for the current message
- **Tools**
  - attach tools to the draft
- **Skills**
  - enable or clear workflow skills
- **Web search**
  - provider-compatible web search tool selection when available
- **Shortcuts**
  - keyboard shortcut reference
- **Input tips**
  - common behavior notes for prompts, tools, and attachments

## Composer active chips bar

The chips bar appears when the current draft has active context.

It can show:

- conversation tools
- standalone attachments
- folder attachment groups
- attached tools
- pending tool calls
- running tool calls
- failed tool calls
- tool outputs

The chips bar is where you inspect and remove draft context before sending.

### Attachments in the composer

Attachments can be added through the attachment picker or by dropping files.

Supported user-facing inputs include:

- multiple files
- folders
- URLs

Attachment behavior to remember:

- files and URLs are deduplicated by identity
- a single local attachment over `16 MiB` becomes a not-readable/error attachment
- folder selection is limited to `128` files per directory selection in the composer
- overflow folders are shown as skipped notices
- folder chips are UI grouping; restored messages may show flat attachment chips
- each attachment has a mode, such as text, file, image, page content, link as text, or not readable

Review attachment modes before sending sensitive work.

### Tools in the composer

Tools can be made available in several ways:

- selected by an assistant preset
- added as conversation-level tools
- attached to the current draft
- exposed through skills
- selected as provider-compatible web search

Tool states are separate:

| State           | Meaning                                    |
| --------------- | ------------------------------------------ |
| **Tool choice** | Tool is available to the model.            |
| **Tool call**   | Model requested a concrete call.           |
| **Tool output** | FlexiGPT ran the call and captured output. |

When tool calls appear, you can usually:

- run a call
- discard a call
- inspect call details
- retry failed output when possible
- remove tool output before sending it back

If a tool requires user options, the composer blocks sending until required options are valid.

### Skills in the composer

Skills are selected from the Skills menu.

The menu shows:

- enabled count
- active count
- available skills
- bundle-level selection
- individual skill selection
- select all
- clear all

Enabled skills define what the skill session may use.
Active skills are the skills currently active in that session.

When you send with enabled skills, FlexiGPT may create or refresh a skill session and include skill-related context or tool choices in the request.

### Sending, running tools, and stopping

The composer supports these action paths:

| Action                 | Meaning                                                                           |
| ---------------------- | --------------------------------------------------------------------------------- |
| **Send**               | Send the current message and selected context.                                    |
| **Run tools only**     | Execute pending runnable tool calls without sending a new model request.          |
| **Run tools and send** | Execute pending calls, then send the outputs back to the model.                   |
| **Stop**               | Abort an in-flight generation. Partial output already received stays in the chat. |

Guardrails:

- required template variables block send
- missing tool/web-search options block send
- pending runnable tool calls must be run or discarded before normal send
- failed runnable calls must be retried or discarded before send
- if auto-execute is enabled for eligible tool calls, FlexiGPT can run them automatically
- if all observed calls in an auto-execute batch succeed, the outputs may be submitted back automatically

## Prompt templates in the composer

Prompt templates are inserted into the editor.

When you insert a template:

- user blocks become visible draft text
- variables become inline editable pills
- required variables block sending until filled
- system/developer blocks contribute instruction context for the send
- a template toolbar appears so you can edit, flatten, or remove the template

Use **Edit Template** in the toolbar when you want local changes for this inserted instance.
Those local changes do not edit the saved template in the prompt catalog.

Use **Flatten** when you want to convert the template’s visible user content into plain text.
After flattening, the template’s variable behavior and template-derived instruction blocks no longer participate as a template.

## Reading results

The message timeline can show:

- Markdown
- syntax-highlighted code
- Mermaid diagrams
- math
- reasoning content when available
- citations when returned by the provider
- token usage
- debug/message details
- tool calls and outputs
- attachments

Use message details when debugging why a response changed.

## Editing and branching

You can edit an earlier user message.

When editing:

- the selected message is loaded into the composer
- later messages are dropped when you resend
- current conversation tools, web search, and skill state are snapshotted
- canceling the edit restores the previous context

This is a branch-and-replay workflow, not a hidden edit of the old response.

## Search, tabs, and continuity

The Chats workspace supports:

- multiple tabs
- scratch tabs
- local conversation search
- reopening saved conversations
- scroll restoration
- local tab restoration

Tabs are UI workspace state.
Conversation content is stored by the backend.

## When to leave Chats

Stay in Chats for active work.

Leave Chats when you need to maintain reusable building blocks:

| Goal                                  | Page              |
| ------------------------------------- | ----------------- |
| Create or version an assistant preset | Assistant Presets |
| Create or version a prompt template   | Prompts           |
| Add or maintain tools                 | Tools             |
| Add or maintain skills                | Skills            |
| Change providers/models               | Model Presets     |
| Add provider keys or debug settings   | Settings          |
