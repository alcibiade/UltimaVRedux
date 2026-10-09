package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestDungeonKlimbOptionsUseCurrentSemanticCell(t *testing.T) {
	tests := []struct {
		name             string
		cell             references.DungeonCell
		grapple          bool
		wantUp           bool
		wantDown         bool
		wantNeedsGrapple bool
	}{
		{name: "no route", cell: 0x00},
		{name: "up ladder", cell: 0x10, wantUp: true},
		{name: "down ladder", cell: 0x20, wantDown: true},
		{name: "both ladder", cell: 0x30, wantUp: true, wantDown: true},
		{name: "pit", cell: 0x60, wantDown: true},
		{name: "opening without grapple", cell: 0x08, wantNeedsGrapple: true},
		{name: "opening with grapple", cell: 0x08, grapple: true, wantUp: true},
		{name: "pit opening with grapple", cell: 0x68, grapple: true, wantUp: true, wantDown: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var cells references.DungeonCells
			cells.SetCell(0, references.Position{X: 1, Y: 1}, test.cell)
			state := newDungeonStateTestGame(t, cells)
			if test.grapple {
				state.PartyState.Inventory.QuestItems.Set(references.Grapple, 1)
			}

			options := state.DungeonKlimbOptionsAtCurrentCell()

			assert.Equal(t, test.wantUp, options.Up)
			assert.Equal(t, test.wantDown, options.Down)
			assert.Equal(t, test.wantNeedsGrapple, options.GrappleRequired)
		})
	}
}

func TestDungeonKlimbBlocksTargetClassesBThroughE(t *testing.T) {
	for kind := references.DungeonPassage; kind <= references.DungeonLiveRoom; kind++ {
		t.Run(fmt.Sprintf("kind_%X", kind), func(t *testing.T) {
			var cells references.DungeonCells
			position := references.Position{X: 1, Y: 1}
			cells.SetCell(3, position, 0x10)
			cells.SetCell(2, position, references.DungeonCell(byte(kind)<<4))
			state := newDungeonStateTestGame(t, cells)
			state.MapState.PlayerLocation.Floor = 3

			plan := state.planDungeonKlimb(DungeonKlimbUp)

			assert.Equal(t, kind >= references.DungeonWall && kind <= references.DungeonRevealedDoor, plan.blocked)
			assert.Equal(t, DungeonFullEndPlayer, plan.completion)
		})
	}
}

func TestDungeonKlimbQueuesLiveAndClearedRoomLandings(t *testing.T) {
	for _, room := range []references.DungeonCell{0xa1, 0xf1} {
		t.Run(fmt.Sprintf("room_%X", room.Kind()), func(t *testing.T) {
			var cells references.DungeonCells
			position := references.Position{X: 1, Y: 1}
			require.True(t, cells.SetCell(0, position, references.DungeonCell(0x20)))
			require.True(t, cells.SetCell(1, position, room))
			state := newDungeonStateTestGame(t, cells)
			state.ShadowlordState = references.NewShadowlordState()
			state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 1}

			token, _ := state.BeginDungeonCommandCycle()
			plan := state.planDungeonKlimb(DungeonKlimbDown)
			state.claimDungeonCommandAction(plan.completion)
			state.executeDungeonKlimbPlan(plan)
			state.CompleteDungeonCommandCycle(token, plan.completion)

			assert.Equal(t, references.FloorNumber(1), state.MapState.PlayerLocation.Floor)
			assert.True(t, state.activeDungeon.roomContinuation.encounterPending)
			assert.Equal(t, 1, state.ActiveDurationEffect.TurnsRemaining)
		})
	}
}

func TestDungeonKlimbInternalTravelPreservesPositionAndFacing(t *testing.T) {
	tests := []struct {
		name              string
		direction         DungeonKlimbDirection
		cell              references.DungeonCell
		wantFloor         references.FloorNumber
		wantLastDirection references.DungeonLastDirection
	}{
		{
			name: "up", direction: DungeonKlimbUp, cell: 0x10,
			wantFloor: 2, wantLastDirection: references.DungeonArrivedFromBelow,
		},
		{
			name: "down", direction: DungeonKlimbDown, cell: 0x20,
			wantFloor: 4, wantLastDirection: references.DungeonArrivedFromAbove,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var cells references.DungeonCells
			position := references.Position{X: 4, Y: 5}
			cells.SetCell(3, position, test.cell)
			state := newDungeonStateTestGame(t, cells)
			state.MapState.PlayerLocation.Position = position
			state.MapState.PlayerLocation.Floor = 3
			state.ShadowlordState = references.NewShadowlordState()
			beforeFacing := state.activeDungeon.facing

			plan := state.planDungeonKlimb(test.direction)
			state.executeDungeonKlimbPlan(plan)

			assert.Equal(t, test.wantFloor, state.MapState.PlayerLocation.Floor)
			assert.Equal(t, position, state.MapState.PlayerLocation.Position)
			assert.Equal(t, beforeFacing, state.activeDungeon.facing)
			assert.Equal(t, test.wantLastDirection, state.activeDungeon.lastDirection)
		})
	}
}

