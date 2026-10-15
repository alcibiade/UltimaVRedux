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

func setScheduleTravelGoal(npc *map_units.NPCFriendly, goal references.Position, floor references.FloorNumber) {
	npc.NPCReference.Schedule.X[1] = byte(goal.X)
	npc.NPCReference.Schedule.Y[1] = byte(goal.Y)
	npc.NPCReference.Schedule.Floor[1] = byte(floor)
}

func TestNPCReceivingFloorEntryThenWalk(t *testing.T) {
	for _, test := range []struct {
		name  string
		floor references.FloorNumber
		tile  indexes.SpriteIndex
	}{
		{"from above", 1, indexes.LadderUp},
		{"from below", -1, indexes.LadderDown},
		{"stairs from above", 1, indexes.StairsNorth},
		{"stairs from below", -1, indexes.StairsWest},
		{"east stairs from above", 1, indexes.StairsEast},
		{"south stairs from below", -1, indexes.StairsSouth},
	} {
		t.Run(test.name, func(t *testing.T) {
			start, entry, goal := references.Position{X: 3, Y: 3}, references.Position{X: 10, Y: 10}, references.Position{X: 13, Y: 10}
			rng := &scriptedOriginalScheduleRNG{t: t}
			controller, npc, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, start)
			npc.SetFloor(test.floor)
			setScheduleTravelGoal(npc, goal, 0)
			controller.dateTime.Hour = 1
			terrain.SetTileByLayer(map_state.MapLayer, &entry, test.tile)
			terrain.SetTileByLayer(map_state.MapLayer, &goal, indexes.LeftBed)
			// Presentation must not hide a valid connection.
			terrain.SetTileByLayer(map_state.MapUnitLayer, &entry, indexes.Fireplace)
			npc.MapUnitDetails().SetCurrentPath([]references.Position{start.Right(), goal})
			playerBefore := controller.mapState.PlayerLocation

			controller.calculateNextNPCPosition(npc)

			require.Equal(t, entry, npc.Pos())
			require.Equal(t, references.FloorNumber(0), npc.Floor())
			require.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
			require.Equal(t, []references.Position{entry.Right(), entry.Right().Right(), goal}, npc.MapUnitDetails().CurrentPath)
			for _, next := range []references.Position{entry.Right(), entry.Right().Right(), goal} {
				before := npc.Pos()
				controller.calculateNextNPCPosition(npc)
				require.Equal(t, next, npc.Pos())
				require.Equal(t, 1, controller.manhattanDistance(before, npc.Pos()))
				if next != goal {
					require.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
				}
			}
			assert.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
			assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			assert.Equal(t, playerBefore, controller.mapState.PlayerLocation)
			assert.Empty(t, rng.calls)
		})
	}
}

func TestNPCIncomingBlockedEntryDefersAndRetries(t *testing.T) {
	for _, obstruction := range []string{"NPC", "party", "destination NPC", "destination party", "route", "missing", "override", "wrong direction"} {
		t.Run(obstruction, func(t *testing.T) {
			entry, goal := references.Position{X: 10, Y: 10}, references.Position{X: 13, Y: 10}
			rng := &scriptedOriginalScheduleRNG{t: t}
			controller, npc, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, entry)
			npc.SetFloor(1) // The same hidden X/Y must not exempt a receiving occupant.
			setScheduleTravelGoal(npc, goal, 0)
			controller.dateTime.Hour = 1
			terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderUp)
			switch obstruction {
			case "NPC", "destination NPC":
				blocker := map_units.NewNPCFriendly(npc.NPCReference, 1)
				blocker.SetPos(entry)
				if obstruction == "destination NPC" {
					blocker.SetPos(goal)
				}
				controller.mapUnits = append(controller.mapUnits, blocker)
			case "party":
				controller.mapState.PlayerLocation.Position = entry
			case "destination party":
				controller.mapState.PlayerLocation.Position = goal
			case "route":
				for _, neighbor := range entry.Neighbors() {
					terrain.SetTileByLayer(map_state.MapLayer, &neighbor, indexes.StoneBrickWall)
				}
			case "missing":
				terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.BrickFloor)
			case "override":
				terrain.SetTileByLayer(map_state.MapOverrideLayer, &entry, indexes.Fireplace)
				terrain.SetTileByLayer(map_state.MapUnitLayer, &entry, indexes.LadderUp)
			case "wrong direction":
				terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderDown)
			}
			npc.MapUnitDetails().SetCurrentPath([]references.Position{goal})
			for range 3 {
				controller.calculateNextNPCPosition(npc)
				require.Equal(t, references.FloorNumber(1), npc.Floor())
				require.Equal(t, entry, npc.Pos())
				require.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
				require.Empty(t, npc.MapUnitDetails().CurrentPath)
			}
			controller.mapUnits = map_units.MapUnits{npc}
			controller.mapState.PlayerLocation.Position = references.Position{X: 30, Y: 30}
			terrain.UnSetTileByLayer(map_state.MapOverrideLayer, &entry)
			terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderUp)
			for _, neighbor := range entry.Neighbors() {
				terrain.SetTileByLayer(map_state.MapLayer, &neighbor, indexes.BrickFloor)
			}
			controller.calculateNextNPCPosition(npc)
			assert.Equal(t, references.FloorNumber(0), npc.Floor())
			assert.Equal(t, entry, npc.Pos())
			assert.NotEmpty(t, npc.MapUnitDetails().CurrentPath)
			assert.Empty(t, rng.calls)
		})
	}
}

