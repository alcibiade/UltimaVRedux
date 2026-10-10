package game_state

// ActionPass handles the Pass command - player chooses to do nothing this turn
func (g *GameState) ActionPass() bool {
	g.SystemCallbacks.Message.ShowCommandPrompt("Pass")
	g.AdvanceOrdinarySurfaceCommandMinutes(1)

	return true
}
