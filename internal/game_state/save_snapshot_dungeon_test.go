package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveDungeonSaveSnapshotRoundTrip(t *testing.T) {
	for _, withWanderer := range []bool{false, true} {
		t.Run(map[bool]string{false: "without wanderer", true: "with wanderer"}[withWanderer], func(t *testing.T) {
			state := dungeonSaveTestState(t, withWanderer, true)
			before, err := state.CaptureSaveSnapshot()
			require.NoError(t, err)
			require.NotNil(t, before.Dungeon)
			require.Len(t, before.Dungeon.Cells, save_snapshot.DungeonCellCount)
			assert.Equal(t, int(state.activeDungeon.cells[7][7][7].Raw()), before.Dungeon.Cells[511])
			sourceCells, ok := state.GameReferences.LocationReferences.GetLocationReference(references.Deceit).CopyDungeonCells()
			require.True(t, ok)
			assert.NotEqual(t, int(sourceCells[6][5][4].Raw()), before.Dungeon.Cells[6*64+5*8+4])
			assert.Empty(t, dungeonProjectionMutations(before.MapMutations))

			hydrated, err := NewGameStateFromSaveSnapshot(
				before,
				state.GameConfig,
				state.GameReferences,
				xTilesVisibleOnGameScreen,
				yTilesVisibleOnGameScreen,
			)
			require.NoError(t, err)
			assert.Nil(t, hydrated.CurrentNPCAIController)
			assert.False(t, hydrated.EventQueueManager.HasActiveWorkflows())
			assert.False(t, hydrated.EventQueueManager.HasActiveBackgroundQueues())
			assert.Equal(t, uint64(0), hydrated.activeDungeon.revision)
			assert.Equal(t, uint64(0), hydrated.dungeonCommandSequence)
			assert.Equal(t, state.MapState.Lighting.TorchMinutesRemaining(), hydrated.MapState.Lighting.TorchMinutesRemaining())
			assert.Equal(t, state.MapState.Lighting.HasMagicLight(), hydrated.MapState.Lighting.HasMagicLight())
			projection := hydrated.MapState.LayeredMaps.GetLayeredMap(references.DungeonMapType, 6)
			assert.NotNil(t, projection)
			position := references.Position{X: 4, Y: 5}
			sourceProjection := state.MapState.LayeredMaps.GetLayeredMap(references.DungeonMapType, 6)
			assert.Equal(t, sourceProjection.SpriteIndexByLayer(map_state.MapLayer, position), projection.SpriteIndexByLayer(map_state.MapLayer, position))

			after, err := hydrated.CaptureSaveSnapshot()
			require.NoError(t, err)
			assert.Equal(t, before, after)
			for index := range 20 {
				assert.Equal(t, state.RollDice(1000), hydrated.RollDice(1000), "RNG roll %d", index)
			}
		})
	}
}

