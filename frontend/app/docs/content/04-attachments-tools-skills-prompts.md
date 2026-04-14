# Attachments, Tools, Skills, and Prompts

These features help FlexiGPT go beyond plain chat text.

## Table of contents <!-- omit from toc -->

- [Attachments: bring source material into the request](#attachments-bring-source-material-into-the-request)
  - [Why attachments matter](#why-attachments-matter)
  - [Best practices for attachments](#best-practices-for-attachments)
- [Prompt templates and system prompts solve different problems](#prompt-templates-and-system-prompts-solve-different-problems)
  - [Prompt templates](#prompt-templates)
  - [System prompts](#system-prompts)
- [Tools: callable capability inside the conversation](#tools-callable-capability-inside-the-conversation)
- [Tool choice, tool call, and tool output are different things](#tool-choice-tool-call-and-tool-output-are-different-things)
- [Human-in-loop and auto-execute tool flows](#human-in-loop-and-auto-execute-tool-flows)
  - [Human-in-loop](#human-in-loop)
  - [Auto-execute and auto-submit](#auto-execute-and-auto-submit)
- [Web search: provider-dependent capability](#web-search-provider-dependent-capability)
- [Skills: reusable workflow modes](#skills-reusable-workflow-modes)
- [A simple way to choose the right feature](#a-simple-way-to-choose-the-right-feature)
- [Recommended working pattern](#recommended-working-pattern)

## Attachments: bring source material into the request

Attachments can include:

- local files
- folders that expand into file attachments
- images
- PDFs
- URLs

### Why attachments matter

Attachments reduce ambiguity. Good examples include:

- attaching the exact file you want explained
- attaching a focused folder instead of describing it vaguely
- attaching a PDF you want summarized
- attaching a URL when the page content matters more than the bare link

### Best practices for attachments

- prefer a few relevant files over a broad dump
- remove stale attachments before sending again
- remember that older attachments only return when the older user turn is included again
- treat URL attachments as content you may be sending, not just as a hyperlink

## Prompt templates and system prompts solve different problems

### Prompt templates

Use a prompt template when the current request should follow a reusable structure.

Examples:

- implementation briefs
- review formats
- rewrite requests
- recurring team conventions

Prompt templates are request-shaped: they help you structure what you are sending now.

### System prompts

Use system prompts when behavior rules should remain active across turns.

A simple way to remember the difference:

- **Prompt template**: shapes the request you are sending now
- **System prompt**: shapes how the assistant should behave across the conversation

## Tools: callable capability inside the conversation

Tools are managed on the **Tools** page and selected or attached in the composer.

A tool makes a capability available to the model for a conversation. In practice, it gives the model something the app can run. Depending on the tool, execution may happen through local runtime behavior, HTTP-backed behavior, or provider-coupled capability.

## Tool choice, tool call, and tool output are different things

Keep these three steps separate:

- **Tool choice**: you make a tool available to the conversation
- **Tool call**: the model proposes using that tool with specific arguments
- **Tool output**: the result produced after the tool runs

That distinction matters because availability, execution, and continued conversation are related but not identical steps.

## Human-in-loop and auto-execute tool flows

### Human-in-loop

Use manual review when you want tighter control.

The flow is:

1. the model proposes a tool call
2. you review it
3. you choose whether to run it
4. the output can then be submitted back into the conversation

### Auto-execute and auto-submit

Use **auto-execute** when a trusted workflow should continue with less manual interruption.

When an eligible tool call appears and the required arguments are present, FlexiGPT can:

1. run the tool automatically
2. capture the result
3. submit that result back to the model so the conversation continues

This is the app's more automated or agentic mode. It is still bounded by the tools you enabled and the execution mode you chose.

## Web search: provider-dependent capability

Web search is exposed separately from normal local tool runtime.

In practice:

- it is SDK-bound and only appears when the current provider SDK matches a compatible web-search bundle
- changing providers can remove incompatible web-search options, and the composer filters mismatched selections out
- it is most useful when freshness matters, not as a default for every task

Treat web search as a capability of the current model setup rather than as a universal local feature.

## Skills: reusable workflow modes

Skills are managed on the **Skills** page and enabled inside a conversation.

From a user perspective, a skill is best used when you want the model to approach a task in a more consistent way across turns.

Examples:

- review-oriented work
- implementation planning
- refactoring workflows
- structured multi-step task framing

## A simple way to choose the right feature

Use this rule of thumb:

- **Attachment**: when the model needs source material
- **Prompt template**: when the request should follow reusable structure
- **System prompt**: when behavior rules should stay active across turns
- **Tool**: when the model may need execution capability
- **Skill**: when the conversation should follow a reusable workflow mode
- **Web search**: when current external information matters

## Recommended working pattern

A reliable everyday pattern is:

1. choose the right assistant preset and model
2. keep the history window intentional
3. add only the attachments that matter now
4. add tools or skills only if the task truly benefits from them
5. send, inspect, and adjust one layer at a time if the result is weak
