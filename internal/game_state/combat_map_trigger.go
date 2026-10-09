package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// CheckTrap resolves room-trigger check against the active encounter. A
// successful match refreshes the complete projection once, even when both
// destination pairs were invalid.
// See ref:DNG-0013.
func (g *GameState) CheckTrap(position *references.Position) bool {
	if position == nil || g.CombatScene == nil || g.CombatScene.CombatMapRef == nil ||
		!g.CombatScene.CombatMapRef.ResolveDungeonTrigger(*position) {
		return false
	}

	combatMap := g.CombatScene.CombatMapRef
	projection := g.MapState.LayeredMaps.GetLayeredMap(references.CombatMapType, 0)
	if projection == nil {
		panic("combat trigger resolved without a layered-map projection")
	}
	for x := references.Coordinate(0); x < references.XCombatMapTiles; x++ {
		for y := references.Coordinate(0); y < references.YCombatMapTiles; y++ {
			tilePosition := references.Position{X: x, Y: y}
			projection.SetTileByLayer(map_state.MapLayer, &tilePosition, combatMap.GetTileNumber(tilePosition))
		}
	}
	g.SystemCallbacks.Screen.UpdateDisplay()

	return true
}
