# Setup Recipes

These recipes are app and workflow setup flows. They help you configure providers, local endpoints, Agents, Workspaces, tools, Skills, and connected services.

For outcome-based LLM tasks, see [Everyday Recipes](/docs?doc=everyday-recipes).

## Table of contents <!-- omit from toc -->

- [Use FlexiGPT with OpenRouter](#use-flexigpt-with-openrouter)
- [Use FlexiGPT with local models](#use-flexigpt-with-local-models)
- [Import your first Agent](#import-your-first-agent)
- [Set up your first Workspace](#set-up-your-first-workspace)
- [Create your first tool-assisted workflow](#create-your-first-tool-assisted-workflow)
- [Create your first skill-backed workflow](#create-your-first-skill-backed-workflow)
- [Create your first MCP-backed workflow](#create-your-first-mcp-backed-workflow)

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
- a built-in local provider preset that matches your server, or a forked provider preset pointing to your endpoint
- provider/model preset enabled in FlexiGPT

Built-in local and self-hosted provider presets include:

- LocalAI
- LM Studio
- `llama.cpp`
- Ollama
- SGLang
- vLLM

These built-ins are good starting points. Because local servers vary in URL, path, headers, API compatibility, model names, and capabilities, the safest durable setup is usually to fork/copy the provider first and then adjust models.

Steps:

1. Start your local model server.
2. Open **Model Presets**.
3. Click **Add Provider**.
4. Use **Prefill from Existing -> Copy Existing Provider**.
5. Choose the closest built-in local provider.
6. Give the forked provider a stable ID and display name.
7. Adjust the provider origin, chat path, SDK/API compatibility type, API-key header, and default headers for your server.
8. Save and enable the provider.
9. Add a placeholder auth key if the provider configuration requires a non-empty key.
10. Under the forked provider, add a model preset or use **Copy Existing Preset** from a close built-in model.
11. Set the model name to the exact name or tag expected by your local server.
12. Enable the model preset and optionally set it as default for that provider.
13. Open **Chats**.
14. Select the local model preset.
15. Send a tiny test prompt.

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
- provider IDs should stay stable after Chats or Agents use them
- use harmless placeholder keys for dummy local auth instead of real production API keys
- confirm the local server is not proxying to a remote service
- test with harmless content first
- for the full provider-first flow, see [Local LLM Setup](/docs?doc=local-llm-setup)

## Import your first Agent

Use this when you want a reusable starting point for a type of work.

Goal:

- Import an Agent file and use it in Chats.

Steps:

1. Open **Agents**.
2. Create or choose a Collection.
3. Select **Import Agent**.
4. Choose an Agent `.yaml` or `.yml` file.
5. Review the preview.
6. Read any warnings and confirmation requests.
7. Confirm the import.
8. Open **Chats**.
9. Open the **Agent** menu.
10. Select the imported Agent.
11. Review and change the draft or context before sending.

Test prompt:

    Review the attached documentation for clarity, missing setup steps, unsupported claims, and reader expectations.
    Return prioritized fixes.

Expected result:

- the Agent can add a useful starting model, instructions, Skills, tools, and connected services
- opening text appears as an editable draft when configured
- you can still change model, Skills, tools, Workspaces, and attachments after loading it

To change an imported Agent:

1. export it
2. edit it in your editor
3. import it under a new name

Or remove the managed copy first, then import the edited file again with the same name.

## Set up your first Workspace

1. Open **Workspaces**.
2. Add the repository or folder.
3. Review the discovered Workspace.
4. Open **Chats**.
5. Open the composer **Workspace** picker.
6. Select the repository setup.
7. Add or remove context before sending.

Use a Workspace for project context and an Agent for a starting way of working.

## Create your first tool-assisted workflow

Use this when a task may need execution, but you want human review.

Goal:

- Add tools to a conversation and run them manually.

Steps:

1. Open **Chats**.
2. Choose an Agent if a starting setup would help.
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

- Enable a Skill in a conversation and optionally add it to an Agent.

Skills can fill three roles:

- template-style skills seed the draft with reusable structure or starter text
- instruction-only skills act like durable system-style instructions
- normal skills add workflow behavior and runtime state across turns

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

To reuse the Skill setup often:

1. Add the Skill to an Agent file.
2. Import the Agent into **Agents**.
3. Load the Agent in Chats.
4. Confirm the Skill selection in the composer.

## Create your first MCP-backed workflow

Use this when you want to configure one MCP server and use it from Chats.

Suggested setup:

- MCP Servers page: one custom or built-in-compatible server
- Context: one server, not many
- Tool exposure: `selected` or `none` until you trust the server
- Previous user turns: usually `0` or `1`

Steps:

1. Open **MCP Servers**.
2. Create or choose a bundle.
3. Add one server, or copy an existing server if it is a close fit.
4. Set transport, trust level, auth mode, and any required setup inputs.
5. Connect the server and confirm that discovery is loaded.
6. Open **Chats**.
7. Open the composer **MCP** chip.
8. Select the server and choose only the context you need.
9. Fill required arguments before sending.
10. Send a small test request.

Starter prompt:

- Use the selected MCP server only if it helps.
- List the tools, resources, or prompts you plan to use, then call out any missing arguments or approval steps before you act.

Expected result:

- the server contributes only the context you selected
- required arguments are filled before send
- manual approval still applies where the tool policy requires it

Troubleshooting:

- check bundle enabled
- check server enabled
- check auth health
- refresh discovery after changing server config
- fill or remove incomplete arguments if send is blocked
