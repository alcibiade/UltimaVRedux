package game_state

// ActionZtats presents a detached dossier without advancing gameplay.
// REDUX DEVIATION: Original command rules routes Z-Stats through normal exploration
// command processing, and original combat scheduler ends the acting character's action.
// Redux keeps this read-only dossier non-consuming in every context.
// See ref:CMD-0002.
func (g *GameState) ActionZtats() {
	g.SystemCallbacks.UI.ShowZStatsDossier(g.CurrentZStatsDossier())
}
