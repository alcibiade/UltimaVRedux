package game_state

// ActionTalkLargeMap handles Talk command on the overworld map.
// Per original Ultima V: Talk on overworld always returns "Funny, no response!"
func (g *GameState) ActionTalkLargeMap() bool {
	g.SystemCallbacks.Message.AddRowStr("Talk-Funny, no response!")
	g.AdvanceOrdinarySurfaceCommandMinutes(1)

	return true
}