func TestNPCFloorTravelSelectsReachableAlternative(t *testing.T) {
	for _, incoming := range []bool{true, false} {
		t.Run(map[bool]string{true: "incoming", false: "departure"}[incoming], func(t *testing.T) {
			start := references.Position{X: 10, Y: 10}
			near, far := references.Position{X: 12, Y: 10}, references.Position{X: 5, Y: 10}
			rng := &scriptedOriginalScheduleRNG{t: t}
			controller, npc, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, start)
			if incoming {
				npc.SetFloor(1)
				setScheduleTravelGoal(npc, start, 0)
			} else {
				setScheduleTravelGoal(npc, start, 1)
			}
			controller.dateTime.Hour = 1
			for _, entry := range []references.Position{near, far} {
				terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderUp)
			}
			for _, neighbor := range near.Neighbors() {
				terrain.SetTileByLayer(map_state.MapLayer, &neighbor, indexes.StoneBrickWall)
			}
			controller.calculateNextNPCPosition(npc)
			if incoming {
				require.Equal(t, far, npc.Pos())
			} else {
				require.Equal(t, start, npc.Pos())
				path := npc.MapUnitDetails().CurrentPath
				require.NotEmpty(t, path)
				require.Equal(t, far, path[len(path)-1])
			}
			for steps := 0; steps < 32 && npc.ArrivedScheduleSlot() != 1; steps++ {
				previous, floor := npc.Pos(), npc.Floor()
				controller.calculateNextNPCPosition(npc)
				if floor == npc.Floor() {
					require.Equal(t, 1, controller.manhattanDistance(previous, npc.Pos()))
				} else {
					require.Equal(t, far, previous, "departure must reach the usable ladder first")
				}
			}
			require.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
			assert.Equal(t, start, npc.Pos())
			assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			assert.Empty(t, rng.calls)
		})
	}
}

func TestNPCBothHiddenScheduleRelocation(t *testing.T) {
	for _, destinationFloor := range []references.FloorNumber{1, 2, -1} {
		start, goal := references.Position{X: 10, Y: 10}, references.Position{X: 20, Y: 20}
		rng := &scriptedOriginalScheduleRNG{t: t}
		controller, npc, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, start)
		npc.SetFloor(1)
		setScheduleTravelGoal(npc, goal, destinationFloor)
		npc.NPCReference.Schedule.Ai[1] = byte(references.Wander3)
		npc.MapUnitDetails().SetCurrentPath([]references.Position{start.Right()})
		controller.dateTime.Hour = 1
		// Observed-floor occupants/terrain are irrelevant to hidden relocation.
		controller.mapState.PlayerLocation.Position = goal
		terrain.SetTileByLayer(map_state.MapLayer, &goal, indexes.Fireplace)
		controller.AdvanceNextTurnCalcAndMoveNPCs()
		require.Equal(t, goal, npc.Pos())
		require.Equal(t, destinationFloor, npc.Floor())
		require.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
		require.Empty(t, npc.MapUnitDetails().CurrentPath)
		for range 3 {
			controller.AdvanceNextTurnCalcAndMoveNPCs()
			require.Equal(t, goal, npc.Pos(), "settled hidden wandering remains inactive")
		}
		assert.Empty(t, rng.calls)
		// ClearMapUnitTiles uses zero for an empty presentation layer, not NoSprites.
		assert.Equal(t, indexes.SpriteIndex(0), terrain.SpriteIndexByLayer(map_state.MapUnitLayer, goal))
		assert.False(t, controller.isPositionOccupiedOnFloor(goal, controller.mapState.PlayerLocation.Floor))
	}
}

