# Ultima V Redux

A modern Go recreation of **Ultima V: Warriors of Destiny**, with enhanced
presentation, quality-of-life improvements, and a developer Debug Workbench.
Playing requires files from your own copy of the original game.

> **Development preview — public cutover pending:** the application is unfinished
> and has known limitations. The repository is not public yet, and anonymous
> installation has not been verified. No source release or version tag has been
> published. The `@main` command below is the intended public preview installation.

This is a free, non-commercial fan project. Project-authored source code remains
MIT-licensed; bundled media has separate terms described in [Asset credits](ASSET_CREDITS.md).

## Installation

Install the latest patch of **Go 1.27** or newer. The module minimum is
**1.27.0**, with **go1.27.1** selected as its preferred toolchain. After public
cutover, install the current development preview with:

```sh
go install github.com/bradhannah/UltimaVRedux/cmd/ultimav@main
ultimav
```

Repeat the installation command to update the preview. Version-tagged source
releases are a future option. For an existing application checkout:

```sh
go build -o bin/ultimav ./cmd/ultimav
./bin/ultimav
```

On Windows, use `bin/ultimav.exe` for the output and launch path. These are local
source-build instructions, not a claim of native platform acceptance.

See the [installation guide](docs/INSTALL.md) for Go prerequisites, finding the
installed executable, original-game setup, configuration recovery, and updates.

The four intended targets are macOS arm64 (Apple Silicon), Linux arm64, Linux
amd64, and Windows amd64. Existing automated results are from Apple Silicon macOS;
they do not establish full native/manual acceptance. Linux and Windows native
behavior remains unverified.
The production installation goal is Go, your original game files, and a desktop
graphics/audio runtime, without Node, Make, an AI tool, or a C compiler. Platform
build and installation acceptance remain separate from that goal.

## Original game files

First-run setup offers three practical routes:

1. Confirm a verified existing installation found through supported GOG/Heroic
   discovery or known local locations.
2. Copy your original game files into the **UltimaV** folder directly under your
   home directory, as shown by setup, then choose **Check Again**.
3. Choose **Open Settings File**, edit the game-data location in your text editor,
   save it, and choose **Check Saved Settings**.

Fresh configuration leaves the data location unset; setup asks for confirmation
even when it finds just one valid installation.
An already valid configured path is retained. The screen shows the resolved home
folder and settings filename; custom locations are entered in the external
settings editor. There are no in-game path fields. The game does not download
original assets for you.

To import a playthrough, the title menu reads `SAVED.GAM` and `SAVED.OOL` only
from the active game-data directory. For a save pair elsewhere, use the
[command-line save converter](docs/details/SAVE_CONVERTER.md).

## Reports and suggestions

The [Game Facts](docs/gamefacts/README.md) cover 39 topics with stable behavioral
IDs, algorithms, compatibility formats, and explicit Redux comparisons. See the
[documentation catalog](docs/README.md) for system contracts and the
[development guide](docs/DEVELOPMENT_GUIDE.md) for contributor setup.

Please [open an issue](https://github.com/bradhannah/UltimaVRedux/issues/new/choose)
for bugs, installation difficulties, behavioral corrections, or suggestions.
Include the game version, operating system, and steps to reproduce the problem.

The maintainer currently owns implementation. Please read
[CONTRIBUTING.md](CONTRIBUTING.md) and open an issue before preparing a code PR.

## License

Project-authored source is provided under the [MIT License](LICENSE). Bundled
fonts, media, and fixtures retain their accompanying notices and applicable
rights; their inclusion does not make all media MIT-licensed. The complete
original Ultima V installation needed to play is externally supplied by the
player. Original game material retains its existing rights.

See [Asset credits and media terms](ASSET_CREDITS.md) for the music sources,
font notices and individual sound-effect attribution, including non-commercial
media licences and remaining provenance limitations.
