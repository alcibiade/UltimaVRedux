# Testing Strategy

This document owns test strategy, fixture selection, and verification procedures.
The Makefile owns executable commands; [ADR-004](ADR/ADR-004-integration-testing-real-data.md)
explains the rationale for real-data integration coverage.

## Principles

- Use standard Go `*_test.go` tests close to the package under test.
- Add a regression test for every bug fix and explain the behavior it protects.
- Keep unit tests independent of copyrighted Ultima V assets.
- Use real game data when parsing, map layout, save state, or cross-system behavior is part of the assertion.
- Drive gameplay randomness through `GameState` and use fixed seeds.
- Drive gameplay timing through game-state clocks and elapsed time, not wall time.
- Validate UI-facing behavior through `SystemCallbacks` mocks rather than presentation packages.
- Test the smallest scope that proves the behavior; do not use full game setup for pure logic.

## Test Scopes

### Unit Tests

Unit tests cover deterministic logic with local fixtures or small value objects. They should be fast, asset-independent, and suitable for `go test -short`.

Use table tests for related inputs and subtests for cases with distinct setup. Keep package-local helpers in `_test.go` files unless production packages need them.

### Integration Tests

Integration tests cover behavior that depends on real Ultima V files or multiple initialized systems. The primary game-state helper is `NewIntegrationTestBuilder` in `internal/game_state/test_builder_helpers.go`.

The builder can configure location, floor, player position, callbacks, and a deterministic random seed. It loads real references and `SAVED.GAM`, then initializes the selected map. Only genuinely absent optional data skips. Explicitly selected missing files, malformed data/configuration, and loader failures fail with their original diagnostic. `FailIfNoGameData` or `ULTIMA5_REQUIRE_DATA=1` also makes optional absence fatal.

Because `make test-integration` filters test names with `-run Integration`, an integration test intended for that target must include `Integration` in its Go test function name. A filename ending in `_integration_test.go` is useful organization but does not affect Go test selection.

### Adapter and UI Tests

Tests under `cmd/ultimav` may require Ebitengine display initialization. Linux CI runs these tests through Xvfb. Keep core gameplay tests out of the adapter package when rendering or input is not part of the behavior.

Ebitengine 2.10 initializes native window services lazily. `internal/display`
initializes the process display manager in `TestMain`, before test goroutines
can query monitors, as required by Cocoa. Main-screen rendering tests run inside
the native loop through `internal/ui/mainscreen/main_test.go`; application
rendering tests use `cmd/ultimav/graphics_test_main_test.go`. These harnesses,
the widget harness, and the sprite harness launch `m.Run` from the first Update
in a goroutine while the native loop continues to finish frames. Running the
entire suite synchronously inside one Update retains frame-local native upload
buffers. The channel transfers the exit code after completion; `RunGame` stays
on the main thread and initializes native services before tests start. Ebitengine
2.10's `Monitor` API requires the main thread before `RunGame` and is concurrent-safe
afterward. A missing display/backend fails the required render gate; it is never
a reason to skip the package.

Advancing frames also advances live input. Native fixtures must supply detached
held-key and text sources rather than read the live keyboard from test goroutines.
Existing application fixtures using `input.NewKeyboard` need separate review;
the harness change alone does not establish input-fixture or native acceptance.

### VM Keyboard/Settings Proof

```bash
ULTIMA5_DATA_PATH="/path/to/ultima5" make vm-settings-proof
```

This opt-in Ebitengine v2.10.4 `exp/vmhost` proof builds a host and a tagged
application test binary into `bin/`. The guest uses production `Game`, title
scene, keyboard handling, configuration dialog, and rendering. Its dedicated
`settingsproof,ebitenginevmguest` test entry point adds read-only observations;
it does not replace the normal native test suite.

The host sends Enter to skip the title presentation, navigates to Configure,
tabs from Display to Gameplay, toggles Repeated Monster Division, cancels,
reopens, and returns to Gameplay. Each press and release receives a completed
guest tick. Assertions check the scene, menu choice, dialog identity, staged
and live values, and unchanged configuration-file bytes. Real host-rendered
pixels from the selected row must change with the toggle and exactly return
to the original pixels after reopening; animated background pixels are excluded.
No golden images or original assets are written to the repository.

