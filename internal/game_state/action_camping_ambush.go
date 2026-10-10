package game_state

// CheckForAmbushDuringHour checks if an ambush should occur during camping
// Returns true based on:
// - 1/64 chance per hour (or forced)
func (g *GameState) CheckForAmbushDuringHour(forceAmbush bool) bool {
	// Force ambush for debug/testing
	if forceAmbush {
		return true
	}

	// 1 in 64 chance per hour
	return g.OneInXOdds(64)
}
