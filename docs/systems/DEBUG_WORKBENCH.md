# Debug Workbench System

## Purpose

The debug workbench is a browser-based development tool for rapidly discovering commands, applying validated game alterations, managing save snapshots, and running repeatable scenarios.

The Go catalog and its exact browser fixture define the current command set. No
text-command compatibility grammar remains; the browser Workbench is the sole
debug interface.

Related documents:

- [Canonical Save Snapshot ADR](../ADR/ADR-009-canonical-save-snapshot.md)
- [Save Snapshot Schema](../details/SAVE_SNAPSHOT_SCHEMA.md)
- [Development Guide](../DEVELOPMENT_GUIDE.md)
- [Debug Workbench UI](../ui/DEBUG_WORKBENCH_UI.md)

## Goals

- Make commands easy to find by intent, category, stable ID, and parameter value.
- Generate friendly controls from typed command schemas.
- Keep all gameplay mutation on the Ebitengine update thread.
- Replace adapter-layer field mutation with authoritative core operations.
- Support immediate, synchronized, workflow, coordinated, and restart-required operations.
- Use canonical `SaveSnapshot` data for immutable checkpoints, rollback, and scenario baselines.
- Preserve deterministic seeds and structured results for repeatability.
- Retain recent navigation surfaces for fast iteration without bypassing command review.
- Keep the browser connected while `GameScene` is replaced.

## Non-Goals

- A remotely accessible administration server.
- Arbitrary reflection-based editing of `GameState`.
- Serializing transient runtime objects.
- Exact mid-combat save restoration.
- Preserving known defects in legacy debug handlers.
- Supporting multiple controlling clients.

## Technology

### Game Process

- Standard `net/http` server bound to `127.0.0.1` only.
- `github.com/coder/websocket` for the WebSocket transport.
- Embedded browser assets.
- Bounded inbound and outbound queues.
- Top-level control plane owned by `Game`, not `GameScene`.
- `ebiten.SetRunnableOnUnfocused(true)` so browser interaction does not suspend updates.

### Browser Client

- Vue 3.
- TypeScript.
- Vite.
- Native CSS without a component framework or utility-CSS dependency.
- Vue composables and reactive state without Pinia or Vue Router.

## Persisted Navigation Preference

System > Redux Modifications exposes `Gameplay.Navigation.FlexibleSextantUse`
through `redux.navigation.flexible_sextant_use.set` with a required boolean
`enabled` argument. Missing legacy configuration defaults to false without
rewriting the file on load. `config.UltimaVConfiguration.SetFlexibleSextantUse`
uses the normal atomic gameplay-preference persistence path, preserves unrelated
configuration, and restores the shared gameplay configuration field on failure.
Normal Configuration settings transactions also retain this preference.

`redux.modifications.status` returns
`ReduxModificationsStatusData.Navigation` as JSON `navigation`, containing
`flexibleSextantUse`. The live-safe setter returns that same typed status and
reports `gameplay.redux.navigation.flexibleSextantUse` only when its value changes.
This persisted preference is `ScenarioExcluded`: preferences outlive scenario
runs, and save-snapshot rollback does not restore them.

## Control Plane

### Persisted player preferences

System > Redux Modifications also owns combat speed, quick-save confirmation,
combat health bars and the simultaneous action-popup limit. These use the same
typed `UltimaVConfiguration` getters and atomic setters as player Configuration.
`combat.presentation.speed.set` remains a compatibility command ID, with integer
`percent` transport from 50 through 400 in steps of 50. It controls master combat
presentation, including animations and action pauses, through
`Gameplay.Combat.SpeedFactor`; Debug configuration is no longer a speed writer.
Active Combat exposes the same command and canonical status value.

`redux.saves.confirm-quick-save.set` and `redux.combat.health-bars.set` require
boolean `enabled`; `redux.presentation.action-popups.set` requires integer `count`
0–3, with 0 meaning Off. `redux.modifications.status` includes
`confirmQuickSave`, `combat.speedPercent`, `combat.showHealthBars`,
`presentation.maxVisibleActionPopups` and `configurationEditing`.
Optional `configurationDiagnostics` retains startup migration notices even if
categorized logging was disabled; Redux Modifications displays these notices.

