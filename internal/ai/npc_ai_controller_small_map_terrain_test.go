package ai

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/test/helpers"
)

func TestOriginalNPCWanderAndApproachAllTerrain(t *testing.T) {
	// Independent collide eligibility, including beaches and excluding chairs.
	// Do not derive expectations from the production bitmap or player passability.
	allowedIDs := []int{
		0, 4, 5, 6, 7, 8, 9, 10, 11, 14, 15,
		29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 44, 45, 48, 49, 50, 51,
		64, 68, 69, 71, 72, 73, 106, 107, 135,
		170, 184, 186, 196, 197, 198, 199, 200, 201,
	}
	for id := range 256 {
		t.Run(fmt.Sprintf("tile%d", id), func(t *testing.T) {
			start := references.Position{X: 10, Y: 10}
			candidate := start.Right()
			rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0}}
			controller, npc, layeredMap := newOriginalScheduleTestController(t, rng, references.Wander3, start)
			controller.mapState.PlayerLocation.Position = candidate.Right()
			layeredMap.SetTileByLayer(map_state.MapLayer, &candidate, indexes.SpriteIndex(id))
			for slot := range npc.NPCReference.Schedule.X {
				require.NotEqual(t, candidate, npc.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(references.NPCScheduleSlot(slot)).Position)
			}
			allowed := slices.Contains(allowedIDs, id)
			want := start
			if allowed {
				want = candidate
			}

			assert.Equal(t, allowed, controller.canOriginalNPCMoveTo(npc, candidate, npcMovementApproach))
			controller.calculateNextNPCPosition(npc)
			assert.Equal(t, want, npc.Pos(), "scheduled original wandering must obey terrain")
			assert.Equal(t, []originalScheduleRNGCall{{0, 255}, {0, 64}}, rng.calls)
			assert.Empty(t, rng.values)

			npc.SetPos(start)
			controller.attractOriginal(npc, references.Talk3)
			assert.Equal(t, want, npc.Pos(), "only east reduces distance to the Avatar")
			assert.Equal(t, []originalScheduleRNGCall{{0, 255}, {0, 64}}, rng.calls, "approach consumes no RNG")
		})
	}
}

// Reuse the strict schedule mock, allowing only a deterministic identity shuffle.
type terrainShuffleRNG struct {
	*scriptedOriginalScheduleRNG
	shuffles int
}

func (r *terrainShuffleRNG) ShuffleDirections(directions []references.Position) {
	r.t.Helper()
	require.Equal(r.t, []references.Position{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}}, directions)
	r.shuffles++
}

func TestAlternateNPCWanderChecksTerrain(t *testing.T) {
	for _, tc := range []struct {
		name string
		tile indexes.SpriteIndex
		want bool
	}{
		{"fireplace", indexes.Fireplace, false},
		{"floor", indexes.BrickFloor, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := references.Position{X: 10, Y: 10}
			candidate := start.Right()
			rng := &terrainShuffleRNG{scriptedOriginalScheduleRNG: &scriptedOriginalScheduleRNG{t: t}}
			controller, npc, layeredMap := newOriginalScheduleTestController(t, rng, references.Wander3, start)
			for _, position := range []references.Position{start.Up(), start.Down(), start.Left()} {
				layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
			}
			layeredMap.SetTileByLayer(map_state.MapLayer, &candidate, tc.tile)

			assert.Equal(t, tc.want, controller.wanderOneTileWithinN(npc, start, 3))
			want := start
			if tc.want {
				want = candidate
			}
			assert.Equal(t, want, npc.Pos())
			assert.Equal(t, 1, rng.shuffles)
			assert.Empty(t, rng.calls)
		})
	}
}

