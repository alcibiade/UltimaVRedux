package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

func TestAtlasGreyhavenLiveReceiptPreservesEvidenceAndOnlyMeasuresLocalSite(t *testing.T) {
	for _, flexible := range []bool{false, true} {
		for _, hour := range []uint8{12, 19} {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
			state.GameConfig.GameplayConfiguration.FlexibleSextantUse = flexible
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
			state.DateTime.Hour = hour
			for _, location := range []references.Location{references.Fogsbane, references.Greyhaven} {
				position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(location)
				state.observeAtlasLandmark(position, "lighthouse", location.String(), journal.AtlasSourceEnter, state.sampleDungeonObservation())
			}
			before := state.atlasLandmarks[1]
			floors := state.GameReferences.LocationReferences.GetLocationReference(references.Greyhaven).ListOfFloors
			for _, floor := range floors {
				state.atlasLandmarks[1] = before
				state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
				state.MapState.PlayerLocation = references.PlayerLocation{Location: references.Greyhaven, Floor: floor, Position: references.Position{X: 5, Y: 8}}
				state.DateTime.Minute = 17
				var changes []journal.AtlasChange
				state.SystemCallbacks.UI.AtlasUpdated = func(change journal.AtlasChange) {
					require.True(t, state.atlasLandmarks[1].Survey.Measured, "callback follows commit")
					changes = append(changes, change)
				}
				clock, rng := state.DateTime, state.GetRNG().Calls()
				state.GrantSextant()
				require.Equal(t, clock, state.DateTime)
				require.Equal(t, rng, state.GetRNG().Calls())
				got := state.atlasLandmarks[1]
				require.Equal(t, before.Name, got.Name)
				require.Equal(t, before.NamedAt, got.NamedAt)
				require.Equal(t, before.NameSource, got.NameSource)
				require.True(t, got.Survey.Measured)
				require.Equal(t, flexible, got.Survey.Flexible)
				require.Equal(t, got.ObservedAt, got.Survey.RecordedAt)
				require.Equal(t, journal.AtlasSourceSextantReceipt, got.Source)
				require.Equal(t, journal.AtlasSourceSextantReceipt, got.SurveySource)
				require.Equal(t, journal.SiteLocation{Map: references.BritanniaUnderworld, X: 104, Y: 216}, got.Survey.Location)
				require.False(t, state.atlasLandmarks[0].Survey.Measured)
				require.Len(t, changes, 1)
				require.Equal(t, journal.AtlasChange{EntryKey: "chart-16", Name: "Greyhaven", Kind: "lighthouse", Surveyed: true, Measured: true}, changes[0])
				require.NoError(t, journal.ValidateAtlasLandmarks(state.atlasLandmarks))
				if !flexible {
					_, available := state.CurrentSextantFix()
					require.False(t, available, "receipt is not strict indoor Use permission")
				}
				state.DateTime.Minute++
				state.GrantSextant()
				require.Equal(t, got, state.atlasLandmarks[1], "duplicate grant preserves first receipt")
				require.Len(t, changes, 1)
				state.CurrentAtlasDossier()
				state.CurrentQuestJournalDossier()
				require.Len(t, changes, 1, "readers never notify")
			}
		}
	}
}

func TestAtlasGreyhavenReceiptRequiresActualGrantAndEnteredContext(t *testing.T) {
	for _, scenario := range []string{"no-entry", "look-only", "elsewhere", "invalid-floor", "already-owned", "already-measured", "no-source", "wrong-name-source", "wrong-kind", "wrong-anchor"} {
		t.Run(scenario, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
			position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(references.Greyhaven)
			if scenario == "look-only" {
				state.observeAtlasLook(position, indexes.Lighthouse)
			} else if scenario != "no-entry" {
				if scenario == "already-measured" {
					state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
					state.DateTime.Hour = 19
				}
				state.observeAtlasLandmark(position, "lighthouse", "GREYHAVEN", journal.AtlasSourceEnter, state.sampleDungeonObservation())
			}
			state.MapState.PlayerLocation.Location = references.Greyhaven
			if scenario == "elsewhere" {
				state.MapState.PlayerLocation.Location = references.Fogsbane
			}
			if scenario == "invalid-floor" {
				state.MapState.PlayerLocation.Floor = 99
			}
			if scenario == "already-owned" {
				state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
			}
			if scenario == "already-measured" {
				state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
			}
			switch scenario {
			case "no-source":
				state.atlasLandmarks[0].Source = ""
			case "wrong-name-source":
				state.atlasLandmarks[0].NameSource = journal.AtlasSourceLook
			case "wrong-kind":
				state.atlasLandmarks[0].Kind = "hut"
			case "wrong-anchor":
				state.atlasLandmarks[0].Site.X++
			}
			before := append([]journal.AtlasLandmarkObservation(nil), state.atlasLandmarks...)
			state.SystemCallbacks.UI.AtlasUpdated = func(journal.AtlasChange) { t.Fatal("unearned receipt") }
			state.GrantSextant()
			require.Equal(t, before, state.atlasLandmarks)
		})
	}
}

