# Getting Started

FlexiGPT is a local-first BYOK AI workspace for repeatable LLM work. The desktop workspace stays local, while each Run is sent to the provider or endpoint selected for that chat.

This page is the shortest path to a useful first request.

## Table of contents <!-- omit from toc -->

- [First successful send](#first-successful-send)
- [Choose your first path](#choose-your-first-path)
- [What to look at before sending](#what-to-look-at-before-sending)
- [A good first request](#a-good-first-request)
- [After the response](#after-the-response)
- [Where to go next](#where-to-go-next)
- [If the first send fails](#if-the-first-send-fails)

## First successful send

1. Open **Models**.
2. Enable a provider preset, or choose **Add Provider** and configure a supported API compatibility style and endpoint.
3. Set a provider API key if that endpoint requires one.
   - For a local endpoint that only checks for a non-empty value, use a harmless placeholder rather than a production key.
4. Add or enable a model under that provider.
5. Open **Chats**, select the model or an Agent starter, type a small request, and send it.

Good first test: `Reply with one sentence confirming the model is working.`

Once that works, attach source material or use a starter Agent.

## Choose your first path

| If you want to...                       | Start here                                                                                                                                   |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Use a normal hosted model               | Add the key and enable the model in **Models**, then use **Chats**.                                                                          |
| Try many hosted models through one key  | Use OpenRouter. See [Providers and Models](/docs?doc=providers-and-models#openrouter).                                                       |
| Use a local model server                | Copy the closest provider preset or add a provider, then configure its endpoint and model. See [Local LLM Setup](/docs?doc=local-llm-setup). |
| Start from a known workflow             | Choose a home screen workflow card or an Agent in **Chats**.                                                                                 |
| Reuse repository context                | Register the repository in **Workspaces**, then select it from the Chat composer.                                                            |
| Work with private or sensitive material | Read [Privacy, Data, and Troubleshooting](/docs?doc=privacy-data-and-troubleshooting) before sending.                                        |

## What to look at before sending

Before each important request, check these layers:

1. **Provider/model**
   - Which endpoint receives the request?
2. **Agent and Workspace**
   - Did you load a starter recipe or reusable repository context?
3. **Instructions and Skills**
   - Are model defaults, selected instruction sources, or enabled Skills changing the request?
4. **Selected context**
   - Are attachments, tools, MCP selections, Workspace content, or web search necessary for this task?
5. **Current message**
   - Is the request concrete enough to act on?
6. **History**
   - Is **Previous user turns** small enough to avoid stale context?

If the output changes unexpectedly, usually one of these layers changed.

## A good first request

Start with one focused task and one focused piece of context.

Examples:

- Explain this attached file.
- Summarize these notes.
- Compare these two approaches.
- Develop this feature in the attached repo. First inspect relevant files, write a short spec, then implement after confirmation.
- Review this code snippet for correctness and test gaps.
- Rewrite this paragraph for a technical README.
- Turn this stack trace and log excerpt into likely causes and next checks.

Avoid beginning with a huge folder, many tools, and long chat history. Add complexity after the basic path works.
If a starter workflow prefills the composer, replace its placeholder with your concrete task before sending.

## After the response

In the message timeline you can usually:

- read rendered Markdown, code, diagrams, and math
- inspect token usage and message details
- review citations when the provider returns them
- copy output
- inspect tool calls and tool outputs
- resolve approval or input waits for an active Run
- inspect MCP App views when an MCP tool returns one
- edit an earlier user message and resend to branch the conversation
- export the current conversation as JSON

## Where to go next

- Learn the main work surface: [Chat Workspace](/docs?doc=chat-workspace)
- Learn the vocabulary and page ownership: [Concepts and Ownership](/docs?doc=concepts-and-ownership)
- Learn how to add files, prompts, tools, skills, and web search: [Composer Context](/docs?doc=composer-context)
- Learn how to use and manage reusable starting points: [Agents](/docs?doc=agents)
- Find the page that owns reusable setup: [Manage Reusable Setup](/docs?doc=reusable-catalogs)
- Configure providers and local models: [Providers and Models](/docs?doc=providers-and-models)
- Try outcome-based tasks: [Everyday Recipes](/docs?doc=everyday-recipes)

## If the first send fails

Check:

- provider auth key is configured in **Models**
- provider is enabled in **Models**
- model is enabled
- model uses the correct provider/API type
- custom endpoint origin includes `http://` or `https://`
- local server is running, if using a local endpoint
- current tool or web-search option is not blocked by missing required arguments
- request still has real user content after removing stale attachments or failed tool outputs