func TestNPCScheduleGoalTerrainExceptionIsExact(t *testing.T) {
	start := references.Position{X: 10, Y: 10}
	candidate := start.Right()
	for _, tile := range []indexes.SpriteIndex{
		indexes.LeftBed, indexes.RightBed, indexes.Fireplace,
		indexes.LockedDoor, indexes.LockedDoorView, indexes.MagicLockDoor, indexes.MagicLockDoorWithView,
	} {
		for _, tc := range []struct {
			name  string
			goal  references.Position
			floor references.FloorNumber
			want  bool
		}{
			{"exact goal", candidate, 0, true},
			{"different x", candidate.Right(), 0, false},
			{"different y", candidate.Down(), 0, false},
			{"different floor", candidate, 1, false},
		} {
			t.Run(fmt.Sprintf("tile%d/%s", tile, tc.name), func(t *testing.T) {
				rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0}}
				controller, npc, layeredMap := newOriginalScheduleTestController(t, rng, references.Wander3, start)
				npc.NPCReference.Schedule = references.CreateNPCScheduledFixedOneLocation(tc.goal, tc.floor)
				layeredMap.SetTileByLayer(map_state.MapLayer, &candidate, tile)
				controller.mapState.PlayerLocation.Position = candidate.Right()

				assert.Equal(t, tc.want, controller.canOriginalNPCMoveTo(npc, candidate, npcMovementWander))
				assert.Equal(t, tc.want, controller.canOriginalNPCMoveTo(npc, candidate, npcMovementSchedule))
				assert.False(t, controller.canOriginalNPCMoveTo(npc, candidate, npcMovementApproach))
				controller.attractOriginal(npc, references.Talk3)
				assert.Equal(t, start, npc.Pos(), "attraction has no schedule-goal exception")
				controller.randomStepOriginal(npc, start, 3)
				want := start
				if tc.want {
					want = candidate
				}
				assert.Equal(t, want, npc.Pos())
				npc.SetPos(start)
				npc.MapUnitDetails().SetCurrentPath([]references.Position{candidate})
				assert.Equal(t, tc.want, controller.moveNPCOnCalculatedPath(npc))
				assert.Equal(t, want, npc.Pos())
			})
		}
	}
}

func TestNPCChairsRequireSameFloorScheduleTravel(t *testing.T) {
	for _, tile := range []indexes.SpriteIndex{144, 145, 146, 147} {
		for _, goalAtChair := range []bool{false, true} {
			for _, floor := range []references.FloorNumber{0, 1} {
				t.Run(fmt.Sprintf("tile%d/goal%t/floor%d", tile, goalAtChair, floor), func(t *testing.T) {
					start := references.Position{X: 10, Y: 10}
					chair, ladder := start.Right(), start.Right().Right()
					rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0}}
					controller, npc, layeredMap := newOriginalScheduleTestController(t, rng, references.Wander3, start)
					goal := ladder
					if goalAtChair {
						goal = chair
					}
					npc.NPCReference.Schedule = references.CreateNPCScheduledFixedOneLocation(goal, floor)
					layeredMap.SetTileByLayer(map_state.MapLayer, &chair, tile)
					layeredMap.SetTileByLayer(map_state.MapLayer, &ladder, indexes.LadderUp)

					assert.False(t, controller.canOriginalNPCMoveTo(npc, chair, npcMovementWander))
					assert.False(t, controller.canOriginalNPCMoveTo(npc, chair, npcMovementApproach))
					controller.randomStepOriginal(npc, start, 3)
					assert.Equal(t, start, npc.Pos(), "even the exact chair goal rejects wandering")
					// A different-floor goal makes this cached route TOLAD rather than MOVE.
					npc.MapUnitDetails().SetCurrentPath([]references.Position{chair, ladder})
					assert.Equal(t, floor == 0, controller.moveNPCOnCalculatedPath(npc))
					if floor == 0 {
						assert.Equal(t, chair, npc.Pos())
					} else {
						assert.Equal(t, start, npc.Pos())
						assert.Empty(t, npc.MapUnitDetails().CurrentPath)
						assert.False(t, controller.moveNPCOnCalculatedPath(npc), "must not leap to the ladder")
					}
				})
			}
		}
	}
}

