package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameState) isSmallMapDoorOccupied(position references.Position, floor references.FloorNumber) bool {
	player := g.MapState.PlayerLocation
	if player.Location.GetMapType() != references.SmallMapType {
		return false
	}
	if player.Floor == floor && player.Position == position {
		return true
	}
	if g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil {
		return false
	}
	for _, unit := range *g.CurrentNPCAIController.GetNpcs() {
		if unit.IsVisible() && !unit.IsEmptyMapUnit() && unit.Floor() == floor && unit.Pos() == position {
			return true
		}
	}

	return false
}
