package game_state

import "github.com/bradhannah/UltimaVRedux/internal/party_state"

// ActionNewOrderSmallMap checks if the New Order command is allowed on small maps.
// Returns true if two non-leader companions can be swapped, false otherwise.
// Original Ultima V: original command rules (party reordering command)
// See ref:CMD-0001.
func (g *GameState) ActionNewOrderSmallMap() bool {
	return g.canReorderParty()
}

// ActionNewOrderLargeMap checks if the New Order command is allowed on large maps.
// Returns true if two non-leader companions can be swapped, false otherwise.
func (g *GameState) ActionNewOrderLargeMap() bool {
	return g.canReorderParty()
}

// ActionNewOrderDungeonMap checks if the New Order command is allowed in dungeons.
// Returns true if two non-leader companions can be swapped, false otherwise.
func (g *GameState) ActionNewOrderDungeonMap() bool {
	return g.canReorderParty()
}

// canReorderParty checks if the party can complete a meaningful non-leader swap.
// Outputs error message if party cannot be reordered.
func (g *GameState) canReorderParty() bool {
	companionCount := 0
	for index := 1; index < party_state.MaxActiveParty; index++ {
		character := g.PartyState.GetCharacter(index)
		if character != nil && character.PartyMap == party_state.PartyMapActive && character.GetNameAsString() != "" {
			companionCount++
		}
	}
	// REDUX DEVIATION: Original command rules permit a meaningless self-swap with
	// only one companion. Redux opens New Order only when a real swap can complete.
	// See ref:CMD-0001.
	if companionCount < 2 {
		g.SystemCallbacks.Message.AddRowStr("Cannot reorder party!")

		return false
	}

	return true
}
