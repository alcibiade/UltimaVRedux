# ADR-001: Centralized Time Inputs over Direct Clock Reads

**Status**: Accepted (implementation updated)
**Date**: 2025-08-25

## Context

Core systems previously read wall time or seeded random generators from wall time. That made animation helpers, AI decisions, and tests depend on when they happened to run.

The production scene has since moved away from the original fixed-step `GameClock` design. The current requirement is central ownership of time inputs, not a specific clock type.

## Decision

### Adapter-Owned Elapsed Time

`GameScene` owns the elapsed-time boundary and publishes it to
`GameState.ElapsedMs` during update. Activation and pause handling must preserve
the accumulated timeline without making core consumers read the platform clock.

Core gameplay consumers use the supplied elapsed value. They do not read wall time directly.

`ElapsedMs` is a shared real-time source for rendering, effects, and event-queue scheduling. It is not a fixed-step gameplay simulation clock and does not by itself provide replay determinism.

The [Time System](../systems/TIME_SYSTEM.md), especially its Journal-pause
contract, owns current publication, freezing, and resume behavior. The adapter
implementation is `cmd/ultimav/gamescene_initialization.go::getElapsedMs`.

### Separate In-Game Time

`GameState.DateTime` is the authoritative Britannian calendar. Turns and gameplay actions advance it explicitly. Real elapsed milliseconds must not advance calendar time implicitly.

### Pure Time Consumers

Animation helpers accept elapsed time and position variation as inputs:

```go
sprite := sprites.GetSpriteIndexWithAnimationBySpriteIndexTick(
	spriteIndex,
	posHash,
	gs.ElapsedMs,
)
```

Tests pass explicit elapsed values, so frame selection remains reproducible even though production presentation uses real elapsed time.

### Controlled Gameplay Randomness

`GameState` owns a Berad RNG `Source`, imported directly from
`github.com/bradhannah/berad-sdk/modules/berad-rng`. The SDK owns the PCG algorithm,
raw-draw accounting, and logarithmic-time restoration. The game retains seed
selection, shared-stream ordering, provider semantics, and save serialization.
Most gameplay code uses the state methods or an injected random-provider interface:

```go
func (g *GameState) SetRandomSeed(seed uint64)
func (g *GameState) RollDice(sides int) int
func (g *GameState) OneInXOdds(odds int) bool
func (g *GameState) RandomIntInRange(min, max int) int
```

Tests reset the seed when they need to reproduce a sequence. AI controllers receive random providers rather than creating wall-time-seeded generators.

Reseeding preserves the shared source pointer. Saves continue to store the seed
and underlying `SourceCalls`, including rejected samples, so hydration resumes
the same next draw. Extraction preserves the existing x/exp/rand sequence and
shuffle order; it does not introduce a new random algorithm or save schema.

Rules that explicitly define a local deterministic seed, such as the day-seeded Shadowlord town schedule, may create an isolated source from that stable input. They must never seed from wall time.

## Consequences

### Positive

- Core logic has no incidental wall-clock dependency.
- Animation functions can be tested with exact elapsed values.
- Gameplay randomness can be reproduced with a known seed.
- Presentation remains smooth because the scene supplies real elapsed time.
- Britannian calendar progression remains action-driven.

### Tradeoffs

- Tests for elapsed-time behavior must set or advance `ElapsedMs` explicitly.
- A production run's presentation timing is not a deterministic replay timeline.
- Callers must distinguish real elapsed time, in-game calendar time, and gameplay turns.
- New gameplay randomness must use `GameState` or an injected provider.

## Current Implementation

- `cmd/ultimav/gamescene.go`: owns scene timing state.
- `cmd/ultimav/gamescene_initialization.go`: exposes the adapter elapsed-time helper.
- `cmd/ultimav/gamescene_update.go`: publishes `GameState.ElapsedMs` and updates consumers.
- `internal/game_state/game_state.go`: owns `DateTime`, `ElapsedMs`, and the centralized random source.
- [Berad RNG contract](https://github.com/bradhannah/berad-sdk/blob/891cf44f456a/modules/berad-rng/docs/API.md):
  shared source mechanics consumed directly by the game.
- `internal/sprites/sprite_animations.go`: selects animation frames from explicit inputs.
- `internal/ai/npc_ai_controller_small_map.go`: defines the small-map RNG provider contract.
- `internal/ai/npc_ai_controller_large_map.go`: requires an injected random provider.

## Validation

- Search core packages for direct wall-clock reads before accepting timing changes.
- Search gameplay code for global or wall-time-seeded random generators.
- Use explicit elapsed values and fixed seeds in tests.
- Verify calendar mutations are driven by turns or explicit debug-domain operations.
