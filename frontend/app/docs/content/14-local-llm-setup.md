# Local LLM Setup

FlexiGPT can talk to local and self-hosted LLM runtimes through provider and model configuration in **Models**.

Use this page when you want inference to run on a machine or endpoint you control, such as LM Studio, `llama.cpp`, Ollama, LocalAI, SGLang, vLLM, or another compatible server.

## Table of contents <!-- omit from toc -->

- [Recommended mental model](#recommended-mental-model)
- [Built-in local providers](#built-in-local-providers)
- [Why fork providers before models](#why-fork-providers-before-models)
- [Provider-first setup flow](#provider-first-setup-flow)
- [Model setup](#model-setup)
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

Local LLM setup has two reusable layers:

1. **Provider configuration**
   - endpoint and API contract
   - origin URL
   - chat path
   - SDK/API compatibility type
   - API-key header
   - default headers
   - provider-wide capability assumptions
2. **Model configuration**
   - model name sent to that provider
   - streaming, timeout, token limits, temperature, reasoning, output format, stop sequences, and model-level capability assumptions

For local models, configure the provider first, then models.

## Built-in local providers

FlexiGPT may ship local or self-hosted provider presets so you can start quickly.
The exact presets available in your installation are shown in **Models**.

Common local-runtime starting points include LocalAI, LM Studio, `llama.cpp`,
Ollama, SGLang, and vLLM. Their actual API route, port, model name, headers,
and supported features are runtime-specific.

Use a preset directly only when its displayed origin, path, headers,
compatibility type, and model configuration match your server closely enough.

## Why fork providers before models

Local runtimes vary a lot:

- ports differ
- routes differ
- some servers expose Chat Completions while others expose Responses-style routes
- headers differ
- model names or tags differ
- reasoning, image, file, JSON, tool, and stop-sequence support differs
- a router may expose many models through one endpoint

Built-in providers are read-only. Instead of trying to edit the built-in, create your own provider by copying/forking it in **Models**.

Do this before model edits because every model under that provider inherits the provider's API contract.

## Provider-first setup flow

1. Start your local runtime.
2. Open **Models**.
3. Create a provider by copying the closest built-in local provider.
4. Give it a clear display name.
5. Adjust provider fields:
   - **SDK Type**
   - **Origin**
   - **Chat Path**
   - **API-Key Header**
   - **Default Headers**
6. Save and enable the provider.
7. Add a matching credential in **Models** if the endpoint needs one.

## Model setup

After the provider is correct:

1. Expand your forked provider in **Models**.
2. Add a model.
3. Use **Copy an Existing Model** if an existing model is a close starting point.
4. Set the model's display name and provider-facing **Model** value.
5. Use the exact model name or tag expected by the local server.
6. Keep **Streaming** enabled only if your server supports it.
7. Start with conservative values:
   - lower prompt token limit
   - lower output token limit
   - longer timeout for slow local hardware
   - temperature only, unless the local model and runtime support reasoning controls
8. Save and enable the model.
9. Optionally set it as the provider's default model.

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

1. Open **Models**.
2. Add a provider key alongside your forked provider/model setup.
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

After provider and model setup, open **Chats**, select the local model, and send:

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

If a feature is not reliable, create a separate model with simpler settings instead of overloading one model configuration for every workflow.

## Local-only safety checklist

Before sending sensitive material to a local model, confirm the selected
provider origin, server routing, selected model, history window, attachments,
MCP selections, tools, web search, and debug settings.

The selected provider endpoint decides where inference happens. See
[Privacy, Data, and Troubleshooting](/docs?doc=privacy-data-and-troubleshooting#sensitive-work-checklist)
for the complete safety checklist.

## Troubleshooting

If the local model does not appear in Chats:

- provider is enabled
- model is enabled
- the provider is not blocked by missing auth key state
- the model is under the provider you intended to use

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