func TestNPCIncomingScheduleChangeAndArrivedWander(t *testing.T) {
	start, entry, oldGoal := references.Position{X: 3, Y: 3}, references.Position{X: 10, Y: 10}, references.Position{X: 14, Y: 10}
	newGoal := entry.Up().Up()
	rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0, 8, 0}}
	controller, npc, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, start)
	npc.SetFloor(1)
	setScheduleTravelGoal(npc, oldGoal, 0)
	controller.dateTime.Hour = 1
	terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderUp)
	controller.calculateNextNPCPosition(npc)
	require.Equal(t, entry, npc.Pos())
	npc.NPCReference.Schedule.X[2], npc.NPCReference.Schedule.Y[2] = byte(newGoal.X), byte(newGoal.Y)
	npc.NPCReference.Schedule.Ai[2] = byte(references.Wander3)
	controller.dateTime.Hour = 2
	controller.calculateNextNPCPosition(npc)
	require.Equal(t, entry.Up(), npc.Pos(), "do not execute the obsolete incoming route")
	controller.calculateNextNPCPosition(npc)
	require.Equal(t, newGoal, npc.Pos())
	require.Equal(t, references.NPCScheduleSlot(2), npc.ArrivedScheduleSlot())
	controller.calculateNextNPCPosition(npc)
	require.Equal(t, newGoal.Right(), npc.Pos())
	controller.calculateNextNPCPosition(npc)
	assert.Equal(t, newGoal.Right().Right(), npc.Pos(), "arrived wandering must not path back to its anchor")
	assert.Equal(t, references.NPCScheduleSlot(2), npc.ArrivedScheduleSlot())
	assert.Empty(t, npc.MapUnitDetails().CurrentPath)
	assert.Empty(t, rng.values)
}