func TestNPCScheduleGoalExceptionLosesToOccupancy(t *testing.T) {
	for _, tile := range []indexes.SpriteIndex{indexes.LeftBed, indexes.Fireplace, indexes.LockedDoor, indexes.OutsidePath1} {
		for _, occupant := range []string{"none", "same floor NPC", "other floor NPC", "Avatar"} {
			t.Run(fmt.Sprintf("tile%d/%s", tile, occupant), func(t *testing.T) {
				start := references.Position{X: 10, Y: 10}
				goal := start.Right()
				rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0}}
				controller, npc, layeredMap := newOriginalScheduleTestController(t, rng, references.Wander3, start)
				npc.NPCReference.Schedule = references.CreateNPCScheduledFixedOneLocation(goal, 0)
				layeredMap.SetTileByLayer(map_state.MapLayer, &goal, tile)
				switch occupant {
				case "same floor NPC", "other floor NPC":
					blocker := map_units.NewNPCFriendly(npc.NPCReference, 1)
					blocker.PlaceAtScheduleSlot(0)
					if occupant == "other floor NPC" {
						blocker.SetFloor(1)
					}
					controller.mapUnits = append(controller.mapUnits, blocker)
				case "Avatar":
					controller.mapState.PlayerLocation.Position = goal
				}
				want := occupant == "none" || occupant == "other floor NPC"
				assert.Equal(t, want, controller.canOriginalNPCMoveTo(npc, goal, npcMovementWander))
				assert.Equal(t, want, controller.canOriginalNPCMoveTo(npc, goal, npcMovementSchedule))
				assert.Equal(t, want, controller.createFreshPathToScheduledLocation(npc), "planning must not exempt occupied goals")
				assert.Equal(t, want, controller.moveNPCOnCalculatedPath(npc))
				expected := start
				if want {
					expected = goal
				}
				assert.Equal(t, expected, npc.Pos())
				npc.SetPos(start)
				controller.randomStepOriginal(npc, start, 3)
				assert.Equal(t, expected, npc.Pos())
			})
		}
	}
}

func TestNPCScheduledPathsUseTerrainAndExactGoal(t *testing.T) {
	for _, tc := range []struct {
		name      string
		tiles     []indexes.SpriteIndex
		atGoal    bool
		avoid     bool
		stepLegal bool
	}{
		{"intermediate hazards and beds", []indexes.SpriteIndex{134, 140, 143, 188, 171, 172}, false, true, false},
		{"blocked exact goal", []indexes.SpriteIndex{171, 172, 188, 151, 152, 185, 187}, true, false, true},
		{"beach planning differs from stepping", []indexes.SpriteIndex{48, 49, 50, 51}, false, true, true},
		{"intermediate chairs", []indexes.SpriteIndex{144, 145, 146, 147}, false, false, true},
		{"chair goal", []indexes.SpriteIndex{144, 145, 146, 147}, true, false, true},
	} {
		for _, tile := range tc.tiles {
			t.Run(fmt.Sprintf("%s/tile%d", tc.name, tile), func(t *testing.T) {
				start, goal := references.Position{X: 10, Y: 10}, references.Position{X: 14, Y: 10}
				controller, npc, layeredMap := newOriginalScheduleTestController(t, &scriptedOriginalScheduleRNG{t: t}, references.Fixed, start)
				npc.NPCReference.Schedule = references.CreateNPCScheduledFixedOneLocation(goal, 0)
				// A one-tile corridor forces use of legal intermediate tiles; only
				// avoidance cases get a detour, so an empty route cannot pass the test.
				for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
					for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
						position := references.Position{X: x, Y: y}
						layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
					}
				}
				for x := start.X; x <= goal.X; x++ {
					position := references.Position{X: x, Y: start.Y}
					layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.BrickFloor)
					if tc.avoid && x > start.X && x < goal.X {
						position.Y--
						layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.BrickFloor)
					}
				}
				candidate := references.Position{X: 12, Y: 10}
				if tc.atGoal {
					candidate = goal
				}
				layeredMap.SetTileByLayer(map_state.MapLayer, &candidate, tile)
				assert.Equal(t, tc.stepLegal, controller.canOriginalNPCMoveTo(npc, candidate, npcMovementSchedule))
				require.True(t, controller.createFreshPathToScheduledLocation(npc))
				path := slices.Clone(npc.MapUnitDetails().CurrentPath)
				require.NotEmpty(t, path)
				assert.Equal(t, goal, path[len(path)-1])
				assert.Equal(t, !tc.avoid, slices.Contains(path, candidate))
				for _, next := range path {
					require.True(t, controller.moveNPCOnCalculatedPath(npc))
					assert.Equal(t, next, npc.Pos())
				}
				assert.Equal(t, goal, npc.Pos())
				assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			})
		}
	}
}

