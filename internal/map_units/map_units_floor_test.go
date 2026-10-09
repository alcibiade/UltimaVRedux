package map_units

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestGetVehicleAtPositionAndFloorOrNilSelectsExactFloor(t *testing.T) {
	t.Parallel()

	position := references.Position{X: 12, Y: 34}
	otherFloorNPC := NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, 1)
	currentFloorVehicle := NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, 1)
	otherFloorNPC.SetPos(position)
	otherFloorNPC.SetFloor(0)
	otherFloorNPC.SetVisible(true)
	currentFloorVehicle.SetPos(position)
	mapUnits := MapUnits{otherFloorNPC, currentFloorVehicle}

	found := mapUnits.GetVehicleAtPositionAndFloorOrNil(position, 1)
	require.NotNil(t, found)
	assert.Same(t, currentFloorVehicle, found)
}

func TestRemoveNPCAtPositionAndFloorRemovesExactFloor(t *testing.T) {
	t.Parallel()

	position := references.Position{X: 12, Y: 34}
	otherFloorVehicle := NewNPCFriendlyVehiceNewRef(references.FrigateVehicle, position, 0)
	currentFloorVehicle := NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, 1)
	otherFloorVehicle.SetPos(position)
	currentFloorVehicle.SetPos(position)
	mapUnits := MapUnits{otherFloorVehicle, currentFloorVehicle}

	require.True(t, mapUnits.RemoveNPCAtPositionAndFloor(position, 1))
	assert.Len(t, mapUnits, 1)
	assert.Same(t, otherFloorVehicle, mapUnits[0])
	assert.True(t, otherFloorVehicle.IsVisible())
	assert.False(t, currentFloorVehicle.IsVisible())
}

func TestMapUnitLookupMissesDoNotAllocateAndMatchesStayDetached(t *testing.T) {
	position := references.Position{X: 12, Y: 34}
	first := NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, 0)
	second := NewNPCFriendlyVehiceNewRef(references.SkiffVehicle, position, 1)
	first.SetPos(position)
	second.SetPos(position)
	units := MapUnits{first, second}
	miss := references.Position{X: 99, Y: 99}
	require.Zero(t, testing.AllocsPerRun(100, func() {
		if units.GetMapUnitAtPositionOrNil(miss) != nil || units.GetMapUnitAtPositionAndFloorOrNil(miss, 1) != nil {
			t.Fatal("unexpected match")
		}
	}))
	flat := units.GetMapUnitAtPositionOrNil(position)
	floor := units.GetMapUnitAtPositionAndFloorOrNil(position, 1)
	require.NotNil(t, flat)
	require.NotNil(t, floor)
	require.Same(t, first, *flat)
	require.Same(t, second, *floor)
	*flat = second
	require.Same(t, first, units[0], "a returned wrapper must not replace a collection entry")
	units.ReplaceWith(MapUnits{first})
	require.Same(t, second, *floor, "retained matches must not follow collection-slot replacement")
}
