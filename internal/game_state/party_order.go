package game_state

// SwapPartyMembers performs an authoritative non-leader party-slot swap.
// The active-player index deliberately remains slot-based.
func (g *GameState) SwapPartyMembers(first, second int) error {
	if err := g.PartyState.SwapMembers(first, second); err != nil {
		return err
	}
	g.SystemCallbacks.Screen.UpdateStatsDisplay()

	return nil
}
