package game_state

func (g *GameState) ActionViewSmallMap() bool {
	if !g.ActionViewSurface() {
		return false
	}

	// Trigger the presentation-only gem view workflow.
	g.SystemCallbacks.Flow.Observation.ShowGemViewInteraction(nil)

	return true
}

func (g *GameState) ActionViewLargeMap() bool {
	if !g.ActionViewSurface() {
		return false
	}

	// Trigger the presentation-only gem view workflow.
	g.SystemCallbacks.Flow.Observation.ShowGemViewInteraction(nil)

	return true
}

// ActionViewSurface consumes the gem while leaving presentation ownership to the caller.
func (g *GameState) ActionViewSurface() bool {
	if !g.PartyState.Inventory.Provisions.Gems.HasSome() {
		g.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

		return false
	}

	// Consume gem before showing view (authentic behavior)
	g.PartyState.Inventory.Provisions.Gems.DecrementByOne()
	g.AdvanceOrdinarySurfaceCommandMinutes(1)

	return true
}

// ActionViewDungeonMap consumes a gem and returns the current dungeon's symbolic view.
func (g *GameState) ActionViewDungeonMap(token DungeonCommandCycle) (DungeonViewModel, bool) {
	g.ClaimDungeonViewCommand(token)
	if !g.PartyState.Inventory.Provisions.Gems.HasSome() {
		g.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

		return DungeonViewModel{}, false
	}

	model, err := g.CurrentDungeonViewModel()
	if err != nil {
		panic(err)
	}
	g.PartyState.Inventory.Provisions.Gems.DecrementByOne()
	g.SystemCallbacks.Screen.RefreshInventoryDisplay()

	return model, true
}
