package main

import (
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

// combatPresentationTiming is an immutable action snapshot. Authored durations
// enter this policy once; gameplay clocks, reading time and audio never do.
// Its zero value is the authored 1x policy.
type combatPresentationTiming struct {
	factor float64
}

func newCombatPresentationTiming(factor float64) combatPresentationTiming {
	// Detached fixtures and the zero policy deliberately mean authored 1x.
	// Explicit preferences are validated by config before reaching this boundary.
	if factor == 0 {
		return combatPresentationTiming{}
	}
	if math.IsNaN(factor) || math.IsInf(factor, 0) || factor < 0.5 || factor > 4 || math.Mod(factor, 0.5) != 0 {
		panic("invalid combat presentation speed factor")
	}

	return combatPresentationTiming{factor: factor}
}

func (timing combatPresentationTiming) speed() float64 {
	if timing.factor == 0 {
		return 1
	}

	return timing.factor
}

func (timing combatPresentationTiming) duration(base uint64) uint64 {
	if base == 0 || timing.speed() == 1 {
		return base
	}
	scaled := math.Round(float64(base) / timing.speed())
	if scaled >= float64(uint64(math.MaxUint64)) {
		return math.MaxUint64
	}

	return max(uint64(scaled), 1)
}

func (timing combatPresentationTiming) durationMs(base int64) int64 {
	if base <= 0 || timing.speed() == 1 {
		return base
	}
	scaled := math.Round(float64(base) / timing.speed())
	if scaled >= float64(math.MaxInt64) {
		return math.MaxInt64
	}

	return max(int64(scaled), 1)
}

func (timing combatPresentationTiming) seconds(baseMs int64) float64 {
	return float64(timing.durationMs(baseMs)) / 1000
}

func (g *GameScene) captureCombatTiming() combatPresentationTiming {
	if g.combatResolutionTiming != nil {
		return *g.combatResolutionTiming
	}

	return newCombatPresentationTiming(g.gameConfig.GetCombatSpeedFactor())
}

// Shared surface/dungeon helpers must explicitly select combat presentation.
func (g *GameScene) capturePresentationTiming(combat bool) combatPresentationTiming {
	if combat {
		return g.captureCombatTiming()
	}

	return combatPresentationTiming{}
}

// Only synchronous resolution/publication may install this context. Deferred
// work retains the value in its own closure; preemption cannot leak a live rate.
func (g *GameScene) withCombatTiming(timing combatPresentationTiming, resolve func()) {
	defer g.combatTimingScope(timing)()
	resolve()
}

func (g *GameScene) combatTimingScope(timing combatPresentationTiming) func() {
	previous := g.combatResolutionTiming
	g.combatResolutionTiming = &timing

	return func() { g.combatResolutionTiming = previous }
}

func (g *GameScene) combatTimingStep(timing combatPresentationTiming, resolve func()) func() {
	return func() { g.withCombatTiming(timing, resolve) }
}

type combatAttackTimingSnapshot struct {
	encounter *game_state.CombatState
	targeting *game_state.CombatAttackTargeting
	timing    combatPresentationTiming
}

// Manual multi-source attacks retain their committed first-source rate even
// while the player aims another source. Identity guards discard stale snapshots.
func (g *GameScene) captureCombatAttackTiming() combatPresentationTiming {
	combat := g.gameState.CombatScene
	if combat == nil || combat.AttackTargeting == nil {
		return g.captureCombatTiming()
	}
	if g.combatAttackTiming.encounter != combat || g.combatAttackTiming.targeting != combat.AttackTargeting {
		g.combatAttackTiming = combatAttackTimingSnapshot{
			encounter: combat, targeting: combat.AttackTargeting, timing: g.captureCombatTiming(),
		}
	}

	return g.combatAttackTiming.timing
}