func TestNPCCachedPathRevalidatesAndClearsRemainder(t *testing.T) {
	for _, obstruction := range []string{"terrain", "override", "NPC", "Avatar", "nonadjacent step"} {
		t.Run(obstruction, func(t *testing.T) {
			start, goal := references.Position{X: 10, Y: 10}, references.Position{X: 13, Y: 10}
			controller, npc, layeredMap := newOriginalScheduleTestController(t, &scriptedOriginalScheduleRNG{t: t}, references.Fixed, start)
			npc.NPCReference.Schedule = references.CreateNPCScheduledFixedOneLocation(goal, 0)
			require.True(t, controller.createFreshPathToScheduledLocation(npc))
			require.Equal(t, []references.Position{start.Right(), start.Right().Right(), goal}, npc.MapUnitDetails().CurrentPath)
			blocked := start.Right()
			switch obstruction {
			case "terrain":
				layeredMap.SetTileByLayer(map_state.MapLayer, &blocked, indexes.Fireplace)
			case "override":
				layeredMap.SetTileByLayer(map_state.MapOverrideLayer, &blocked, indexes.LockedDoor)
			case "NPC":
				blocker := map_units.NewNPCFriendly(npc.NPCReference, 1)
				blocker.SetPos(blocked)
				blocker.SetFloor(0)
				controller.mapUnits = append(controller.mapUnits, blocker)
			case "Avatar":
				controller.mapState.PlayerLocation.Position = blocked
			case "nonadjacent step":
				npc.MapUnitDetails().DequeueNextPosition()
			}

			assert.False(t, controller.moveNPCOnCalculatedPath(npc))
			assert.Equal(t, start, npc.Pos())
			assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			assert.False(t, controller.moveNPCOnCalculatedPath(npc), "next turn must not leap past the blocked step")
			assert.Equal(t, start, npc.Pos())

			layeredMap.SetTileByLayer(map_state.MapLayer, &blocked, indexes.BrickFloor)
			layeredMap.UnSetTileByLayer(map_state.MapOverrideLayer, &blocked)
			controller.mapUnits = map_units.MapUnits{npc}
			controller.mapState.PlayerLocation.Position = references.Position{X: 30, Y: 30}
			assert.False(t, controller.moveNPCOnCalculatedPath(npc), "removing the obstruction must not revive the discarded path")
			require.True(t, controller.createFreshPathToScheduledLocation(npc))
			require.True(t, controller.moveNPCOnCalculatedPath(npc))
			assert.Equal(t, blocked, npc.Pos(), "a fresh route restarts at the adjacent step")
		})
	}
}

