package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestNPCScheduledTravelRecoversOpposingTraffic(t *testing.T) {
	rng := &scriptedOriginalScheduleRNG{t: t, values: []int{0, 8, 2, 0}}
	start, goal := references.Position{X: 8, Y: 10}, references.Position{X: 6, Y: 8}
	controller, northbound, layeredMap := newOriginalScheduleTestController(t, rng, references.Fixed, start)
	for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
			position := references.Position{X: x, Y: y}
			layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
		}
	}
	// Synthetic narrow aisle with a passing space, not copied original map data.
	for _, position := range []references.Position{
		{X: 8, Y: 11}, start, {X: 8, Y: 9}, {X: 8, Y: 8}, {X: 7, Y: 8}, {X: 7, Y: 9},
	} {
		layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.BrickFloor)
	}
	layeredMap.SetTileByLayer(map_state.MapLayer, &goal, indexes.LeftBed)

	southStart, southGoal := references.Position{X: 8, Y: 9}, references.Position{X: 8, Y: 11}
	southRef := northbound.NPCReference
	southRef.Schedule = references.CreateNPCScheduledFixedOneLocation(southStart, 0)
	southbound := map_units.NewNPCFriendly(southRef, 1)
	southbound.PlaceAtScheduleSlot(0)
	controller.mapUnits = append(controller.mapUnits, southbound)
	for i, npc := range []*map_units.NPCFriendly{northbound, southbound} {
		destination := []references.Position{goal, southGoal}[i]
		npc.NPCReference.Schedule.X[1], npc.NPCReference.Schedule.Y[1] = byte(destination.X), byte(destination.Y)
	}
	controller.dateTime.Hour = 1
	for _, npc := range []*map_units.NPCFriendly{northbound, southbound} {
		require.NotEqual(t, npc.ArrivedScheduleSlot(), npc.NPCReference.Schedule.DesiredSlot(*controller.dateTime))
		require.False(t, controller.createFreshPathToScheduledLocation(npc), "each traveller initially blocks the other's route")
	}

	for turn, positions := range [][2]references.Position{
		{start, {X: 7, Y: 9}},
		{{X: 8, Y: 9}, {X: 7, Y: 9}},
		{{X: 8, Y: 8}, {X: 8, Y: 9}},
		{{X: 7, Y: 8}, {X: 8, Y: 10}},
		{goal, southGoal},
	} {
		controller.AdvanceNextTurnCalcAndMoveNPCs()
		require.Equal(t, positions[0], northbound.Pos(), "northbound turn %d", turn)
		require.Equal(t, positions[1], southbound.Pos(), "southbound turn %d", turn)
		require.NotEqual(t, northbound.Pos(), southbound.Pos())
	}
	assert.Equal(t, references.NPCScheduleSlot(1), northbound.ArrivedScheduleSlot())
	assert.Equal(t, references.NPCScheduleSlot(1), southbound.ArrivedScheduleSlot())
	controller.AdvanceNextTurnCalcAndMoveNPCs()
	assert.Equal(t, goal, northbound.Pos(), "arrived fixed guards must stay settled")
	assert.Equal(t, southGoal, southbound.Pos())
	assert.Equal(t, []originalScheduleRNGCall{{0, 255}, {0, 255}, {0, 64}, {0, 255}}, rng.calls)
	assert.Empty(t, rng.values)
}

