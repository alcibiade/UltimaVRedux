package game_state

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/require"
)

func TestAtlasDaylightEntryLearnsNameWithoutSextantAndSurvivesSave(t *testing.T) {
	for _, test := range []struct {
		location references.Location
		key      string
		generic  string
		charted  bool
	}{
		{references.Moonglow, "chart-01", "Towne 1", true},
		{references.Lycaeum, "chart-09", "Keep 1", true},
		{references.PalaceOfBlackthorn, "chart-13", "Unnamed castle", true},
		{references.BuccaneersDen, "chart-34", "Towne 9", true},
		{references.Fogsbane, "chart-14", "Lighthouse 1", true},
		{references.SinVraalsHut, fmt.Sprintf("observed-%d", references.SinVraalsHut), "Unnamed hut", false},
		{references.Stonegate, fmt.Sprintf("observed-%d", references.Stonegate), "Unnamed keep", false},
		{references.Bordermarch, fmt.Sprintf("observed-%d", references.Bordermarch), "Unnamed keep", false},
		{references.Farthing, fmt.Sprintf("observed-%d", references.Farthing), "Unnamed keep", false},
		{references.Windemere, fmt.Sprintf("observed-%d", references.Windemere), "Unnamed keep", false},
		{references.IolosHut, fmt.Sprintf("observed-%d", references.IolosHut), "Unnamed hut", false},
		{references.SuteksHut, fmt.Sprintf("observed-%d", references.SuteksHut), "Unnamed hut", false},
		{references.GrendelsHut, fmt.Sprintf("observed-%d", references.GrendelsHut), "Unnamed hut", false},
		{references.Cove, fmt.Sprintf("observed-%d", references.Cove), "Unnamed village", false},
		{references.Paws, fmt.Sprintf("observed-%d", references.Paws), "Unnamed village", false},
		{references.NorthBritanny, fmt.Sprintf("observed-%d", references.NorthBritanny), "Unnamed village", false},
		{references.EastBritanny, fmt.Sprintf("observed-%d", references.EastBritanny), "Unnamed village", false},
		{references.WestBritanny, fmt.Sprintf("observed-%d", references.WestBritanny), "Unnamed village", false},
	} {
		t.Run(test.location.String(), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
				WithSystemCallbacks().WithRandomSeed(42).Build()
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
			if !test.charted {
				for _, entry := range state.CurrentAtlasDossier().Entries {
					require.NotEqual(t, test.key, entry.Key, "undiscovered places have no placeholder list entry or pin")
				}
			}
			require.EqualValues(t, 12, state.DateTime.Hour)
			position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(test.location)
			state.MapState.PlayerLocation.Position = position.Down()
			require.True(t, state.ActionLookLargeMap(references.Up))
			require.Len(t, state.atlasLandmarks, 1)
			require.Empty(t, state.atlasLandmarks[0].Name)
			require.Equal(t, test.generic, atlasEntryByKey(t, state.CurrentAtlasDossier(), test.key).Name)
			before := state.CurrentAtlasDossier()
			require.False(t, atlasEntryByKey(t, before, test.key).NameKnown)
			require.True(t, atlasEntryByKey(t, before, test.key).Discovered)
			state.MapState.PlayerLocation.Position = position
			require.Equal(t, EnterLargeMapTransition, state.ResolveEnterLargeMap())
			require.Empty(t, state.atlasLandmarks[0].Name, "admission has not presented the introduction")
			runDungeonWorkflow(t, state, "enter_location")
			require.Equal(t, test.location, state.MapState.PlayerLocation.Location)
			learned := state.atlasLandmarks[0]
			require.NotEmpty(t, learned.Name)
			require.Equal(t, journal.AtlasSourceEnter, learned.NameSource)
			require.True(t, learned.NamedAt.Known)
			require.EqualValues(t, 12, learned.NamedAt.Time.Hour)
			require.Equal(t, journal.SurveyedLocation{}, learned.Survey)
			require.Empty(t, learned.SurveySource)
			entry := atlasEntryByKey(t, state.CurrentAtlasDossier(), test.key)
			require.NotEqual(t, test.generic, entry.Name)
			require.True(t, entry.NameKnown)
			require.True(t, entry.Discovered)
			require.Zero(t, entry.Survey)
			after := state.CurrentAtlasDossier()
			require.Len(t, after.Entries, len(before.Entries))
			for _, unchanged := range before.Entries {
				if unchanged.Key != test.key {
					require.Equal(t, unchanged, atlasEntryByKey(t, after, unchanged.Key), "entry reveals only the visited site")
				}
			}
			require.True(t, strings.EqualFold(learned.Name, entry.Name), "presentation only changes casing of learned text")
			if test.location == references.Fogsbane {
				require.Equal(t, "FOGSBANE", learned.Name)
				require.Equal(t, "Fogsbane", entry.Name)
			}
			state.ProcessWindThroughElapsed(state.ElapsedMs)
			restored := restoreAtlasSnapshot(t, state)
			require.Equal(t, learned, restored.atlasLandmarks[0])
			require.Equal(t, entry, atlasEntryByKey(t, restored.CurrentAtlasDossier(), test.key))
			restored.TransitionToLocation(references.BritanniaUnderworld, 0, position.Down(), TransitionFullLocation, false)
			require.True(t, restored.ActionLookLargeMap(references.Up))
			require.Equal(t, learned.Name, restored.atlasLandmarks[0].Name)
			require.Equal(t, learned.NamedAt, restored.atlasLandmarks[0].NamedAt)
			require.Equal(t, learned.NameSource, restored.atlasLandmarks[0].NameSource)
			reloaded := restoreAtlasSnapshot(t, restored)
			require.Equal(t, restored.atlasLandmarks, reloaded.atlasLandmarks)
			entry = atlasEntryByKey(t, reloaded.CurrentAtlasDossier(), test.key)
			require.Equal(t, test.charted, entry.Charted)
			require.False(t, entry.Survey.Measured)
			require.True(t, entry.NameKnown)
			_, plottable := entry.PlotPosition()
			require.False(t, plottable, "all unmeasured sites stay unplottable after hydration")
			if !test.charted {
				require.Equal(t, journal.SiteLocation{Map: references.BritanniaUnderworld, X: int(position.X), Y: int(position.Y)}, reloaded.atlasLandmarks[0].Site)
				reloaded.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
				reloaded.GameConfig.GameplayConfiguration.FlexibleSextantUse = true
				reloaded.DateTime.Hour = 12
				require.False(t, atlasEntryByKey(t, reloaded.CurrentAtlasDossier(), test.key).Survey.Measured,
					"getting the tool and enabling flexibility do not backfill a reading")
				reloaded.TransitionToLocation(test.location, 0, references.Position{X: 15, Y: 15}, TransitionFullLocation, false)
				require.True(t, reloaded.useSextant(), "an explicit interior reading can measure the already discovered place")
				measured := atlasEntryByKey(t, reloaded.CurrentAtlasDossier(), test.key)
				require.Equal(t, entry.Name, measured.Name)
				require.True(t, measured.Survey.Measured)
				require.True(t, measured.Survey.Flexible)
				plotted, ready := measured.PlotPosition()
				require.True(t, ready)
				require.Equal(t, reloaded.atlasLandmarks[0].Site, plotted, "pin uses the measured exterior, never room coordinates")
				reloaded.GameConfig.GameplayConfiguration.FlexibleSextantUse = false
				reloaded.ProcessWindThroughElapsed(reloaded.ElapsedMs)
				remembered := restoreAtlasSnapshot(t, reloaded)
				require.Equal(t, measured, atlasEntryByKey(t, remembered.CurrentAtlasDossier(), test.key),
					"recorded measurements survive disabling flexibility and save/load")
			}
		})
	}
}

