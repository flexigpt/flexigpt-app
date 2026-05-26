# Recipes and Starter Workflows

These recipes are outcome-based starting points.
Use the home screen workflow cards or apply the matching assistant preset in Chats.

## Table of contents <!-- omit from toc -->

- [Code review](#code-review)
- [Explain this file](#explain-this-file)
- [Generate docs](#generate-docs)
- [Research brief](#research-brief)
- [Model comparison](#model-comparison)
- [Architecture review](#architecture-review)
- [Bug investigation](#bug-investigation)
- [Prompt template creation](#prompt-template-creation)
- [Safe tool execution](#safe-tool-execution)

## Code review

Use when you want correctness, maintainability, security, reliability, and test-risk feedback.

Suggested setup:

- Assistant preset: **Reviewing Code**
- Context: changed files, diff, PR description, failing tests, or relevant logs
- Tools: read-oriented file and text tools if you want workspace-aware review

Starter prompt:

```text
Review the attached code or diff for correctness, security, reliability, maintainability, and test gaps.
Focus on concrete issues. Rank findings by severity and include narrow fixes.
```

## Explain this file

Use when you want to understand an unfamiliar file or module.

Suggested setup:

- Assistant preset: **Local Reader** or **Local Developer**
- Context: one file first; add nearby files only when needed

Starter prompt:

```text
Explain the attached file. Cover its purpose, main flows, important types/functions, dependencies, and risky or surprising behavior.
End with a short "what to inspect next" list.
```

## Generate docs

Use when you want README text, usage docs, API docs, or internal implementation notes.

Suggested setup:

- Assistant preset: **Docs Writer** or **Local Reader**
- Context: source files, existing README/docs, examples, screenshots, or command output

Starter prompt:

```text
Generate clear user-facing documentation from the attached context.
Include overview, setup, common workflow, examples, troubleshooting, and limitations.
Keep claims grounded in the provided files.
```

## Research brief

Use when you want a digest of notes, URLs, PDFs, or copied source material.

Suggested setup:

- Assistant preset: **Research Brief Writer**
- Optional: provider-supported web search if current information matters

Starter prompt:

```text
Turn the attached research material into a brief with: key findings, evidence, disagreements or uncertainty, recommendations, and next questions.
Separate observed facts from interpretation.
```

## Model comparison

Use when you want to compare answer quality, cost trade-offs, reasoning style, or provider behavior on the same task.

Suggested setup:

- Use the same conversation prompt, attachments, and **Previous user turns** setting.
- Send once with one model preset.
- Switch only the model preset.
- Send the same request again in another tab or branch.

Starter prompt:

```text
Answer this task using the attached context.
Then list assumptions, uncertainty, and the strongest reason your answer might be wrong.
```

## Architecture review

Use when you want a system-structure recommendation, boundary review, or design risk assessment.

Suggested setup:

- Assistant preset: **Designing System Architecture**
- Context: README, architecture docs, code structure, API contracts, deployment notes, or constraints

Starter prompt:

```text
Review the attached architecture or code structure.
Identify current boundaries, risks, coupling, unclear ownership, and the simplest viable improvements.
Return a concise recommendation with trade-offs and revisit triggers.
```

## Bug investigation

Use when you have logs, stack traces, failing tests, or a confusing behavior report.

Suggested setup:

- Assistant preset: **Bug Investigator** or **Spec Driven Development**
- Context: stack trace, logs, failing command, relevant code, recent diff

Starter prompt:

```text
Investigate this bug from the attached logs, stack trace, and code.
Identify the likely root cause, evidence, missing evidence, narrow fix, and verification steps.
Do not assume uninspected files behave correctly.
```

## Prompt template creation

Use when a request format is repeated often.

Suggested setup:

- Page: **Prompts**
- Create either a generic prompt template or an instructions-only system prompt template

Starter prompt:

```text
Turn this repeated task into a reusable FlexiGPT prompt template.
Include variables, defaults, required fields, and an example filled-in request.
```

## Safe tool execution

Before enabling tools broadly:

- prefer manual review for new tools
- keep auto-execute off for tools that write files, call network endpoints, or run shell/script commands
- inspect tool arguments before running them
- inspect tool outputs before resubmitting them
- disable tools that are not needed for the current workflow
