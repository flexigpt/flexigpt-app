# Provider and Local Model Setup

FlexiGPT is BYOK: bring your own provider key or compatible local endpoint.

The usual setup path is:

1. Create or choose a provider/model preset in **Model Presets**.
2. Add the matching key in **Settings -> Auth Keys** or from the provider card.
3. Set a default provider and default model.
4. Open **Chats** and send a small test prompt.

- [OpenRouter](#openrouter)
- [Custom compatible endpoints](#custom-compatible-endpoints)
- [Local OpenAI-compatible servers](#local-openai-compatible-servers)
- [llama.cpp server](#llamacpp-server)
- [Ollama-style local setup](#ollama-style-local-setup)
- [Troubleshooting provider setup](#troubleshooting-provider-setup)
- [Local-first reminder](#local-first-reminder)

## OpenRouter

OpenRouter is useful when you want one endpoint that can reach many hosted models.

Steps:

1. Create an OpenRouter API key from OpenRouter.
2. In FlexiGPT, open **Settings -> Auth Keys**.
3. Add or update the key for the built-in OpenRouter provider.
4. Open **Model Presets**.
5. Ensure OpenRouter and the desired model preset are enabled.
6. Set OpenRouter as default if you want it to be the first chat option.

Test prompt:

```text
Reply with one sentence confirming the provider and model you are using.
```

## Custom compatible endpoints

Use a custom provider when your endpoint implements a supported API style.

Supported setup styles in the UI include:

- OpenAI Chat Completions-compatible
- OpenAI Responses-compatible
- Anthropic Messages-compatible
- Google GenerateContent-compatible

When adding a provider, check:

- SDK/API compatibility type
- origin, for example `https://api.example.com`
- chat path, for example `/v1/chat/completions`
- API-key header name
- default headers JSON
- at least one model preset under the provider

Keep provider IDs stable because model presets and workflows may reference them.

## Local OpenAI-compatible servers

Many local model servers expose OpenAI-compatible endpoints.
Use the custom provider flow and choose the OpenAI Chat Completions-compatible SDK type when the server supports `/v1/chat/completions`.

Typical local values:

- Origin: `http://localhost:11434` or `http://localhost:8080`
- Chat path: `/v1/chat/completions`
- API-key header: often `Authorization`, but some local servers ignore it
- API key: use a placeholder only if the server requires a non-empty value

If a local server does not require a key, you may still need to add a local placeholder key depending on provider configuration.
Do not reuse real production keys for local dummy endpoints.

## llama.cpp server

If you run a `llama.cpp` server with OpenAI-compatible routes, configure it as an OpenAI-compatible local provider.

Example server command shape:

```shell
llama-server -m /path/to/model.gguf --host 127.0.0.1 --port 8080
```

Then configure:

- Origin: `http://127.0.0.1:8080`
- Chat path: `/v1/chat/completions`
- Model name: the model name expected by your server

Limitations depend on the model and server build:

- context length may be lower than hosted models
- tool support may be limited or unavailable
- multimodal support depends on the local stack
- performance depends on hardware and quantization

## Ollama-style local setup

If your Ollama setup exposes an OpenAI-compatible endpoint, use it through a compatible custom provider.

Typical values:

- Origin: `http://localhost:11434`
- Chat path: `/v1/chat/completions`
- Model name: the local model tag, for example `llama3.1`

If your local Ollama endpoint differs, use the path and model name from your local server configuration.

## Troubleshooting provider setup

If the first request fails:

- confirm the provider is enabled
- confirm the model preset is enabled
- confirm an auth key exists and is non-empty when required
- confirm the origin includes the scheme, such as `http://` or `https://`
- confirm the chat path matches the selected SDK/API compatibility type
- try a tiny prompt before testing long attachments
- check debug logs only if needed, because raw request/response logging can expose sensitive content

If the model responds but quality is weak:

- raise max prompt/output limits if too low
- choose a stronger model preset
- reduce stale history through **Previous user turns**
- attach only the source files that matter
- avoid enabling tools or web search unless needed

## Local-first reminder

Local providers keep inference on your machine only when the endpoint and model are truly local.
Hosted compatible endpoints still receive request content.
Review the provider origin before sending sensitive work.
