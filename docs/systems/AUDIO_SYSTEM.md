# Audio System

**Purpose**: Document the audio architecture that exists now. Historical rationale belongs in [ADR-007](../ADR/ADR-007-audio-system-architecture.md).

## Current Architecture

`internal/audio.AudioManager` borrows the process-wide Ebitengine audio context
and owns its voices. A host may create the context first, choosing its sample
rate; otherwise the first manager creates the 44.1 kHz default. Managers never
close the shared device. They coordinate these components:

| Component | Current responsibility |
|---|---|
| `SoundEffectPlayer` | Embedded sound assets, active sound instances, one-shot or looped streams, independent fade gains, and an eight-instance limit that protects existing loops |
| `BackgroundMusicSystem` | Embedded finite music playback, current track, and track volume |
| `MusicManager` | Location/context selection, finite score policy, interruption ownership, and session history |
| `AmbientSoundManager` | Persistent proximity loops for tile families/dynamic sources and optional non-spatial dungeon backgrounds |
| `WalkingSoundManager` | Surface-sensitive footstep selection |
| `PositionalAudioCalculator` | Distance attenuation and a calculated pan value |
| `AudioConfiguration` | Sound-effect, music, and ambient volume plus distance/rolloff settings |

`AudioManager` creates all of these in its constructor. A missing manager or subsystem after successful construction is a programming error. `GameScene` may intentionally have no `AudioManager` when sound is disabled or when recoverable audio initialization fails; its required `AudioCallbacks` are still wired so core gameplay never depends on a concrete audio package.

Audio assets are embedded under `internal/audio/assets/`. The decoder supports
OGG, MP3, and WAV. The authored library selects an exact filename and extension;
playback never guesses between similarly named files.

With Ebitengine 2.10, owned gameplay voices retire through
`Player.PauseAndStopReading`, a synchronous barrier that stops decoder read-ahead.
The calibrated `gainPlayer.Close` wrapper retains terminal ownership: retired
voices cannot play or seek again. Managers drop those references after retirement;
decoder readers never acquire manager locks while the barrier waits. The raw
preview/source-set players likewise stop reading before their owners release them.
Ordinary Pause remains resumable.

`AudioManager.Close` is terminal and idempotent. `SetEnabled(true)` cannot reopen
a closed manager; subsequent playback and category setters report `ErrAudioClosed`.
`GetConfiguration` returns a detached snapshot, and category setters are the live
mutation boundary. Walking requests serialize their shared variant sequence.
The manager facade coordinates configuration and voices; callers must not combine
concurrent direct subsystem control with facade operations.

The in-game ambient preview applies recording calibration, cue gain, and category
gain through the calibrated player. Category changes affect its existing voice;
global audio disable retires it together with ordinary playback.

## Integration Boundary

Core gameplay identifies sounds with `game_state.SoundEffect` and calls `SystemCallbacks.Audio`. It does not import `internal/audio`.

The current callback contract is:

```go
type AudioCallbacks struct {
    Play func(SoundEffect) *SoundBuilder
    Stop func(SoundEffect) *StopBuilder

    PlaySoundEffect   func(SoundEffect)
    RequestPlayback  func(SoundPlaybackRequest)
    RequestStop      func(SoundStopRequest)
    UpdateLocationMusic   func()
    PauseBackgroundMusic  func()
    ResumeBackgroundMusic func()
}

type SoundPlaybackRequest struct {
    Effect   SoundEffect
    Mode     SoundPlaybackMode
    Volume   float64
    FadeInMs int64
}

type SoundStopRequest struct {
    Effect    SoundEffect
    FadeOutMs int64
}
```

`NewAudioCallbacks` requires the playback request, stop request, and three music lifecycle functions. The constructor adds `Play`, `Stop`, and the widely used `PlaySoundEffect` one-shot convenience. Gameplay still depends only on `game_state` types; the `GameScene` adapter translates requests to `AudioManager` operations.

Both `NewAudioCallbacks` and `NewSystemCallbacks` validate all eight resulting
capabilities. Missing wiring is a programming error and panics during
construction, even when sound is disabled. Aggregate validation preserves the
supplied functions without invoking or rebuilding them; action call sites do
not silently ignore missing callbacks.

## Supported Usage

Use the convenience callback for an ordinary effect:

```go
gs.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundUnlock)
```

Use builders when loop, fade-in, or stop behavior is needed:

```go
gs.SystemCallbacks.Audio.Play(game_state.SoundWind).Loop().Start()
gs.SystemCallbacks.Audio.Stop(game_state.SoundWind).FadeOut(500).Now()

gs.SystemCallbacks.Audio.Play(game_state.SoundEarthquake).Volume(0.5).FadeIn(500).Start()
gs.SystemCallbacks.Audio.Stop(game_state.SoundEarthquake).FadeOut(500).Now()
```

`SoundBuilder.Start` emits exactly one `SoundPlaybackRequest`. The request mode is
authoritative: `OneShot()` creates a finite stream and `Loop()` creates an infinite
stream. There is no competing `IsOneShot` metadata flag. One-shots receive
natural-duration cleanup; loops remain active until an explicit stop request.

Builder volume is clamped to `0.0` through `1.0` and propagated with optional fade-in. Effective playback volume composes request or positional volume, the cue modifier, recording calibration, the category preference, and the fade envelope once in linear amplitude. Positive calibration may produce gain above unity: the shared calibrated player applies that excess to decoded PCM while keeping the device slider in `[0,1]`. PCM samples saturate rather than overflow; zero is mute. User sliders alone use `ConvertSliderToVolume`.

Timed playback and playback-side fade-out are intentionally not builder features. Event workflows own duration and dispatch a stop request from their normal or `Finally` cleanup. Stop fade-out begins at each player's current volume, including when it interrupts fade-in.

## Sequencing With Effects

Audio and visual effects are independent callbacks coordinated by the owning sequence. There is no combined screen-shake/audio callback.

```go
manager.Workflow("earthquake").
    Priority(event_queue.PriorityHigh).
    Sequential().
    BlockInput().
    First(func() {
        callbacks.Audio.Play(game_state.SoundEarthquake).FadeIn(500).Start()
    }).
    ThenSequence(game_state.NewPartyStateBuilder(gs).
        ToAllParty().
        WithShake(4000, 9.6).
        Build()).
    Finally(func() {
        callbacks.Audio.Stop(game_state.SoundEarthquake).FadeOut(500).Now()
        callbacks.Visual.SetScreenShakeOffset(0, 0)
    }).
    Start()
```

Use `TriggerFeedback` instead when the entire sequence is presentation-only and must not block input. Gameplay mutations do not belong in feedback callbacks.

## Sound Effects

`internal/audio/assets/library.json` owns recording identities, exact file
assignments, cue display names, authored volume modifiers, and in-game audition
instructions. `soundEffectMetadataMap` in `internal/audio/sound_helpers.go` is an
immutable projection of that validated manifest. Each entry supplies:

```go
type SoundEffectMetadata struct {
    Name           string
    RecordingID    string
    VolumeModifier float64
}
```

The manifest is validated once when constructing derived metadata. Asset streams
are lazy-loaded under the manager's exclusive lock on first use. Every known cue
must reference a registered recording. A recording with `file: null` is an
intentional silent slot, including formerly metadata-less SFX. An unknown cue,
unknown recording reference, non-null missing/unsupported file, malformed JSON,
duplicate key, or missing mandatory manifest field is invalid embedded data.
Parsing returns a contextual error; loading an invalid production manifest panics.

Schema v3 requires exhaustive authored `taxonomy` groups and `routing`. Each recording
belongs to one acoustic family (what its file sounds like, or the intended
family of a null-file slot); each cue belongs to one semantic profile defining
target acoustic family, event phase/outcome, mix role, emitters and contexts.
Profiles are controlled reusable definitions, not playback instructions.
`sfx`, `music` and `ambience` remain derived from cue IDs. The catalog reports
differences between cue intent and assigned recording as review findings, not
invalid assets or proof of listening quality. A newly created empty editor
slot starts in `unreviewed` until its intended family is selected. Older schemas
are rejected rather than silently guessed into classifications or routing.

Acoustic family is independent of gameplay context and playback category:
Nearby Fire targets `fire` while using the Ambience slider; a ritual earthquake
targets `impact` with `ritual` context, and magical ritual cues target `magic`.
Prepared empty slots should describe their intended acoustic content consistently
with the owning cue. Preserve genuine mismatch warnings for unrelated stand-ins;
do not reclassify a healing recording as physical impact to hide an assignment issue.

The original game's calls and Redux audio opportunities are separate evidence:
ordinary Get, Ready, Board/Exit and Ignite had no verified direct original cue.
Redux reserves silent `file: null` slots at committed results so future sound
authoring needs no gameplay rewiring. Spell activation and a potion's actual
effect remain separate from consumption. Visual-only `PartyStateBuilder` calls
request no audio; explicit `WithSound(SoundNoEffect)` requests the real dud cue.

Environmental hazards request semantic effects rather than catalog numbers. Lava
and fireplace damage use `SoundFireBurn`, initially backed by the credited
burning-cardboard recording. Mountain-climbing and trapdoor falls use
`SoundFallingMinor` and `SoundFallingSevere`; their prepared null-file recording
slots are silent until filled. The original DOS routines use generic damage noise
without distinguishing fall severity; this semantic split is Redux presentation
policy rather than an original sound distinction.

