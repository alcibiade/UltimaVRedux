# Visual Effects System

**Purpose**: Describe the visual-effect contracts and rendering techniques that exist in the current codebase. This document is the authority for masking, dissolve effects, screen flash, screen shake, inversion, projectiles, and transient impact effects.

## Architecture

Visual effects cross three layers:

| Layer | Responsibility | Primary code |
|---|---|---|
| Core gameplay | Chooses effects and sequences them without importing Ebitengine | `internal/game_state/` |
| Presentation contract | Exposes required rendering mutations through `VisualCallbacks` | `internal/game_state/system_callbacks.go` |
| Ebitengine adapter | Stores effect state and renders it in `GameScene.Draw` | `cmd/ultimav/gamescene_visual_effects.go`, `cmd/ultimav/gamescene_draw_effects.go` |

`NewVisualCallbacks` validates its core callbacks. Code that receives initialized `SystemCallbacks` should call required visual callbacks directly rather than treating them as optional. Projectile callbacks are attached by `GameScene`, and `ShowCombatImpact` is validated by `NewSystemCallbacks`.

The current visual/workflow callback surface is:

```go
type VisualCallbacks struct {
    SetWandererDissolveLevel         func(float64)
    InitializeWandererDissolveEffect func()
    CleanupWandererDissolveEffect    func()
    SetAvatarDissolveLevel           func(float64)
    InitializeAvatarDissolveEffect   func()
    CleanupAvatarDissolveEffect      func()
    SetMapFadeLevel                  func(float64)
    SetScreenFlashColor              func(r, g, b float64)
    SetScreenFlashIntensity          func(float64)
    SetScreenShakeOffset             func(x, y float64)
    SetScreenInvert                  func(intensity float64, durationMs int)
    BlockAllInput                    func(bool)
    IsAnyKeyPressed                  func() bool
    GetPressedKey                    func() *semantic_input.Key
    ResetTeleportationSourceRenderer func()
    StabilizeTeleportationDestinationRenderer func(game_state.MoongateDestination)
    StartProjectile                  func(from, to references.Position, kind references.MissileType) uint64
    UpdateProjectileProgress         func(float32)
    ClearProjectile                  func()
    PlayKapowAtPosition              func(references.Position)
    ShowCombatImpact                 func(game_state.CombatAttackFeedback)
}
```

Flash and shake callbacks only set current render state. Timing and cleanup belong to the caller's event sequence.

## Event Ownership

Use an event-queue workflow when the sequence mutates gameplay, owns input, or must be awaited. Use feedback for presentation-only effects that must not block or mutate gameplay. `PartyStateBuilder` belongs inside a workflow when it applies damage or status changes.

Every workflow must set priority before `Sequential()`, start with `First()`, and use `Finally()` for state that must be reset. Feedback builders also start with `First()` and use `Finally()` for cleanup.

```go
manager.TriggerFeedback("impact", event_queue.PriorityNormal,
    func(sequence *event_queue.SequentialEventBuilder) {
        sequence.First(func() {
            callbacks.Visual.SetScreenFlashColor(1, 0, 0)
        }).Animate(200, func(progress float32) {
            wave := 1 - math.Abs(float64(progress)*2-1)
            dampen := 1 - float64(progress)
            callbacks.Visual.SetScreenFlashIntensity(wave * 0.7)
            callbacks.Visual.SetScreenShakeOffset(0, math.Sin(float64(progress)*math.Pi*8)*6*dampen)
        }).Finally(func() {
            callbacks.Visual.SetScreenFlashIntensity(0)
            callbacks.Visual.SetScreenShakeOffset(0, 0)
        })
    },
)
```

Do not start an independent workflow inside another workflow callback when the parent must wait for it. Add reusable steps with `ThenSequence`, or use `AwaitFeedback` when a workflow must wait for serialized presentation feedback.

## Transient Status Icons

`UICallbacks.ShowTransientIcon` accepts a semantic `TransientIconKind`; core
gameplay never supplies sprites, colours, or durations. `GameState` publishes
only after an authoritative transaction observes the required state change.
Multi-character transactions aggregate before publishing, and excluded
low-level setters, hydration, debug edits, recurring poison damage,
resurrection, and Regeneration Ring ticks remain silent.

| Kind | Presentation | Duration |
|---|---|---:|
| Eating | Food, white | 1000 ms |
| Starving | Food, light red | 500 ms |
| Healing | Healer sign, light green | 1500 ms |
| Poison onset | Poison field, green | 1500 ms |
| Poison cure | Poison field, light cyan | 1500 ms |
| Level up | Star pattern, yellow | 3000 ms |
| Karma increase | Shrine, light green | 1500 ms |
| Karma decrease | Shrine, light red | 1500 ms |

`ActiveEffectsOverlay` owns the fixed lookup and records deadlines from
`GameState.ElapsedMs`. Registration first prunes expired records. Re-registering
the same kind preserves its stack position and refreshes its deadline; distinct
kinds append in registration order. At most five records are retained, with the
oldest evicted before a sixth distinct kind is added. Opposite poison and karma
kinds are distinct and may coexist.

