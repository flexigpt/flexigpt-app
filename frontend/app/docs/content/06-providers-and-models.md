# Providers and Models

FlexiGPT is BYOK: bring your own provider key, custom compatible endpoint, or local model server.

This page owns the reusable provider and model setup. It does not describe
every one-turn composer override; use the Chat controls for temporary request
preferences.

## Table of contents <!-- omit from toc -->

- [Usual setup path](#usual-setup-path)
- [Built-in hosted providers](#built-in-hosted-providers)
- [OpenRouter](#openrouter)
- [Local and self-hosted endpoints](#local-and-self-hosted-endpoints)
- [Copy a provider before adapting a local endpoint](#copy-a-provider-before-adapting-a-local-endpoint)
- [Custom compatible endpoints](#custom-compatible-endpoints)
- [Model defaults and availability](#model-defaults-and-availability)
- [Local OpenAI-compatible servers](#local-openai-compatible-servers)
- [Runtime-specific local notes](#runtime-specific-local-notes)
- [Capability expectations](#capability-expectations)
- [Troubleshooting provider setup](#troubleshooting-provider-setup)
- [Local-first reminder](#local-first-reminder)

## Usual setup path

1. Open **Models**.
2. Enable a provider preset, or choose **Add Provider** for a custom endpoint.
3. Select the API compatibility style, origin, path, key header, and non-secret default headers that match the endpoint.
4. Save a provider API key if the endpoint requires one.
5. Add or enable at least one model under that provider.
6. Set a default provider and provider default model if useful.
7. Confirm both provider and model are enabled.
8. Open **Chats**.
9. Select the model.
10. Send a tiny test prompt before using attachments or tools.

Test prompt:

    Reply with one sentence confirming this provider and model are working.

## Built-in hosted providers

Built-in hosted providers can include:

- OpenAI
- Anthropic Claude
- Google Gemini API
- xAI
- Mistral AI
- Hugging Face
- OpenRouter

Steps:

1. Create an API key in the provider console.
2. Open **Models**.
3. Add or update the key alongside the matching provider/model setup.
4. Stay in **Models**.
5. Confirm the provider is enabled.
6. Confirm the model is enabled.
7. Optionally set the provider/model as default.
8. Open **Chats** and send a tiny prompt.

If the model does not appear in Chats, check:

- provider is enabled
- model is enabled
- auth key exists and is non-empty if required
- provider SDK/API type is configured correctly
- provider origin and path are correct
- model preset is not disabled by missing capability or key state

## OpenRouter

OpenRouter is useful when you want one endpoint that can route to many hosted models.

Steps:

1. Create an OpenRouter API key.
2. Open **Models**.
3. Add or update the key for the built-in OpenRouter provider.
4. Stay in **Models**.
5. Enable the OpenRouter provider.
6. Enable the OpenRouter model you want.
7. Set OpenRouter as default if desired.
8. Open **Chats**.
9. Select an OpenRouter model.
10. Send a tiny test prompt.

Test prompt:

    Reply with one sentence confirming the provider and model you are using.

OpenRouter is not a local model path. Request content goes to OpenRouter and is handled according to the selected OpenRouter model/provider behavior.

Capabilities can vary by model:

- context size
- tool support
- web search support
- reasoning parameters
- output verbosity
- JSON schema output
- multimodal input

When comparing OpenRouter models, keep everything constant except the model preset.

## Local and self-hosted endpoints

The **Models** page may include presets for common local or self-hosted runtimes.
Treat every preset as a starting point, not a guarantee of a server's port,
path, authentication behavior, model name, or capability support.

Use [Local LLM Setup](/docs?doc=local-llm-setup) for the local endpoint
workflow and runtime-specific checks.

## Copy a provider before adapting a local endpoint

For local LLMs, prefer this order:

1. add a provider by copying the closest existing provider
2. edit provider settings
3. then add or copy models under that provider

Provider settings come first because they control the shared API contract:

- SDK/API compatibility type
- origin URL
- chat path
- API-key header name
- default headers
- provider-wide capability assumptions

Recommended flow:

1. Open **Models**.
2. Choose **Add Provider** and, when useful, use **Copy an Existing Provider**.
3. Give the provider a clear display name.
4. Adjust origin, path, SDK type, headers, and API-key header for your server.
5. Save and enable the provider.
6. Add a placeholder credential only if the local server requires a non-empty value.
7. Add, copy, or edit models under that provider.
8. Select the model in **Chats** and send a tiny test prompt.

Copying a provider deliberately does not copy its credential.

## Custom compatible endpoints

Use a custom provider when your endpoint implements one of the supported API styles:

- OpenAI Chat Completions-compatible
- OpenAI Responses-compatible
- Anthropic Messages-compatible
- Google GenerateContent-compatible

When adding or forking a custom provider, check:

- provider name is stable
- SDK/API compatibility type matches the endpoint
- origin includes scheme, such as `http://` or `https://`
- chat path matches the selected API type
- API-key header name is correct
- default headers are valid JSON if provided
- at least one model exists under the provider

## Model defaults and availability

A model record stores the provider-facing model name and model-specific defaults.
The available fields depend on the selected provider and model capabilities, but can include:

- streaming
- temperature or reasoning settings
- request timeout and token limits
- cache control
- system prompt and stop sequences
- output verbosity or JSON Schema output

A model is runnable in Chats only when its provider is available and enabled,
the provider has a credential when required, and the model itself is available
and enabled.

Use the provider card to choose its default model. A default provider or model
must be changed before it can be disabled or deleted.

## Local OpenAI-compatible servers

Many local servers expose an OpenAI-compatible API, but they do not all expose the same API family. Check whether your server expects Chat Completions, Responses, or another compatible route before choosing the SDK type.

Typical values:

| Field          | Example                                              |
| -------------- | ---------------------------------------------------- |
| Origin         | `http://127.0.0.1:8080` or `http://localhost:11434`  |
| Chat path      | `/v1/chat/completions`                               |
| API-key header | often `Authorization`                                |
| API key        | placeholder if the server requires a non-empty value |
| Model name     | whatever your local server expects                   |

Steps:

1. Start the local server.
2. Open **Models**.
3. Copy/fork the closest built-in local provider if one exists.
4. Otherwise add a provider preset from scratch.
5. Choose the SDK type that matches the server route.
6. Set origin and chat path.
7. Add a model with the local model name.
8. Enable provider and model.
9. Add a placeholder key in **Models** if required.
10. Open **Chats**, select the local model, and test.

Test prompt:

    Reply with "local endpoint works" and no extra text.

Local endpoints may not support every hosted-model feature. Start without tools, web search, files, or images. Add capabilities one at a time after basic chat works.

## Runtime-specific local notes

Keep runtime-specific launch instructions and endpoint details with the local
runtime itself. In FlexiGPT, verify the runtime against the values you entered
in **Models**:

- server origin and port
- API compatibility style and path
- API-key header behavior
- model name or tag
- supported features such as streaming, tool calls, files, images, and JSON output

## Capability expectations

Changing the selected provider/model can change available features.

Possible differences:

- web search may disappear when SDK type changes
- SDK tools are filtered by provider SDK compatibility
- reasoning controls may change
- temperature may be unavailable when reasoning is active for some models
- output verbosity may not be supported
- JSON schema output may not be supported
- file/image attachment modes may not be supported
- stop sequences may be disabled with reasoning for some models

If a workflow breaks after switching models, inspect:

- selected provider/model
- selected Agent
- web-search chip
- tool chips
- advanced parameters
- attachment modes

## Troubleshooting provider setup

If the first request fails, check:

- provider enabled
- model enabled
- auth key exists and is non-empty if required
- origin includes `http://` or `https://`
- chat path matches SDK/API compatibility type
- local server is running
- local server accepts the configured model name
- firewall or proxy is not blocking the request
- tiny prompt works before attachments or tools

If the request works but quality is weak, try:

- stronger model
- higher prompt/output limits
- fewer stale previous user turns
- more focused attachments
- stronger instruction skill or Agent
- lower temperature for stricter work
- stronger reasoning level if supported

If tools or web search do not appear, check:

- selected provider SDK type
- tool/provider compatibility
- web-search tool compatibility
- tool plugin enabled
- tool enabled
- Agent availability reason
- model capability support

If local model output is strange, try:

- smaller prompt
- no attachments
- no tools
- no web search
- lower temperature
- shorter history
- model-specific prompt style
- confirm model name and endpoint path

## Local-first reminder

FlexiGPT stores conversations and workflow catalogs locally. Inference requests go to the selected provider or endpoint.

For local-only inference:

- confirm provider origin is local, such as `http://127.0.0.1`
- confirm the local server is not proxying to a remote service
- avoid real production API keys for dummy local endpoints
- test with a harmless prompt first
