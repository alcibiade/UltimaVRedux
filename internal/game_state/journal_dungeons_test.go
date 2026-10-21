package game_state

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestDungeonJournalRecognizesGoethAnswerWithoutSolvingHisRunes(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithRandomSeed(42).WithSystemCallbacks().Build()
	require.Empty(t, state.journalDungeons())
	source := KnowledgeSource{Kind: KnowledgeSourceDialogue, DisplayName: "Goeth"}
	require.True(t, state.DiscoverWordOfPowerKnowledge("destard", source))
	learned := state.journalDungeons()
	require.Len(t, learned, 1)
	require.Equal(t, "Destard", learned[0].Name)
	require.Equal(t, "AIPONI", learned[0].Word)
	require.True(t, learned[0].WordIsClue)
	require.Equal(t, "Goeth", learned[0].Source)
	require.Equal(t, "Not inspected", learned[0].Entrance)
	require.Empty(t, learned[0].Location, "learning never invents entrance coordinates")
	require.True(t, state.journalDungeonIdentityKnown(references.Destard))
	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	reloaded, err := NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
	require.NoError(t, err)
	require.Equal(t, learned, reloaded.journalDungeons(), "existing persisted knowledge supplies the association")
	entrance := state.GetDungeonEntranceCoordinates()[references.Destard]
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.DateTime.Hour = 19
	state.observeDungeonEntrance(references.Destard, entrance, state.sampleDungeonObservation(), false)
	entries := state.journalDungeons()
	require.Len(t, entries, 1)
	require.Equal(t, "Destard", entries[0].Name)
	require.Equal(t, "Britannia K'I\", E'I\"", entries[0].Location)
	require.Equal(t, "AIPONI", entries[0].Word)
	state.SetDungeonState(references.Destard, true)
	state.observeDungeonEntrance(references.Destard, entrance, state.sampleDungeonObservation(), true)
	entries = state.journalDungeons()
	require.Equal(t, "Open", entries[0].Entrance)
	require.Equal(t, WordInopia, entries[0].Word)
	require.False(t, entries[0].WordIsClue)
	state.SetDungeonState(references.Destard, false)
	state.observeDungeonEntrance(references.Destard, entrance, state.sampleDungeonObservation(), true)
	require.Equal(t, "Sealed", state.journalDungeons()[0].Entrance)
	require.Contains(t, state.CurrentQuestJournalDossier().Chronicle[0].Summary, "AIPONI")
}

func TestDungeonJournalKeepsPrivateRecognitionOutOfCompleteReader(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithRandomSeed(42).WithSystemCallbacks().Build()
	// Noncanonical coordinates model supported historical private evidence.
	state.dungeonJournal[DeceitIndex] = journal.DungeonObservation{
		Known: true,
		Sites: [journal.DungeonExteriorSiteCount]journal.DungeonSiteObservation{
			{Seen: true, Location: journal.SiteLocation{Map: references.BritanniaUnderworld, X: 237, Y: 213}},
		},
	}
	before := state.dungeonJournal
	clock, calls := state.DateTime, state.GetRNG().Calls()
	callbacks.Reset()
	dossier := state.CurrentQuestJournalDossier()
	require.Len(t, dossier.Dungeons, 1)
	entry := dossier.Dungeons[0]
	require.Equal(t, "Britannia Not measured", entry.Location)
	require.Equal(t, journal.SurveyedLocation{}, entry.Sites[0].Survey)
	require.Equal(t, journal.OptionalGameTime{}, entry.Sites[0].ObservedAt, "historical date remains absent")
	require.Equal(t, journal.DungeonSiteEntry{}, entry.Sites[1])
	require.NotContains(t, fmt.Sprintf("%+v", dossier), "237")
	require.NotContains(t, fmt.Sprintf("%+v", dossier), "213")
	require.NotContains(t, fmt.Sprintf("%+v", dossier), "N'F\"")
	require.NotContains(t, fmt.Sprintf("%+v", dossier), "O'N\"")
	dossier.Dungeons[0].Sites[0].Survey = journal.SurveyedLocation{Measured: true}
	require.Equal(t, before, state.dungeonJournal, "reader mutation must not alias private state")
	require.Equal(t, clock, state.DateTime)
	require.Equal(t, calls, state.GetRNG().Calls())
	require.Empty(t, callbacks.Messages)
	require.Empty(t, callbacks.TimeAdvanced())
}

