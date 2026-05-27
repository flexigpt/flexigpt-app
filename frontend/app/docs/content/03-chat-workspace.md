# Chat Workspace

The **Chats** page is the main place where FlexiGPT turns reusable setup into work. It brings together tabs, local conversation search, the message timeline, model controls, assistant presets, the composer, streaming responses, and edit/resend flows.

This page is about working in the chat workspace. For the details of attachments, prompt templates, tools, skills, and web search inside the composer, see [Composer Context](/docs?doc=composer-context).

## Table of contents <!-- omit from toc -->

- [Normal chat flow](#normal-chat-flow)
- [Workspace areas](#workspace-areas)
- [Control bar](#control-bar)
- [Assistant preset dropdown](#assistant-preset-dropdown)
- [Composer context entry points](#composer-context-entry-points)
- [Sending and stopping](#sending-and-stopping)
- [Reading results](#reading-results)
- [Editing and branching](#editing-and-branching)
- [Search, tabs, and continuity](#search-tabs-and-continuity)
- [When to leave Chats](#when-to-leave-chats)

## Normal chat flow

1. Open **Chats** or choose a home screen workflow card.
2. Pick an assistant preset if you want a known workflow shape.
3. Confirm the model preset and provider.
4. Set **Previous user turns** intentionally.
5. Add only the context the task needs.
6. Send.
7. Inspect the response, citations, token usage, message details, and tool calls.
8. Adjust one layer at a time.

## Workspace areas

The Chats workspace coordinates:

- chat tabs
- scratch tabs
- local conversation search
- conversation restoration
- active message timeline
- streaming responses
- composer draft state
- model and parameter controls
- assistant preset application
- message editing and replay
- conversation export

Most active work should start here.

## Control bar

The request control bar sits above the editor and controls how the next turn runs.

| Control                      | What it affects                                                                                                   |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **Assistant**                | Applies a starter recipe for starting text, model, instructions, tools, web search, and skills.                   |
| **Model**                    | Chooses the provider/model preset.                                                                                |
| **Temperature or reasoning** | Controls model style or reasoning behavior where supported.                                                       |
| **Effort/verbosity**         | Controls output verbosity where supported.                                                                        |
| **Previous user turns**      | Controls how much earlier user context is resent.                                                                 |
| **Advanced parameters**      | Streaming, token limits, timeout, cache control, output format, stop sequences, and provider-specific parameters. |

Use this bar when changing how the turn runs. Use the editor when changing what you are asking.

## Assistant preset dropdown

Assistant presets are starter recipes. They seed the composer, but they do not lock the conversation.

In Chats, the assistant preset dropdown can show:

- the selected preset
- whether it is **In sync** or **Modified**
- which preset-managed sections changed
- actions to view, reset, reapply, or clear to base

Use **View** to inspect what a preset contributes to the current chat. The detailed rules for preset contents and versioning live in [Reusable Catalogs](/docs?doc=reusable-catalogs#assistant-presets).

Expected behavior:

- if a preset defines starting text, applying it can seed the composer draft
- if a preset defines a model, applying it selects that model preset
- if a preset defines instruction templates, it selects those saved instruction sources
- if a preset defines tools or web search, it applies those selections
- if a preset defines skills, it enables those skills and may mark some active
- if a preset has no opinion about a section, applying it usually leaves that section alone

## Composer context entry points

The composer bottom bar is where you add message context:

- **Attachments**
  - files, folders, images, PDFs, URLs
- **System prompt**
  - model default prompt toggle and saved system/developer prompt sources
- **Prompts**
  - reusable prompt templates for the current message
- **Tools**
  - tool choices for the draft or conversation
- **Skills**
  - workflow modes for the current session
- **Web search**
  - provider-compatible web search selection
- **Shortcuts and tips**
  - input behavior reference

The active chips bar shows what is currently attached or pending. Review chips before sending important work.

## Sending and stopping

Common actions:

| Action                 | Meaning                                                                           |
| ---------------------- | --------------------------------------------------------------------------------- |
| **Send**               | Send the current message and selected context.                                    |
| **Run tools only**     | Execute pending runnable tool calls without sending a new model request.          |
| **Run tools and send** | Execute pending calls, then send the outputs back to the model.                   |
| **Stop**               | Abort an in-flight generation. Partial output already received stays in the chat. |

The composer can block send when:

- required prompt template variables are missing
- tool or web-search options are incomplete
- pending runnable tool calls must be run or discarded
- failed runnable calls must be retried or discarded
- current request has no usable content

## Reading results

The timeline can show:

- Markdown
- syntax-highlighted code
- Mermaid diagrams
- math
- reasoning content when available
- citations when returned by the provider
- token usage
- message/request details
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

Tabs are UI workspace state. Conversation content is stored by the backend.

## When to leave Chats

Stay in Chats for active work.

Leave Chats when maintaining reusable building blocks:

| Goal                                  | Page              |
| ------------------------------------- | ----------------- |
| Create or version an assistant preset | Assistant Presets |
| Create or version a prompt template   | Prompts           |
| Add or maintain tool definitions      | Tools             |
| Add or maintain skills                | Skills            |
| Change providers or model presets     | Model Presets     |
| Add provider keys or debug settings   | Settings          |