The host fixes its render surface at 1280×960, disables guest sound, isolates
HOME/configuration/save locations and working directory, and removes the
temporary home after reaping the guest. Connection/idle waits are bounded to
10 seconds and the runtime to 30 seconds; builds are outside that runtime
deadline. Use the resource protection described below for the complete command.
Success requires both host completion and a successful guest assertion exit.

The guest needs no display, but the host still requires native graphics; use
`xvfb-run -a make vm-settings-proof` on headless Linux. This verifies injected
engine keyboard handling and rendered settings rollback. Physical keyboard,
IME/clipboard, platform-window behavior and gameplay replay require their own
acceptance. In particular, the application's existing wall-time input/debounce
and scene timing mean VM tick counts alone do not prove gameplay determinism.

### Regression Tests

A regression test should state the user-visible failure or violated invariant, the root cause when it is not obvious, and why the assertion prevents recurrence. An issue link is useful when one exists. Avoid dates and historical counts in names unless the date is part of the domain behavior.

## Real Game Data

The complete original Ultima V installation is externally provisioned. Retained
bundled assets and fixtures carry their existing notices and do not replace it.
For game-state integration tests, set `ULTIMA5_DATA_PATH` to a directory containing
the required files, including `SAVED.GAM`, `LOOK2.DAT`, `CASTLE.TLK`, and `MISCMAPS.DAT`:

```bash
export ULTIMA5_DATA_PATH="/path/to/ultima5"
```

`test/originaldata.ResolveDirectory` selects `ULTIMA5_DATA_PATH`, then
`ULTIMA5_DATA_DIR`, then repository `OLD`. It never reads HOME or falls back
after a broken explicit selection. Required files depend on the test; selected
data must be complete for those assertions.

To reproduce a legacy import with a save pair separate from the runtime data,
set `ULTIMA5_IMPORT_SOURCE` for the production import integration test. The
source pair is read-only and the selected native slot is written to temporary
storage; the test verifies Slot 7 creation, description/identity persistence,
canonical capture, and gameplay activation. Without this optional source
override, the test uses `SAVED.GAM` from the selected runtime data directory.

```bash
ULTIMA5_DATA_PATH="/path/to/original-data" \
ULTIMA5_IMPORT_SOURCE="/path/to/save/SAVED.GAM" \
go test ./cmd/ultimav -run '^TestIntegrationLegacyImportCreatesSelectedSlotAndStartsGame$' -count=1
```

Use default skipping for public asset-independent runs. Private original-data
validation is a mandatory local gate for fidelity changes: set an explicit data
directory and run `make test-provisioned` and `make test-race-provisioned`.
`ULTIMA5_REQUIRE_DATA=1 make validate` makes its logic test scope strict;
use `ULTIMA5_REQUIRE_DATA=1 make quality-gate` for both required scopes, or
`make test-provisioned` for the aggregate zero-skip test gate.
`ULTIMA5_REQUIRE_DATA=1 make test` and
`ULTIMA5_REQUIRE_DATA=1 make test-race` select the same strict test policy.
Strict gates reject every test/subtest skip, including non-data
skips; use a supported, fully provisioned local platform. No hosted private
asset runner is configured. Do not commit the externally supplied original
installation or publish it as a test artifact; retain notices on bundled fixtures.

Configuration tests use instance-owned sources: `NewUltimaVConfigurationWithPath`
uses in-memory preferences and explicit game data without reading HOME or
enabling persistence. `LoadUltimaVConfiguration(configFilePath, dataFilePath)`
reads only the selected YAML file without rewriting it. Tests that exercise
startup's default-file creation isolate HOME with a temporary directory.

## Deterministic Gameplay Tests

Configure the seed before invoking behavior that can consume randomness:

```go
gs, callbacks := NewIntegrationTestBuilder(t).
	WithLocation(references.Britain).
	WithSystemCallbacks().
	WithRandomSeed(12345).
	Build()
```