func TestAtlasStrongholdProjectionPreservesPersistedKind(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().WithRandomSeed(42).Build()
	for _, location := range []references.Location{references.Lycaeum, references.EmpathAbbey, references.SerpentsHold, references.Stonegate} {
		position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(location)
		state.MapState.PlayerLocation.Position = position.Down()
		require.True(t, state.ActionLookLargeMap(references.Up))
	}
	require.Len(t, state.atlasLandmarks, 4)
	before := append([]journal.AtlasLandmarkObservation(nil), state.atlasLandmarks...)
	dossier := state.CurrentAtlasDossier()
	for _, key := range []string{"chart-09", "chart-10", "chart-11"} {
		require.Equal(t, "stronghold", atlasEntryByKey(t, dossier, key).Kind)
		require.Equal(t, journal.AtlasKeeps, atlasEntryByKey(t, dossier, key).Category)
	}
	keepKey := fmt.Sprintf("observed-%d", references.Stonegate)
	keep := atlasEntryByKey(t, dossier, keepKey)
	require.Equal(t, "keep", keep.Kind)
	require.Equal(t, "Unnamed keep", keep.Name)
	require.False(t, keep.NameKnown)
	require.Equal(t, journal.AtlasKeeps, keep.Category)
	require.False(t, keep.Charted)
	require.False(t, keep.Survey.Measured)
	for _, observation := range state.atlasLandmarks {
		kind := "castle"
		if observation.SiteKey == uint8(references.Stonegate) {
			kind = "keep"
		}
		require.Equal(t, kind, observation.Kind, "public strongholds retain their original exterior-kind evidence")
	}
	require.Equal(t, before, state.atlasLandmarks, "opening the projection never rewrites history")
	require.NoError(t, journal.ValidateAtlasLandmarks(state.atlasLandmarks))
	restored := restoreAtlasSnapshot(t, state)
	require.Equal(t, before, restored.atlasLandmarks)
	require.Equal(t, dossier, restored.CurrentAtlasDossier())
}

