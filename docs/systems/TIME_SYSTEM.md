# Time System

The game has two separate time domains. They must not be conflated: elapsed presentation time schedules updates and effects, while Britannian calendar time advances only through gameplay actions.

## Central Elapsed Time

`GameState.ElapsedMs` is the shared elapsed-millisecond value consumed by event queues, gameplay-adjacent animations, wind cadence, overlays, and other time-based systems. `GameScene` is the platform adapter that samples elapsed wall time and copies it into `GameState` during update.

Core systems consume the supplied value rather than sampling wall time themselves. Tests can therefore set `ElapsedMs` directly and advance event queues deterministically. Elapsed milliseconds do not advance the Britannian date or charge an action cost.

Purely cosmetic presentation and resource-lifecycle code may sample wall time directly when it cannot affect gameplay state, ordering, or randomness. Current examples include dialog fades and cursor blink, audio fades and cleanup, transient render-only effects, and position-offset sprite overlays. Gameplay timing must not use those clocks.

### Elapsed Time And Shared RNG Scope

[ADR-001](../ADR/ADR-001-central-gameclock.md) specifies centralized time inputs,
not fixed-step or seed-only replay. `GameScene.updateWindAndEffects` publishes
`ElapsedMs`, advances wind, runs queues, then updates effects. Wind's
`AdvanceThrough` processes every due 55 ms tick using the **shared gameplay RNG**,
including during combat presentation waits. There is no isolated wind stream.
Longer actual waits can therefore admit more wind ticks and change later rolls,
although waiting alone adds no Britannian calendar minutes.

Combat signature effects record `StartedMs` at event admission. Gate phases
each measure their own admission against `GameState.ElapsedMs`; a hitch cannot
skip a phase or let the workflow expire it before effects update. They never inherit the preceding
frame's age. Their no-extra-RNG contract is evaluated at the same supplied
elapsed/action timeline, accounting for due wind ticks and authoritative action
draws; it does not promise duration-independent replay.
`cmd/ultimav/gamescene_combat_signature_timing_test.go` exercises the production
wind/queue/effect update ordering after a 700 ms hitch, retaining a newly admitted
gate's 125/600/125 ms phases and original turn token for Dragon and Daemon
summoners. Its oracle includes
`WindSystem.AdvanceThrough`, gate creation draws and next-RNG equality.

### Journal Pause

`GameScene` owns one Journal session independently of top-widget focus. Opening
latches the last published `GameState.ElapsedMs`; `getElapsedMs` returns that
frozen value until the owner closes. `GameScene.Update` branches before finale,
wind, queues, automatic doors, and gameplay routing, updating only safe modal/
audio presentation and the cosmetic notice. Skipping every gameplay queue is
necessary: freezing deadlines alone would still allow ready callbacks to run.

Tab/search changes never release the pause. Dismissal rebases the scene elapsed
epoch to the frozen value, removes the exact dialog, and returns for that whole
frame, so neither wall-time catch-up nor closing-key gameplay occurs. It does
not alter wind phase, RNG, queue deadlines, or reactivate the scene. Disposal
cleans session ownership without acknowledging departing checkpoint markers.
The Draw FPS limiter uses independent cosmetic time so the frozen gameplay
clock cannot freeze Journal rendering. This policy is Journal-specific, not a
change to every modal or a restriction on deliberate developer Workbench actions.

Implementation: `cmd/ultimav/gamescene_quest_journal.go`,
`cmd/ultimav/gamescene_input.go`, `cmd/ultimav/gamescene_initialization.go`, and
`cmd/ultimav/gamescene_draw_core.go`. Injected-clock
assertions are in `cmd/ultimav/gamescene_quest_journal_test.go`.

## Britannian Calendar

`datetime.UltimaDate` stores:

```go
type UltimaDate struct {
    Year   uint16
    Month  byte
    Day    byte
    Hour   byte
    Minute byte
    Turn   uint32
}
```

