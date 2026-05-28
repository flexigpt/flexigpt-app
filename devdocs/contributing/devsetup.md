# Dev Setup and Commands

## Setup

- Checkout code.
- Init and checkout all submodules (currently only flatpak shared modules): `git submodule update --init --recursive`
- Install base prerequisites `node`, `go`, `golangci-lint`, `task (taskfile.dev)`. Pinned versions of these can be found in file at repo root named: `.tool-versions`. Recommended: use `asdf` to manage the base prerequisites. If you have asdf installed you can do `asdf install` to install all required versions of this project.

- Install pnpm packages

  - Enable corepack and intall pnpm: `corepack enable; corepack install`
    - If using asdf: `asdf reshim nodejs`
  - Download all pnpm modules: `pnpm i`

- Install go tools prerequisites:

  - Run the script as `task installtools`
  - Wails issues:
    - Linux issue: If you are using latest Linux version (example: Ubuntu 24.04) and it is not supporting `libwebkit2gtk-4.0-dev`, then you might encounter an issue in wails doctor: `libwebkit not found`. To resolve this issue you can install `libwebkit2gtk-4.1-dev`. We do use the tag `-tags webkit2_41` during build.
    - If you have any issue in wails install, refer documentation page at [Wails Install Guide](https://wails.io/docs/gettingstarted/installation).

- Run a build: `task build-withbindings`

## Runner commands

- All the available commands are via taskfile at `taskfile.yml`. A select few scripts are mentioned below.

- Build wails app in dev mode: `task build-withbindings`

  - After this you will see the binary path at the end of build.
  - You can execute it directly. Dev console is available in this mode.
  - Debug console in the app is available via `Ctrl + Shift + F12`

- Build and run the backend: `task run-gobackend`

  - Backend OpenAPI 3.1 docs should be visible at: `http://localhost:8080/docs`

- Build wails app and run it as a dev server: `task run-watch`

  - Ideally this is to be used mainly for UI development via the local server running inside the browser.
  - Wails has some issues reloading both backend and frontend file changes together in some cases.

- Packaging commands

  - Ideally this should be done in github actions as secrets etc are setup there properly.
  - Create a flatpak on a local machine with some local defaults: `task pack-flatpak`

## VSCode setup

Note: Similar can apply to other IDEs.

- Recommended plugins are at: `./.vscode/extensions.json`
  - You can check and install them at the notification time, or later by checking them via VSCode command pallette
- Settings config for VSCode is at: `./.vscode/settings.json`.
  - These get applied as workspace settings if you open this project in VSCode.