The overlay is presentation-only: it creates no workflow, owns no input,
consumes no RNG, and mutates no gameplay or save state. Scene replacement and
disposal clear all records idempotently; ordinary same-scene map transitions do
not.

## Screen Flash

`SetScreenFlashColor` selects RGB components in the `0.0` to `1.0` range. `SetScreenFlashIntensity` supplies the current alpha. `drawScreenFlashEffect` draws the color over the map area only; the right-side UI and dialogs remain unaffected. Flash rendering is skipped in a dungeon while no light source is active.

The caller owns the intensity curve. Current effects commonly use a triangle or sine wave and explicitly restore intensity to zero. `PartyStateBuilder` provides `FlashRed`, `FlashWater`, `FlashGray`, `FlashLongGray`, `FlashGreen`, `FlashOrange`, `FlashPurple`, `FlashWhite`, and `FlashInverted` presets.

## Screen Shake

Screen shake is a camera offset, not a self-timed subsystem:

```go
callbacks.Visual.SetScreenShakeOffset(xPixels, yPixels)
```

The event sequence computes each offset and must restore `(0, 0)`. `PartyStateBuilder.ShakeModerate` currently uses 200 ms and a 6-pixel range; `ShakeStrong` uses 300 ms and a 10-pixel range. Its deterministic oscillation is damped toward zero without consuming gameplay RNG.

The offset is applied to world presentation, including map rendering, world entities, projectiles, kapow effects, and combat impact effects. Static UI, the combat HUD, output panes, and modal dialogs are drawn without the camera offset.

Audio is coordinated explicitly in the same workflow or feedback sequence:

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
        callbacks.Visual.SetScreenShakeOffset(0, 0)
        callbacks.Audio.Stop(game_state.SoundEarthquake).FadeOut(500).Now()
    }).
    Start()
```

## Screen Inversion

`SetScreenInvert(intensity, durationMs)` drives a Kage shader over the composed screen. A positive duration uses `GameScene` elapsed time for automatic expiration. A duration of zero means an external event animation owns the intensity and must restore it to zero. `PartyStateBuilder.FlashInverted` uses the externally controlled form. Inversion shares the screen-flash guard and is skipped in a dungeon while no light source is active.

## Pixel Dissolve And Masking

`internal/visual.PixelDissolveEffect` progressively changes alpha values in a shuffled mask. It supports:

- `DissolveModeReveal`: transparent to opaque
- `DissolveModeHide`: opaque to transparent
- forward and reverse progress updates in the `0.0` to `1.0` range
- a persistent CPU pixel buffer and Ebitengine mask image
- an optional persistent scaled mask rendered with nearest-neighbor filtering
- deterministic shuffle order from a presentation-owned source

Create the effect once, update progress, and reuse its mask:

```go
maskWidth, maskHeight := visual.CalculateMaskSizeForDialog(width, height)
effect := visual.NewPixelDissolveEffect(
    width, height,
    maskWidth, maskHeight,
    visual.DissolveModeReveal,
    presentationSource,
)

effect.UpdateProgress(progress)