func TestAtlasDiscoveryOnlyEntryWithFlexibleSextantEarnsOnlyVisitedPin(t *testing.T) {
	for _, location := range []references.Location{
		references.Bordermarch, references.Stonegate, references.Farthing, references.Windemere,
		references.IolosHut, references.SuteksHut, references.SinVraalsHut, references.GrendelsHut,
		references.Cove, references.Paws, references.NorthBritanny, references.EastBritanny, references.WestBritanny,
	} {
		t.Run(location.String(), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
				WithSystemCallbacks().WithRandomSeed(42).Build()
			state.GameConfig.GameplayConfiguration.FlexibleSextantUse = true
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
			state.DateTime.Hour = 12
			key := fmt.Sprintf("observed-%d", location)
			before := state.CurrentAtlasDossier()
			require.Len(t, before.Entries, 34, "sextant ownership and flexible policy reveal no hidden places")
			position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(location)
			state.MapState.PlayerLocation.Position = position
			require.Equal(t, EnterLargeMapTransition, state.ResolveEnterLargeMap())
			require.Empty(t, state.atlasLandmarks, "workflow admission earns nothing before entry completes")
			runDungeonWorkflow(t, state, "enter_location")
			require.Equal(t, location, state.MapState.PlayerLocation.Location)
			after := state.CurrentAtlasDossier()
			require.Len(t, after.Entries, 35, "only the entered hidden place is added")
			entry := atlasEntryByKey(t, after, key)
			require.True(t, entry.Discovered)
			require.True(t, entry.NameKnown)
			require.NotEmpty(t, entry.Name)
			require.True(t, entry.Survey.Measured)
			require.True(t, entry.Survey.Flexible)
			pin, plotted := entry.PlotPosition()
			require.True(t, plotted)
			require.Equal(t, journal.SiteLocation{Map: references.BritanniaUnderworld, X: int(position.X), Y: int(position.Y)}, pin)
			for _, unchanged := range before.Entries {
				require.Equal(t, unchanged, atlasEntryByKey(t, after, unchanged.Key), "no remote discovery or survey")
			}
			state.ProcessWindThroughElapsed(state.ElapsedMs)
			restored := restoreAtlasSnapshot(t, state)
			require.Equal(t, entry, atlasEntryByKey(t, restored.CurrentAtlasDossier(), key))
		})
	}
}

