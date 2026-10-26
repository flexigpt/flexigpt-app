# Chat Workspace

The **Chats** page is where reusable setup becomes an active Session and Run. It brings together tabs, saved conversation search, the timeline, model controls, Agents, Workspaces, the composer, runtime waits, and edit/resend flows.

This page is about working in the chat workspace. For the details of attachments, tools, skills, and web search inside the composer, see [Composer Context](/docs?doc=composer-context).

## Table of contents <!-- omit from toc -->

- [Normal chat flow](#normal-chat-flow)
- [Workspace areas](#workspace-areas)
- [Control bar](#control-bar)
- [Agent menu](#agent-menu)
- [Composer context entry points](#composer-context-entry-points)
- [Sending and stopping](#sending-and-stopping)
- [Reading results](#reading-results)
- [MCP App results](#mcp-app-results)
- [Editing and branching](#editing-and-branching)
- [Search, tabs, and continuity](#search-tabs-and-continuity)
- [When to leave Chats](#when-to-leave-chats)

## Normal chat flow

1. Open **Chats** or choose a home screen workflow card.
2. Choose an Agent if you want a known workflow shape.
3. Confirm the model and provider.
4. Set **Previous user turns** intentionally.
5. Add only the context the task needs.
6. Send and review any preparation warning, approval request, or runtime wait.
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
- Agent application
- message editing and replay
- conversation export

Most active work should start here.

## Control bar

The request control bar sits above the editor and controls how the next turn runs.

| Control                      | What it affects                                                                                                   |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **Agent**                    | Loads a starter setup for model, instructions, opening text, tools, Skills, and connected services.               |
| **Model**                    | Chooses the provider/model configuration.                                                                         |
| **Temperature or reasoning** | Controls sampling or reasoning behavior where supported by the selected model.                                    |
| **Verbosity or effort**      | Controls output behavior where supported by the selected provider/model.                                          |
| **Previous user turns**      | Controls how much earlier user context is resent.                                                                 |
| **Advanced parameters**      | Streaming, token limits, timeout, cache control, output format, stop sequences, and provider-specific parameters. |

Use this bar when changing how the turn runs. Use the editor when changing what you are asking.

## Agent menu

Agents are starter recipes. Loading one seeds the chat, but does not lock the conversation.

In Chats, the **Agent** menu lets you:

- choose an available Agent
- inspect its setup and diagnostics
- review the draft and context it adds before sending

Use the [Agents](/docs?doc=agents) page to manage Plugins, import or export Agent files, and enable or disable Agents.

Depending on its contents, loading an Agent can:

- select a model
- add instructions to the chat
- seed editable opening text when the composer is empty
- make tools available to the model
- enable Skills, with some ready to use immediately
- select connected services or prompt you to finish local setup
- leave any area it does not define unchanged

## Composer context entry points

The composer bottom bar is where you add message context:

- **Attachments**
  - files, folders, images, PDFs, URLs
- **Workspace**
  - one repository or folder setup and the specific content selected from it
- **Tools**
  - native FlexiGPT tools and provider-compatible tools for the Run
- **MCP**
  - connected MCP servers and their discovered tools, resources, prompts, and server instructions
- **Skills**
  - instruction Skills, active Skill state, and user-message templates
- **Web search**
  - provider-compatible web search selection
- **Keyboard shortcuts**
  - chat and composer actions

The active chips bar shows what is currently attached or pending. Review chips before sending important work.

## Sending and stopping

Common actions:

| Action                     | Meaning                                                                           |
| -------------------------- | --------------------------------------------------------------------------------- |
| **Send**                   | Send the current message and selected context.                                    |
| **Run tools only**         | Execute pending runnable tool calls without sending a new model request.          |
| **Run tools and continue** | Execute pending calls, then continue the active Run with its results.             |
| **Stop**                   | Abort an in-flight generation. Partial output already received stays in the chat. |

The composer can block send when:

- tool or web-search options are incomplete
- a selected MCP prompt or resource template needs an argument
- a selected Workspace is still resolving or is no longer available
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
- recognized unified diff blocks with in-line apply controls for local file changes

Use message details when debugging why a response changed.

## MCP App results

Some MCP tool results can include an inline MCP App view.

- An App result is not automatically added as model context.
- The App can request a draft message, a staged model-context update, an external link, or an app-initiated tool call.
- FlexiGPT applies the server App policy and asks for approval where required.
- Staged App context is shown in the composer and can be removed before the next continuation or send.

Use the MCP server policy and discovery details to decide whether an App is appropriate for the workflow.

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

Stay in Chats for active work and per-Run choices.

Leave Chats when changing reusable definitions. The short ownership map is in
[Manage Reusable Setup](/docs?doc=reusable-catalogs); the detailed pages are
[Agents](/docs?doc=agents), [Workspaces](/docs?doc=workspaces),
[MCP Servers](/docs?doc=mcp-servers), [Skills](/docs?doc=reusable-catalogs#skills),
and [Providers and Models](/docs?doc=providers-and-models).
