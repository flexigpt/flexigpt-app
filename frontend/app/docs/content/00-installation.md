# Installation

Download the latest release from [GitHub Releases](https://github.com/flexigpt/flexigpt-app/releases), then install the package for your platform.

## macOS

- Download the `.pkg` release package.
- Open the installer and follow the setup steps.

Local app data is stored under:

- `~/Library/Containers/io.github.flexigpt.client/Data/Library/Application Support/flexigpt/`

## Windows

- Download the `.exe` release package.
- Run the installer and follow the setup steps.

Notes:

- Windows builds have seen more limited testing so far.

## Linux

- Download the `.flatpak` release package.
- If Flatpak is not already installed on your system, enable it first.

For many Debian or Ubuntu based systems, that looks like:

```shell
sudo apt update
sudo apt install -y flatpak
sudo apt install -y gnome-software-plugin-flatpak
flatpak remote-add --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo
```

You can then install and inspect the package with:

```shell
flatpak install --user FlexiGPT-xyz.flatpak
flatpak info io.github.flexigpt.client
```

Run the app with:

```shell
flatpak run io.github.flexigpt.client
```

Local app data is stored under: `~/.var/app/io.github.flexigpt.client/data/flexigpt`

- Known first-launch issue on Linux with Nvidia drivers
  - If you use Nvidia proprietary drivers, the app may open a blank window and close. Try launching it with:

    ```shell
    flatpak run --env=WEBKIT_DISABLE_COMPOSITING_MODE=1 io.github.flexigpt.client
    ```

If that works, the issue is likely the known WebKit rendering problem on some Linux setups.
