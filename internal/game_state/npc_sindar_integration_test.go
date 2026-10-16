package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Resolve the installed actor slot independently of its one-based conversation
// number. These assertions read configured game references, never a generated dump.
func newSindarScheduleState(t *testing.T) (*GameState, *map_units.NPCFriendly) {
	t.Helper()
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Trinsic).
		WithPlayerAt(15, 15).
		WithRandomSeed(19).
		Build()
	location := state.GameReferences.LocationReferences.GetLocationReference(references.Trinsic)
	program, found := state.GameReferences.ConversationCatalog.Original(semantic.OriginalSourceTowne, 32)
	require.True(t, found, "installed town conversation 32 must exist")
	definition, err := program.Definition()
	require.NoError(t, err)
	require.NotEmpty(t, definition.Name)
	name, ok := definition.Name[0].(*semantic.SayAction)
	require.True(t, ok)
	require.NotEmpty(t, name.Content)
	text, ok := name.Content[0].(*semantic.TextPart)
	require.True(t, ok)
	require.Contains(t, text.Text, "Sindar")

	actorSlot := -1
	for slot, reference := range *location.GetNPCReferences() {
		if reference.DialogNumber == 32 {
			require.Equal(t, -1, actorSlot, "Sindar's Trinsic actor must be unambiguous")
			actorSlot = slot
		}
	}
	require.GreaterOrEqual(t, actorSlot, 0)
	require.NotEqual(t, 32, actorSlot, "conversation identity is not the actor index")
	var sindar *map_units.NPCFriendly
	for _, unit := range *state.CurrentNPCAIController.GetNpcs() {
		if friendly, friendlyOK := unit.(*map_units.NPCFriendly); friendlyOK && friendly.MapUnitDetails().NPCNum == actorSlot {
			sindar = friendly
			break
		}
	}
	require.NotNil(t, sindar)
	require.Equal(t, (*location.GetNPCReferences())[actorSlot], sindar.NPCReference)
	// Bind the reported itinerary to the configured installation. A different
	// data edition fails this fixture rather than silently claiming reproduction.
	require.Equal(t, references.NPCSchedule{
		Ai:    [3]byte{byte(references.Fixed), byte(references.Fixed), byte(references.BigWander)},
		X:     [3]byte{22, 25, 29},
		Y:     [3]byte{25, 7, 2},
		Floor: [3]byte{0, 1, 1},
		Time:  [4]byte{6, 0, 1, 5},
	}, sindar.NPCReference.Schedule)
	day := sindar.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(0)
	require.Contains(t, []indexes.SpriteIndex{indexes.LeftBed, indexes.RightBed}, location.GetTileNumber(day.Position, day.Floor))
	for _, slot := range []references.NPCScheduleSlot{0, 1, 2} {
		anchor := sindar.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(slot)
		tile := state.GameReferences.TileReferences.GetTile(location.GetTileNumber(anchor.Position, anchor.Floor))
		require.NotNil(t, tile)
		t.Logf("installed Sindar anchor: slot=%d pos=%v floor=%d tile=%d/%s ordinaryNPCTerrain=%t",
			slot, anchor.Position, anchor.Floor, tile.Index, tile.Name, tile.IsNPCTerrainPassable())
	}
	// The configured-data run established a fireplace, not ordinary walkable
	// terrain, at slot 1. NPC-0004 permits the exact scheduled destination;
	// NPC-0002 hidden relocation does not require a visible-floor walking route.
	upper := sindar.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(1)
	require.Equal(t, indexes.SpriteIndex(indexes.Fireplace), location.GetTileNumber(upper.Position, upper.Floor))
	state.DateTime.Hour, state.DateTime.Minute = 23, 50
	state.CurrentNPCAIController.HardResetNPCs(false)
	t.Logf("installed Sindar: Trinsic actor=%d dialogue=%d schedule=%+v bed=%v floor=%d",
		actorSlot, sindar.NPCReference.DialogNumber, sindar.NPCReference.Schedule, day.Position, day.Floor)

	return state, sindar
}

