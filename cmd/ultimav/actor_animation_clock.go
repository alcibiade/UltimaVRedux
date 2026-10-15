package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
)

// timeStopAnimationDivisor is the chosen Redux body-only near-still treatment.
// It does not scale gameplay, lighting, feedback, or environmental clocks.
const timeStopAnimationDivisor int64 = 20

// actorAnimationClock retains twentieths of a millisecond across every rate
// boundary. Even 1ms updates advance it; resuming normal time never drops the
// fractional remainder or returns to the unrelated global animation phase.
type actorAnimationClock struct {
	sourceMs    int64
	phaseMs     int64
	remainder   int64
	slow        bool
	initialized bool
}

func (clock *actorAnimationClock) update(now int64, slow bool) {
	now = max(now, 0)
	if !clock.initialized || now < clock.sourceMs {
		*clock = actorAnimationClock{sourceMs: now, phaseMs: now, remainder: 0, slow: slow, initialized: true}

		return
	}
	delta := now - clock.sourceMs
	if clock.slow {
		fraction := delta%timeStopAnimationDivisor + clock.remainder
		clock.phaseMs += delta/timeStopAnimationDivisor + fraction/timeStopAnimationDivisor
		clock.remainder = fraction % timeStopAnimationDivisor
	} else {
		clock.phaseMs += delta
	}
	clock.sourceMs = now
	clock.slow = slow
}

type actorAnimationClockEntry struct {
	clock actorAnimationClock
	live  bool
}

type actorAnimationClockSet map[game_state.ActorAnimationIdentity]actorAnimationClockEntry

// updateActorAnimationClocks runs at update and committed-state refresh
// boundaries. Visibility never controls membership, so viewport departure does
// not reset a surviving actor. Removed lifetimes release their retained pointers.
func (g *GameScene) updateActorAnimationClocks() {
	if g.disposed || g.gameState == nil {
		g.clearActorAnimationClocks()

		return
	}
	if g.actorAnimationClocks == nil {
		g.actorAnimationClocks = make(actorAnimationClockSet)
	}
	for key, entry := range g.actorAnimationClocks {
		entry.live = false
		g.actorAnimationClocks[key] = entry
	}
	g.gameState.ForEachActorAnimation(func(key game_state.ActorAnimationIdentity, slow bool) {
		entry := g.actorAnimationClocks[key]
		entry.clock.update(g.gameState.ElapsedMs, slow)
		entry.live = true
		g.actorAnimationClocks[key] = entry
	})
	for key, entry := range g.actorAnimationClocks {
		if !entry.live {
			delete(g.actorAnimationClocks, key)
		}
	}
}

func (g *GameScene) clearActorAnimationClocks() {
	g.actorAnimationClocks = nil
}

// actorAnimationTime is read-only. Draw consumes the last settled update value;
// a newly constructed scene uses the current epoch until its first update.
func (g *GameScene) actorAnimationTime(key game_state.ActorAnimationIdentity, valid bool) int64 {
	if valid {
		if entry, found := g.actorAnimationClocks[key]; found {
			return entry.clock.phaseMs
		}
	}

	return g.gameState.ElapsedMs
}

func (g *GameScene) combatActorAnimationTime(
	identity game_state.CombatTurnIdentity,
	expectedID game_state.CombatantID,
) int64 {
	key, _, valid := g.gameState.CombatActorAnimation(identity, expectedID)

	return g.actorAnimationTime(key, valid)
}

func (g *GameScene) surfaceActorAnimationTime(unit map_units.MapUnit, ordinaryTime int64) int64 {
	key, _, valid := g.gameState.SurfaceActorAnimation(unit)
	if !valid {
		return ordinaryTime
	}

	return g.actorAnimationTime(key, true)
}

func (g *GameScene) dungeonActorAnimationTime() int64 {
	key, _, valid := g.gameState.DungeonActorAnimation()

	return g.actorAnimationTime(key, valid)
}
