package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	passInitialDelayMs int64 = 300
	passRepeatDelayMs  int64 = 120
)

type explorationPassContext struct {
	state    *game_state.GameState
	location references.Location
	floor    references.FloorNumber
}

type explorationPassRepeat struct {
	active  bool
	nextAt  int64
	context explorationPassContext
}

func (g *GameScene) passInputContext() explorationPassContext {
	location := g.gameState.MapState.PlayerLocation

	return explorationPassContext{state: g.gameState, location: location.Location, floor: location.Floor}
}

func (g *GameScene) explorationPassEligible() bool {
	if g.transientInputOwnedElsewhere() || g.inputBlocked || g.gameState.CombatScene != nil ||
		g.gameState.EventQueueManager.HasActiveWorkflows() || g.gameState.HasPendingOverworldContact() ||
		g.gameState.HasPendingDungeonRoomEncounter() || g.gameState.IsSailing() {
		return false
	}
	switch g.gameState.MapState.PlayerLocation.Location.GetMapType() {
	case references.SmallMapType, references.LargeMapType, references.DungeonMapType:
		return true
	case references.CombatMapType, references.CampingMapType:
		return false
	}

	return false
}

func (g *GameScene) stopExplorationPass() {
	if g.transientCommandInput.pass.active {
		g.keyboard.SuppressUntilRelease(ebiten.KeySpace)
	}
	g.transientCommandInput.pass = explorationPassRepeat{}
}

// explorationPassKey returns one due Pass at most. A late frame never catches up
// old intervals. The accepted key still traverses the normal map command route.
func (g *GameScene) explorationPassKey() (*ebiten.Key, bool) {
	state := &g.transientCommandInput
	if !state.sample.Held[ebiten.KeySpace] {
		state.pass = explorationPassRepeat{}

		return nil, false
	}
	if !g.explorationPassEligible() || state.sample.HasModifiers() {
		g.stopExplorationPass()

		return nil, false
	}
	if g.keyboard.IsSuppressedUntilRelease(ebiten.KeySpace) {
		return nil, true
	}
	now := g.getElapsedMs()
	if !state.pass.active {
		if !g.keyboard.ConsumePressEdge(state.sample, ebiten.KeySpace) {
			return nil, true
		}
		state.pass = explorationPassRepeat{active: true, nextAt: now + passInitialDelayMs, context: g.passInputContext()}
	} else {
		if now < state.pass.nextAt {
			return nil, true
		}
		state.pass.nextAt = now + passRepeatDelayMs
	}
	space := ebiten.KeySpace

	return &space, true
}