func TestNPCScheduleRouteReclassificationFullTurns(t *testing.T) {
	gameRefs := helpers.LoadTestGameReferences(t, helpers.LoadTestConfiguration(t))
	slr := gameRefs.LocationReferences.GetLocationReference(references.LordBritishsCastle)
	ladder := slr.GetClosestLadder(references.Position{X: 1, Y: 1}, 0, 1)
	start, chair := ladder.Left().Left(), ladder.Left()
	detour := []references.Position{start.Up(), chair.Up(), ladder.Up(), ladder}
	for _, position := range append(slices.Clone(detour), start, chair) {
		require.True(t, isSmallMapPosition(position))
		require.Equal(t, ladder, slr.GetClosestLadder(position, 0, 1), "fixture must retain one ladder goal throughout travel")
	}

	for _, tc := range []struct {
		name     string
		oldFloor byte
		newFloor byte
	}{
		{"fresh ladder route", 1, 1},
		{"MOVE to TOLAD same endpoint", 0, 1},
		{"MOVE to TOLAD obsolete endpoint", 0, 1},
		{"TOLAD to MOVE new chair goal", 1, 0},
		{"TOLAD to MOVE same endpoint", 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rng := &scriptedOriginalScheduleRNG{t: t}
			controller, npc, layeredMap := newOriginalScheduleTestController(t, rng, references.Fixed, start)
			controller.slr = slr
			for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
				for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
					position := references.Position{X: x, Y: y}
					layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
				}
			}
			for _, position := range append(slices.Clone(detour), start, chair) {
				layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.Desert1)
			}
			layeredMap.SetTileByLayer(map_state.MapLayer, &chair, indexes.ChairBackForward)
			layeredMap.SetTileByLayer(map_state.MapLayer, &ladder, slr.GetTileNumber(ladder, 0))

			// Slot zero remains the arrived slot; advancing the clock activates travel.
			schedule := &npc.NPCReference.Schedule
			for _, slot := range []int{1, 2} {
				schedule.X[slot], schedule.Y[slot] = byte(ladder.X), byte(ladder.Y)
			}
			schedule.Floor[1], schedule.Floor[2] = tc.oldFloor, tc.newFloor
			controller.dateTime.Hour = 1
			if tc.name != "fresh ladder route" {
				require.True(t, controller.createFreshPathToScheduledLocation(npc))
				wantOldPath := detour
				if tc.oldFloor == 0 {
					wantOldPath = []references.Position{chair, ladder}
				}
				require.Equal(t, wantOldPath, npc.MapUnitDetails().CurrentPath,
					"chair shortcut is preferred for MOVE but forbidden for TOLAD")
			}

			wantFirst, wantRemaining := start, detour
			switch tc.name {
			case "MOVE to TOLAD obsolete endpoint":
				oldGoal := chair.Up()
				schedule.X[1], schedule.Y[1] = byte(oldGoal.X), byte(oldGoal.Y)
				// Its first step is legal even under TOLAD: endpoint validation,
				// not a blocked chair, must discard this obsolete route.
				npc.MapUnitDetails().SetCurrentPath([]references.Position{start.Up(), oldGoal})
			case "TOLAD to MOVE new chair goal":
				schedule.X[2], schedule.Y[2] = byte(chair.X), byte(chair.Y)
				wantFirst, wantRemaining = chair, nil
			case "TOLAD to MOVE same endpoint":
				// A chair appears on the cached detour before the schedule changes.
				// MOVE must allow this remaining step without retaining TOLAD rules.
				position := detour[0]
				layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.ChairBackLeft)
				wantFirst, wantRemaining = position, detour[1:]
			}
			controller.dateTime.Hour = 2
			require.NotEqual(t, schedule.DesiredSlot(*controller.dateTime), npc.ArrivedScheduleSlot())

			controller.calculateNextNPCPosition(npc)

			assert.Equal(t, wantFirst, npc.Pos())
			if len(wantRemaining) == 0 {
				assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			} else {
				assert.Equal(t, wantRemaining, npc.MapUnitDetails().CurrentPath)
			}
			for _, next := range wantRemaining {
				controller.calculateNextNPCPosition(npc)
				require.Equal(t, next, npc.Pos(), "each subsequent turn must progress, not retry the chair shortcut")
				assert.Equal(t, references.FloorNumber(0), npc.Floor())
			}
			assert.Empty(t, npc.MapUnitDetails().CurrentPath)
			assert.Empty(t, rng.calls)
			if tc.newFloor == 1 {
				assert.Equal(t, ladder, npc.Pos())
				assert.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
				// Stop on the ladder; the next turn would teleport to the new floor.
			} else {
				assert.Equal(t, references.NPCScheduleSlot(2), npc.ArrivedScheduleSlot())
			}
		})
	}
}

