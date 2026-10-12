# Development Guide

This guide covers local source development. Players should use
[Installation](INSTALL.md). Discuss changes through the
[issue-first contribution process](../CONTRIBUTING.md) before preparing a PR.
Architecture and system documents describe durable contracts independently of any
particular editor or assistant.

## Setup

The application source is public, with a development preview installable from
`main`. No version tag or packaged release has been published. For local
development, provision:

- Go: use Go 1.27's latest patch or newer; `go.mod` declares `go 1.27.0` and
  `toolchain go1.27.1`. Provision the selected toolchain before offline validation.
  Ebitengine is pinned to v2.10.4.
- Git and Make for the contributor workflows below.
- Python **3.9+** for standalone public Game Facts validation.
- Repository-pinned Go tools, installed by `make tools` into `bin/tools`.
- Optional Node/npm when changing or validating the Debug Workbench frontend;
  provision its dependencies separately before offline checks.

Player installation needs neither Node, Make, an AI tool, nor a C compiler;
see [installation acceptance](INSTALL.md#installation-acceptance) for the verified
macOS scope and deferred native Linux/Windows testing.
Race-instrumented developer tests have separate native compiler requirements.
The text editor adapter uses the supported, deprecated `exp/textinput.Field`
for IME composition and selection; replacing it requires preserving those
contracts and native input acceptance. Cross-builds do not prove native acceptance.

```sh
make tools
make validate
```

`make tools` is networked provisioning: it downloads Go modules and installs
pinned goimports and golangci-lint. Install Python and optional browser tools
separately. `make validate` is local/offline and fails rather than downloading a
missing prerequisite. See [Testing](TESTING.md) for the public asset-independent
lane, frontend checks, and explicitly provisioned original-data tests.

Use [original-data setup](INSTALL.md#locate-the-original-game-data) to discover an
existing GOG/Heroic installation, use the displayed home/UltimaV folder,
or edit the settings file externally and recheck. Fresh setup requires explicit
confirmation even for one result; valid configured locations are retained.
There are no in-game data or legacy-save path fields. A complete original
installation is external even though the application retains bundled assets and
test fixtures with their notices. Tests needing original files should receive an
explicit data path; public asset-independent validation must not silently rely
on a developer's local files.

## Lint Policy

The source import retains inherited lint debt rather than broadening this work
into unrelated legacy cleanup. All 94 configured linters, thresholds, and existing
exclusions remain enabled as defined in `.golangci.yml`. The new-code cutoff is
the fixed sanitized application-source checkpoint rather than a moving branch.
Migration-specific changed-line findings were checked before that checkpoint.
The fixed baseline is `f31ba69759594face92838f0ce33855f46aca4e3`; retain this
commit in checkout history. The config verifier rejects moving refs and abbreviated
hashes. Advancing the cutoff requires an explicit reviewed policy change.

`make lint` requires zero findings on new code relative to the recorded source
baseline. `make lint-all` disables revision filtering and output caps so inherited
debt remains inspectable. A passing new-code gate is not a zero-debt assertion;
changed-line filters also have declaration and transitive-change limitations.

When manually modifying a source file, resolve all configured findings in that
file using an unfiltered lint report, not only findings on the edited lines.
Leave manual cleanup in otherwise untouched files alone. Repository-wide autofix
is a separate, reviewed pass; files changed only by autofix are exempt from further
manual cleanup. Reject automatic rewrites that change domain identities, callback
ordering, or dynamically growing loop semantics. Formatting and tests remain
required after reviewing the automatic diff. This policy does not move the fixed
lint baseline or relax enabled checks.

Canonical lint checks enumerate the requested packages with `go list` and run
serial batches of four through the same pinned linter/configuration. Each package
receives the complete configured analysis once; an earlier failure does not omit later batches. This
bounds retained analyzer graphs without changing the configured rules or revision
filter. Duplicate-code detection additionally runs over the full inventory so
cross-batch duplication remains visible. `lint-all` retains uncapped, unfiltered
output in every batch and the whole-inventory duplicate pass.

## Starting The Debug Workbench

```sh
make dev ARGS="-debug-workbench"
```

Open `http://127.0.0.1:8585`. For another loopback port:

```sh
make dev ARGS="-debug-workbench -debug-workbench-port 8686"
```

The server binds only to `127.0.0.1`; browser requests execute through the game
update thread. Use the navigation-only Command Palette (`Ctrl+K` / `Command+K`)
to find the owning instrument. Selecting a surface may load a read-only query
but never executes a mutation. The live catalog supplies current commands and
availability reasons. See [Workbench architecture](systems/DEBUG_WORKBENCH.md)
for review, revision, scenario, and rollback contracts, and
[Workbench UI](ui/DEBUG_WORKBENCH_UI.md) for interaction ownership.

## Common Commands

| Command | Purpose |
|---|---|
| `go build -o bin/ultimav ./cmd/ultimav` | Build local production source without Make. |
| `go run ./cmd/save-convert --help` | Inspect the headless converter; follow the [save converter procedure](details/SAVE_CONVERTER.md) for explicit inputs and output modes. |
| `make dev` | Build and run the game for development. |
| `make test` | Run the Go test suite. |
| `make test-integration` | Run tests whose names match `Integration`, not the complete real-data suite. |
| `make test-provisioned` / `make test-race-provisioned` | Run every package with explicit original data and zero skips. |
| `make vm-settings-proof` | Run the opt-in keyboard/settings rollback proof with original data and a native graphics context; see [Testing](TESTING.md#vm-keyboardsettings-proof). |
| `make format` | Apply repository formatting. |
| `make lint` | Run the repository-pinned linter. |
| `make web-build` | Rebuild Debug Workbench browser assets after frontend changes. |
| `make validate` | Run standard local validation after final edits. |
| `make quality-gate` | Run the final pre-PR validation gate when required. |

Run focused tests while developing. Use deterministic seeds and callback mocks
for gameplay regressions. Review contracts, failure paths, cancellation, resource
commitment, and save behavior as well as happy paths. Humans review and merge PRs.

## Local release packaging

Release archives are built locally; no GitHub Actions runner or credit is needed.
Use Python 3.9+, Git, and the exact toolchain selected by `go.mod`. Provision that
toolchain and `go mod download` separately while network access is available.
The packaging command is offline and fails if a toolchain, module, or dependency
notice is missing. It uses shipped assets without a frontend rebuild.

From a **clean checkout** of the intended commit, after local validation:

```sh
python3 scripts/package_release.py --revision HEAD --version v0.1.0-preview.1 --output /existing/output-parent/ultimav-v0.1.0-preview.1
```

The version is an example, not an announced release. Supply an explicit version
and revision; the revision must resolve to current HEAD. An existing local tag of
that name must identify the same commit. No tag is required or created. Untracked
and modified files block packaging. The output parent must exist, the output
directory must be new and outside the checkout, and concurrent use of the same
output is rejected. An interrupted process may leave a sibling `.lock` directory;
remove it only after confirming its packaging process has ended.

The script exports the commit to a temporary source tree, builds sequentially with
`CGO_ENABLED=0`, two Go scheduler threads, `-p=1`, and a 256 MiB Go soft memory
limit. These settings reduce pressure; they are not a hard process-tree RSS cap.
Ambient Go flags, workspaces, and architecture tuning are disabled. It produces:

- `ultimav_<version>_darwin_arm64.tar.gz`
- `ultimav_<version>_windows_amd64.zip`
- `ultimav_<version>_linux_amd64.tar.gz`
- `ultimav_<version>_linux_arm64.tar.gz`
- `SHA256SUMS`

Only the game executable is included; the optional save converter remains a
separate source installation. Each archive includes an explicit-path launch guide,
installation guidance, version/commit/toolchain metadata and binary checksum,
project/media credits, font companion notices, the audio attribution manifest,
and top-level Go/dependency license notices. Review notice coverage when changing
dependencies, particularly licenses in nested directories. Source game data,
settings, saves, and unrelated checkout files are not copied into the payload.
Go build metadata is trimmed of local source paths; `release.json` records the
commit because the exported source has no Git directory. Archive timestamps are
not normalized, so byte-for-byte reproducibility is not claimed.

The final output directory appears only when all four builds and checksums finish.
Failures clean up staged output and do not replace an earlier release. Test the
packaging logic with `python3 -B -m unittest discover -s scripts -p test_package_release.py -v`.
Run `make validate` for implementation changes; packaging does not replace the
project's validation gates or platform acceptance.

Before distribution, verify checksums and archive membership, extract into a clean
directory outside the checkout, and invoke the extracted macOS executable by its
explicit path with `--help` and bounded startup. Record interactive original-data
setup, keyboard, graphics, audio, and save/load acceptance separately per platform.
Linux and Windows cross-builds alone are not native acceptance. Initial archives
are unsigned and require the [download guidance](INSTALL.md#downloaded-binaries).

Uploading is a separate, explicitly authorized operation. With a reviewed and
pushed tag, a maintainer can create a **draft prerelease** using `gh release create
<tag> <archive-paths> <checksum-path> --repo bradhannah/UltimaVRedux --verify-tag
--draft --prerelease --title <title> --notes-file <public-release-notes>`.
Enumerate only reviewed artifacts. Existing drafts can receive reviewed files via
`gh release upload <tag> <paths> --repo bradhannah/UltimaVRedux`; avoid overwriting
published artifacts. Tag creation/push, uploading, and publishing are separate
decisions. Release notes must identify the source and honest platform acceptance,
original-data requirements, save compatibility, and relevant known limitations.

## Finding The Right Guidance

| Work | Durable reference |
|---|---|
| Architecture or package boundaries | [Architecture Guidelines](ARCHITECTURE_GUIDELINES.md) |
| Player commands and input | [Input Handling](systems/INPUT_HANDLING.md) |
| Deterministic time or randomness | [Time System](systems/TIME_SYSTEM.md) |
| Event workflows and feedback | [Event Queue System](systems/EVENT_QUEUE_SYSTEM.md) |
| Dialogs and modal feedback | [Modal Dialog System](ui/MODAL_DIALOG_SYSTEM.md) |
| Reusable UI widgets | [UI Widget System](ui/UI_WIDGET_SYSTEM.md) |
| Debug Workbench | [Debug Workbench System](systems/DEBUG_WORKBENCH.md) |
| Original mechanics or formats | [Game Facts](gamefacts/README.md) and [rule conventions](gamefacts/CONVENTIONS.md) |
| Testing and review | [Testing](TESTING.md) and [repository invariants](../AGENTS.md) |
| Documentation maintenance | [Documentation Catalog](README.md) |

For documentation changes, preserve non-obvious ordering, ownership, cleanup,
failure and compatibility contracts. Use canonical rule IDs for original behavior;
do not replace detailed explanations with code-link stubs. Current Redux code and
tests establish implementation, ADRs explain decisions, and Game Facts distinguish
original rules from intentional differences and unresolved behavior.

See [Coding Conventions](CODING_CONVENTIONS.md), [Error Handling](ERROR_HANDLING.md),
and the [Documentation Catalog](README.md) for the complete retained policy and
reference set.