func TestNPCIncomingInterruptedByReturnToLastArrivedFloor(t *testing.T) {
	anchor, entry, goal := references.Position{X: 3, Y: 3}, references.Position{X: 10, Y: 10}, references.Position{X: 14, Y: 10}
	rng := &scriptedOriginalScheduleRNG{t: t}
	controller, npc, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, anchor)
	schedule := &npc.NPCReference.Schedule
	// A (slot 1) is selected at midnight and again at 02:00; B (slot 2)
	// interrupts it at 01:00. The fourth time marker returns to slot 1.
	schedule.Time = [4]byte{6, 0, 1, 2}
	setScheduleTravelGoal(npc, anchor, 1)
	schedule.X[2], schedule.Y[2] = byte(goal.X), byte(goal.Y)
	schedule.Floor[2] = 0
	npc.PlaceAtScheduleSlot(1)
	terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderUp)
	controller.dateTime.Hour = 1
	controller.calculateNextNPCPosition(npc)
	require.Equal(t, entry, npc.Pos())
	require.Equal(t, references.FloorNumber(0), npc.Floor())
	require.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
	controller.calculateNextNPCPosition(npc)
	interrupted := entry.Right()
	require.Equal(t, interrupted, npc.Pos())
	require.NotEmpty(t, npc.MapUnitDetails().CurrentPath)
	require.Equal(t, goal, npc.MapUnitDetails().CurrentPath[len(npc.MapUnitDetails().CurrentPath)-1])
	// Return to A before reaching B. An occupied departure ladder must keep
	// travel pending, not teleport or mistake the retained arrival for settling.
	controller.mapState.PlayerLocation.Position = entry
	controller.dateTime.Hour = 2
	require.Equal(t, references.NPCScheduleSlot(1), schedule.DesiredSlot(*controller.dateTime))
	for range 2 {
		controller.calculateNextNPCPosition(npc)
		require.Equal(t, interrupted, npc.Pos())
		require.Equal(t, references.FloorNumber(0), npc.Floor())
		require.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
		require.Empty(t, npc.MapUnitDetails().CurrentPath, "discard the obsolete route to B")
	}
	controller.mapState.PlayerLocation.Position = references.Position{X: 30, Y: 30}
	controller.calculateNextNPCPosition(npc)
	require.Equal(t, interrupted, npc.Pos(), "departure planning precedes movement")
	require.Equal(t, []references.Position{entry}, npc.MapUnitDetails().CurrentPath)
	controller.calculateNextNPCPosition(npc)
	require.Equal(t, entry, npc.Pos(), "walk back to the real departure connection")
	require.Equal(t, references.FloorNumber(0), npc.Floor())
	require.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
	controller.calculateNextNPCPosition(npc)
	require.Equal(t, anchor, npc.Pos())
	require.Equal(t, references.FloorNumber(1), npc.Floor())
	require.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
	require.Empty(t, npc.MapUnitDetails().CurrentPath)
	controller.calculateNextNPCPosition(npc)
	assert.Equal(t, anchor, npc.Pos())
	assert.Equal(t, references.FloorNumber(1), npc.Floor())
	assert.Empty(t, rng.calls)
}

func TestNPCIncomingLadderEdges(t *testing.T) {
	for _, entry := range []references.Position{{X: 0, Y: 0}, {X: 5, Y: 31}, {X: 31, Y: 5}, {X: 31, Y: 31}} {
		controller, npc, terrain := newOriginalScheduleTestController(t, &scriptedOriginalScheduleRNG{t: t}, references.Fixed,
			references.Position{X: 3, Y: 3})
		npc.SetFloor(1)
		setScheduleTravelGoal(npc, references.Position{X: 15, Y: 15}, 0)
		controller.dateTime.Hour = 1
		terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderUp)
		controller.calculateNextNPCPosition(npc)
		require.Equal(t, references.FloorNumber(0), npc.Floor())
		require.Equal(t, entry, npc.Pos())
		require.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
	}
}

func TestNPCIncomingWalkReplansAroundNewOccupant(t *testing.T) {
	entry, goal := references.Position{X: 10, Y: 10}, references.Position{X: 13, Y: 10}
	rng := &scriptedOriginalScheduleRNG{t: t}
	controller, npc, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, entry)
	npc.SetFloor(1)
	setScheduleTravelGoal(npc, goal, 0)
	controller.dateTime.Hour = 1
	terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderUp)
	controller.calculateNextNPCPosition(npc)
	require.Equal(t, entry, npc.Pos())
	require.NotEmpty(t, npc.MapUnitDetails().CurrentPath)
	require.Equal(t, entry.Right(), npc.MapUnitDetails().CurrentPath[0])
	blocker := map_units.NewNPCFriendly(npc.NPCReference, 1)
	blocker.SetPos(entry.Right())
	controller.mapUnits = append(controller.mapUnits, blocker)
	for steps := 0; steps < 16 && npc.Pos() != goal; steps++ {
		previous := npc.Pos()
		require.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
		controller.calculateNextNPCPosition(npc)
		require.Equal(t, 1, controller.manhattanDistance(previous, npc.Pos()))
		require.NotEqual(t, blocker.Pos(), npc.Pos())
	}
	require.Equal(t, goal, npc.Pos())
	assert.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
	assert.Empty(t, npc.MapUnitDetails().CurrentPath)
	assert.Empty(t, rng.calls)
}