func TestNPCCachedLadderRouteSurvivesCloserUnreachableLadder(t *testing.T) {
	gameRefs := helpers.LoadTestGameReferences(t, helpers.LoadTestConfiguration(t))
	slr := gameRefs.LocationReferences.GetLocationReference(references.LordBritishsCastle)
	goal := slr.GetClosestLadder(references.Position{X: 16, Y: 16}, 0, 1)
	start := goal.Left()
	path := []references.Position{start.Up(), goal.Up(), goal}
	otherLadder := slr.GetClosestLadder(path[0], 0, 1)
	require.NotEqual(t, goal, otherLadder)
	require.Equal(t, goal, slr.GetClosestLadder(start, 0, 1))

	rng := &scriptedOriginalScheduleRNG{t: t}
	controller, npc, layeredMap := newOriginalScheduleTestController(t, rng, references.Fixed, start)
	controller.slr = slr
	for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
			position := references.Position{X: x, Y: y}
			layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
		}
	}
	for _, position := range append(slices.Clone(path), start) {
		layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.BrickFloor)
	}
	for _, position := range []references.Position{goal, otherLadder} {
		layeredMap.SetTileByLayer(map_state.MapLayer, &position, slr.GetTileNumber(position, 0))
	}
	for _, position := range []references.Position{otherLadder.Up(), otherLadder.Down(), otherLadder.Left(), otherLadder.Right()} {
		require.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), layeredMap.GetTileByLayer(map_state.MapLayer, &position).Index,
			"the competing ladder is unreachable, not merely a less convenient destination")
	}
	npc.NPCReference.Schedule.X[1], npc.NPCReference.Schedule.Y[1] = byte(goal.X), byte(goal.Y)
	npc.NPCReference.Schedule.Floor[1] = 1
	controller.dateTime.Hour = 1
	npc.MapUnitDetails().SetCurrentPath(slices.Clone(path))

	for step, next := range path {
		controller.calculateNextNPCPosition(npc)

		require.Equal(t, next, npc.Pos(), "turn %d must retain the selected ladder through the detour", step)
		wantClosest := goal
		if step == 0 {
			wantClosest = otherLadder
		}
		require.Equal(t, wantClosest, slr.GetClosestLadder(npc.Pos(), 0, 1))
		assert.Equal(t, path[step+1:], npc.MapUnitDetails().CurrentPath, "the cached endpoint must remain the original ladder")
		assert.Equal(t, references.FloorNumber(0), npc.Floor())
		assert.Equal(t, references.NPCScheduleSlot(0), npc.ArrivedScheduleSlot())
	}
	assert.Empty(t, rng.calls)
	// Stop at the original ladder, before the next turn teleports to floor one.
}

func TestNPCFullTurnDiscardsObsoleteOrInactivePaths(t *testing.T) {
	for _, state := range []string{"changed same-floor goal", "arrived", "fixed override"} {
		t.Run(state, func(t *testing.T) {
			start := references.Position{X: 10, Y: 10}
			oldGoal, newGoal := start.Right().Right(), start.Up().Up()
			controller, npc, _ := newOriginalScheduleTestController(t, &scriptedOriginalScheduleRNG{t: t}, references.Fixed, start)
			schedule := &npc.NPCReference.Schedule
			schedule.X[1], schedule.Y[1] = byte(oldGoal.X), byte(oldGoal.Y)
			schedule.X[2], schedule.Y[2] = byte(newGoal.X), byte(newGoal.Y)
			controller.dateTime.Hour = 1
			require.True(t, controller.createFreshPathToScheduledLocation(npc))
			require.Equal(t, []references.Position{start.Right(), oldGoal}, npc.MapUnitDetails().CurrentPath)
			want := start
			switch state {
			case "changed same-floor goal":
				controller.dateTime.Hour = 2
				want = start.Up()
			case "arrived":
				npc.SetPos(oldGoal)
				want = oldGoal
			case "fixed override":
				npc.MapUnitDetails().SetOverriddenAiType(references.Fixed)
			}

			controller.calculateNextNPCPosition(npc)

			assert.Equal(t, want, npc.Pos(), "must not execute the old path's adjacent first step")
			if state == "changed same-floor goal" {
				assert.Equal(t, []references.Position{newGoal}, npc.MapUnitDetails().CurrentPath)
				controller.calculateNextNPCPosition(npc)
				assert.Equal(t, newGoal, npc.Pos())
			} else {
				assert.Empty(t, npc.MapUnitDetails().CurrentPath)
				controller.calculateNextNPCPosition(npc)
				assert.Equal(t, want, npc.Pos(), "inactive routes must stay discarded on later turns")
			}
		})
	}
}