func TestDungeonJournalSurveyIsSiteSpecificDurableAndReadOnly(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
		WithRandomSeed(42).WithSystemCallbacks().Build()
	entrance := state.GetDungeonEntranceCoordinates()[references.Deceit]
	state.DateTime.Hour = 19
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	clock, calls := state.DateTime, state.GetRNG().Calls()
	callbacks.Reset()
	state.observeDungeonEntrance(references.Deceit, entrance, state.sampleDungeonObservation(), false)
	require.Equal(t, journal.SurveyedLocation{}, state.journalDungeons()[0].Sites[0].Survey)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.DateTime.Hour = 12
	state.MapState.PlayerLocation.Position = entrance
	state.CurrentQuestJournalDossier()
	require.False(t, state.dungeonJournal[DeceitIndex].Sites[0].Survey.Measured, "clock changes and debug repositioning do not survey")
	state.DateTime = clock
	state.CurrentQuestJournalDossier()
	state.CurrentSextantFix()
	require.False(t, state.dungeonJournal[DeceitIndex].Sites[0].Survey.Measured, "acquiring or reading does not backfill")
	state.observeDungeonEntrance(references.Deceit, entrance, state.sampleDungeonObservation(), false)
	survey := state.journalDungeons()[0].Sites[0].Survey
	require.True(t, survey.Measured)
	require.True(t, survey.RecordedAt.Known)
	require.Equal(t, int(entrance.X), survey.Location.X)
	require.Equal(t, int(entrance.Y), survey.Location.Y)
	require.Equal(t, clock, state.DateTime)
	require.Equal(t, calls, state.GetRNG().Calls())
	require.Empty(t, callbacks.Messages)
	require.Empty(t, callbacks.TimeAdvanced())

	state.DateTime.Hour = 12
	state.MapState.PlayerLocation.Floor = references.FloorNumber(references.UNDERWORLD)
	state.SetDungeonState(references.Deceit, true)
	state.observeDungeonEntrance(references.Deceit, entrance, state.sampleDungeonObservation(), false)
	entry := state.journalDungeons()[0]
	require.Equal(t, "Britannia: Sealed; Underworld: Open", entry.Entrance)
	require.Contains(t, entry.Location, "Underworld Not measured")
	require.Equal(t, survey, entry.Sites[0].Survey)
	require.Equal(t, journal.SurveyedLocation{}, entry.Sites[1].Survey)
	require.Equal(t, references.FloorNumber(references.UNDERWORLD), state.dungeonJournal[DeceitIndex].Sites[1].Location.Floor)

	state.MapState.PlayerLocation.Floor = 0
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	state.observeDungeonEntrance(references.Deceit, entrance, state.sampleDungeonObservation(), false)
	require.Equal(t, survey, state.journalDungeons()[0].Sites[0].Survey, "daylight/tool loss never erases or redates a survey")
	require.Equal(t, 12, state.journalDungeons()[0].Sites[0].ObservedAt.Time.Hour)
	require.Equal(t, 19, state.journalDungeons()[0].Sites[0].Survey.RecordedAt.Time.Hour)
	state.SetDungeonState(references.Deceit, false)
	require.True(t, state.journalDungeons()[0].Sites[0].Open, "seal memory is not a remote live query")
}

func TestDungeonJournalDoesNotTransferSurveyToDifferentSite(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).WithSystemCallbacks().Build()
	state.DateTime.Hour = 12
	state.dungeonJournal[DeceitIndex] = journal.DungeonObservation{Known: true,
		Sites: [journal.DungeonExteriorSiteCount]journal.DungeonSiteObservation{{
			Seen: true, Location: journal.SiteLocation{X: 213, Y: 237},
			Survey: journal.SurveyedLocation{Measured: true, Location: journal.SiteLocation{X: 213, Y: 237}},
		}},
	}
	state.observeDungeonEntrance(references.Deceit, state.GetDungeonEntranceCoordinates()[references.Deceit], state.sampleDungeonObservation(), false)
	require.Equal(t, journal.SurveyedLocation{}, state.journalDungeons()[0].Sites[0].Survey)
}

