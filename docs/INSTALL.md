# Installing and starting Ultima V Redux

> **Development preview — public cutover pending.** The repository is not public
> yet and anonymous installation has not been verified. Local source builds are
> available. No source release or version tag has been published; `@main` is the
> intended installation target for the public preview.

## What you need

- Your own original Ultima V game data files.
- A supported desktop platform with a working graphics and audio environment.
- Go 1.27 or newer, using the latest patch. The module declares `go 1.27.0` and selects
  `toolchain go1.27.1`; allow Go to obtain that toolchain or provision it yourself.
- Network access for Go to download the public module and its dependencies.

The four intended targets are macOS arm64 (Apple Silicon), Linux arm64, Linux
amd64, and Windows amd64. Existing automated results are from Apple Silicon macOS,
not full native/manual acceptance. Linux and Windows remain unverified natively;
intended targets are not a claim that every build or native installation has
passed. Intel macOS and other architectures need separate acceptance.

The production installation goal requires neither Make, Node, an AI tool, nor a C
compiler. The public module route will also avoid a source checkout. Acceptance
of that installation contract is still platform-specific.
Linux still needs the applicable desktop graphics/audio runtime; exact
distribution prerequisites will be documented from native validation. Cross-build
success alone does not establish native graphics, audio, or input acceptance.

## Build the local source

From an existing application checkout:

```sh
go build -o bin/ultimav ./cmd/ultimav
./bin/ultimav
```

For Windows, build with `-o bin/ultimav.exe` and launch `.\bin\ultimav.exe`.
Alternatively, `go run ./cmd/ultimav` builds and runs the local source. Dependency
and toolchain downloads need network access unless already cached.

Required embedded/generated assets are included in the staged application. A
production build does not require a frontend rebuild. Bundled assets and fixtures
are not a complete original game installation and do not replace the external
files described below. Contributors changing the frontend or running validation
should use the separate [development guide](DEVELOPMENT_GUIDE.md).

## Install the current development preview

After public cutover, install directly from `main`:

```sh
go install github.com/bradhannah/UltimaVRedux/cmd/ultimav@main
```

The command downloads and builds the game and places the executable in Go's
executable directory. It does not install or download original Ultima V data.
`main` is a moving development preview, not a versioned release. Version tags may
be offered later through the [releases page](https://github.com/bradhannah/UltimaVRedux/releases).

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

Fresh configuration leaves the data location unset. With an unset or unusable
data location, setup checks supported GOG/Heroic
installation records and known installation locations, plus the home-folder
route below. Confirm a verified result: even a single result needs your explicit
confirmation. When multiple valid copies are found, choose the intended one.

An already valid configured location is retained without discovery on every start,
including a historical `./OLD` value that still resolves to a valid installation.
Fresh discovery does not select the working directory or an implicit `./OLD`.
Discovery is not an exhaustive scan of your computer; a copy may exist elsewhere.

### Use the recommended personal folder

Setup shows a recommended folder directly under your resolved home directory:

```text
<home>/
  UltimaV/
    DATA.OVL
    ...the other required original game files...
```

Copy the original game files into the displayed folder, then choose **Check
Again**, then confirm the verified installation. Supplying only `DATA.OVL` is not
sufficient. Use the exact location shown by the game: `$HOME/UltimaV` on macOS or
Linux, or the displayed home-directory equivalent on Windows. This is not a
Documents-folder lookup.

This is the simple fallback; a verified installation elsewhere does not need to
be moved. Custom locations use the external configuration editor below. Setup has
no in-game folder-path field.

### Edit the configuration in your text editor

Choose **Open Settings File** to request your external editor on a best-effort
basis. Setup shows the exact configuration filename and
the data-path setting to edit; look for the existing `DataFilePath` setting, whose
serialized key may appear as `datafilepath`.

1. Replace the existing setting's value with the folder containing your original
   game files. Do not add a duplicate setting.
2. Use an absolute folder path, not the path to `DATA.OVL` itself.
3. Preserve other settings and save the file.
4. Return to the game and choose **Check Saved Settings**. It rereads the actual
   saved file and validates its selected installation.
5. During first-run setup, confirm the freshly checked location before continuing.
   From the title menu, follow the next-launch notice instead.

You can paste the path in your operating system's normal text editor. In-game
clipboard support is not required. If the editor cannot be launched, open the
displayed file yourself and use the same recheck action.

For YAML paths, single quotes avoid treating Windows backslashes as escape
sequences. Setup displays a quoted example. Malformed or unreadable settings are
reported and are not silently rewritten; correct them in the editor and recheck.

The existing setup contract distinguishes first-run recovery from changing a
working installation: first-run selection continues startup; changing the data
folder from the title menu takes effect on the next launch. Follow the displayed
restart instructions.

## Import an original save

**Import from Legacy** on the title menu reads the exact `SAVED.GAM`/`SAVED.OOL`
pair in the **active game-data directory**. It does not search other providers,
previous locations, or a pending data-folder change. If the pair is missing or
unusable, the recovery menu offers **Check Again** and **Back**. There is no
in-game legacy-path field.

After preparation, choose a native slot and its optional description, then
explicitly confirm any overwrite. The original pair is preserved. To select a
pair stored elsewhere or create a standalone native snapshot, follow the
[save converter procedure](details/SAVE_CONVERTER.md).

## Troubleshooting

| Problem | First step |
| --- | --- |
| `ultimav` is not found | Check `GOBIN`/`GOPATH` and use the executable's full path. |
| No original installation is detected | Use the displayed home/UltimaV folder or edit the configuration. |
| Files are missing or unusable | Confirm that the full original data set is present and readable; inspect the setup details. |
| A custom data path is needed | Edit it in the external settings file, then choose Check Saved Settings and confirm. |
| A configured folder stopped working | Use setup recovery to select or configure a valid folder. |
| Build or startup fails | Report the release, OS/architecture, Go version, command, and relevant error text. |

Before attaching diagnostics, remove personal paths or other information you do
not intend to publish. Do not upload original game assets. See
[reporting guidance](../CONTRIBUTING.md).

## Update

After public cutover, repeat the same command to update to the current preview:

```sh
go install github.com/bradhannah/UltimaVRedux/cmd/ultimav@main
```

Review current save-compatibility guidance before updating a playthrough. This
does not select a stable version tag; tagged releases are a future option.

## Release acceptance

The maintainer must verify anonymous `@main` installation, launch outside a checkout,
asset-independent setup, original-data recovery, and native input on each
advertised OS/architecture. The application and required embedded/generated
assets must be present in the module; Go installation does not run frontend or
asset-generation commands. Remove the draft notice only after that acceptance is
complete for the published targets.