For lower-level tests, call `GameState.SetRandomSeed` before the first random operation. Advance `GameState.ElapsedMs`, `DateTime`, or the relevant deterministic system explicitly when testing time-dependent behavior.

Call `GameState.FinishTurn` only when the assertion includes end-of-turn processing such as NPC movement, hazards, enemy maintenance, or other turn effects. Do not add it to callback-only or isolated unit tests: unrelated turn processing can obscure the behavior being tested. In multi-command workflows, finish each turn at the same boundary used by gameplay.

## System Callback Tests

`MockSystemCallbacks` in `internal/game_state/mock_callbacks_helpers.go` installs the mandatory callback groups and records messages, audio, visuals, flow, dialogs, and related interactions. Prefer its assertion methods, such as `AssertMessageContains`, `AssertSoundEffectPlayed`, and `AssertTimeAdvanced`, when they express the contract clearly. Call `Reset` between independent actions in one test.

Assert both state mutation and externally visible feedback when both are part of the command contract. Do not import Ebitengine into core tests just to observe presentation behavior.

## Writing Tests

Keep setup local until several tests need the same valid fixture; extracted
helpers call `t.Helper()`. Diagnostic logging belongs behind `t.Log`/`t.Logf`;
retain a diagnostic test only when assertions enforce a useful verdict. Calling
the code without checking its outcome is not regression coverage.

Assert authoritative before/after state, rejected and cancelled paths, resource
effects, and required callbacks. Persisted changes need round-trip coverage and,
when the supported contract includes it, migration coverage. Prefer detached
structural snapshots (tile IDs, frames, positions) over brittle pixel snapshots;
use pixel assertions when the rendering operation itself is the contract.

### Fixture Entry Points

| Scope | Start here | Important boundary |
|---|---|---|
| Real-data gameplay | `internal/game_state/test_builder_helpers.go`: `NewIntegrationTestBuilder`, `WithLocation`, `WithFloor`, `WithPlayerAt`, `WithRandomSeed`, `WithSystemCallbacks` | Configure before the action; original-data failures are not passing early returns. |
| Isolated callback contract | `internal/game_state/mock_callbacks_helpers.go`: `NewMockSystemCallbacks`, `ToSystemCallbacks`, assertion methods, `Reset` | Use the builder's callbacks when it already owns setup. |
| Semantic conversation | `internal/conversation/semantic_engine_test.go`: `semanticTestConversation`, `semanticTestCallbacks`, `TestSemanticConversationEngineProtocolFlow` | All gameplay conversations and inspection use the semantic catalog. |
| Authentic semantic import | `internal/conversation/extractor/` and `internal/conversation/catalog/` tests | Test extraction and the patched runtime catalog when original content matters. |
| Command dispatch | `cmd/ultimav/gamescene_command_context_routing_test.go` and map-specific input tests | Prove primary routing and completion ownership separately from terminal authoritative effects. |

An integration test normally follows this shape, using the action and assertion
appropriate to its fixture:

```go
gs, callbacks := NewIntegrationTestBuilder(t).
    WithLocation(references.Britain).
    WithPlayerAt(15, 15).
    WithSystemCallbacks().
    WithRandomSeed(12345).
    Build()

result := gs.ActionLookSmallMap(references.Up)
assert.True(t, result)
assert.NotEmpty(t, callbacks.Messages)
```

### Semantic And Real-Data Conversation Fixtures

Use `semanticTestConversation` and `semanticTestCallbacks` for deterministic
protocol/action coverage. For original content, import with `extractor.Import`
or load the production patched catalog with `catalog.LoadOriginal`, resolve the
one-based source/dialog identity, and initialize the semantic engine with a
distinct actor slot. Do not infer actor-state identity from a dialog number.

`semantic_engine_original_test.go` covers all 135 imported conversations with a
bounded smoke path; `semantic_engine_regressions_original_test.go` covers complex
name, donation, wait, and recruitment paths. Only absent optional installations
skip; selected-data or decoding failures must fail with the original cause.
Use provisioned mode to require this evidence rather than counting optional
skips as coverage. See the [conversation system](systems/LINEAR_CONVERSATION_SYSTEM.md).

