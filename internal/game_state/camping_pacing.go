package game_state

import "github.com/bradhannah/UltimaVRedux/internal/camping"

// RequestCampingAcceleration latches pacing only during the current sleep loop.
// Completion, calendar advancement and random decisions remain with the workflow.
func (g *GameState) RequestCampingAcceleration() bool {
	return g.CampingScene.RequestAcceleration()
}

func campingCycleDelay(scene *camping.CampingScene) int64 {
	if scene.IsAccelerated() {
		return CampingFastCycleDelayMs
	}

	return CampingCycleDelayMs
}
