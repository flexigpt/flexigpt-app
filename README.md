# FlexiGPT

[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/license/mpl-2-0)
[![Go Report Card](https://goreportcard.com/badge/github.com/flexigpt/flexigpt-app)](https://goreportcard.com/report/github.com/flexigpt/flexigpt-app)
[![lint](https://github.com/flexigpt/flexigpt-app/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/flexigpt/flexigpt-app/actions/workflows/lint.yml)
[![test](https://github.com/flexigpt/flexigpt-app/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/flexigpt/flexigpt-app/actions/workflows/test.yml)

FlexiGPT is a local-first desktop app for working with multiple LLM providers from one workspace.

Use your own provider keys, switch between models and providers, keep conversations and search local, and build repeatable agentic/human-in-loop workflows with assistant presets, prompt templates, tools and skills.

> Early access
>
> FlexiGPT is under active development. Expect breaking changes, evolving built-ins, and incomplete areas between releases.

- [Why FlexiGPT](#why-flexigpt)
- [Quick start](#quick-start)
- [Key Features](#key-features)
  - [Multi-provider connectivity with built-in presets](#multi-provider-connectivity-with-built-in-presets)
  - [Unified chat workspace](#unified-chat-workspace)
  - [Human-in-loop and agentic workflows](#human-in-loop-and-agentic-workflows)
  - [Rich response rendering and inspection](#rich-response-rendering-and-inspection)
  - [Local-first context and history](#local-first-context-and-history)
- [Documentation](#documentation)
- [Install](#install)
  - [MacOS](#macos)
  - [Windows](#windows)
  - [Linux](#linux)
- [Built With](#built-with)
- [Contributing](#contributing)
- [License](#license)

## Why FlexiGPT

- One desktop workspace for multiple provider families and compatible APIs
- Reusable assistant presets, model presets, prompts, tools, and skills. Rich builtins to get you started and fully customizable to suit your specific needs.
- In chat, agentic flows through tunable tool auto-executes.
- File, folder, image, PDF, and URL context support
- Built for iterative, real-world usage: compare models, attach context, reuse setups, inspect responses, and keep your workspace local by default.

## Quick start

1. Download the latest release from [GitHub Releases](https://github.com/flexigpt/flexigpt-app/releases).
2. Install the package for your platform:
   - macOS: `.pkg`
   - Windows: `.exe`
   - Linux: `.flatpak`
3. Launch FlexiGPT and open **Settings**.
4. In **Auth Keys**, add at least one provider API key.
5. Open **Chats**.
6. Pick an **Assistant Preset** and **Model Preset**, or keep the defaults.
7. Type a message, optionally add attachments, prompts, tools, or skills, then send.

FlexiGPT does not bill you directly. Usage costs and rate limits come from the provider account behind the API key you add.

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

Detailed usage docs live in [`frontend/app/docs/content/`](./frontend/app/docs/content/) and are also bundled into the app.

Recommended reading order:

| Doc                                                                                        | What it covers                                                                                                                             |
| ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------ |
| [Getting started](./frontend/app/docs/content/01-getting-started.md)                       | First-run setup and your first successful chat                                                                                             |
| [Core concepts](./frontend/app/docs/content/02-core-concepts.md)                           | The main FlexiGPT concepts: providers, assistant presets, model presets, prompts, tools (+ auto-execute), skills, attachments, and context |
| [App tour and chat workflow](./frontend/app/docs/content/03-app-tour-and-chat-workflow.md) | Where things live in the UI and the normal day-to-day workflow                                                                             |
| [Getting better results](./frontend/app/docs/content/04-getting-better-results.md)         | Best practices for improving output quality and troubleshooting weak results                                                               |
| [Privacy, storage, usage`](./frontend/app/docs/content/05-privacy-storage-and-usage.md)    | What stays local, what can be sent to providers, and debug/privacy caveats                                                                 |

## Install

### MacOS

- Download the `.pkg` release package.
- Click to install the `.pkg`. It will walk you through the installation process.
- Local data (settings, conversations, logs) is stored at:
  - `~/Library/Containers/io.github.flexigpt.client/Data/Library/Application\ Support/flexigpt/`

### Windows

- Download the `.exe` release package.
- Click to install the `.exe`. It will walk you through the installation process.
- Note: Windows builds have undergone very limited testing.

### Linux

- Download the `.flatpak` release package.
- If Flatpak is not installed, enable it for your distribution
  - Ubuntu/Debian/etc (APT based systems):

    ```shell
    sudo apt update # update packages
    sudo apt install -y flatpak # install flatpak
    sudo apt install -y gnome-software-plugin-flatpak # optional, enables flathub packages in gnome sofware center
    flatpak remote-add --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo
    ```

  - Some additional helper commands can be found in this [script](./scripts/initialize_flatpak.sh)

- Install the package
  - `flatpak install --user FlexiGPT-xyz.flatpak`
  - `flatpak info io.github.flexigpt.client`

- Running the app
  - Using launcher GUI: You can launch the app from your distributions's launcher. E.g: In Ubuntu: Press the window key, type flexigpt and click on icon.
  - Using terminal: `flatpak run io.github.flexigpt.client`
  - Known issue with Nvidia drivers:
    - If you use Nvidia and its proprietary drivers, you _may_ see that the run command open a blank screen and close, the workaround for it is to run the app as: `flatpak run --env=WEBKIT_DISABLE_COMPOSITING_MODE=1 io.github.flexigpt.client`
    - Open bugs in upstream projects that cause this: [Webkit issue 180739](https://bugs.webkit.org/show_bug.cgi?id=180739), [Webkit issue 262607](https://bugs.webkit.org/show_bug.cgi?id=262607), [Debian issue 1082139](https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=1082139), [Wails issue 2977](https://github.com/wailsapp/wails/issues/2977)

- Your local data (settings, conversations, logs) will be at:
  - `~/.var/app/io.github.flexigpt.client/data/flexigpt`

## Built With

- Data storage: `JSON` and `SQLite` files in local filesystem.
- [Go](https://go.dev/) backend.
- [Wails](https://wails.io/) as a desktop application building platform.
- Official Go SDKs by [OpenAI](https://github.com/openai/openai-go) and [Anthropic](https://github.com/anthropics/anthropic-sdk-go).
- [Vite](https://vite.dev/) + [React Router v7](https://reactrouter.com/) frontend in [Typescript](https://www.typescriptlang.org/). [DaisyUI](https://daisyui.com/) with [TailwindCSS](https://tailwindcss.com/) for styling.
- Tooling: [Golangci-lint](https://golangci-lint.run/), [Knip](https://knip.dev/), [ESLint](https://eslint.org/), [Prettier](https://prettier.io/), [GitHub actions](https://github.com/features/actions)

## Contributing

- The dev setup docs are located at [devsetup.md](./docs/contributing/devsetup.md)

## License

Copyright (c) 2024 - Present - Pankaj Pipada

All source code in this repository, unless otherwise noted, is licensed under the Mozilla Public License, v. 2.0. See [`LICENSE`](./LICENSE) for details.
