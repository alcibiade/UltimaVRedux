# Berad Audio

Experimental, media-free Go audio runtime for Ebitengine games. This module is
incubating inside Ultima V Redux with its own dependency graph:

```go
import audio "github.com/bradhannah/UltimaVRedux/modules/berad-audio"
import "github.com/bradhannah/UltimaVRedux/modules/berad-audio/backend/ebitengine"
```

No independently published version is claimed. Redux still uses its existing
implementation during staging; it has no local replacement or workspace dependency.
The ambient controller is available; music integration and the existing Bubble
Tea editor are not yet module APIs. The API is pre-stability; applications should
pin reviewed revisions once they are available remotely.

## Ownership and usage

Supply a playback `Definition`, an asset `fs.FS`, and an explicit encoded-byte
budget to `NewLibrary`. Metadata/windows/slices and unique asset bytes are frozen.
`Definition()` returns a detached view. Encoded bytes are eager-resident; decoded
streams are created per voice. The budget is not a total audio heap guarantee.

```go
host := ebitengine.Backend{Context: hostAudioContext}
options := audio.DefaultOptions(host, ebitengine.Decode)
runtime, err := audio.NewRuntime(library, options)
// Handle err; defer runtime.Close at the application's presentation boundary.
```

The host owns the Ebitengine context. The runtime never closes it. Device progress
requires the running Ebitengine game loop. The core package uses only the standard
library; Ebitengine and codec dependencies enter through the backend adapter.

`DefaultOptions` provides unity volume, eight voices and `time.Now`. A fully
explicit `Options` must supply its backend, decoder, clock and positive limit;
zero volume deliberately mutes. `DefaultPlayOptions` supplies unity request gain.

```go
request := audio.DefaultPlayOptions()
request.Loop = true
owner := runtime.NewOwner()
result, err := owner.Play("hangar/motor", request)
// Handle err/result; Release fades this lifetime, Close finishes it immediately.
```

- `PlayResult` distinguishes admitted, empty, inaudible and capacity-dropped
  requests. Unknown identities and invalid options remain errors, even while muted.
- Finite muted requests do not decode or advance selection. Loops retain their
  voice while category-muted so later volume changes preserve continuity.
- Owner-local named slots replace only after preparation succeeds. Independent
  `Runtime.Play` rejects slots. Empty slots allow overlap.
- `Voice.Stop` affects one instance; `Owner.Release` terminally rejects new play
  and releases its voices; `Owner.Close` finishes its tails. `StopCue` is explicitly
  cue-wide across owners. Zero/stale voice handles are safe.
- `Voice.SetGain`, `Pause`, `Resume`, `Seek` and `Position` provide presentation
  control without reopening retired voices. Pause retains the decoder and freezes
  its envelope; category changes still update a paused voice's mix. Positions are
  relative to the selected window. Invalid seeks do not change it.
- `Voice.Status` distinguishes playing, paused, releasing, completed, stopped,
  evicted and failed states. Retirement stores status without retaining its decoder.
  Unexpected loop completion is a failure, unlike healthy finite completion.
- Admission evicts the oldest finite voice; loops are protected. A new finite
  voice is dropped when all slots are loops; a new loop returns `ErrVoiceLimit`.
  Failed/dropped requests never advance the recording cycle.
- Call `Update` regularly, including paused/modal presentation frames. It services
  fades and retirement and drains contextual failures once. Pending errors are
  bounded to 32 with an overflow count. `Close` also reports pending failures.
- Snapshots report the last serviced gain without sampling clocks or advancing
  selection. Native device errors are limited by the host backend's capabilities.

Controls are mutex-serialized; decode/device/clock calls may block and must not
re-enter the runtime. Backend reads never acquire the runtime mutex. Read and seek
must be serialized by the backend. Retirement synchronously stops in-flight reads
before closing a decoder exactly once, including windows and loops.

Presentation time retains monotonic timestamps and clamps backward samples. It is
independent of game time and device consumption. Fades are linear amplitude, not
sample-accurate scheduling. Calibration above unity saturates signed 16-bit PCM;
this is not a mixer limiter or a claim of headroom across concurrent voices.

## Playback data and authoring

`Definition` is an in-memory projection, not a new JSON format. It contains only
recordings (exact file, calibration, optional source-relative window) and cues
(ordered recording IDs and explicit linear gain). Empty file means intentional
silence; cycle members cannot be empty. The caller supplies category/policy by
choosing runtimes/controllers and gains; score-selection policy remains game-owned.

