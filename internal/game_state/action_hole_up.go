package game_state

// action_hole_up.go - Entry points for Hole Up & Camp command
// Camping (large map): see action_hole_up_camping.go
// Bed rest (small map): see action_hole_up_bed.go

func (g *GameState) ActionHoleUpCombatMap() bool {
	// Cannot camp during active combat
	g.SystemCallbacks.Message.AddRowStr("Not here!")

	return false
}

// ActionHoleUpDungeonMap claims the retained full dungeon command cycle.
func (g *GameState) ActionHoleUpDungeonMap(token DungeonCommandCycle) {
	g.ClaimDungeonFullCommand(token)
}