The developer NPC lookup script's semantic text projection and identity joins
have asset-independent tests: `python3 -B scripts/test_npc_lookup.py`.

### Automated Versus Manual Evidence

Injected primary-handler tests prove routing, retained input, and time/action
ownership. Focused core/workflow tests prove terminal effects. Neither proves
physical keyboard behavior, readable presentation, or audible cue ordering.
Record those observations in the relevant issue after a real game-window run;
do not keep a second completion ledger in a system guide.

## Commands

### Resource-Sensitive Runs

Do not launch concurrent validation commands or overlapping agent test jobs on a
memory-constrained workstation. Start with one relevant package, an explicit
timeout, and serial execution. For example:

```bash
GOMAXPROCS=1 GOFLAGS=-p=1 GOMEMLIMIT=512MiB \
go test ./internal/map_state -run '^TestDaylight' -count=1 -parallel=1 -timeout=60s
```

`GOMEMLIMIT` is a soft Go-runtime target, not an operating-system memory cap. It
does not bound all native/GPU allocations or the combined footprint of child
processes. After an unexplained memory incident, use a separate process-tree
watchdog with native-memory accounting and termination on budget overrun or
launcher cancellation before attempting larger runs. Stop if that protection is
unavailable or triggered; do not retry unbounded or report an interrupted gate
as passing. The canonical gate and its required checks remain unchanged.

Keep failure diagnostics bounded too. Avoid handing live `GameState` graphs or
back-reference-bearing values such as `Lighting` to an automatic structural
diff formatter. Compare scalar values or binding identity directly; for large
detached snapshots, retain the complete equality check but report a bounded
failure message rather than formatting the entire snapshot. Use fail-fast
assertions for invariant failures before continuing expensive loops.

### Canonical Commands

The `Makefile` is the command authority:

| Command | Purpose |
| --- | --- |
| `make tools` | Networked provisioning: download modules and install pinned Go tools into `bin/tools`. |
| `make test` | Run uncached logic JSON tests without a display, with the complete package inventory and other required scope assignments. |
| `make test-render` | Run required native/Ebitengine-linked tests; use Xvfb on headless Linux and a native backend on macOS/Windows. |
| `make test-all` | Run the aggregate logic and render gate sequentially. |
| `make test-short` | Run `go test -short ./... -timeout 90s`. Only tests that honor `testing.Short()` are skipped. |
| `make test-integration` | Run tests whose names match `Integration` across all packages. |
| `make vm-settings-proof` | Run the opt-in native-host/VM-guest keyboard and rendered settings rollback proof. |
| `make test-race` | Run all Go tests with the race detector. |
| `make test-provisioned` / `make test-race-provisioned` | Require explicit original data and zero test/subtest skips, without/with the race detector. |
| `make lint-config-check` | Verify pinned v2 schema and detect both intentional errcheck fixture violations. |
| `make lint-all` | Run the pinned configured linters across all packages without revision filtering, per-linter caps, same-message caps, or line deduplication. Existing path exclusions and production-only test handling still apply. |
| `make core-dependencies-check` | Check transitive production and internal/external-test dependency closures for all six core roots. |
| `make web-build` | Install locked frontend dependencies, run frontend tests, and build Debug Workbench assets. |
| `make web-test-browser` | Build current source UI, then run four real Chromium smoke tests with mock transport; headless shell must already be installed. |
| `make check` | Run formatting checks, lint, vet, and logic Go tests. |
| `make pre-commit` | Run `scripts/pre-commit-checks.sh`. |
| `make policy-check` | Run the script's architecture/input/logging policy checks without repeating Go formatting, vet, build, or tests. |
| `make docs-check` | Validate retained Markdown links and anchors, catalog coverage and ownership, and documentation authority boundaries. |
| `make gamefacts-check` | Validate the public Game Facts catalog, canonical rule IDs, relationships, and public references with Python 3.9+. |
| `make tooling-test` | Run pure Python developer-tool regressions. |
| `make validate` | Run `check`, `policy-check`, `build-check`, documentation, public Game Facts, and developer-tool validation locally with `GOPROXY=off`, `GOSUMDB=off`, and `GOTOOLCHAIN=local`. Missing prerequisites fail, not download. |
| `make quality-gate` | Run validation, render tests, frontend build/browser tests, and bounded debug race tests once each. |