func TestNPCTerrainUsesOverridesNotMapUnitSprites(t *testing.T) {
	for _, tc := range []struct {
		name     string
		base     indexes.SpriteIndex
		override indexes.SpriteIndex
		sprite   indexes.SpriteIndex
		occupied bool
		want     bool
	}{
		{"brick override opens fireplace", indexes.Fireplace, indexes.BrickFloor, indexes.NoSprites, false, true},
		{"fireplace override blocks brick", indexes.BrickFloor, indexes.Fireplace, indexes.NoSprites, false, false},
		{"NPC sprite cannot block brick terrain", indexes.BrickFloor, indexes.NoSprites, indexes.TownsPerson_KeyIndex, false, true},
		{"floor sprite cannot open fireplace", indexes.Fireplace, indexes.NoSprites, indexes.BrickFloor, false, false},
		{"sprite cannot hide blocked override", indexes.BrickFloor, indexes.Fireplace, indexes.BrickFloor, false, false},
		{"sprite cannot hide open override", indexes.Fireplace, indexes.BrickFloor, indexes.Fireplace, false, true},
		{"authoritative unit blocks without sprite", indexes.BrickFloor, indexes.NoSprites, indexes.NoSprites, true, false},
		{"authoritative unit wins over open override", indexes.Fireplace, indexes.BrickFloor, indexes.BrickFloor, true, false},
		{"zero base with absent overlay", 0, indexes.NoSprites, indexes.NoSprites, false, true},
		{"zero base with zero overlay", 0, 0, indexes.NoSprites, false, true},
		{"zero overlay cannot open fireplace", indexes.Fireplace, 0, indexes.NoSprites, false, false},
		{"zero overlay preserves brick", indexes.BrickFloor, 0, indexes.NoSprites, false, true},
		{"nonzero overlay blocks zero base", 0, indexes.Fireplace, indexes.NoSprites, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := references.Position{X: 10, Y: 10}
			candidate, goal := start.Right(), start.Right().Right()
			rng := &scriptedOriginalScheduleRNG{t: t, values: []int{8, 0}}
			controller, npc, layeredMap := newOriginalScheduleTestController(t, rng, references.Wander3, start)
			npc.NPCReference.Schedule = references.CreateNPCScheduledFixedOneLocation(goal, 0)
			layeredMap.SetTileByLayer(map_state.MapLayer, &candidate, tc.base)
			layeredMap.SetTileByLayer(map_state.MapOverrideLayer, &candidate, tc.override)
			layeredMap.SetTileByLayer(map_state.MapUnitLayer, &candidate, tc.sprite)
			if tc.occupied {
				blocker := map_units.NewNPCFriendly(npc.NPCReference, 1)
				blocker.SetPos(candidate)
				blocker.SetFloor(0)
				controller.mapUnits = append(controller.mapUnits, blocker)
			}
			for _, purpose := range []npcMovementPurpose{npcMovementWander, npcMovementApproach, npcMovementSchedule} {
				assert.Equal(t, tc.want, controller.canOriginalNPCMoveTo(npc, candidate, purpose), "purpose %d", purpose)
			}
			controller.randomStepOriginal(npc, start, 3)
			want := start
			if tc.want {
				want = candidate
			}
			assert.Equal(t, want, npc.Pos())

			// The corridor prevents the planner from disguising incorrect layer
			// selection by taking a different route (tile zero has a higher cost).
			npc.SetPos(start)
			for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
				for _, y := range []references.Coordinate{9, 11} {
					position := references.Position{X: x, Y: y}
					layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
				}
			}
			assert.Equal(t, tc.want, controller.createFreshPathToScheduledLocation(npc))
			if tc.want {
				assert.Equal(t, []references.Position{candidate, goal}, npc.MapUnitDetails().CurrentPath)
			}
		})
	}
}