Numeric parameter metadata includes optional `step`, anchored at `minimum`.
Go validates positive steps and grid membership before dispatch, and TypeScript
forms and specialized controls use those same catalog constraints. Out-of-grid
legacy speed requests fail validation rather than panicking. All persistent
setters are `ScenarioExcluded`, since save snapshots do not roll preferences back.
An open staged Configuration dialog rejects immediate preference mutations with
`configuration_editing` and an Apply/Cancel explanation. Failed persistence keeps
the live model and source unchanged; successful mutations refresh owning status
without discarding unrelated browser drafts.

```text
Browser/Vue
    |
    | JSON WebSocket messages
    v
Debug server read loop
    |
    | bounded immutable requests
    v
Application debug bridge
    |
    | drained at an update boundary
    v
Game / GameScene command executor
    |
    +--> authoritative GameState or PartyState operation
    +--> event-queue workflow
    +--> snapshot candidate and scene replacement
    |
    | structured results and events
    v
Debug server write loop --> Browser/Vue
```

Network goroutines never read or mutate live gameplay objects directly.

### Execution Ownership

Command ownership follows the lifetime of the resource being inspected or changed:

| Owner | Appropriate commands | Scene replacement behavior |
|---|---|---|
| Application (`Game`) | Process-wide logging, static catalogs, and services that survive gameplay scenes | Execute on the update thread without requiring a gameplay scene |
| Scene (`GameScene`) | Gameplay state, renderer state, dialogs, workflows, and scene configuration | Require the expected scene generation and reject stale requests |

Do not route process-wide commands through `GameScene` merely because one exists. Renderer and gameplay operations must not move to application scope to avoid scene validation.

## Command Contract

Each logical command has a stable definition independent of text grammar:

### Debug Tool Eligibility

Workbench operations are reserved for exceptional state, data that normal
gameplay cannot inspect, or setup that materially shortens development and test
iterations. Routine interactions that are immediately available in game do not
qualify. For additions, define a stable typed command, its owner and availability
reason, revision/review requirements, and detached result before wiring the UI.
Keep Go contracts and TypeScript protocol/forms synchronized. Add catalog/wire
coverage and rejection tests, then rebuild embedded assets with `make web-build`.
Browser controls must not bypass update-thread execution or confirmation.

```go
type CommandDefinition struct {
	ID               string
	Category         string
	Name             string
	Description      string
	Aliases          []string
	SearchTerms      []string
	Parameters       []ParameterDefinition
	ExecutionModes   []ExecutionMode
	Destructive      bool
	ScenarioTerminal bool
	ScenarioExcluded bool
}
```

`ExecutionModes` advertises the possible modes for discovery and UI planning. The authoritative executor resolves one effective mode from the command arguments and current context. For example, changing equipment may be synchronized live during combat, while changing party membership requires leaving combat and may require scene restart.

Scenario eligibility accepts a command when at least one advertised execution mode is scenario-safe and `ScenarioExcluded` is false. `ScenarioTerminal` is the single authority for final-step ordering in both catalog validation and browser planning. Registry construction also rejects commands with `ScenarioRuntime` parameters unless their execution registration provides a runtime resolver.

Parameter definitions require stable IDs and may represent:

- Enumerated choices.
- Bounded integers.
- Integer lists.
- Booleans.
- Pattern-constrained strings.
- Free text.
- Player characters.
- Locations.
- Monsters.
- Equipment and inventory items.
- Dynamic options provided from an update-thread snapshot.

The schema distinguishes serialized values from display labels and supports required values, defaults, descriptions, context availability, and control hints.

### Structured Invocation

Handlers accept typed arguments. Browser forms invoke stable command IDs directly; no text-console grammar is registered or translated.

### Retry-Safe State Changes

Remote mutations set an explicit desired state instead of toggling server state. Boolean commands accept values such as `enabled=true`, return the resulting state, and remain safe under retries and replay. UI labels may say “toggle,” but the emitted command must be an idempotent setter.

### Results

Commands return structured results rather than writing only to `text.Output`:

