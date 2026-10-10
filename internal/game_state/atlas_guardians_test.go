package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

func TestAtlasGuardiansInstalledArtwork(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	world := state.GameReferences.OverworldLargeMapReference
	require.EqualValues(t, indexes.CodexAngelLeft, world.GetSpriteIndex(232, 235))
	require.EqualValues(t, indexes.CodexAngelRight, world.GetSpriteIndex(234, 235))
	require.NotEqualValues(t, indexes.CodexAngelLeft, world.GetSpriteIndex(233, 235))
	require.NotEqualValues(t, indexes.CodexAngelRight, world.GetSpriteIndex(233, 235))
}

func TestAtlasGuardiansSuccessfulLookAndFiniteTargets(t *testing.T) {
	for _, target := range []references.Position{{X: 232, Y: 235}, {X: 234, Y: 235}} {
		state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
		var changes []journal.AtlasChange
		state.SystemCallbacks.UI.AtlasUpdated = func(change journal.AtlasChange) { changes = append(changes, change) }
		state.MapState.PlayerLocation.Position = target.Down()
		state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
		require.True(t, state.ActionLookLargeMap(references.Up))
		require.Len(t, state.atlasLandmarks, 1)
		observed := state.atlasLandmarks[0]
		require.Equal(t, journal.AtlasGuardiansSiteKey, observed.SiteKey)
		require.Equal(t, "guardians", observed.Kind)
		require.Empty(t, observed.Name)
		require.Zero(t, observed.NamedAt)
		require.Zero(t, observed.Survey)
		require.Equal(t, "Guardians of the Codex", changes[0].Name)
		require.True(t, changes[0].Discovered)
		require.False(t, changes[0].Named)
		state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
		state.DateTime.Hour = 12
		require.True(t, state.ActionLookLargeMap(references.Up))
		require.Len(t, changes, 1, "strict daylight earns no new knowledge")
		state.DateTime.Hour = 19
		require.True(t, state.ActionLookLargeMap(references.Up))
		require.Len(t, changes, 2)
		require.True(t, changes[1].Surveyed)
		require.False(t, changes[1].Named)
		entry := atlasEntryByKey(t, state.CurrentAtlasDossier(), "chart-25")
		require.Len(t, state.CurrentAtlasDossier().Entries, 34, "one stable supplied row")
		point, plotted := entry.PlotPosition()
		require.True(t, plotted)
		require.Equal(t, journal.SiteLocation{Map: references.BritanniaUnderworld, X: 233, Y: 235}, point)
		require.Equal(t, string(journal.AtlasSourceLook), entry.SurveySource)
		require.Zero(t, entry.NamedAt)
		require.NoError(t, journal.ValidateAtlasLandmarks(state.atlasLandmarks))
		require.NoError(t, validateAtlasLandmarkCompatibility(state.atlasLandmarks, state.GameReferences))
		survey := entry.Survey
		require.True(t, state.ActionLookLargeMap(references.Up))
		require.Len(t, changes, 2)
		require.Equal(t, survey, state.atlasLandmarks[0].Survey)
	}
}

func TestAtlasGuardiansObscuredWrongArtworkAndWorldStayQuiet(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.DateTime.Hour = 19
	target := references.Position{X: 232, Y: 235}
	state.MapState.PlayerLocation.Position = target.Down()
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapUnitLayer, &target, indexes.HorseRight)
	require.True(t, state.ActionLookLargeMap(references.Up))
	require.Empty(t, state.atlasLandmarks)
	state.SystemCallbacks.UI.AtlasUpdated = func(journal.AtlasChange) { t.Fatal("invalid target published knowledge") }
	state.observeAtlasLook(references.Position{X: 233, Y: 235}, indexes.Hut)
	state.observeAtlasLook(references.Position{X: 10, Y: 10}, indexes.CodexAngelLeft)
	state.observeAtlasLook(target, indexes.CodexAngelRight)
	state.MapState.PlayerLocation.Floor = -1
	state.observeAtlasLook(target, indexes.CodexAngelLeft)
	state.MapState.PlayerLocation.Location, state.MapState.PlayerLocation.Floor = references.Greyhaven, 0
	state.observeAtlasLook(target, indexes.CodexAngelLeft)
	require.Empty(t, state.atlasLandmarks)
}
