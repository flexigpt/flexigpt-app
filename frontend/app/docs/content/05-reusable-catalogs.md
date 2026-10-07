# Reusable Catalogs

The pages outside **Chats** maintain reusable building blocks: Agents, built-in tools, Skills, MCP server catalogs, models, and title-bar settings.

Use Chats to load an Agent and apply reusable setup to a conversation. Use the appropriate pages to inspect, enable, disable, or manage the reusable definitions they own.

## Table of contents <!-- omit from toc -->

- [Catalog ownership](#catalog-ownership)
- [Agents](#agents)
- [Tools](#tools)
- [Skills](#skills)
- [Models](#models)
- [Title-bar Settings](#title-bar-settings)
- [Built-in and custom content](#built-in-and-custom-content)
- [Choosing the right page](#choosing-the-right-page)

## Catalog ownership

| Goal                                                 | Page                        |
| ---------------------------------------------------- | --------------------------- |
| Reuse a whole workflow setup                         | Agents                      |
| Create reusable workflow modes or skill-based drafts | Skills                      |
| Browse available built-in Go capabilities            | Tools                       |
| Create or maintain MCP server catalogs               | MCP Servers                 |
| Configure providers, models, and auth keys           | Models                      |
| Change theme or logging/debug options                | Title-bar **Settings** menu |

## Agents

Agents are reusable starting points for a type of work. An Agent can prepare a model choice, instructions, opening text, tools, Skills, and connected services without locking the chat.

Use **Chats -> Agent** to load an Agent. Use [Agents](/docs?doc=agents) to manage Plugins, import or export Agent files, inspect setup, and enable or disable Agents.

After loading an Agent, you can still change the model, draft, instructions, Skills, tools, connected services, attachments, Workspaces, and previous user turns. See [What an Agent can set up](/docs?doc=agents#what-an-agent-can-set-up) and [Manage Agents](/docs?doc=agents#manage-agents) for details.

## Tools

The **Tools** page lists the built-in Go tool capabilities available in FlexiGPT. Runtime execution happens in Chats.

From a user perspective:

- the Tools page shows the built-in capabilities that can be selected
- the Chats composer decides which tools are available to a conversation or message
- the model can then request tool calls
- tool calls can be run manually or auto-executed if configured

Safety expectations:

- start with manual review
- inspect tool arguments before running
- keep auto-execute off for risky tools
- inspect outputs before sending them back
- remove tools not needed for the current workflow

The available tools are built-in Go implementations or provider-bound capabilities. FlexiGPT does not support user-created HTTP tools or arbitrary custom tool definitions. Use the Tools page as the source of truth for available tools and their arguments.

## Skills

The **Skills** page manages skill plugins and skills.

Important behavior:

- built-in embedded skills are generally read-only
- user-created skills are filesystem skills
- skill names must be unique within a plugin
- skill refs include a skill ID to avoid stale identity confusion
- skills can be enabled or disabled
- skills can show presence status such as present, missing, error, or unknown

In Chats, a template-style skill renders into plain composer text and does not remain selected after insertion. An instruction-only skill can be selected as a system instruction source independently of any template text. Selected sources are combined with the enabled model default rather than replacing one another.

Use skills when you want a reusable workflow mode, a template-style draft starter, or instruction-only behavior that shapes the request context. See [Composer Context](/docs?doc=composer-context#templates-and-instruction-sources) for the per-message flow.

Use Agents to preload Skills for common workflows.

## Models

The **Models** page owns provider, model, and provider-auth setup.

It controls:

- default provider and provider auth keys
- provider enablement
- model enablement
- provider SDK/API compatibility type
- provider origin and path
- API key header name
- default headers
- model parameters and capability overrides
- default model per provider

Use **Models** when changing how requests run or managing a provider key. Use Agents when changing the kind of workflow you want to start from.

For local and self-hosted LLMs, the recommended customization order is:

1. copy/fork an existing provider
2. adjust provider-level settings
3. then copy, add, or edit models under that provider

Do provider first because the provider owns the shared endpoint contract:

- SDK/API compatibility type
- origin URL
- chat path
- API-key header name
- default headers
- provider-wide capability assumptions

Built-in local providers such as LocalAI, LM Studio, `llama.cpp`, Ollama, SGLang, and vLLM are useful defaults, but local server ports, paths, headers, model names, and feature support vary. In **Models**, copy the closest built-in provider, adjust its endpoint and compatibility settings, then add or copy a model under that provider.

See [Local LLM Setup](/docs?doc=local-llm-setup) for a full local runtime setup flow.

## Title-bar Settings

The title-bar **Settings** menu controls theme and logging/debug options. It does not contain provider auth keys.

Configure provider auth keys in **Models**, alongside the applicable provider/model setup. Provider secrets are stored through the OS keyring and are not shown as raw values.

Be careful with debug options. Raw request/response logging can include prompts, attachments, tool outputs, and sensitive provider responses.

## Built-in and custom content

Models, Skills, and Agents can include built-in and custom content:

- built-in content ships with the app
- custom content is stored locally
- built-in definitions are generally read-only
- built-in items can usually be enabled or disabled
- custom entries can usually be edited/deleted according to that page’s rules

Tools are different: FlexiGPT exposes only built-in Go tools. They can be selected in Chats, but cannot be created, imported, or edited as custom tool definitions.

For content that supports customization, a practical workflow is:

1. start from built-in content
2. inspect what it does
3. create custom versions when you need durable changes
4. keep built-ins enabled only if useful

## Choosing the right page

| If you want to...                                    | Go to...                                |
| ---------------------------------------------------- | --------------------------------------- |
| Start a reusable workflow                            | Chats -> Agent menu                     |
| Inspect an Agent's setup                             | Agents                                  |
| Import, export, enable, or manage an Agent           | Agents                                  |
| Create reusable skill-based drafts or behavior rules | Skills                                  |
| Browse available built-in tools                      | Tools                                   |
| Enable a tool in a chat                              | Chats -> Tools or an Agent              |
| Create or maintain skill definitions                 | Skills                                  |
| Create or maintain MCP server catalogs               | MCP Servers                             |
| Select MCP server context for a turn                 | Chats -> Composer -> MCP                |
| Enable Skills in a chat                              | Chats -> Skills or an Agent             |
| Add an API key                                       | Models                                  |
| Add a local/custom provider                          | Models                                  |
| Change theme or logging/debug options                | Title-bar Settings menu                 |
| Compare models                                       | Chats, changing only the selected model |
