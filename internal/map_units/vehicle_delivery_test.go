package map_units

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVehiclePendingDeliverySnapshotAndActivation(t *testing.T) {
	t.Parallel()

	vehicle := NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		references.Position{X: 0x27, Y: 0xdd},
		references.MainFloor,
	)
	vehicle.GetVehicleDetails().MarkPendingDelivery()
	assert.True(t, vehicle.GetVehicleDetails().IsPendingDelivery())
	assert.True(t, vehicle.GetVehicleDetails().Snapshot().PendingDelivery)

	units := make(MapUnits, 0, MaximumNpcsPerMap)
	stored := units.InsertVehicle(*vehicle)
	require.NotNil(t, stored)
	assert.NotSame(t, vehicle, stored)
	assert.True(t, stored.GetVehicleDetails().IsPendingDelivery())

	activated := units.ActivatePendingVehicleDelivery()
	assert.Same(t, stored, activated)
	assert.False(t, stored.GetVehicleDetails().IsPendingDelivery())
	assert.Nil(t, units.ActivatePendingVehicleDelivery(), "delivery must activate only once")
}

func TestMapUnitsVehicleInsertionCapacityAndCompatibility(t *testing.T) {
	t.Parallel()

	units := make(MapUnits, MaximumNpcsPerMap-1)
	for index := range units {
		units[index] = NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, index)
	}
	assert.True(t, units.HasCapacity())

	vehicle := NewNPCFriendlyVehiceNewRef(
		references.SkiffVehicle,
		references.Position{X: 10, Y: 11},
		references.MainFloor,
	)
	stored := units.InsertVehicle(*vehicle)
	require.NotNil(t, stored)
	assert.False(t, units.HasCapacity())
	assert.Same(t, stored, units[len(units)-1])
	assert.False(t, units.AddVehicle(*vehicle), "legacy wrapper must report a full controller")

	var nilUnits *MapUnits
	assert.False(t, nilUnits.HasCapacity())
	assert.Nil(t, nilUnits.InsertVehicle(*vehicle))
}
