# Composer Context

The composer is where you decide what the next message includes. This page covers how to use attachments, prompt templates, system prompts, tools, skills, and web search while composing a chat message.

This page is about using context in Chats. To create or maintain reusable definitions, see [Reusable Catalogs](/docs?doc=reusable-catalogs).

## Table of contents <!-- omit from toc -->

- [Quick chooser](#quick-chooser)
- [Attachments](#attachments)
- [System prompts](#system-prompts)
- [Prompt templates](#prompt-templates)
- [Tools](#tools)
- [Web search](#web-search)
- [Skills](#skills)
- [MCP servers](#mcp-servers)
- [Active chips bar](#active-chips-bar)
- [Recommended pattern](#recommended-pattern)

## Quick chooser

| Need                                        | Use                                                  |
| ------------------------------------------- | ---------------------------------------------------- |
| Bring exact source material into a request  | Attachment                                           |
| Keep behavior rules active across turns     | System prompt                                        |
| Reuse current-message structure             | Prompt template                                      |
| Let the model ask FlexiGPT to run something | Tool                                                 |
| Use server-discovered context               | MCP                                                  |
| Need recent web information                 | Web search, if compatible with the selected provider |
| Use a reusable workflow mode                | Skill                                                |

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

- attachments added to the current draft belong to the message you send
- older attachments return only when their older user turn is included by **Previous user turns**
- deleting an attachment from the current draft does not rewrite older messages
- folder chips are UI grouping; restored messages may show flat attachment chips
- files and URLs are deduplicated by identity
- a single local attachment over `16 MiB` becomes not-readable/error context
- folder selection is limited to `128` files per directory selection

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

## System prompts

System prompts are durable behavior instructions.

Use them for rules like:

- answer in a specific style
- follow a review rubric
- ask clarifying questions before acting
- avoid unsupported claims
- produce structured output

In Chats, system prompt sources can include:

- the selected model preset’s default prompt, when included
- selected saved system/developer prompts
- system/developer blocks from inserted prompt templates
- a restored previous conversation prompt

The system prompt dropdown lets you:

- toggle model default prompt inclusion
- select saved system prompts
- add a new resolved system prompt
- fork an existing saved prompt
- clear selected prompt sources

Add/Fork from this menu is for simple resolved instruction text. Use the **Prompts** page for full variable support and versioning.

## Prompt templates

Prompt templates are reusable message structures.

Use them for repeated formats such as:

- code review checklist
- implementation brief
- bug investigation template
- documentation generation prompt
- research synthesis format
- rewrite request

When inserted into the composer:

- user blocks become visible draft text
- variables become editable inline pills
- required variables block sending until filled
- system/developer blocks contribute instruction context for the send
- the template toolbar lets you edit, flatten, or remove the inserted instance

Local edits in the toolbar affect only the inserted instance. They do not edit the saved template.

Use **Flatten** when you want to convert visible template content into plain text. After flattening, variable behavior and template-derived instruction blocks no longer participate as a template.

## Tools

Tools let the model ask FlexiGPT to run a capability.

Keep these states separate:

| State           | Meaning                                             |
| --------------- | --------------------------------------------------- |
| **Tool choice** | The tool is available to the model.                 |
| **Tool call**   | The model requested a specific call with arguments. |
| **Tool output** | FlexiGPT ran the call and captured the result.      |

Tools can be made available by:

- an assistant preset
- conversation-level tool selection
- per-message attached tool selection
- skills
- provider-compatible web search

Manual tool flow:

1. choose or attach tools
2. send a message
3. inspect any proposed tool call
4. run or discard the call
5. inspect the output
6. send the output back if useful

Use manual review first for new tools, network tools, file tools, shell/script tools, or any tool with execution risk.

Auto-execute can run eligible trusted calls with less interruption. Use it only for trusted, low-risk workflows.

Some tools define required user options. If required options are missing, the composer blocks send. Use the tool options editor to provide valid JSON arguments.

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

Skills are reusable workflow modes. They are not just saved text.

The composer tracks:

| State              | Meaning                                               |
| ------------------ | ----------------------------------------------------- |
| **Enabled skills** | Skills allowed in this conversation/session.          |
| **Active skills**  | Skills currently active in the skill runtime session. |

When you send with enabled skills, FlexiGPT may create or refresh a skill session and include skill-related context or tool choices in the request.

Assistant presets can enable skills and mark some as **preload as active**.

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
- tool outputs

Use it to inspect and remove context before sending.

## Recommended pattern

1. choose an assistant preset if you want a starter workflow
2. replace any preset starting-text placeholder with your task
3. confirm provider/model
4. keep **Previous user turns** small and intentional
5. attach only the source material that matters
6. use prompt templates for repeated structure
7. add tools only when execution is useful
8. use manual tool review first
9. enable skills when a workflow mode helps
10. inspect output, citations, tool outputs, and message details
11. adjust one layer at a time
