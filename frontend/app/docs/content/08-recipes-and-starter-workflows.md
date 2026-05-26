# Recipes and Starter Workflows

These recipes are outcome-based flows.

Use the home screen workflow cards when available, or apply the matching assistant preset in Chats.

## Table of contents <!-- omit from toc -->

- [How to use these recipes](#how-to-use-these-recipes)
- [Compare models](#compare-models)
  - [Goal](#goal)
  - [Steps](#steps)
  - [Keep constant](#keep-constant)
  - [Change only](#change-only)
  - [Starter prompt](#starter-prompt)
  - [Compare on](#compare-on)
  - [Common mistake](#common-mistake)
- [Use FlexiGPT with OpenRouter](#use-flexigpt-with-openrouter)
  - [Prerequisites](#prerequisites)
  - [Steps](#steps-1)
  - [Test prompt](#test-prompt)
  - [What to expect](#what-to-expect)
  - [Troubleshooting](#troubleshooting)
- [Use FlexiGPT with local models](#use-flexigpt-with-local-models)
  - [Prerequisites](#prerequisites-1)
  - [Steps](#steps-2)
  - [Test prompt](#test-prompt-1)
  - [What to expect](#what-to-expect-1)
  - [Safety check](#safety-check)
- [Create your first assistant preset](#create-your-first-assistant-preset)
  - [Goal](#goal-1)
  - [Steps](#steps-3)
  - [Test prompt](#test-prompt-2)
  - [Expected result](#expected-result)
  - [Next version ideas](#next-version-ideas)
- [Create your first prompt template](#create-your-first-prompt-template)
  - [Goal](#goal-2)
  - [Steps](#steps-4)
  - [Expected result](#expected-result-1)
  - [When to use instructions-only instead](#when-to-use-instructions-only-instead)
- [Create your first tool-assisted workflow](#create-your-first-tool-assisted-workflow)
  - [Goal](#goal-3)
  - [Steps](#steps-5)
  - [Starter prompt](#starter-prompt-1)
  - [Expected result](#expected-result-2)
  - [Creating a new tool](#creating-a-new-tool)
- [Create your first skill-backed workflow](#create-your-first-skill-backed-workflow)
  - [Goal](#goal-4)
  - [Steps](#steps-6)
  - [Starter prompt](#starter-prompt-2)
  - [Add the skill to an assistant preset](#add-the-skill-to-an-assistant-preset)
- [Analyze a file](#analyze-a-file)
  - [Suggested setup](#suggested-setup)
  - [Steps](#steps-7)
  - [Starter prompt](#starter-prompt-3)
- [Review code](#review-code)
  - [Suggested setup](#suggested-setup-1)
  - [Starter prompt](#starter-prompt-4)
  - [Expected result](#expected-result-3)
- [Investigate a bug](#investigate-a-bug)
  - [Suggested setup](#suggested-setup-2)
  - [Starter prompt](#starter-prompt-5)
- [Review architecture](#review-architecture)
  - [Suggested setup](#suggested-setup-3)
  - [Starter prompt](#starter-prompt-6)
- [Generate documentation](#generate-documentation)
  - [Suggested setup](#suggested-setup-4)
  - [Starter prompt](#starter-prompt-7)
- [Create a research brief](#create-a-research-brief)
  - [Suggested setup](#suggested-setup-5)
  - [Starter prompt](#starter-prompt-8)
- [Safe tool execution checklist](#safe-tool-execution-checklist)

## How to use these recipes

For reliable results:

- change one layer at a time
- keep attachments focused
- keep **Previous user turns** intentional
- inspect assistant preset details before assuming what it does
- start tools in manual mode
- use local providers only after confirming the endpoint is actually local

## Compare models

Use this when you want to compare answer quality, latency, reasoning style, or provider behavior.

### Goal

Run the same task through different model presets while keeping everything else the same.

### Steps

1. Open **Chats**.
2. Start a fresh chat or duplicate the task in another tab.
3. Choose an assistant preset if needed.
4. Set **Previous user turns** to a fixed value.
5. Add the same attachments.
6. Add the same prompt templates, system prompts, tools, web search, and skills.
7. Send the task with model A.
8. Open another tab or branch the task.
9. Change only the **Model** dropdown to model B.
10. Send the same task.
11. Compare the results.

### Keep constant

- assistant preset
- prompt text
- system prompt sources
- prompt template variables
- attachments and attachment modes
- previous user turns
- tool selections
- skills
- web-search setting

### Change only

- model preset

### Starter prompt

```text
Answer this task using only the attached context.
Then list assumptions, uncertainty, and the strongest reason your answer might be wrong.
```

### Compare on

- correctness
- completeness
- clarity
- citation quality
- latency
- token usage
- tool behavior
- ability to follow constraints
- cost from your provider account

### Common mistake

Do not compare one model with extra context and another without it.
That measures context differences, not model quality.

## Use FlexiGPT with OpenRouter

Use OpenRouter when you want one provider endpoint that can access many hosted models.

### Prerequisites

- OpenRouter account
- OpenRouter API key
- OpenRouter provider/model preset enabled in FlexiGPT

### Steps

1. Create an API key in OpenRouter.
2. Open **Settings -> Auth Keys**.
3. Add or update the key for OpenRouter.
4. Open **Model Presets**.
5. Confirm the OpenRouter provider is enabled.
6. Confirm the model preset you want is enabled.
7. Open **Chats**.
8. Select an OpenRouter model preset.
9. Send a small test prompt.

### Test prompt

```text
Reply with one sentence confirming the provider and model you are using.
```

### What to expect

OpenRouter is still a remote hosted provider path.
Request content goes to the OpenRouter endpoint and then the selected model provider path according to OpenRouter behavior.

Features may vary by model:

- tool support
- web search support
- reasoning controls
- output format support
- context length
- multimodal support

### Troubleshooting

If the model does not appear in Chats:

- check OpenRouter provider is enabled
- check the model preset is enabled
- check auth key exists and is non-empty
- check the model preset is compatible with the selected provider SDK setup
- try a tiny prompt before testing attachments or tools

## Use FlexiGPT with local models

Use this when you want inference to run through a local endpoint you control.

### Prerequisites

- a local model server running
- an API compatibility mode supported by FlexiGPT
- a provider/model preset pointing to the local endpoint

Supported compatible styles include:

- OpenAI Chat Completions-compatible
- OpenAI Responses-compatible
- Anthropic Messages-compatible
- Google GenerateContent-compatible

Many local servers use OpenAI Chat Completions-compatible routes.

### Steps

1. Start your local model server.
2. Open **Model Presets**.
3. Add or edit a provider preset for the local endpoint.
4. Use an origin like `http://127.0.0.1:8080` or `http://localhost:11434`.
5. Use the correct chat path, often `/v1/chat/completions`.
6. Add at least one model preset using the local model name expected by your server.
7. Enable the provider and model preset.
8. Add a placeholder auth key if your provider configuration requires a non-empty key.
9. Open **Chats**.
10. Select the local model preset.
11. Send a tiny test prompt.

### Test prompt

```text
Reply with "local model test ok" and no extra text.
```

### What to expect

Local models may differ from hosted models:

- smaller context window
- slower output
- limited or no tool support
- limited or no file/image support
- different output formatting
- weaker instruction following
- no provider-side web search

### Safety check

Local-first only means local inference when the selected provider origin is actually local.
Check the provider origin before sending private work.

## Create your first assistant preset

Use this when you keep rebuilding the same setup by hand.

### Goal

Create a reusable assistant preset that starts a documentation review workflow.

### Steps

1. Open **Assistant Presets**.
2. Click **Add Bundle** if you do not already have a custom bundle.
3. Use:
   - bundle slug: `my-assistants`
   - display name: `My Assistants`
4. Expand the custom bundle.
5. Click **Add Assistant Preset**.
6. Fill:
   - display name: `Docs Reviewer`
   - slug: `docs-reviewer`
   - version: `v1.0.0`
   - enabled: on
7. Select a starting model preset.
8. Set **Include Model System Prompt**:
   - `Include` if you want the model preset’s default prompt
   - `Do Not Include` if this assistant should rely only on selected instructions
   - `Not Set` if the preset should not decide
9. Add instruction templates if you have resolved instructions-only prompts.
10. Leave tools and skills empty for the first version.
11. Save.
12. Open **Chats**.
13. Select the new assistant preset.
14. Click **View** in the assistant dropdown to inspect what it supplies.

### Test prompt

```text
Review the attached documentation for clarity, missing setup steps, unsupported claims, and reader expectations.
Return prioritized fixes.
```

### Expected result

The assistant preset should seed the selected sections.
You can still change the model, prompts, tools, skills, and attachments after applying it.

### Next version ideas

Create a new version that adds:

- a stricter instruction template
- a local reader skill
- manual read-only tools
- different output verbosity
- lower temperature or stronger reasoning

## Create your first prompt template

Use this when you repeat the same request format.

### Goal

Create a reusable bug investigation template.

### Steps

1. Open **Prompts**.
2. Click **Add Bundle** if needed.
3. Use:
   - bundle slug: `my-prompts`
   - display name: `My Prompts`
4. Expand the custom bundle.
5. Click **Add Template**.
6. Fill:
   - display name: `Bug Investigation`
   - slug: `bug-investigation`
   - version: `v1.0.0`
   - enabled: on
7. Add a `user` block like:

   ```text
   Investigate this bug.

   Symptom:
   {{symptom}}

   Evidence:
   {{evidence}}

   Relevant constraints:
   {{constraints}}

   Return:
   1. likely root cause
   2. evidence
   3. missing evidence
   4. minimal fix
   5. verification steps
   ```

8. Add variables:
   - `symptom`, string, required, user
   - `evidence`, string, required, user
   - `constraints`, string, not required, user, default `None provided`
9. Save.
10. Open **Chats**.
11. Use **Prompts** in the composer bottom bar.
12. Select `Bug Investigation`.
13. Fill required variable pills.
14. Send.

### Expected result

The template inserts a structured draft.
Required variables must be filled before sending.

### When to use instructions-only instead

Use an instructions-only template if all blocks are `system` or `developer`.
That kind of template becomes a saved system prompt source and can be selected by assistant presets.

## Create your first tool-assisted workflow

Use this when a task may need execution, but you want human review.

### Goal

Add tools to a conversation and run them manually.

### Steps

1. Open **Chats**.
2. Choose a normal assistant preset first.
3. Open the **Tools** picker in the composer bottom bar.
4. Attach a read-oriented or low-risk tool.
5. Keep auto-execute off for the first run.
6. Ask the model to use the tool only if needed.
7. When a tool call appears, inspect it.
8. Click **Run** if the arguments are safe.
9. Inspect the tool output.
10. Send the output back if it helps.

### Starter prompt

```text
Use the available tools only if they are necessary.
Before using a tool, choose the narrowest safe call.
After tool output is available, explain what you learned and what remains uncertain.
```

### Expected result

The model may propose a tool call.
You stay in control of whether it runs.

### Creating a new tool

Use the **Tools** page to create or maintain tool definitions.
For first custom tools, prefer a simple HTTP-style tool with:

- clear display name
- narrow description
- required args schema
- safe timeout
- predictable response
- manual review first

I did not include a field-by-field custom tool form here because the Tools page add/edit implementation was not included in the latest attached files.
Use the Tools page UI and start with manual execution before enabling auto-execute.

## Create your first skill-backed workflow

Use this when you want a reusable workflow mode across turns.

### Goal

Enable a skill in a conversation and optionally add it to an assistant preset.

### Steps

1. Open **Skills**.
2. Confirm the skill bundle and skill are enabled.
3. If creating a custom skill, use a custom bundle and filesystem skill location.
4. Open **Chats**.
5. Open the **Skills** menu in the composer bottom bar.
6. Enable one relevant skill.
7. Send a task that benefits from that workflow mode.
8. Inspect whether the result follows the intended workflow.

### Starter prompt

```text
Use the enabled skill workflow where helpful.
Explain the steps you are taking and call out any assumptions or missing context.
```

### Add the skill to an assistant preset

1. Open **Assistant Presets**.
2. Create a new version of your custom assistant preset.
3. Add the skill under **Enabled Skills**.
4. Turn on **Preload as active** if you want it active immediately.
5. Save.
6. Apply the preset in Chats and use **View** to confirm the skill selection.

## Analyze a file

Use this when you want to understand an unfamiliar file.

### Suggested setup

- Assistant preset: Local Reader or similar reader preset
- Context: one file first
- Previous user turns: small, usually `0` or `1`

### Steps

1. Open **Chats**.
2. Apply a reader-style assistant preset.
3. Attach one file.
4. Confirm attachment mode is readable.
5. Send a focused request.

### Starter prompt

```text
Explain the attached file.
Cover its purpose, main flows, important types/functions, dependencies, and risky or surprising behavior.
End with a short "what to inspect next" list.
```

## Review code

Use this for correctness, maintainability, security, reliability, and test-risk feedback.

### Suggested setup

- Assistant preset: Reviewing Code
- Context: changed files, diff, PR description, tests, logs
- Tools: read-only tools if useful
- Previous user turns: small and intentional

### Starter prompt

```text
Review the attached code or diff for correctness, security, reliability, maintainability, and test gaps.
Focus on concrete issues.
Rank findings by severity and include narrow fixes.
```

### Expected result

Good output should include:

- high-impact findings first
- exact evidence
- minimal fixes
- test suggestions
- uncertainty or missing context

## Investigate a bug

Use this when you have logs, stack traces, failing tests, or confusing behavior.

### Suggested setup

- Assistant preset: Bug Investigator or Spec Driven Development
- Context: stack trace, logs, failing command, relevant code, recent diff
- Tools: manual first

### Starter prompt

```text
Investigate this bug from the attached logs, stack trace, and code.
Identify the likely root cause, evidence, missing evidence, narrow fix, and verification steps.
Do not assume uninspected files behave correctly.
```

## Review architecture

Use this for system boundaries, coupling, ownership, or design risk.

### Suggested setup

- Assistant preset: Designing System Architecture
- Context: README, architecture docs, code map, API contracts, constraints

### Starter prompt

```text
Review the attached architecture or code structure.
Identify current boundaries, risks, coupling, unclear ownership, and the simplest viable improvements.
Return a concise recommendation with trade-offs and revisit triggers.
```

## Generate documentation

Use this for README text, usage docs, API docs, or internal guides.

### Suggested setup

- Assistant preset: Docs Writer or Local Reader
- Context: source files, existing docs, examples, screenshots, command output

### Starter prompt

```text
Generate clear user-facing documentation from the attached context.
Include overview, setup, common workflow, examples, troubleshooting, and limitations.
Keep claims grounded in the provided files.
```

## Create a research brief

Use this for notes, URLs, PDFs, or copied source material.

### Suggested setup

- Assistant preset: Research Brief Writer
- Optional: provider-supported web search if current information matters

### Starter prompt

```text
Turn the attached research material into a brief with:
- key findings
- evidence
- disagreements or uncertainty
- recommendations
- next questions

Separate observed facts from interpretation.
```

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
