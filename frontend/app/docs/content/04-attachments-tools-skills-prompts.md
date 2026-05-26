# Attachments, Tools, Skills, and Prompts

Attachments, tools, skills, and prompts are the building blocks that turn FlexiGPT from plain chat into a repeatable AI workspace.

This page explains when to use each one, how they behave in the composer, and what to check before sending.

## Table of contents <!-- omit from toc -->

- [Quick chooser](#quick-chooser)
- [Attachments](#attachments)
  - [Attachment types](#attachment-types)
  - [Attachment modes](#attachment-modes)
  - [File and folder limits](#file-and-folder-limits)
  - [Attachment expectations](#attachment-expectations)
- [System prompts](#system-prompts)
- [Prompt templates](#prompt-templates)
  - [Template kinds](#template-kinds)
  - [Variables and resolved state](#variables-and-resolved-state)
  - [Templates inside the composer](#templates-inside-the-composer)
- [Tools](#tools)
  - [Tool choice, tool call, and tool output](#tool-choice-tool-call-and-tool-output)
  - [Conversation tools versus attached tools](#conversation-tools-versus-attached-tools)
  - [Manual tool flow](#manual-tool-flow)
  - [Auto-execute flow](#auto-execute-flow)
  - [Tool options and blocking behavior](#tool-options-and-blocking-behavior)
  - [Web search](#web-search)
- [Skills](#skills)
  - [Enabled versus active skills](#enabled-versus-active-skills)
  - [Skill sessions](#skill-sessions)
  - [Skills in assistant presets](#skills-in-assistant-presets)
- [Recommended working pattern](#recommended-working-pattern)

## Quick chooser

| Need                                        | Use                                                  |
| ------------------------------------------- | ---------------------------------------------------- |
| Bring source material into a request        | Attachment                                           |
| Keep behavior rules active across turns     | System prompt                                        |
| Reuse request structure                     | Prompt template                                      |
| Let the model ask FlexiGPT to run something | Tool                                                 |
| Use a reusable workflow mode                | Skill                                                |
| Need recent web information                 | Web search, if compatible with the selected provider |

## Attachments

Attachments are message-scoped context.

Use them when the model needs exact source material instead of a vague description.

Good attachment use cases:

- explain a file
- review a diff
- summarize a PDF
- compare two documents
- analyze a folder
- include a URL’s page content
- provide an image to a multimodal model

### Attachment types

The composer can add:

- local files
- folders
- images
- PDFs and other readable documents
- URLs

The frontend asks the backend to normalize these inputs.
The frontend does not directly materialize raw filesystem content itself.

### Attachment modes

Attachments can have different modes depending on type and provider support.

Common modes include:

| Mode             | Meaning                                                    |
| ---------------- | ---------------------------------------------------------- |
| **Text**         | Extract readable text and include it in the message.       |
| **File**         | Send as a file attachment when the provider supports it.   |
| **Image**        | Send as an image attachment when the provider supports it. |
| **Page content** | Fetch/extract page text from a URL.                        |
| **Link as text** | Send only the URL as text.                                 |
| **Image as URL** | Send the URL as an image reference.                        |
| **File as URL**  | Send the URL as a file reference.                          |
| **Not readable** | The app could not safely read or attach the content.       |

If more than one mode is available, the attachment chip lets you change it before sending.

### File and folder limits

Composer-side limits include:

- maximum single attachment size: `16 MiB`
- maximum files per directory selection: `128`

If a file is too large or unreadable, it is shown as a not-readable/error attachment.

If a folder has too many files, FlexiGPT attaches the allowed subset and shows skipped folder notices.

### Attachment expectations

Important details:

- attachments are deduplicated by identity, such as path or URL
- folder chips are UI grouping, not a permanent stored folder object
- restored messages may show folder-selected files as flat attachment chips
- attachments on older turns return when those older turns are included by **Previous user turns**
- URL attachments can fetch and send page-derived content depending on mode

Before sending sensitive work, check:

- selected provider/model
- attachment list
- attachment modes
- previous user turns
- debug logging state

## System prompts

System prompts are durable behavior instructions.

Use them for rules like:

- answer in a specific style
- follow a review rubric
- avoid unsupported claims
- ask clarifying questions before acting
- produce structured output

In Chats, system prompt sources can include:

- the selected model preset’s default prompt
- saved system/developer prompt templates
- a restored previous conversation prompt
- system/developer blocks from inserted prompt templates

The system prompt dropdown shows active source count and lets you:

- toggle model default prompt inclusion
- select saved system prompts
- add a new saved system prompt
- fork an existing saved prompt
- clear selected prompt sources

Add/Fork from the system prompt dropdown creates an instructions-only prompt template in a custom prompt bundle.
This quick Add/Fork flow only accepts fully resolved prompt text; unresolved `{{variable}}` placeholders must be removed or handled in the Prompt Bundles page.

## Prompt templates

Prompt templates are reusable message structures.

Use them when you repeat a request format, such as:

- code review checklist
- implementation brief
- bug investigation template
- documentation generation prompt
- research synthesis format
- rewrite request

### Template kinds

FlexiGPT derives template kind from block roles:

| Kind                  | How it is derived                       | Where it is useful                                                 |
| --------------------- | --------------------------------------- | ------------------------------------------------------------------ |
| **Instructions Only** | Every block is `system` or `developer`. | Saved system prompt sources and assistant preset instruction refs. |
| **Generic**           | At least one block is `user`.           | Composer prompt picker for current-message structure.              |

Assistant presets can only select instruction templates that are:

- enabled
- in an enabled bundle
- instructions-only
- resolved

### Variables and resolved state

Templates can use placeholders such as:

```text
Review this code for {{review_focus}}.
```

Variables must be declared in the template.

Variables can have:

- type
- required flag
- source
- description
- default
- static value
- enum values

A template is **resolved** when every placeholder can be filled locally by a static value or default.

A user variable without a default can still be useful in a generic template, but the composer will require a value before sending if it is required.

### Templates inside the composer

When inserted into the composer:

- user blocks become visible draft text
- variables become editable inline pills
- required variables block sending
- system/developer blocks contribute instruction context for the send
- the template toolbar lets you edit, flatten, or remove that inserted instance

Local edits in the template toolbar affect only the current inserted instance.
They do not change the saved prompt template.

## Tools

Tools let the model ask FlexiGPT to run a capability.

Depending on the tool, execution may be:

- local Go-backed behavior
- HTTP-backed behavior
- provider/SDK-backed behavior
- skill-backed behavior

### Tool choice, tool call, and tool output

Keep these separate:

| State           | Meaning                                        |
| --------------- | ---------------------------------------------- |
| **Tool choice** | The tool is available to the model.            |
| **Tool call**   | The model requested a specific tool run.       |
| **Tool output** | FlexiGPT ran the tool and captured the result. |

The composer shows these as different chips.

### Conversation tools versus attached tools

There are two ordinary tool selection paths:

| Tool selection        | Behavior                             |
| --------------------- | ------------------------------------ |
| **Conversation tool** | Persists across turns until removed. |
| **Attached tool**     | Belongs to the current draft.        |

Assistant presets usually apply tools as conversation-level selections.
Prompting a tool into the current draft can make it available for that message.

### Manual tool flow

Manual review is the safest default.

The flow is:

1. choose or attach tools
2. send a message
3. the model proposes a tool call
4. inspect the call
5. run or discard it
6. inspect the output
7. send the output back if useful

Use this for new tools, network tools, file tools, shell/script workflows, or any workflow where execution risk matters.

### Auto-execute flow

Auto-execute lets trusted tool calls run with less interruption.

When an eligible tool call appears and required args are present, FlexiGPT can:

1. run the call automatically
2. capture the output
3. submit successful outputs back to the model when the auto-execute batch is complete

Auto-execute applies to runnable function/custom-style tools and selected skill tool flows.
Restored historical tool calls are shown for review but are suppressed from automatic re-execution.

Use auto-execute only for trusted, low-risk workflows.

### Tool options and blocking behavior

Some tools define a user-args schema.
If required options are missing, the composer blocks send.

You may see:

- `Args: OK`
- `Args: Optional`
- `Args: N Missing`

Use the tool options editor to enter a JSON object.
The editor can show:

- required keys
- optional keys
- example options
- the JSON schema

### Web search

Web search is handled separately from ordinary local tool runtime.

Important expectations:

- it is SDK/provider-family dependent
- it only appears when compatible with the selected provider SDK
- switching providers can clear incompatible web-search choices
- the chat runtime currently restores a single active web-search configuration
- web-search options can block sending when required fields are missing

Use web search when freshness matters.
Do not enable it by default for private or local-only work.

## Skills

Skills are reusable workflow modes.

They can help the model work with a more structured approach across turns.

Examples:

- review mode
- implementation planning
- refactoring workflow
- documentation workflow
- multi-step investigation

### Enabled versus active skills

The composer tracks two skill states:

| State              | Meaning                                               |
| ------------------ | ----------------------------------------------------- |
| **Enabled skills** | Skills allowed in this conversation/session.          |
| **Active skills**  | Skills currently active in the skill runtime session. |

The skills menu shows both counts.

### Skill sessions

When enabled skills are present, FlexiGPT can create or refresh a skill session.
That session lets skill-aware behavior participate in the request.

Skill state can be saved with messages and restored later.

### Skills in assistant presets

Assistant presets can include skill selections.

Each selection can optionally be marked **preload as active**.
When the preset applies:

- selected skills become enabled
- preload-as-active skills become active
- the composer may ensure a skill session when needed

User-created skills are filesystem skills.
Embedded filesystem skills are built-in and generally read-only in the UI.

## Recommended working pattern

A reliable pattern is:

1. choose an assistant preset if you want a starter workflow
2. confirm the provider/model
3. keep **Previous user turns** small and intentional
4. add only the attachments that matter
5. add prompt templates for repeated structure
6. add tools only when execution is useful
7. use manual tool review first
8. enable skills when you want a reusable workflow mode
9. inspect output, citations, tool outputs, and message details
10. adjust one layer at a time