Each playback request creates one fresh decoded stream and one Ebitengine player. `SoundEffectPlayer` tracks active instances by generated ID and request mode. `AudioManager.Update` retires finite voices when the player reports completion and advances fades; there is no one-shot cleanup goroutine. The eight-voice limit evicts the oldest finite voice, not an existing workflow-owned loop. A new loop fails explicitly if all eight slots are loops; a routine one-shot is dropped with audio debug logging in that case. Stop requests still target every instance of their effect.

Morning Star and Halberd commitment requests `SoundCombatSwing`, including hits,
misses and empty attacks at range 1 or 2. Their missile type still governs hit
resolution; it does not select the projectile-launch recording or admit a
travelling graphic. Ordinary melee retains its swing and bows retain
`SoundCombatRangedLaunch`. The latter's production recording is currently an
empty slot, which previously also silenced reach attack onset. The original
plays a synthesized glide after accepted aim and before hit/miss resolution,
while skipping travelling missile presentation for the two reach weapons.
The attack/obstruction contract is [COMBAT-0004](../gamefacts/COMBAT.md#combat-0004).
Redux deliberately uses the existing physical swing foley for these weapons
rather than reproducing the shared synthesized glide.

The reused `combat_swing_01` is XxChr0nosxX's CC0
[Swing.mp3](https://freesound.org/people/XxChr0nosxX/sounds/268227/), with the existing
350–910 ms playback window. Its bytes and cue gain are unchanged. This fixes the
silent mapping; callback and decoder tests do not establish in-game listening
acceptance.

Creature interference requests `SoundCombatInterrupted` when
`CombatEventInterruption` enters the scene's Battle Report. The same committed
rejection keeps the existing classic-output message and identifies the physical
source or Cast; preview queries, report rendering and history browsing request
nothing. Each rejected source/action owns exactly one one-shot request, with no
loop or input-blocking audio workflow. The original prints interruption text
without an audio call, so this semantic cue is Redux feedback policy; see
[COMBAT-0012](../gamefacts/COMBAT.md#combat-0012).
`combat_interrupted_01` is intentionally `file: null`, with complete provenance
and combat/failure classification in `internal/audio/assets/library.json`.
Assigning a future brief interruption recording and rebuilding requires no new
event routing. Callback/catalog tests establish dispatch and safe silence, not
listening acceptance.

Metal-floor footsteps are a bounded exception to independent overlap. The source
`sounds/footsteps/stone_steps_01.ogg` contains four isolated footfalls in 1.93 seconds.
Four metadata-only recording entries share that file: `stone_steps_01` uses
40–260 ms, `stone_steps_02` 560–790 ms, `stone_steps_03` 1060–1270 ms, and
`stone_steps_04` 1570–1780 ms. `SoundWalkMetal1` cycles them in authored order;
each new grate request replaces the preceding grate voice. The source bytes
are unchanged. Brick/stone flooring retains its existing four concrete-file
assignments and authored surface-cue cycling. These windows isolate four measured
bursts; they do not claim perceptual acceptance as metal-grate sounds.
Catalog assets expose each clip's window alongside full-source measurements;
cycle moments link every member and expose their selection policy.
The excerpt belongs to the recording, not the cue. Assigning a new single-footfall
recording without a `playbackWindow` does not inherit the old trim. Window bounds
are validated and must fit the decoded source. Duration accounting for SFX and
music uses stereo PCM frames, not byte count divided only by sample rate.

Fade bookkeeping and sound cleanup use wall-clock time because they are presentation resource management, not authoritative gameplay timing.

## Music

`BackgroundMusicSystem` plays one finite embedded track and reports actual device
completion. `MusicManager` maps locations and named contexts to tracks and owns
the presentation-only policy in `internal/audio/music_policy.go`. Scheduling uses
an injected presentation clock and fakeable playback boundary; it never reads or
advances gameplay time or consumes gameplay randomness.

The provisional tuning is deliberately subdued. These are code-authored policy
values, **not listening-accepted settings** or new player preferences:

| Treatment | Provisional value |
|---|---|
| First exploration presentation of a theme in the application session | −10 dB relative to the existing calibrated track/category mix |
| Every later presentation of that theme | −16 dB relative to that same mix; no cumulative attenuation |
| Ambience-only interval after completion or switching away | 5 minutes of unsuppressed presentation time |
| Score fade-in and location-switch fade-out | 2 seconds; one voice, no crossfade |
| Local music release grace after the final audible bard/instrument voice | 3 seconds |

Floor changes and locations sharing a theme preserve its player, position, gain
and quiet interval. A different theme fades the old one out, then respects the
global quiet interval. Returning before the fade finishes reverses the envelope
without restarting. Rapid travel cannot chain introductions. A theme receives
its first treatment only once per application session; absence does not reset it.

Pause/menu, journal, mute/disable and cinematic owners independently suppress the
score and freeze scheduling. Existing camping/death pause callbacks are balanced
with a nested workflow count. Cinematic ownership also covers isolated overlay
scenes, Shadowlord presentations and the finale; routine action workflows do not
duck music. `GameScene` updates audio once at the end of every frame, including
modal/cinematic early returns. The title scene advances its own audio manager.

Bard proximity voices (including their fade tails) and playing harpsichord/lute
voices take precedence over score. A release grace prevents chatter between notes
or at the edge of bard range. Muted or missing local recordings do not claim
ownership. Interruptions pause the score immediately; restoration fades in from
silence at its saved position. A changed location discards stale paused score and
returns through the quiet interval instead. Resuming one owner cannot release
another active interruption, including an interruption begun during silence.

Settings preview owns playback under the menu hold and auditions the authored
mix independently of policy attenuation. On return, a retained or recreated
stream seeks to the saved score position—even when the preview used the same
track. Preview completion never advances exploration history. Explicit named
context cues may bypass exploration cooldown, play once at the subdued initial
level, then restore current exploration through a quiet interval.

Active tactical combat selects the manifest's `context/combat` assignment
(initially **Engagement and Malee**, track `combat`), bypassing exploration quiet
intervals and theme-repeat history. Combat gain follows the encounter role, not
the selected track's name. The
scene publishes the authoritative encounter identity each frame, covering
surface encounters, dungeon fights and camping ambushes. The score fades in at
its authored combat mix and the user's Music volume, without the extra exploration
attenuation, and plays once per encounter; frame/turn updates do
not restart its opening, and exploration stays silent if the score ends before
combat does. Menu, mute and special-workflow ownership still apply. Combat exit
fades the score out and restores the actual destination's exploration policy
through a quiet interval. World-map bards are not sources on a tactical map.

Frigate sails own the separate **Sailing** music moment (`music/track/hornpipe`,
recording *Capn Johnes Hornpipe*). Raising sails starts one finite presentation;
ordinary movement, turning, and interrupted automatic sailing do not restart it.
Natural completion leaves exploration music silent until the sails are furled.
Furling or leaving the frigate fades the track; raising sails again permits a new
presentation, even during an exploration quiet interval. Combat takes precedence
and a presentation already started before combat does not replay afterward.
Buccaneers Den and the named ship locations no longer select Hornpipe merely
because the player enters them. `routing.naval.sailing` is the authored music
assignment, independent of onboard ambience and the Music volume preference.
A missing recording or failed playback attempt consumes that hoist's presentation;
frame updates do not repeatedly retry it. Furling and raising sails permits a new
attempt. Sailing alone is sufficient to trigger lazy music-asset preparation.

`Game` retains detached `MusicHistory` across load, new-game and main-menu round
trips. Scene activation imports only heard-theme history, remaining quiet time,
and the explicit music on/off preference; live players, pending cues and
interruption owners never transfer. Replacing an active score starts a full quiet
interval. Failed scene candidates cannot mutate the retained history. History
is not saved to disk and resets on application restart. The music toggle uses
enabled policy, so a quiet interval is not mistaken for music being off.

Missing recordings and failed starts stay silent. Exploration defers retry by a
full quiet interval; encounter and sailing presentations are consumed until a new
encounter or hoist. Natural completion closes its player. Policy tests cover completion,
repeat gain, transition reversal, nested owners, preview replacement, local music,
mute/disable, special cues and detached session history without real-time waits;
player/stream tests additionally verify restoration of decoded-stream position.
Those tests do not prove device consumption: Ebitengine initializes its device
only while the game loop runs. Native playback and listening are separate checks.
Final acceptance requires representative in-game long-session listening, with
recording identities, category sliders, listening setup and accepted policy values
recorded in the owning issue.

## Ambient And Positional Audio

All seated tavern bards use the dedicated `tavern_lute_01` recording slot through
`ambient/AmbientBard`; `sfx/SoundTavernBardMusic` shares that slot. It deliberately
has `file: null` and unknown provenance pending the user's recording, so tavern
performances are silent and claim no local-music ownership. Populate that manifest
entry (including truthful provenance) and rebuild to enable playback through the
existing proximity, fade and volume lifecycle. Camping's `SoundBardLute` retains
the short `bard_lute_01` recording. Loop-seam and extended listening acceptance
apply when the future tavern recording is supplied.

Fogsbane, Stormcrow, Greyhaven and Waveguide supply `AmbientLighthouse` throughout
every small-map floor, using the existing seagulls recording and calibrated cue
level without distance attenuation. This location source reuses the static
lighthouse cue's voice: movement, floor changes, and overlapping visible tile
sources cannot create duplicate loops. On exit it returns to normal proximity
attenuation if a lighthouse tile is audible, otherwise fades and retires through
the ambient manager. The overworld tile's authored radius remains unchanged;
Ambience volume, mute and suppression apply normally.

`GameScene.updateAmbientSounds` gathers terrain tiles and dynamic sources such as
visible moongates and seated bards. Its scan radius follows authored ambient
ranges; occupants do not hide terrain sources. `AmbientSoundManager` coalesces
static sources to the strongest audible source per cue identity, after each tile
family's range and relative-level overrides. A quieter nearby torch cannot mask
a louder brazier using the same sound. Ties prefer the nearer source; changes in
the winning source update the existing player's gain/range without restarting it.
`AmbientSoundType` is a
string identity, not a closed enum: adding an ambient cue/profile does not require
a Go constant. Dynamic sources include cue identity and position in their keys.
The manager composes distance, source/context, recording calibration, category,
and fade gains. Infinite decoded loops retain their players during updates,
strongest-source changes, and fade reversals. Scene suppression/global disable
close active voices; restoration rebuilds sources on the next scene update.

### Authored Routing And Animation Families

The manifest's `routing` object is authoritative for:

| Field | Contract |
|---|---|
| `ambient` | Cue ID → `radiusTiles`, `fadeInMs`, `fadeOutMs`. Radius 0 is a non-spatial background; positive radii are proximity sources. Radius is limited to 32 tiles and fades to 60000 ms. |
| `tiles` | Canonical tile ID → optional `ambientCue`, `surface`, `radiusTiles`, and `gain`. Empty ambience means none; omitted surface inherits `defaultSurface`. Missing radius/gain inherit the sound's range and 100% mix. Explicit gain 0 mutes the source. |
| `surfaces` | Surface identity → ordered SFX cue IDs. An empty list is a silent surface. |
| `defaultSurface` | Required known surface for tiles without an override. |
| `music` | Stable location/world/context target → music cue, or explicit `null` silence. Missing targets inherit required `default`. |
| `dungeon` | Dungeon location target → non-spatial ambient cue, or explicit `null` silence. Missing targets inherit required `default`, initially silent. |
| `naval` | Required `frigate`, `skiff`, and `sailing` situations. Frigate/skiff accept non-spatial ambient cues; sailing accepts a music cue. `null` is deliberate silence. Code owns boarding and sail-state eligibility. |
| `emitters` | `bard` and `moongate` → spatial ambient cue. Code still owns emitter activity. |

`internal/sprites/indexes/animations.go` supplies the atlas-family identity shared
by rendering and audio. Fountain frames 216–219 are one Fountain target, stored
only under `216`; noncanonical member overrides are rejected. Visual frame changes
do not rewrite the map tile, start additional players, or restart an envelope.
Tile overrides can give a brazier, torch and wall sconce different hearing
distances and levels while all reuse the same Fire sound and recording. The
relative `gain` is a multiplier (0–2, displayed as 0–200%); it composes with cue
gain, recording calibration, category volume, distance attenuation, and fades.
Tile radius overrides must be positive and at most 32 tiles; the scene scan
includes their ranges. Removing the ambient assignment also removes obsolete
range/level overrides. Animation frames inherit their canonical family's overrides.

Pixel effects on one tile, directional sprites, stream shapes, and distinct
magical fields do not become atlas-frame families merely because they animate or
are adjacent in the atlas.

Fountain ambience uses `ambient/AmbientFountain` (display name **Fountain**) and
`fountain_gurgle_01`. Drinking requests the separate one-shot
`sfx/SoundFountainDrink` / `fountain_drink_01` recording slot.

Britannia and the Underworld have separate `world/overworld` and
`world/underworld` music targets, despite sharing a gameplay location ID. Both
initially retain their preceding overworld-track assignment. Selection changes
retain the existing finite music, quiet-period and interruption policies.

Dungeon backgrounds belong to the dungeon visit, including tactical rooms whose
pre-combat location is that dungeon. They use one persistent cue-keyed loop at
full spatial gain, independent of listener motion. Nearby tile sounds coexist
with the background. Departure removes the source and fades it out. Dungeon
background selection does not create an outdoor background layer or per-floor
overrides. The default remains silent; all eight dungeon locations explicitly
select one of two prepared, intentionally silent Moments:

| Dungeon style / locations | Cue and recording | Availability |
|---|---|---|
| Cave/mine: Despise, Destard, Doom, Shame, Hythloth | `ambient/dungeon_cave` / `dungeon_cave_01` | Empty recording slot; 0.22 provisional cue gain |
| Brick: Deceit, Wrong, Covetous | `ambient/dungeon_brick` / `dungeon_brick_01` | Empty recording slot; 0.16 provisional cue gain |

The grouping follows `dungeonWallSetForLocation`: cave and mine wall sets share
the cave-drip slot, while brick walls use a separate ambience slot. Both have
`file: null` and create no playback voice until a recording is assigned. Once
populated, they fade in over 1500 ms and out over 1200 ms, use the Ambience
preference (including live changes and zero volume), and are retired by global
audio disable. Moves, turns, floors and dungeon combat retain the same loop;
returning from combat does not restart it. Town, surface and Underworld
exploration receive neither bed. Nearby tile effects and encounter music remain
independently controlled.

**Supplying the recordings:**

In the Audio Editor, select **Dungeon Cave Drips** or **Dungeon Brick Airflow**
in **Moments**, or import into the corresponding empty recording in
**Recordings**. Save the library and rebuild the game to embed the recordings.
Their dungeon assignments and loop lifecycle are already wired; no gameplay
code change is needed. Calibrate and tune the actual recordings when supplied,
recording their provenance then rather than retaining another asset's metadata.

`internal/audio/dungeon_ambience_test.go` verifies production routing, empty-slot
silence, and stable voice ownership, mute/live gain and exit fades with a
test-only recording assignment. These are wiring checks, not listening
acceptance of future recordings.

Frigate and skiff situations already have separate **Frigate Ambience** and
**Skiff Ambience** Moments assigned to them. Each Moment may have a recording
or an intentionally empty slot; the editor shows its current availability.
An empty slot remains silent until a recording is assigned. The onboard
loop uses the Ambience volume preference and follows the boarded vessel through
movement and sail toggles, fading on departure or vessel change. Sails control
the separate, finite Music-category Sailing moment, not the onboard loop.
Skiffs have no sails and never trigger Sailing music. Per-step `SoundMoveSkiff`
and `SoundMoveFrigate` effects remain independent authorable cues.

`AudioManager.PlaySoundEffectAt` uses `PositionalAudioCalculator` for distance volume. The calculator also returns stereo pan, but Ebitengine playback does not currently apply that pan. Do not claim stereo positioning until that TODO is implemented.

## Volume Settings

External configuration uses integer sliders from 0 through 10. `ConvertSliderToVolume` maps them to amplitude values and currently caps slider value 10 at `0.75`. Non-preference gains are continuous amplitude multipliers and never pass through that curve. Authored recording `calibrationGainDb`, when present in `assets/library.json`, is converted by `10^(dB/20)` and is validated to `[-48,12]` dB. An absent value is unreviewed and plays at unity; zero-amplitude mute is not represented by finite dB. Cue `volumeModifier` remains the existing contextual amplitude coefficient. The shared `gainStream` supports boosts above unity in SFX, music, ambience and editor audition. Sample clipping and the sum of concurrent voices still require listening and suitable headroom.

There is no master-volume field. Effective volume uses the category setting plus per-sound, per-track, ambient, or positional modifiers.

Runtime category changes update active SFX, music, and ambient players. SFX retain request gain and their current fade factor; music retains its track and contextual gain; ambience retains source/distance and its current envelope. Updating a category does not restart playback or modify the authored cue coefficient. Configuration-dialog Cancel restores the previous category preferences.

## Standalone Audio Editor

`cmd/audio-editor` is a Go/Bubble Tea developer tool. It reads the real
`internal/audio/assets/library.json` and recordings from disk, independently of
the game and browser Workbench. Run from the checkout root:

```bash
make audio-editor
./bin/audio-editor
# Or run without first building a binary:
go run ./cmd/audio-editor
```

`-library /path/to/assets/library.json` selects another library. `-check`
validates schema, references and authored trim bounds without audio playback or
file writes. `-probe healing_01` briefly plays a recording and checks that the
audio device advances without a graphics/game loop. It verifies device progress,
not human listening acceptance.

Editor audition owns a process-wide Oto context directly, independent of
Ebitengine's game-loop-driven audio initialization. It uses signed 16-bit stereo
PCM at 44.1 kHz and the same window/loop decoder and gain stream as gameplay.
The playhead subtracts device-buffered bytes from decoded progress; seeks are
frame-aligned and source-relative window mapping remains in `Audition`.
Voice replacement and shutdown use `PauseAndStopReading` before discarding the
voice. Oto permits one device context per process: standalone audition and the
game's Ebitengine context belong in separate processes.

### Editing And Playback

The Recordings workspace edits source assignments, shared calibration, notes,
and non-destructive playback windows. Moments edits the actual production cues,
their recording assignments and contextual amplitude. A missing recording is
visible as an empty slot. Choosing a different source clears its old trim.
A manual calibration remains locked to the recording identity; auto-managed
calibration is recomputed for the replacement before the edit is published.

There are four workspaces: **Recordings**, **Moments**, **Tiles**, and
**Locations**. The active tab uses a filled high-contrast keycap, uppercase text,
and `▶ … ◀` markers; the heading also names the workspace so focus remains clear
without colour. Each workspace retains its search and selected item.

**Tiles** browses canonical animation families and individual non-family tiles.
Tiles and Locations use aligned State, Source, and Assignment columns (a compact
state symbol in narrow panes). Assignments are green when playable, amber when
empty/checking, and red for missing/invalid audio; no source or silence is grey.
The selected source cell is highlighted without hiding the assignment's colour.
`F` opens read-only animation details; member frames inherit the single family assignment.
`f` opens a compatible **Choose ambient sound** dialog, including no source. Insert marks
additional families for one atomic bulk edit; Escape clears marks. `s` assigns a
footstep surface; `D` sets the default surface; `S` opens an ordered-list draft for
that surface's shared footsteps. Add/remove/reorder controls replace comma-separated
cue-ID entry; an empty list means silence. Apply is one undoable edit and Escape
cancels the draft. These edits do not change passability.

**Locations** browses music assignments, dungeon backgrounds, code-owned dynamic
emitters, and **Naval situations** (Aboard frigate, Aboard skiff, Sailing). `s`
cycles the scope. The `f` picker offers compatible cues,
inheritance and explicit silence where applicable. Dungeon backgrounds require
radius-zero profiles; tile and emitter assignments require positive radii. Naval
onboard situations accept radius-zero ambience and Sailing accepts Music; the
picker labels them separately. New non-spatial ambience can be authored with
`Ctrl+A` and radius zero. The chosen recording and shared fade can be reviewed
through the sound settings before saving the library.

To hear onboard ambience, open **Moments** and select **Frigate Ambience** or
**Skiff Ambience**. Use `f` to choose a recording (or create/import one through
Recordings) and save. In **Locations**, select **Naval situations** with `s` to
review or change either vessel's assignment. New radius-zero Moments can also
be created with `Ctrl+A`; vessels may share an appropriate recording.
**Sailing** initially selects the Hornpipe
recording in the Music category; its `f` picker changes the selected track.
Sailing music plays once per hoist, whereas onboard ambience loops continuously.
The music manager owns one finite score player: a new music request cancels the
previous request, retiring its player before starting the replacement. A
superseded request cannot start later. Hoisting sails and entering/exiting combat
replace an older context cue; per-frame state refreshes do not issue new requests.

In these workspaces, `Ctrl+f` cycles All, Assigned, Silent/none, Needs audio, and
Surface overrides (Tiles) or Inherited (Locations). Intentionally unassigned
tiles are not errors. `o` opens the assigned Moment; `U` opens navigable configured
usages from any workspace; Escape returns from a usage jump. Usage counts describe
tile types/families and assignments, not placed objects in original maps or proof
of listening acceptance.

`Ctrl+A` opens a labelled **New ambient sound** dialog. It generates an editable
identifier from the name, offers an existing recording or an empty slot, and
exposes distance and fades as separate fields. In Tiles, `B` opens **Tile sound**:
hearing distance and relative level can each use the sound default or a per-family
override. The dialog previews those draft overrides and affects only that tile
family, including all its frames. `b` opens the shared sound defaults; `B` in
Moments also edits those shared defaults. Shared fades are shown in seconds and
stored in milliseconds. Changing a shared default affects only sources that have
not overridden that setting. The right pane explicitly labels effective values
as sound defaults or tile overrides.
Tab/Up/Down changes focus, Enter edits a field or activates Apply/Cancel,
Left/Right adjusts numeric/choice controls while they are not in text-entry mode.
Tab or Up/Down reaches the visible **Apply changes** and **Cancel** actions;
Enter activates the selected action. Footstep sequences have those same actions,
including empty sequences. No operation depends on modified Enter support.
Invalid values remain visible with their validation error. Draft previews do not
mutate the library; Apply is one undoable edit. New ambient identities need no
enum changes. `r` changes an assigned sound's recording; `i` opens its recording's
file picker. Both explicitly identify the shared recording being edited.

The Tiles table always shows independent Ambient and Footsteps columns; preview
controls never change assignment display or filter semantics. `p` previews ambience,
`P` previews a footstep, and `n` advances/previews the next surface cue. Enter
plays/restarts the current preview; Space starts, pauses or resumes without
restarting an active voice. Pausing an ambient preview freezes its fade and
resuming continues from that point. Left/Right changes temporary listener distance;
`L` exercises fade-out. Ambient preview shares the runtime attenuation formula.
Preview state is never saved. Music preview starts immediately, independently of
the game's quiet-period scheduling. The persistent Preview panel names the actual
sound and shows PLAYING/PAUSED/STOPPED and transport controls. Assignment workspaces
give the waveform's space to readable settings; trimming remains in Recordings/Moments.

The editor uses the full terminal height. The browser takes spare rows, the
waveform grows with the terminal, and a persistent Soundscape status strip shows
the assigned sounds and their playback states. The right pane switches between
Details (`d`) and Soundscape (`m`). Details wrap and list shared usages individually; focus
Details with `d` and scroll with arrows, PgUp/PgDown, or Shift+Up/Down. The active pane has a cyan
border and `ACTIVE` label. Keyboard hints stay at the bottom, with coloured
keycaps separate from action text, and change with the focused pane. Shifted
bindings are labelled explicitly, such as `Shift+s` versus `s`.
Shortcuts pack across the available width rather than occupying fixed short
rows. Primary shortcuts stay compact; `?` opens scrollable full help. The footer
contains status and hints, never editable input. Search expands into a highlighted
in-place field above the browser and leaves workspace tabs visible. Gain, trim
bounds and calibration targets use a prominent right-pane editor; longer text
and file paths use centred dialogs over a dimmed workspace. Input focus captures
typing instead of firing workspace shortcuts. Notes support multiple lines:
Enter inserts a line, Tab moves to **Apply**/**Cancel**, and Enter activates the
selected action. Numeric errors retain their field/value. Picker dialogs
use Enter to choose and Space to preview/pause. Dialog bodies scroll to the focused
control on small terminals while their title and action hints remain visible.
Below 60 × 24, editing shortcuts are paused so invisible controls cannot mutate
the library; Stop, Save, and Quit remain available with their normal draft guards.
Preview-volume controls opened over a duplicate draft are visibly foregrounded
and own input until dismissed.

One typed input-owner resolver determines command routing, foreground presentation,
and contextual hints. Recording-assignment dialogs accept only their chooser
actions; unrelated gain, crop, history, metadata, and workspace shortcuts cannot
open hidden editors behind them. Nested recording pickers restore their parent
chooser's selection and filter on return.

Inline fields, form fields, and routing searches share grapheme-aware text editing:
Left/Right moves the caret, Shift+Left/Right selects, Backspace/Delete removes a
whole grapheme or selection, Home/End moves within a line, and Alt+Left/Right
(or Alt+b/f) moves by word. Ctrl+w removes the previous word; Ctrl+u clears the
field. Notes also support Up/Down with a retained display column. Carets remain
visible when long single-line fields scroll. Bracketed paste inserts its payload
without terminal-key decorations; pasted key names never execute commands.
Notes retain authored whitespace, and path fields retain exact filesystem paths.

Entering a form field places the caret at the end without silently selecting or
replacing its value. Enter/Tab finishes the field; Escape restores the value it
had on entry. A subsequent Escape cancels the enclosing form. Routing-picker
search is a detached draft: Enter applies it and Escape restores the active filter.

Ctrl+x stops audio even while typing or during a file operation. The ordinary
`x` shortcut is available outside text entry. Ctrl+c during Save/import requests
exit after the operation settles; successful completion checks for unsaved work,
while failure keeps the editor open with the error visible. Publication is never
interrupted halfway by a keyboard quit. Labels marked **working** describe the
current authored document; the separate saved/unsaved indicator describes persistence.

Availability badges describe technical playback status, not listening approval:

| Badge | Meaning |
|---|---|
| Green `● READY` | Decoding and the selected playback window are valid |
| Yellow `○ NO FILE` | Intentionally empty recording slot |
| Red `! MISSING` | Assigned file does not exist |
| Red `! INVALID` | File is unreadable/undecodable, a directory, or has invalid trim bounds |
| Muted `… CHECKING` | File exists and background validation is pending |

`Ctrl+f` cycles All, Existing, Missing and Problems, with counts for the current
workspace. Existing means the assigned path exists, including recordings still
being checked or with decoding errors. Missing includes empty slots and absent
files; Problems includes absent assigned paths and invalid sources. Search
combines with the filter. Moments inherit availability from their assigned
recording, shown beside their name. Background checks preserve the selected ID
when it remains visible. `F5` rechecks files changed outside the editor.
Each workspace remembers its search, filter and selection. An asterisk marks
edited recordings/cues, including moments affected by a shared recording edit.
Tiles and Locations also retain independent routing filters/scopes. F5 rechecks
sources from any workspace when no editor or chooser owns input.

The browser has aligned column headings. **Recordings** shows identity, actual
acoustic family, **Cal dB** and **Uses** whenever the pane has room. Calibration
is displayed to one decimal with `A` for Auto, `M` for Manual, or `— U` when unset;
these describe gain ownership, not listening approval. Uses counts distinct
production cues referencing the recording, including all cycle memberships,
and updates immediately after an assignment edit.

**Moments** shows name, target acoustic family and assigned recording, or a clip
count with `↻` for a cycle. A yellow `!` marks a target/recording-family mismatch
if **any** assigned clip differs; it is a review prompt, not a playback error.
At compact widths the status becomes its coloured symbol, and long family names
use display abbreviations such as `mechanical`. Very narrow panes retain identity
and family and omit the extra columns; full calibration, shared uses and
assignments remain in Details. On the smallest screens the bottom list legend
is omitted to preserve a visible waveform row. Headings are included in page
height calculations, so paging still advances actual visible data rows.

Search accepts plain substrings and independent facets: `category:sfx`,
`family:magic`, `phase:resolution`, `outcome:failure`,
`role:foreground_feedback`, `emitter:object`, and `context:combat`.
In Recordings, family matches the recording itself or a cue use; other facets
match its uses. `category:unassigned` finds unused recordings. Classification
is independent of technical availability (`Ctrl+f`).

Plain search words match names, IDs, files, notes, original filenames/creators,
classification fields, keywords and the cue's `howToHear` trigger description.
Recordings also match their linked moments' intent; Moments also match metadata
on their assigned recordings. Thus `lockpick` can find a physical-unlock slot,
`grating` can find the metal footfall and its source clip, and `alchemy` can find
mixing feedback without knowing a filename. Multiple words are ANDed.

Optional `searchTerms` arrays on recordings and cues contain custom phrases.
`taxonomy.profileSearchTerms` adds reusable synonyms per semantic profile;
these supplement rather than overwrite an individual cue's keywords. Phrases
must be trimmed, nonempty and case-insensitively unique within each array.
Shift+K edits custom terms as a comma-separated list; Shift+H in Moments edits
the existing `howToHear` field, shown as **Typical trigger** in Details.
Single-use recordings show that trigger near the top; shared recordings list
their moments and each moment's trigger further down. Descriptions express
intended gameplay context, not listening approval or proof of reachability.

Pressing `/` opens a highlighted **FILTER** draft in the browser and status
row, with **EDIT FILTER** at the top and the cursor kept visible for long text.
Opening an existing filter places the caret at its end. Backspace removes the
preceding grapheme; Ctrl+U explicitly clears the field. Enter applies the draft;
Escape retains the original filter.

Structurally valid manifests with missing files can be opened and repaired.
Production loading and Save remain strict: missing/unreadable files, decode
errors and invalid bounds must be repaired before saving. The selected-item
panel explains the problem and offers `f` to choose a replacement or assignment.

| Key | Action |
|---|---|
| Tab / Shift+Tab | Next / previous workspace |
| 1 / 2 / 3 / 4 | Open Recordings / Moments / Tiles / Locations directly |
| Ctrl+f | Cycle availability filter |
| F5 | Recheck files and refresh waveform from disk |
| w / d / m | Focus waveform / show Details / open Soundscape controls; Escape returns to browser |
| a | Add the selected sound to its Soundscape slot; ambience loops, music plays once, SFX fires |
| Up/Down | Browse the current list |
| Ctrl+P / Ctrl+N | Previous / next page in the focused browser, details, file picker, variants, or mix |
| PgUp/PgDown or Shift+Up/Down | Additional paging shortcuts (terminal modifier support varies) |
| / | Edit the highlighted filter with caret navigation; Ctrl+U clears, Enter applies, Escape cancels |
| Enter or p | Play/restart the selected recording/region immediately |
| Space | Pause/resume; replay a completed preview |
| Shift+S | Switch full-source versus selected-region playback |
| l | Toggle preview looping |
| +/- | Live saved calibration (Recordings, 0.5 dB steps) or cue gain (Moments, 0.05 steps) |
| g | Enter exact calibration in dB or cue gain in amplitude |
| v | Open independent 0–10 preview category sliders and Game Mix / Reference controls |
| i / o | Set start/end at the source-relative playhead and select that marker for nudging |
| [ / ] | Select start/end marker; arrows now nudge that marker |
| Left/Right, Shift+Left/Right | Outside text entry, seek/nudge by 10 ms or 100 ms; inside fields, move/select text |
| s / e | Enter exact start/end in whole milliseconds |
| z | Zoom around the selected marker or playhead |
| t | Suggest a trim in the working copy: -40 dB threshold plus 10 ms edge padding |
| r | Reset the recording to its full source (remove the playback window) |
| f | Choose a source file, or choose a recording for the selected moment |
| n | Create an empty recording ID in Recordings (lowercase snake_case) |
| Shift+d | Duplicate the selected recording as a new metadata-only clip |
| Shift+v | Open ordered variants for an SFX moment |
| n in a cycling moment | Preview the next clip; Enter restarts the current clip |
| Shift+N | Edit the recording's note |
| Ctrl+L | Edit the selected recording's license in a dialog (Recordings/Moments) |
| Shift+K | Edit custom search keywords on the selected recording or moment |
| Shift+H | Edit the selected moment's typical-trigger sentence |
| Shift+C | Edit a recording's acoustic family or a moment's classification profile |
| Shift+A | Calculate/recalculate auto-gain for the selected recording; Enter accepts, Escape cancels |
| Shift+T | Edit the auto-gain target (uncalibrated or auto mode only) |
| u / Ctrl+R | Undo/redo authored edits |
| Ctrl+S | Validate and save the library; finish or cancel an open draft first |
| x / Ctrl+x | Stop all voices, retaining Soundscape assignments; Ctrl+x also works while typing or busy |
| q / Ctrl+C | Quit; unsaved edits require y to discard; Ctrl+C during file publication waits for completion |

Input fields accept Backspace/Delete and Ctrl+U to clear explicitly. The file picker opens folders
with Enter, goes up with Backspace, accepts a typed path with `/`, and previews
candidates with `p`. Enter imports/assigns a file; Escape cancels. The moment
recording picker uses Space for preview/pause with the moment's gain and Enter to confirm
the assignment. Previewing candidates alone never changes an assignment.

The waveform identifies the arrow target. An active preview seeks within its
playing selection and immediately displays the actual clamped source position;
with playback stopped, arrows move the cursor across the full source. `[`/`]`
select crop markers. Zoom follows the arrow target, so a numeric bound can change
while its screen column remains centered. A crop edit preserves a retained
standalone preview's playing/paused state and its source position, clamped to the new region.
Space resumes a paused edited preview; Enter explicitly restarts it.

### Game-Matched Preview Volumes

The default **Game Mix** monitoring mode uses independent Music, SFX and Ambience
sliders, initially **7, 7 and 7** respectively. These are temporary authoring
comparison settings, not changed player defaults. The header always shows them.
Press `v` to open the right-hand preview controls, Up/Down to select a row and
Left/Right or +/- to adjust it; Enter, Escape or `v` closes the panel while
retaining its temporary settings. `r` in this panel toggles Reference mode.

The editor calls the same `audio.ConvertSliderToVolume` function as the game:
7 is currently ×0.55, 10 is ×0.75 and 0 is mute. The former additional linear
listening multiplier has been removed. A Moment plays at recording calibration
× cue gain × its category slider, assuming unity request/context/distance gain.
Runtime attenuation, fades and scene suppression are not simulated. Each
cue-based comparison layer uses its own category, even when multiple categories
share a recording. Category changes update preview and layers without restarting.

Bare Recordings have no selected cue, so they apply calibration and the chosen
category but no cue gain. With **Recording bus: Auto**, a single category shared
by all linked cues wins; mixed-category recordings fall back to SFX. Unassigned
music paths use Music, unassigned ambience-family recordings use Ambience, and
other unassigned recordings use SFX. The controls allow an explicit Music/SFX/
Ambience override for these bare-recording auditions. This never overrides a
selected Moment's category; use Moments to hear the game-matched authored cue.

**Reference** plays the calibrated source at unity monitoring gain, bypassing
both category and cue gains. Temporary layer levels, mute and solo still apply.
File-picker candidates bypass the old assigned recording's calibration in both
modes. Their destination cue/category applies in Game Mix when auditioning for
a Moment. Details shows the preview route and combined gain; mix rows identify
their category or Reference mode.

These controls never dirty/save the manifest, alter calibration, or read/write
the player's configuration. Ctrl+S continues to save only authored library data.
Restarting the editor restores the comparison defaults, so choose explicit
slider settings when assessing candidate defaults for players.

### Auto-Gain And Manual Overrides

Details use cyan field labels and neutral light values, with the existing status
badge colours unchanged. Loudness details show the measurement method, target,
suggested calibration, predicted sample peak and limiting reason.

Auto-gain is offline recording calibration, not a compressor or runtime level
controller. It never rewrites audio bytes or adjusts a cue/category slider.

| Mode | Selected-region or source changes | Manual editing |
|---|---|---|
| Uncalibrated | Displays a new suggestion without changing gain | Shift+A then Enter accepts auto; +/- or g creates a manual calibration |
| Auto | Updates working-copy calibration with the selected region; trim plus gain is one undoable edit | +/- or g in Recordings locks the value |
| Manual | Keeps the exact gain across trim, source replacement, duplication and reopen; suggestion computation is paused | Only explicit Shift+A plus confirmation returns to auto |

Moving/seeking the playhead alone never changes calibration. Shift+A from Moments
affects that moment's selected shared recording; +/- and g in Moments still edit
only its cue gain. Ctrl+S saves; undo/redo retain both the gain and ownership mode.
Existing numeric gains without a mode are treated as manual, protecting prior edits.
Loading a source, refreshing availability, and undo/redo hydration do not author
calibration. A family change recomputes an existing AUTO gain in the same document
transaction as the family edit. AUTO confirmation is bound to its recording and
document revision; a stale suggestion cannot be applied to another target.
`calibrationMode` is `auto` or `manual` and requires `calibrationGainDb`.
Optional `calibrationTargetDb` overrides the family default and is restricted to
`[-60,-6]` dB. Invalid or silent selections have no suggestion.

Analysis is streamed during background source loading and cached in 10 ms energy
buckets. Region boundaries weight overlapping buckets; boundary peaks are
conservative. Sustained audio uses K weighting at 44.1 kHz, 400 ms blocks with
100 ms hops, a -70 LUFS absolute gate and a -10 LU relative gate. Short effects
use the 95th percentile of active 50 ms stereo RMS windows, excluding windows
below -70 dBFS or more than 40 dB below the strongest window. Sustained selections
shorter than 400 ms fall back to the short-effect estimator. Values are explicitly
labelled **estimated**; this is not a certified LUFS or true-peak meter.

The agreed initial authoring targets are -23 for score/instrument, -26 for
ambience/nature/fire, and -20 for other families, in the units of the displayed
estimator. They remain editable conservative starting points, not
listening-approved mix levels. Suggestions cap
the calibrated sample peak at -3 dBFS and gain at `[-48,+12]` dB; the UI identifies
when headroom or boost limits prevent reaching the target. Individual source
headroom does not guarantee a multi-voice mix cannot clip. Make a quieter footstep
using the moment's cue gain after calibration, not by changing the category slider.

Auto calibration may be saved as a provisional production baseline before
contextual listening. Accepting this method and its starting targets does not
accept every recording, assignment, loop, trigger, or combined mix. Neither a
persisted gain nor Manual mode constitutes listening approval; retain that
evidence separately with the recording identity and listening context.

The waveform takes roughly a third of the terminal height (up to 24 rows),
while preserving browser space and compacting the mix when necessary on a small
terminal. It uses stereo peak envelopes in 10 ms buckets, preserving
transients in either channel rather than averaging the channels together. The
vertical display uses a labelled -72 to 0 dB scale and eighth-cell bar glyphs,
so quiet sounds are visible. Nonzero signals below the display floor receive a
minimal bar; digital zero remains a flat baseline. This visual scaling changes
neither playback gain, the original peak measurements, nor the -40 dB silence
suggestion threshold. The selection and markers are highlighted, excluded
samples dimmed, and the playhead
uses source-file milliseconds even for trimmed or looping playback. Waveform
zoom retains a whole-file overview, with the selected region and zoom viewport
highlighted. Waveform
decoding is cancellable background work; stale results cannot replace a newer
selection. Silence suggestions remain undoable edits, not automatic saves or
proof that a quiet onset/tail should be removed. Cuts have no implicit fades or
zero-crossing adjustment; audition for clicks before saving.

### Shared-File Clips And Ordered Variants

`Shift+d` opens a small new-clip draft with the next unused numeric suffix,
preserving zero padding (`_009` becomes `_010`). The duplicate references the
same physical file and copies provenance/calibration, but gets its own window
and a fresh note. If the selected clip ends before the source does, the draft
starts there and ends at the source end. Otherwise it starts with the same
region. Up/Down chooses a field and Enter edits it, or `s`/`e` opens a bound directly.
Left/Right edits the caret while a bound field is open. Outside text entry,
`[`/`]` and 10/100 ms arrow nudges work on duplicate markers. `p` previews the draft; Space pauses
or resumes; `i`/`o` mark and select a boundary at the current source playhead. `c` creates it;
Escape cancels. Both bounds validate together, so moving a start past the old
end while editing does not mutate the original clip.

When an SFX moment is selected, or the recording has one unambiguous SFX use,
the draft offers an unchecked `a` toggle to append the new clip to that moment.
Creation and optional membership are one undoable metadata edit. No audio bytes
are copied or rewritten, and source-file duplication is not required.

In Moments, `Shift+v` opens ordered SFX variants. Up/Down in the browser chooses a
clip; Enter previews it; `a` opens a recording picker to append; Delete removes a
member, while `d` retains its Details meaning. Ctrl+Up/Down reorders;
Shift+Up/Down pages. Waveform focus keeps Up/Down from changing the editing target.
Escape first leaves focused marker/details editing, then returns from the list.
One remaining member becomes an ordinary
single assignment. At least one member must remain. Outside this list, `n`
previews the next member of a cycling moment; Enter restarts the current one. Editor preview position
is independent of the running game's cursor. Comparison mix layers audition
the selected clip; their repeat interval does not change production selection.

Single cues keep `recordingId`. An SFX cycle instead uses an ordered
`recordingIds` array and `selection: "cycle"`. These forms are mutually
exclusive; every member must be a distinct known recording with an assigned
file. This first variant policy is for SFX only, not music/ambient scheduling.
The SFX player selects the next member under its existing playback lock and
advances that cue's cursor only after starting a player. Rejected or dropped
requests do not advance it, looping a clip does not select another clip, and
other cues do not affect the sequence. A new audio player system starts each
cycle at the first member. This consumes no gameplay randomness or time.

### Soundscape Audition

The optional right-hand Soundscape pane presents four named slots using the
existing independent audition voices: **Ambient 1**, **Ambient 2**, **Music**,
and **SFX**. `m` opens/focuses it even when empty. Escape or `m` returns keyboard
focus to the browser while leaving the pane visible; `d` shows Details instead.
The compact status strip always remains visible and identifies assigned sounds
as playing (`▶`), paused (`Ⅱ`), finished (`✓`), stopped (`■`), muted (`○`), or
empty (`—`). A separate `P` entry identifies any additional standalone preview.

To audition two ambient beds, music, and effects together:

1. In **Moments**, select an ambient cue such as Nearby Fire and press `a`.
   It starts looping in the first empty ambient slot.
2. Select another ambient cue and press `a` to fill the second slot. When both
   are assigned, the pane asks which to replace: `1`/`2` or arrows select a slot,
   Enter confirms, Escape cancels. Existing beds continue during the prompt.
3. Select a music Moment and press `a`. It occupies the single Music slot and
   plays **once**, regardless of the standalone preview loop toggle. Another
   score replaces that slot. In Soundscape, `l` enables an explicitly labelled
   audition-only loop; it does not change the game's finite music policy.
4. Browse SFX Moments and press Enter to fire over the backgrounds. The SFX slot
   keeps playing while browsing another cue; the next trigger replaces that
   voice rather than accumulating voices. Cycle selection still follows the
   editor's normal next-clip cursor. Use `v` to adjust category levels live.

Within Soundscape, arrows select slots, Enter restarts, Space pauses/resumes,
`u` mutes, `s` solos, +/- adjusts temporary layer gain, and Delete/Backspace
removes a slot. `l` toggles once/loop for beds and music, or once/repeat for SFX.
Repeat retriggers every 500 ms and never accumulates voices. Slot mute, solo and
temporary gain survive replacing its source. Failed replacements preserve the
existing playing slot. Stop All (`x`) closes voices and cancels pending/repeat
playback but retains assignments; Enter restarts an individual stopped slot.

Browsing and workspace changes preserve the soundscape. File import, clip-list
edits, duplication, and document undo/redo clear it to avoid stale assignments;
quitting closes every voice. Adding a sound closes the standalone preview to
avoid doubling an already auditioned recording. No soundscape assignments are
saved as production cue assignments or persistent sessions.

Preview mode/category sliders, layer gain, mute/solo, and repeat intervals are temporary and
never written to the manifest or game preferences. Calibration and cue gain are
authored fields, constrained by the production validator. Preview uses the same
decoder, calibration formula and bounded stream as production, without invoking
gameplay or consuming game time/RNG. Final contextual acceptance still happens
in the game.

### Save And Import Contract

`LibraryEditor` keeps one working document and bounded undo/redo history.
Ctrl+S validates it, checks for external manifest edits, and atomically replaces
the file using a flushed same-directory temporary file. Unchanged entries retain
their exact bytes; deleting a routing override rewrites only its containing map,
preserving other sections. A no-op Save leaves the entire file untouched. There is no
intermediate tuning-export format or automatic source-audio rewrite.

Save never silently commits an unfinished field, form, or duplicate: finish or
cancel that draft first. Duplicate drafts participate in quit confirmation even
when the authored document is clean. Ctrl+S from quit confirmation saves an
already-applied document and returns to editing rather than discarding it.

File operations carry their starting document revision and operation identity.
Stale or replayed completions cannot replace a newer working copy. Waveform
loads bind a generation and recording identity; editing also verifies the loaded
source path. Background availability scans do not retarget active editors.
Import failure reloads the original assigned source while preserving the error,
instead of leaving an auditioned candidate under the old recording identity.

Repository audio files are referenced by relative path. Outside files are copied
byte-for-byte to `sounds/imported/` (or `music/imported/` when replacing a music
recording), using recording ID and content hash in the name to avoid overwriting
another file. Unknown provenance stays literal `unknown`. Undo restores the
assignment but deliberately leaves imported files on disk for Git review.

The game still embeds the production manifest/assets at build time. Save affects
editor preview immediately; rebuild/restart the game to use those edits. Review
shared-recording usages before trimming or calibrating, and review the Git diff
before committing authored changes.

## In-Game Configuration

The shared `ConfigurationDialog` owns detached Sound, Display, and Gameplay
settings. Its Sound tab owns preview playback; Apply persists the complete
transaction and Cancel reverts previewed audio. The scene callback only removes
the exact modal and restores its parent scene's audio policy. The in-game route
opens on Sound above the still-paused Escape menu. The title-menu route pauses
title audio, opens on Display, and restores title music after dismissal. Tab
changes section; Up/Down selects an entry; Left/Right changes the selected
value; Enter applies; and Escape reverts. The card retains stable dimensions
while sections change and introduces no pointer input. Display
apply/confirmation behavior is documented in
[Modal Dialog System](../ui/MODAL_DIALOG_SYSTEM.md).

## Failure Policy

Recoverable audio initialization and playback errors are returned and logged by
the `GameScene` adapter. Initial load or decode failure for a required embedded
sound or music asset panics because production data is expected to be complete.
Fresh-stream re-decode failure degrades to silence; music logs that fallback,
while sound effects currently substitute a silent stream directly. Disabling
sound is a supported silent mode. Do not add nil checks around required
`SystemCallbacks.Audio` functions; their constructors validate the contract.

These returned-error paths do not contain every backend failure. In Ebitengine
2.10, device initialization and asynchronous player errors can surface through
the engine update hook after manager construction and can terminate `RunGame`.
Successful construction or a nil immediate playback error is not device-readiness
or fail-soft acceptance.

An absent non-null manifest file assignment is invalid embedded application data,
not an intentional placeholder or ordinary runtime degradation.

### Authoring The Library

The manifest is hand-authored, tracked source. Report generation never updates
it. SFX vocabulary comes from the typed identity table and music from known track
IDs. Ambient vocabulary is authored in `routing.ambient` and the matching cue
entries. An arbitrary SFX key cannot create a gameplay event; a new ambient cue
can be assigned to tile families or dungeon backgrounds without a Go identifier.

Example entries (shown as a fragment, not a complete replacement manifest):

```json
{
  "schemaVersion": 3,
  "recordings": {
    "magic_failure_fizzle_01": {
      "file": null,
      "provenance": {
        "originalFilename": "unknown",
        "sourceType": "unknown",
        "sourceUrl": "unknown",
        "sourceCreator": "unknown",
        "license": "unknown",
        "importedFrom": "unknown",
        "processing": []
      }
    }
  },
  "cues": {
    "sfx/SoundSpellFailure": {
      "name": "Spell Failure",
      "recordingId": "magic_failure_fizzle_01",
      "volumeModifier": 0.7,
      "howToHear": "Reach a spell failure/no-effect branch; cancelling may be silent."
    }
  },
  "taxonomy": {
    "recordingFamilies": {"magic": ["magic_failure_fizzle_01"]},
    "cueProfiles": {"magic_failure": ["sfx/SoundSpellFailure"]},
    "profiles": {
      "magic_failure": {
        "targetAcousticFamily": "magic", "phase": "resolution",
        "outcome": "failure", "mixRole": "foreground_feedback",
        "emitters": ["player"], "contexts": ["spellcasting"]
      }
    }
  }
}
```

For an existing empty slot, add the selected recording under `assets/sounds/`
(subdirectories are supported) and replace only `file: null` with its exact path
relative to `assets/`, such as `sounds/magic/magic_failure_fizzle_01.wav`.
Rebuild the game or run the catalog command to validate the assignment. Changing
the manifest is not live reload. Keep experiments outside embedded assets.

Recording IDs use lowercase letters, digits, and underscores; descriptive
`snake_case` with numbered variants is recommended. File moves change `file`,
not the recording ID. Replacing bytes changes SHA-256 and invalidates any prior
listening evidence for those bytes. A null file is deliberate; an omitted `file`,
empty string, guessed extension, or nonexistent non-null path is an error.
Every recording, including an empty slot, has structured `provenance`. Required
text fields use the literal `unknown` when information is unavailable; do not
leave them blank or infer a creator/license from a filename. `processing` is an
array of known steps, empty when none are documented. A recording's optional
`note` is for general review/playback observations, separate from provenance.
`playbackWindow` is an optional bounded excerpt for SFX, music, and ambience.
Bounds are source-relative whole milliseconds, start inclusive and end exclusive,
validated against decoded duration rather than an arbitrary maximum timestamp.
Optional `calibrationGainDb` belongs to the recording and therefore
affects all its usages. It may hold a provisional Auto baseline or a locked Manual
adjustment; neither implies listening acceptance. A missing value stays `null`
in the catalog, not an approved zero adjustment. See
[Auto-Gain And Manual Overrides](#auto-gain-and-manual-overrides) for the agreed
starting targets and ownership rules. The manifest's `volumeModifier` preserves
the separate cue/context coefficient.

To replace only Push while it shares `healing_01`, register a new recording such
as `object_push_wood_01` and change only `sfx/SoundPushObject.recordingId`. Do not
overwrite the shared healing recording unless all its usages should change.

Unassigned ambient cues can have slots too; filling one does not create
tile/source integration until a routing target selects it. Generic declarations or static references are not a
promise of an audible gameplay route. The guide separates this evidence from
recording availability and gives intended, not listening-certified, instructions.

## Source And Tests

### Asset And Moment Catalog

`internal/audio/catalog.go` builds a detached inventory directly from production
SFX metadata, music selection, ambient definitions, and embedded recordings.
It creates no players or audio context. The ambient manager and catalog share
the same definition functions, so the inventory does not own a second set of
file assignments, gains, distances, or fades.

From the repository root, with `bin/` available:

```bash
go run ./cmd/audio-catalog
# Optional whole-recording sample-peak and RMS measurements:
go run ./cmd/audio-catalog -measure
```

The command writes `bin/audio-catalog.json` and
`bin/audio-generation-guide.md`. `-source`, `-output`, and `-guide` override the
checkout root and output paths. Output parents must already exist. These are
regenerable review artifacts, not committed backlog files, saved mixer scenarios,
or accepted-tuning imports. No original Ultima V data is needed. Run against the
same checkout that built the tool; it combines embedded recordings/definitions
with checkout source evidence.

The JSON has two linked views:

- **Assets:** exact embedded path, SHA-256 of encoded bytes, format, decoded
  duration, optional sample peak/RMS, provenance, calibration/listening/loop
  review, and inverse moment usage. Empty registered slots and unregistered files
  remain visible; the latter require an explicit recording ID before assignment.
- **Moments:** semantic cue or music/location/ambient context, actual assignment,
  configuration coefficient, mix role/family, lifecycle/overlap description,
  source evidence, how-to-hear instructions, and draft generation brief. Every
  declared SFX has a cue entry; explicitly proposed coverage candidates remain
  separate from the production cue vocabulary.

Identity rules:

| Kind | Identity | Change rule |
|---|---|---|
| Recording | Explicit manifest key, e.g. `magic_failure_fizzle_01` | Independent of path; replacing bytes changes SHA-256, not identity. |
| Semantic SFX | `sfx/SoundPushObject`, etc. | Symbol name, never display label or `iota` ordinal; renames require identity review. |
| Music | `music/track/<track-id>`, `music/location/<Location-symbol>`, `music/context/<context>` | Derived from production track identities and location/context mapping. |
| Ambience | `ambient/AmbientForest`, etc. | Stable sound-type symbol; static sprite bindings and dynamic-source references remain evidence. |
| Coverage candidate | `proposed/<category>/<semantic-name>` | A proposal, not a live trigger or approved requirement to generate material. |

Review dimensions deliberately remain independent:

- `availability`: `present`, `missing`, or `not_assigned` for proposed moments.
- `assignmentReview`: `unreviewed`, `placeholder`, or `incorrect` initially.
- `listeningReview` and recording `loopReview`: initially `unreviewed`; merely
  decoding a file or finding a caller cannot approve it.
- `triggerCoverage`: `not_scanned`, `playback_request_found`,
  `source_reference_found`, `no_static_reference_found`, `declared_only`, or
  `proposed_not_integrated`.

`AttachSourceEvidence` scans non-test Go function bodies under `cmd/ultimav` and
`internal`, excluding catalog implementation from reference evidence. It ignores
comments, string literals, and top-level declarations. It records file, line,
function, and symbol. Literal cue arguments to known playback/builder methods
receive a stronger `playback_request_found` classification; it does **not**
resolve indirect control flow, prove a
branch is reachable, or certify actual playback. Shared dispatch evidence for a
music location is not a walkthrough of that location. A source-tree fingerprint
covers scanned non-test Go files, including catalog code. A separate library
fingerprint covers the embedded manifest. Tests catch missing cue coverage,
invalid authored mappings, dangling links, and accidental acceptance claims.

`library.json` owns provenance as well as assignments. The former text credit
list has been migrated into individual recording entries; Git retains its
history. There is no filename-prefix credit lookup. Source URLs alone do not
establish license terms. `catalog_notes.go` owns initial listening observations
and draft briefs, not provenance or playback assignments. Recording-
specific diagnoses are bound to their old recording IDs and are not applied to
new candidate assignments. Review notes still require human maintenance; report
generation cannot accept recordings. GitHub Issues remain
the sole work tracker; the generated guide is an inventory and production aid.
The Markdown is table-first: grouped SFX, ambience, music tracks, location/context
bindings, proposed moments, recordings/slots, and recording provenance. The
provenance table is generated directly from the JSON and preserves source credit,
original filenames, import lineage, and conversion/editing history independently.
Full generation briefs follow
the tables through row links; mixing-only issues such as bard/score priority do
not automatically receive irrelevant new-recording prompts.

### Provenance And Filename Conventions

In **Recordings** or **Moments**, `Ctrl+L` opens a centred **Edit License** dialog
for the selected recording (including the selected clip in a Moment's variant
cycle). The dialog names the recording and shows its shared Moment count. Edit
the prefilled text, use Ctrl+U to clear, Enter to apply, or Escape to cancel.
Blank values are rejected without closing the dialog; use literal `unknown`
when the license is undocumented. Apply changes only `provenance.license` as one
undoable edit; Ctrl+S saves it to the manifest. Empty or missing audio files can
have their license edited without decoding or changing their file, trim, gains,
or other provenance. This records the supplied terms; it does not verify them.

Production filenames match their recording IDs plus the existing format extension.
SFX are organized under `sounds/ambience/`, `sounds/effects/`,
`sounds/footsteps/`, `sounds/instruments/`, `sounds/interactions/`, and
`sounds/magic/`; music lives under `music/`. These paths are relative to
`internal/audio/assets/`. Name a recording for its content, not an incorrect
current usage: leaf footsteps remain leaf footsteps even when assigned to wood.

All provenance is authored in each recording's `provenance` object:

| Field | Meaning |
|---|---|
| `originalFilename` | Filename before production cleanup, or `unknown`. It is not necessarily the remote source's original filename. |
| `sourceType` | How the material originated, such as `download`, `midi_conversion`, or `unknown`. |
| `sourceUrl` | Original download/source reference, or `unknown`. |
| `sourceCreator` | Known source creator/handle, or `unknown`; conversion alone does not establish authorship. |
| `license` | Known license terms, or `unknown`. |
| `importedFrom` | Repository/import lineage when known; distinct from the original source URL. |
| `processing` | Documented steps with `description`, `tool`, and `performedBy`; `[]` means no steps documented, not proof of no editing. |

All text fields must be nonempty; use the literal `unknown` for missing facts.
An absent/null provenance object or processing array is invalid. Unknown values
remain explicit in generated JSON and the readable provenance table. Unregistered
files have unknown provenance until a recording entry is authored.

The existing music tracks were converted by Brad Hannah in Audacity from MIDI
files found online. Their provenance records this confirmed processing history
while leaving the MIDI source URL, source creator, and license as `unknown`.
Their `originalFilename` preserves the pre-cleanup MP3 name, not a guessed MIDI
filename. Prompt-shaped SFX filenames are not evidence of a generation platform;
those source details remain unknown until supplied.

When moving or renaming an asset, keep its recording ID stable and update `file`.
Preserve the original filename and provenance, and compare byte hashes if the
operation is intended to be a pure rename. The catalog does not infer credit
from the new filename, directory, or cue assignment.

### Player-Facing Attribution

Main-menu Acknowledgments uses `RecordingCredits()` to read the validated
embedded manifest without building the diagnostic catalog, decoding/hash-scanning
recordings, constructing players, or requiring sound to be enabled. It returns
one detached credit per non-null recording, in stable recording-ID order, with
cloned processing slices. Cue reuse does not multiply credits; empty future
slots are omitted. Improvements to the manifest flow into credits on rebuild.

The document separates music and effects, identifies recordings by retained
original filename (falling back to the current basename when unknown), and
groups only identical source type/creator/URL/license/processing combinations.
Each distinct work name and every distinct supplied license/processing statement
survives. Freeform `license` strings can contain the only creator/work/source
credit; they are rendered verbatim, without parsing them into guessed metadata.
Source URLs are plain text. Internal import URLs, hashes, cue IDs and calibration
are not displayed as creator attribution. Missing attribution is labelled as
not recorded, and processing/conversion is always separate from authorship.

The bundled-asset reconciliation retains three unassigned legacy recordings:
`legacy_water_01` preserves the Water/danielpodlovics supplied credit already
recorded on `coin_splash_01`; its MP3 conversion history remains unknown.
`legacy_healing_01` / `legacy_healing_02` preserve the original manifest's
`HealingSound.wav` / `HealingSound2.wav` identities and recorded import lineage,
with authorship and license still unknown. They do not inherit the
credits of the newer replacements assigned to `healing_01` / `healing_02`.
Registering these retained files does not assign them to playback cues. The
retired `Credit.txt`'s nine source links remain covered by the existing manifest
and provenance tests; a metadata-only inventory regression checks all bundled
audio has a recording entry.

### Calibration And Recording Acceptance

Keep three decisions separate:

1. **Recording calibration:** choose consistent perceived strength within a
   family/variant set, starting from the agreed provisional Auto baseline where
   useful. Preserve the original recording and provenance. An unset
   `calibrationGainDb` is JSON `null`, not an approved zero adjustment. A numeric
   gain records treatment, not human listening acceptance.
2. **Context treatment:** decide how that calibrated recording should sit in a
   specific gameplay moment. Shared healing audio and its quieter per-character
   tick may have different context gains without duplicating the recording.
3. **User preferences:** record music/SFX/ambient slider settings as listening
   context. They must not silently become authored production defaults.

Use decibels for authored relative adjustments: amplitude multiplier
`10^(dB/20)`, and `20*log10(amplitude)` for nonzero amplitude. Mute is a separate
zero-amplitude state, not a finite dB value. The existing `configuredGain` is
**not** a calibrated level or final output amplitude: it reports the cue's
authored contextual amplitude. The catalog reports recording calibration
separately, and its gain semantics describe continuous runtime composition.
Do not infer accepted mix targets from existing coefficients.

The optional measurement pass streams stereo signed 16-bit decoded PCM at
44.1 kHz. Duration divides decoded byte length by both sample rate and four bytes
per stereo frame. Sample peak and whole-recording RMS are diagnostic measures,
not LUFS, true peak, or a perceptual score. A null measured level means silence
when `pcmMeasured` is true; otherwise null means not measured. Long silence can
depress RMS, and a peak-normalized
impact can still sound far louder than an ambient bed. The editor's selected-region
estimators and agreed family starting targets are described under
[Auto-Gain And Manual Overrides](#auto-gain-and-manual-overrides). Use suitable
audio tools when certified integrated/short-term loudness or true-peak measurements
are needed; the starting targets are not a universal LUFS or final mix standard.

Reference listening procedure:

1. Record output device, category sliders, OS/hardware listening volume, and
   source revision/fingerprint. Keep this setup fixed for A/B comparisons.
2. Choose a provisionally useful reference per family: isolated grass footsteps,
   ordinary door interaction, healing, forest/stream ambience, and an exploration
   track are starting candidates, **not** automatically accepted standards.
3. Compare variants in isolation, then over the actual environmental mix. Frequent
   footsteps/interactions should stay restrained; special-event scale must not
   turn routine success into a loud fanfare.
4. Check quiet indoor interaction, sustained grate walking, forest/stream with
   score, tavern bard versus score, magical sources, spell layering, and extended
   exploration. Listen on both headphones and speakers at comfortable levels.
5. Retain exact authored dB/context values and observations. Evaluate the agreed
   provisional targets in context and record listening approval separately from
   automatic calibration or a saved Manual adjustment. Mark missing
   triggers/metadata separately from missing material. The catalog does not imply
   every declared cue is audible.

For ElevenLabs or other sourcing, use the generated guide to choose a small
batch. Start with clearly incorrect physical cues such as push, unlock, and glass
break. Each brief should specify one event versus a sequence, duration/tail,
material or magic family, variants, loop requirements, and exclusions such as
voice/music. Keep generation prompt/source details, candidate files, and editing
notes with the delivered material. Trim leading silence, inspect tails/seams,
check clipping, and audition repetition before integrating. A generated file is
a candidate, not a passed listening check. Absent metadata/trigger coverage still
requires code integration after the recording is supplied.

- `internal/audio/audio_manager.go`: facade and public operations
- `internal/audio/sound_effects.go`: sound instances, assets, fades, and cleanup
- `internal/audio/background_music.go`: music assets and playback
- `internal/audio/music_manager.go`: location/context selection
- `internal/audio/ambient_sounds.go`: proximity ambience
- `internal/audio/configuration.go`: volume conversion and validation
- `internal/audio/library.go`: strict authored-manifest validation and resolution
- `internal/audio/assets/library.json`: editable recordings, cue assignments, and audition instructions
- `internal/audio/catalog*.go`: derived inventory, source evidence, annotations, and report
- `cmd/audio-editor/main.go`: standalone Bubble Tea editor and read-only/device probes
- `internal/audio/editorui/`: recording/moment browser, waveform and keyboard editing
- `internal/audio/library_editor*.go`: disk-backed document, import, undo/redo and atomic Save
- `internal/audio/recording_stream.go`: common seekable bounded playback region
- `internal/audio/recording_analysis.go`: disk-source loading and stereo peak envelope
- `internal/audio/audition.go`: independent seekable editor voice
- `cmd/audio-catalog/main.go`: offline JSON/recording-guide export
- `internal/audio/sound_excerpt.go`: bounded metal-footfall selection
- `internal/game_state/sound_builder.go`: callback builders
- `cmd/ultimav/gamescene_audio.go`: `GameScene` adapter
- `internal/audio/*_test.go`: focused audio tests
- `internal/game_state/sound_builder_test.go`: request construction and single-dispatch tests

Use [Workbench Logging Control](LOGGING_SYSTEM.md#runtime-control) with the audio category for runtime diagnostics.
