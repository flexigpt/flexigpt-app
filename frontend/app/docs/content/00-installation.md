# Installation

Download the latest release from [GitHub Releases](https://github.com/flexigpt/flexigpt-app/releases), then install the package for your platform.

## macOS

- Download the `.pkg` release package.
- Open the installer and follow the setup steps.

## Windows

- Download the `.exe` release package from [GitHub Releases](https://github.com/flexigpt/flexigpt-app/releases).
- Run the installer.
- If Microsoft Defender SmartScreen shows a **Windows protected your PC** warning:
  - This can happen because current Windows builds are not signed with a Windows code-signing certificate. Windows may therefore show the publisher as unknown.
  - Select **More info**.
  - Select **Run anyway**.
    - If **Run anyway** is not available, your device or organization policy may block unsigned apps. In that case, use a device where you are allowed to install unsigned applications or contact your administrator.
  - Continue through the installer setup steps.

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
flatpak install --user FlexiGPT-vX.Y.Z.flatpak
flatpak info io.github.flexigpt.client
```

Run the app with:

```shell
flatpak run io.github.flexigpt.client
```

- Known first-launch issue on Linux with Nvidia drivers
  - If you use Nvidia proprietary drivers, the app may open a blank window and close. Try launching it with:

    ```shell
    flatpak run --env=WEBKIT_DISABLE_COMPOSITING_MODE=1 io.github.flexigpt.client
    ```

If that works, the issue is likely the known WebKit rendering problem on some Linux setups.

For platform-specific storage locations, what is stored locally, and where app data is kept, see **Privacy, Storage, and Troubleshooting**.