The Britannian calendar uses 13 months, 28 days per month (364 days per year), 24 hours, and 60 minutes. Month 12 advances to month 13; month 13 advances to month 1 of the next year. `Advance(minutes)` increments `Turn` once per invocation, applies the calendar rollover logic, and clamps a single request to nine hours. `SetTimeOfDay` moves to one of the named anchors: Morning 05:00, Noon 12:00, Evening 17:00, Midnight 00:00, or Dusk 20:00.

Calendar consumers include NPC schedules, food checks, visibility, lighthouse behavior, moon phases, moongates, and date/time display.

GameState applies temporal modifiers before normalizing to the calendar's
maximum interval. Calendar, carried-light fuel, camp cooldown and returned elapsed
minutes all use that same accepted duration. Positive requests replace the prior
hour observation even under Time Stop; zero requests leave it alone. Player updates
observe one current/previous-hour pair, never replay a backlog of elapsed hours.
Rest can overwrite an unprocessed crossing; sailing interruption returns to command
input without an extra update. See [PROV-0002](../gamefacts/PROVISIONS_CONSUMPTION.md#prov-0002).

## Action Costs

Authentic action cost is a gameplay rule, not a property of `ElapsedMs`.

- `GameState.AdvanceGameplayMinutes(minutes)` applies an explicit command or workflow cost through the authoritative calendar domain.
- Normal surface routing calls `CompleteSurfaceCommandCycle` or `CompleteSurfaceMovementCycle`: ordinary completion advances the calendar, runs player-turn effects, then runs the world-actor phase. `FinishTurn` delegates surface work to that seam when needed and remains a compatibility/combat boundary elsewhere.
- Small-map movement costs one minute.
- Large-map completion applies terrain or vehicle movement cost. Raised-sail wind and HMS Cape timing belongs to the automatic sailing wait cadence; the eventual accepted movement receives ordinary outdoor completion.
- Combat advances one calendar minute at each tenth selected actor action through the DEX turn queue.
- Camping advances five game minutes per camping cycle; its 200 ms presentation delay is independent of that cost.
- A complete merchant Talk owns one minute, including repeated family actions
  or cancellation. Inn Rest additionally owns its overnight steps; the normal
  Talk minute is still applied during final workflow cleanup after the Inn
  result is acknowledged. An accepted Redux harbour escort is a separate
  verified transition that adds three minutes, for four total; unavailable or
  cancelled travel adds nothing and never affects the vessel purchase.
- `ProcessTurnEffects` owns a complete non-calendar player turn. Ship repair
  instead calls its narrow world-actor phase before each explicit five-minute
  advance so ordinary hazards and durations are not repeated. Those special
  advances also skip meal processing; the one ordinary completion owns the
  resulting food/starvation check.

Individual action methods may charge time directly and therefore must not also receive an unconditional `FinishTurn` from the input layer. The authoritative original-game cost matrix and known deviations belong in [TIME_COSTS.md](../gamefacts/TIME_COSTS.md); this document describes how costs are applied.

### Inn Calendar Maintenance And Overnight Time

Every calendar advance through `advanceElapsedGameplayMinutes` compares the
before and after Britannian month ordinal. For each crossed 28-day month
boundary, all 16 roster records with `MonthsAtInn < 25` increment once; values
25 and above remain unchanged. This global maintenance applies to ordinary
gameplay time and Inn time, including year rollover. It deliberately mirrors
the original all-roster update rather than filtering to companions currently
stored at an Inn.

A successful `RestAtInn` uses `advanceInnRestMinutes`, a dedicated elapsed-time
path with this exact order:

```text
successful payment and Faulinei resolution
-> move to the Inn sleep position and mark Good members Sleeping
-> clear the active duration spell
-> advance 5 minutes twelve times
-> while hour != 6: process eligible regeneration rings, then advance 9 minutes
-> restore morning HP/MP/status and move to the wake position
-> later, complete the owning Talk workflow's one-minute cleanup
```

Each 5- or 9-minute invocation advances the calendar/turn counter, global Inn
month counters, lighting, magic-light minutes, and hourly `LastCamped`
maintenance. It intentionally skips ordinary hourly food consumption and does
not apply Quickness reduction or Time Stop suppression. Cancellation, refusal,
Leave, Pick Up, and Preview never use this overnight path and preserve the
active duration spell. If Rest leaves the whole active party Dead, the workflow
hands off to resurrection after Inn cleanup and skips the separate Talk minute.

### Dungeon Command Cycles

Dungeon exploration does not use `FinishTurn` directly. `GameState` issues an
opaque command-cycle token when a recognized dungeon command is accepted.
Ordinary commands apply their precharge immediately. Cast and Use reserve a
deferred token and charge it only when a final valid attempt commits, so their
rejection and cancellation are free. Every token is completed exactly once with
one of two modes:

- command-only completion retains the precharge without full dungeon
  end-player processing, as used by facing changes;
- full completion runs the implemented common dungeon end-player phase without
  charging another minute, as used by advance and back-up attempts even when
  blocked. Wandering actors, fields, rooms, and hazards extend that phase in
  their owning dungeon children.

Quickness alternates charged/actor-active and free/actor-inactive cycles. Time
Stop suppresses both the precharge and dungeon actors, while a full completion
still decrements its duration. Tokens are transient and stable save capture is
permitted only between command cycles. Native snapshots retain
`quicknessSkipNext`, so loading resumes the exact next charged/free phase.
They do not store the command token or sequence. Supported versions and promotion
rules belong in the [snapshot contract](../details/SAVE_SNAPSHOT_SCHEMA.md#snapshot-version-compatibility).

Redux begins the cycle when an accepted command key is received rather than
before idle input polling. This avoids retaining a token while the player is
idle. Most accepted commands preserve source-style rejection/cancellation
costs; Cast and Use deliberately defer charging until final confirmation.

## Turn Effects

`ProcessTurnEffects` handles the non-calendar work associated with a completed
turn, including a separable map-specific NPC/enemy phase, environmental hazards,
wake checks, duration effects, toggle items, and party-wipe detection. Standing
hazard mutation completes synchronously; its flash is presentation-only feedback
and may coalesce without changing gameplay.

Normal surface completion advances the owning command's calendar minutes,
runs one ordered player update, then the actor phase. That player update owns
town wake checks before standing hazards, the Codex guardian and random Underworld
earthquake on large maps, poison, observed-hour food/starvation, durations,
toggle items, the noncombat Regeneration Ring sweep, and party-wipe detection.
Completed, rejected, and cancelled surface results enter this seam by default
when `ConsumesNormalCycle` is true. Explicitly free, deferred, still-active, and
map-transition results bypass it. Camping's internal five-minute steps
deliberately bypass it and its owner invokes one final player update.

Slow outdoor terrain is a distinct owner: its extra actor phases run before its
extra calendar advance, followed by ordinary completion. Each extra phase's
direct contact resolves before the next phase. Tactical combat or a gameplay
workflow retains the remaining phases and time costs until the existing
deferred-contact input boundary resumes them. This movement continuation is
transient, rejects native capture while pending, and is cleared on shutdown;
neither the interrupted phase nor its costs are replayed. Living physical retreat
reported as `CombatDefeat` resumes lower outdoor actors and the owed command
tail after restoration and presentation ownership release; actual party death,
terminal handoff, cancellation or invalid origin clears it. The production
keyboard/retreat/summary-dismissal regression in
`cmd/ultimav/gamescene_outdoor_retreat_continuation_test.go` verifies a very-slow
step completes four extra minutes plus two ordinary minutes exactly once on the
subsequent scene update. Blocked outdoor movement has no completion; blocked town
movement still does. Door timers run
before town actor gates even under Time Stop. An expired automatic door timer
remains pending while the tracked doorway is occupied and retries before later
actor phases; an NPC that vacates during a phase permits closure on the next
update. Automatic sailing waits, repair,
camping, and transition suppression retain their separate completion contracts.

Horse/carpet cadence skips alternate actor phases without skipping ordinary
calendar/status work. Town gates riding before Time Stop and Quickness; Space
Pass bypasses only the town riding gate. Outdoors, Time Stop and Quickness gate
before riding, including Pass. Town riding and Quickness phases reset on a new
town visit, not a floor change; outdoor phases survive town visits and foot
travel does not reset riding phase. Native saves persist these independent phases;
see [snapshot schema](../details/SAVE_SNAPSHOT_SCHEMA.md).

Outdoor actor execution follows [ENEMY-0006](../gamefacts/ENEMIES.md#enemy-0006): generation first,
then descending NPC slots with an attack check immediately before each actor's
movement, then cleanup. An attack (including an obstructed fireball) suppresses
all later movement in that phase; later actors still evaluate attacks. An actor
that moves into contact waits for its next eligible phase to attack. Wrapped
nearby actors survive cleanup at map seams.

Sea Serpent/Dragon fireball rolls and adjacent carpet/skiff damage belong to
this same gated phase, including Underworld Pass. Direct contact and ranged
party/hull damage resolve authoritatively in actor order. Detached ranged
results share the pirate projectile workflow's FIFO presentation lane, without
additional actor phases or damage in animation callbacks. Tactical contact
retains the existing pending-encounter handoff at the completion boundary.
On-foot whirlpool contact commits damage in actor order and presents a captured
result. Vehicle contact suspends the descending scan while its owning workflow
performs the hit/transport at fade-out. After fade-in, the same phase resumes
below the saved slot on the current world's controller, without repeating
generation, player updates, or cadence gates. No old-map actor pointers cross
the handoff. Prior attacks continue to suppress movement, but lower-slot
attacks use the resolved position, vehicle and HP. The continuation is owned
by the live workflow and is discarded on cancellation, never run by cleanup.
Reusable transport steps in `internal/game_state/whirlpool.go` install no
finalizer. The outdoor owner composes vortex-audio/fade cleanup and phase
cancellation in one `Finally`, so adding transport steps cannot replace the
continuation's cleanup.
See [outdoor attack rules](../gamefacts/ENEMIES.md#enemy-0006).

Time stop deliberately separates these concerns:

- Turn-owned durations continue to process.
- Calendar advancement is suppressed if Time Stop was active at completion entry.
- Actors check Time Stop after the player update, so its final duration decrement
  permits that completion's actor phase (subject to mounted/Quickness gates).
- The time-stop duration itself still counts down.

Combat field effects and DEX scheduling are handled by combat-specific paths rather than by the small/large-map branch.

## Daylight And Lunar State

- `IsDayLight` is true from 05:00 through 19:59.
- `IsFullDaylight` is true from 06:00 through 19:59.
- `GetVisibilityFactorWithoutTorch` interpolates at hour 5 and hour 20 and returns the supplied nighttime baseline outside daylight.
- `GetCurrentMoonPhase` is the compatibility display helper and returns Trammel's phase; `GetTrammelMoonPhase` and `GetFeluccaMoonPhase` expose each moon explicitly.
- `GetMoongateMoonPhase` follows routing rules by selecting Trammel before noon and Felucca from noon onward.

## Key Files

- `internal/datetime/ultima_date.go`
- `internal/game_state/turn.go`
- `internal/game_state/inn_shop.go`
- `internal/game_state/dungeon_command_cycle.go`
- `internal/game_state/game_state.go`
- `cmd/ultimav/gamescene_update.go`
- `cmd/ultimav/gamescene_callbacks.go`
- `docs/gamefacts/TIME_COSTS.md`