```go
type Result struct {
	SessionID              string
	RequestID              string
	CommandID              string
	SceneGeneration        uint64
	Invocation             *CommandInvocation
	EffectiveExecutionMode ExecutionMode
	Success                bool
	Summary                string
	ErrorCode              string
	ErrorMessage           string
	Changes                []StateChange
	Data                   json.RawMessage
}
```

Presentation adapters may render the summary, changes, and typed data differently. The browser must not scrape formatted console output.

Display status distinguishes requested and observed mode, pending transition,
last normal window size, actual window and monitor dimensions, outer container,
and the published logical render surface. Display mode and preset-size commands
are presentation operations: they request runtime-only state through
`DisplayManager` and never persist or bypass the player Configuration dialog's
Keep/Revert transaction. Unobserved platform requests expire after a bounded
transition interval so the instrument remains retryable.

## Execution Modes

| Mode | Purpose | Example |
|---|---|---|
| Query | Read an update-thread snapshot | Combat status |
| Live-safe | Apply one focused operation next update | Set gold |
| Synchronized live | Update related runtime structures atomically | Equip a weapon during combat |
| Workflow | Run gameplay orchestration and presentation | Moongate teleport |
| Coordinated alteration | Validate and apply several durable domains | Replace party inventory and statuses |
| Scene restart | Hydrate a detached complete state | Load or broadly edit a save snapshot |
| Presentation | Change or test game-window/browser presentation without gameplay mutation | Toggle an overlay or test a visual effect |

The server resolves the effective execution mode and availability for each request. Client metadata is advisory only.

## Specialized Instruments

Purpose-built instruments handle interactions that are poorly served by
generated forms, but they remain clients of the typed catalog. Companion
queries and mutations use the same stable command IDs, request validation,
update-thread bridge, stale-state checks, structured results, and session
reconciliation as generated controls. Immutable reference catalogs may be
application-owned so they remain available across gameplay scene replacement;
gameplay and renderer operations retain their normal scene ownership.

The current instrument inventory is maintained in [Debug Workbench
UI](../ui/DEBUG_WORKBENCH_UI.md). The Go catalog definitions under
`internal/debugging/` and their exact browser fixture define the current command
inventory; do not maintain a parallel Markdown ledger.

