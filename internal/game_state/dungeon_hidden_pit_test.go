package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDungeonHiddenPitTestGame(
	t *testing.T,
	cells references.DungeonCells,
	seed uint64,
) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state := newDungeonStateTestGame(t, cells)
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	state.SetRandomSeed(seed)
	state.ShadowlordState = references.NewShadowlordState()
	state.EventQueueManager = event_queue.NewEventQueueManager(
		func() int64 { return state.ElapsedMs },
		&eventQueueSystemCallbacks{gameState: state},
	)
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		character.SetMaxHp(100)
		character.SetHp(100)
	}
	callbacks.Reset()

	return state, callbacks
}

func startDungeonHiddenPitTestMove(
	t *testing.T,
	state *GameState,
	movement DungeonMovement,
) DungeonMoveResult {
	t.Helper()
	token, _ := state.BeginDungeonCommandCycle()
	result := state.tryDungeonMove(movement)
	if result.TriggersHiddenPit {
		state.ActionDungeonHiddenPitFall(token, result)
	} else {
		state.CompleteDungeonCommandCycle(token, result.Completion)
	}

	return result
}

func drainDungeonHiddenPitWorkflow(t *testing.T, state *GameState) {
	t.Helper()
	startElapsedMs := state.ElapsedMs
	for range 100 {
		state.ElapsedMs += dungeonPitDamageCadenceMs
		state.EventQueueManager.Update()
		if !state.EventQueueManager.HasWorkflow(dungeonHiddenPitWorkflowName) {
			state.ElapsedMs = startElapsedMs
			return
		}
	}
	require.False(t, state.EventQueueManager.HasWorkflow(dungeonHiddenPitWorkflowName))
}

func TestDungeonHiddenPitExactTriggerMatrix(t *testing.T) {
	tests := []struct {
		name    string
		cell    references.DungeonCell
		trigger bool
	}{
		{name: "61", cell: 0x61, trigger: true},
		{name: "69", cell: 0x69, trigger: true},
		{name: "60", cell: 0x60, trigger: false},
		{name: "68", cell: 0x68, trigger: false},
		{name: "62", cell: 0x62, trigger: false},
		{name: "6a", cell: 0x6a, trigger: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var cells references.DungeonCells
			target := references.Position{X: 2, Y: 1}
			require.True(t, cells.SetCell(0, target, test.cell))
			require.True(t, cells.SetCell(1, target, 0xfa))
			state, callbacks := newDungeonHiddenPitTestGame(t, cells, 61)

			result := startDungeonHiddenPitTestMove(t, state, DungeonAdvance)

			assert.Equal(t, DungeonMoveMoved, result.Outcome)
			assert.Equal(t, test.trigger, result.TriggersHiddenPit)
			assert.Equal(t, test.trigger, callbacks.InputBlocked)
			assert.Equal(t, !test.trigger, state.IsAtDungeonCommandBoundary())
			if test.trigger {
				drainDungeonHiddenPitWorkflow(t, state)
				assert.Equal(t, references.FloorNumber(1), state.MapState.PlayerLocation.Floor)
			} else {
				assert.Equal(t, references.FloorNumber(0), state.MapState.PlayerLocation.Floor)
			}
			assert.Equal(t, !test.trigger, state.IsAtDungeonCommandBoundary())
		})
	}
}

