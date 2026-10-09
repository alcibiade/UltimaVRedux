package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestFlexibleSextantInteriorLaterSurveyNativeSaveChronology(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
		WithRandomSeed(42).WithSystemCallbacks().Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	state.DateTime.Hour = 8
	position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(references.Moonglow)
	state.MapState.PlayerLocation.Position = position
	require.Equal(t, EnterLargeMapTransition, state.ResolveEnterLargeMap())
	runDungeonWorkflow(t, state, "enter_location")
	require.Len(t, state.atlasLandmarks, 1)
	entered := state.atlasLandmarks[0]
	require.Equal(t, journal.SurveyedLocation{}, entered.Survey)
	state.DateTime.Hour = 12
	state.GameConfig.GameplayConfiguration.FlexibleSextantUse = true
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	require.True(t, state.useSextant())
	measured := state.atlasLandmarks[0]
	require.True(t, measured.Survey.Measured)
	require.True(t, measured.Survey.Flexible)
	require.Equal(t, entered.Name, measured.Name, "measurement cannot rename an earned entry")
	require.Equal(t, entered.NamedAt, measured.NamedAt, "first naming provenance is preserved")
	require.Equal(t, entered.NameSource, measured.NameSource)
	require.Equal(t, entered.Site, measured.Survey.Location, "interior use records only the canonical world anchor")
	require.NotEqual(t, entered.ObservedAt, measured.Survey.RecordedAt, "this regression must measure later than entry")
	require.Equal(t, measured.Survey.RecordedAt, measured.ObservedAt, "later successful local measurement owns the last observation date")
	state.GameConfig.GameplayConfiguration.FlexibleSextantUse = false
	state.ProcessWindThroughElapsed(state.ElapsedMs)
	restored := restoreAtlasSnapshot(t, state)
	require.False(t, restored.GameConfig.GameplayConfiguration.FlexibleSextantUse)
	require.Equal(t, state.atlasLandmarks, restored.atlasLandmarks)
	require.Equal(t, measured.Survey, restored.CurrentAtlasDossier().Entries[0].Survey)
}
