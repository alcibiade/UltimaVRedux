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

func TestCombatExitRouteRequiresSameDungeonRoomExit(t *testing.T) {
	combat := &CombatState{
		ExitContext: CombatExitDungeonRoom,
		ExitRoute:   CombatExitNone,
	}

	assert.True(t, combat.establishExitRoute(CombatExitUp))
	assert.Equal(t, CombatExitUp, combat.ExitRoute)
	assert.True(t, combat.establishExitRoute(CombatExitUp))
	assert.False(t, combat.establishExitRoute(CombatExitDown))
	assert.Equal(t, CombatExitUp, combat.ExitRoute)
}

func TestCombatTransitionRecordsDungeonExitContext(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(42).
		Build()

	require.NoError(t, gs.TransitionToCombatMap(
		0,
		references.DungeonCombatTerritory,
		CombatEntryNorth,
	))
	assert.Equal(t, CombatExitDungeonEncounter, gs.CombatScene.ExitContext)
	gs.TransitionFromCombatMap()

	require.NoError(t, gs.TransitionToDungeonCombatMap(
		references.DeceitDungeon,
		0,
		CombatEntryNorth,
	))
	assert.Equal(t, CombatExitDungeonRoom, gs.CombatScene.ExitContext)
}

func TestVictoryFanfareFadesWhenLeavingCombatMap(t *testing.T) {
	for _, test := range []struct {
		name    string
		victory bool
		reason  CombatEndReason
		want    []SoundStopRequest
	}{
		{name: "victory", victory: true, reason: CombatVictory,
			want: []SoundStopRequest{{Effect: SoundCombatVictory, FadeOutMs: combatVictoryExitFadeMs}}},
		{name: "without victory", victory: false, reason: CombatEvacuated, want: []SoundStopRequest{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := newCombatExitTestState(t)
			state.CombatScene.Victory = test.victory

			state.EndCombat(test.reason)

			assert.Equal(t, test.want, callbacks.StopRequests)
			assert.Nil(t, state.CombatScene)
		})
	}
}

func TestCombatExitProcessesCrossedProvisionHours(t *testing.T) {
	state, _ := newCombatExitTestState(t)
	state.DateTime.Hour, state.DateTime.Minute = 5, 59
	state.lastProcessedHour = 5
	state.PartyState.Inventory.Provisions.Food.Set(100)
	foodBefore := state.PartyState.Inventory.Provisions.Food.Get()
	state.AdvanceGameplayMinutes(1)
	require.NotEmpty(t, state.pendingProvisionHours)

	state.EndCombat(CombatVictory)

	assert.Empty(t, state.pendingProvisionHours)
	assert.Less(t, state.PartyState.Inventory.Provisions.Food.Get(), foodBefore)
}

func TestCombatEdgeExitRejectsDifferentDungeonRoomDirection(t *testing.T) {
	gs, mock := newCombatExitTestState(t)
	gs.CombatScene.ExitContext = CombatExitDungeonRoom
	firstPC := gs.CombatScene.GetActiveCombatantIndex()
	gs.CombatScene.SetPartyMemberPosition(firstPC, references.Position{X: 0, Y: 5})

	assert.Equal(t, CombatMoveFled, gs.MoveCombatCharacter(references.Left))
	assert.Equal(t, CombatExitWest, gs.CombatScene.ExitRoute)

	secondPC := firstActiveCombatPCExcept(gs.CombatScene, firstPC)
	require.NotEqual(t, -1, secondPC)
	gs.CombatScene.CurrentTurnIndex = secondPC
	gs.CombatScene.SetPartyMemberPosition(secondPC, references.Position{X: 10, Y: 5})
	mock.Reset()

	assert.Equal(t, CombatMoveBlocked, gs.MoveCombatCharacter(references.Right))
	assert.True(t, gs.CombatScene.PartyActive[secondPC])
	assert.Contains(t, mock.Messages, "All must use the same exit!")
}

func TestCombatEdgeExitRejectsIndividualDepartureFromFrigate(t *testing.T) {
	gs, mock := newCombatExitTestState(t)
	activePC := gs.CombatScene.GetActiveCombatantIndex()
	gs.CombatScene.SetPartyMemberPosition(activePC, references.Position{X: 0, Y: 5})
	npcRef := references.NPCReference{}
	npcRef.NpcType = references.Vehicle
	gs.PartyVehicle = *map_units.NewNPCFriendlyVehicle(references.FrigateVehicle, npcRef)
	mock.Reset()

	assert.Equal(t, CombatMoveBlocked, gs.MoveCombatCharacter(references.Left))
	assert.True(t, gs.CombatScene.PartyActive[activePC])
	assert.Equal(t, CombatExitNone, gs.CombatScene.ExitRoute)
	assert.Contains(t, mock.Messages, "Stay with ship!")
}