func TestNPCIncomingUsesUnoccupiedPersistentAlternative(t *testing.T) {
	goal := references.Position{X: 15, Y: 15}
	near, far := goal.Left(), goal.Right().Right().Right()
	rng := &scriptedOriginalScheduleRNG{t: t}
	controller, npc, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, far)
	npc.SetFloor(1)
	setScheduleTravelGoal(npc, goal, 0)
	controller.dateTime.Hour = 1
	terrain.SetTileByLayer(map_state.MapLayer, &near, indexes.LadderUp)
	// A persistent override can supply a connection; presentation cannot hide it.
	terrain.SetTileByLayer(map_state.MapOverrideLayer, &far, indexes.LadderUp)
	terrain.SetTileByLayer(map_state.MapUnitLayer, &far, indexes.Fireplace)
	controller.mapState.PlayerLocation.Position = near
	// A hidden actor at these coordinates must not block the receiving floor.
	hidden := map_units.NewNPCFriendly(npc.NPCReference, 1)
	hidden.SetPos(far)
	hidden.SetFloor(2)
	controller.mapUnits = append(controller.mapUnits, hidden)
	controller.calculateNextNPCPosition(npc)
	assert.Equal(t, references.FloorNumber(0), npc.Floor())
	assert.Equal(t, far, npc.Pos())
	assert.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
	require.NotEmpty(t, npc.MapUnitDetails().CurrentPath)
	assert.Equal(t, goal, npc.MapUnitDetails().CurrentPath[len(npc.MapUnitDetails().CurrentPath)-1])
	assert.Empty(t, rng.calls)
}

func TestNPCIncomingActorsCannotShareEntry(t *testing.T) {
	entry, goal := references.Position{X: 10, Y: 10}, references.Position{X: 13, Y: 10}
	rng := &scriptedOriginalScheduleRNG{t: t}
	controller, first, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, entry)
	first.SetFloor(1)
	setScheduleTravelGoal(first, goal, 0)
	second := map_units.NewNPCFriendly(first.NPCReference, 1)
	second.PlaceAtScheduleSlot(0)
	second.SetFloor(1)
	setScheduleTravelGoal(second, goal.Down(), 0)
	controller.mapUnits = append(controller.mapUnits, second)
	controller.dateTime.Hour = 1
	terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderUp)
	controller.AdvanceNextTurnCalcAndMoveNPCs()
	require.Equal(t, references.FloorNumber(0), first.Floor())
	require.Equal(t, entry, first.Pos())
	require.Equal(t, references.FloorNumber(1), second.Floor())
	require.Empty(t, second.MapUnitDetails().CurrentPath)
	controller.AdvanceNextTurnCalcAndMoveNPCs()
	require.Equal(t, references.FloorNumber(0), second.Floor())
	require.Equal(t, entry, second.Pos())
	require.NotEqual(t, first.Pos(), second.Pos())
	assert.Equal(t, references.NPCScheduleSlot(0), second.ArrivedScheduleSlot())
	assert.Empty(t, rng.calls)
}

func TestNPCDepartureBlockedConnectionRetries(t *testing.T) {
	for _, test := range []struct {
		name  string
		floor references.FloorNumber
		tile  indexes.SpriteIndex
	}{
		{"up", 1, indexes.LadderUp},
		{"down", -1, indexes.LadderDown},
		{"stairs up", 1, indexes.StairsEast},
		{"stairs down", -1, indexes.StairsSouth},
	} {
		t.Run(test.name, func(t *testing.T) {
			start, ladder, goal := references.Position{X: 10, Y: 10}, references.Position{X: 12, Y: 10}, references.Position{X: 20, Y: 20}
			rng := &scriptedOriginalScheduleRNG{t: t}
			controller, npc, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, start)
			setScheduleTravelGoal(npc, goal, test.floor)
			controller.dateTime.Hour = 1
			// Neither missing nor occupied connections may retain an obsolete route.
			npc.MapUnitDetails().SetCurrentPath([]references.Position{start.Right()})
			controller.calculateNextNPCPosition(npc)
			require.Equal(t, start, npc.Pos())
			require.Empty(t, npc.MapUnitDetails().CurrentPath)
			terrain.SetTileByLayer(map_state.MapLayer, &ladder, test.tile)
			controller.mapState.PlayerLocation.Position = ladder
			controller.calculateNextNPCPosition(npc)
			require.Equal(t, start, npc.Pos())
			require.Empty(t, npc.MapUnitDetails().CurrentPath)
			controller.mapState.PlayerLocation.Position = references.Position{X: 30, Y: 30}
			controller.calculateNextNPCPosition(npc)
			require.Equal(t, start, npc.Pos(), "departure planning does not move this turn")
			for _, next := range []references.Position{start.Right(), ladder} {
				controller.calculateNextNPCPosition(npc)
				require.Equal(t, next, npc.Pos())
				require.Equal(t, references.FloorNumber(0), npc.Floor())
				require.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
			}
			controller.calculateNextNPCPosition(npc)
			assert.Equal(t, goal, npc.Pos())
			assert.Equal(t, test.floor, npc.Floor())
			assert.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
			assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			assert.Empty(t, rng.calls)
		})
	}
}