func passSindarScheduleMinute(t *testing.T, state *GameState, sindar *map_units.NPCFriendly) {
	t.Helper()
	before := state.DateTime
	previous, previousFloor, previousArrival := sindar.Pos(), sindar.Floor(), sindar.ArrivedScheduleSlot()
	state.BeginSurfaceCommandRouting()
	require.True(t, state.ActionPass())
	state.EndSurfaceCommandRouting()
	state.CompleteSurfacePassCycle(references.SmallMapType)
	before.Advance(1)
	require.Equal(t, before, state.DateTime, "ordinary Pass must advance exactly one minute")
	slot := sindar.NPCReference.Schedule.DesiredSlot(state.DateTime)
	goal := sindar.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(slot)
	observedFloor := state.MapState.PlayerLocation.Floor
	branch := "settled"
	if previousArrival != slot {
		switch {
		case previousFloor != observedFloor && goal.Floor != observedFloor:
			branch = "hidden relocation"
		case previousFloor != observedFloor:
			branch = "receiving entry"
		case goal.Floor != observedFloor:
			branch = "departure ladder"
		default:
			branch = "observed walk"
		}
	}
	t.Logf("%02d:%02d actor=%d branch=%s pos=%v/%d->%v/%d desired=%d arrived=%d->%d route=%v",
		state.DateTime.Hour, state.DateTime.Minute, sindar.MapUnitDetails().NPCNum, branch,
		previous, previousFloor, sindar.Pos(), sindar.Floor(), slot, previousArrival, sindar.ArrivedScheduleSlot(),
		sindar.MapUnitDetails().CurrentPath)
	if sindar.Floor() == observedFloor {
		require.NotEqual(t, state.MapState.PlayerLocation.Position, sindar.Pos(), "Sindar cannot overlap the party")
		for _, unit := range *state.CurrentNPCAIController.GetNpcs() {
			if unit != sindar && unit.Floor() == observedFloor {
				require.NotEqual(t, unit.Pos(), sindar.Pos(), "Sindar cannot overlap actor %d", unit.MapUnitDetails().NPCNum)
			}
		}
	}
}

func requireSindarReceivingRoute(t *testing.T, state *GameState, sindar *map_units.NPCFriendly) {
	t.Helper()
	goal := sindar.NPCReference.Schedule.GetIndividualNPCBehaviourByUltimaDate(state.DateTime)
	path := sindar.MapUnitDetails().CurrentPath
	require.NotEmpty(t, path)
	require.Equal(t, goal.Floor, sindar.Floor())
	require.Equal(t, goal.Position, path[len(path)-1])
	previous := sindar.Pos()
	for _, next := range path {
		require.Contains(t, previous.Neighbors(), next, "receiving route must contain only cardinal steps")
		require.GreaterOrEqual(t, next.X, references.Coordinate(0))
		require.Less(t, next.X, references.XSmallMapTiles)
		require.GreaterOrEqual(t, next.Y, references.Coordinate(0))
		require.Less(t, next.Y, references.YSmallMapTiles)
		tile := state.GetCurrentLayeredMap().GetSmallMapPersistentTerrainTile(&next, state.DateTime.Hour)
		require.NotNil(t, tile)
		// Destination terrain has an explicit exception; intermediate terrain does not.
		require.True(t, next == goal.Position || tile.IsNPCTerrainPassable(),
			"receiving route crosses forbidden terrain %s at %v", tile.Name, next)
		t.Logf("Sindar receiving route: %v -> %v terrain=%d/%s destination=%t",
			previous, next, tile.Index, tile.Name, next == goal.Position)
		previous = next
	}
}