func TestDungeonJournalStillDoesNotAssociateShameOrDoomWordsAlone(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	for _, dungeon := range []string{"shame", "doom"} {
		require.True(t, state.DiscoverWordOfPowerKnowledge(dungeon, DialogueKnowledgeSource("Speaker")))
	}
	entries := state.journalDungeons()
	require.Len(t, entries, 1, "only the generic gate clue appears")
	require.Equal(t, "Final Gate", entries[0].Name)
}

func TestDungeonJournalFinalGateFollowsCodexKnowledge(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
		WithRandomSeed(42).WithSystemCallbacks().Build()
	require.Empty(t, state.journalDungeons(), "no placeholder before the word is learned")
	require.True(t, state.DiscoverWordOfPowerKnowledge("doom", RevelationKnowledgeSource("Codex of Ultimate Wisdom")))
	reloadGate := func() journal.DungeonEntry {
		t.Helper()
		snapshot, err := state.CaptureSaveSnapshot()
		require.NoError(t, err)
		reloaded, err := NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences,
			xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
		require.NoError(t, err)
		entries := reloaded.journalDungeons()
		require.Len(t, entries, 1)

		return entries[0]
	}

	before, events, clock, calls := state.dungeonJournal, state.QuestJournalEvents(), state.DateTime, state.GetRNG().Calls()
	callbacks.Reset()
	partial := state.CurrentQuestJournalDossier().Dungeons
	require.Equal(t, []journal.DungeonEntry{{Name: "Final Gate", Word: "VERAMOCOR", Source: "Codex of Ultimate Wisdom",
		Description: "The Codex revealed VERAMOCOR to unlock the gate.", Gate: true}}, partial)
	require.Equal(t, partial[0], reloadGate(), "existing word-only saves gain the partial row")
	require.Equal(t, before, state.dungeonJournal, "projection grants no observation")
	require.Equal(t, events, state.QuestJournalEvents())
	require.Equal(t, clock, state.DateTime)
	require.Equal(t, calls, state.GetRNG().Calls())
	require.Empty(t, callbacks.Messages)

	require.True(t, state.RecordCompleteCodexRevelation())
	complete := state.journalDungeons()
	require.Len(t, complete, 1)
	require.Equal(t, codexRevelationText, complete[0].Description)
	require.Equal(t, complete[0], reloadGate(), "complete saves show the full clue")
	require.Equal(t, events, state.QuestJournalEvents(), "the gate adds no Chronicle event")
	require.NotContains(t, strings.ToLower(fmt.Sprintf("%+v", complete)), "doom")
	require.Equal(t, [8]bool{}, state.DungeonStates, "the clue unseals nothing")

	entrance := state.GetDungeonEntranceCoordinates()[references.Doom]
	state.MapState.PlayerLocation.Floor = references.FloorNumber(references.UNDERWORLD)
	state.observeDungeonEntrance(references.Doom, entrance, state.sampleDungeonObservation(), false)
	observed := state.journalDungeons()
	require.Len(t, observed, 2)
	require.Equal(t, "Doom", observed[0].Name)
	require.Empty(t, observed[0].Word, "the hidden key alone never enriches an observed Doom row")
	require.Equal(t, complete[0], observed[1])
}

func TestChronicleAcquisitionMilestonesAreFirsts(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithRandomSeed(42).WithSystemCallbacks().Build()
	for range 2 {
		state.recordQuestItemAcquired(references.Crown)
		state.recordQuestItemAcquired(references.WoodenBox)
		state.recordItemAcquired("minoc-odd-keys")
	}
	entries := state.CurrentQuestJournalDossier().Chronicle
	require.Len(t, entries, 3)
	require.Contains(t, entries[0].Title, "Crown")
	require.NotContains(t, entries[1].Summary, "Orb")
	require.NotContains(t, entries[2].Summary, "skull")
}