func TestDungeonKlimbSuccessfulFloorChangeGeneratesExactlyOnce(t *testing.T) {
	const seed = 1265
	var cells references.DungeonCells
	position := references.Position{X: 1, Y: 1}
	require.True(t, cells.SetCell(3, position, 0x10))
	state := newDungeonStateTestGame(t, cells)
	state.MapState.PlayerLocation.Floor = 3
	state.SetRandomSeed(seed)
	state.ShadowlordState = references.NewShadowlordState()
	plan := state.planDungeonKlimb(DungeonKlimbUp)

	expected := newDungeonStateTestGame(t, cells)
	expected.MapState.PlayerLocation.Floor = 2
	expected.SetRandomSeed(seed)
	expected.generateDungeonWanderer()
	require.NotNil(t, expected.activeDungeon.wanderer)

	state.executeDungeonKlimbPlan(plan)

	actualSnapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	expectedSnapshot, active := expected.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, expectedSnapshot.Wanderer, actualSnapshot.Wanderer)
	assert.Equal(t, expected.GetRNG().Calls(), state.GetRNG().Calls())
}

func TestDungeonKlimbBlockedFloorChangeDoesNotGenerate(t *testing.T) {
	var cells references.DungeonCells
	position := references.Position{X: 1, Y: 1}
	require.True(t, cells.SetCell(3, position, 0x10))
	require.True(t, cells.SetCell(2, position, 0xb0))
	state := newDungeonStateTestGame(t, cells)
	state.MapState.PlayerLocation.Floor = 3
	plan := state.planDungeonKlimb(DungeonKlimbUp)
	require.True(t, plan.blocked)
	beforeCalls := state.GetRNG().Calls()

	state.executeDungeonKlimbPlan(plan)

	assert.Nil(t, state.activeDungeon.wanderer)
	assert.Equal(t, beforeCalls, state.GetRNG().Calls())
}

func TestDungeonKlimbBoundaryExitsUseRoutePlaneAndClearAuthority(t *testing.T) {
	tests := []struct {
		name           string
		floor          references.FloorNumber
		direction      DungeonKlimbDirection
		wantWorldFloor references.FloorNumber
	}{
		{name: "floor zero up to Britannia", floor: 0, direction: DungeonKlimbUp, wantWorldFloor: 0},
		{name: "floor seven down to Underworld", floor: 7, direction: DungeonKlimbDown, wantWorldFloor: -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := newRealDungeonKlimbRouteState(t, test.floor, test.direction)
			dungeon := state.MapState.PlayerLocation.Location
			state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}
			callbacks.Reset()

			token, start := state.BeginDungeonCommandCycle()
			state.ActionKlimbDungeonMap(token, test.direction)
			assert.True(t, callbacks.InputBlocked)
			runDungeonWorkflow(t, state, "dungeon_klimb")

			assert.Equal(t, 1, start.ElapsedMinutes)
			assert.Equal(t, references.BritanniaUnderworld, state.MapState.PlayerLocation.Location)
			assert.Equal(t, test.wantWorldFloor, state.MapState.PlayerLocation.Floor)
			assert.Equal(t, state.GetDungeonEntranceCoordinates()[dungeon], state.MapState.PlayerLocation.Position)
			_, active := state.DungeonStateSnapshot()
			assert.False(t, active)
			assert.True(t, state.IsAtDungeonCommandBoundary())
			assert.Equal(t, 5, state.ActiveDurationEffect.TurnsRemaining,
				"boundary exit must not run dungeon end-player processing")
			assert.False(t, callbacks.InputBlocked)
			require.NotEmpty(t, callbacks.MapFadeLevels)
			assert.Zero(t, callbacks.MapFadeLevels[len(callbacks.MapFadeLevels)-1])
		})
	}
}

