# Setup Recipes

These recipes are app and workflow setup flows. They help you configure providers, local endpoints, assistant presets, prompt templates, tools, and skills.

For outcome-based LLM tasks, see [Everyday Recipes](/docs?doc=everyday-recipes).

## Table of contents <!-- omit from toc -->

- [Use FlexiGPT with OpenRouter](#use-flexigpt-with-openrouter)
- [Use FlexiGPT with local models](#use-flexigpt-with-local-models)
- [Create your first assistant preset](#create-your-first-assistant-preset)
- [Create your first prompt template](#create-your-first-prompt-template)
- [Create your first tool-assisted workflow](#create-your-first-tool-assisted-workflow)
- [Create your first skill-backed workflow](#create-your-first-skill-backed-workflow)

## Use FlexiGPT with OpenRouter

Use OpenRouter when you want one provider endpoint that can access many hosted models.

Prerequisites:

- OpenRouter account
- OpenRouter API key
- OpenRouter provider/model preset enabled in FlexiGPT

Steps:

1. Create an API key in OpenRouter.
2. Open **Settings -> Auth Keys**.
3. Add or update the key for OpenRouter.
4. Open **Model Presets**.
5. Confirm the OpenRouter provider is enabled.
6. Confirm the model preset you want is enabled.
7. Open **Chats**.
8. Select an OpenRouter model preset.
9. Send a small test prompt.

Test prompt:

    Reply with one sentence confirming the provider and model you are using.

Expectations:

- OpenRouter is still a remote hosted provider path.
- Request content goes to OpenRouter and then through the selected model provider path according to OpenRouter behavior.
- Features vary by model: tools, web search, reasoning controls, output format, context length, and multimodal support.

Troubleshooting:

- check OpenRouter provider is enabled
- check the model preset is enabled
- check auth key exists and is non-empty
- check the model preset is compatible with the selected provider SDK setup
- try a tiny prompt before testing attachments or tools

## Use FlexiGPT with local models

Use this when you want inference to run through a local endpoint you control.

Prerequisites:

- local model server running
- API compatibility mode supported by FlexiGPT
- provider/model preset pointing to the local endpoint

Supported compatible styles include:

- OpenAI Chat Completions-compatible
- OpenAI Responses-compatible
- Anthropic Messages-compatible
- Google GenerateContent-compatible

Many local servers use OpenAI Chat Completions-compatible routes.

Steps:

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

Test prompt:

    Reply with "local model test ok" and no extra text.

Expectations:

- smaller context window than hosted frontier models
- slower output depending on hardware
- limited or no tool support
- limited or no file/image support
- different output formatting
- weaker instruction following
- no provider-side web search

Safety check:

- local-first only means local inference when the selected provider origin is actually local
- confirm the local server is not proxying to a remote service
- test with harmless content first

## Create your first assistant preset

Use this when you keep rebuilding the same setup by hand.

Goal:

- Create a reusable assistant preset that starts a documentation review workflow.

Steps:

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

Test prompt:

    Review the attached documentation for clarity, missing setup steps, unsupported claims, and reader expectations.
    Return prioritized fixes.

Expected result:

- the assistant preset seeds the selected sections
- you can still change model, prompts, tools, skills, and attachments after applying it

Next version ideas:

- stricter instruction template
- local reader skill
- manual read-only tools
- different output verbosity
- lower temperature or stronger reasoning

## Create your first prompt template

Use this when you repeat the same request format.

Goal:

- Create a reusable bug investigation template.

Steps:

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

Expected result:

- the template inserts a structured draft
- required variables must be filled before sending

Use an instructions-only template instead if all blocks are `system` or `developer`. That kind of template becomes a saved system prompt source and can be selected by assistant presets.

## Create your first tool-assisted workflow

Use this when a task may need execution, but you want human review.

Goal:

- Add tools to a conversation and run them manually.

Steps:

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

Starter prompt:

    Use the available tools only if they are necessary.
    Before using a tool, choose the narrowest safe call.
    After tool output is available, explain what you learned and what remains uncertain.

Expected result:

- the model may propose a tool call
- you stay in control of whether it runs

Creating a new tool:

- use the **Tools** page to create or maintain tool definitions
- for first custom tools, prefer a simple HTTP-style tool with clear display name, narrow description, required args schema, safe timeout, predictable response, and manual review first
- use the Tools page UI as the source of truth for current create/edit fields

## Create your first skill-backed workflow

Use this when you want a reusable workflow mode across turns.

Goal:

- Enable a skill in a conversation and optionally add it to an assistant preset.

Steps:

1. Open **Skills**.
2. Confirm the skill bundle and skill are enabled.
3. If creating a custom skill, use a custom bundle and filesystem skill location.
4. Open **Chats**.
5. Open the **Skills** menu in the composer bottom bar.
6. Enable one relevant skill.
7. Send a task that benefits from that workflow mode.
8. Inspect whether the result follows the intended workflow.

Starter prompt:

    Use the enabled skill workflow where helpful.
    Explain the steps you are taking and call out any assumptions or missing context.

Add the skill to an assistant preset:

1. Open **Assistant Presets**.
2. Create a new version of your custom assistant preset.
3. Add the skill under **Enabled Skills**.
4. Turn on **Preload as active** if you want it active immediately.
5. Save.
6. Apply the preset in Chats and use **View** to confirm the skill selection.
