# Development Guide

This guide covers local source development. Players should use
[Installation](INSTALL.md). Discuss changes through the
[issue-first contribution process](../CONTRIBUTING.md) before preparing a PR.
Architecture and system documents describe durable contracts independently of any
particular editor or assistant.

## Setup

Use the application-bearing checkout: source is staged on
`feat/graphics-test-trim`, while `main` remains documentation-only. No source
release or tag has been published. From that checkout, provision:

- Go: `go.mod` declares `go 1.25.0` and `toolchain go1.27.1`. Provision the selected
  toolchain before offline validation. Ebitengine is pinned to v2.10.4.
- Git and Make for the contributor workflows below.
- Python **3.9+** for standalone public Game Facts validation.
- Repository-pinned Go tools, installed by `make tools` into `bin/tools`.
- Optional Node/npm when changing or validating the Debug Workbench frontend;
  provision its dependencies separately before offline checks.

Production installation needs neither Node, Make, an AI tool, nor a C compiler.
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
existing installation, use the displayed Documents/UltimaV folder, enter a folder,
or edit the settings file and recheck. A complete original installation is external
even though the application retains bundled assets and test fixtures with their
notices. Tests needing original files should receive an explicit data path; public
asset-independent validation must not silently rely on a developer's local files.

## Lint Policy

The source import retains inherited lint debt rather than broadening this work
into unrelated legacy cleanup. All 94 configured linters, thresholds, and existing
exclusions remain enabled as defined in `.golangci.yml`. The new-code cutoff is
the sanitized application-source checkpoint, not the documentation-only `main`.
Migration-specific changed-line findings were checked before that checkpoint.
The fixed baseline is `f31ba69759594face92838f0ce33855f46aca4e3`; retain this
commit in checkout history. The config verifier rejects moving refs and abbreviated
hashes. Advancing the cutoff requires an explicit reviewed policy change.

`make lint` requires zero findings on new code relative to the recorded source
baseline. `make lint-all` disables revision filtering and output caps so inherited
debt remains inspectable. A passing new-code gate is not a zero-debt assertion;
changed-line filters also have declaration and transitive-change limitations.

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
