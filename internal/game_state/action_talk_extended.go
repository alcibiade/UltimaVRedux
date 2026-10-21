package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// FindNPCInDirection matches conversation targeting: check the adjacent tile, then one
// additional tile only when the otherwise-empty adjacent terrain is talk-through.
// See ref:TALK-0005.
//
//nolint:ireturn // The unit found may be an NPC, enemy or vehicle; callers check which.
func (g *GameState) FindNPCInDirection(startPos references.Position, direction references.Direction, maxDistance int) (*references.Position, map_units.MapUnit) {
	if maxDistance < 1 {
		return nil, nil
	}

	unitAt := func(position references.Position) map_units.MapUnit {
		candidate := g.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(
			position,
			g.MapState.PlayerLocation.Floor,
		)
		if candidate == nil {
			return nil
		}
		unit := *candidate
		if unit.IsEmptyMapUnit() || !unit.IsVisible() || g.isQuestItemNPCHidden(unit) {
			return nil
		}

		return unit
	}

	adjacent := direction.GetNewPositionInDirection(&startPos)
	if adjacent == nil {
		return nil, nil
	}
	if unit := unitAt(*adjacent); unit != nil {
		return adjacent, unit
	}

	if maxDistance < 2 {
		return nil, nil
	}
	intermediateTile := g.GetLayeredMapByCurrentLocation().GetTileTopMapOnlyTile(adjacent)
	if g.MapState.PlayerLocation.Location.GetMapType() == references.SmallMapType {
		intermediateTile = g.GetLayeredMapByCurrentLocation().GetSmallMapTerrainTile(adjacent, g.DateTime.Hour)
	}
	if intermediateTile == nil || !intermediateTile.IsTalkOverable() {
		return nil, nil
	}

	farther := direction.GetNewPositionInDirection(adjacent)
	if farther == nil {
		return nil, nil
	}
	if unit := unitAt(*farther); unit != nil {
		return farther, unit
	}

	return nil, nil
}