func TestNPCIncomingEntryAtAnchorRecordsPhysicalArrival(t *testing.T) {
	entry := references.Position{X: 10, Y: 10}
	controller, npc, terrain := newOriginalScheduleTestController(t, &scriptedOriginalScheduleRNG{t: t}, references.Fixed, entry)
	npc.SetFloor(1)
	setScheduleTravelGoal(npc, entry, 0)
	controller.dateTime.Hour = 1
	terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderUp)
	controller.calculateNextNPCPosition(npc)
	assert.Equal(t, entry, npc.Pos())
	assert.Equal(t, references.FloorNumber(0), npc.Floor())
	assert.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
	assert.Empty(t, npc.MapUnitDetails().CurrentPath)
}

func TestNPCIncomingWalkPermitsExactFireplaceAnchor(t *testing.T) {
	entry, goal := references.Position{X: 10, Y: 10}, references.Position{X: 13, Y: 10}
	rng := &scriptedOriginalScheduleRNG{t: t}
	controller, npc, terrain := newOriginalScheduleTestController(t, rng, references.Fixed, entry)
	npc.SetFloor(1)
	setScheduleTravelGoal(npc, goal, 0)
	controller.dateTime.Hour = 1
	terrain.SetTileByLayer(map_state.MapLayer, &entry, indexes.LadderUp)
	terrain.SetTileByLayer(map_state.MapLayer, &goal, indexes.Fireplace)
	blocked := entry.Right()
	terrain.SetTileByLayer(map_state.MapLayer, &blocked, indexes.Fireplace)
	controller.calculateNextNPCPosition(npc)
	require.Equal(t, entry, npc.Pos())
	require.NotEmpty(t, npc.MapUnitDetails().CurrentPath)
	for steps := 0; steps < 16 && npc.Pos() != goal; steps++ {
		previous := npc.Pos()
		require.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
		controller.calculateNextNPCPosition(npc)
		require.Equal(t, 1, controller.manhattanDistance(previous, npc.Pos()))
		require.NotEqual(t, blocked, npc.Pos(), "the destination exception cannot admit unrelated fireplaces")
	}
	require.Equal(t, goal, npc.Pos())
	assert.Equal(t, references.NPCScheduleSlot(1), npc.ArrivedScheduleSlot())
	assert.Empty(t, npc.MapUnitDetails().CurrentPath)
	assert.Empty(t, rng.calls)
}

func TestNPCHiddenScheduleRespectsOverridePrecedence(t *testing.T) {
	start, goal := references.Position{X: 10, Y: 10}, references.Position{X: 20, Y: 20}
	rng := &scriptedOriginalScheduleRNG{t: t}
	controller, npc, _ := newOriginalScheduleTestController(t, rng, references.Fixed, start)
	npc.SetFloor(1)
	setScheduleTravelGoal(npc, goal, 2)
	controller.dateTime.Hour = 1
	npc.MapUnitDetails().SetOverriddenAiType(references.Fixed)
	npc.MapUnitDetails().SetCurrentPath([]references.Position{start.Right()})
	controller.calculateNextNPCPosition(npc)
	assert.Equal(t, start, npc.Pos())
	assert.Equal(t, references.FloorNumber(1), npc.Floor())
	assert.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
	assert.Empty(t, npc.MapUnitDetails().CurrentPath)
	assert.Empty(t, rng.calls)
}
