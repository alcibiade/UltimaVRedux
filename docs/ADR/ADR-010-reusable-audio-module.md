# ADR-010: Stage Berad Audio as an independent module

**Status:** Accepted for the initial extraction boundary

## Context

Audio runtime and authoring mechanisms are useful to another Ebitengine game.
The neutral facade currently shares a package with Redux policy and media, so
same-repository neutral tests do not establish an independent import boundary.
Redux is also installed through version-qualified `go install`; a local workspace
or replacement cannot become a required part of that consumer contract.

## Decision

Stage a media-free module at `modules/berad-audio`, declaring
`github.com/bradhannah/UltimaVRedux/modules/berad-audio`. Keep production assets,
semantic enums/callbacks, world interpretation, score scheduling/history and saved
preferences in Redux. Keep the Bubble Tea editor as the supported authoring UI.

The first module surface is a standard-library-only cue runtime with an explicit
playback definition, borrowed backend, decoder and presentation clock. A separate
Ebitengine package supplies the supported backend/codecs. It adapts the established
window/gain/read-failure/admission/ownership mechanisms rather than exporting the
game manager's subsystem graph. Explicit owner-local slots and playback outcomes
remove ambiguity at the application boundary.

The ambient controller uses the same voice primitives but owns its own admission,
reconciliation and retry policy. A game-owned source planner retains grouping and
world interpretation, passing source multipliers without duplicating the module's
authored cue/calibration gain. The staged Redux bridge is tested in a separate
consumer before the pinned-dependency cutover; it does not make the core depend
on the game. Paused, completed, failed and releasing voices have distinct states.

The finite music player shares the voice mechanisms while retaining muted onset,
transactional replacement and saved-position restoration after preview playback.
Score selection, quiet intervals and interruption/history decisions stay in Redux.

The playback definition is an in-memory projection, not another persisted asset
format. Schema-3 authoring data and the existing editor are retained while the
generic authoring/routing contracts are established. Music integration separates
transport from game policy without adding a speculative mixer,
new codecs, engine framework or frontend.

The initial staging period kept the game on its existing runtime. The authorized
module checkpoint is now available remotely, and the game pins it through ordinary
module resolution. Live scenes construct `GameAudioManager` over that dependency.
Never merge a dependency on an unavailable revision. Test-fixture substitutions
remain outside published application metadata.

## Consequences

- An isolated copied-module build and a separate consumer prove that the runtime
  cannot accidentally rely on game types, media, logging, settings or original data.
- Root Go package patterns exclude nested modules; an explicit root logic-test
  entry runs the module's test/vet/build and consumer checks. Root lint explicitly
  checks the module, and the full gate's race stage instruments its tests.
- Legacy runtime classes are test-only compatibility fixtures, excluded from normal
  builds. New facade/module tests verify the live path, while the older low-level
  regressions remain available during test migration. Shared TUI authoring helpers
  remain production code; no legacy playback fallback is compiled into the game.
- The module's initial API is experimental. A later repository move requires a
  deliberate import-path migration. No release, native acceptance or stable
  version promise follows from creating this directory.
- Public source installation resolves the pinned module without a local replacement
  or workspace. No independently named repository or stable release is implied.

See the [module contract](../../modules/berad-audio/README.md) and
[current Redux audio system](../systems/AUDIO_SYSTEM.md).
