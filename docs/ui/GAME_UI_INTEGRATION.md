# Game UI Integration

**Purpose**: Document how `GameScene` wires widgets, callbacks, update ownership, rendering order, transient feedback, and resolution changes. Widget internals and dialog selection are documented separately.

## Owned Presentation State

`GameScene` owns:

- `widgets.DialogStack` for modal and non-blocking dialogs
- `text.Output` for the right-side game record
- `input.Keyboard` and the primary/secondary command state machine
- borders, character summary, compact provision summary, display-only status rail, wind display, and transient icon overlay
- visual-effect state such as map fade, flash intensity, shake offset, and inversion
- optional `audio.AudioManager` when sound is available
- feature UI such as the combat HUD, detached Z-Stats and Quest Journal modals,
  and transient action-feedback records

Core gameplay reaches this layer through `SystemCallbacks`; it does not construct Ebitengine widgets.

The surface sky and Winds readouts share the rasterized outer map-frame center
and blue/white inward bookend geometry. The sky's glyph cells are spread across
a strip about 50% wider where the frame has room, capped on narrow screens to
preserve the rounded corners while retaining the glyphs' original font scale.
Its black backing spans the full upper border band beneath the decorated strip,
covering the frame's otherwise exposed lower white rule. Both readouts are
presentation-only; frame resizing rebuilds their graphics without advancing
the calendar or consuming gameplay randomness.

The bottom frame uses `mapBorderLabel`: large maps retain Winds, while ordinary
small maps show the current friendly location name through the same centred
blue/white bookends. It does not append a floor suffix or change wind authority.
Combat, camping, shrine/Codex and other non-surface-frame scenes suppress the
label. `WindDisplay.DrawLabel` measures and rebuilds its Output/chrome when the
label or layout changes.

## Initialization

`wireSystemCallbacks` constructs the required message, visual, audio, screen, flow, talk, UI, and combat callback groups. Constructors validate mandatory functions before `NewSystemCallbacks` accepts the aggregate. Code running after successful initialization should use those callbacks directly.

Supply projectile/impact callbacks inside `NewVisualCallbacks` and MessageDialog
callbacks inside `NewScreenCallbacks`. `cmd/ultimav/gamescene_callbacks.go`
shows complete named-family construction; do not attach mandatory members later.

Audio is the exception at the concrete-resource level: sound-disabled mode or a recoverable audio initialization error leaves `GameScene.audioManager` nil, while the `AudioCallbacks` functions remain present and safely perform no playback.

## Update Order

