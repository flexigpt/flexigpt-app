# FlexiGPT

[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/license/mpl-2-0)
[![Go Report Card](https://goreportcard.com/badge/github.com/flexigpt/flexigpt-app)](https://goreportcard.com/report/github.com/flexigpt/flexigpt-app)
[![lint](https://github.com/flexigpt/flexigpt-app/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/flexigpt/flexigpt-app/actions/workflows/lint.yml)
[![test](https://github.com/flexigpt/flexigpt-app/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/flexigpt/flexigpt-app/actions/workflows/test.yml)

FlexiGPT is a local-first BYOK AI workspace for power users and teams who need repeatable prompts, tools, skills, model choices, assistants/agents, and private local history across multiple LLM providers.

## Who FlexiGPT is for

FlexiGPT is built for people who use LLMs as part of repeatable work, not just one-off chat.

- Power/Local-first users who want provider choice, private local history, and full control over configuration and orchestration.
- Developers and technical writers who reuse assistants/agents, prompts, attachments, tools, and model setups.
- Consultants and small teams who want consistent assistant workflows without sending chat history through another hosted app.

## Install

1. Download the latest release from [GitHub Releases](https://github.com/flexigpt/flexigpt-app/releases)
   - macOS: `.pkg`, Linux: `.flatpak`, Windows: `.exe`
2. Install the package. [Detailed installation steps are here](./frontend/app/docs/content/00-installation.md)
3. Launch FlexiGPT

## Quick start

- Get an API key for your provider.
  - [OpenAI](https://platform.openai.com/settings/organization/api-keys), [Anthropic Claude](https://platform.claude.com/settings/keys), [Google Gemini](https://aistudio.google.com/api-keys), [xAI](https://console.x.ai/team/default/api-keys), [MistralAI](https://console.mistral.ai/home?profile_dialog=api-keys)
  - [OpenRouter](https://openrouter.ai/workspaces/default/keys), [Hugging Face](https://huggingface.co/settings/tokens)
- Add the key in Settings -> Auth Keys.
- Chat. Start from a built-in assistant preset or build your own reusable workflow with prompts, attachments, tools, and skills.

Good first workflows:

- Open the home screen and choose one of the built-in workflow cards: Analyze File, Code Review, Bug Investigation, or Architecture Review.
- Attach the relevant file, folder, notes, PDF, URL, or code snippet.
- Send the prefilled prompt as-is or adjust it for your task.
- If you need editing or shell access, switch to Local Editor or Local Developer Assistant Preset.
- Reuse or customize the assistant preset once the workflow fits your style.

FlexiGPT does not bill you directly. Usage costs and limits come from the provider account behind the key you configure.
FlexiGPT does not proxy normal LLM calls through a FlexiGPT-hosted service; requests go directly to the provider or compatible endpoint you configure.

## Screenshots

<p float="left">
  <img src="images/home.png" alt="FlexiGPT local-first workspace home" width="512" />
  <img src="images/latex_code.png" alt="Rich chat rendering with Markdown, LaTeX, and code" width="512" />
  <img src="images/mermaid.png" alt="Rich chat rendering with Mermaid" width="512" />
</p>

<p float="left">
  <img src="images/assistants.png" alt="Reusable assistant presets and workflow setup" width="512" />
  <img src="images/settings.png" alt="Local settings and provider auth keys" width="512" />
</p>

[All images are here](./images/)

## Key features

### Provider-independent model choices with built-in presets

- First-class built-in support for OpenAI, Anthropic, Google Gemini API, xAI, Mistral, Hugging Face, OpenRouter, and local `llama.cpp`
- Support for compatible custom endpoints across OpenAI Chat Completions, OpenAI Responses, Anthropic Messages, and Google GenerateContent style APIs
- Built-in providers and curated model presets for leading models so you can start quickly without manually defining endpoints or defaults first
- API keys are stored securely through the OS keyring, not in plain-text exported settings

### Repeatable AI workspace

- One interface for chats, tabs, reusable assistant presets, model presets, prompt templates, attachments, tools, skills, search, and exports
- Build repeatable workflows by combining model choices, instructions, attachments, tools, and skills
- Switch providers or models as you iterate
- Multi-tab conversations with local history search and resume flows
- Export the current conversation as JSON

### Assistants, tools, and agentic workflows with human-in-loop controls

- Assistant presets bundle model choice, instructions, tools, and skills into reusable starting setups
- Tools can be attached per conversation and configured for manual review or auto-execution
- When an auto-execute tool is called, FlexiGPT can run it and automatically submit the result back to the model, enabling assistant/agent-style workflows inside a normal chat flow
- Keep tools manual when you want tighter control over execution

### Rich response rendering and inspection

- Markdown rendering with syntax-highlighted code blocks
- Mermaid diagram rendering with zoom and source or image export workflows
- KaTeX math rendering
- Citations, token usage, and per-message request/response details for inspection and debugging
- Message-level controls for copying, inspection, and follow-up iteration

### Private local context and history

- Local conversation storage and full-text search
- File, folder, image, PDF, and URL attachments
- Bundled offline docs shipped inside the app
- Conversations, workflow catalogs, and configuration are stored locally; selected request context is sent to the provider or endpoint you choose when you send.
- Use your own provider accounts; FlexiGPT does not proxy or bill model usage.

## Documentation

### Repository-only install notes

- [Installation](./frontend/app/docs/content/00-installation.md)

### Bundled in-app user guide

- [Getting Started](./frontend/app/docs/content/01-getting-started.md)
- [Core Concepts](./frontend/app/docs/content/02-core-concepts.md)
- [Chats, Composer, and Everyday Workflow](./frontend/app/docs/content/03-chats-composer-and-everyday-workflow.md)
- [Attachments, Tools, Skills, and Prompts](./frontend/app/docs/content/04-attachments-tools-skills-prompts.md)
- [Presets, Providers, and Settings](./frontend/app/docs/content/05-presets-providers-settings.md)
- [Storage, Data Control, and Troubleshooting](./frontend/app/docs/content/06-storage-data-control-and-troubleshooting.md)
- [Security, Privacy, and Trust Model](./frontend/app/docs/content/07-security-privacy-and-trust-model.md)
- [Recipes and Starter Workflows](./frontend/app/docs/content/08-recipes-and-starter-workflows.md)
- [Provider and Local Model Setup](./frontend/app/docs/content/09-provider-and-local-model-setup.md)

### Bundled in-app architecture reference

- [Architecture Overview](./frontend/app/docs/content/11-architecture-overview.md)
- [Backend Roles and Responsibilities](./frontend/app/docs/content/12-backend-roles-and-responsibilities.md)
- [Frontend Roles and Responsibilities](./frontend/app/docs/content/13-frontend-roles-and-responsibilities.md)
- [Chats Workspace and Composer Design](./frontend/app/docs/content/14-chats-workspace-and-composer-design.md)

## Built with

- Data storage: `JSON` and `SQLite` files in local filesystem.
- [Go](https://go.dev/) backend.
- [Wails](https://wails.io/) as a desktop application building platform.
- Official Go SDKs by [OpenAI](https://github.com/openai/openai-go), [Anthropic](https://github.com/anthropics/anthropic-sdk-go), and [Google GenAI](https://github.com/googleapis/go-genai).
- [Vite](https://vite.dev/) + [React Router v7](https://reactrouter.com/) frontend in [Typescript](https://www.typescriptlang.org/). [DaisyUI](https://daisyui.com/) with [TailwindCSS](https://tailwindcss.com/) for styling.
- Tooling: [Golangci-lint](https://golangci-lint.run/), [Knip](https://knip.dev/), [ESLint](https://eslint.org/), [Prettier](https://prettier.io/), [GitHub actions](https://github.com/features/actions)

## Contributing

Developer setup is documented in: [devsetup.md](./docs/contributing/devsetup.md)

## License

Copyright (c) 2024 - Present - Pankaj Pipada

All source code in this repository, unless otherwise noted, is licensed under the Mozilla Public License, v. 2.0. See [`LICENSE`](./LICENSE) for details.