func TestAtlasNoticeDeltasAndInteriorUse(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	var changes []journal.AtlasChange
	state.SystemCallbacks.UI.AtlasUpdated = func(change journal.AtlasChange) { changes = append(changes, change) }
	position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(references.Fogsbane)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	state.observeAtlasLook(position, indexes.Lighthouse)
	require.Equal(t, journal.AtlasChange{EntryKey: "chart-14", Name: "Unnamed lighthouse", Kind: "lighthouse", Discovered: true}, changes[0])
	state.observeAtlasLook(position, indexes.Lighthouse)
	require.Len(t, changes, 1)
	state.observeAtlasLandmark(position, "lighthouse", "FOGSBANE", journal.AtlasSourceEnter, state.sampleDungeonObservation())
	require.Equal(t, journal.AtlasChange{EntryKey: "chart-14", Name: "Fogsbane", Kind: "lighthouse", Named: true}, changes[1])
	state.MapState.PlayerLocation.Location = references.Fogsbane
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.GameConfig.GameplayConfiguration.FlexibleSextantUse = true
	require.True(t, state.useSextant())
	require.Equal(t, journal.AtlasChange{EntryKey: "chart-14", Name: "Fogsbane", Kind: "lighthouse", Surveyed: true, Measured: true}, changes[2])
	require.True(t, state.useSextant())
	require.Len(t, changes, 3)
}

func TestNewSystemCallbacksRequiresAtlasUpdated(t *testing.T) {
	valid := NewMockSystemCallbacks(t).ToSystemCallbacks()
	valid.UI.AtlasUpdated = nil
	_, err := NewSystemCallbacks(valid.Message, valid.Visual, valid.Audio, valid.Screen, valid.Flow, valid.Talk, valid.UI, valid.Combat)
	require.ErrorIs(t, err, errAtlasUpdatedCallbackRequired)
}

func TestAtlasNoticeCapturedExteriorAndSurfaceDungeonOnly(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	var changes []journal.AtlasChange
	state.SystemCallbacks.UI.AtlasUpdated = func(change journal.AtlasChange) { changes = append(changes, change) }
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.DateTime.Hour, state.DateTime.Minute = 5, 59
	captured := state.sampleDungeonObservation()
	position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(references.Moonglow)
	state.DateTime.Hour = 6
	state.MapState.PlayerLocation.Location = references.Moonglow
	state.observeAtlasLandmark(position, "towne", "MOONGLOW", journal.AtlasSourceEnter, captured)
	require.Equal(t, journal.AtlasChange{EntryKey: "chart-01", Name: "Moonglow", Kind: "towne", Discovered: true, Named: true, Surveyed: true, Measured: true}, changes[0])
	require.Equal(t, captured.at, state.atlasLandmarks[0].Survey.RecordedAt)
	state.observeDungeonEntrance(references.Deceit, state.GetDungeonEntranceCoordinates()[references.Deceit], captured, false)
	require.Equal(t, journal.AtlasChange{EntryKey: "dungeon-Deceit", Name: "Deceit", Kind: "dungeon", Discovered: true, Surveyed: true, Measured: true}, changes[1])
	state.observeDungeonEntrance(references.Deceit, state.GetDungeonEntranceCoordinates()[references.Deceit], captured, false)
	require.Len(t, changes, 2)
	underworld := captured
	underworld.sextant.Floor = -1
	state.observeDungeonEntrance(references.Deceit, state.GetDungeonEntranceCoordinates()[references.Deceit], underworld, false)
	state.observeDungeonEntrance(references.Doom, state.GetDungeonEntranceCoordinates()[references.Doom], underworld, false)
	require.Len(t, changes, 2, "no Britannia notice for Underworld or Doom")
	day := captured
	day.sextant.Hour, day.at.Time.Hour = 12, 12
	state.observeDungeonEntrance(references.Despise, state.GetDungeonEntranceCoordinates()[references.Despise], day, false)
	require.True(t, changes[2].Discovered)
	require.False(t, changes[2].Measured)
	state.observeDungeonEntrance(references.Despise, state.GetDungeonEntranceCoordinates()[references.Despise], captured, false)
	require.True(t, changes[3].Surveyed)
	require.False(t, changes[3].Discovered)
}

