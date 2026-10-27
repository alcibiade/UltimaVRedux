package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ActionGatherLargeMap gathers wood or stone from the adjacent overworld tile.
// REDUX DEVIATION: CRAFT-0002: Redux-only command; tool harvests thin forests and fell trees.
func (g *GameState) ActionGatherLargeMap(direction references.Direction) bool {
	if !g.IsCraftingEnabled() {
		g.SystemCallbacks.Message.AddRowStr("What?")

		return false
	}
	if !g.isOnOverworldSurface() || g.PartyVehicle.GetVehicleDetails().VehicleType != references.NoPartyVehicle {
		g.SystemCallbacks.Message.AddRowStr("Not here!")

		return false
	}
	target := direction.GetNewPositionInDirection(&g.MapState.PlayerLocation.Position)
	if target == nil {
		g.SystemCallbacks.Message.AddRowStr("Nothing to gather!")

		return false
	}
	position := *target.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
	overworld := g.overworldMap()
	terrain := overworld.SpriteIndexByLayer(map_state.MapLayer, position)
	source, ok := references.GatherSourceForTile(terrain)
	if !ok || g.ObjectPresentAt(&position) {
		g.SystemCallbacks.Message.AddRowStr("Nothing to gather!")

		return false
	}

	inventory := &g.PartyState.Inventory
	hasTool := inventory.Tools.HasSome(source.Tool)
	defer g.advanceCraftingLabor(references.GatherMinutes)
	if !hasTool {
		if !source.BareHands {
			g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("You need a %s!", source.Tool.Name()))

			return false
		}
		if !g.OneInXOdds(references.GatherBareHandOdds) {
			g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("You find no %s.", source.Material.Name()))

			return false
		}
		inventory.Materials.IncrementBy(source.Material, 1)
		g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("You gather 1 %s by hand.", source.Material.Name()))

		return true
	}

	quantity := g.RandomIntInRange(source.MinYield, source.MaxYield)
	inventory.Materials.IncrementBy(source.Material, uint16(quantity)) //nolint:gosec // Yields are small positive tunables.
	if source.Depletes {
		overworld.SetTileByLayer(map_state.MapLayer, &position, source.After)
	}
	g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("You gather %d %s.", quantity, source.Material.Name()))
	g.SystemCallbacks.Audio.Play(SoundPushObject).Start()

	return true
}