maskedBuffer.Clear()
maskedBuffer.DrawImage(contentBuffer, nil)
maskOptions := &ebiten.DrawImageOptions{}
maskOptions.Blend = ebiten.BlendDestinationIn
maskedBuffer.DrawImage(effect.GetMask(), maskOptions)
```

`BlendDestinationIn` is required here because the destination already contains the sprite or dialog colors. The mask's alpha controls destination visibility while preserving those colors. Using `BlendSourceIn` in this order selects the mask's colors instead and produces a silhouette rather than a color-preserving dissolve.

`CalculateMaskSizeForDialog` derives mask dimensions from the display scale and clamps each dimension to the implementation's safety range. Do not duplicate that calculation at call sites.

`MessageDialog` uses this path with persistent content and masked buffers. Tile dissolves use a one-to-one sprite mask through `DissolveEffectState`. Presentation effects must construct their own stable source; they must never receive the persisted gameplay RNG source.

## Terrain Visibility Dithering

Ordinary top-down small and large maps use a world-anchored visibility-edge
mask, including basement and underworld floors. This is a presentation-only
bonus-terrain treatment: it deliberately exposes a narrow portion of actual
neighboring terrain without changing gameplay visibility, light radii, source
collection, interaction, or entity visibility. Outdoor terrain clues can be
revealed. Indoor rooms and structural boundary tiles retain whole-tile coverage:
visible tiles stay intact, and hidden tiles stay black. There is no indoor
inward fade, rounded erosion, or continuous edge darkening.

- `internal/sprites/quarter_mask.go` contains reusable, standard-library-only
  geometry and ordered-threshold calculations. Four quarter pieces assemble
  each tile's coverage; reflections reuse the small piece set. Geometry is
  independent of `GameState`, sprite identities, clocks, and Ebitengine.
- `internal/sprites/dither_mask_image.go` owns a reusable CPU buffer and
  standalone GPU texture. Transparent mask pixels keep the composed world;
  opaque black pixels reject it. Standalone storage avoids atlas-dependent
  nearest-sampling differences at fractional camera offsets.
- `cmd/ultimav/gamescene_terrain_dither.go` owns the Ultima adapter, sampling,
  trust bounds, mode gate, projection, and final placement. Its constants are
  the outdoor tuning surface: six native pixels outward, one inward, and a
  two-pixel convex-corner radius. The pattern remains fixed at 4x4 world-pixel
  phase. A quarter touching indoor space or a structural boundary keeps its
  original visibility coverage rather than letting an outdoor neighbor bleed
  through that room boundary.
- `GameState.GetBonusTerrainTile` resolves effective map terrain without item
  stacks, corpses, or Avatar reflections. Legacy artifact sprites use their
  known floor underlay; ordinary `GetEffectiveTile` behavior is preserved.

Sampling follows existing frame visibility, not a new LOS calculation. A
missing or untrusted neighbor sample preserves the original hard edge rather
than creating bonus coverage. Small-map exterior tiles retain their existing
boundary treatment. Bonus cells cannot seed additional bonus cells.

### Shared Map Exposure

`internal/map_state/map_exposure.go` supplies an immutable, four-neighbor
classification independent of player position, visibility, lighting, and
Ebitengine. `ClassifyMapExposure` consumes a plain boundary grid and optional
outdoor seed positions; `LayeredMap.ExposureAt` exposes `ExposureIndoor`,
`ExposureOutdoor`, `ExposureBoundary`, or out-of-domain `ExposureUnknown`.
Floors without an exposure classification also return `ExposureUnknown`; callers
must select their map-context policy rather than interpreting unknown as indoor
or outdoor. Large-map dithering retains its existing outdoor treatment.

Small-map floors build this result once in `ReloadSmallMapFloor` from immutable
source architecture. Walls, windows, doors, embedded fireplaces, and source
`Bookcase`/`SignWarning` wall fixtures are separators. Fences, vegetation, water,
other furniture, archways, and portcullises do
not partition exposure. Eligible border cells and explicit outdoor seeds flood
outward-connected space; other non-boundary cells are enclosed indoors.
Door opening, clock changes, visibility changes, and runtime map overrides do
not reclassify the building. Recreating the source floor recreates its exposure.

`smallMapOutdoorSeeds` in `internal/map_state/layered_maps.go` marks Yew and
Minoc's enclosed ground-floor exteriors plus three disconnected upper wall
walks at Lord British's castle as outdoors.
Serpent's Hold's hall and courtyard already reach
the border and need no seed. Its floor-0 room at `(10,25)` is a separate 30-cell
indoor component, covered by original-data and scene regression tests.
Britain rooms `(22,1)` on floor 0 and `(12,1)` on floor 1 exposed missing wall
fixtures: a bookcase at `(22,0)` and a wall-mounted warning sign at `(11,0)`.
Those fixtures are structural barriers, not openings to the map border.

The [sunlight-aware tinting path](../gamefacts/LIGHTING.md#light-0007) consumes this
same classification and seed catalog, without duplicating the flood fill.
Its daytime baselines and cosmetic opening targets do not alter the dither
mask or gameplay visibility. Structural boundary cells remain distinct so that
lighting can shade wall and opening surfaces independently of the dither
renderer's whole-tile protection policy. No roof-inference system is involved.

### Final Composition

The mask is drawn after world actors, glows, ordinary effects, and projectiles,
but before debug diagnostics, transient feedback icons, static UI, and dialogs.
World effects lacking an existing visibility gate check their normal visible
anchor before composition. The final mask cannot make hidden entity admission
safe by itself. Mask and terrain share camera, layout, and shake transforms;
uncovered aperture strips are explicitly blackened using GPU vertex precision.

X-ray, fully opaque map fades, active over-fade dissolves, combat/camping,
first-person dungeons, mini-map presentations, and finale ownership bypass both
bonus admission and masking. Partial fades and ordinary flash/inversion remain
below rejection. No new workflows, callbacks, save fields, or gameplay RNG
sources are involved. Light-pool perimeter dithering is not implemented.

Automated coverage lives in `internal/sprites/quarter_mask_test.go`,
`internal/sprites/dither_mask_image_test.go`,
`internal/game_state/bonus_terrain_test.go`,
`cmd/ultimav/gamescene_terrain_dither_test.go`, and
`cmd/ultimav/gamescene_terrain_dither_rendering_test.go`. Whole-tile indoor
coverage is tested in `cmd/ultimav/gamescene_terrain_dither_indoor_test.go`;
classification is tested in `internal/map_state/map_exposure_test.go` and
`internal/map_state/map_exposure_original_test.go`. These assertions establish
composition and classification, not final visual acceptance.

## Sprite-Shaped Masks

For glows, poison, sleep, and combat overlays that must follow a sprite silhouette, use:

```go
mask := sprites.GetSpriteMask(sprite)
pipeline.ApplyOverlayWithMask(screen, drawOptions, elapsedMs, opacity, mask)
```

`GetSpriteMask` converts every non-transparent source pixel to opaque white and caches the result by sprite image pointer. Sprites are immutable at runtime, so callers should reuse the cache rather than regenerating masks per frame. The cache helpers exist primarily for tests and diagnostics.

## Generic Tile Dissolves

`GameScene` owns multiple simultaneous tile dissolves:

```go
id := g.StartTileDissolve(pos, spriteIndex, visual.DissolveModeReveal, true)
g.UpdateTileDissolveProgress(id, progress)
g.StopTileDissolve(id)
```

Use `StartTileDissolveWithOptions` when the effect needs a fixed-grid position, a non-default outline, or `RenderLayerOverFade`. Store the returned ID and remove that exact instance. `StopAllTileDissolves` is reserved for scene-wide cleanup.

```go
var dissolveID int
manager.Workflow("materialize_item").
    Priority(event_queue.PriorityNormal).
    Sequential().
    First(func() {
        dissolveID = g.StartTileDissolve(pos, spriteIndex, visual.DissolveModeReveal, true)
    }).
    Animate(500, func(progress float32) {
        g.UpdateTileDissolveProgress(dissolveID, float64(progress))
    }).
    Finally(func() {
        g.StopTileDissolve(dissolveID)
    }).
    Start()
