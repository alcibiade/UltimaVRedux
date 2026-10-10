package game_state

import (
	"testing"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveSnapshotDoorVisitAllFloorsRoundTrip(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(10, 14).WithRandomSeed(527).WithSystemCallbacks().Build()
	locked := findDoorVisitSourceAtLocation(t, state, references.SerpentsHold, indexes.LockedDoor)
	require.Equal(t, references.FloorNumber(0), locked.Floor)
	require.Equal(t, AnSanctDoorUnlocked, state.CastAnSanct(&locked.Position))
	relocked := references.Position{X: 11, Y: 14}
	require.Equal(t, MagicLockSuccess, state.MagicLockDoor(&relocked), "explicit lock on an originally unlocked source door")
	basement := state.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, -1)
	basement.SetTileByLayer(map_state.MapOverrideLayer, &locked.Position, indexes.RegularDoorView)
	state.TransitionToLocation(references.SerpentsHold, 1, state.MapState.PlayerLocation.Position, TransitionFloorOnly, false)
	restored := restoreDoorVisitSnapshot(t, state)
	for _, current := range []*GameState{state, restored} {
		current.TransitionToLocation(references.SerpentsHold, 0, current.MapState.PlayerLocation.Position, TransitionFloorOnly, false)
		assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), current.GetCurrentLayeredMap().GetTileTopMapOnlyTile(&locked.Position).Index)
		assert.Equal(t, indexes.SpriteIndex(indexes.MagicLockDoor), current.GetCurrentLayeredMap().GetTileTopMapOnlyTile(&relocked).Index)
		assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoorView), current.MapState.LayeredMaps.
			GetLayeredMap(references.SmallMapType, -1).GetTileTopMapOnlyTile(&locked.Position).Index)
		require.Equal(t, MagicUnlockSuccess, current.MagicUnlockDoor(&relocked))
		current.reloadCurrentSmallMapTerrain()
		assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), current.GetEffectiveTile(&relocked))
	}
	before, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	after, err := restored.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Equal(t, before, after, "identical subsequent transitions preserve maps, party/resources, clock and RNG")
	assert.Equal(t, state.RollDice(100), restored.RollDice(100))
	require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(references.Right))
	assert.NoError(t, state.ValidateSaveSnapshotEligibility(), "completed player-open doors have native continuation state")
	state.MapState.CloseOpenDoor()
	assert.NoError(t, state.ValidateSaveSnapshotEligibility())
}

func TestSaveSnapshotDoorVisitOutdoorContinuationDoesNotReviveCache(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(10, 14).WithSystemCallbacks().Build()
	door := references.Position{X: 11, Y: 14}
	require.Equal(t, MagicLockSuccess, state.MagicLockDoor(&door))
	state.TransitionToLocation(references.BritanniaUnderworld, 0, references.Position{X: 100, Y: 100}, TransitionFullLocation, false)
	require.Nil(t, state.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0))
	restored := restoreDoorVisitSnapshot(t, state)
	for _, current := range []*GameState{state, restored} {
		current.TransitionToLocation(references.SerpentsHold, 0, references.Position{X: 10, Y: 14}, TransitionFullLocation, false)
		assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), current.GetEffectiveTile(&door))
	}
}

func TestSaveSnapshotDoorVisitSecretOverrideContinuation(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.LordBritishsCastle).WithSystemCallbacks().Build()
	target := findDoorVisitSource(t, state, indexes.StoneBrickWallSecret)
	state.TransitionToLocation(target.Location, target.Floor, target.Position.Down(), TransitionFullLocation, false)
	state.SearchManager.handleSecretDoorSearch(&target.Position, state.GameReferences.TileReferences.GetTile(indexes.StoneBrickWallSecret))
	require.Equal(t, AnSanctDoorUnlocked, state.CastAnSanct(&target.Position))
	restored := restoreDoorVisitSnapshot(t, state)
	restored.reloadCurrentSmallMapTerrain()
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), restored.GetCurrentLayeredMap().GetTileTopMapOnlyTile(&target.Position).Index)
	assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWallSecret), restored.GetCurrentSmallLocationReference().GetTileNumber(target.Position, target.Floor))
}

func TestPortcullisSleepSkipsAndNativeContinuation(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(14, 23).WithRandomSeed(525).WithSystemCallbacks().Build()
	gate := references.Position{X: 14, Y: 22}
	for _, hour := range []byte{4, 5, 19, 20} {
		state.DateTime.Hour, state.DateTime.Minute = hour, 59
		// Clock edits also request moongate growth/contraction. Reach the same
		// stable save boundary as the scene, without changing the calendar.
		date, calls, gateTile := state.DateTime, state.GetRNG().Calls(), state.GetEffectiveTile(&gate)
		state.UpdateMoongateTransition()
		advanceMoongateWorkflow(state, 18)
		require.False(t, state.IsMoongateTransitionActive())
		require.Equal(t, state.desiredMoongateStage(), state.MoongateStage())
		assert.Equal(t, date, state.DateTime)
		assert.Equal(t, calls, state.GetRNG().Calls())
		assert.Equal(t, gateTile, state.GetEffectiveTile(&gate))
		state.ProcessWindThroughElapsed(state.ElapsedMs)
		restored := restoreDoorVisitSnapshot(t, state)
		assert.Equal(t, state.GetEffectiveTile(&gate), restored.GetEffectiveTile(&gate))
		restored.advanceInnRestMinutes(9)
		want := indexes.SpriteIndex(indexes.Portcullis)
		if restored.DateTime.Hour >= 5 && restored.DateTime.Hour < 20 {
			want = indexes.BrickFloor
		}
		assert.Equal(t, want, restored.GetEffectiveTile(&gate))
		for range 3 {
			restored.advanceInnRestMinutes(5)
			assert.Equal(t, want, restored.GetEffectiveTile(&gate), "repeated advances in hour 5/20 must not XOR-flip the gate")
		}
		restored.reloadCurrentSmallMapTerrain()
		assert.Equal(t, want, restored.GetEffectiveTile(&gate))
		for _, mutation := range restored.captureMapMutations() {
			assert.False(t, mutation.Position.Location == int(references.SerpentsHold) && mutation.Position.Floor == 0 &&
				mutation.Position.X == int(gate.X) && mutation.Position.Y == int(gate.Y), "derived gate state is never persisted")
		}
	}
}

func restoreDoorVisitSnapshot(t *testing.T, state *GameState) *GameState {
	t.Helper()
	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	data, err := save_snapshot.EncodeCanonical(save_snapshot.Envelope{
		Format: save_snapshot.EnvelopeFormat, EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
		Metadata: save_snapshot.Metadata{Name: "Door visit continuation", CreatedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)},
		Snapshot: snapshot,
	})
	require.NoError(t, err)
	decoded, err := save_snapshot.Decode(data)
	require.NoError(t, err)
	restored, err := NewGameStateFromSaveSnapshot(decoded.Snapshot, state.GameConfig, state.GameReferences,
		xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
	require.NoError(t, err)
	restored.SystemCallbacks = NewMockSystemCallbacks(t).ToSystemCallbacks()
	recaptured, err := restored.CaptureSaveSnapshot()
	require.NoError(t, err)
	require.Equal(t, snapshot, recaptured, "capture/encode/decode/hydrate/recapture parity")
	return restored
}