Redux's schema-3 manifest, provenance, taxonomy and routing remain authoritative
for its TUI. Do not serialize `Definition` as a replacement authoring document.
All production recordings/notices remain game-owned. Examples/tests generate
their PCM/WAV; no bundled game media or original installation is required.

## Ambient sources

`NewAmbientController(library, DefaultAmbientOptions(backend, decode))` owns an
exclusive lane using the same decoder/voice/gain/envelope machinery as ordinary
playback. Defaults are 16 active-or-releasing voices, at most 4096 desired sources,
and retry delays of 1, 2, 4, 8, 16, then 30 seconds. Options are explicit/copied.

Submit a full `[]AmbientSource` to `Reconcile`, and call `Update` on presentation
frames independently of discovery. Each source supplies a stable ID, cue, loop
mode, fade durations and source gain in `[0,2]`. Source gain contains only intensity,
distance and caller overrides; the library applies cue gain/calibration exactly
once. Only single-recording cues are currently accepted. Finite ambient sources
have immediate onset and play once per presence or Cue/Loop change.

- Skipping Reconcile retains the preceding desired snapshot; an explicitly empty
  snapshot means departure and starts releases. All input is validated before
  desired state changes; a duplicate ID, unknown/cycle cue or invalid gain/fade
  rejects the entire snapshot. Playback failures after acceptance enter backoff,
  not rollback of the desired snapshot.
- Sources with different IDs remain independent even when sharing a cue. The
  consumer owns grouping, map queries/projection and source eligibility. The
  controller has no tile, lighthouse, bard, dungeon, vessel or JSON vocabulary.
- Live gain changes keep the decoder. Re-entry during release reverses its
  envelope from its current value, without restarting. This controller operation
  does not reopen a terminal `Owner.Release` or stopped voice.
- Admission ranks pre-category/pre-envelope gain, then retained voices, then ID.
  Category mute does not change rank or restart loops. Zero-intensity sources may
  retain loops for later intensity changes. Release tails occupy capacity; this
  lane never uses the SFX oldest-finite eviction rule.
- Failed replacement preparation keeps the old cue sounding. Retry attempts are
  presence/Cue/Loop-scoped and capped; repeated snapshots and temporarily successful
  reopenings do not reset recurring-failure backoff. Long gaps allow one attempt,
  not catch-up bursts. Backoff/completed sources do not reserve empty capacity.
- Empty recording slots retire prior voices without retries. `SetEnabled(false)`
  clears desired/active/retry state immediately; re-enable requires a fresh
  snapshot. `Reset` clears the lane without changing enabled preference. `Close`
  is terminal and does not close shared devices or sibling runtimes.
- `Snapshot` separates desired cue from actual voice (important after a failed
  replacement), includes completion/retry diagnostics, and never samples time.
  `Voice.TargetGain` excludes fade gain: a game can distinguish a muted source
  from the first zero-volume attack frame when implementing local-music priority.

`QuadraticDistanceGain` is an optional unit-agnostic helper preserving Redux's
ambient curve, including its one-unit inner range and radius-zero background
convention. The caller resolves world wrapping first. Positional SFX may use a
different profile; no stereo pan or occlusion is implied.

Redux owns the JSON-to-playback bridge. During staging it is compiled in the
game-owned `testdata/berad-audio-redux` integration fixture, alongside a source
planner shared with the old source-collection policy. It is not a dependency or
part of this module. The live application cutover still needs a resolvable pinned
module revision.

## Validation

From the application checkout, run `make audio-module-check`. The root logic test
inventory also executes this check, so `make validate` and `make quality-gate`
include the nested module. It copies only this module into a temporary tree,
disables workspaces/network resolution, rejects game imports/replacements, runs
uncached tests, vet, build and a separate consumer with a test-only local
replacement. Package completion and zero skips are checked from JSON events.
An additional game-owned bridge fixture projects the actual Redux JSON with
synthetic audio payloads and verifies grouping, gains, windows, variants and
source identity against this module. That integration build may import Redux;
the independent-module and unrelated-consumer proofs remain separate.
The root lint targets check the isolated module and staged bridge with the pinned
linter without a revision cutoff. The bridge's two test-only replacements are
checked explicitly instead of applying the blanket replacement prohibition to
that temporary fixture. The module itself permits no replacements. Analyzer
caches are scoped to each temporary tree. The full gate's race stage repeats the consumer/module
tests with race instrumentation; ordinary logic tests do not require race tools.
This establishes local independence, not remote publication/installation proof.

Provision the existing pinned tools and module dependencies through `make tools`
before offline checks. Controlled-transport tests and synthetic WAV decoding are
not native playback/listening acceptance.
