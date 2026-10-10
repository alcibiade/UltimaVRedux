package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSkiffBridgeCollisionUsesEffectiveTerrainDirectionAndOccupancy(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).
		WithPlayerAt(15, 15).WithSystemCallbacks().WithRandomSeed(42).Build()
	units := state.CurrentNPCAIController.GetNpcs()
	units.ReplaceWith(nil)
	bridge := references.Position{X: 15, Y: 14}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &bridge, indexes.TrollBridgeHoriz)
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle,
		state.MapState.PlayerLocation.Position, state.MapState.PlayerLocation.Floor)
	vehicle := state.PartyVehicle.GetVehicleDetails()
	vehicle.SetPartyVehicleDirection(references.Up)
	require.True(t, state.IsPassable(&bridge), "skiff can enter the water channel under the effective bridge tile")
	vehicle.SetPartyVehicleDirection(references.Right)
	assert.False(t, state.IsPassable(&bridge), "cannot row along the bridge deck")
	vehicle.SetPartyVehicleDirection(references.Up)
	ref := references.NPCReference{Location: references.Britain,
		Schedule: references.CreateNPCScheduledFixedOneLocation(bridge, 0)}
	ref.SetKeyIndex(indexes.TownsPerson_KeyIndex)
	occupant := map_units.NewNPCFriendly(ref, 0)
	occupant.SetPos(bridge)
	occupant.SetFloor(0)
	units.ReplaceWith([]map_units.MapUnit{occupant})
	assert.False(t, state.IsPassable(&bridge), "ordinary destination occupancy is still checked")
	occupant.SetFloor(1)
	assert.True(t, state.IsPassable(&bridge), "objects on another floor do not obstruct the channel")
	units.ReplaceWith(nil)
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &bridge, indexes.Grass)
	assert.False(t, state.IsPassable(&bridge), "allowing bridges must not allow landing on grass")
}
