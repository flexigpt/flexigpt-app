# Providers and Models

FlexiGPT is BYOK: bring your own provider key, custom compatible endpoint, or local model server.

This page covers hosted providers, custom endpoints, and local model setup.

## Table of contents <!-- omit from toc -->

- [Usual setup path](#usual-setup-path)
- [Built-in hosted providers](#built-in-hosted-providers)
- [OpenRouter](#openrouter)
- [Built-in local and self-hosted runtimes](#built-in-local-and-self-hosted-runtimes)
- [Fork a local provider before editing models](#fork-a-local-provider-before-editing-models)
- [Custom compatible endpoints](#custom-compatible-endpoints)
- [Local OpenAI-compatible servers](#local-openai-compatible-servers)
- [Runtime-specific local notes](#runtime-specific-local-notes)
  - [LM Studio](#lm-studio)
  - [`llama.cpp`](#llamacpp)
  - [Ollama](#ollama)
  - [LocalAI, SGLang, and vLLM](#localai-sglang-and-vllm)
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

## Built-in local and self-hosted runtimes

FlexiGPT includes built-in provider and model presets for common local or self-hosted runtimes:

| Built-in provider | Default origin           | Compatibility style in the preset  | Typical use                                                |
| ----------------- | ------------------------ | ---------------------------------- | ---------------------------------------------------------- |
| LocalAI           | `http://127.0.0.1:8080`  | OpenAI Responses-compatible        | LocalAI or compatible local server                         |
| LM Studio         | `http://127.0.0.1:1234`  | OpenAI Responses-compatible        | LM Studio local server                                     |
| `llama.cpp`       | `http://127.0.0.1:8080`  | OpenAI Chat Completions-compatible | `llama-server` with compatible routes                      |
| Ollama            | `http://127.0.0.1:11434` | Anthropic-compatible               | Ollama-compatible local route matching the built-in preset |
| SGLang            | `http://127.0.0.1:30000` | OpenAI Responses-compatible        | Self-hosted SGLang endpoint                                |
| vLLM              | `http://127.0.0.1:8000`  | OpenAI Responses-compatible        | Self-hosted vLLM endpoint                                  |

These presets are defaults, not guarantees. Local runtime behavior varies by server version, launched model, command-line flags, routing layer, and hardware.

Use a built-in provider directly when your server matches the default origin, path, headers, and model names. Otherwise, copy/fork the provider first and edit the fork.

For a guided local setup, see [Local LLM Setup](/docs?doc=local-llm-setup).

## Fork a local provider before editing models

For local LLMs, prefer this order:

1. fork/copy the provider preset
2. edit provider settings
3. then add or copy model presets under that provider

Provider settings come first because they control the shared API contract:

- SDK/API compatibility type
- origin URL
- chat path
- API-key header name
- default headers
- provider-wide capability assumptions

Recommended flow:

1. Open **Model Presets**.
2. Click **Add Provider**.
3. Use **Prefill from Existing -> Copy Existing Provider**.
4. Choose the closest built-in local provider, such as LM Studio, `llama.cpp`, Ollama, LocalAI, SGLang, or vLLM.
5. Give the new provider a stable provider ID, such as `my-lmstudio` or `workstation-vllm`.
6. Adjust origin, path, SDK type, headers, and API-key header for your server.
7. Save and enable the provider.
8. Add a placeholder auth key in **Settings -> Auth Keys** if the server requires one.
9. Add, copy, or edit model presets under the forked provider.
10. Select the forked model preset in **Chats** and send a tiny test prompt.

Keep provider IDs stable after use. Chats, assistant presets, and model references may depend on them.

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
- at least one model preset exists under the provider

Keep provider IDs stable because chats, assistant presets, and model refs may depend on them.

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
2. Open **Model Presets**.
3. Copy/fork the closest built-in local provider if one exists.
4. Otherwise add a provider preset from scratch.
5. Choose the SDK type that matches the server route.
6. Set origin and chat path.
7. Add a model preset with the local model name.
8. Enable provider and model.
9. Add a placeholder key in **Settings -> Auth Keys** if required.
10. Open **Chats** and test.

Test prompt:

    Reply with "local endpoint works" and no extra text.

Local endpoints may not support every hosted-model feature. Start without tools, web search, files, or images. Add capabilities one at a time after basic chat works.

## Runtime-specific local notes

Use these notes after reading the provider-first flow above.

### LM Studio

- Start the LM Studio local server.
- If the default port is still `1234`, try the built-in LM Studio provider directly.
- If you changed the port, base URL, or route style, copy/fork LM Studio first and edit the fork.
- Model names often need to match the repository/model identifier exposed by LM Studio.

### `llama.cpp`

- Start `llama-server` with the host, port, context, and model file you want.
- The built-in `llama.cpp` provider uses an OpenAI Chat Completions-compatible setup on `http://127.0.0.1:8080`.
- If your launch command uses another port or path, fork the provider and update the origin/path.
- Context length and multimodal support depend on the model file and server flags.

### Ollama

- The built-in Ollama provider uses the compatibility style configured in its preset.
- If your Ollama setup exposes a different route, such as an OpenAI-compatible route, fork the provider and adjust SDK type and path to match your server.
- Use the local model tag that your Ollama instance expects.

### LocalAI, SGLang, and vLLM

- Start the server with the model and route style you intend to use.
- If the built-in default port matches, enable the built-in provider and test.
- If the server uses a different port, proxy path, auth header, or feature set, fork the provider first.
- For self-hosted deployments, confirm whether the endpoint is still local-only or reachable on your network.

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
