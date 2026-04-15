# FlexiGPT

[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/license/mpl-2-0)
[![Go Report Card](https://goreportcard.com/badge/github.com/flexigpt/flexigpt-app)](https://goreportcard.com/report/github.com/flexigpt/flexigpt-app)
[![lint](https://github.com/flexigpt/flexigpt-app/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/flexigpt/flexigpt-app/actions/workflows/lint.yml)
[![test](https://github.com/flexigpt/flexigpt-app/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/flexigpt/flexigpt-app/actions/workflows/test.yml)

FlexiGPT is a local-first desktop workspace for multi-provider LLM chats.

It brings together reusable assistant presets, model presets, prompt templates, attachments, tools, and skills in one app while keeping conversations and configuration local.

> Early access
>
> FlexiGPT is under active development. Expect some rough edges, evolving built-ins, and ongoing UX and docs refinement between releases.

## Install

1. Download the latest release from [GitHub Releases](https://github.com/flexigpt/flexigpt-app/releases).
2. Install the package for your platform:
   - macOS: `.pkg`
   - Windows: `.exe`
   - Linux: `.flatpak`
   - [Detailed installation steps are here](./frontend/app/docs/content/00-installation.md)
3. Launch FlexiGPT.

## Quick start

- Get an API key for your provider.
- Add the key in Settings -> Auth Keys.
- Chat.

FlexiGPT does not bill you directly. Usage costs and limits come from the provider account behind the key you configure.

## Key Features

### Multi-provider connectivity with built-in presets

- First-class support for OpenAI, Anthropic, Google Gemini API, DeepSeek, xAI, Hugging Face, OpenRouter, and local `llama.cpp`
- Support for compatible custom endpoints across OpenAI Chat Completions, OpenAI Responses, and Anthropic Messages style APIs
- Built-in providers and curated model presets for leading models so you can get started quickly without manually defining endpoints or defaults first
- API keys are stored securely through the OS keyring, not in plain-text exported settings

### Unified chat workspace

- One interface for chats, tabs, attachments, prompts, tools, skills, presets, search, and exports
- Switch providers or models as you iterate
- Multi-tab conversations with local history search and resume flows
- Export the current conversation as JSON

### Human-in-loop and agentic workflows

- Assistant presets bundle model choice, instructions, tools, and skills into reusable starting setups
- Tools can be attached per conversation and configured for manual review or auto-execution
- When an auto-execute tool is called, FlexiGPT can run it and automatically submit the result back to the model, enabling agentic workflows inside a normal chat flow
- Keep tools manual when you want tighter control over execution

### Rich response rendering and inspection

- Markdown rendering with syntax-highlighted code blocks
- Mermaid diagram rendering with zoom and source or image export workflows
- KaTeX math rendering
- Citations, token usage, and per-message request/response details for inspection and debugging
- Message-level controls for copying, inspection, and follow-up iteration

### Local-first context and history

- Local conversation storage and full-text search
- File, folder, image, PDF, and URL attachments
- Bundled offline docs shipped inside the app
- Use your own provider accounts; FlexiGPT does not proxy or bill model usage

## Documentation

The main docs are bundled inside the app and mirrored in this repository under `frontend/app/docs/content/`.

### User guide

- [Installation](./frontend/app/docs/content/00-installation.md)
- [Getting Started](./frontend/app/docs/content/01-getting-started.md)
- [Core Concepts](./frontend/app/docs/content/02-core-concepts.md)
- [Chats, Composer, and Everyday Workflow](./frontend/app/docs/content/03-chats-composer-and-everyday-workflow.md)
- [Attachments, Tools, Skills, and Prompts](./frontend/app/docs/content/04-attachments-tools-skills-prompts.md)
- [Presets, Providers, and Settings](./frontend/app/docs/content/05-presets-providers-settings.md)
- [Privacy, Storage, and Troubleshooting](./frontend/app/docs/content/06-privacy-storage-and-troubleshooting.md)

### Architecture

- [Architecture Overview](./frontend/app/docs/content/11-architecture-overview.md)
- [Backend Roles and Responsibilities](./frontend/app/docs/content/12-backend-roles-and-responsibilities.md)
- [Frontend Roles and Responsibilities](./frontend/app/docs/content/13-frontend-roles-and-responsibilities.md)
- [Chats Workspace and Composer HLD](./frontend/app/docs/content/14-chats-workspace-and-composer-hld.md)

## Built With

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
