# Agents

Agents are reusable starting points for Chats. They are declarations of desired
contributions, not a separate autonomous execution engine.

An Agent can prepare a model choice, instruction Text, user-message starter
Text, tools, Skills, and MCP server references. FlexiGPT resolves those
contributions again when it prepares the Run.

After an Agent loads, you can still change the draft, model, instructions, tools, Skills, connected services, attachments, Workspaces, and history setting.

## Table of contents <!-- omit from toc -->

- [Use an Agent in Chats](#use-an-agent-in-chats)
- [Starter workflows on the home page](#starter-workflows-on-the-home-page)
- [What an Agent can set up](#what-an-agent-can-set-up)
- [Change the setup after loading](#change-the-setup-after-loading)
- [Manage Agents](#manage-agents)
- [Import an Agent file](#import-an-agent-file)
- [Export an Agent](#export-an-agent)
- [Connected service setup](#connected-service-setup)
- [Built-in and your own Agents](#built-in-and-your-own-agents)
- [Troubleshooting](#troubleshooting)
  - [An Agent does not appear in Chats](#an-agent-does-not-appear-in-chats)
  - [An Agent cannot be applied](#an-agent-cannot-be-applied)
  - [A connected service needs attention](#a-connected-service-needs-attention)

## Use an Agent in Chats

1. Open **Chats**.
2. Open the **Agent** menu above the composer.
3. Choose an available Agent.
4. Review the draft and context that appear.
5. Change anything you need before sending.

The Agent menu is a starting-point picker. It does not run an Agent on its own and does not take control away from you after loading.

## Starter workflows on the home page

The home page keeps the three high-value starters:

| Starter               | Agent it loads          | Best for                                                           |
| --------------------- | ----------------------- | ------------------------------------------------------------------ |
| **Develop a Feature** | Spec Driven Development | Focused repository changes from discovery through verification.    |
| **Review Code**       | Reviewing Code          | Diffs, changed files, pull requests, and test or risk review.      |
| **Investigate a Bug** | Bug Investigator        | Logs, errors, failing tests, stack traces, and likely root causes. |

Each card finds its built-in Agent when the app starts. The card loads that Agent into Chats, where its own opening request appears in the composer.

If a starter is unavailable, the card takes you to **Agents** so you can review its status.

## What an Agent can set up

Depending on its contents, an Agent can provide:

| Agent part              | What you see in Chats                                                            |
| ----------------------- | -------------------------------------------------------------------------------- |
| Model                   | A model selection for the next request.                                          |
| Instruction Text        | Guidance resolved into the Run's instruction context.                            |
| User-message Text       | An editable draft when the composer is empty.                                    |
| Tools                   | Tools available to the model.                                                    |
| Skills                  | Session Skills, active Skills, or rendered instruction sources.                  |
| Connected services      | MCP servers selected for the chat.                                               |
| Connected service setup | A prompt to finish local setup when a service needs credentials or other values. |

The Agent details view groups these contributions and shows whether each is
available, unavailable, or ambiguous. A starter preview is useful for planning,
but final Run preparation resolves the declarations again.

An Agent may leave any of these areas unchanged.

## Change the setup after loading

An Agent is a starter recipe, not a locked mode.

After loading one, you can:

- replace or edit the draft
- select another model
- change reasoning, temperature, output length, or other model settings
- add or remove instruction sources
- add or remove files, folders, links, and Workspace context
- add or remove tools
- change Skill selections
- change connected service selections
- change the amount of previous chat history included

When viewing the currently selected Agent, available tool, Skill, Text, and MCP
contributions can be excluded from the next Run. The model contribution is not
an exclusion control.

## Manage Agents

Open **Agents** from the sidebar to manage reusable Agent files.

Agents are grouped into Agent Plugins.

On this page you can:

- create a Plugin
- rename or describe your own Plugin
- import an Agent file
- inspect an Agent's setup
- export an Agent file
- enable or disable an Agent
- remove a managed Agent
- review connected service setup needs

Managed imported Agent declarations are immutable in the app. You cannot edit
one field at a time in the Agent page.

To change one, export it, edit it externally, then either:

1. export it
2. edit the file in your editor
3. import it under a new name

or remove the managed copy first and import the edited declaration again.

## Import an Agent file

1. Open **Agents**.
2. Create or choose a Plugin.
3. Select **Import Agent**.
4. Choose an Agent `.json`, `.yaml`, or `.yml` file.
5. Review the preview.
6. Read any warnings and confirmation requests.
7. Confirm the import.

The preview checks that the file is valid and shows what it can use right now.

An Agent may still import when an optional dependency is not currently available. For example, a connected service might need to be installed or configured later.

## Export an Agent

Use **Export** from an Agent row or open the Agent details.

Export produces the Agent YAML shown in Agent details. It is useful for
reviewing a managed or built-in declaration before creating your own version.

Do not assume the export is a live editor. The app may show canonical or
normalized YAML depending on the declaration source.

Exporting a built-in Agent does not make the built-in copy editable. Import the
export into an editable Agent Plugin when you want your own managed version.

## Connected service setup

Some Agents use MCP connected services.

After importing or opening an Agent, its details can show setup needs such as:

- a local path
- a normal text value
- an API key
- OAuth client details
- browser authorization

Use **Configure** when setup is needed.

Connected service values are stored locally. Secret values are not written back
into the Agent file and are not shown again after saving.

Configure only supplies declared installation inputs. Connection, OAuth
authorization, policy approval, and discovery still happen through
**MCP Servers** and during Run preparation.

## Built-in and your own Agents

Built-in Agents ship with FlexiGPT.

You can inspect and export them, but you cannot edit their original definitions.

Managed non-built-in Agents can be enabled, disabled, exported, removed, and
re-imported through their Agent Plugin.

A Plugin can still show an unavailable entry after an Agent is removed. That records the original choice and becomes available again if you restore a matching Agent later.

## Troubleshooting

### An Agent does not appear in Chats

Check:

- the Agent is enabled
- its Plugin is enabled
- the Agent is available on the **Agents** page
- the Agent file imported successfully
- any required setup warnings have been addressed

### An Agent cannot be applied

Open the Agent details and review the starter diagnostics.

Common causes:

- the selected model is unavailable
- a tool is unavailable for the selected model
- a connected service needs setup
- a referenced Skill is unavailable
- a required file or source is unavailable

### A connected service needs attention

Open the Agent details and use **Configure**. If it still cannot connect, open **MCP Servers** and review its setup, authorization, and discovery status.
