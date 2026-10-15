package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// DungeonInjuryOwner ties transient feedback to its game, dungeon visit and floor.
// It is not persisted and does not constrain movement or command completion.
type DungeonInjuryOwner struct {
	state   *GameState
	dungeon *DungeonState
	floor   references.FloorNumber
}

// CaptureDungeonInjuryOwner captures the current authoritative dungeon visit.
func (g *GameState) CaptureDungeonInjuryOwner() DungeonInjuryOwner {
	return DungeonInjuryOwner{state: g, dungeon: g.requireActiveDungeon(), floor: g.MapState.PlayerLocation.Floor}
}

// MatchesDungeonInjuryOwner prevents delayed feedback leaking into another visit.
func (g *GameState) MatchesDungeonInjuryOwner(owner DungeonInjuryOwner) bool {
	player := g.MapState.PlayerLocation

	return owner.state == g && owner.dungeon != nil && owner.dungeon == g.activeDungeon &&
		g.CombatScene == nil && player.Location == owner.dungeon.location && player.Floor == owner.floor
}