func TestAtlasHydrationDoesNotBackfillPreviouslyVisitedLocation(t *testing.T) {
	for _, location := range []references.Location{references.Moonglow, references.Lycaeum, references.Fogsbane, references.SinVraalsHut, references.Stonegate} {
		t.Run(location.String(), func(t *testing.T) {
			// Being inside a site in an existing save is not evidence of a newly
			// presented introduction or an independently authorised measurement.
			state, _ := NewIntegrationTestBuilder(t).WithLocation(location).WithPlayerAt(15, 30).
				WithSystemCallbacks().WithRandomSeed(42).Build()
			require.Empty(t, state.atlasLandmarks)
			restored := restoreAtlasSnapshot(t, state)
			require.Empty(t, restored.atlasLandmarks)
			dossier := restored.CurrentAtlasDossier()
			require.Len(t, dossier.Entries, 34)
			require.Equal(t, "Lighthouse 1", atlasEntryByKey(t, dossier, "chart-14").Name)
			for _, entry := range dossier.Entries {
				require.False(t, entry.Discovered)
				require.False(t, entry.NamedAt.Known)
				require.False(t, entry.Survey.Measured)
				require.NotEqual(t, "hut", entry.Kind)
				require.NotEqual(t, "keep", entry.Kind)
			}
		})
	}
}

func TestAtlasExistingEnterNameSurvivesHydrationWithoutNewVisit(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().WithRandomSeed(42).Build()
	position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(references.Moonglow)
	state.atlasLandmarks = []journal.AtlasLandmarkObservation{{
		SiteKey: uint8(references.Moonglow), Kind: "towne", Source: journal.AtlasSourceLook,
		Site: journal.SiteLocation{Map: references.BritanniaUnderworld, X: int(position.X), Y: int(position.Y)},
		Name: "MOONGLOW", NameSource: journal.AtlasSourceEnter,
	}}
	restored := restoreAtlasSnapshot(t, state)
	entry := atlasEntryByKey(t, restored.CurrentAtlasDossier(), "chart-01")
	require.True(t, entry.NameKnown, "old explicit Enter provenance remains sufficient without a date")
	require.True(t, entry.Discovered, "actual Look provenance remains sufficient without a historical date")
	require.Equal(t, "Moonglow", entry.Name)
	require.Equal(t, journal.AtlasSourceEnter, entry.NameSource)
	require.False(t, entry.NamedAt.Known)
	require.False(t, entry.Survey.Measured)
	require.Equal(t, state.atlasLandmarks, restored.atlasLandmarks)
}

func restoreAtlasSnapshot(t *testing.T, state *GameState) *GameState {
	t.Helper()
	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	require.Empty(t, save_snapshot.Validate(snapshot))
	require.NoError(t, ValidateSaveSnapshotCompatibility(snapshot, state.GameReferences))
	data, err := save_snapshot.EncodeCanonical(save_snapshot.Envelope{
		Format: save_snapshot.EnvelopeFormat, EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
		Metadata: save_snapshot.Metadata{Name: "Atlas evidence", CreatedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		Snapshot: snapshot,
	})
	require.NoError(t, err)
	decoded, err := save_snapshot.Decode(data)
	require.NoError(t, err)
	restored, err := NewGameStateFromSaveSnapshot(decoded.Snapshot, state.GameConfig, state.GameReferences,
		xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
	require.NoError(t, err)
	callbacks := NewMockSystemCallbacks(t)
	restored.SystemCallbacks = callbacks.ToSystemCallbacks()
	restored.SetupEnvironmentalSystemCallbacks()
	callbacks.BindGameState(restored)
	recaptured, err := restored.CaptureSaveSnapshot()
	require.NoError(t, err)
	require.Equal(t, snapshot, recaptured, "full capture/encode/decode/hydrate/recapture parity")
	return restored
}