Useful focused Go commands:

```bash
go test ./internal/game_state -run '^TestSpecificBehavior$' -count=1
go test ./internal/game_state -run 'Integration' -v
go test ./internal/debugging/... -race
go test ./internal/astar -bench . -benchmem
```

On a headless Linux host, wrap Ebitengine-dependent tests with Xvfb:

```bash
xvfb-run -a go test ./cmd/ultimav
```

Provision separately before offline validation: `make tools` installs
`goimports@v0.37.0` and `golangci-lint@v2.12.2`. Python 3.9+ is required for public
Game Facts checks. Node/npm is needed for frontend builds and frontend-specific
validation; it is not a player prerequisite. Provision the module's selected Go
toolchain (`go 1.25.0`, `toolchain go1.27.1`) before offline validation.
Race-instrumented tests require a native C toolchain even though production
installation does not. Browser provisioning is separate from testing
(`npx playwright install --with-deps --only-shell chromium` from the frontend directory after
dependency installation). `make validate` does not install frontend dependencies
or browsers.

Frontend DOM tests run one worker without parallel test files; Chromium tests
also use one worker. This bounds concurrent fixture/browser lifetimes in the
local quality gate without excluding tests. These settings do not limit runtime
Workbench behavior or replace host memory supervision.

Frontend type checking runs four serial `vue-tsc` processes: application sources,
root-component tests, other DOM tests, and browser tests/configuration. The
canonical `tsconfig.json` remains the complete source inventory and compiler-option
authority. `typecheck.mjs` includes Vue roots in that inventory, rejects missing,
duplicate, unexpected or empty partitions, and requires identical compiler options
in every partition. Each process checks its full imported source/declaration
closure; `skipLibCheck` is not enabled. The 512 MiB old-space and 8 MiB semi-space
settings encourage bounded V8 collection, but are not process RSS caps. `npm test`
also runs the partition-accounting regressions before the existing DOM suite.

The browser command first builds the current source into ignored `browser/.build`,
then starts Playwright. Its global setup serves those fresh assets on loopback
inside the runner process; it never reuses an existing server or the checked-in
embedded bundle. Teardown closes the server and removes the temporary build.
This avoids retaining Vite's compiler/transform graph beside Chromium. Tests use
the same application production build as the embedded Workbench, with synthetic
transport; Vite development-server/HMR behavior is not covered by this suite.

Playwright's runner/workers use 64 MiB old-space and 2 MiB semi-space settings.
The secondary Debug Workbench has a deliberately small browser acceptance scope:
four smoke tests in `browser/workbench.spec.ts` cover desktop startup/status and
navigation at 1280px, reachable inventory controls at 800px, explicit mutation
confirmation/cancellation with visible server rejection, and keyboard palette
navigation/Escape/focus restoration. They use bundled real Chromium headless shell
with Playwright's default browser launch settings and screenshots only on failure.
Full Chrome, custom Chromium feature overrides, extensions, persistent profiles,
actual desktop zoom, dock matrices and exhaustive scenario browser workflows are
not required. This is an intentional coverage reduction, not equivalent acceptance
of those removed cases. Existing DOM/Go tests retain detailed behavior checks;
two focused App DOM regressions protect stale scenario Start and delayed rollback
review revocation. The synthetic transport never performs live game mutations or
certifies the interface's live usability. These execution settings still require
aggregate process-tree supervision on constrained hosts.

`lint-config-check` verifies the actual configuration, then runs a diagnostic
copy with only the immutable `issues.new-from-rev` cutoff removed. It requires two errcheck
findings (unchecked type assertion and blank-assigned error) and clean checked
controls. Normal lint retains that revision filter; it is not an unfiltered
whole-codebase count. `core-dependencies-check` uses `go list -deps -test`
including import remapping for GameState, party/map state, AI, references, and
config; missing/incomplete metadata fails closed.