func TestDungeonFullEndPlayerDispatchesHiddenPitUnderParty(t *testing.T) {
	var cells references.DungeonCells
	position := references.Position{X: 1, Y: 1}
	require.True(t, cells.SetCell(0, position, references.DungeonCell(0x69)))
	state, _ := newDungeonHiddenPitTestGame(t, cells, 612)

	token, _ := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

	assert.True(t, state.EventQueueManager.HasWorkflow(dungeonHiddenPitWorkflowName))
	assert.False(t, state.IsAtDungeonCommandBoundary())
	drainDungeonHiddenPitWorkflow(t, state)
	assert.Equal(t, references.FloorNumber(1), state.MapState.PlayerLocation.Floor)
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func TestDungeonHiddenPitBlockedTargetDoesNotTrigger(t *testing.T) {
	var cells references.DungeonCells
	target := references.Position{X: 2, Y: 1}
	require.True(t, cells.SetCell(0, target, 0xb0))
	state, callbacks := newDungeonHiddenPitTestGame(t, cells, 62)

	result := startDungeonHiddenPitTestMove(t, state, DungeonAdvance)

	assert.Equal(t, DungeonMoveTerrainBlocked, result.Outcome)
	assert.False(t, result.TriggersHiddenPit)
	assert.False(t, callbacks.InputBlocked)
	assert.Equal(t, references.Position{X: 1, Y: 1}, state.MapState.PlayerLocation.Position)
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func TestDungeonHiddenPitBackUpTriggersAtSuccessfulDestination(t *testing.T) {
	var cells references.DungeonCells
	target := references.Position{X: 0, Y: 1}
	require.True(t, cells.SetCell(0, target, 0x61))
	require.True(t, cells.SetCell(1, target, 0xfa))
	state, callbacks := newDungeonHiddenPitTestGame(t, cells, 63)

	result := startDungeonHiddenPitTestMove(t, state, DungeonBackUp)

	assert.Equal(t, target, result.Target)
	assert.True(t, result.TriggersHiddenPit)
	assert.True(t, callbacks.InputBlocked)
	drainDungeonHiddenPitWorkflow(t, state)
	assert.Equal(t, target, state.MapState.PlayerLocation.Position)
	assert.Equal(t, references.FloorNumber(1), state.MapState.PlayerLocation.Floor)
	assert.False(t, state.IsAtDungeonCommandBoundary())
}

func TestDungeonHiddenPitChainsMutationsAndDeterministicPartyDamage(t *testing.T) {
	const seed = 6401
	var cells references.DungeonCells
	target := references.Position{X: 2, Y: 1}
	require.True(t, cells.SetCell(0, target, 0x69))
	require.True(t, cells.SetCell(1, target, 0x61))
	require.True(t, cells.SetCell(2, target, 0x61))
	require.True(t, cells.SetCell(3, target, 0xfa))
	state, callbacks := newDungeonHiddenPitTestGame(t, cells, seed)
	state.PartyState.GetCharacter(1).PartyMap = party_state.PartyMapActive
	dead := state.PartyState.GetCharacter(2)
	dead.PartyMap = party_state.PartyMapActive
	dead.SetStatus(party_state.Dead)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}

	result := startDungeonHiddenPitTestMove(t, state, DungeonAdvance)
	require.True(t, result.TriggersHiddenPit)
	assert.True(t, callbacks.InputBlocked)
	assert.False(t, state.IsAtDungeonCommandBoundary())
	assert.Equal(t, uint64(0), state.GetRNG().Calls())

	drainDungeonHiddenPitWorkflow(t, state)

	wantHP := []uint16{100, 100}
	wantRNG := beradrng.New(seed)
	for range 3 {
		for index := range wantHP {
			wantHP[index] -= uint16(wantRNG.Intn(8) + 1)
		}
	}
	assert.Equal(t, wantHP[0], state.PartyState.GetCharacter(0).GetCurrentHp())
	assert.Equal(t, wantHP[1], state.PartyState.GetCharacter(1).GetCurrentHp())
	assert.Equal(t, uint16(100), dead.GetCurrentHp())
	assert.Equal(t, uint64(6), state.GetRNG().Calls(), "fixed-room landing defers generation until combat returns")
	assert.Equal(t, references.FloorNumber(3), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, target, state.MapState.PlayerLocation.Position)
	snapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, references.DungeonFacingEast, snapshot.Facing)
	assert.Equal(t, references.DungeonArrivedFromAbove, snapshot.LastDirection)
	assert.Equal(t, uint64(7), snapshot.Revision)
	assert.Nil(t, snapshot.Wanderer)
	assert.True(t, state.activeDungeon.generateWandererAfterRoom)
	for floor := references.FloorNumber(0); floor <= 2; floor++ {
		cell, ok := snapshot.Cells.Cell(floor, target)
		require.True(t, ok)
		assert.Equal(t, references.DungeonCell(0x68), cell)
	}
	landing, ok := snapshot.Cells.Cell(3, target)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0xfa), landing)
	assert.Len(t, callbacks.SoundEffectsPlayed, 6)
	assert.Equal(t, 5, state.ActiveDurationEffect.TurnsRemaining,
		"end-player effects wait until the pit-landed room returns")
	assert.Equal(t, 1, callbacks.BlockInputCalls)
	assert.Equal(t, 1, callbacks.UnblockInputCalls)
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, state.IsAtDungeonCommandBoundary())
	assert.True(t, state.HasPendingDungeonRoomEncounter())
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestDungeonHiddenPitMarksEligibleDestinationBit08(t *testing.T) {
	var cells references.DungeonCells
	target := references.Position{X: 2, Y: 1}
	require.True(t, cells.SetCell(0, target, 0x61))
	require.True(t, cells.SetCell(1, target, 0x10))
	state, _ := newDungeonHiddenPitTestGame(t, cells, 64)

	startDungeonHiddenPitTestMove(t, state, DungeonAdvance)
	drainDungeonHiddenPitWorkflow(t, state)

	snapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	destination, ok := snapshot.Cells.Cell(1, target)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0x18), destination)
}