func TestActiveDungeonSaveSnapshotPreservesQuicknessNextAction(t *testing.T) {
	for _, skipNext := range []bool{false, true} {
		t.Run(map[bool]string{false: "charged actor phase", true: "free actor phase"}[skipNext], func(t *testing.T) {
			state := dungeonSaveTestState(t, false, skipNext)
			before, err := state.CaptureSaveSnapshot()
			require.NoError(t, err)
			hydrated, err := NewGameStateFromSaveSnapshot(before, state.GameConfig, state.GameReferences, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
			require.NoError(t, err)
			hydrated.SystemCallbacks = NewMockSystemCallbacks(t).ToSystemCallbacks()
			hydrated.SetupEnvironmentalSystemCallbacks()

			wantToken, wantStart := state.BeginDungeonCommandCycle()
			gotToken, gotStart := hydrated.BeginDungeonCommandCycle()
			assert.Equal(t, wantStart, gotStart)
			assert.Equal(t, state.ActionIgnite(), hydrated.ActionIgnite())
			state.CompleteDungeonCommandCycle(wantToken, DungeonFullEndPlayer)
			hydrated.CompleteDungeonCommandCycle(gotToken, DungeonFullEndPlayer)

			want, err := state.CaptureSaveSnapshot()
			require.NoError(t, err)
			got, err := hydrated.CaptureSaveSnapshot()
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestActiveDungeonSaveSnapshotPreservesExactWandererMovementContinuation(t *testing.T) {
	state := dungeonSaveTestState(t, true, false)
	state.activeDungeon.wanderer = &DungeonWanderer{
		Monster: references.MonsterSpider, Floor: 6,
		Position: references.Position{X: 6, Y: 5}, PreviousPosition: references.Position{X: 5, Y: 5},
		OnCeiling: true,
	}
	for _, position := range []references.Position{
		{X: 6, Y: 4}, {X: 7, Y: 5}, {X: 6, Y: 6}, {X: 5, Y: 5},
	} {
		require.True(t, state.activeDungeon.cells.SetCell(6, position, 0x00))
	}
	saved, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)

	hydrated, err := NewGameStateFromSaveSnapshot(
		saved,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)

	wantResult := state.moveDungeonWanderer()
	gotResult := hydrated.moveDungeonWanderer()

	assert.Equal(t, wantResult, gotResult)
	wantDungeon, active := state.DungeonStateSnapshot()
	require.True(t, active)
	gotDungeon, active := hydrated.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, wantDungeon.Wanderer, gotDungeon.Wanderer)
	assert.Equal(t, state.GetRNG().Calls(), hydrated.GetRNG().Calls())
	for index := range 20 {
		assert.Equal(t, state.RollDice(1000), hydrated.RollDice(1000), "RNG roll %d", index)
	}
}

func TestActiveDungeonHydrationDoesNotGenerateMissingWandererOrConsumeRNG(t *testing.T) {
	state := dungeonSaveTestState(t, false, false)
	saved, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	require.Nil(t, saved.Dungeon.Wanderer)

	hydrated, err := NewGameStateFromSaveSnapshot(
		saved,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)

	assert.Nil(t, hydrated.activeDungeon.wanderer)
	assert.Equal(t, saved.Random.SourceCalls, hydrated.GetRNG().Calls())
	recaptured, err := hydrated.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Equal(t, saved.Random, recaptured.Random)
	assert.Nil(t, recaptured.Dungeon.Wanderer)
}

func TestActiveDungeonSaveEligibilityRequiresStableMatchingAuthority(t *testing.T) {
	state := dungeonSaveTestState(t, false, false)
	require.NoError(t, state.ValidateSaveSnapshotEligibility())

	token, _ := state.BeginDungeonCommandCycle()
	_, err := state.CaptureSaveSnapshot()
	assert.ErrorIs(t, err, ErrSaveSnapshotNotRepresentable)
	assert.Contains(t, err.Error(), "active dungeon command cycle")
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
	require.NoError(t, state.ValidateSaveSnapshotEligibility())

	state.activeDungeon.location = references.Doom
	_, err = state.CaptureSaveSnapshot()
	assert.ErrorIs(t, err, ErrSaveSnapshotNotRepresentable)
	assert.Contains(t, err.Error(), "does not match")
}

func TestActiveDungeonSaveSnapshotRejectsMalformedDTOBeforeHydration(t *testing.T) {
	state := dungeonSaveTestState(t, true, false)
	valid, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	invalid := valid
	dungeon := *valid.Dungeon
	dungeon.Cells = append([]int(nil), valid.Dungeon.Cells[:save_snapshot.DungeonCellCount-1]...)
	invalid.Dungeon = &dungeon

	hydrated, err := NewGameStateFromSaveSnapshot(invalid, nil, nil, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
	assert.Nil(t, hydrated)
	assert.ErrorIs(t, err, save_snapshot.ErrInvalidSnapshot)
	assert.Len(t, valid.Dungeon.Cells, save_snapshot.DungeonCellCount)
}

func TestActiveDungeonSaveSnapshotSkipsRetainedTownMap(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithFloor(0).
		WithPlayerAt(2, 3).
		WithSystemCallbacks().
		Build()
	townMap := state.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
	require.NotNil(t, townMap)
	mutationPosition := references.Position{X: 4, Y: 5}
	townMap.SetTileByLayer(map_state.MapOverrideLayer, &mutationPosition, 1)

	state.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.Deceit,
		Position: references.Position{X: 1, Y: 1},
		Floor:    0,
	}
	dungeon, err := state.PrepareDungeonState(
		references.Deceit,
		references.DungeonFacingEast,
		references.DungeonArrivedFromAbove,
	)
	require.NoError(t, err)
	state.installDungeonState(dungeon)
	state.CurrentNPCAIController = nil

	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	for _, mutation := range snapshot.MapMutations {
		assert.NotEqual(t, int(references.Britain), mutation.Position.Location)
	}
	require.NoError(t, ValidateSaveSnapshotCompatibility(snapshot, state.GameReferences))

	hydrated, err := NewGameStateFromSaveSnapshot(
		snapshot,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)
	recaptured, err := hydrated.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Equal(t, snapshot, recaptured)
}

func TestDungeonTransitionClearsSourceTownDoorTimerBeforeSave(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithFloor(0).
		WithPlayerAt(2, 3).
		WithSystemCallbacks().
		Build()
	target := references.Right.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
	require.NotNil(t, target)
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, target, indexes.RegularDoor)
	require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(references.Right))
	require.True(t, state.MapState.HasTransientDoor())

	dungeon, err := state.PrepareDungeonState(
		references.Deceit,
		references.DungeonFacingEast,
		references.DungeonArrivedFromAbove,
	)
	require.NoError(t, err)
	state.installPreparedDungeonState(dungeon)
	state.TransitionToLocation(
		references.Deceit,
		0,
		references.Position{X: 1, Y: 1},
		TransitionFullLocation,
		false,
	)

	assert.False(t, state.MapState.HasTransientDoor())
	_, err = state.CaptureSaveSnapshot()
	assert.NoError(t, err)
}

func dungeonSaveTestState(t *testing.T, withWanderer, quicknessSkipNext bool) *GameState {
	t.Helper()
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithFloor(6).
		WithPlayerAt(2, 3).
		WithRandomSeed(252252).
		WithSystemCallbacks().
		Build()
	state.CurrentNPCAIController = nil

	var cells references.DungeonCells
	index := 0
	for floor := range references.DungeonFloorCount {
		for y := range int(references.YDungeonMapTiles) {
			for x := range int(references.XDungeonMapTiles) {
				cells[floor][y][x] = references.DungeonCell((index*37 + 11) & 0xff)
				index++
			}
		}
	}
	source, ok := state.GameReferences.LocationReferences.GetLocationReference(references.Deceit).CopyDungeonCells()
	require.True(t, ok)
	if cells[6][5][4] == source[6][5][4] {
		cells[6][5][4] ^= 0xff
	}
	dungeon, err := newDungeonState(
		references.Deceit,
		cells,
		references.DungeonFacingWest,
		references.DungeonArrivedFromBelow,
	)
	require.NoError(t, err)
	dungeon.quicknessSkipNext = quicknessSkipNext
	dungeon.revision = 42
	if withWanderer {
		dungeon.wanderer = &DungeonWanderer{
			Monster: references.MonsterSpider, Floor: 4,
			Position: references.Position{X: 6, Y: 5}, PreviousPosition: references.Position{X: 5, Y: 5},
			OnCeiling: true,
		}
	}
	state.installDungeonState(dungeon)
	state.dungeonCommandSequence = 91
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 37}
	state.PartyState.Inventory.Provisions.Torches.Set(5)
	state.ElapsedMs = 12_345
	state.MagicLightMinutesRemaining = 83
	state.MapState.Lighting.SetMagicLight(true)
	state.MapState.Lighting.RestoreTorchMinutes(73)
	state.windSystem.Restore(environment.WindSnapshot{Direction: environment.East, UpdateCounter: 2, NextTickMs: 12_377})
	for range 17 {
		state.RollDice(30)
	}

	return state
}

func dungeonProjectionMutations(mutations []save_snapshot.MapMutationSnapshot) []save_snapshot.MapMutationSnapshot {
	result := make([]save_snapshot.MapMutationSnapshot, 0)
	for _, mutation := range mutations {
		if references.Location(mutation.Position.Location).GetMapType() == references.DungeonMapType {
			result = append(result, mutation)
		}
	}

	return result
}