`test-gate -suite=logic|render|all` accepts only those closed scopes (default: all).
It inventories every `./...` package, uses `partitionPackages` to classify the
production and generated test-binary dependency closures, then compares the
selected package inventory with uncached `go test -json -count=1` events. It prevents
inherited selection flags from hiding assertions, and reports each skipped
test/subtest and reason. Raw JSON goes to stdout and the inventory to stderr;
the gate creates no test-output artifact. A process success without the expected
package/assertion inventory is not sufficient. Unselected packages are reported
as assigned to the other required scope, not missing or passed. Selected but
unexecuted packages remain missing and fail. Summary totals cover only selected
packages, and optional original-data skips remain observations rather than
fidelity acceptance. Inherited GOFLAGS build settings remain intact; explicit
`-run`, `-skip`, `-list`, `-bench`, `-fuzz=`, count, short, race, and concurrency
flags prevent inherited selection or fuzz campaigns from changing the gate.

Native pixel tests must use bounded offscreen targets, explicit owned-image
cleanup, and a positive visual oracle. Keep broad state/layout/resolution matrices
in non-raster assertions, with only representative native cases for clipping,
composition, shaders, and fractional sampling. In application tests,
`newPresentationRaster` creates standalone offscreen storage: a temporary large
canvas must not share an atlas that persistent tiny sprites could keep alive.
Do not enlarge a fixture merely to test pointer identity, borrowed ownership, or
source-file dimensions; those assertions have smaller or CPU-only counterparts.

### Resource-Bounded Runs

Conversation test callbacks simulate elapsed pause time instead of sleeping.
Their timing assertions inspect the simulated clock, while production adapter
tests remain responsible for real pause scheduling. `make validate` executes one
uncached logic test pass; it does not repeat the suite through standalone
pre-commit. The standalone `make pre-commit` command retains its full checks.

Do not run broad suites concurrently. Prefer a focused package/test selection
with bounded build and test concurrency, especially for asset-heavy adapter tests:

```bash
GOMAXPROCS=2 GOMEMLIMIT=256MiB go test -p=1 -parallel=1 ./internal/game_state -run '^TestSpecificBehavior$' -count=1
```

For canonical non-race validation, use `GOMAXPROCS=2 GOMEMLIMIT=256MiB
GOFLAGS='-p=1 -parallel=1' make validate`. `GOMEMLIMIT` is a per-process soft Go
heap limit, not a hard system-memory cap; graphics/native allocations are not
covered. Full asset-backed race runs need separate resource planning and should
not run beside other test suites. An aborted or timed-out run is not validation
evidence for the tests it did not finish.

### Explicit Resource Profiles

Canonical test/validation Make targets accept `TEST_PROFILE=constrained` (the
default) or `TEST_PROFILE=throughput`. The direct inventory command accepts the
corresponding `-profile` flag; use Make to bound the outer `go run` bootstrap too.

| Profile | Go CPUs / soft memory per process | Logic scope | Native/Ebitengine-linked scope |
|---|---|---|---|
| `constrained` | 2 / 256MiB | `-p=1 -parallel=1` | `-p=1 -parallel=1` |
| `throughput` | 4 / 2GiB | `-p=2 -parallel=2` | `-p=1 -parallel=1` |

```bash
make test TEST_PROFILE=throughput
make test-provisioned TEST_PROFILE=throughput
ULTIMA5_REQUIRE_DATA=1 make validate TEST_PROFILE=throughput
make quality-gate TEST_PROFILE=throughput
```

Absent CPU/memory settings receive the selected defaults. Explicit inherited or
command-line `GOMAXPROCS`/`GOMEMLIMIT` must match the profile (canonical `256MiB` or
`2GiB` spelling); conflicts fail rather than silently overriding a smaller
budget. Unset conflicting values before selecting another profile. These exports
apply to canonical validation, not normal game execution. Caller GOFLAGS build
settings are retained; explicit gate arguments enforce concurrency and the
existing anti-filter/uncached/race/short rules.

