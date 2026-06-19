# MCP Servers

MCP (Model context protocol) servers let FlexiGPT reuse server-discovered tools, resources, resource templates, prompts, and app context in a chat turn. The server definitions live locally in FlexiGPT, while the endpoint itself can be local or remote.

Use this page to understand what the MCP Servers page owns and how it fits into Chats. For the composer-side selection flow, see [Composer Context](/docs?doc=composer-context#mcp-servers).

## Table of contents <!-- omit from toc -->

- [What MCP servers are](#what-mcp-servers-are)
- [What the page owns](#what-the-page-owns)
- [Connecting and discovery](#connecting-and-discovery)
- [What Chats can use](#what-chats-can-use)
- [Safety and trust](#safety-and-trust)

## What MCP servers are

A bundle groups one or more MCP servers.

Use bundles when you want to keep related servers together.

Important points:

- built-in bundles and servers ship with the app and are usually read-only
- custom bundles and servers are local content and can be created, edited, copied, or deleted
- the backend keeps the bundle, server, auth, setup, and runtime metadata on your machine
- the server itself may still be a local or remote endpoint
- discovery is separate from the composer selection you make for one chat turn

## What the page owns

| Area    | What it controls                                                                                         |
| ------- | -------------------------------------------------------------------------------------------------------- |
| Bundle  | Group servers together and control the bundle enabled state.                                             |
| Server  | Transport, trust level, auth mode, setup inputs, default policy, apps policy, and tool policy overrides. |
| Runtime | Connect, disconnect, refresh, auth health, and discovery snapshots.                                      |
| Secrets | Store auth values and setup secrets through the normal MCP secret flow.                                  |

Common page actions:

- add a bundle
- add or edit a server
- copy an existing server as a starting point
- configure setup inputs
- connect, disconnect, or refresh a server
- inspect runtime details and discovery counts
- manage OAuth settings when a server needs a loopback callback

## Connecting and discovery

The page keeps connection state and discovery state separate.

- use Connect after the server config is complete
- use Refresh when the server's tools, resources, or prompts change
- open Details to inspect server info, capabilities, config, and discovery counts
- watch the auth health badge for OAuth or other authorization states
- if a server needs browser authorization, complete the callback while FlexiGPT stays open

## What Chats can use

Once a server is ready, the Chats composer can select its active context for a turn.

From the `MCP` chip you can choose:

- tools
- resources
- resource templates
- prompts
- server instructions

Practical rules:

- tool exposure can be `all`, `selected`, or `none`
- app-only tools stay in the UI, but are not exposed to the model
- required arguments block send until they are filled
- keep instructions on only when they help the task
- refresh discovery on the server page when the server contents change

## Safety and trust

MCP servers can be useful, but they are still a trust boundary.

Start with:

- one server
- manual review
- selected tool exposure
- narrow context
- low-risk prompts or resources first

Check these before expanding a workflow:

- trust level
- approval rule
- execution mode
- auth health
- discovery freshness
- whether the endpoint is local or remote
