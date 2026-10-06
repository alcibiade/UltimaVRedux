# Installing and starting Ultima V Redux

> **Draft for the first source release.** This repository currently contains
> documentation only. No tagged, installable game version has been published here.
> The following is the installation contract to validate before release.

## What you need

- Your own original Ultima V game data files.
- A supported desktop platform with a working graphics and audio environment.
- The Go toolchain version documented with the release. The minimum and tested
  toolchains for this repository will be confirmed when the application is staged.
- Network access for Go to download the public module and its dependencies.

Initial validation targets are macOS arm64 (Apple Silicon), Windows amd64, and
Linux amd64, in that order. Intel macOS and other architectures need separate
acceptance before being advertised as supported.

The intended player installation requires neither a source checkout nor Make or
Node. Removing the application's native compiler requirement is part of release
preparation. Linux still needs the applicable desktop graphics/audio runtime;
exact distribution prerequisites will be documented from native validation.

## Install the latest released version

After the first release is available:

```sh
go install github.com/bradhannah/UltimaVRedux/cmd/ultimav@latest
```

The command downloads and builds the game and places the executable in Go's
executable directory. It does not install or download original Ultima V data.
To select a specific published release, replace `latest` with its actual version
tag from the [releases page](https://github.com/bradhannah/UltimaVRedux/releases).

## Start the game

If Go's executable directory is on your `PATH`, run:

```sh
ultimav
```

If the command is not found, inspect:

```sh
go env GOBIN
go env GOPATH
```

Go installs executables into `GOBIN` when set; otherwise it uses `bin` under the
first `GOPATH` entry. With Go's default settings, launch directly as follows.

macOS or Linux:

```sh
"$HOME/go/bin/ultimav"
```

Windows PowerShell:

```powershell
& "$HOME\go\bin\ultimav.exe"
```

Use your actual executable directory if you customized Go's environment. The
installed game must work from an unrelated directory; running from a source
checkout is not part of the player setup contract.

## Locate the original game data

### Use an existing installation

On first-run recovery, setup checks supported installation records and a bounded
catalogue of common GOG/Heroic and local locations. Confirm a verified result.
When multiple valid copies are found, choose the one you intend to use.

An already valid configured location should not require discovery on every start.
Discovery is not an exhaustive scan of your computer; a copy may exist elsewhere.

### Use the recommended personal folder

If discovery finds no usable copy, setup shows a recommended folder in your actual
Documents directory:

```text
Documents/
  UltimaV/
    DATA.OVL
    ...the other required original game files...
```

Copy the original game files into the displayed folder, then choose **Check
Again**. Supplying only `DATA.OVL` is not sufficient. Use the exact location shown
by the game, which may reflect redirected or localized Documents folders.

This is the simple fallback; a verified installation elsewhere does not need to
be moved. **Enter a Folder Path** remains available when you prefer manual entry.

### Edit the configuration in your text editor

Choose **Open Settings File**. Setup shows the exact configuration filename and
the data-path setting to edit; look for the existing `DataFilePath` setting, whose
serialized key may appear as `datafilepath`.

1. Replace the existing setting's value with the folder containing your original
   game files. Do not add a duplicate setting.
2. Use an absolute folder path, not the path to `DATA.OVL` itself.
3. Preserve other settings and save the file.
4. Return to the game and choose **Check Saved Settings**.

You can paste the path in your operating system's normal text editor. In-game
clipboard support is not required. If the editor cannot be launched, open the
displayed file yourself and use the same recheck action.

For YAML paths, single quotes avoid treating Windows backslashes as escape
sequences. Setup should present an appropriately quoted example. Report malformed
or unreadable configuration through the recovery UI rather than deleting settings
as a first step.

The existing setup contract distinguishes first-run recovery from changing a
working installation: first-run selection continues startup; changing the data
folder from the title menu takes effect on the next launch. Follow the displayed
restart instructions.

## Troubleshooting

| Problem | First step |
| --- | --- |
| `ultimav` is not found | Check `GOBIN`/`GOPATH` and use the executable's full path. |
| No original installation is detected | Use the displayed Documents/UltimaV folder or edit the configuration. |
| Files are missing or unusable | Confirm that the full original data set is present and readable; inspect the setup details. |
| Pasting in the game is unavailable | Use discovery, the recommended folder, manual typing, or the external editor. |
| A configured folder stopped working | Use setup recovery to select or configure a valid folder. |
| Build or startup fails | Report the release, OS/architecture, Go version, command, and relevant error text. |

Before attaching diagnostics, remove personal paths or other information you do
not intend to publish. Do not upload original game assets. See
[reporting guidance](../CONTRIBUTING.md).

## Update

Once releases exist, rerun the installation command to install the latest released
version. Read that release's notes for save compatibility and any additional
requirements before updating a playthrough.

## Release acceptance

The maintainer must verify tagged installation, launch outside a checkout,
asset-independent setup, original-data recovery, and native input on each
advertised OS/architecture. The application and required embedded/generated
assets must be present in the module; Go installation does not run frontend or
asset-generation commands. Remove the draft notice only after that acceptance is
complete for the published targets.
