# ADR-010: Extract Berad Audio and migrate it to the SDK

**Status:** Accepted; incubation completed and SDK dependency cutover implemented

## Context

Audio runtime and authoring mechanisms are useful to another Ebitengine game.
The neutral facade currently shares a package with Redux policy and media, so
same-repository neutral tests do not establish an independent import boundary.
Redux is also installed through version-qualified `go install`; a local workspace
or replacement cannot become a required part of that consumer contract.

## Decision

Initially stage a media-free module at `modules/berad-audio`, declaring
`github.com/bradhannah/UltimaVRedux/modules/berad-audio`. This established independent
runtime proof and a live game cutover before the separate SDK was available.
The runtime now resides at `github.com/bradhannah/berad-sdk/modules/berad-audio`.
Keep production assets,
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
module checkpoint became available remotely and the game pinned it through ordinary
module resolution. The subsequent SDK migration preserves the public runtime API
apart from import identity, with SDK-owned module/consumer checks registered before
publication. Redux now pins SDK revision `9e7e3ca45b46` and removes the incubation
module directory. Live scenes construct `GameAudioManager` over that dependency.
Never merge a dependency on an unavailable revision. Test-fixture substitutions
remain outside published application metadata.

## Consequences

- An isolated copied-module build and a separate consumer prove that the runtime
  cannot accidentally rely on game types, media, logging, settings or original data.
- SDK validation owns independent module test/vet/build, race and consumer checks,
  with source/tool/input-bound result reuse. The game retains only its bridge
  tests and lint against the exact published SDK pin. It needs no SDK checkout;
  its temporary bridge substitutes the candidate game alone.
- Legacy runtime classes are test-only compatibility fixtures, excluded from normal
  builds. New facade/module tests verify the live path, while the older low-level
  regressions remain available during test migration. Superseded authoring helpers
  are likewise test-only; no legacy editor or playback fallback is compiled into the game.
- The module's initial API remains experimental. This is an explicit import-path
  migration, not a stable-version or new native acceptance promise. Authoring/TUI
  extraction now lives in the independently versioned `tools/berad-audio` module.
- Public source installation resolves the pinned module without a local replacement
  or workspace. The SDK is independently public; no stable release is implied.

Reviewed SDK guidance is commit `45169251e58951593f25cd2f323ea4dea47784ff`;
the approved TUI and game-policy boundaries are preserved.

The tools cutover pins runtime/authoring `ed95f13c63b7` and tools `9cdfc4b0755f`.
Shared schema-3 editing/persistence lives in runtime `authoring`; terminal/session,
catalog and optional probes live in the tools module. Game launchers supply Redux
policy and retain source-evidence enrichment. The former TUI directory is removed;
old authoring bodies survive only as test-only parity fixtures. Make build/cleanup
targets preserve the existing workflow without a sibling SDK prerequisite.
Standalone SDK tool-install acceptance is deferred until reviewed default-branch
availability or separately authorized tagging; local builds do not supply that proof.

See the [module contract](https://github.com/bradhannah/berad-sdk/blob/9e7e3ca45b4616f5f7b1e8c177a756f3f224e1e0/modules/berad-audio/README.md) and
[current Redux audio system](../systems/AUDIO_SYSTEM.md).