func TestCombatKlimbEstablishesRouteWithoutAutomaticPartyExit(t *testing.T) {
	gs, _ := newCombatExitTestState(t)
	gs.CombatScene.ExitContext = CombatExitDungeonRoom
	activePC := gs.CombatScene.GetActiveCombatantIndex()
	activePosition := gs.CombatScene.GetPartyMemberPosition(activePC)
	gs.GetCurrentLayeredMap().SetTileByLayer(
		map_state.MapLayer,
		&activePosition,
		indexes.LadderUp,
	)
	activeBefore := gs.CombatScene.countActivePartyMembers()

	require.True(t, gs.ActionKlimbCombatMap(references.Up))

	assert.Equal(t, CombatExitUp, gs.CombatScene.ExitRoute)
	assert.False(t, gs.CombatScene.PartyActive[activePC])
	assert.Equal(t, activeBefore-1, gs.CombatScene.countActivePartyMembers())
	assert.NotNil(t, gs.CombatScene, "one climber must not automatically evacuate the party")
}

func TestDungeonRoomGrateCombatKlimbEstablishesDownRoute(t *testing.T) {
	gs, _ := newCombatExitTestState(t)
	gs.CombatScene.ExitContext = CombatExitDungeonRoom
	activePC := gs.CombatScene.GetActiveCombatantIndex()
	activePosition := gs.CombatScene.GetPartyMemberPosition(activePC)
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &activePosition, indexes.Grate)

	assert.Equal(t, CombatKlimbCompleted, gs.BeginCombatKlimb())
	assert.Equal(t, CombatExitDown, gs.CombatScene.ExitRoute)
	assert.False(t, gs.CombatScene.PartyActive[activePC])
}

func TestGeneratedDungeonDualLadderCombatKlimbChoosesVerticalRoute(t *testing.T) {
	tests := []struct {
		name      string
		direction references.Direction
		want      CombatExitRoute
	}{
		{name: "up", direction: references.Up, want: CombatExitUp},
		{name: "down", direction: references.Down, want: CombatExitDown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, _ := newCombatExitTestState(t)
			var cells references.DungeonCells
			source := references.Position{X: 4, Y: 4}
			require.True(t, cells.SetCell(0, source, references.DungeonCell(0x30)))
			combatMap, err := references.BuildGeneratedDungeonCombatMap(
				references.Deceit, &cells, 0, source, references.DungeonFacingNorth,
			)
			require.NoError(t, err)
			gs.CombatScene.CombatMapRef = combatMap
			activePC := gs.CombatScene.GetActiveCombatantIndex()
			center := references.Position{X: 5, Y: 5}
			gs.CombatScene.SetPartyMemberPosition(activePC, center)
			gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &center, indexes.LadderUp)

			require.True(t, gs.ActionKlimbCombatMap(test.direction))

			assert.Equal(t, test.want, gs.CombatScene.ExitRoute)
		})
	}
}

func TestVictoryEscapeDungeonRoomRequiresEstablishedRoute(t *testing.T) {
	gs, mock := newCombatExitTestState(t)
	gs.CombatScene.ExitContext = CombatExitDungeonRoom
	gs.CombatScene.Victory = true
	mock.Reset()

	assert.False(t, gs.ActionEscapeCombatMap())
	assert.NotNil(t, gs.CombatScene)
	assert.Contains(t, mock.Messages, "Choose an exit first.")
}

func TestVictoryEscapeAppliesEstablishedDungeonLadderRoute(t *testing.T) {
	gs, _ := newCombatExitTestState(t)
	gs.CombatScene.ExitContext = CombatExitDungeonRoom
	gs.CombatScene.ExitRoute = CombatExitUp
	gs.CombatScene.Victory = true
	attachCombatExitDungeonAuthority(t, gs, references.Deceit)
	gs.CombatScene.PreCombatPlayerLocation = references.PlayerLocation{
		Location: references.Deceit,
		Position: references.Position{X: 3, Y: 4},
		Floor:    3,
	}

	require.True(t, gs.ActionEscapeCombatMap())

	assert.Nil(t, gs.CombatScene)
	assert.Equal(t, references.Deceit, gs.MapState.PlayerLocation.Location)
	assert.Equal(t, references.FloorNumber(2), gs.MapState.PlayerLocation.Floor)
	assert.Equal(t, references.Position{X: 3, Y: 4}, gs.MapState.PlayerLocation.Position)
	snapshot, active := gs.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, references.DungeonArrivedFromBelow, snapshot.LastDirection)
}