```

## Combat Attack Eligibility

`cmd/ultimav/gamescene_combat_attack_preview.go` draws two disconnected,
compact foot-corner brackets above monster artwork and below the aiming reticle.
The user approved this foreground placement after dense-swarm composition showed
that neighboring sprites could hide every blue pixel in the strict under-sprite
design. This supersedes the earlier underlay requirement, not the eligibility or
colour contract.

The face uses semantic `CombatAttackEligibleFace` (EGA light blue);
`CombatAttackEligibleBorder` is black. Each arm is two source pixels long and
one pixel thick, surrounded on all sides by a full one-pixel black border, not
an offset shadow. Both complete brackets fit inside the 16-pixel cell, inward
of flee/status cues. Their faces moved up one source pixel (Y=9..10), so the
backing ends at Y=12 exclusive, before the allegiance outline beginning at Y=12.
The original HP strip still begins at Y=13. Faces and
expanded backing rectangles follow `combatEntityDrawOptions`, including
battlefield shake, and round transformed edges to crisp pixel boundaries.

The brackets are opaque and steady: no pulse, alpha fade, glow, adaptive colour,
sprite tint, full tile box, or dimming of unavailable enemies. In
`drawAvatarOverlay`, the eligibility pass follows all actor artwork and direction
indicators, immediately before `drawCombatAttackCrosshair`. The existing aiming
reticle has priority; the aimed square's brackets are suppressed while other
eligible enemies retain theirs. Reticle allegiance colours and HP/status cues
keep their separate meanings.

Eligibility comes only from `GameState.PreviewCombatAttack`, shared with Tab.
Before physical aiming, a target needs at least one currently available source;
during aiming, markers and text use only the actual current source. The core
owns range, ammo/interference, visibility, live combat-terrain admission, and
intended-shot obstruction/bypass facts. Occupants on authored
`BlackSquare`/`BrickWallArchway` terrain receive no bracket, name, remembered
target snap, or Tab attack/approach selection. The renderer adds neither a LOS
formula nor guaranteed-hit claims.
Queries and drawing consume no gameplay RNG, resources, time, or transcript
output.

No marker cache or persistent registration exists. Each read reflects actor,
movement, readied equipment/ammo, source, target life/visibility, and targeting
state. Enemy/auto-pass turns, pending attacks, input-blocked workflows, and
unrelated targeting suppress assistance. Cancelling Aim removes its readout and
restores normal any-source markers when eligible; combat exit yields no preview.
Projectile finalizers retain their separate cleanup ownership; see
[physical attack ownership](INPUT_HANDLING.md#combat-physical-attack-ownership).

Core eligibility/purity and scene bracket pixel geometry, steady rendering,
reticle priority, readout, and cleanup assertions are in
`internal/game_state/combat_attack_preview_test.go` and
`cmd/ultimav/gamescene_combat_attack_preview_test.go`.
`cmd/ultimav/gamescene_combat_marker_composition_test.go` adds the real outer
actor pass plus dense all-nine, center-only, and checkerboard marker layouts at
native, 720p, and 768p scales with subpixel origins. Assertions preserve both
marked faces and complete outlines, HP/status raster footprints, unmarked and
unrelated pixels, and reticle priority.

Composed-render assertions do not replace live swarm keyboard, HiDPI, or
gameplay acceptance, which remain separate validation work.

## Persistent Combat Allegiance

`GameState.ProjectCombatActor` is the read-only exact-entry side/disclosure
projection shared by live health-frame outlines and target/current-actor text. It delegates
side to `combatIdentityLoyalty`, including intrinsic Saduj hostility and
clone-local possession. Its monster `CombatantID` guard rejects recycled slots;
PC clones retain their distinct `Slot`. Encounter-only charm and summon flags
are recorded at successful producers, never inferred from loyalty. A dividing
monster creates a fresh hostile child without inheriting those flags.

Every disclosed battlefield actor uses a steady one-source-pixel **cyan Ally /
red Enemy exterior outline around its black HP frame**, paired with explicit
Ally/Enemy text. This is the revised design following visual review of the larger
upper-left symbols. The exterior is `(x=1, y=12, width=15, height=6)` inside the
18-pixel replacement sprite. The existing black HP frame remains exactly
`(x=2, y=13, width=13, height=4)`, including its separator, fill dimensions,
health-threshold colours and opacity. The outline never recolours or covers the
HP interior. Non-combat world/vehicle callers of `drawCombatHealthBar` receive
no allegiance outline.

HP and allegiance render together in a foreground pass after actor/status
artwork, protecting both from overlapping 18-pixel neighbors. Teleport/capture
frames partition body and health into full-size temporary images before scaling;
their health region receives the same mask, opacity and presented position exactly
once in that foreground pass. Gate health frames stay steady after the hidden
prelude. The possession cap remains upper-right. Insect stacks emit one aggregate
frame with their unchanged health segments and member order; pips sit on source
row 2 so a northern neighbor's outline cannot cover them. Hidden members do not
contribute pips or segments. Attack brackets sit above the health frame, selection
remains underneath and the aiming reticle retains foreground priority.

An encounter-local presentation observer compares only continuously disclosed
exact identities. A visible side change immediately updates the outline and adds
concise transcript feedback. The exterior stays cyan/red without a white flash
or pulse replacing the side meaning. Newly created or
newly revealed entries establish silent baselines. Hidden actors, temporary
hidden signature phases, recycled slots and combat exit cannot produce stale
transition feedback. No gameplay mutation, time advancement, RNG draw or input
owner is introduced. Historical event names and colours remain immutable.

Actual-size render fixtures can be captured by setting `ULTIMA5_REVIEW_IMAGES`
to an existing directory outside the repository and running the focused
`TestCombatHealthFramesActualSizeLookalikes` and
`TestCombatSignatureAdmissionAfterHitchRetainsFullScheduledLifetime` tests.
These are inspectable 1280x720 compositions, not live or joint visual acceptance.
`TestCombatFullHUDCombinedComposition` additionally draws the battlefield, full
combat HUD, roster, transcript and provisions together at 1280x720 and 3840x2160.
Set `ULTIMA5_REVIEW_IMAGES` to an existing external directory to export those
frames. The assertions include reverse spatial draw order with adjacent poisoned
actors, preserving complete HP fills, separators and exterior outlines through
the final HUD composition. Exports include `combat-health-frames-full-hud-1280.png`,
`combat-health-frames-full-hud-3840.png`, and grass/water comparisons with paired
full, quarter and one-hit-point lookalikes. Native and noninteger-scale HP byte
preservation, insect-stack attribution, signature opacity/masking, temporary
concealment and source/clone identity are covered by
`cmd/ultimav/gamescene_combat_health_frame_test.go` and the combat polish tests.

## Combat Signature Feedback And Conditions

`GameScene.recordCombatEvent` consumes already-committed `CombatEvent` results.
`cmd/ultimav/gamescene_combat_signature_effect.go` owns concurrent, short-lived
snapshots for these successful outcomes:

| Event | Presentation | Audio cue request |
|---|---|---|
| `CombatEventGating` | 125 ms arrival disturbance, 600 ms scattered-pixel materialization, 125 ms settling of the **created** Daemon at its committed position. | `SoundMonsterGate`, once at materialization onset |
| `CombatEventPossession` | Stepped blue caster-to-victim link and onset corners; a blue marker remains on the possessed entry afterward. | `SoundMonsterPossession` |
| `CombatEventTeleport` | Departure, hidden interval and arrival at the published endpoints. | `SoundMonsterTeleport` |
| `CombatConditionDraggedUnder` | Corpser victim hides/dissolves, with an Underground cue. | `SoundCorpserCapture` |
| `CombatConditionRegurgitated` | The released exact victim reveals again. | `SoundCorpserRelease` |

Routine underground turns do not replay capture. These snapshots never summon,
move, charm, release, apply damage or complete a turn. They retain the encounter,
exact PC/clone identity or immutable monster `CombatantID`, and original event
positions. A reused slot cannot hide or animate its replacement. Gate/capture
dissolve seeds are presentation-owned; effect updates consume no extra gameplay
RNG. `StartedMs` is captured from `GameState.ElapsedMs` at admission. Gates
advance at most one named phase per update and anchor the new phase to that
update's clock; a hitch cannot consume an unpresented phase. Their live predicate
remains true until effect update completes settling, including when the workflow
polls before visual effects. Nominal duration is 850 ms; hitches extend it.
Repeated updates at the same elapsed value do not age it. Wind continues on the shared main RNG during these
waits: reproducibility requires the same elapsed/action timeline, not just the
same seed. See [timing/RNG scope](TIME_SYSTEM.md#elapsed-time-and-shared-rng-scope).

Live authored `NOTHING`/`ARCH` underlays suppress actor-local disclosure, and
invisible monsters do not gain visible signature sprites. Possession snapshots
also retain admission-time origin disclosure: an invisible caster gets no
origin cue or linking mote, while the visible victim cue remains. A disclosed
Daemon origin survives its subsequent retirement or slot reuse.

The effects themselves create no workflow and acquire no input. The existing
scheduled monster/auto-pass workflow awaits live signature effects before its
exact-once turn completion, and clears them in `Finally`. Expiry, invalidated
identity or position, possession recovery, combat exit/replacement and scene
disposal also release the dissolve, composed frame, body/health layers and
highlight images. Player summon
and `In Por` retain their own workflow finalizers; cancelling those presentations
must not summon or teleport again.
Active signature effects are transient and reject scene snapshot capture.

### Conditions, Selection And History

`cmd/ultimav/gamescene_combat_conditions.go` projects Possessed, Underground and
Sleep from each active exact combat identity. `internal/ui/mainscreen/combat_conditions.go`
formats those detached conditions for the combat turn HUD, which shows the
disclosed acting entry's own combination. Concealed/underground actors use
`Unknown` rather than leaking their names or clone identity in live readouts.
Source and clone conditions remain independent.
Persistent conditions such as Poisoned can coexist; this projection never
mutates the party roster to force a colour or label.

The top-right party panel uses the same portraits, two-line name/HP/status/MP
layout, XLarge font, and line spacing in combat and exploration. In particular,
`Pois.` remains in its ordinary status column with its ordinary typography;
entering combat does not shrink portraits or move status text into a compact
block. Encounter-only details belong to the combat HUD and exact-actor
battlefield indicators, not a second roster layout.

The ongoing Possessed marker is a blue cap inside the sprite's upper edge and
is paired with explicit `Possessed` text. It remains distinct from active
selection, target indicators and HP colour, including when an unpossessed source
and possessed clone coexist. Underground and Sleep are independent labels;
rendering a short onset effect does not own their lifetime.

`cmd/ultimav/gamescene_combat_actor_conditions.go` shares exact-entry composition
between ordinary sprites and signature frames. Teleport and Corpser conditions,
health outlines and HP are composed before applying the frame's dissolve/alpha. Gating
first highlights newly revealed **sprite-only** pixels, then composes conditions
and masks the silhouette; the persistent health outline and untinted HP follow. Highlights
never recolour semantic marks. The entire gate replacement suppresses ordinary
actor rendering, including the sprite-free prelude. The white leading pixels
and staged pacing are approved Redux presentation embellishments to the original
fizzle, not changes to gating mechanics. All masks, frames and highlight buffers
are released on completion and every invalidation/cancellation exit. Teleport selection
follows the departure position, disappears during the hidden interval, and
returns at arrival rather than exposing the committed destination early.
Targeting remains in the foreground; HP colour remains independent of status.

`cmd/ultimav/gamescene_combat_report_identity.go` captures actor names,
allegiance colours and disclosure permissions when an event enters report
history. Known historical names remain stable through retirement or slot reuse.
An actor concealed at admission
stays `Unknown` in that historical event even if later revealed. Both compact
and detailed player reports suppress concealed coordinates, diagnostics and
creature types, including typed gate/vanish/summon/repel payloads; later reveal
or slot reuse cannot disclose them retroactively. Workbench raw diagnostics
retain authoritative data independently of this player-facing policy. Live turn
HUD identity is queried independently: invisible monsters have no sprite, name or
DEX detail; the optional concealed forecast slot remains anonymous (`???`).
This is a Redux information-display policy, not a reconstruction of the DOS HUD.

Original gate creation includes a fizzle reveal;
possession and Corpser capture also have original feedback calls. The modern
blue link, marker, text composition and timing are Redux presentation choices.
Gameplay/presentation boundaries are documented in [COMBAT-0012](../gamefacts/COMBAT.md#combat-0012)
and the [species mapping](../gamefacts/ENEMIES.md#enemy-0002).

### Sound Availability

The five named signature cues above are wiring placeholders awaiting recordings,
not listening acceptance. The authoritative recording assignments are in
`internal/audio/assets/library.json`; a `file: null` slot is silent. Existing
empty magic, poison, sleep, miss/graze/flee and projectile/impact slots likewise
do not become audible because a caller requests them. `SoundMagic` is not proof
of an audible fallback. Callback tests establish which cue is requested;
recording work remains incomplete. See
[Audio System](AUDIO_SYSTEM.md) for the manifest and playback boundary.

### Practical Keyboard And Workbench Walkthrough

This is a repeatable inspection procedure, not a record of manual acceptance.

1. Launch the [Debug Workbench](../DEVELOPMENT_GUIDE.md#starting-the-debug-workbench)
   with local original data. From stable exploration, use Encounter Options and
   Start Encounter (`combat.encounter.options`, `combat.encounter.start`) to
   choose a map, exact count, friends off and an explicit seed. Retain the seed,
   party/loadout, map and keyboard action sequence for comparison; exact replay
   also requires the supplied elapsed timeline. Use Active
   Combat (`combat.status`) to inspect IDs, positions, conditions and queue state.
2. Start with Dragons to observe gating without the caster's possession branch;
   separately use Daemons/Shadowlords for possession, Wisps for teleport, Ghosts
   for invisibility, and Corpsers on a map with usable grass. Use keyboard Space
   to pass and arrows to position the party until the relevant committed event
   occurs. Keep at least two living PCs for possession. Probability gates mean
   an arbitrary seed does not guarantee a special on its first turn.
3. Compare the event report with the battlefield and Active Combat state. A gate
   reveals one newly created Daemon; teleport shows both endpoints; successful
   possession leaves blue marker/text after its link expires. A Corpser's
   damaging surviving capture hides the victim and labels it Underground;
   release reveals it once. Selection follows the visible teleport phase, and
   compound status/HP cues remain legible independently.
   Check at native and scaled window sizes with adjacent actors.
4. For clone composition, prepare the required spells/resources through existing
   Workbench controls, then use keyboard `C` and the spell dialog for
   `In Quas Xen`. Compare source/clone IDs and simultaneous conditions; one
   entry's possession/sleep/underground state must not colour every copy. Browse
   report history with its displayed keyboard hints after invisibility changes
   or retirement: history retains admission-time identity/disclosure, including
   detailed coordinates and typed creature names, while live invisible turn
    slots remain anonymous. Combat entry, condition changes and exit must retain
    the same party portraits and name/HP/status/MP typography as exploration.
5. Let effects expire and leave combat through ordinary victory/evacuation;
   confirm no stale marker, hidden actor, dissolve or blocked input survives.
   At an eligible quiescent boundary, Workbench Discard Active Encounter can
   separately inspect teardown. It is not a way to interrupt an input-owning
   effect mid-animation. Mid-phase shutdown and replacement are covered by the
   automated lifecycle tests below; scenario Cancel does not cancel gameplay
   workflows. Audio listening requires populated manifest recordings.
6. Open a multi-item combat chest and compare each temporary icon with its
   discovery description. Acknowledge with a fresh Enter/Escape, wait several
   seconds, then pick up items one at a time: persistent pile display must stay
   stable until its contents change. Inspect overlapping fields/body markers
   where available; a field must not cover the winning ground object or reveal
   preview. A nested unopened chest remains closed. Search/Jimmy/An Sanct retain
   their exact selected chest's knowledge independently of display order. Check
   uncertain-safe key loss, armed failure/success, and known-safe refusal; use
   surface/first-person dungeon chests for save/load and fresh-visit lifetime
   checks because combat saves are unsupported.
7. Ready a Bow and a melee off-hand, then let an adjacent hostile creature make
   a melee attack against that PC. While the remembered attacker remains eligible,
   use keyboard `A`: one classic interference message and one Battle Report entry
   identify the creature, PC, and Bow, and the remaining source is still offered.
   Repeat from a fresh melee engagement for `Tab` and `C`; Cast rejection does
   not enter the spell dialog or spend MP/prepared spells. Browse/redraw the
   report and check that no message/cue repeats. The interrupted-action recording
   is intentionally empty; this walkthrough can inspect report/command behavior,
   while listening acceptance requires a future assigned recording.

Generic Workbench VFX previews inspect rendering primitives; real encounter
events above exercise authoritative producer-to-presentation routing. Neither
previewing a primitive nor reading an automated assertion constitutes a live
keyboard/audiovisual acceptance pass.

## Projectile Travel

`gamescene_projectile.go` renders the source, resolved endpoint, and missile
type supplied by authoritative gameplay. Tactical attacks use the scheduled
monster workflow and `CombatProjectileLaunch`; surface fireballs and pirate
broadsides share the outdoor FIFO workflow. The renderer never retraces a shot
or rolls gameplay RNG. Large-map coordinates take the shortest wrapped path.

`ProjectileAnimationSequence` starts at the source, animates to the resolved
endpoint, clears the projectile, then invokes impact. The owning workflow's
finalizer also clears transient state. Scheduled monster impact and completion
are idempotent: shutdown resolves an already-prepared attack once; encounter
teardown discards the old encounter's pending impact. Outdoor damage is already
resolved before presentation and must never be replayed by flight or cleanup.

Queue admission gives a new flight progress zero rather than charging the
previous update's pre-launch interval (see
[animation admission](EVENT_QUEUE_SYSTEM.md#animation-admission-and-time)).
Timing uses native pixels, independent of display scaling: ordinary missiles
travel 13 pixels per 40 ms, rocks 8, and town missiles 6. The Euclidean path and
100 ms minimum are deliberate Redux presentation choices; original evidence is
recorded in [COMBAT-0012](../gamefacts/COMBAT.md#combat-0012).

### Rendering Families And Automated Probes

| Rendered family | Tactical enemy references exercised by the scheduled-flight test |
|---|---|
| Rock outline | Gargoyle, Ettin |
| Directional arrow | Bard |
| Directional axe | Guard, Troll |
| Blue magic | Wanderer, Lord British, Sea Horse, Squid, Reaper, Mage |
| Red magic | Blackthorn, Sea Serpent, Devil, Dragon, Shadowlord |
| Green magic | Mimic, Snake |
| Violet magic | Gazer |
| Cannonball silhouette | Pirate ships use the outdoor workflow rather than the creature-reference inventory |

`gamescene_projectile_rendering_test.go` exercises all 19 ranged references
through normal scheduled turns and reads Ebitengine pixels at origin and during
flight. The shared-family raster probes cover grass, brick, water and black
backgrounds at 960×540, 1280×720 and 1920×1080 logical sizes, including shake.
Surface and underworld visibility probes reject hidden centers under the normal
terrain-dither path, including fractional camera offsets. Combat projectiles
remain event feedback, following the existing combat admission rules below;
they do not disclose an actor sprite under concealed terrain.

`gamescene_combat_scheduler_projectile_test.go` covers the ordinary Ettin's
250 ms seek-delay boundary, intermediate frames, hit/scatter/blocked endpoints,
short legal shots, exact-once impact/completion, shutdown and encounter exit.
`gamescene_projectile_world_test.go` exercises actual outdoor Pass completion
for Dragon/Sea Serpent fireballs and pirate cannonballs, including wrapped
origins, obstruction, cancellation, and presentation-only RNG accounting.
Melee/reach exclusion remains covered by `gamescene_reach_weapon_test.go`.

These probes establish raster contribution and bounded position, not perceptual
recognition or live acceptance. A live pass must still record the build and
display settings, allow a normal Ettin turn, observe recognisable stone travel,
and repeat representative physical, coloured and cannonball flights at supported
window/fullscreen and display scales. Neither these pixel assertions nor a VFX
primitive preview substitutes for a recorded gameplay observation.

## Other Current Effects

- Projectiles are started with a world source, target, and missile type; `Animate` updates progress and `Finally` clears the active projectile.
- Kapow effects use the explosion sprite and a scale triangle over their short lifetime.
- Combat impact effects render local hit feedback and floating results from `CombatAttackFeedback`.
  Its typed presentation metadata separates melee, projectile, spell, and
  environmental delivery while retaining source equipment for future presets.
  Melee feedback draws a short concurrent EGA strike above battlefield actors;
  projectile, spell, and environmental feedback never reuses the melee strike
  or attacker lunge. Projectiles, cone footprints and generic impacts remain
  event feedback and are not suppressed by authored-terrain occupant admission.
  Player reports separately apply the admission-time disclosure rules above.
- Combat actor-local silhouettes, selection frames, division tethers, field
  removal cues, and summon/repel/teleport identity cues consult the actor's live
  map-only underlay. `BlackSquare` and `BrickWallArchway` suppress those direct
  location disclosures without cancelling or delaying the underlying effect.
- Committed combat Sceptre use records a short actor cue with the exact combat
  identity and position before barrier or direction resolution. Successful
  `CombatEventFieldDispelled` events independently record a collapsing cue at
  the removed field position, so Sceptre and `An Grav` share producer-neutral
  result feedback. Both concurrent effect slices use `GameState.ElapsedMs`,
  consume no gameplay RNG, add no workflow delay, and clear outside combat.
- Map fade is a black map-area overlay controlled by `SetMapFadeLevel`.
- Moongate renderers project the authoritative `GameState` stage; callbacks
  refresh presentation resources but do not own an independent growth state.

## Verification

Focused tests live beside the owning packages:

- `internal/visual/` for dissolve progress and scaling
- `internal/sprites/tile_overlays_test.go` for sprite-mask caching
- `internal/game_state/party_state_builder_test.go` for flash and shake callbacks
- `cmd/ultimav/gamescene_draw_effects_test.go` for rendering calculations
- `cmd/ultimav/gamescene_input_combat_test.go` for combat effect integration
- `cmd/ultimav/gamescene_combat_sceptre_effect_test.go` for Sceptre actor and
  producer-neutral field-dispel cue lifecycle
- `cmd/ultimav/gamescene_combat_signature_effect_test.go` for production event
  routing, cue requests, unchanged gameplay state/RNG, concurrent expiry,
  recycled-slot rejection, terrain/invisibility and exit/replacement/disposal
- `cmd/ultimav/gamescene_combat_signature_lifecycle_test.go` for scheduler-owned
  waiting/completion, queue shutdown, and cancellation at every player Daemon
  scroll summon or `In Por` presentation phase
- `cmd/ultimav/gamescene_combat_signature_timing_test.go` for hitch-time admission,
  full effect lifetime, exact turn-token ownership and the shared-wind RNG oracle
- `cmd/ultimav/gamescene_combat_signature_disclosure_test.go` for concealed versus
  retired possession origins and immutable report disclosure after slot reuse
- `cmd/ultimav/gamescene_combat_signature_composition_test.go` for teleport
  selection phases, compound actor conditions/HP, targeting priority and cleanup
- `cmd/ultimav/gamescene_combat_conditions_test.go` and
  `internal/ui/mainscreen/combat_conditions_test.go` for exact source/clone
  conditions, combined HUD labels, marker/selection/HP independence, recovery and
  invisible live HUD versus admission-time report history
- `internal/ui/mainscreen/character_summary_layout_test.go` for original-size
  portraits and identical name/resource/status rendering across combat entry,
  conditions, recovery and exploration, including font-metric preservation

The browser Workbench VFX instruments are the supported runtime inspection surface.
