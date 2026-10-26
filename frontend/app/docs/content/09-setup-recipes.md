# Setup Recipes

These recipes are app and workflow setup flows. They help you configure providers, local endpoints, Agents, Workspaces, tools, Skills, and connected services.

Each recipe is intentionally short. Follow the linked page when you need field-by-field setup, safety, or troubleshooting detail.

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

Steps:

1. Create an API key in OpenRouter.
2. Open **Models**.
3. Configure the OpenRouter provider credential, then enable a model under it.
4. Open **Chats**, select that model, and send a tiny text-only test.
5. Add attachments, tools, or web search only after the basic request succeeds.

OpenRouter is a remote provider path. Model capabilities can differ by model,
so inspect the current model controls after switching models. For the complete
provider flow, see [Providers and Models](/docs?doc=providers-and-models#openrouter).

## Use FlexiGPT with local models

Use this when you want inference to run through a local endpoint you control.

Prerequisites:

- local model server running
- a built-in local provider that matches your server, or a forked provider pointing to your endpoint
- provider/model enabled in FlexiGPT

Built-in local and self-hosted provider presets include:

- LocalAI, LM Studio, `llama.cpp`, Ollama, SGLang, and vLLM, when included in your installed catalog

Steps:

1. Start your local model server.
2. Open **Models**.
3. Add a provider by copying the closest preset, or configure a compatible endpoint from scratch.
4. Set the actual origin, path, compatibility type, key-header behavior, and model name expected by the server.
5. Enable provider and model, then send a tiny text-only test in **Chats**.

For local-only checks, runtime caveats, and troubleshooting, see
[Local LLM Setup](/docs?doc=local-llm-setup).

## Import your first Agent

Use this when you want a reusable starting point for a type of work.

Goal:

- Import an Agent file and use it in Chats.

Steps:

1. Open **Agents**.
2. Create or choose a Plugin.
3. Select **Import Agent**.
4. Choose an Agent `.json`, `.yaml`, or `.yml` file.
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
6. Select one repository setup and choose the context, instruction Skills, and MCP references needed for the task.
7. Resolve any changed or unavailable Workspace warning before sending.

Use a Workspace for project context and an Agent for a starting way of working.

## Create your first tool-assisted workflow

Use this when a task may need execution, but you want human review.

Goal:

- Add tools to a conversation and run them manually.

Steps:

1. Open **Chats**.
2. Choose an Agent if a starting setup would help.
3. Open the **Tools** picker in the composer bottom bar.
4. Select an available read-oriented or low-risk native or provider-compatible tool.
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

Available tools:

- FlexiGPT exposes only built-in Go tools. It does not provide a custom or HTTP-tool creation flow.
- Use the **Tools** page as the source of truth for available tools, arguments, and execution behavior.

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
2. Confirm the Skill Plugin and Skill are enabled.
3. Create a managed Skill, register a Skill directory, attach a Skill path, or attach an existing Skill Artifact.
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
2. Create or choose a plugin.
3. Add one server and choose its transport, trust level, and authentication mode.
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

- check plugin enabled
- check server and required installation inputs
- check auth health
- refresh discovery after changing server config
- fill or remove incomplete arguments if send is blocked