func TestCombatExitAppliesCardinalRouteOnlyToDungeonRooms(t *testing.T) {
	gs, _ := newCombatExitTestState(t)
	originalPosition := references.Position{X: 3, Y: 4}
	gs.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.Deceit,
		Position: originalPosition,
		Floor:    3,
	}
	attachCombatExitDungeonAuthority(t, gs, references.Deceit)

	gs.applyCombatExitRoute(CombatExitDungeonEncounter, CombatExitWest)
	assert.Equal(t, originalPosition, gs.MapState.PlayerLocation.Position)

	gs.applyCombatExitRoute(CombatExitDungeonRoom, CombatExitWest)
	assert.Equal(t, references.Position{X: 2, Y: 4}, gs.MapState.PlayerLocation.Position)
	snapshot, active := gs.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, references.DungeonFacingWest, snapshot.Facing)
	assert.Equal(t, references.DungeonLastWest, snapshot.LastDirection)
}

func TestCombatLadderBoundaryExitReturnsToDungeonEntrance(t *testing.T) {
	gs, _ := newCombatExitTestState(t)
	expectedPosition := gs.GetDungeonEntranceCoordinates()[references.Deceit]
	attachCombatExitDungeonAuthority(t, gs, references.Deceit)
	gs.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.Deceit,
		Position: references.Position{X: 3, Y: 4},
		Floor:    0,
	}

	gs.applyCombatExitRoute(CombatExitDungeonRoom, CombatExitUp)

	assert.Equal(t, references.BritanniaUnderworld, gs.MapState.PlayerLocation.Location)
	assert.Equal(t, expectedPosition, gs.MapState.PlayerLocation.Position)
	assert.Equal(t, references.FloorNumber(references.OVERWORLD), gs.MapState.PlayerLocation.Floor)

	attachCombatExitDungeonAuthority(t, gs, references.Deceit)
	gs.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.Deceit,
		Position: references.Position{X: 3, Y: 4},
		Floor:    combatMaxDungeonFloor,
	}
	gs.applyCombatExitRoute(CombatExitDungeonRoom, CombatExitDown)
	assert.Equal(t, references.BritanniaUnderworld, gs.MapState.PlayerLocation.Location)
	assert.Equal(t, expectedPosition, gs.MapState.PlayerLocation.Position)
	assert.Equal(t, references.FloorNumber(references.UNDERWORLD), gs.MapState.PlayerLocation.Floor)
}

func TestVictoryEndsWhenLastPCLeavesEstablishedRoute(t *testing.T) {
	gs, _ := newCombatExitTestState(t)
	gs.CombatScene.Victory = true
	activePC := gs.CombatScene.GetActiveCombatantIndex()
	for index := range gs.CombatScene.PartyActive {
		if index == activePC || !gs.CombatScene.PartyActive[index] {
			continue
		}
		gs.CombatScene.PartyActive[index] = false
		gs.CombatScene.TurnQueue.MarkFled(index, true)
	}
	gs.CombatScene.SetPartyMemberPosition(activePC, references.Position{X: 0, Y: 5})

	require.Equal(t, CombatMoveFled, gs.MoveCombatCharacter(references.Left))
	gs.AdvanceCombatTurn()

	assert.Nil(t, gs.CombatScene)
}

// TestSelectedPCDepartureClearsSoloSelection prevents AdvanceCombatTurn from
// looping forever while looking for a solo-selected PC that already fled.
func TestSelectedPCDepartureClearsSoloSelection(t *testing.T) {
	gs, _ := newCombatExitTestState(t)
	departingPC := gs.CombatScene.GetActiveCombatantIndex()
	remainingPC := firstActiveCombatPCExcept(gs.CombatScene, departingPC)
	require.NotEqual(t, -1, remainingPC)
	require.True(t, gs.SetActivePlayer(departingPC))
	gs.CombatScene.SetSoloMoveCharacter(departingPC)
	gs.CombatScene.MonsterManager.ForEach(func(index int, _ *CombatMonster) {
		gs.CombatScene.TurnQueue.MarkDead(index, false)
	})
	clearCombatMonstersForTest(gs.CombatScene)
	gs.CombatScene.Victory = true
	gs.CombatScene.SetPartyMemberPosition(departingPC, references.Position{X: 0, Y: 5})

	require.Equal(t, CombatMoveFled, gs.MoveCombatCharacter(references.Left))

	assert.False(t, gs.CombatScene.IsSoloMoveActive())
	assert.False(t, gs.HasActivePlayer())
	gs.FinishTurn()
	gs.AdvanceCombatTurn()
	assert.True(t, gs.HasActivePlayer())
	assert.NotEqual(t, departingPC, gs.GetActivePlayerIndex())
}

func newCombatExitTestState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	gs, mock := newBritainCombatTestState(t, 42)

	return gs, mock
}

func attachCombatExitDungeonAuthority(t *testing.T, state *GameState, dungeon references.Location) {
	t.Helper()
	active, err := state.PrepareDungeonState(
		dungeon,
		references.DungeonFacingEast,
		references.DungeonArrivedFromAbove,
	)
	require.NoError(t, err)
	state.activeDungeon = active
}

func firstActiveCombatPCExcept(combat *CombatState, excluded int) int {
	for index, active := range combat.PartyActive {
		if active && index != excluded {
			return index
		}
	}

	return -1
}
