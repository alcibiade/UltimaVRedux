package ai

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNPCOccupiedDoorsOriginalMovementAndPlacement(t *testing.T) {
	refs := helpers.LoadTestGameReferences(t, helpers.LoadTestConfiguration(t))
	for _, closed := range []indexes.SpriteIndex{
		indexes.RegularDoor, indexes.RegularDoorView, indexes.LockedDoor,
		indexes.LockedDoorView, indexes.MagicLockDoor, indexes.MagicLockDoorWithView,
	} {
		t.Run(fmt.Sprintf("tile%d", closed), func(t *testing.T) {
			start := references.Position{X: 10, Y: 10}
			door := start.Right()
			rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0}}
			controller, npc, _ := newOriginalScheduleTestController(t, rng, references.Wander3, start)
			controller.mapState.PlayerLocation.Location = references.Britain
			controller.mapState.LayeredMaps.ResetAndCreateSmallMap(
				refs.LocationReferences.GetLocationReference(references.Britain), controller.tileRefs, 11, 11)
			controller.mapState.LayeredMaps.SetDoorOccupancyProvider(func(location references.Location, position references.Position, floor references.FloorNumber) bool {
				for _, unit := range controller.mapUnits {
					if location == references.Britain && unit.IsVisible() && !unit.IsEmptyMapUnit() && unit.Pos() == position && unit.Floor() == floor {
						return true
					}
				}
				return false
			})
			layer := controller.mapState.GetLayeredMapByCurrentLocation()
			layer.SetTileByLayer(map_state.MapLayer, &door, closed)
			// Regular doors allow ordinary wandering. Locked variants only permit
			// the original exact schedule-destination exception (ref:NPC-0004).
			if !closed.IsUnlockedDoor() {
				npc.NPCReference.Schedule = references.CreateNPCScheduledFixedOneLocation(door, 0)
			}
			controller.randomStepOriginal(npc, start, 3)
			require.Equal(t, door, npc.Pos())
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.GetSmallMapTerrainTile(&door, 0).Index)
			assert.Equal(t, closed, layer.GetTileTopMapOnlyTile(&door).Index, "persistent lock remains separate")
			assert.Equal(t, closed, controller.npcTerrainAt(door).Index, "terrain eligibility is not broadened")
			assert.True(t, controller.mapState.IsPositionOpenDoor(&door))
			assert.False(t, controller.mapState.HasTransientDoor())
			assert.Empty(t, rng.values)

			npc.SetPos(start)
			assert.Equal(t, closed, layer.GetSmallMapTerrainTile(&door, 0).Index, "departure is immediate without sprite refresh")
			npc.NPCReference.Schedule = references.CreateNPCScheduledFixedOneLocation(door, 0)
			controller.HardResetNPCs(true)
			require.Equal(t, door, npc.Pos())
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.GetSmallMapTerrainTile(&door, 0).Index,
				"schedule placement establishes openness before presentation")
			npc.SetPos(start)
			npcRefs := []references.NPCReference{npc.NPCReference}
			controller.slr = references.NewSingleSmallMapReference(references.Britain, &npcRefs, refs.DataOvl)
			controller.PopulateMapFirstLoad()
			require.Len(t, controller.mapUnits, 1)
			assert.Equal(t, door, controller.mapUnits[0].Pos())
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.GetSmallMapTerrainTile(&door, 0).Index,
				"initial population uses newly generated authoritative actors")
		})
	}
}
