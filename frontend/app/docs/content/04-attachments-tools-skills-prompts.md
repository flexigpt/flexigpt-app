# Attachments, Tools, Skills, and Prompts

These features are where FlexiGPT moves beyond plain chat text.

## Attachments: bring the right source material into the turn

The composer can attach more than one kind of context. Attachments can include:

- local files
- folders that expand into file attachments
- images
- PDFs
- URLs

### Why attachments matter

Attachments reduce ambiguity. Good uses include:

- attaching the exact file you want explained
- attaching a focused folder instead of describing it vaguely
- attaching a PDF you want summarized
- attaching a URL when the page content matters more than the bare link

### What the app does with attachments

The attachment pipeline normalizes attachments into content blocks that can become:

- text content
- image content
- file content

That means the practical effect of an attachment depends on its type and on provider support.

### Best practices for attachments

- prefer a few relevant files over a broad dump
- remove stale attachments before sending again
- remember that older attachments only return when the older user turn is included again
- treat URL attachments as fetched content, not just a hyperlink reference

## Tools: explicit callable capability

Tools are managed on the **Tools** page and attached in the composer.

### What a tool does in the workflow

A tool is a capability that can be made available to the model for a conversation.

Depending on the tool definition, execution can happen through:

- local Go-backed runtime
- HTTP-backed runtime
- provider-side SDK behavior for specific tool categories

### Conversation tools versus tool calls

There are two different ideas to keep separate:

- **Tool choice**: you make a tool available to the conversation
- **Tool call**: the model decides to call that tool with specific arguments

The runtime then decides whether the tool is:

- waiting for manual action
- auto-executable
- blocked because required arguments are still missing

### Auto-execute and manual review

The composer tool runtime supports both styles.

- Use **manual** when you want to inspect a tool call before it runs.
- Use **auto-execute** when the workflow is trusted enough to let the tool loop continue automatically.

When an auto-executable call is produced and the required arguments are satisfied, FlexiGPT can execute it and continue the tool-assisted flow without requiring a separate manual send for each step i.e auto-submit.

### Tool outputs become context

After execution, tool outputs can become part of what the next request sees.

This is why tool-assisted conversations feel different from plain chat: the model is no longer working only from your typed text.

## Web search: provider-dependent, not a generic local tool

Web search is exposed separately from normal local tool runtime.

In practice:

- it only appears when compatible with the current provider setup
- changing providers can remove incompatible web-search options
- it is useful when freshness matters, not as a default for every task

Because it is provider-coupled, web search should be treated as a request capability of the current model setup rather than as a universal local feature.

## Skills: reusable workflow frames

Skills are managed on the **Skills** page and enabled inside a conversation. The apps skill runtime supports:

- skill sessions
- runtime skill discovery through available/active skills prompt instructions
- skill tools such as load, unload, and read-resource behavior

From a user perspective, skills are best used when you want the model to approach a task in a consistent way across turns.

Examples of when skills make sense:

- review-oriented work
- implementation planning
- refactoring workflows
- structured multi-step task framing

## Prompts: reusable request structure

Prompt templates are managed on the **Prompts** page and inserted from the composer.

They are useful when you want repeatable request structure, such as:

- implementation briefs
- review formats
- rewrite requests
- architecture writeups
- team-specific output conventions

Prompt templates are different from system prompts:

- **Prompt templates** shape the current request content
- **System prompts** shape the assistant's persistent instruction context for the chat setup

## A simple way to choose the right feature

Use this quick rule of thumb:

- **Attachment**: when the model needs source material
- **Prompt template**: when the request should follow reusable structure
- **System prompt**: when behavior rules should stay active across turns
- **Tool**: when the model may need execution capability
- **Skill**: when the conversation should follow a reusable working mode
- **Web search**: when current external information matters

## Recommended working pattern

A reliable everyday pattern is:

1. choose the right assistant preset and model
2. keep the history window intentional
3. add only the attachments that matter now
4. add tools or skills only if the task truly benefits from them
5. send, inspect, and adjust one layer at a time if the result is weak
