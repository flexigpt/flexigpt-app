# Reusable Catalogs

The pages outside **Chats** maintain reusable building blocks: Agents, tools, Skills, MCP server catalogs, Workspaces, providers, and models.

This page is an ownership map, not a second setup guide. Use it to find the
right management page, then follow that page's dedicated guide for the actual
workflow.

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

Use an Agent when you want a reusable Chat starting point. Agent details show
its currently resolved model, tools, Skills, Text, and MCP declarations.

Use **Chats -> Agent** to apply one. Use [Agents](/docs?doc=agents) for imports,
exports, Plugin management, availability, and immutable managed declarations.

## Tools

The **Tools** page is the catalog for native FlexiGPT tools and
provider-compatible tools. It is not a tool authoring page.

Use **Chats -> Tools** to select tools for a Run. Use **MCP** in the composer
for tools discovered from an MCP server. FlexiGPT does not offer user-created
HTTP or arbitrary custom tool definitions.

## Skills

The **Skills** page manages Skill Plugins and source-backed or managed Skills.
You can register a directory, attach an existing Skill or path to a Plugin,
create a managed Skill, inspect its document and resources, and enable or
disable it.

Use [Composer Context](/docs?doc=composer-context#skills) for the distinction
between user-message templates, enabled Skills, active Skills, and explicit
instruction sources.

## Models

The **Models** page owns provider compatibility, endpoint settings,
write-only credentials, model records, enablement, and defaults.

Use [Providers and Models](/docs?doc=providers-and-models) for the canonical
provider and model setup flow. Use [Local LLM Setup](/docs?doc=local-llm-setup)
only when the endpoint is local or self-hosted.

## Title-bar Settings

The title-bar **Settings** menu controls theme and logging/debug options. It does not contain provider auth keys.

Configure provider auth keys in **Models**, alongside the applicable provider/model setup. Credential values are write-only in the UI and are never shown again after saving.

Be careful with debug options. Raw request/response logging can include prompts, attachments, tool outputs, and sensitive provider responses.

## Built-in and custom content

Models, Skills, and Agents can include built-in and custom content:

- built-in content ships with the app
- custom content is stored locally
- built-in definitions are generally read-only, but may be enabled or disabled
- built-in items can usually be enabled or disabled
- custom entries can usually be edited/deleted according to that page’s rules

Tools are different: native tools and provider-compatible tools are selected
from the available catalog, but cannot be created, imported, or edited as
custom tool definitions.

When an item is unavailable, inspect its page for its Artifact state, enabled
state, Plugin state, source diagnostics, setup requirements, or dependencies.

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
