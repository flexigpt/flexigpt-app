# FlexiGPT

[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/license/mpl-2-0)
[![Go Report Card](https://goreportcard.com/badge/github.com/flexigpt/flexigpt-app)](https://goreportcard.com/report/github.com/flexigpt/flexigpt-app)
[![lint](https://github.com/flexigpt/flexigpt-app/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/flexigpt/flexigpt-app/actions/workflows/lint.yml)
[![test](https://github.com/flexigpt/flexigpt-app/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/flexigpt/flexigpt-app/actions/workflows/test.yml)

FlexiGPT is a _cross-platform desktop client_ that lets you chat with multiple Large Language Models (LLMs) from a single, streamlined interface.

> ⚠ Early Access Notice:
>
> The project is under active development; expect breaking changes and incomplete features between releases.
> This repository is intended as a standalone application, not a reusable library.
> Public Go package APIs are not stable and may change between releases.

- [Quick Start](#quick-start)
- [Key Features](#key-features)
  - [Multi-provider Connectivity with Model Presets](#multi-provider-connectivity-with-model-presets)
  - [Unified Chat Workspace](#unified-chat-workspace)
  - [Persistent Conversation History](#persistent-conversation-history)
  - [Attachments](#attachments)
  - [Prompt Templates](#prompt-templates)
  - [Themes (Light / Dark / Custom)](#themes-light--dark--custom)
- [Install](#install)
  - [MacOS](#macos)
  - [Windows](#windows)
  - [Linux](#linux)
- [Built With](#built-with)
- [Contributing](#contributing)
- [License](#license)

## Quick Start

- Install FlexiGPT [for your OS](#install).

- Launch FlexiGPT, then open _Settings_.

- In _Auth Keys_ -> Paste the API key for your provider. Example links to provider pages for getting API keys:
  - [OpenAI](https://platform.openai.com/api-keys), [Anthropic](https://platform.claude.com/settings/keys), [Google Gemini](https://aistudio.google.com/api-keys).
  - Atleast one provider needs to be configured with an API key before chatting.
  - FlexiGPT doesn’t bill you; all usage is metered by each provider on your own account.

- Start chatting!

## Key Features

### Multi-provider Connectivity with Model Presets

- First-class connectors for OpenAI, Anthropic, Google, DeepSeek, xAI, Hugging Face, OpenRouter, Local LLama.cpp.
- _Plus any local/remote OpenAI chat completions or OpenAI responses or Anthropic messages compatible endpoint_.
- API keys are stored securely in your OS keyring - never in plain text.

- _Model Presets_ bundle a model selection (e.g. `gpt-5.1`) together with recommended defaults such as temperature, token limits, system prompt and reasoning parameters. Built-in presets for all supported LLM providers, with one-click loading during chat. Tweak values per conversation without altering the saved preset.

- <details>
  <summary>Feature details</summary>
  - Key Concepts

    | Term                 | Description                                                                                                    |
    | -------------------- | -------------------------------------------------------------------------------------------------------------- |
    | Model Preset         | Named set of defaults for a single model (temperature, max tokens, reasoning, etc.).                           |
    | Provider Preset      | Collection of model presets that belong to one provider and defines that provider’s default preset.            |
    | Default Model Preset | The preset automatically applied when the provider is selected and the user has not chosen a different preset. |

  - Presets are organized per provider. A provider may expose many presets and may declare one of them as its default.
  - A preset can be enabled or disabled at any time; existing chat history is not modified.
  - Per-conversation overrides are possible: after selecting a preset the user may fine-tune individual parameters (temperature, reasoning, etc.) without changing the stored preset.
  - Supports models with multiple behaviors: non-reasoning, reasoning with levels (high/medium/low), hybrid reasoning (with reasoning tokens).

  - Selecting a Preset in Chat
    - In the chat input bar, open the dropdown and choose any preset listed.
    - The preset’s parameters appear next to the dropdown; use the slider or number fields to make conversation-specific adjustments if desired.
    - The adjusted parameters affect only the current conversation and are discarded when the session ends.

  - Managing Presets
    - Click the _Model Presets_ icon in the sidebar.
    - Select a provider or create a new one.
    - The app provides built-in presets. These will be listed.
    - Create, edit, delete, enable/disable presets as required.
    - Mark one preset as _Default_ for the provider; it loads automatically when a model from that provider is first chosen.

  - Notes
    - Deleting a preset does not alter existing conversations; those sessions keep a snapshot of the parameters that were active when the message was sent.
    - Global defaults are applied for any parameter left blank in the preset.

  </details>

### Unified Chat Workspace

- Switch models mid-conversation, chain results from one model into the next, and fine-tune generation parameters on the fly.
  - Previous messages and context is preserved as a full thread in the new models API calls too.
- Attach files, images, and PDFs, use prompt templates, load and resume previous conversations, export full conversation or a full message or code/mermaid diagram or its image inside a single message seamlessly in the same interface.

- Productivity tooling inside the chat
  - Streaming responses.
  - Code snippets with copy/export
  - Mermaid diagrams with copy/export/zoom
  - Full Math/LaTeX rendering
  - Readily available chat API requests and response details

### Persistent Conversation History

- Auto-saved sessions which are resumable at any time, with full-text local-only search and storage in local files.

- <details>
  <summary>Feature details</summary>
  - Every chat session is persisted as a _Conversation_ containing its title, timestamps and full message sequence.
  - Conversations are stored locally; you can reload a conversation and continue from last point at any time.
  - A full-text search bar provides instant retrieval across titles and message contents. The search is fully local.

  - Key Concepts

    | Term              | Description                                                                      |
    | ----------------- | -------------------------------------------------------------------------------- |
    | Conversation      | A saved chat session composed of messages and metadata.                          |
    | Conversation Item | Lightweight record (ID, title, creation time) shown in lists and search results. |
    | Message Roles     | `system`, `user`, `assistant`; each recorded with timestamp and details.         |

  - Using Conversation History
    - Click the _Search Bar_ to open the history dropdown.
    - Recent conversations are listed chronologically; type in the search bar to filter by keywords appearing in either titles or message text.
    - Select a row to reload the full conversation and continue chatting.

  - Automatic Saving
    - A new conversation record is started automatically when you open a fresh chat window.
    - The title is generated heuristically from the first user message but can be renamed at any time.
    - Messages are appended in real-time; timestamps are stored for precise ordering.

  </details>

### Attachments

- Attach Local or web-based files (text/code/images/PDFs) to a conversation.
- Attach a directory (auto crawl and attach of files within)

- <details>
  <summary>Feature details</summary>
  - Multiple local files or a single directory can be selected at a time to attach to the message.
    - Code and text files become in-context text.
    - Images are sent as binary blobs.
    - Local extraction and attachment as text is supported for PDFs by default. Users can toggle the mode to send PDFs as blobs if supported by the API.

  - Web pages/URLs can be attached to the conversation.
    - By default readable content will be extracted and attached as in-context text for any web page.
    - Image URLs and PDF URLs are sent as binary blobs, with user switch available to convert them as links of extract PDF as text and send.

  - Attachments you add to a conversation are available to all subsequent turns in that conversation.
  - You can edit and send any message at any point in time. Attachments can be modified in edit mode too.
  - On each send, the currently attached blob files (images, PDFs in file mode) are re-read for the messages in this turn.
  - Anything attached as text is stored with the conversation and not reread again.
  - Max 16MB attachments are supported as of now.

  </details>

### Prompt Templates

- Turn complex prompts into reusable `templates` with variables.
- If required, group into toggle-able bundles.

- <details>
  <summary>Feature details</summary>
  - Prompt Templates allow you to store complex prompts (with variables) and reuse them.
  - Templates can be organized into Bundles, which act as configurable packs that can be enabled or disabled.
  - Templates and Bundles can be disabled or removed at any time without affecting existing chat history.

  - Key Concepts

    | Term            | Description                                                                                    |
    | --------------- | ---------------------------------------------------------------------------------------------- |
    | Prompt Template | A reusable prompt that can contain variables.                                                  |
    | Prompt Bundle   | A collection of templates that can be toggled on or off as a unit and shared with other users. |

  - Invoking a Template
    - Invoke the template menu using mouse click or keyboard shortcut.
    - An auto-complete list appears, ranked by relevance.
    - Use `↑/↓` to navigate if necessary, then press **Enter**.
    - If two bundles provide the same slug, both options are listed.
    - The most relevant match is selected for processing.

  - Managing Prompt Templates
    - Click the _Prompts_ icon in the sidebar.
    - Select a current prompt bundle or create a new one.
    - The app provides built-in bundles and templates. These will be listed.
    - Create, edit, delete, enable/disable bundles/templates as required.

  </details>

### Themes (Light / Dark / Custom)

- **System/Light/Dark Themes** supported with auto-detect & manual toggle.
- An option to use any of the custom [**DaisyUI themes**](https://daisyui.com/docs/themes/?lang=en#list-of-themes).

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

All source code in this repository, unless otherwise noted, is licensed under the Mozilla Public License, v. 2.0.
See [LICENSE](./LICENSE) for details.