The aggregate gate first runs logic packages, waits for completion, then runs the
render scope serially. Throughput raises concurrency only in the logic scope.
Classification uses Go's production and generated test-binary dependency
closures, including internal/external tests and
transitive Ebitengine subpackage imports. It is not a hardcoded list of three
UI packages. All Ebitengine subpackages, including audio, conservatively assign
the package to render: this is an honest linked-dependency scope, not a claim
that every assertion in that package needs native rendering.
Missing/incomplete dependency metadata fails closed. The original
repository inventory, including no-test packages, remains authoritative; a failed
lane leaves unexecuted work visibly incomplete. The profile never changes tests
to `t.Parallel`.

Large native application packages (main packages with at least 128 internal test
files) compile once and execute in at most seven serial feature contexts: combat,
dungeon, surface rendering, astronomy, title/creation, dialogs, and remaining workflows.
Each context is one native process for the whole feature group, not one per test.
The gate discovers every top-level test, example, and fuzz seed workload from the
compiled binary; every discovered name is assigned exactly once. It verifies all
top-level terminals, retains subtest failures/skips in the aggregate inventory,
and rejects omissions, unexpected assertions, duplicate execution, and nonzero
process exits. No discovery run is counted as validation. All contexts use the
package's original working directory and preserve the selected race/short/data
policy. Temporary test binaries are removed after the pass.

Constrained native contexts default `GOGC` to 50 when the caller has not selected
a value. This makes unreachable Go owners eligible for native resource cleanup
sooner; it neither forces GC after each test nor caps driver memory. Explicit
caller settings remain visible and unchanged. The 256 MiB soft Go memory target
is still separate from measured process and native-memory budgets.

Canonical validation/test Make targets also default `GOGC` to 50; caller-selected
values are preserved. Normal game execution does not receive these test defaults.

Make recipes are serialized even under inherited `-j`; this does not prevent two
separate validation commands from overlapping. The final quality-gate debugging
race pass retains conservative package/test concurrency. Do not run multiple
gates or agent test jobs together. A 2GiB target is per process, not an aggregate
native-memory cap. Use external supervision and host headroom assessment; the
profile is not a memory watchdog. Interrupted or unexecuted lanes are not
validation evidence.

## Continuous Integration

`.github/workflows/test.yml` currently defines the CI scope:

- The Debug Workbench frontend runs `make web-build`, checks embedded asset parity, provisions Chromium, then runs `make web-test-browser`.
- Debug control-plane race tests run on Ubuntu under Xvfb for `./internal/debugging/...`.
- The bounded logic matrix runs every logic-assigned package on Linux, macOS,
  and Windows, with DISPLAY/WAYLAND_DISPLAY cleared and core dependency checks.
- The required render matrix runs every Ebitengine-linked package on Linux under
  Xvfb and macOS/Windows using native backends. Backend failures fail the job.
- Production builds target Linux, macOS, and Windows with `CGO_ENABLED=0`, using
  shipped embedded/generated assets without a generation step. A successful
  build is not native graphics, audio, or input acceptance.
- Ubuntu quality checks provision tools and Workbench dependencies separately,
  require the Go/TypeScript scenario wire assertion with
  `ULTIMA5_REQUIRE_WEB_CONTRACT=1`, then run `make validate` without a display.
  This runs logic and static checks, not a second Linux native/full-suite pass.
- Go versions come from `go.mod`; the workflow does not duplicate a version literal.

These are public asset-independent jobs; optional-data skips remain visible.
They do not replace mandatory local strict private-data validation. Browser
tests exercise real DOM/input/layout with synthetic transport at desktop and
compact-laptop widths, not live game mutations. The four-test smoke scope above
does not establish full Workbench usability, desktop zoom, or physical-keyboard/
game-window acceptance.

## Test Review Checklist

- The test name describes behavior and includes `Integration` when target selection requires it.
- The test fails for the bug or missing behavior before the fix.
- Randomness and gameplay time are controlled.
- Real data is used only when the behavior depends on it.
- Missing data skips or fails intentionally.
- Callback assertions cover user-visible behavior without UI coupling.
- `FinishTurn` is called only at authentic turn boundaries.
- Assertions target stable behavior rather than incidental positions, logs, or implementation details.
- The focused test and the appropriate Make target pass.