func TestDungeonKlimbDownOntoDeceitHiddenPitRunsPitTrap(t *testing.T) {
	position := references.Position{X: 1, Y: 3}
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithFloor(0).
		WithPlayerAt(int(position.X), int(position.Y)).
		WithRandomSeed(253).
		WithSystemCallbacks().
		Build()
	before, active := state.DungeonStateSnapshot()
	require.True(t, active)
	current, ok := before.Cells.Cell(0, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0x60), current)
	landing, ok := before.Cells.Cell(1, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0x69), landing)

	activeParty := 0
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		if character.PartyMap != party_state.PartyMapActive {
			continue
		}
		activeParty++
		character.SetStatus(party_state.Good)
		character.SetMaxHp(200)
		character.SetHp(200)
	}
	state.ActiveDurationEffect = DurationEffect{
		EffectType:     DurationEffectProtection,
		TurnsRemaining: 5,
	}
	callbacks.Reset()
	beforeRNGCalls := state.GetRNG().Calls()

	token, start := state.BeginDungeonCommandCycle()
	state.ActionKlimbDungeonMap(token, DungeonKlimbDown)

	assert.True(t, state.EventQueueManager.HasWorkflow("dungeon_klimb"))
	assert.True(t, callbacks.InputBlocked)
	assert.False(t, state.IsAtDungeonCommandBoundary())
	runDungeonWorkflow(t, state, "dungeon_klimb")

	after, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, 1, start.ElapsedMinutes)
	assert.Equal(t, references.FloorNumber(7), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, position, state.MapState.PlayerLocation.Position)
	assert.Equal(t, references.DungeonArrivedFromAbove, after.LastDirection)
	for floor := references.FloorNumber(1); floor <= 6; floor++ {
		cell, cellOK := after.Cells.Cell(floor, position)
		require.True(t, cellOK)
		assert.Equal(t, references.DungeonCell(0x68), cell, "displayed level %d", floor+1)
	}
	finalLanding, ok := after.Cells.Cell(7, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0xfa), finalLanding)
	assert.Equal(t, uint64(6*activeParty+6), state.GetRNG().Calls()-beforeRNGCalls,
		"ordinary-floor generation and actor movement precede damage; room landing defers regeneration")
	assert.True(t, state.activeDungeon.generateWandererAfterRoom)
	assert.Equal(t, 5, state.ActiveDurationEffect.TurnsRemaining,
		"end-player effects wait until the pit-landed room returns")
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, state.IsAtDungeonCommandBoundary())
}

func TestDungeonKlimbTravelsBetweenRealDataFloors(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithFloor(1).
		WithSystemCallbacks().
		Build()
	position, found := findRealDungeonKlimbPosition(state, 1, DungeonKlimbUp)
	require.True(t, found)
	state.MapState.PlayerLocation.Position = position
	beforePosition := state.MapState.PlayerLocation.Position
	beforeFacing, active := state.DungeonStateSnapshot()
	require.True(t, active)
	callbacks.Reset()

	token, _ := state.BeginDungeonCommandCycle()
	state.ActionKlimbDungeonMap(token, DungeonKlimbUp)
	runDungeonWorkflow(t, state, "dungeon_klimb")

	assert.Equal(t, references.FloorNumber(0), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, beforePosition, state.MapState.PlayerLocation.Position)
	after, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, beforeFacing.Facing, after.Facing)
	assert.Equal(t, references.DungeonArrivedFromBelow, after.LastDirection)
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func findRealDungeonKlimbPosition(
	state *GameState,
	floor references.FloorNumber,
	direction DungeonKlimbDirection,
) (references.Position, bool) {
	snapshot, active := state.DungeonStateSnapshot()
	if !active {
		return references.Position{}, false
	}
	for y := references.Coordinate(0); y < references.YDungeonMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XDungeonMapTiles; x++ {
			position := references.Position{X: x, Y: y}
			cell, ok := snapshot.Cells.Cell(floor, position)
			if !ok {
				return references.Position{}, false
			}
			up := cell.Kind() == references.DungeonLadderUp || cell.Kind() == references.DungeonLadderBoth
			down := cell.Kind() == references.DungeonLadderDown ||
				cell.Kind() == references.DungeonLadderBoth ||
				cell.Kind() == references.DungeonPitOrTrap
			if direction == DungeonKlimbUp && up || direction == DungeonKlimbDown && down {
				boundary := direction == DungeonKlimbUp && floor == 0 ||
					direction == DungeonKlimbDown && floor == references.DungeonFloorCount-1
				if !boundary {
					targetFloor := floor + 1
					if direction == DungeonKlimbUp {
						targetFloor = floor - 1
					}
					target, targetOK := snapshot.Cells.Cell(targetFloor, position)
					if !targetOK {
						return references.Position{}, false
					}
					if dungeonKlimbTargetBlocked(target.Kind()) {
						continue
					}
				}

				return position, true
			}
		}
	}

	return references.Position{}, false
}

func newRealDungeonKlimbRouteState(
	t *testing.T,
	floor references.FloorNumber,
	direction DungeonKlimbDirection,
) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	for dungeon := references.Deceit; dungeon <= references.Doom; dungeon++ {
		state, callbacks := NewIntegrationTestBuilder(t).
			WithLocation(dungeon).
			WithFloor(floor).
			WithSystemCallbacks().
			Build()
		position, found := findRealDungeonKlimbPosition(state, floor, direction)
		if found {
			state.MapState.PlayerLocation.Position = position

			return state, callbacks
		}
	}
	t.Fatalf("no real floor %d dungeon Klimb route", floor)

	return nil, nil
}