func TestNPCScheduledTravelRecoveryRespectsMovementRules(t *testing.T) {
	for _, tc := range []struct {
		name     string
		tile     indexes.SpriteIndex
		gate     int
		occupied bool
		avatar   bool
		override bool
		cached   bool
		wantMove bool
	}{
		{name: "distant blockage", tile: indexes.BrickFloor, gate: 8, wantMove: true},
		{name: "failed gate", tile: indexes.BrickFloor},
		{name: "scheduled chair permission", tile: indexes.ChairBackForward, gate: 8, wantMove: true},
		{name: "left bed is not destination", tile: indexes.LeftBed, gate: 8},
		{name: "right bed is not destination", tile: indexes.RightBed, gate: 8},
		{name: "fireplace", tile: indexes.Fireplace, gate: 8},
		{name: "wall", tile: indexes.StoneBrickWall, gate: 8},
		{name: "occupied passing space", tile: indexes.BrickFloor, gate: 8, occupied: true},
		{name: "Avatar in passing space", tile: indexes.BrickFloor, gate: 8, avatar: true},
		{name: "blocked terrain override", tile: indexes.BrickFloor, gate: 8, override: true},
		{name: "blocked cached step", tile: indexes.BrickFloor, gate: 8, cached: true, wantMove: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := []int{tc.gate}
			calls := []originalScheduleRNGCall{{0, 255}}
			if tc.gate != 0 {
				values = append(values, 2) // West, regardless of other legal directions.
				calls = append(calls, originalScheduleRNGCall{0, 64})
			}
			rng := &scriptedOriginalScheduleRNG{t: t, values: values}
			start, goal := references.Position{X: 9, Y: 15}, references.Position{X: 9, Y: 10}
			controller, npc, layeredMap := newOriginalScheduleTestController(t, rng, references.Fixed, start)
			// A wall across the whole map blocks the route several tiles away.
			for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
				position := references.Position{X: x, Y: 12}
				layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
			}
			npc.NPCReference.Schedule.X[1], npc.NPCReference.Schedule.Y[1] = byte(goal.X), byte(goal.Y)
			controller.dateTime.Hour = 1
			candidate := start.Left()
			layeredMap.SetTileByLayer(map_state.MapLayer, &candidate, tc.tile)
			if tc.occupied {
				blocker := map_units.NewNPCFriendly(npc.NPCReference, 1)
				blocker.SetPos(candidate)
				controller.mapUnits = append(controller.mapUnits, blocker)
			}
			if tc.avatar {
				controller.mapState.PlayerLocation.Position = candidate
			}
			if tc.override {
				layeredMap.SetTileByLayer(map_state.MapOverrideLayer, &candidate, indexes.Fireplace)
			}
			require.False(t, controller.createFreshPathToScheduledLocation(npc))
			if tc.cached {
				blocked := start.Up()
				layeredMap.SetTileByLayer(map_state.MapOverrideLayer, &blocked, indexes.Fireplace)
				npc.MapUnitDetails().SetCurrentPath([]references.Position{blocked, goal})
			}

			controller.calculateNextNPCPosition(npc)

			want := start
			if tc.wantMove {
				want = candidate
			}
			assert.Equal(t, want, npc.Pos())
			assert.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
			assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			assert.Equal(t, calls, rng.calls, "recovery attempts only one chosen direction")
			assert.Empty(t, rng.values)
		})
	}
}

func TestNPCScheduledTravelRecoveryDoesNotRunOffFloor(t *testing.T) {
	rng := &scriptedOriginalScheduleRNG{t: t}
	start, goal := references.Position{X: 9, Y: 15}, references.Position{X: 9, Y: 10}
	controller, npc, layeredMap := newOriginalScheduleTestController(t, rng, references.Fixed, start)
	for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
		position := references.Position{X: x, Y: 12}
		layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
	}
	npc.SetFloor(1)
	npc.NPCReference.Schedule.X[1], npc.NPCReference.Schedule.Y[1] = byte(goal.X), byte(goal.Y)
	npc.NPCReference.Schedule.Floor[1] = 1
	controller.dateTime.Hour = 1
	require.False(t, controller.createFreshPathToScheduledLocation(npc))

	controller.AdvanceNextTurnCalcAndMoveNPCs()

	assert.Equal(t, goal, npc.Pos())
	assert.Equal(t, references.FloorNumber(1), npc.Floor())
	assert.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
	assert.Empty(t, npc.MapUnitDetails().CurrentPath)
	assert.Empty(t, rng.calls)
}
