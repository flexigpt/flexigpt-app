# Getting better results

In the LLM world, good results usually come from choosing the right model, the right context, and the right workflow controls, not from writing a longer prompt.

## 1. Start with the right model or assistant preset

### Choose the right model preset

- Use faster or lighter presets for exploration and iteration.
- Use stronger or reasoning-capable presets for multi-step analysis.
- Use lower-creativity settings when you want factual, structured, or repeatable output.

### Use assistant presets when the task repeats

Assistant presets are better than manually rebuilding the same setup every time. Use them when the task needs a repeatable combination of:

- model choice
- instructions
- tools
- skills

## 2. Control how much history you resend

The **Previous user turns** control is one of the biggest quality knobs in the app.

Use it intentionally:

- reduce it when the thread has drifted or become noisy
- increase it when the current turn depends on earlier decisions
- set it to `0` when you want a clean turn inside the same conversation
- use `All` only when the full thread is still relevant

Because attachments and tool outputs live on messages, this control changes more than text history. It also affects how much older context is resent.

## 3. Add focused context, not more context

Attachments help most when they remove ambiguity.

Good habits:

- attach the exact file, page, or snippet that matters
- prefer a few relevant files over a large folder dump
- verify how PDFs or URLs are being treated
- remove stale attachments before sending

Be careful with:

- large folders that may hit current attach limits
- oversized files that are rejected
- unreadable files or pages that do not produce useful context

## 4. Use instructions and templates for consistency

### System prompts

Use system prompts when you want durable behavior, such as:

- response style rules
- coding standards
- documentation conventions
- review checklists
- output constraints

### Prompt templates

Use prompt templates when you want reusable structure inside the message body, such as:

- bug report requests
- implementation briefs
- architecture reviews
- rewrite or editing formats

If a template requires variables, fill them before sending.

## 5. Use tools, skills, and web search only when they solve a real problem

### Tools

Add tools when the task needs capabilities beyond plain text generation.

Examples:

- file or text manipulation
- structured utility flows
- explicit tool-call based workflows

Use auto-execute selectively:

- turn it on for trusted, repeatable workflows where you want the model and tool loop to move faster
- leave it off when you want to inspect each tool call before it runs
- be especially cautious with tools that can modify files, run commands, or produce expensive external actions

### Skills

Enable skills when you want the model to follow a reusable working style, such as review, refactoring, or spec-driven work.

### Web search

Use web search when the answer depends on current external information. Leave it off when the task is self-contained.

## 6. Tune advanced parameters last

The advanced parameters modal is useful, but it is rarely the first thing to change.

Reach for it when you need to:

- increase timeout
- constrain output length
- force structured output
- adjust stop sequences

Most of the time, better model choice and better context matter more.

## 7. Recover quickly when a conversation goes off track

If a thread is going bad, do not keep piling on corrective text.

Usually the better fix is to change one of these first:

- the model preset
- the history window
- the attachments
- the tool or skill selection
- the system instructions

If the original request was wrong, edit and resend that earlier user message instead of continuing a bad branch.

## Troubleshooting order

If an answer is weak, try this order:

1. simplify the request
2. reduce or reset unrelated history
3. add the missing file or source
4. switch to a better model preset
5. add or refine system instructions
6. use a better assistant preset for the task
7. inspect message details, tool outputs, or citations

## Quick request checklist

Before an important request, confirm:

- Is the selected model right for this task?
- Is the history window too small or too large?
- Are the right attachments included?
- Are the right tools or skills enabled?
- Are the instructions helping instead of conflicting?
- Does the output format need to be constrained?
