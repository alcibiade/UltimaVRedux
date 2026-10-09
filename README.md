# Ultima V Redux

A modern Go recreation of **Ultima V: Warriors of Destiny**, with enhanced
presentation, quality-of-life improvements, and a developer Debug Workbench.
Playing requires files from your own copy of the original game.

> **Source-release preparation:** application source is now staged on
> `feat/graphics-test-trim`. `main` is still documentation-only, and no source
> release or version tag has been published. The staged checkout can be built
> locally; the versioned installation command below is a future release target.

## Installation

For an existing checkout of the staged application, use Go **1.25.0 or newer**;
`go.mod` selects **go1.27.1** as the preferred toolchain:

```sh
go build -o bin/ultimav ./cmd/ultimav
./bin/ultimav
```

On Windows, use `bin/ultimav.exe` for the output and launch path. These are local
source-build instructions, not a claim of native platform acceptance.

Once the first source release is published, the intended player installation is:

```sh
go install github.com/bradhannah/UltimaVRedux/cmd/ultimav@latest
ultimav
```

See the [installation guide](docs/INSTALL.md) for Go prerequisites, finding the
installed executable, original-game setup, configuration recovery, and updates.

The initial platform targets are Apple Silicon macOS, Windows amd64, and Linux
amd64. Native installation and startup must be validated before each target is
advertised as supported. Native Windows/Linux acceptance and public installation
availability are not yet claimed. The production installation requires Go and a
desktop graphics/audio environment, not Node, Make, an AI tool, or a C compiler.

## Original game files

First-run setup offers three practical routes:

1. Confirm a verified existing installation found through supported GOG/Heroic
   discovery or known local locations.
2. Copy your original game files into the **Documents/UltimaV** folder shown by
   setup, then choose **Check Again**.
3. Choose **Open Settings File**, edit the game-data location in your text editor,
   save it, and choose **Check Saved Settings**.

The setup screen shows your actual Documents and settings paths. Manual folder
entry remains available. These routes do not depend on pasting text into the game.
The game does not download original assets for you.

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