func TestDungeonHiddenPitCompletionGeneratesExactlyOnce(t *testing.T) {
	const seed = 2265
	var cells references.DungeonCells
	target := references.Position{X: 2, Y: 1}
	require.True(t, cells.SetCell(0, target, 0x61))
	state, _ := newDungeonHiddenPitTestGame(t, cells, seed)

	result := startDungeonHiddenPitTestMove(t, state, DungeonAdvance)
	require.True(t, result.TriggersHiddenPit)
	drainDungeonHiddenPitWorkflow(t, state)

	actual, active := state.DungeonStateSnapshot()
	require.True(t, active)
	expected := newDungeonStateTestGame(t, actual.Cells)
	expected.MapState.PlayerLocation.Floor = state.MapState.PlayerLocation.Floor
	expected.MapState.PlayerLocation.Position = target
	expected.SetRandomSeed(seed)
	expected.RollDice(8) // One living party member takes damage before regeneration.
	expected.generateDungeonWanderer()
	expectedSnapshot, active := expected.DungeonStateSnapshot()
	require.True(t, active)

	assert.Equal(t, expectedSnapshot.Wanderer, actual.Wanderer)
	assert.Equal(t, expected.GetRNG().Calls(), state.GetRNG().Calls())
}

func TestDungeonHiddenPitBottomFloorSafelyRevealsDamagesAndClamps(t *testing.T) {
	tests := []struct {
		name string
		pit  references.DungeonCell
		want references.DungeonCell
	}{
		{name: "61 reveals to 60", pit: 0x61, want: 0x60},
		{name: "69 reveals to 68", pit: 0x69, want: 0x68},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const seed = 65
			var cells references.DungeonCells
			target := references.Position{X: 2, Y: 1}
			require.True(t, cells.SetCell(references.DungeonFloorCount-1, target, test.pit))
			state, callbacks := newDungeonHiddenPitTestGame(t, cells, seed)
			state.MapState.PlayerLocation.Floor = references.DungeonFloorCount - 1

			startDungeonHiddenPitTestMove(t, state, DungeonAdvance)
			drainDungeonHiddenPitWorkflow(t, state)

			wantRNG := beradrng.New(seed)
			wantHP := uint16(100 - (wantRNG.Intn(8) + 1))
			assert.Equal(t, references.FloorNumber(references.DungeonFloorCount-1), state.MapState.PlayerLocation.Floor)
			assert.Equal(t, wantHP, state.PartyState.GetCharacter(0).GetCurrentHp())
			assert.Equal(t, uint64(4), state.GetRNG().Calls(), "damage precedes one successful generation")
			snapshot, active := state.DungeonStateSnapshot()
			require.True(t, active)
			cell, ok := snapshot.Cells.Cell(references.DungeonFloorCount-1, target)
			require.True(t, ok)
			assert.Equal(t, test.want, cell)
			assert.False(t, callbacks.InputBlocked)
			assert.True(t, state.IsAtDungeonCommandBoundary())
		})
	}
}