Audio auditioning and direct library editing are separate from the Workbench.
The standalone terminal editor is described in
[Audio System](AUDIO_SYSTEM.md#standalone-audio-editor); it exposes
no browser mixer or game-audio mute commands.

## Safety Rules

### All Mutations

- Validate all expected failures before the first mutation.
- Execute only at an update-thread boundary.
- Use stable identities rather than display names or mutable indexes where possible.
- Emit actual resulting values after clamping or normalization.
- Use centralized RNG and record explicit seeds for repeatable scenarios.
- Reject stale revision-based edits rather than silently overwriting newer state.

Dynamic provider choices are published as detached immutable discovery snapshots from `Game.Update`. Catalog requests read only the published snapshot, and commands using a provider include its expected options revision. Execution rejects stale revisions and revalidates the selected target before mutation.

Continuous discovery refresh is owned by the running Workbench server, including
when it is listening without a browser. Ordinary gameplay without that server
does not build developer options each tick. The bridge, catalog, and scenario
executor remain initialized: dynamic command admission refreshes current options
on the update thread before checking the expected revision, and the scenario
runner refreshes before stamping each step. Startup publishes fresh choices
before accepting HTTP readers. Shutdown/server failure retires reviewed options
across the dormant interval; scene replacement still invalidates generations and
options immediately. Network goroutines change only atomic publication demand,
never read option providers or live gameplay state.

Static command definitions are immutable and retained internally per `Game`.
Dynamic provider data is detached at publication, unchanged snapshots are reused,
and external discovery readers still receive deep copies of both definitions and
choices. This sharing does not permit modifying published or borrowed metadata.

### Combat

Selected equipment, HP, MP, and supported status edits may execute live when their authoritative operation synchronizes combat-derived state.

Party ordering, membership, Dexterity, combat location, turn ownership, and broad state replacement are not live-safe. They require combat exit or a scene restart from a safe baseline.

Because snapshot capture is unavailable during combat, restart-required alterations are unavailable until combat exits. The workbench must not offer a restart path that cannot first capture its required rollback snapshot.

#### Combat Target Identity

Each tactical encounter receives a new identity when combat is created or replaced. Each monster receives an immutable ID that is never reused within that encounter. Manager indexes and display names are telemetry only and must not target mutations.

Reviewed combat mutations carry the encounter ID, combatant ID, and expected snapshot or condition. Execution revalidates all three so a delayed request cannot affect a replacement encounter or a combatant whose state changed after review.

#### Reinforcement Admission

`GameState.SpawnDebugCombatMonsters` checks exact manager/queue identities for
both active and historical monsters; matching population counts alone is not
enough. Retired, fled and replaced records can remain in history without staying
in active rotation. Capacity comes from free source slots `6..31`, free slots
in the shared 32-object table, and distinct unoccupied authored spawn positions,
with the existing 16-monster per-request bound. Removing a corpse does not free
its retained dead source slot; released flight/retirement slots can be reused.

Preflight validates the complete batch before changing the seed, consuming RNG
or allocating IDs. An over-capacity exact request fails atomically; automatic
and maximum counts respect available capacity. Occupancy uses exact live source
and clone entries, so a remaining clone still blocks after its source departs.
Underground PCs do not block; protection fields block, while harmful fields may
share a spawn square but still consume object capacity. Loose objects and
corpses also occupy spawn positions. These are authored-position admission
rules, not an additional creature-terrain movement test.

Authority and regression coverage are in
`internal/game_state/active_combat_debug.go` and
`internal/game_state/active_combat_debug_spawn_test.go`.

### Snapshot Loads

- Capture a rollback `SaveSnapshot` automatically.
- Validate the complete stored snapshot and reviewed revision.
- Construct the replacement scene before discarding the current scene.
- Commit scene replacement only after rollback persistence and dormant candidate construction succeed.
- Keep the application-owned WebSocket connected across scene replacement.

## Equipment Transaction

`GameState.AlterEquipmentLoadout` owns the atomic inventory/equipment
orchestration shared by the normal Ready dialog and the reviewed
`party.equipment.loadout.update` Workbench action. Its current command contract
is defined in `internal/debugging/party_equipment_catalog.go`; authoritative
transaction behavior remains in `GameState` and its tests.

## WebSocket Protocol

Messages use a versioned JSON envelope:

```json
{
  "version": 1,
  "id": "request-42",
  "type": "command.execute",
  "payload": {}
}
```

Client-to-server message types:

- `session.hello`
- `catalog.request`
- `command.execute`
- `command.ack`
- `operation.status`

Server-to-client message types:

- `session.ready`
- `catalog.snapshot`
- `command.accepted`
- `command.completed`
- `command.failed`
- `protocol.error`

Use request IDs to reconcile reconnects and avoid duplicate mutations. Before
WebSocket acceptance, the server validates the request Host, Origin, and
process token and enforces single-controller ownership. Session creation or
resumption occurs afterward through `session.hello`.

Completed command results remain retained by session until `command.ack` arrives. A reconnect that supplies its previous session ID receives pending acceptance and completion messages again. The browser acknowledges every completed delivery, but a request ID already completed returns before changing pending state, scene generation, result history, instrument state, scenario drafts, or navigation. A repeated acceptance received before completion is ignored rather than acknowledged because `command.ack` releases a completed server record.

## Backpressure

- The game update never waits for a browser write.
- Each connection has one reader loop and one writer loop.
- Outbound queues are bounded.
- Submitted operations and completed or failed results remain retained by session until acknowledgment.
- Outbound saturation disconnects the slow client; retained operations replay after session resumption.
- Client result retention is bounded, and the UI reports trimmed results.

## Read-Only Log Transport

`/logs` is separate from the single-controller command WebSocket. It receives a distinct read-only token that cannot authorize commands and never acquires a controller session.

Accepted structured entries are retained in one bounded process-wide ring with monotonically increasing sequence numbers. Each reader supplies an independent cursor; reads never advance another reader. Responses report the next cursor, remaining backlog, and records overwritten before the requested cursor. Multiple tabs may poll concurrently, while filters, pause/follow state, and additional client retention remain local to each window.

## Save Snapshots

There is no separate debug gameplay state. The Workbench operates on canonical `SaveSnapshot` values, but it does not expose arbitrary field editing or a general patch protocol.

```text
stable GameScene --capture--> validated SaveSnapshot --> immutable named record
named/rollback record --revision check--> persist rollback --> dormant candidate
                                             |
                                             v
                                  atomic scene replacement
```

Initial snapshot capture is disallowed in transient contexts that are not semantically restorable. The workbench reports the reason and suggests exiting the context or using a scenario baseline.

The five catalog commands are:

| ID | Behavior |
|---|---|
| `save.snapshot.list` | List detached metadata, scenario references, and current capture/load/delete eligibility. |
| `save.snapshot.get` | Return metadata plus compact location, calendar, moon phase, and party detail; it does not return editable full JSON. |
| `save.snapshot.capture` | Create a new immutable `project:<id>` or `personal:<id>` record from a stable active scene. |
| `save.snapshot.load` | Require the reviewed SHA-256 revision, persist an automatic rollback, build a dormant candidate, and atomically replace the scene. |
| `save.snapshot.delete` | Revision-conditionally delete an unreferenced project or personal record; rollback records are not user-deletable. |

The default roots are `debug/snapshots/`, `~/.ultima_v_redux/snapshots/`, and `~/.ultima_v_redux/snapshots/rollback/`. Override them with `--snapshot-project-dir`, `--snapshot-personal-dir`, and `--snapshot-rollback-dir`. Records use the strict native format described in [Save Snapshot Schema](../details/SAVE_SNAPSHOT_SCHEMA.md), which owns supported versions and migration semantics. They are immutable after creation, canonically hashed, and protected with a shared stable-order scenario/snapshot advisory lock set. Personal and rollback storage is private (`0700` directories and `0600` files); project storage remains collaborative. Snapshot compatibility is checked against active runtime references, and unverified or incompatible records cannot be loaded.

## Scenarios

Scripted scenarios are durable, versioned JSON definitions. They are stored in two filesystem repositories:

- Project / Shared scenarios default to `debug/scenarios/` and are intended for shared, version-controlled definitions.
- Personal / Local scenarios are stored under `~/.ultima_v_redux/scenarios/` and remain outside the project repository.

Every scenario is addressed by a source-qualified reference such as `project:guard-check` or `personal:movement-test`. The slug after the colon is both the JSON filename and the definition's `id`; repository reads and writes reject mismatches, invalid references, and non-regular files. Saves use atomic file replacement.

The top-level Scenarios workspace opens on Library, groups definitions by Project / Shared and Personal / Local, and opens a selected definition in Details. Details exposes run, edit, duplicate, and delete actions. Create refuses an existing destination; update and delete require the loaded revision; duplicate requires the reviewed source revision and an absent explicit destination. Project writes and destructive actions receive explicit review in the browser.

Repository mutations are cooperative across Workbench instances and processes. A mutation acquires OS-held advisory locks on `.scenario-repository.lock` files in both configured repository roots in stable order, then performs its absence or revision checks while those locks are held. Update, delete, and duplicate recheck the relevant revision immediately before rename, removal, or destination creation. The operating system releases a lock when a process exits; the lock file itself may remain and is harmless. This protects cooperating writers, but an arbitrary external editor that ignores the advisory lock can still modify a file between the final revision check and rename/remove; portable filesystem APIs do not eliminate that residual race.

Legacy string character tokens on `party.character.update` and `party.equipment.loadout.update` are migrated in memory to one-based active-party positions when a definition is loaded. Revision calculation still uses the original decoded canonical definition, and loading never rewrites the source file. List/get results report each migrated JSON path, code, and before/after value; the runner and direct `--scenario` startup use the migrated definition transparently. Persisting the change requires an explicit revision-checked update, and duplicate is rejected until that update removes the pending migration.

### Scenario Definition

Schema version 2 contains an ID, name, description, discriminated baseline object, optional typed launch parameters, and at least one ordered step. It is the only accepted scenario schema. There is intentionally no schema-v1 migration or backward compatibility; old files are rejected and may be recreated in the editor. The supported baselines are:

- `current`: start from the active game state without resetting it.
- `init`: replace the active scene from a clean external `INIT.GAM` before executing steps while preserving the current Avatar name; if the incoming scene has no usable identity, the runner restores the default `Avatar` name.
- `snapshot`: replace the active scene from a reviewed immutable `project:<id>` or `personal:<id>` snapshot revision.

Project scenarios may reference only project snapshots so shared definitions remain portable. Personal scenarios may reference project or personal snapshots. Rollback snapshots cannot be scenario baselines. Referenced named snapshots cannot be deleted until those scenario references are removed.

Launch parameters may provide defaults and constraints. Command arguments are either JSON literals or explicit references in the form `{"parameter":"parameter-id"}`. Binding validates supplied values, applies defaults, preserves JSON types, and rejects unknown or missing required parameters.

Edit Scenario builds command steps directly from the scenario-safe command catalog. The command chooser searches names, IDs, categories, descriptions, aliases, and search terms, then groups matches by catalog category. A selected command is configured and validated before insertion, including literal values, compatible launch-parameter bindings, and an explicit insertion position. Its controls use each command's typed parameter metadata, including enums, bounded integers, booleans, strings, and revisioned dynamic options. Parameters marked `scenarioRuntime` are omitted from durable definitions and resolved from authoritative live state on the update thread immediately before execution. Resolver-backed character targets use explicit generic Party Position 1-6 planning choices: manually authored steps cannot enter a definition without a target, while captured live identity tokens convert to their one-based generic position. Editors can also add launch parameters and positioned notes directly, reorder or duplicate steps, and replace a command with explicit review when incompatible arguments will be reset.

Version 2 supports only two step types:

- `command`: invoke one normalized typed Workbench command with bound arguments.
- `note`: retain documentation in scenario order; the runner records it as skipped because it does not mutate game state.

Queries, presentation commands, and scenario-management commands cannot be command steps. Definitions contain no raw mutation closures or arbitrary Go expressions.

### Update-Thread Runner

Browser launches pass through Run Review. The review captures the saved definition revision, active scene generation, and, for named baselines, the snapshot revision; it presents typed launch-parameter controls and the ordered execution plan. `scenario.run.start` rejects stale definition, scene, or baseline revisions before mutation. Direct `--scenario` startup does not display the browser review screen; it loads and pins the current definition and named-baseline revisions before submitting the same typed runner request. After validation, the runner binds launch parameters and preflights every command against the authoritative catalog before queueing any step. The application-owned runner then advances during `Game.Update`, resolves runtime-managed preconditions, applies normal catalog normalization, and assigns the current scene generation and dynamic-options revision immediately before each invocation. It waits while the gameplay scene reports that scenario commands are not ready. Commands such as `combat.encounter.start` that deliberately enter a permanently unready context are terminal scenario steps; the editor labels them as final and inserts captured, added, and recorded setup steps immediately before them. Validation still rejects malformed external definitions containing later notes or commands.

Only one scripted run may be queued or running. Runs continue application-side independently of the browser connection. `scenario.run.status` merges the current process run with the newest completed private history records, including scene generation, step progress, bound command identities, and structured results. Successful, failed, and cancelled terminal runs are atomically retained newest-first on disk with a fixed capacity of 20; active runs remain process-local until they reach a terminal state.

Run-history schema v2 is independent of scenario-definition schema v2. It adds
`launch.parameterValues` containing actual accepted values (including defaults,
false, zero, and unused parameters), `baselineApplied`, and optional RNG
provenance observed when the baseline becomes active, before steps. Seed and
source-call counts are decimal strings to preserve uint64 precision in browsers.
Missing RNG means unknown, not seed zero. Frozen history v1 is read without
inventing launch values; its missing `launch` makes historical Review Again
unsupported. Reading does not rewrite old files. This history promotion does
not enable scenario-definition v1 migration.

`scenario.run.cancel` is application-owned and bound to the current run ID, not
the scene generation. It stops remaining dispatch when processed on the update
thread: the runner can advance before the bridge drains that request. It does
not interrupt an executing command, cancel production workflows, undo completed
mutations, or restore an applied baseline. Remaining step records retain their
pending state; progress/results are not fabricated. Terminal cancellation is
idempotent, persists history once, and releases only the scenario mutation gate;
ordinary workflow readiness still applies. A stale run ID cannot cancel a newer
run, and durable history grants no cancellation authority after process restart.

For `init` and named-snapshot baselines, the runner persists an automatic rollback before replacing the active scene and reports its reference/revision in Run History. Execution stops on the first command failure. Successful mutations and the failure-time state remain active for inspection; rollback is available as a recovery snapshot but is not restored automatically. A `current` baseline performs no scene replacement and creates no launch rollback.

Review Again prefills actual historical launch values, then requires a fresh
Run Review. A changed definition revision or unavailable recorded named-baseline
revision blocks the historical shortcut rather than substituting current
defaults. It is a new run, not deterministic session replay; diagnostic RNG is
not reapplied. A parameter named Seed has no effect unless bound to a command.
The existing `rng.seed.set` command remains combat-only, not a general surface
scenario reseeding API. Review Rollback hands the recorded reference/revision to
the existing Saves review, rejects changed/unavailable records, and requires
explicit confirmation before load. Cancellation and failure never auto-load it.

### Scenario Limitations

The current schema has no assertion, wait, branching, loop, snapshot-patch, or explicit rollback step. It does not capture final snapshots or automatically restore either the launch state, rollback record, or `INIT.GAM` after a run. In-progress runs are not persisted or resumed after process exit. Exact-party patching remains unavailable; use existing authoritative commands where their current contracts are sufficient.

## Running The Workbench

The server is disabled by default and never binds outside `127.0.0.1`. See the
[development quick start](../DEVELOPMENT_GUIDE.md#starting-the-debug-workbench) for launch flags and the
default browser endpoint. Frontend source is under
`internal/debugging/webui/client`; `make web-build` type-checks it and refreshes
the committed embedded assets.

## Running A Scenario From The CLI

`--scenario` skips the title menu and queues a source-qualified scripted scenario. Repeat `--scenario-param` for launch values:

```bash
make dev ARGS='--scenario project:guard-check --scenario-param seed=12345 --scenario-param enabled=true'
```

Each `--scenario-param` uses `key=value`. Valid JSON values retain their type; otherwise the value is treated as a string. Duplicate keys and malformed entries are rejected.

The relevant paths are configurable at launch:

- `--scenario-project-dir` selects the project repository and defaults to `debug/scenarios`.
- The personal repository is `~/.ultima_v_redux/scenarios`.
- `--scenario-history-dir` selects the private bounded run-history repository and defaults to `~/.ultima_v_redux/scenario-runs`.
- `--scenario-init` selects the clean baseline file and defaults to `data/init/INIT.GAM`.

`INIT.GAM` is copyrighted original-game data and is not included in this repository. It is required only for scenarios using the `init` baseline; such a run fails before its command steps if the external file cannot be loaded. A `current` baseline uses the active scene and does not load `INIT.GAM`.

## Validation Gates

- Exact zero-registration and Go/browser catalog parity tests.
- Schema and protocol decoding tests.
- Update-thread ownership tests.
- Queue saturation and reconnect tests.
- Fixed-seed command/scenario tests.
- Snapshot capture/hydration round-trip tests.
- Snapshot promotion fixture tests.
- Scene replacement cleanup tests.
- `go test -race` coverage for the bridge and server.
- Browser type-check and production build.
- Four Playwright Chromium smoke tests with synthetic transport cover desktop
  startup/navigation/status, one compact layout, mutation confirmation/cancellation
  and visible rejection, and keyboard palette focus/dismissal. Detailed component
  and protocol behavior remains in DOM/Go tests. Browser dock matrices, actual
  desktop zoom and exhaustive scenario workflows are outside this secondary tool's
  acceptance scope. These tests are not live-game mutation or usability evidence.
- Full repository validation for command-catalog, transport, and browser-interface changes.
