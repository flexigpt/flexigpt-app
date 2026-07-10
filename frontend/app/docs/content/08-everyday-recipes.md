# Everyday Recipes

These recipes are outcome-based. They focus on work you want done by the model.

For setup tasks such as OpenRouter, local models, creating assistant presets, tools, or skills, see [Setup Recipes](/docs?doc=setup-recipes).

## Table of contents <!-- omit from toc -->

- [How to use these recipes](#how-to-use-these-recipes)
- [Develop a feature](#develop-a-feature)
- [Analyze a file](#analyze-a-file)
- [Review code](#review-code)
- [Investigate a bug](#investigate-a-bug)
- [Review architecture](#review-architecture)
- [Generate documentation](#generate-documentation)
- [Create a research brief](#create-a-research-brief)
- [Compare models](#compare-models)
- [Safe tool execution checklist](#safe-tool-execution-checklist)

## How to use these recipes

For reliable results:

- change one layer at a time
- keep attachments focused
- keep **Previous user turns** intentional
- inspect assistant preset details before assuming what it does
- when a workflow card or assistant preset pre-fills starting text, replace the placeholder line (between <>) with your real task
- start tools in manual mode
- use local providers only after confirming the endpoint is actually local

## Develop a feature

Use this when you want FlexiGPT to inspect a repo, scope a bounded change, edit code, and verify the result.

Suggested setup:

- Home starter: **Develop a Feature**
- Assistant preset: **Spec Driven Development**
- Context: repo path, changed files, issue text, requirements, screenshots, failing tests, or design notes
- Tools: read/write/shell capable preset, with write and shell calls reviewed manually
- Previous user turns: usually `0` or `1`

Steps:

1. Open the home screen.
2. Choose **Develop a Feature**.
3. Replace the prefilled placeholder with a repo path and the feature request.
4. Add any issue text, acceptance criteria, screenshots, or relevant files.
5. Send the draft.
6. Review the proposed spec or scope if the assistant asks for confirmation.
7. Let it implement the confirmed scope.
8. Review write and shell tool calls before running them.
9. Inspect verification results and remaining gaps.

Equivalent prompt if starting from blank:

    Develop this feature in `/absolute/path/to/repo`:
    <describe the change>

    First inspect relevant files, write a concise spec, ask for confirmation if scope or behavior is unclear, then implement and verify with focused tests or checks.

## Analyze a file

Use this when you want to understand an unfamiliar file.

Suggested setup:

- Assistant preset: Local Reader or similar reader preset
- Context: one file first
- Previous user turns: usually `0` or `1`

Steps:

1. Open **Chats**.
2. Apply a reader-style assistant preset.
3. Attach one file.
4. Confirm attachment mode is readable.
5. Send a focused request.

Equivalent prompt if starting from blank:

    Explain the attached file.
    Cover its purpose, main flows, important types/functions, dependencies, and risky or surprising behavior.
    End with a short "what to inspect next" list.

## Review code

Use this for correctness, maintainability, security, reliability, and test-risk feedback.

Suggested setup:

- Assistant preset: Reviewing Code
- Context: changed files, diff, PR description, tests, logs
- Tools: read-only tools if useful
- Previous user turns: small and intentional

Equivalent prompt if starting from blank:

    Review the attached code or diff for correctness, security, reliability, maintainability, and test gaps.
    Focus on concrete issues.
    Rank findings by severity and include narrow fixes.

Good output should include:

- high-impact findings first
- exact evidence
- minimal fixes
- test suggestions
- uncertainty or missing context

If the result includes a unified diff you want to apply locally, see [Unified Diff Apply](/docs?doc=unified-diff-apply).

## Investigate a bug

Use this when you have logs, stack traces, failing tests, or confusing behavior.

Suggested setup:

- Assistant preset: Bug Investigator or Spec Driven Development
- Context: stack trace, logs, failing command, relevant code, recent diff
- Tools: manual first

Equivalent prompt if starting from blank:

    Investigate this bug from the attached logs, stack trace, and code.
    Identify the likely root cause, evidence, missing evidence, narrow fix, and verification steps.
    Do not assume uninspected files behave correctly.

## Review architecture

Use this for system boundaries, coupling, ownership, or design risk.

Suggested setup:

- Assistant preset: Designing System Architecture
- Context: README, architecture docs, code map, API contracts, constraints

Equivalent prompt if starting from blank:

    Review the attached architecture or code structure.
    Identify current boundaries, risks, coupling, unclear ownership, and the simplest viable improvements.
    Return a concise recommendation with trade-offs and revisit triggers.

## Generate documentation

Use this for README text, usage docs, API docs, or internal guides.

Suggested setup:

- Assistant preset: Docs Writer or Local Reader
- Context: source files, existing docs, examples, screenshots, command output

Equivalent prompt if starting from blank:

    Generate clear user-facing documentation from the attached context.
    Include overview, setup, common workflow, examples, troubleshooting, and limitations.
    Keep claims grounded in the provided files.

## Create a research brief

Use this for notes, URLs, PDFs, or copied source material.

Suggested setup:

- Assistant preset: Research Brief Writer
- Optional: provider-supported web search if current information matters

Equivalent prompt if starting from blank:

    Turn the attached research material into a brief with:
    - key findings
    - evidence
    - disagreements or uncertainty
    - recommendations
    - next questions

    Separate observed facts from interpretation.

## Compare models

Use this when you want to compare answer quality, latency, reasoning style, or provider behavior.

Goal:

- Run the same task through different model presets while keeping everything else the same.

Steps:

1. Open **Chats**.
2. Start a fresh chat or duplicate the task in another tab.
3. Choose an assistant preset if needed.
4. Set **Previous user turns** to a fixed value.
5. Add the same attachments.
6. Add the same tools, skills, and web-search setting.
7. Send the task with model A.
8. Open another tab or branch the task.
9. Change only the **Model** dropdown to model B.
10. Send the same task.
11. Compare the results.

Keep constant:

- assistant preset
- draft text
- attachments and attachment modes
- previous user turns
- tool selections
- selected skills and active skill session state
- web-search setting

Change only:

- model preset

Equivalent prompt if starting from blank:

    Answer this task using only the attached context.
    Then list assumptions, uncertainty, and the strongest reason your answer might be wrong.

Compare on:

- correctness
- completeness
- clarity
- citation quality
- latency
- token usage
- tool behavior
- ability to follow constraints
- cost from your provider account

Common mistake:

- Do not compare one model with extra context and another without it. That measures context differences, not model quality.

## Safe tool execution checklist

Before enabling tools broadly:

- prefer manual review for new tools
- inspect tool arguments
- inspect tool outputs
- keep auto-execute off for tools that write files, call network endpoints, or run shell/script commands
- do not enable tools that are not needed
- retry or discard failed tool calls before sending
- treat tool error outputs as data you must choose to send deliberately
- use auto-execute only for trusted, low-risk workflows
