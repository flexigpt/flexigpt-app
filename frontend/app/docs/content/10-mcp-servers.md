# MCP Servers

MCP (Model context protocol) servers let FlexiGPT reuse server-discovered tools, resources, resource templates, prompts, and app context in a chat turn. The server definitions live locally in FlexiGPT, while the endpoint itself can be local or remote.

Use this page to understand what the MCP Servers page owns and how it fits into Chats. For the composer-side selection flow, see [Composer Context](/docs?doc=composer-context#mcp-servers).

## Table of contents <!-- omit from toc -->

- [What MCP servers are](#what-mcp-servers-are)
- [What the page owns](#what-the-page-owns)
- [Define a server](#define-a-server)
- [Installation values and credentials](#installation-values-and-credentials)
- [Connecting and discovery](#connecting-and-discovery)
- [Policies and MCP Apps](#policies-and-mcp-apps)
- [What Chats can use](#what-chats-can-use)
- [Safety and trust](#safety-and-trust)

## What MCP servers are

A plugin groups one or more MCP servers.

Use plugins when you want to keep related servers together.

Important points:

- built-in plugins and servers ship with the app and are usually read-only
- custom plugins and servers are local content and can be created, edited, copied, or deleted
- the backend keeps the plugin, server, auth, setup, and runtime metadata on your machine
- the server itself may still be a local or remote endpoint
- discovery is separate from the composer selection you make for one chat turn
- configuring a server does not automatically make it connected, authorized, or selected for a chat

## What the page owns

| Area    | What it controls                                                                                         |
| ------- | -------------------------------------------------------------------------------------------------------- |
| Plugin  | Group servers together and control the plugin enabled state.                                             |
| Server  | Transport, trust level, auth mode, setup inputs, default policy, apps policy, and tool policy overrides. |
| Runtime | Connect, disconnect, refresh, auth health, and discovery snapshots.                                      |
| Secrets | Store write-only auth values and installation secrets separately from the server definition.             |

Common page actions:

- add a plugin
- add or edit a server
- configure setup inputs
- connect, disconnect, or refresh a server
- inspect runtime details and discovery counts
- manage OAuth settings when a server needs a loopback callback

## Define a server

Custom MCP servers support two transports:

| Transport           | Use it for                                                                                                               |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| **Stdio**           | A local command that FlexiGPT starts with configured arguments and environment values.                                   |
| **Streamable HTTP** | A local or remote MCP endpoint. Remote endpoints require HTTPS; plain HTTP is limited to loopback development endpoints. |

For HTTP servers, choose the authentication mode that matches the endpoint:

- none
- API key
- OAuth
- client credentials

The server definition contains non-secret settings. API keys, OAuth client
credentials, and declared secret environment/header values are stored through
the MCP setup flow and are not displayed after saving.

## Installation values and credentials

An MCP server can declare local installation inputs such as:

- text values
- paths
- secrets
- OAuth client credentials

Use **Setup** to provide or replace those values. Saving setup values does not
replace a separate OAuth authorization flow, connection step, or tool policy.

When OAuth is required:

1. save the server and any installation inputs
2. choose **Connect**
3. complete browser authorization while FlexiGPT remains open
4. wait for the server to finish connecting and discovering capabilities

## Connecting and discovery

The page keeps connection state and discovery state separate.

- use Connect after the server config is complete
- use Refresh when the server's tools, resources, or prompts change
- open Details to inspect server info, capabilities, config, and discovery counts
- watch the auth health badge for OAuth or other authorization states
- if a server needs browser authorization, complete the callback while FlexiGPT stays open
- reconnect or refresh after changing a server definition, credential, setup value, or server-side capabilities

## Policies and MCP Apps

Each server has a policy that controls how discovered tools and MCP Apps behave.

The server editor can configure:

- trust level
- default approval rule
- manual or automatic execution mode
- additional approval requirements for unknown-risk, write, or destructive tools
- per-tool policy overrides
- whether MCP Apps are advertised and rendered
- whether an App may request tool calls
- whether opening links or staging model context needs approval

An MCP App can appear inline after a compatible MCP tool result. It is still
scoped to the Run that created it. App requests for a user draft, model-context
update, external link, or app-initiated tool call remain subject to the active
policy and Runtime approval flow.

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
- refresh discovery after the server's capabilities change
- an Agent can select a connected service as part of its starting setup
- if an Agent or Workspace shows **Configure**, finish installation setup before using that service

Agent setup never shares secret values through the Agent file.

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
