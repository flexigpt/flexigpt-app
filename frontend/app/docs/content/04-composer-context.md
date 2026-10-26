# Composer Context

The composer is where you decide what the next Run includes. This page covers instruction sources, templates, attachments, Workspaces, tools, Skills, MCP context, and provider-compatible web search.

This page is about using context in Chats. To create or maintain reusable definitions, see [Reusable Catalogs](/docs?doc=reusable-catalogs).

## Table of contents <!-- omit from toc -->

- [Quick chooser](#quick-chooser)
- [Templates and instruction sources](#templates-and-instruction-sources)
- [Attachments](#attachments)
- [Tools](#tools)
  - [Reviewing tool calls](#reviewing-tool-calls)
  - [Error results](#error-results)
- [Web search](#web-search)
- [Skills](#skills)
- [MCP servers](#mcp-servers)
- [Workspaces](#workspaces)
- [MCP servers](#mcp-servers-1)
- [Active chips bar](#active-chips-bar)
- [Recommended pattern](#recommended-pattern)

## Quick chooser

| Need                                                  | Use                                                  |
| ----------------------------------------------------- | ---------------------------------------------------- |
| Bring exact source material into a request            | Attachment                                           |
| Reuse current-message structure                       | User-message template                                |
| Add reusable system-style guidance                    | Model default or selected instruction source         |
| Let the model request a native or provider capability | Tool                                                 |
| Use server-discovered context                         | MCP                                                  |
| Reuse repository context                              | Workspace                                            |
| Need recent web information                           | Web search, if compatible with the selected provider |
| Enable reusable instruction Skill state               | Skill                                                |

## Templates and instruction sources

Template-style skills and instruction sources solve different parts of a request.

- A template-style skill renders reusable text into the current draft. Templates with arguments or resources ask for configuration first.
- After insertion, the rendered content is ordinary plain composer text. There is no active template to keep selected or collapse later.
- When a template has verified resource files, you can choose which of those files to attach with the rendered text.
- Edit or remove inserted template text as you would manually written text. Doing so does not change selected instruction sources, attachments, Workspaces, or tools.

Instruction sources shape the system instructions sent with the request. They are independent of template text:

- the selected model default is added first when enabled
- selected instruction-only skill sources are then appended in the order you selected them
- sources are combined; a later source does not silently replace an earlier one

Use the **Skills** picker to select or remove instruction sources. Clearing a source changes the next request's instructions but does not alter text previously inserted by a template.

## Attachments

Attachments are message-scoped source material.

Use them for:

- explaining a file
- reviewing a diff
- summarizing a PDF
- comparing documents
- analyzing a folder
- including a URL’s page content
- providing an image to a multimodal model

The composer can add:

- local files
- folders
- images
- PDFs and other readable documents
- URLs

Attachment behavior:

- files and folders can be dropped into the active Chat composer
- a drop made outside Chats is queued while FlexiGPT opens Chats
- attachments added to the current draft are available to the model when that message is sent
- after sending, an attachment remains with that user message in conversation history
- an older attachment is available again only when its older user turn is included by **Previous user turns**
- deleting an attachment from the current draft does not rewrite older messages
- folder chips are UI grouping; restored messages may show flat attachment chips
- files and URLs are deduplicated by identity
- a single local attachment over `16 MiB` becomes not-readable/error context
- folder selection keeps a bounded set of files actively attached and shows additional candidates as not added

Attachment modes can include:

| Mode             | Meaning                                                    |
| ---------------- | ---------------------------------------------------------- |
| **Text**         | Extract readable text and include it in the message.       |
| **File**         | Send as a file attachment when the provider supports it.   |
| **Image**        | Send as an image attachment when the provider supports it. |
| **Page content** | Fetch or extract page text from a URL.                     |
| **Link as text** | Send only the URL as text.                                 |
| **Image as URL** | Send the URL as an image reference.                        |
| **File as URL**  | Send the URL as a file reference.                          |
| **Not readable** | The app could not safely read or attach the content.       |

Before sending sensitive or large context, check attachment modes.

## Tools

Tools let the model request a capability during a Run.

The Tools picker can show:

- native FlexiGPT tools implemented by the app
- provider-compatible SDK tools, including provider web search where supported

MCP-discovered tools are not ordinary catalog tools. Select them through the
separate **MCP** picker.

FlexiGPT does not provide a user-created HTTP or arbitrary custom-tool definition flow.

Keep these states separate:

| State           | Meaning                                             |
| --------------- | --------------------------------------------------- |
| **Tool choice** | The tool is available to the model.                 |
| **Tool call**   | The model requested a specific call with arguments. |
| **Tool output** | FlexiGPT ran the call and captured the result.      |

Tools can be made available by:

- an Agent
- conversation-level tool selection
- per-message attached tool selection
- skills
- provider-compatible web search

### Reviewing tool calls

Use the **Tools** picker to configure native and provider-compatible tools before sending. The selections used by a Run are shown with that message so you can inspect the history later.

Review selected tools before every important Run. Remove a tool that no longer belongs in the task instead of relying on an earlier task's narrow scope.

Manual tool flow:

1. choose or attach tools
2. send a message
3. inspect any proposed tool call
4. run, retry, skip, or submit the failure as an error result
5. inspect the output
6. continue the Run when the output is useful

Use manual review first for an unfamiliar built-in tool or any tool whose arguments or side effects you have not verified.

Auto-execute can run eligible trusted calls with less interruption. Use it only for trusted, low-risk workflows.

Some tools define required user options. If required options are missing, the composer blocks send. Use the tool options editor to provide valid JSON arguments.

### Error results

If a runnable tool call fails before producing output, retry or discard the call before sending. When an executed tool returns an error result, the composer marks the output separately. Retry a supported result, discard it, or deliberately send the error output to the model as context.

Error output is not retried automatically. Treat it as data: inspect it and decide whether it belongs in the next request.

## Web search

Web search is handled separately from ordinary local tools.

Expectations:

- it is provider/SDK dependent
- it appears only when compatible with the selected provider
- switching providers can clear incompatible web-search choices
- web-search options can block sending when required fields are missing
- web search may send query context to the selected provider path

Use web search when freshness matters. Do not enable it by default for private or local-only work.

## Skills

Skills have two user-facing insert behaviors:

- **User-message templates** render text into the draft. They do not remain selected after insertion.
- **Instruction Skills** can be enabled for the Session. A simple instruction Skill without arguments or resources can also be rendered as an explicit instruction source.

The composer tracks:

| State              | Meaning                                               |
| ------------------ | ----------------------------------------------------- |
| **Enabled skills** | Skills allowed in this conversation/session.          |
| **Active skills**  | Enabled instruction Skills preloaded as active state. |

An Agent or Workspace can contribute Skill selections. Inspect the selected Skills panel after loading either one.

Selected instruction-only sources are independent of template text and are assembled with the selected model default as described in [Templates and instruction sources](#templates-and-instruction-sources).

Agents can enable Skills and mark some ready to use immediately.

## MCP servers

## Workspaces

Use a Workspace when repository or folder context should be reusable.

The Workspace picker allows one Workspace at a time. It resolves the current
Workspace plan before sending, then lets you choose its available instruction
files, reference files, instruction Skills, and MCP server references.

If a saved Workspace item changed or became unavailable, resolve that warning
in the Workspace picker before sending. Manage directory discovery and
repository-wide enablement on [Workspaces](/docs?doc=workspaces).

## MCP servers

The `MCP` chip turns model context protocol server discovery into per-turn context. It does not edit the server catalog.

Use it when a configured server should contribute tools, resources, resource templates, prompts, or server instructions to the next message.

The usual flow is:

1. Configure and connect the server on [MCP Servers](/docs?doc=mcp-servers).
2. Open the `MCP` chip in Chats.
3. Select one or more enabled servers.
4. Choose `all`, `selected`, or `none` for tool exposure.
5. Add only the resources, prompts, and arguments that the task needs.
6. Keep `Include instructions` on only when the server instructions are useful.

A few practical notes:

- app-only tools stay visible, but are not exposed to the model
- required arguments block send until they are filled
- if the server produces app context updates, clear them when you no longer need them
- an MCP App result does not become model context until you explicitly approve and stage an update
- refresh discovery on the server page when the server contents change

## Active chips bar

The active chips bar shows live draft context such as:

- conversation tools
- standalone attachments
- folder attachment groups
- attached tools
- pending tool calls
- running tool calls
- failed tool calls
- selected Workspace content
- staged MCP App context
- tool outputs

Use it to inspect and remove context before sending.

## Recommended pattern

1. choose an Agent if you want a starter workflow
2. replace any Agent opening-text placeholder with your task
3. confirm provider/model
4. keep **Previous user turns** small and intentional
5. attach only the source material that matters
6. add tools only when execution is useful
7. use manual tool review first
8. enable skills when a workflow mode helps
9. inspect output, citations, tool outputs, and message details
10. adjust one layer at a time