func TestSindarInstalledOvernightPassTravel(t *testing.T) {
	state, sindar := newSindarScheduleState(t)
	controller := state.CurrentNPCAIController
	player := state.MapState.PlayerLocation
	departed, entered, arrived := false, false, false
	seenHiddenSlots := make(map[references.NPCScheduleSlot]bool)
	visibleSteps := 0
	// 23:50 through 07:30, without sleeping, changing floors or reloading NPCs.
	for range 460 {
		previous, previousFloor, previousArrival := sindar.Pos(), sindar.Floor(), sindar.ArrivedScheduleSlot()
		passSindarScheduleMinute(t, state, sindar)
		require.Same(t, controller, state.CurrentNPCAIController)
		require.Equal(t, player, state.MapState.PlayerLocation)
		slot := sindar.NPCReference.Schedule.DesiredSlot(state.DateTime)
		goal := sindar.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(slot)
		if state.DateTime.Minute == 0 {
			expectedSlots := map[byte]references.NPCScheduleSlot{0: 1, 1: 2, 5: 1, 6: 0}
			if expected, changed := expectedSlots[state.DateTime.Hour]; changed {
				require.Equal(t, expected, slot, "installed itinerary selection at %02d:00", state.DateTime.Hour)
			}
		}
		switch {
		case previousFloor == 0 && sindar.Floor() != 0:
			departureTile := state.GetCurrentLayeredMap().GetSmallMapPersistentTerrainTile(&previous, state.DateTime.Hour)
			require.True(t, references.IsSpecificLadderOrStairs(departureTile.Index, references.LadderOrStairUp))
			require.Equal(t, goal.Position, sindar.Pos())
			require.Equal(t, slot, sindar.ArrivedScheduleSlot())
			departed = true
		case previousFloor != 0 && sindar.Floor() == 0:
			entry := sindar.Pos()
			entryTile := state.GetCurrentLayeredMap().GetSmallMapPersistentTerrainTile(&entry, state.DateTime.Hour)
			require.True(t, references.IsSpecificLadderOrStairs(entryTile.Index, references.LadderOrStairUp))
			require.NotEqual(t, goal.Position, entry, "morning return must not teleport to bed")
			require.Equal(t, previousArrival, sindar.ArrivedScheduleSlot())
			requireSindarReceivingRoute(t, state, sindar)
			entered = true
		case sindar.Floor() != 0 && goal.Floor != 0:
			require.Equal(t, goal.Position, sindar.Pos(), "hidden schedule change must relocate, not wander")
			require.Equal(t, slot, sindar.ArrivedScheduleSlot())
			require.Empty(t, sindar.MapUnitDetails().CurrentPath)
			seenHiddenSlots[slot] = true
		case sindar.Floor() == 0 && previous != sindar.Pos():
			require.Contains(t, previous.Neighbors(), sindar.Pos(), "visible movement must be an adjacent step")
			position := sindar.Pos()
			tile := state.GetCurrentLayeredMap().GetSmallMapPersistentTerrainTile(&position, state.DateTime.Hour)
			require.NotNil(t, tile)
			beach := tile.Index >= indexes.OutsidePath1 && tile.Index <= indexes.OutsidePath4
			require.True(t, (goal.Floor == sindar.Floor() && position == goal.Position) || beach || tile.IsNPCTerrainPassable(),
				"visible travel cannot cross forbidden terrain %s at %v", tile.Name, position)
			require.False(t, goal.Floor != sindar.Floor() && tile.IsChair(), "departure steps cannot cross chairs")
			if entered {
				visibleSteps++
			}
		}
		if entered && sindar.Pos() == goal.Position && sindar.Floor() == goal.Floor {
			require.Equal(t, slot, sindar.ArrivedScheduleSlot())
			arrived = true
		}
		if entered && !arrived {
			require.Equal(t, references.NPCScheduleSlot(1), sindar.ArrivedScheduleSlot(), "entry alone is not arrival")
		}
	}
	assert.True(t, departed, "night departure must finish before the morning return")
	assert.True(t, seenHiddenSlots[1])
	assert.True(t, seenHiddenSlots[2], "the same-hidden-floor 01:00 relocation must occur")
	assert.True(t, entered, "must observe morning entry on the ground floor")
	assert.True(t, arrived, "must walk from entry to bed within the bounded morning observation")
	assert.Positive(t, visibleSteps)
	assert.Equal(t, references.NPCScheduleSlot(0), sindar.ArrivedScheduleSlot())
}

func TestSindarPlayerFloorChangeResetsPendingTravel(t *testing.T) {
	state, sindar := newSindarScheduleState(t)
	// Advance the same continuous Pass sequence up to the actual morning entry.
	entered := false
	for range 430 {
		before := sindar.Floor()
		passSindarScheduleMinute(t, state, sindar)
		if before != 0 && sindar.Floor() == 0 {
			entered = true
			break
		}
	}
	require.True(t, entered)
	requireSindarReceivingRoute(t, state, sindar)
	require.Equal(t, references.NPCScheduleSlot(1), sindar.ArrivedScheduleSlot())
	controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	require.True(t, ok)
	desired := sindar.NPCReference.Schedule.DesiredSlot(state.DateTime)
	goal := sindar.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(desired)
	// The real player-floor path deliberately resets all schedules. Retained
	// dialogue and override state must survive that reset on the same actor.
	sindar.NPCReference.Schedule.Ai[1] = byte(references.Talk3)
	require.True(t, sindar.ApplyTalk3OneShot())
	sindar.MapUnitDetails().SetOverriddenAiType(references.Fixed)
	require.True(t, state.GoUpOneFloor())
	require.Same(t, controller, state.CurrentNPCAIController)
	require.Equal(t, goal.Position, sindar.Pos())
	require.Equal(t, goal.Floor, sindar.Floor())
	require.Equal(t, desired, sindar.ArrivedScheduleSlot())
	require.Empty(t, sindar.MapUnitDetails().CurrentPath)
	require.Equal(t, references.Fixed, sindar.MapUnitDetails().GetOverriddenAiType())
	require.Equal(t, byte(references.Wander3), sindar.NPCReference.Schedule.Ai[1])
	require.True(t, state.GoDownOneFloor())
	assert.Equal(t, goal.Position, sindar.Pos())
	assert.Equal(t, desired, sindar.ArrivedScheduleSlot())
}
