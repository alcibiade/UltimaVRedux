package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// DungeonFieldDispelResult reports one authoritative packed-field removal.
type DungeonFieldDispelResult struct {
	Change  DungeonCellChange
	Cleared bool
}

// ClearNearbySceptreBarriers replaces every original 0x70-0x7f terrain
// barrier in the source-ordered 3x3 area centered on the acting character.
func (g *GameState) ClearNearbySceptreBarriers() []references.Position {
	mapType := g.MapState.PlayerLocation.Location.GetMapType()
	origin := g.MapState.PlayerLocation.Position
	if mapType == references.CombatMapType {
		identity := g.CombatScene.GetCurrentTurnIdentity()
		if !identity.IsPartyMember {
			panic("ClearNearbySceptreBarriers requires an active party member")
		}
		origin = g.CombatScene.GetCombatantPosition(identity)
	}
	if mapType != references.LargeMapType && mapType != references.SmallMapType &&
		mapType != references.CombatMapType {
		panic("ClearNearbySceptreBarriers requires a surface, town, or combat map")
	}

	cleared := make([]references.Position, 0, 9)
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			position := references.Position{
				X: origin.X + references.Coordinate(dx),
				Y: origin.Y + references.Coordinate(dy),
			}
			if mapType == references.LargeMapType {
				position = *position.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			} else if g.IsOutOfBounds(position) {
				// REDUX DEVIATION: original map inspection rules aliases invalid town coordinates to
				// town[31][31] and leaves combat bounds undefined. Redux clips them.
				// See ref:QUEST-0003.
				continue
			}

			tile := g.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, position)
			if mapType == references.CombatMapType {
				tile = g.CombatScene.CombatMapRef.GetTileNumber(position)
			}
			if tile < indexes.ShadowlordBoundary1 || tile > indexes.ShadowlordBoundary16 {
				continue
			}
			if len(cleared) == 0 {
				g.CommitCastUseAttempt()
			}

			if mapType == references.CombatMapType {
				g.setCombatTerrain(position, indexes.Grass)
			} else {
				g.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Grass)
			}
			cleared = append(cleared, position)
		}
	}

	return cleared
}

// UseExplorationSceptre applies the safe surface/town Sceptre effect.
func (g *GameState) UseExplorationSceptre() []references.Position {
	// Original surface/town map buffers are not stored in
	// SAVED.GAM. Native snapshots preserve this authoritative in-map mutation.
	cleared := g.ClearNearbySceptreBarriers()
	if len(cleared) == 0 {
		g.CommitCastUseAttempt()
		// REDUX DEVIATION: original item-use rules falls through to field dispelling here and reads
		// inactive dungeon memory outside dungeons. Redux safely has no effect.
		// See ref:QUEST-0003.
		g.SystemCallbacks.Message.AddRowStr("No effect!")
	}

	return cleared
}

// DispelCurrentOrFacingDungeonField removes the current packed field before
// trying the wrapped facing cell, matching field dispelling.
// See ref:QUEST-0003.
func (g *GameState) DispelCurrentOrFacingDungeonField() DungeonFieldDispelResult {
	target, ok := g.currentOrFacingDungeonCell(references.DungeonEnergyField)
	if !ok {
		return DungeonFieldDispelResult{}
	}

	return g.dispelDungeonFieldAt(target.floor, target.position)
}

func (g *GameState) dispelDungeonFieldAt(
	floor references.FloorNumber,
	position references.Position,
) DungeonFieldDispelResult {
	dungeon := g.requireActiveDungeon()
	cell, ok := dungeon.cells.Cell(floor, position)
	if !ok {
		panic("invalid dungeon field target")
	}
	if cell.Kind() != references.DungeonEnergyField {
		return DungeonFieldDispelResult{}
	}

	change, err := g.TransformDungeonCell(floor, position, func(before references.DungeonCell) references.DungeonCell {
		return references.DungeonCell(before.Raw() & 0x08)
	})
	if err != nil {
		panic(err)
	}

	return DungeonFieldDispelResult{Change: change, Cleared: true}
}

// UseDungeonSceptre clears one packed dungeon field without consuming the item.
func (g *GameState) UseDungeonSceptre() DungeonFieldDispelResult {
	g.CommitCastUseAttempt()
	result := g.DispelCurrentOrFacingDungeonField()
	if result.Cleared {
		g.SystemCallbacks.Message.AddRowStr("Field destroyed!")
	} else {
		g.SystemCallbacks.Message.AddRowStr("No effect!")
	}

	return result
}
