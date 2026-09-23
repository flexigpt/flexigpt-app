# Agents

Agents are reusable starting points for Chats.

An Agent can prepare a useful combination of model choice, instructions, tools, Skills, connected services, and opening text. Loading an Agent does not lock the chat. It only gives you a starting point.

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
| Instructions            | Guidance added to the system instructions for the chat.                          |
| Opening text            | An editable draft when the composer is empty.                                    |
| Tools                   | Tools available to the model.                                                    |
| Skills                  | Skills enabled for the chat, with some ready to use immediately.                 |
| Connected services      | MCP servers selected for the chat.                                               |
| Connected service setup | A prompt to finish local setup when a service needs credentials or other values. |

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

## Manage Agents

Open **Agents** from the sidebar to manage reusable Agent files.

Agents are grouped into Collections.

On this page you can:

- create a Collection
- rename or describe your own Collection
- import an Agent file
- inspect an Agent's setup
- export an Agent file
- enable or disable an Agent
- remove a managed Agent
- review connected service setup needs

You cannot edit an imported Agent one field at a time in the app.

To change one:

1. export it
2. edit the file in your editor
3. import it under a new name

Or:

1. export it
2. remove the current managed Agent
3. edit the file
4. import it again with the same name

## Import an Agent file

1. Open **Agents**.
2. Create or choose a Collection.
3. Select **Import Agent**.
4. Choose an Agent `.yaml` or `.yml` file.
5. Review the preview.
6. Read any warnings and confirmation requests.
7. Confirm the import.

The preview checks that the file is valid and shows what it can use right now.

An Agent may still import when an optional dependency is not currently available. For example, a connected service might need to be installed or configured later.

## Export an Agent

Use **Export** from an Agent row or open the Agent details.

Export is available for:

- built-in Agents
- imported Agents
- Agents discovered from your files

The exported file keeps its original structure. Inline instructions and inline connected-service details stay inside the exported Agent file.

Exporting a built-in Agent does not make the built-in copy editable. Import the exported file into one of your Collections when you want your own version.

## Connected service setup

Some Agents use MCP connected services.

After importing or opening an Agent, its details can show setup needs such as:

- a local path
- a normal text value
- an API key
- OAuth client details
- browser authorization

Use **Configure** when setup is needed.

Connected service values are stored locally. Secret values are not written back into the Agent file and are not shown again after saving.

For server-wide setup, connection, discovery, and troubleshooting, use **MCP Servers**.

## Built-in and your own Agents

Built-in Agents ship with FlexiGPT.

You can inspect and export them, but you cannot edit their original definitions.

Your own Agents live in your local Collections. You can enable, disable, export, remove, and re-import them.

A Collection can still show an unavailable entry after an Agent is removed. That records the original choice and becomes available again if you restore a matching Agent later.

## Troubleshooting

### An Agent does not appear in Chats

Check:

- the Agent is enabled
- its Collection is enabled
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