The normal `GameScene.Update` path is listed below. An active Journal session
is intercepted before queues and gameplay updates, including on its dismissal
frame; see [Journal pause](../systems/TIME_SYSTEM.md#journal-pause) and
[input routing](../systems/INPUT_HANDLING.md#routing-order). Ordinary modals do
not imply that full pause.

1. Clear per-frame keyboard consumption.
2. In `updateWindAndEffects`, update ambient audio and audio fades.
3. Publish current elapsed time to `GameState.ElapsedMs`, then update wind and lighthouse state.
4. Update workflow, background, and feedback event-queue lanes.
5. Reconcile scene-owned transient-feedback clocks, then update non-blocking dialogs, timed overlays, visual effects, and the wind display.
6. Update the transient status icon overlay.
7. Handle temporary animation locks such as auto-open doors and rising moongates.
8. Update only the highest modal dialog; return if one exists.
9. Honor the global `inputBlocked` flag.
10. Resolve pending combat, turn, and auto-talk state.
11. Update moongate presentation.
12. Route secondary command input before primary command input.

This order lets modal dissolve and dismissal state advance while a workflow owns the global input block. Do not update a modal from both the feature and `GameScene`; the stack is its frame owner.

## Dialog Integration

Push dialogs directly by role:

```go
g.dialogStack.PushModalDialog(dialog)
g.dialogStack.PushNonBlockingDialog(feedback)
```

Retain exact instances for cleanup:

```go
g.dialogStack.RemoveWidgetInstance(dialog)
```

Avoid type-based removal and nonexistent pop APIs. Multiple overlays and message dialogs may coexist.

Common event-queue helpers in `gamescene_dialog_helpers.go` return sequence modifiers for confirmation, yes/no, direction, text input, sleep duration, and party selection. Add them with `ThenSequence` so the parent workflow owns and waits for their steps.

## Action Feedback

`SystemCallbacks.UI.ShowActionFeedback` routes to `GameScene.ShowActionFeedback`:

1. Optionally append the message to the main output.
2. Create `widgets.NewActionFeedbackDialog`.
3. Attach its scene-owned animation clock, show it, and push it as non-blocking.
4. Measure complete wrapped content and admit it to a newest-first stack of at most three cards.
5. Place the stack within the surface, dungeon-HUD, or combat-battlefield safe area, evicting oldest cards when complete cards do not fit.
6. Animate rearrangement from current positions while pausing only the affected cards' remaining reveal, hold, or hide budget.
7. Remove the exact instance after its 400 ms reveal, 1600 ms readable hold, and 300 ms hide complete.

Rapid feedback calls remain independently owned and never block gameplay. Fitting cards retain complete content and visible gaps; an oversized newest singleton keeps its beginning visible and clips at the lower feedback boundary. Resize and map-mode changes remeasure and retarget the live stack without refreshing lifetime. Scene replacement and disposal remove every retained exact instance.

## Smooth Surface Movement

`Gameplay.Presentation.SmoothSurfaceMovement` changes only the camera projection after authoritative movement and synchronous completion. Walking, horses, carpets, manual skiff movement, and furled-frigate rowing use the configured ordinary duration and curve; only horses and carpets apply the 25-percent faster presentation factor.

Automatic raised-sail frigate translations use the selected ordinary curve but finish within the existing 55 ms sailing phase. Heading changes, wind or calm waits, blocked attempts, contacts, hazards, and transitions do not fabricate movement. The automatic-sailing workflow is the only workflow admitted by that specialized projection, and every world layer shares the same wrapped camera offset. Presentation never delays movement completion, clock advancement, world actors, contact, or RNG.

`ShowTimedOverlayMessage` has separate GameScene ownership for one modal timed overlay. `installTimedOverlay` first removes the previously owned exact instance, shows and pushes the replacement, and derives one deadline from `GameState.ElapsedMs`. Expiry, scene replacement, and disposal converge on `clearTimedOverlay`, which cannot remove an unrelated or covering modal. These overlays use `WithExternallyManagedDuration`; their widget `Update` method does not run a competing frame-count timer. New feature code should use the established callback or these scene helpers rather than writing the timed-overlay slot directly.

`ShowTransientIcon` is the non-blocking semantic status channel. `GameState`
emits one of eight fixed kinds after an actual eating, starvation, explicit
healing, poison transition, Wanderer level-up, or clamped moral-karma change.
`GameScene` maps that meaning to presentation and registers it with
`ActiveEffectsOverlay`; gameplay code does not choose the sprite, tint, or
duration. The overlay uses gameplay elapsed time, refreshes repeated kinds,
stacks distinct kinds up to five records, and is cleared only by scene
replacement or disposal. It never enters the dialog stack or event queue and
never owns input.

## Provision Change Feedback

### Ammunition Summary And Shortages

The provisions panel has **four normal-font rows**: compact label/quantity pairs
centered in thirds on the first two rows, followed by ammunition and calendar
rows centered in matching equal halves:

| Row | Left | Middle | Right |
|---|---|---|---|
| 1 | `F` — Food | `K` — Keys | `G` — Gems |
| 2 | `$` — Gold | `T` — Torches | `Ka` — Karma |
| 3 | `Ar` centered in left half | | `Qr` centered in right half |
| 4 | Date centered in left half | | Time centered in right half |

Each resource column reserves a fixed label slot wide enough for both rows and
a fixed quantity slot wide enough for its maximum value. That compact block is
centered in its third. Labels are right-aligned; quantities are left-aligned
after a small scaled gap, giving both rows the same numeric starting X coordinate.
Neither changing digit counts nor `G` versus `Ka` shifts the quantity anchor.
Ammunition blocks reserve
two-digit quantity slots and fixed `LOW`/`OUT` slots, including while healthy,
with a full normal-font space between the quantity slot and the warning;
changing the count or warning cannot shift the label, count, or neighboring block.
Each complete ammunition block is centered in the same half as the date or time
below it. Labels, quantities, status words, date, and time all use the **same
normal font size**, with no smaller fallbacks. The calendar centers each string
within its half, giving the date the extra pixel when the row width is odd; the
largest supported numeric date fits at normal size. The panel retains its
existing height and frame clearance at every resolution.

`Ar` and `Qr` labels stay cyan, and their quantities always stay white. Only the
status word is coloured: yellow `LOW` at 1–20 and red `OUT` at zero, with no
warning above 20. Status words use the same font size as quantities. Other
resource quantities stay white. Ammunition counters are read-only
snapshots of party equipment stock, independent of readied weapons.

`ProvisionSummaryHeight` reserves the four scaled rows. The supplies area grows
upward into the message panel, retaining the bottom inset and the map, party and
status-band geometry. The shortened transcript clip fits **11 visible message
rows**, with 16-character wrapping and the same 64-row buffer and Journal
History through resize.

Scene before/after observation emits one nonblocking action-feedback
notice, also appended to ordinary Output, when stock crosses from above 20 into
1–20 or from positive to zero. It compares
final balances, not gross transaction activity; low-stock increases do not rearm
LOW, while any positive refill allows a later OUT. Activation/hydration silently
baseline stock. These notices use `ShowActionFeedback(notice, true)`, not a modal
or the recent-provisions strip.
Sources: `internal/ui/mainscreen/ammunition.go`, `provision_summary.go` in that
package, and `cmd/ultimav/gamescene_provision_changes.go`.

### Provision Observation

`GameScene` observes detached provision snapshots at entry to and exit from
`Update`, including early returns for merchant modals and workflow input locks.
`internal/ui/mainscreen/provision_changes.go` owns the presentation model;
`provision_change_panel.go` in that package owns its measured `text.Output`
buffers. Observation never changes prices, time, RNG, inventory, or input.

Each quantity stack retains runtime-only gross gained/lost counters in
`internal/party_state/quantity_activity.go`. They count **actual post-limit
changes**, including `Set`, and exclude failed decrements and unchanged writes.
This small amount-level observation seam preserves opposing mutations inside one
command or queue update, which plain before/after balance subtraction cannot do.
Transaction call sites and mutation ordering are unchanged. The counters are not
part of native or legacy saves and must not be used for gameplay decisions.

### Presentation And Coalescing

- The strip is centered in the current **map viewport**, not the entire screen.
  Normal gameplay anchors it near lower-map feedback. Merchant/modal interaction
  or active action feedback moves it to map-top, keeping the lower region
  available. Geometry and modal overlap are re-evaluated on resize.
- Typography scales against the logical **screen width**, independently of the
  narrower map/battlefield placement bounds. Ordinary rows use main-screen body
  text; the food-loss heading retains its larger title size. Fresh feedback uses
  the same typography in exploration and combat.
- A single opaque **Recent provisions** strip is drawn above dialogs only when
  its measured bounds do not overlap the active modal's bounds (merchant parents
  expose their shell; transaction modals expose their own bounds). With unknown
  or overlapping modal bounds it draws underneath the dialog stack. This protects
  tight layouts such as Dungeon View, including its heading near the screen top.
  Small ordinary grids and a simple food-loss notice fit above the merchant
  shell's 10% top edge at supported logical resolutions, including 1280×720.
  A larger combined food/resource card follows the same measured overlap rule,
  preserving choices, receipts, and keyboard hints. With no modal, the strip
  draws above the stack on surface, dungeon, and combat scenes.
- Resource order is Food, Gold, Keys, Gems, Torches, Skull keys. Only changed
  resources appear, ordinarily as `Food +20 -6  Now 218`. Actual food losses use
  a featured normal-title-size `FOOD -6` heading, with remaining stock and any
  simultaneous gross gain on separate body-size lines. Other resources retain
  their compact grid beside it, or below it when width requires stacking. This
  is recent actual loss, not a last-meal record or consumption-rate prediction.
  Labels, signed gains, signed losses, and current values use existing distinct
  semantic dialog colours.
  The signs and names carry the meaning independently of colour.
- Up to six resources appear. The compact resource grid uses at most three
  measured columns, ordinarily one or two rows plus a heading; featured food
  occupies its own block beside or above the remaining grid. Long cumulative numbers
  reduce the column count instead of clipping, abbreviating, or shrinking text;
  that exceptional taller strip yields to any overlapping active modal.
- Each resource has a **six-second idle timeout** on `GameState.ElapsedMs`.
  Time obscured by an overlapping or unmeasurable modal still counts toward expiry;
  dismissing the dialog does not restore an expired receipt.
  A further actual change refreshes only that resource and adds to its separate
  gain/loss accumulators. Food gained and eaten in the same update therefore
  remain `+N -M`, even when the final balance is unchanged. Repeated eating uses
  one entry instead of producing a notification backlog. `Now`/`Remaining` report
  the latest observed balance, not the balance when the entry first appeared.
- One transaction's changed resources appear together on the next observation.
  This is a bounded recent summary, not an ordered transaction history. There
  are no animations, sounds, acknowledgement keys, queue ownership, or focus
  changes. Draw and resize do not refresh lifetime.
- Activation silently baselines loaded/initialized quantities. State replacement,
  a rewound clock, and reset activity counters discard prior feedback; scene
  replacement and disposal clear it. Native hydration creates a new `GameState`
  and never replays loaded stock as a reward. Live developer inventory changes
  are observable like other live mutations, while candidate-scene setup is
  absorbed by activation's baseline.

### Provision Feedback Context Boundaries

Old receipts belong to the interaction that produced them. Cleanup is
presentation-only and never resets ammunition-shortage observation or inventory.

- A change of location, floor, or combat instance silently adopts current
  provisions as the new baseline. This covers town/dungeon entry and exit,
  stairs, camping-map setup/return, combat entry (including ambushes), and combat
  exit before its summary. Ordinary movement within the same map preserves feedback.
- Moongate/teleport source-renderer cleanup clears receipts even for a destination
  on the same map and floor.
- Opening the atlas or quest journal immediately clears receipts. Their frozen
  scene clocks cannot preserve old receipts for redisplay after reading.
- Entering the first modal clears already-displayed rows while preserving fresh,
  unobserved activity generated by that interaction. This includes NPC/merchant
  conversations, rest setup, and Dungeon View. Nested approval/result dialogs
  preserve fresh transaction feedback; they do not restart its timeout.
- `HideAllOverlays` clears and suppresses provision feedback until
  `ShowAllOverlays` silently baselines it again. Map fades and the finale also
  suppress feedback, keeping bed rest, death/resurrection, and other faded
  scripted presentations free of stale receipts. Suppressed changes are not
  replayed afterward. Fresh changes after the boundary remain eligible.

`cmd/ultimav/gamescene_provision_lifecycle_test.go` covers transitions, ordinary
movement, nested modals, frozen readers, teleport cleanup, and scripted suppression.
The production tavern purchase tests continue to verify fresh merchant receipts.

### Combat Loot Feedback

Combat chest discovery uses `CombatLootDialog`: immutable admitted icon/text
pairs accumulate at 500 ms intervals, with measured keyboard scrolling for longer
lists. Each row centres the icon and measured wrapped text block on a shared
vertical centre. A fresh Enter/Escape after the reveal acknowledges the list; the workflow
cleans its exact dialog and completes its deferred token once. Discovery leaves
the items on the ground. Persistent display is stable in original slot order:
the lowest ordinary-object slot wins; when only fields/dead markers remain, the
highest slot wins. Fields share that selection despite their separate overlay
pass, and are suppressed at the temporary reveal position. Chest priority and
continuous icon cycling are absent. Removing the displayed object exposes the
next winner without changing object slots, chest knowledge, or command targeting
and Get order. Terrain concealment applies to both the pile and preview.
Trap consequences appear above the discovery list and in the battle report.
The report records each discovery at the same reveal cadence, explicitly as an
item on the ground; an empty chest is reported without implying an acquisition.

Combat Get publishes one `CombatEventPickup` with a detached actor name, target
position, item identity, and actual post-cap quantity. The full-sized **Battle
Report** renders labelled `LOOT` or `PICKUP FAILED` entries instead of map-overlay
receipts. Ordinary player/enemy movement is discarded before admission to the
64-event report history; teleportation, fleeing, and other consequences remain.
Pickup gains already reported there are absorbed from the provision observer's
gain baseline, preventing a second map-overlay receipt. Unrelated gains and all
losses in that same update remain observable.

Pickup, discovery, and trap entries receive a 2400 ms scene-clock accent. Only
time with their complete entry exposed in the report counts; modals, party-spell
targeting, and browsing elsewhere preserve it. Rendering and resizing do not
consume the budget. `F3` jumps to the oldest pending result (or the latest result
when none are pending), acknowledges its pending marker, and renews its accent.
New events preserve a browsed endpoint. `End` returns live; neither key consumes
an action, gameplay time, or RNG.

`CombatState` separately accumulates actual collections by item identity in
first-acquired order. Revealed treasure, failed/capped-out acquisitions, later
consumption, and report eviction do not inflate or erase those totals. Victory
leaves the encounter open for looting; actual departure snapshots the totals into
`CombatConclusion` before clearing combat state.

GameScene queues ordinary conclusions until the exit/recovery workflow and other
modals finish, then automatically presents `NewCombatSummaryDialog`, a bounded
scrollable readout using `CombatLootDialog`. Fresh Enter/keypad Enter/Escape
dismisses the exact instance. `F4` at primary input reopens the last presented
summary without a gameplay action. Summary retention is scene/session-local;
scene replacement and disposal clear pending, visible, and retained summaries.
The special Doom-mirror finale keeps its existing handoff instead of opening a
collection modal.

The completed-battle summary is a lower, map-centred card: its reference width
is 540 logical pixels at 1280 screen width, scaled with the UI and capped to the
available viewport. Its bottom edge is fixed at 88% of screen height, clamped
inside the viewport. Measured content grows upward to half the viewport height,
then scrolls within a clipped body while the heading and acknowledgement stay
visible. Quantities are right-aligned, long names wrap without shrinking the
body font, and scroll hints appear only for overflow. Empty totals use a compact
`No items collected.` state without the inventory notice. Chest discovery
retains its own reveal layout. The summary's stepped border uses the standard
dialog pixel unit independently of its larger content padding, so its frame
does not become heavier when the map pane expands.

Implementation entry points are `internal/game_state/combat_collection.go`,
`cmd/ultimav/gamescene_combat_report.go`, and
`cmd/ultimav/gamescene_combat_summary.go`; their `_test.go` files cover accounting,
history/attention, and summary ownership respectively.

### Combat Interruption Feedback

`GameState` publishes `CombatEventInterruption` when remembered melee interference
actually rejects a physical source or Cast. It preserves the classic
`"<creature> interferes!"` output and records the exact interrupter, interrupted
actor/clone, and weapon or casting context. Manual Attack, its confirmation
recheck, and Tab's source loop publish through the same boundary; previews never
record attempts. Rejection keeps the existing remaining-source and turn rules.

`recordCombatEvent` admits one Battle Report entry and requests
`SoundCombatInterrupted` once. Both styled and plain report text explain who
interrupted whose action. History uses admission-time names, allegiance and
disclosure; invisible interrupters remain `Unknown`, including concealed detail
coordinates, even after later reveal or slot reuse. Rendering, wrapping and
history navigation never request another sound or mutate gameplay. The manifest's
`combat_interrupted_01` is a named `file: null` slot; assigning its future recording
enables playback through the existing routing after a rebuild.

### Finite Mutation Coverage Matrix

All rows reach the same quantity-stack observation seam, including aliases to
the quantities and paths that do not call `RefreshInventoryDisplay`. Production
entry points below are in `internal/game_state/` unless otherwise qualified.

| Resource | Gain paths | Loss paths |
|---|---|---|
| Food | `tavern_shop.go` meals, rations, and actual charity; `action_get.go` food/stack pickups; `dungeon_interactions.go` chest food; `effect_spells.go` CreateFood; `death_resurrection.go` minimum food restoration | `internal/party_state/inventory.go` meal consumption (surface, dungeon, combat, camping, bed/inn rest); `combat_monster_hit.go` gremlin theft |
| Gold | `action_get.go` stack/loot pickups; `dungeon_interactions.go` chests; `arms_shop.go` sales; `action_bridge_trolls.go` signed transfers | Arms, guild, reagent, horse, ship, healer, inn, and tavern `*_shop.go` payments; `shadowlord_spawn.go` shop/street theft; `guard_extortion.go` tribute/half-gold outcomes; `party_karma.go` donations/conversation payments; `shrine_service.go`; `wishing_well_service.go`; `action_bridge_trolls.go` |
| Keys | `guild_shop.go`; `action_get.go` stacks; `dungeon_interactions.go` chests | `action_jimmy.go` and `dungeon_interactions.go` committed key use; `guard_arrest.go` and `blackthorn_interrogation.go` confiscation; `shadowlord_spawn.go` theft |
| Gems | `guild_shop.go`; `action_get.go` stacks; `dungeon_interactions.go` chests | `action_view.go` surface/dungeon View; `shadowlord_spawn.go` theft |
| Torches | `guild_shop.go`; `action_get.go` stacks/wall torches; `dungeon_interactions.go` chests | `action_ignite.go`; `shadowlord_spawn.go` theft |
| Skull keys | `action_get.go` stacks; conversation grants | `action_use_commits.go` committed Skull Key use |
| All six, common boundaries | `internal/party_state/inventory.go` provision insertion; `cmd/ultimav/game_action_callbacks.go` conversation grants; live `debug_inventory_locker.go` edits | Quantity `Set`/decrement methods, including live `world_setup.go`/`starvation_scenario.go` setup actions |
| Silent/no-change boundaries | Legacy/native hydration, candidate setup, cancellation, rejected/stale transactions, saturated no-op grants | Failed decrements, empty-resource attempts, unchanged `Set`, zero-quantity changes |

A failed **action** may still commit a real cost (for example a used Jimmy key),
and a refused ration order may grant actual charity. Feedback follows the
quantities, not the action's success flag; it must show those real changes.

Regression coverage lives in `internal/party_state/quantity_activity_test.go`,
`internal/ui/mainscreen/provision_changes_test.go`,
`internal/ui/mainscreen/provision_change_panel_test.go`, and
`cmd/ultimav/gamescene_provision_changes_test.go`. These cover gross movements,
caps/no-ops, all six real pickups, real consumption, Britain ration approval and
results, modal update ordering, save hydration, expiry/reset/disposal, rapid
opposing changes, and supported layouts. Rendered composition regressions in
`cmd/ultimav/gamescene_provision_composition_test.go` cover four ordinary entries
over Dungeon View's heading, tight measured/unmeasured modals above a merchant,
and unobstructed merchant feedback. Visual gameplay review should include a
Britain purchase with its result dialog still open, a pickup, consumption, and
rapid repeated changes at minimum resolution.

## Message Output And MessageDialog

`MessageCallbacks.AddRowStr` always writes to the right-side output. If the highest modal is a visible, non-static `MessageDialog`, `GameScene.addRowStr` also mirrors the text into that dialog.

`appendLineBuilderDualOutput` returns a `LineBuilder` that writes styled content to the top `MessageDialog` and plain text to the persistent game output. When no `MessageDialog` is active, it returns the main output builder.

Map and Journal updates use ordinary main Output notices with normal History
retention. Their scene-local queue coalesces pending same-site changes and places
map updates before the coalesced journal notice, flushing only after both
frame-entry and frame-exit command/modal/workflow blockers clear. Reader hints
style keys and descriptions separately. No timed top-centre discovery overlay or
extra rendering layer remains; see [shared notices](WIDGET_COMPONENTS.md#shared-map-and-journal-notices).

Camping uses the current screen callback bridge:

- `CreateAndPushMessageDialog`, a non-returning notification that creates and pushes the dialog
- `HideMessageDialog`
- `PopMessageDialog`
- content and prompt mutation callbacks
- fade-duration query and configuration callbacks

These callbacks operate on the top modal and exist for the current camping integration. Direct feature code should retain an exact `MessageDialog` pointer when it can do so, avoiding ambiguity if another modal is pushed.

## SystemCallbacks UI Boundary

Current UI-facing callback groups include:

- `MessageCallbacks` for persistent text output and prompts
- `UICallbacks` for timed overlays, action feedback, signs, and transient icons
- `ScreenCallbacks` for summaries, display refresh, overlay visibility, and the camping MessageDialog bridge
- `FlowCallbacks` for presentation workflows grouped by consequence, rest, world interaction, observation, Shadowlord, item, and sacred-site concerns
- `TalkCallbacks` for creating and pushing the concrete linear conversation dialog
- `CombatCallbacks.PublishEvent` for presentation-neutral combat history

`UICallbacks.ShowQuestJournalDossier` is mandatory. `ActionQuestJournal`
passes it a detached projection; `GameScene.ShowQuestJournalDossier` prepares
release debouncing, pushes one `QuestJournalDialog`, and removes that exact
instance on dismissal. Browsing never calls time, dungeon-cycle, turn, or
combat-action completion.

`UICallbacks.ShowZStatsDossier` follows the same detached, read-only ownership
boundary across exploration, dungeon, and combat contexts. Its eight-section
header uses measured label widths, responsive typography, and explicit spacing
so static and dynamic labels remain separated at every supported resolution.
Supply details come from the embedded inventory catalog and explain each
provision's source-backed gameplay purpose without changing quantities or
mechanics.

Catalog rows include parenthetical quantities, including zero for visible
catalog entries. Spells show prepared counts; armory rows use compact typography
and explicitly label available and readied counts. Width-aware name fitting
keeps count suffixes visible at supported resolutions without wrapping a
one-line output buffer. Character equipped-slot rows remain uncounted, and
special possessions retain their owned-only filter.

Spell details keep a compact, content-measured summary beside the portrait.
Circle, MP, and prepared count share a summary row; the formula state remains
visible. The detail body begins below both the portrait and summary. Canonical
spell details fit one page at supported resolutions; genuinely oversized bodies
retain Space paging without paging the summary. Single-page details omit the
Space hint and page indicator. Coverage is in
`internal/ui/widgets/ztats_catalog_regression_test.go`.

Mandatory callbacks are supplied as complete named values and checked during initialization. Optional indicators explicitly documented as optional may be omitted; do not generalize that policy to required callbacks or install mandatory callbacks later by mutating the callback set.

## Render Order

`GameScene.Draw` currently composes these major layers:

1. Map image with camera shake.
2. Moongates, item stacks, world quest items, map units, and Avatar; combat selection underlays precede actors, while eligibility brackets follow actor artwork and precede the aiming reticle.
3. Default tile dissolves.
4. Map fade.
5. Over-fade tile dissolves.
6. Screen flash and screen inversion.
7. Debug overlays and combat HUD.
8. Static borders, the right-rail status panel, wind, transient map icons, clipped game output, party summary, and compact provisions.
9. The provision-change strip when the active modal's bounds overlap or are unknown;
   all dialog-stack entries from bottom to top; then the strip
   when the active modal has verified free space for it (or there is no modal).
10. Kapow/combat impact effects and the active projectile on the non-surface, non-dungeon path.
11. Debug text.

On ordinary top-down surface maps, world effects and projectiles instead render
before the terrain visibility mask, debug overlays, transient icons, static UI,
and dialogs. See [final world composition](../systems/VISUAL_EFFECTS_SYSTEM.md#final-composition).
Only the non-surface, non-dungeon path retains impact/projectile layers above
dialogs. Change either order only as an intentional presentation decision backed
by tests.

Screen shake is applied to world presentation, not the static HUD and dialog layers.

## Combat HUD

`CombatHUD` is a non-modal GameScene component and never enters `DialogStack`. Core combat publishes `CombatEvent` records through required `CombatCallbacks.PublishEvent`. GameScene keeps encounter-local history and renders it beside a non-mutating turn forecast.

The forecast must not advance combat turns or consume gameplay RNG. The HUD remains static during battlefield shake while battlefield-local impact labels follow world transforms.

Combat terrain admission is a presentation filter, not a queue filter. A monster
on authored `BlackSquare`/`BrickWallArchway` terrain retains its turn forecast
slot, scheduling, and actions, but the HUD omits its portrait and identifying
detail. By default, `Gameplay.Combat.ShowConcealedTurnOrderSlots` preserves that
blank forecast slot. Players may disable the preference to omit and compact
concealed entries within the same four-action forecast; this does not inspect
beyond the forecast or mutate scheduling. A concealed player-controlled
monster's command prompt uses the neutral `Unknown` label. The same admission
rule suppresses actor-local battlefield decorations; a terrain trigger
immediately restores the ordinary identity presentation without recreating the
actor.

The combat sidebar uses a fixed **45% Turn Order / 55% Events** split via
`combatHUDTurnHeightPercent`, retaining the existing border overlap. This is
separate from the gameplay right rail's unified status row. Current PC names use
`combatEntityName` in both normal and auto-pass presentation, including the full
`CLONE` identity. The compressed NEXT queue retains `combatEntityLabel`
abbreviations. Current actor placement measures wrapped text as well as sprite
height before placing equipment readouts and NEXT.

`appendCombatAttackReadout` consumes the detached core preview through
`GameScene.combatAttackPreview`. Before aiming it lists each attack source and
its own effective range as `(RN)`; during physical aiming it lists only the
current source, the target name (or `Square` without a revealed enemy) with
distance `(DN)`, and a concise presentation-only shot status. The target itself
labels the aiming row because an additional `Aim:` prefix cannot fit the longest
visible names at 1280x720 without reducing the established font size. The core
`CombatShotStatus` values remain unchanged.
It does not collapse mixed sources to their longest range or predict hit chance.
Input-blocked workflows, enemy/auto-pass turns, and unrelated targeting suppress
the assistance readout. The battlefield uses the same authoritative preview for
[eligibility brackets](../systems/VISUAL_EFFECTS_SYSTEM.md#combat-attack-eligibility).
Eligibility brackets use the approved compact within-cell foreground foot-corner
placement, above monster artwork but below the reticle and clear of HP/status
cues, avoiding neighboring sprites' occlusion of an underlay. Composed-render
and exhaustive supported-resolution row/layout/name assertions are in
`cmd/ultimav/gamescene_combat_marker_composition_test.go` and
`cmd/ultimav/gamescene_combat_attack_preview_test.go`. Manual acceptance should
also exercise long names, multiple sources, resize, HiDPI, and crowded fights.

## Unified Status Rail

`ActiveEffectsPanel` owns the existing status band's outer bounds. Construction,
`initializeResizeableVisualElements`, and `syncRightRailLayout` all pass
`rightRail.activeBorder`; `calculateStatusBandLayout` preserves the established
interior padding as one shared row. Actually worn artifacts compact from the
left, while ongoing effects compact from the right. Both groups use the same
measured cell size. A short blue-white separator appears only when both groups
have entries; there are no section captions, inactive wells, or fixed midpoint
rule. Band height, party summary and map viewport stay fixed. The provisions
footer's [four-row layout](#ammunition-summary-and-shortages) takes its additional
height from the message panel, whose transcript has 11 visible lines,
16-character wrapping, clipping, and the same history buffer through resize.

`GameState.GetWornArtifactStatuses` returns a detached fixed array in **Crown,
Amulet of Lord British, Black Badge** order. These are party-level Use toggles,
not ordinary character equipment. The renderer selects only entries whose
authoritative `Worn` flag is true and compacts them in that order. Unowned and
owned-but-unworn artifacts have no slot, subdued preview, or corner marker. An
active toggle still renders when ownership is inconsistent: ownership and worn
state remain separate facts, and presentation neither infers wearing from
quantity nor repairs gameplay state.

`wornArtifactImage` in `cmd/ultimav/active_effects_panel_worn.go` uses Crown sprite
437, royal Amulet sprite 439, and the existing cached
`text.GetBitmapFonts().IBM.GetCharImage(0x1D)` for the Badge. The Badge is the
original 8x8 worn-status font symbol, not a guessed world tile, generic star, or
custom badge image. Character-sheet compatibility is described by
[GFX-0005](../gamefacts/GRAPHICS_FORMATS.md#gfx-0005); active Badge state remains
distinct from ownership as in [CRIME-0001](../gamefacts/CRIME_PRISON.md#crime-0001).
The panel adds no original-data asset. Wrapped sprites are cropped to their
16x16 artwork; glyphs/artwork retain aspect ratio and nearest-neighbor sampling.
Worn and effect entries share the compact Effects scale. Their artwork is
unboxed and does not expand to fill unused space; only the outer rail owns a
complete frame.

The Effects group derives `GetPersistentEffectStatuses` on each draw, in
category order: torch, magical light, **Crown suppression OR one duration
spell**, intoxication, automatic Shadowlord influence, starvation, and active
frigate HMS Cape. Its presentation omits `PersistentEffectCrownSuppression`
because the worn Crown at
the left already communicates that permanent inherent benefit; gameplay still
projects the semantic effect. Other effects retain deterministic order and
compact from the right with stable cell size and spacing. No entry is silently
truncated. Seven is a conservative projection bound, not proof that all seven
categories are reachable together through ordinary gameplay. Tiny defensive
bounds retain projection entries but cannot promise readable artwork below the
configured minimum.

The effect row reserves an additional right gutter, scaled from the content
height, before both groups calculate their shared cell size. Worn artifacts
retain their left anchor. Lone and full effect rows use the same inset edge.

A worn Crown therefore appears exactly once in the rail. This presentation-only
deduplication does not assert that Crown mechanics are identical to In An or
remove `PersistentEffectCrownSuppression` from authority. Donning Crown clears
the duration spell; activating any duration spell removes Crown. Removing or
expiring a replacement never resumes its predecessor. Magical light remains
independent, as do Amulet and Badge; all three artifacts may be worn together.
Normal artifact sentinel 255 is permanent. Wearing does not imply missing
Amulet gameplay protection or add
speculative Amulet/Badge effect entries.

The rail is display-only: no focus, input, popup, numeric timer, callback
registration, gameplay mutation, clock advance, or RNG draw. Effect artwork
retains its tint and animation but has no under-icon accent bar or surrounding
box. Atmosphere remains part of the detached gameplay projection; this panel
does not render its former pulsing underline. Active
`smashed` intoxication uses a magenta potion with danger atmosphere after the
Crown/duration category; decrements retain one entry, zero/fresh-town reset
removes it, and native continuation restores it. The Tavern's separate `drunk`
purchase counter is not presented.

The panel draws in the static rail wherever that rail is present, including
combat, dark first-person dungeons, vehicles, camping, and shrine/Codex views.
It is independent of Avatar visibility, smooth movement, world shake, and the
terrain mask, while intentional whole-rail suppression and modal covering remain.
The Avatar-attached artifact renderer and its sprite-list projection are removed,
not retained as a compatibility option. Activation/removal flashes and feedback,
world quest-item rendering, and the separate transient `ActiveEffectsOverlay`
remain live. Native hydration supplies authority, not persistent UI registrations.

Projection, active-only compaction, pixel sources, absent under-icon accents, shared-cell geometry,
seven-entry containment, context, read-only, and resize assertions are in
`internal/game_state/persistent_effects_test.go`,
`internal/game_state/toggle_item_state_test.go`,
`cmd/ultimav/active_effects_panel_test.go`,
`cmd/ultimav/gamescene_draw_core_test.go`,
`cmd/ultimav/gamescene_output_layout_test.go`, and
`cmd/ultimav/gamescene_initialization_test.go`. Manual acceptance should cover
supported resolutions, HiDPI, aspect-fit containers, darkness, movement/shake,
and modal coverage; execution results belong in the relevant issue.

## Resolution Changes

`Game.Layout` computes the largest positive 16:9 logical surface that fits the
outer container and publishes it through `DisplayManager`. `DrawFinalScreen`
clears the physical target to black and uniformly centers that logical surface;
non-16:9 windows and monitors therefore produce stable bars rather than stretch
or crop game content.

`Game.Update` observes platform display state once per frame and invalidates the
active scene when mode, window, monitor, container, or logical dimensions
change. `Draw` compares the logical size with `lastCheckedResolution` and calls
`initializeResizeableVisualElements` when required. That method rebuilds
GameScene-owned borders and scaled fonts while preserving the main transcript
buffer. It also recalculates one bounds-driven right-rail layout shared by the
status rail, clipped output, and provisions. Long-lived cached-layout
dialogs call their dimension-remembering `ensureLayout()` from update and draw,
rebuilding only presentation-derived resources while retaining workflow state.
Other dialogs that survive a resolution change may expose `Refresh`; ownership
code is responsible for invoking component-specific refresh behavior where
needed.

Use the canonical `display.GetManager()` identity, percentage placement, shared
typography helpers, and actual panel bounds. `GetCurrentSize` is the logical
layout size; use explicit manager APIs when platform window, monitor, or
container dimensions are required. The game loop updates that same manager so
registered callbacks observe changes. Do not call Ebitengine window-size APIs
from core gameplay.

## Integration Rules

- Keep gameplay mutation in `GameState` or workflow steps, not widget drawing code.
- Use required callbacks directly after validated initialization.
- Push every interactive dialog through `DialogStack`.
- Remove exact instances; hidden does not mean unowned.
- Let GameScene update modal and non-blocking lanes exactly once.
- Use `text.Output` for all text rendering.
- Use `ThenSequence` when the parent workflow must await a dialog helper.
- Use `Finally` for guaranteed dialog, audio, input, and visual cleanup.

See [UI Widget System Architecture](UI_WIDGET_SYSTEM.md),
[Widget Components](WIDGET_COMPONENTS.md), and [Modal Dialog System](MODAL_DIALOG_SYSTEM.md).
