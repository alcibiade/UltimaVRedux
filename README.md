# Ultima V Redux

A modern Go recreation of **Ultima V: Warriors of Destiny**, with enhanced
presentation, quality-of-life improvements, and a developer Debug Workbench.
Playing requires files from your own copy of the original game.

> **Publication preparation:** this checkout currently contains documentation and
> issue forms. The application and its first installable release have not yet been
> staged here. The command below is the intended release installation path, not a
> currently available release.

## Installation

Once the first source release is available, the intended installation is:

```sh
go install github.com/bradhannah/UltimaVRedux/cmd/ultimav@latest
ultimav
```

See the [installation guide](docs/INSTALL.md) for Go prerequisites, finding the
installed executable, original-game setup, configuration recovery, and updates.

The initial platform targets are Apple Silicon macOS, Windows amd64, and Linux
amd64. Native installation and startup must be validated before each target is
advertised as supported. No platform acceptance is claimed by this documentation
bootstrap.

## Original game files

The intended first-run setup offers three practical routes:

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

The [Game Facts pilot](docs/gamefacts/README.md) describes Karma using specific
behavioral references, algorithms, and explicit Redux comparisons.

Please [open an issue](https://github.com/bradhannah/UltimaVRedux/issues/new/choose)
for bugs, installation difficulties, behavioral corrections, or suggestions.
Include the game version, operating system, and steps to reproduce the problem.

The maintainer currently owns implementation. Please read
[CONTRIBUTING.md](CONTRIBUTING.md) and open an issue before preparing a code PR.

## License

Project-authored source is provided under the [MIT License](LICENSE). Original
Ultima V game data is not supplied by this project and retains its existing rights.
