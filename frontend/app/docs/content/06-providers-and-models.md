# Providers and Models

FlexiGPT is BYOK: bring your own provider key, custom compatible endpoint, or local model server.

This page covers hosted providers, custom endpoints, and local model setup.

## Table of contents <!-- omit from toc -->

- [Usual setup path](#usual-setup-path)
- [Built-in hosted providers](#built-in-hosted-providers)
- [OpenRouter](#openrouter)
- [Custom compatible endpoints](#custom-compatible-endpoints)
- [Local OpenAI-compatible servers](#local-openai-compatible-servers)
- [llama.cpp server](#llamacpp-server)
- [Ollama-style local setup](#ollama-style-local-setup)
- [Capability expectations](#capability-expectations)
- [Troubleshooting provider setup](#troubleshooting-provider-setup)
- [Local-first reminder](#local-first-reminder)

## Usual setup path

1. Open **Model Presets**.
2. Enable or create a provider preset.
3. Enable or create at least one model preset under that provider.
4. Add the matching key in **Settings -> Auth Keys**.
5. Set default provider/model if desired.
6. Open **Chats**.
7. Select the model preset.
8. Send a tiny test prompt before using attachments or tools.

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
2. Open **Settings -> Auth Keys**.
3. Add or update the key for the matching provider.
4. Open **Model Presets**.
5. Confirm the provider is enabled.
6. Confirm the model preset is enabled.
7. Optionally set the provider/model as default.
8. Open **Chats** and send a tiny prompt.

If the model does not appear in Chats, check:

- provider is enabled
- model preset is enabled
- auth key exists and is non-empty if required
- provider SDK/API type is configured correctly
- provider origin and path are correct
- model preset is not disabled by missing capability or key state

## OpenRouter

OpenRouter is useful when you want one endpoint that can route to many hosted models.

Steps:

1. Create an OpenRouter API key.
2. Open **Settings -> Auth Keys**.
3. Add or update the key for the built-in OpenRouter provider.
4. Open **Model Presets**.
5. Enable the OpenRouter provider.
6. Enable the OpenRouter model preset you want.
7. Set OpenRouter as default if desired.
8. Open **Chats**.
9. Select an OpenRouter model preset.
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

## Custom compatible endpoints

Use a custom provider when your endpoint implements one of the supported API styles:

- OpenAI Chat Completions-compatible
- OpenAI Responses-compatible
- Anthropic Messages-compatible
- Google GenerateContent-compatible

When adding a custom provider, check:

- provider name is stable
- SDK/API compatibility type matches the endpoint
- origin includes scheme, such as `http://` or `https://`
- chat path matches the selected API type
- API-key header name is correct
- default headers are valid JSON if provided
- at least one model preset exists under the provider

Keep provider IDs stable because chats, assistant presets, and model refs may depend on them.

## Local OpenAI-compatible servers

Many local servers expose an OpenAI-compatible API.

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
2. Open **Model Presets**.
3. Add a provider preset.
4. Choose OpenAI Chat Completions-compatible SDK type if your server supports `/v1/chat/completions`.
5. Set origin and chat path.
6. Add a model preset with the local model name.
7. Enable provider and model.
8. Add a placeholder key in **Settings -> Auth Keys** if required.
9. Open **Chats** and test.

Test prompt:

    Reply with "local endpoint works" and no extra text.

Local endpoints may not support every hosted-model feature. Start without tools, web search, files, or images. Add capabilities one at a time after basic chat works.

## llama.cpp server

If you run a `llama.cpp` server with OpenAI-compatible routes, configure it as an OpenAI-compatible provider.

Example server command shape:

    llama-server -m /path/to/model.gguf --host 127.0.0.1 --port 8080

Then configure:

- Origin: `http://127.0.0.1:8080`
- Chat path: `/v1/chat/completions`
- SDK type: OpenAI Chat Completions-compatible
- Model name: the model name expected by your server

Common limitations:

- lower context length
- limited tool support
- limited structured output support
- limited multimodal support
- slower generation
- quantization-specific quality trade-offs

## Ollama-style local setup

If your Ollama setup exposes an OpenAI-compatible endpoint, use the compatible custom provider path.

Typical values:

- Origin: `http://localhost:11434`
- Chat path: `/v1/chat/completions`
- SDK type: OpenAI Chat Completions-compatible
- Model name: local model tag, for example `llama3.1`

If your endpoint differs, use the path and model name from your local Ollama setup.

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
- assistant preset modified state
- web-search chip
- tool chips
- advanced parameters
- attachment modes

## Troubleshooting provider setup

If the first request fails, check:

- provider enabled
- model preset enabled
- auth key exists and is non-empty if required
- origin includes `http://` or `https://`
- chat path matches SDK/API compatibility type
- local server is running
- local server accepts the configured model name
- firewall or proxy is not blocking the request
- tiny prompt works before attachments or tools

If the request works but quality is weak, try:

- stronger model preset
- higher prompt/output limits
- fewer stale previous user turns
- more focused attachments
- clearer system prompt
- lower temperature for stricter work
- stronger reasoning level if supported

If tools or web search do not appear, check:

- selected provider SDK type
- tool/provider compatibility
- web-search tool compatibility
- tool bundle enabled
- tool enabled
- assistant preset availability reason
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
