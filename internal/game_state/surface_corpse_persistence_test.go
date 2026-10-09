package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestSurfaceCorpseOrderUsesPersistedSearchStateTail(t *testing.T) {
	state := &GameState{}
	state.SearchManager = NewSearchSystemManager(state)
	state.MapState.PlayerLocation = references.PlayerLocation{Location: 4, Floor: 0}
	locations := references.GetAllSearchLocations()

	for _, searchIndex := range []int{44, 40, 43} {
		location := locations[searchIndex]
		result := &AuthenticSearchResult{
			Index:       searchIndex,
			ObjectClass: location.ObjectClass,
			Position: references.Position{
				X: references.Coordinate(location.X), Y: references.Coordinate(location.Y),
			},
		}
		require.True(t, state.addSurfaceCorpse(result))
		state.SearchManager.authenticLocationManager.CommitFound(result)
	}

	found, skullDay, reagentDays := state.SearchManager.snapshotState()
	restored := &GameState{}
	restored.SearchManager = NewSearchSystemManager(restored)
	restored.MapState.PlayerLocation = state.MapState.PlayerLocation
	restored.SearchManager.restoreState(found, skullDay, reagentDays)
	restored.restoreSurfaceCorpsesForCurrentMap()
	require.Len(t, restored.SurfaceCorpses, 3)
	assert.Equal(t, []int{44, 40, 43}, []int{
		restored.SurfaceCorpses[0].SearchIndex,
		restored.SurfaceCorpses[1].SearchIndex,
		restored.SurfaceCorpses[2].SearchIndex,
	})

	restored.removeSurfaceCorpse(1)
	found, skullDay, reagentDays = restored.SearchManager.snapshotState()
	reloaded := &GameState{}
	reloaded.SearchManager = NewSearchSystemManager(reloaded)
	reloaded.MapState.PlayerLocation = state.MapState.PlayerLocation
	reloaded.SearchManager.restoreState(found, skullDay, reagentDays)
	reloaded.restoreSurfaceCorpsesForCurrentMap()
	require.Len(t, reloaded.SurfaceCorpses, 2)
	assert.Equal(t, 44, reloaded.SurfaceCorpses[0].SearchIndex)
	assert.Equal(t, 43, reloaded.SurfaceCorpses[1].SearchIndex)
}

func TestNativeSnapshotPreservesBuriedMoongateSource(t *testing.T) {
	gameReferences := getBaseGameReferences()
	if gameReferences == nil || cachedIntegrationCfg == nil {
		t.Skip("integration test requires original Ultima V game data")
	}
	state := getBritain2GameState()
	state.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.BritanniaUnderworld,
		Floor:    0,
		Position: references.Position{X: 224, Y: 133},
	}
	state.initializeSnapshotMapsAndControllers()
	buryMoonstoneForTest(state.moonstoneSystem, 0)
	state.DateTime.Hour = 20
	state.initializeMoongateStage()
	require.True(t, state.IsPlayerOnActiveMoongate())

	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	hydrated, err := NewGameStateFromSaveSnapshot(
		snapshot, cachedIntegrationCfg, gameReferences,
		xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)
	assert.True(t, hydrated.IsPlayerOnActiveMoongate())
}

func TestNativeSnapshotPreservesCurrentMapSurfaceCorpseOrder(t *testing.T) {
	gameReferences := getBaseGameReferences()
	if gameReferences == nil || cachedIntegrationCfg == nil {
		t.Skip("integration test requires original Ultima V game data")
	}
	state := getBritain2GameState()
	state.MapState.PlayerLocation = references.PlayerLocation{
		Location: 4,
		Floor:    0,
		Position: references.Position{X: 12, Y: 2},
	}
	state.initializeSnapshotMapsAndControllers()
	locations := references.GetAllSearchLocations()
	for _, searchIndex := range []int{44, 40, 43} {
		location := locations[searchIndex]
		result := &AuthenticSearchResult{
			Index:       searchIndex,
			ObjectClass: location.ObjectClass,
			Position: references.Position{
				X: references.Coordinate(location.X), Y: references.Coordinate(location.Y),
			},
		}
		require.True(t, state.addSurfaceCorpse(result))
		state.SearchManager.authenticLocationManager.CommitFound(result)
	}

	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	hydrated, err := NewGameStateFromSaveSnapshot(
		snapshot, cachedIntegrationCfg, gameReferences,
		xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)
	require.Len(t, hydrated.SurfaceCorpses, 3)
	assert.Equal(t, []int{44, 40, 43}, []int{
		hydrated.SurfaceCorpses[0].SearchIndex,
		hydrated.SurfaceCorpses[1].SearchIndex,
		hydrated.SurfaceCorpses[2].SearchIndex,
	})
}