func TestAtlasShrineNoticeNameAndSurveyAreOneCommittedChange(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.DateTime.Hour = 19
	state.MapState.PlayerLocation.Position = references.AtlasShrinePositions()[0]
	var changes []journal.AtlasChange
	state.SystemCallbacks.UI.AtlasUpdated = func(change journal.AtlasChange) { changes = append(changes, change) }
	require.Equal(t, EnterLargeMapInteraction, state.ResolveEnterLargeMap())
	require.Len(t, changes, 1)
	require.Equal(t, "the shrine of Honesty", changes[0].Name)
	require.True(t, changes[0].Discovered && changes[0].Named && changes[0].Surveyed && changes[0].Measured)
}

func TestAtlasReceiptAndGuardiansSaveRoundTripReadersStaySilent(t *testing.T) {
	for _, hour := range []uint8{12, 19} {
		t.Run(fmt.Sprintf("strict-receipt-hour-%d", hour), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
			state.GameConfig.GameplayConfiguration.FlexibleSextantUse = false
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
			state.DateTime.Hour = hour
			position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(references.Greyhaven)
			state.MapState.PlayerLocation.Position = position
			require.Equal(t, EnterLargeMapTransition, state.ResolveEnterLargeMap())
			runDungeonWorkflow(t, state, "enter_location")
			state.GrantSextant()
			state.ProcessWindThroughElapsed(state.ElapsedMs)
			immediate := restoreAtlasSnapshot(t, state)
			require.Equal(t, references.Greyhaven, immediate.MapState.PlayerLocation.Location)
			require.Equal(t, state.atlasLandmarks, immediate.atlasLandmarks)
			_, fix := immediate.CurrentSextantFix()
			require.False(t, fix, "strict interior fix stays unavailable after native hydration")
			state.TransitionToLocation(references.BritanniaUnderworld, 0, references.Position{X: 232, Y: 236}, TransitionFullLocation, false)
			state.DateTime.Hour = 19
			require.True(t, state.ActionLookLargeMap(references.Up))
			state.SystemCallbacks.UI.AtlasUpdated = func(journal.AtlasChange) { t.Fatal("capture or reader replayed notice") }
			restored := restoreAtlasSnapshot(t, state)
			restored.SystemCallbacks.UI.AtlasUpdated = func(journal.AtlasChange) { t.Fatal("hydrated reader replayed notice") }
			require.Equal(t, state.atlasLandmarks, restored.atlasLandmarks)
			require.Equal(t, state.CurrentAtlasDossier(), restored.CurrentAtlasDossier())
			require.EqualValues(t, hour, restored.atlasLandmarks[0].Survey.RecordedAt.Time.Hour)
			require.False(t, restored.atlasLandmarks[0].Survey.Flexible)
			require.Equal(t, journal.AtlasSourceSextantReceipt, restored.atlasLandmarks[0].SurveySource)
			require.Equal(t, journal.AtlasGuardiansSiteKey, restored.atlasLandmarks[1].SiteKey)
		})
	}
}

func TestAtlasDebugOwnedPresenceAndHydrationCannotSupplyReceipt(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	position, _ := state.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(references.Greyhaven)
	state.observeAtlasLandmark(position, "lighthouse", "GREYHAVEN", journal.AtlasSourceEnter, state.sampleDungeonObservation())
	before := state.atlasLandmarks[0]
	state.SystemCallbacks.UI.AtlasUpdated = func(journal.AtlasChange) { t.Fatal("debug presence or reader manufactured receipt") }
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.GameConfig.GameplayConfiguration.FlexibleSextantUse = true
	state.CurrentAtlasDossier()
	state.CurrentQuestJournalDossier()
	restored := restoreAtlasSnapshot(t, state)
	restored.SystemCallbacks.UI.AtlasUpdated = func(journal.AtlasChange) { t.Fatal("hydrated ownership manufactured receipt") }
	restored.MapState.PlayerLocation.Location = references.Greyhaven
	restored.GrantSextant()
	require.Equal(t, before, restored.atlasLandmarks[0])
}