func TestRealDeceitDisplayedLevelTwoHiddenPitChainAndSaveRoundTrip(t *testing.T) {
	const seed = 6601
	position := references.Position{X: 1, Y: 3}
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithFloor(1).
		WithPlayerAt(0, 3).
		WithRandomSeed(seed).
		WithSystemCallbacks().
		Build()
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
	callbacks.Reset()

	before, active := state.DungeonStateSnapshot()
	require.True(t, active)
	wantPristine := [...]references.DungeonCell{0x60, 0x69, 0x61, 0x61, 0x61, 0x61, 0x61, 0xfa}
	for floor, want := range wantPristine {
		cell, ok := before.Cells.Cell(references.FloorNumber(floor), position)
		require.True(t, ok)
		assert.Equal(t, want, cell, "displayed level %d", floor+1)
	}

	result := startDungeonHiddenPitTestMove(t, state, DungeonAdvance)
	require.True(t, result.TriggersHiddenPit)
	assert.Equal(t, position, state.MapState.PlayerLocation.Position)
	assert.True(t, callbacks.InputBlocked)
	assert.False(t, state.IsAtDungeonCommandBoundary())
	_, err := state.CaptureSaveSnapshot()
	assert.ErrorIs(t, err, ErrSaveSnapshotNotRepresentable)

	drainDungeonHiddenPitWorkflow(t, state)

	after, active := state.DungeonStateSnapshot()
	require.True(t, active)
	for floor := references.FloorNumber(1); floor <= 6; floor++ {
		cell, ok := after.Cells.Cell(floor, position)
		require.True(t, ok)
		assert.Equal(t, references.DungeonCell(0x68), cell, "displayed level %d", floor+1)
	}
	landing, ok := after.Cells.Cell(7, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0xfa), landing)
	assert.Equal(t, references.FloorNumber(7), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, position, state.MapState.PlayerLocation.Position)
	assert.Equal(t, references.DungeonFacingEast, after.Facing)
	assert.Equal(t, references.DungeonArrivedFromAbove, after.LastDirection)
	assert.Equal(t, uint64(6*activeParty), state.GetRNG().Calls(),
		"fixed-room landing defers type selection and placement until combat returns")
	assert.Nil(t, after.Wanderer)
	assert.True(t, state.activeDungeon.generateWandererAfterRoom)
	assert.Len(t, callbacks.SoundEffectsPlayed, 6*activeParty)
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, state.IsAtDungeonCommandBoundary())
	_, err = state.CaptureSaveSnapshot()
	assert.ErrorIs(t, err, ErrSaveSnapshotNotRepresentable)

	require.NoError(t, state.ResolvePendingDungeonRoomEncounter())
	require.Equal(t, CombatExitDungeonRoom, state.CombatScene.ExitContext)
	require.Equal(t, 4, state.CombatScene.MonsterManager.Count())
	for monsterIndex := range 4 {
		assert.Equal(t, references.MonsterSeaSerpent,
			state.CombatScene.MonsterManager.GetMonster(monsterIndex).MonsterType())
	}
	assert.Equal(t, 2, state.CombatScene.ObjectCount())
	state.CombatScene.ExitRoute = CombatExitSouth
	state.EndCombat(CombatVictory)
	assert.Equal(t, references.Position{X: 1, Y: 4}, state.MapState.PlayerLocation.Position)
	assert.False(t, state.HasPendingDungeonRoomEncounter())
	afterExit, active := state.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, references.DungeonFacingSouth, afterExit.Facing)
	assert.Equal(t, references.DungeonLastSouth, afterExit.LastDirection)
	room, ok := afterExit.Cells.Cell(7, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0xaa), room, "victory clears the room before applying the south exit")
	assert.True(t, state.IsAtDungeonCommandBoundary())

	saved, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	require.Len(t, saved.Redux.DungeonDiscoveries, 6)
	for index, discovery := range saved.Redux.DungeonDiscoveries {
		assert.Equal(t, int(references.Deceit), discovery.Location)
		assert.Equal(t, index+1, discovery.Floor)
		assert.Equal(t, int(position.X), discovery.X)
		assert.Equal(t, int(position.Y), discovery.Y)
		assert.Equal(t, save_snapshot.DungeonDiscoveryHiddenPit, discovery.Kind)
	}
	hydrated, err := NewGameStateFromSaveSnapshot(
		saved,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)
	recaptured, err := hydrated.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Equal(t, saved, recaptured)
	fresh, err := hydrated.PrepareDungeonState(
		references.Deceit,
		references.DungeonFacingEast,
		references.DungeonArrivedFromAbove,
	)
	require.NoError(t, err)
	for floor := references.FloorNumber(1); floor <= 6; floor++ {
		cell, ok := fresh.Cell(floor, position)
		require.True(t, ok)
		assert.Equal(t, references.DungeonCell(0x68), cell, "fresh displayed level %d", floor+1)
	}
	freshLanding, ok := fresh.Cell(7, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0xaa), freshLanding)
}
