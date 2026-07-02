# Local LLM Setup

FlexiGPT can talk to local and self-hosted LLM runtimes through provider and model presets.

Use this page when you want inference to run on a machine or endpoint you control, such as LM Studio, `llama.cpp`, Ollama, LocalAI, SGLang, vLLM, or another compatible server.

## Table of contents <!-- omit from toc -->

- [Recommended mental model](#recommended-mental-model)
- [Built-in local providers](#built-in-local-providers)
- [Why fork providers before models](#why-fork-providers-before-models)
- [Provider-first setup flow](#provider-first-setup-flow)
- [Model preset setup](#model-preset-setup)
- [Auth keys for local endpoints](#auth-keys-for-local-endpoints)
- [Runtime notes](#runtime-notes)
  - [LM Studio](#lm-studio)
  - [`llama.cpp`](#llamacpp)
  - [Ollama](#ollama)
  - [LocalAI](#localai)
  - [SGLang and vLLM](#sglang-and-vllm)
- [Testing](#testing)
- [Capabilities and limitations](#capabilities-and-limitations)
- [Local-only safety checklist](#local-only-safety-checklist)
- [Troubleshooting](#troubleshooting)

## Recommended mental model

Local LLM setup has two layers:

1. **Provider preset**
   - endpoint and API contract
   - origin URL
   - chat path
   - SDK/API compatibility type
   - API-key header
   - default headers
   - provider-wide capability assumptions
2. **Model preset**
   - model name sent to that provider
   - streaming, timeout, token limits, temperature, reasoning, output format, stop sequences, and model-level capability assumptions

For local models, configure the provider first, then models.

## Built-in local providers

FlexiGPT ships with built-in local and self-hosted provider presets so you can start quickly.

| Built-in provider | Default origin           | Compatibility style in the preset  | Good starting point for                              |
| ----------------- | ------------------------ | ---------------------------------- | ---------------------------------------------------- |
| LocalAI           | `http://127.0.0.1:8080`  | OpenAI Responses-compatible        | LocalAI or compatible local server                   |
| LM Studio         | `http://127.0.0.1:1234`  | OpenAI Responses-compatible        | LM Studio local server                               |
| `llama.cpp`       | `http://127.0.0.1:8080`  | OpenAI Chat Completions-compatible | `llama-server` compatible route                      |
| Ollama            | `http://127.0.0.1:11434` | Anthropic-compatible               | Ollama-compatible route matching the built-in preset |
| SGLang            | `http://127.0.0.1:30000` | OpenAI Responses-compatible        | Self-hosted SGLang                                   |
| vLLM              | `http://127.0.0.1:8000`  | OpenAI Responses-compatible        | Self-hosted vLLM                                     |

These are useful defaults, not fixed requirements.

Use a built-in directly only when your local server matches its origin, path, headers, compatibility type, and model names closely enough.

## Why fork providers before models

Local runtimes vary a lot:

- ports differ
- routes differ
- some servers expose Chat Completions while others expose Responses-style routes
- headers differ
- model names or tags differ
- reasoning, image, file, JSON, tool, and stop-sequence support differs
- a router may expose many models through one endpoint

Built-in providers are read-only. Instead of trying to edit the built-in, create your own provider by copying/forking it.

Do this before model edits because every model under that provider inherits the provider's API contract.

## Provider-first setup flow

1. Start your local runtime.
2. Open **Model Presets**.
3. Click **Add Provider**.
4. Use **Prefill from Existing -> Copy Existing Provider**.
5. Pick the closest built-in local provider.
6. Give the fork a stable provider ID.
   - Example: `my-lmstudio`
   - Example: `workstation-vllm`
   - Example: `ollama-laptop`
7. Give it a clear display name.
8. Adjust provider fields:
   - **SDK Type**
   - **Origin**
   - **Chat Path**
   - **API-Key Header Key**
   - **Default Headers**
9. Save the provider.
10. Enable it.
11. Add a matching auth key if needed.

Keep the provider ID stable after you use it. Chats, assistant presets, and model references may point to it.

## Model preset setup

After the provider is correct:

1. Expand your forked provider in **Model Presets**.
2. Click **Add Model Preset**.
3. Use **Copy Existing Preset** if a built-in model is close to your local model.
4. Set **Model Preset ID** to a stable local ID.
5. Set **Model Name** to the exact model name or tag expected by your local server.
6. Set a friendly **Preset Label**.
7. Keep **Streaming** on if your server supports streaming.
8. Start with conservative values:
   - lower prompt token limit
   - lower output token limit
   - longer timeout for slow local hardware
   - temperature only, unless the local model and runtime support reasoning controls
9. Save and enable the model preset.
10. Optionally set it as the default model for the forked provider.

If a model behaves strangely, first reduce complexity before changing many settings:

- turn off tools
- turn off web search
- avoid images and files
- lower token limits
- reduce **Previous user turns**
- test with plain text only

## Auth keys for local endpoints

Some local servers ignore API keys. Others require a non-empty header value even though the value is not a real secret.

If your local provider requires a key:

1. Open **Settings -> Auth Keys**.
2. Add a provider key for your forked provider ID.
3. Use a harmless placeholder such as `local-placeholder`.

Avoid using real production API keys for dummy local endpoints.

## Runtime notes

### LM Studio

- Start the LM Studio local server.
- The built-in LM Studio provider expects `http://127.0.0.1:1234`.
- If your port or route differs, fork LM Studio and edit the fork.
- Model names often need to match the model identifier exposed by LM Studio.

### `llama.cpp`

- Start `llama-server` with your model file, host, port, and context length.
- Example command shape:

      llama-server -m /path/to/model.gguf --host 127.0.0.1 --port 8080

- The built-in `llama.cpp` provider uses an OpenAI Chat Completions-compatible setup.
- If your server uses another port or path, fork the provider and edit origin/path.
- Context length depends on both the model and server launch flags.

### Ollama

- The built-in Ollama provider uses the compatibility style configured in its preset.
- If your Ollama setup exposes a different route, fork the provider and adjust SDK type and path.
- Use the exact local model tag your server expects.
- If the model tag changes, update or copy the model preset under your forked provider.

### LocalAI

- Start LocalAI with the models and routes you want exposed.
- The built-in LocalAI provider is a good starting point when the server uses the default local origin.
- If LocalAI is behind a reverse proxy or has different auth headers, fork the provider first.

### SGLang and vLLM

- Start the runtime with the model and API mode you intend to use.
- Built-ins target common local ports, but deployments often change ports, paths, and auth.
- For shared team machines, confirm whether the endpoint is reachable only locally, on your LAN, or from a wider network.

## Testing

After provider and model setup, open **Chats**, select the local model preset, and send:

    Reply with "local model test ok" and no extra text.

If that works, test one capability at a time:

1. longer prompt
2. streaming
3. attachments as text
4. images, if the model and runtime support them
5. tools, if the runtime supports tool calling
6. JSON output, if the runtime supports structured output

## Capabilities and limitations

Local runtime support can differ from hosted models.

Common differences:

- smaller usable context window
- slower generation
- limited reasoning controls
- limited or no tool calling
- limited or no file/image support
- inconsistent JSON schema behavior
- stop sequence support may differ
- output may depend heavily on quantization and prompt style

If a feature is not reliable, create a separate model preset with simpler settings instead of overloading one preset for every workflow.

## Local-only safety checklist

Before sending sensitive material to a local model:

- confirm the selected provider is your local or self-hosted provider
- confirm the origin is local, such as `http://127.0.0.1` or a trusted self-hosted address
- confirm the runtime is not proxying to a remote model provider
- use a harmless placeholder key for dummy local auth
- keep **Previous user turns** small
- remove stale attachments
- turn off web search
- avoid network tools unless you need them
- keep raw debug logging off unless actively diagnosing an issue

Local-first app storage does not automatically make every inference request local. The selected provider endpoint decides where inference happens.

## Troubleshooting

If the local model does not appear in Chats:

- provider is enabled
- model preset is enabled
- the provider is not blocked by missing auth key state
- the model preset is under the provider you intended to use

If the request fails:

- local server is running
- origin includes `http://` or `https://`
- origin port is correct
- chat path matches the selected SDK/API compatibility type
- API-key header is what the server expects
- placeholder auth key exists if required
- model name matches what the server exposes

If output is weak or malformed:

- test with plain text only
- reduce prompt and output token limits
- reduce stale history
- lower temperature
- remove tools, web search, files, and images
- use a model-specific prompt style
- verify the same prompt directly against the local server if possible
