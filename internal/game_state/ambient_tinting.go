package game_state

// AmbientTintDarkness is the cosmetic night strength for the current location.
// It must not be used for visibility, carried-light fuel, or gameplay time.
func (g *GameState) AmbientTintDarkness() float32 {
	if g.MapState.IsUnderworld() && !g.IsInCombat() {
		// REDUX DEVIATION: calendar advancement uses time-independent underground
		// sight, not RGB lighting. Underworld exploration uses Redux's existing
		// full-night color composition while leaving gameplay sight untouched.
		// See ref:LIGHT-0008.
		return 1
	}

	return g.DateTime.GetAmbientTintDarkness()
}
