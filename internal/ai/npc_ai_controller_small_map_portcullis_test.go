package ai

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNPCPortcullisPlanningAndCachedStep(t *testing.T) {
	start, gate, goal := references.Position{X: 8, Y: 11}, references.Position{X: 8, Y: 10}, references.Position{X: 8, Y: 8}
	rng := &scriptedOriginalScheduleRNG{t: t, values: []int{0}}
	controller, npc, layer := newOriginalScheduleTestController(t, rng, references.Fixed, start)
	for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
		position := references.Position{X: x, Y: gate.Y}
		layer.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
	}
	arch := gate.Up()
	layer.SetTileByLayer(map_state.MapLayer, &arch, indexes.BrickWallArchway)
	layer.SetTileByLayer(map_state.MapLayer, &gate, indexes.BrickFloor)
	for slot := range npc.NPCReference.Schedule.X {
		npc.NPCReference.Schedule.X[slot], npc.NPCReference.Schedule.Y[slot] = byte(goal.X), byte(goal.Y)
	}
	controller.dateTime.Hour, controller.dateTime.Minute = 19, 59
	require.True(t, controller.createFreshPathToScheduledLocation(npc))
	require.True(t, controller.canOriginalNPCMoveTo(npc, gate, npcMovementSchedule))
	assert.True(t, controller.canHorseMoveTo(npc, gate))
	require.Equal(t, gate, npc.MapUnitDetails().CurrentPath[0])
	controller.dateTime.Advance(1)
	require.Equal(t, byte(20), controller.dateTime.Hour)
	assert.Equal(t, indexes.SpriteIndex(indexes.Portcullis), controller.npcTerrainAt(gate).Index)
	assert.False(t, controller.canOriginalNPCMoveTo(npc, gate, npcMovementSchedule))
	assert.False(t, controller.canHorseMoveTo(npc, gate))
	controller.calculateNextNPCPosition(npc)
	assert.Equal(t, start, npc.Pos(), "the cached daytime route cannot step through the newly closed gate")
	assert.Empty(t, npc.MapUnitDetails().CurrentPath)
	assert.False(t, controller.createFreshPathToScheduledLocation(npc))
	assert.Empty(t, rng.values)

	controller.dateTime.Hour, controller.dateTime.Minute = 4, 59
	assert.False(t, controller.createFreshPathToScheduledLocation(npc))
	controller.dateTime.Advance(1)
	require.True(t, controller.createFreshPathToScheduledLocation(npc))
	assert.True(t, controller.canOriginalNPCMoveTo(npc, gate, npcMovementSchedule))
	blocker := map_units.NewNPCFriendly(npc.NPCReference, 1)
	blocker.SetPos(gate)
	controller.mapUnits = append(controller.mapUnits, blocker)
	assert.False(t, controller.createFreshPathToScheduledLocation(npc), "open gate still respects occupancy")
	assert.False(t, controller.canOriginalNPCMoveTo(npc, gate, npcMovementSchedule))
	controller.dateTime.Hour = 20
	assert.Equal(t, indexes.SpriteIndex(indexes.Portcullis), controller.npcTerrainAt(gate).Index)
	assert.Equal(t, gate, blocker.Pos(), "lowering does not displace an occupant")
	controller.mapUnits = controller.mapUnits[:1]
	layer.SetTileByLayer(map_state.MapOverrideLayer, &gate, indexes.BrickFloor)
	assert.True(t, controller.createFreshPathToScheduledLocation(npc), "stored overrides have the same priority for NPCs")
	assert.True(t, controller.canOriginalNPCMoveTo(npc, gate, npcMovementSchedule))
}
