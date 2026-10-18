# Berad Audio

Experimental, media-free Go audio runtime for Ebitengine games. This module is
incubating inside Ultima V Redux with its own dependency graph:

```go
import audio "github.com/bradhannah/UltimaVRedux/modules/berad-audio"
import "github.com/bradhannah/UltimaVRedux/modules/berad-audio/backend/ebitengine"
```

No independently published version is claimed. Redux still uses its existing
implementation during staging; it has no local replacement or workspace dependency.
Music/ambient controllers and the existing Bubble Tea editor are not yet module
APIs. The API is pre-stability; applications should pin reviewed revisions once
they are available remotely.

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
choosing runtimes and gains; richer music/ambient controllers remain separate work.

Redux's schema-3 manifest, provenance, taxonomy and routing remain authoritative
for its TUI. Do not serialize `Definition` as a replacement authoring document.
All production recordings/notices remain game-owned. Examples/tests generate
their PCM/WAV; no bundled game media or original installation is required.

## Validation

From the application checkout, run `make audio-module-check`. The root logic test
inventory also executes this check, so `make validate` and `make quality-gate`
include the nested module. It copies only this module into a temporary tree,
disables workspaces/network resolution, rejects game imports/replacements, runs
uncached tests, vet, build and a separate consumer with a test-only local
replacement. Package completion and zero skips are checked from JSON events.
The root lint targets check the isolated module with the pinned repository linter
without a revision cutoff. The full gate's race stage repeats the consumer/module
tests with race instrumentation; ordinary logic tests do not require race tools.
This establishes local independence, not remote publication/installation proof.

Provision the existing pinned tools and module dependencies through `make tools`
before offline checks. Controlled-transport tests and synthetic WAV decoding are
not native playback/listening acceptance.
