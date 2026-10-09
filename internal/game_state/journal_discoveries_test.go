package game_state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/shrine"
)

func journalTestCallbacks() *SystemCallbacks {
	return &SystemCallbacks{UI: UICallbacks{JournalChanged: func(int, int) {}}}
}

func newJournalTestState() *GameState {
	return &GameState{
		DateTime:        datetime.UltimaDate{Year: 139, Month: 2, Day: 3},
		SystemCallbacks: journalTestCallbacks(),
		playerKnowledge: newPlayerKnowledge(), questJournal: newQuestJournal(),
		ShadowlordState: references.NewShadowlordState(),
	}
}

func TestJournalIndependentDiscoveriesAndExplicitBinding(t *testing.T) {
	state := newJournalTestState()
	var changes [][2]int
	state.SystemCallbacks.UI.JournalChanged = func(tab, entry int) { changes = append(changes, [2]int{tab, entry}) }
	require.True(t, state.DiscoverJournalKnowledge("domain.hatred", DialogueKnowledgeSource("Michael")))
	assert.Equal(t, [][2]int{{journal.ChronicleTab, 1}}, changes, "nameless domain does not touch a fixed row")
	note := state.CurrentQuestJournalDossier().Chronicle[0]
	state.DateTime.Day++
	require.True(t, state.DiscoverShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeName, DialogueKnowledgeSource("Sin'Vraal")))
	row := state.CurrentQuestJournalDossier().Shadowlords[1]
	assert.Equal(t, "Astaroth", row.DisplayName)
	assert.Empty(t, row.Association)
	assert.Empty(t, row.Opposes)
	assert.Empty(t, row.Flame)
	require.True(t, state.DiscoverJournalKnowledge("flame.love", DialogueKnowledgeSource("Barbra")))
	assert.Empty(t, state.CurrentQuestJournalDossier().Shadowlords[1].Flame)
	require.True(t, state.DiscoverShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeAssociation, DialogueKnowledgeSource("Sin'Vraal")))
	row = state.CurrentQuestJournalDossier().Shadowlords[1]
	assert.Equal(t, "Hatred", row.Association)
	assert.Equal(t, "Love", row.Opposes)
	assert.Equal(t, "Flame of Love", row.Flame)
	assert.Equal(t, "Empath Abbey", row.Location)
	assert.Equal(t, note, state.CurrentQuestJournalDossier().Chronicle[0], "earlier independent note stays nameless with its original date and source")
	before := append([][2]int(nil), changes...)
	assert.False(t, state.DiscoverJournalKnowledge("domain.hatred", DialogueKnowledgeSource("Other")))
	assert.False(t, state.DiscoverShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeAssociation, DialogueKnowledgeSource("Other")))
	assert.Equal(t, before, changes)
	_, known := state.JournalKnowledge("banishment.method")
	assert.False(t, known)
}

func TestJournalDirectAssociationImpliesDomainWithoutExtraEvent(t *testing.T) {
	state := newJournalTestState()
	source := DialogueKnowledgeSource("Sin'Vraal")
	require.True(t, state.DiscoverShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeAssociation, source))
	entry, known := state.JournalKnowledge("domain.hatred")
	require.True(t, known)
	assert.Equal(t, source, entry.Source)
	assert.False(t, state.DiscoverJournalKnowledge("domain.hatred", DialogueKnowledgeSource("Michael")))
	require.Len(t, state.QuestJournalEvents(), 1)
	row := state.CurrentQuestJournalDossier().Shadowlords[1]
	assert.Equal(t, "Shadowlord II", row.DisplayName)
	assert.Equal(t, "Hatred", row.Association)
	assert.Empty(t, row.Flame)
	assert.Empty(t, row.Location)
}

func TestJournalKnowledgeClosedKeysAndProvenance(t *testing.T) {
	state := newJournalTestState()
	ids := []string{"domain.falsehood", "domain.hatred", "domain.cowardice", "flame.truth", "flame.love", "flame.courage", "banishment.method"}
	for _, id := range ids {
		require.True(t, state.DiscoverJournalKnowledge(id, ObservationKnowledgeSource("Look (Lycaeum)")))
		entry, known := state.JournalKnowledge(id)
		require.True(t, known)
		assert.Equal(t, KnowledgeCategoryJournal, entry.Key.Category)
		assert.False(t, entry.Unread)
		entry.DiscoveredOn.Day = 20
		assert.False(t, state.DiscoverJournalKnowledge(id, RevelationKnowledgeSource("Revelation")))
	}
	for _, id := range []string{"", "journal.flame.truth", "flame.Truth", "domain.hate", "arbitrary"} {
		assert.False(t, state.DiscoverJournalKnowledge(id, DialogueKnowledgeSource("Speaker")))
		_, known := state.JournalKnowledge(id)
		assert.False(t, known)
	}
	for index, event := range state.QuestJournalEvents() {
		assert.Equal(t, uint64(index+1), event.Sequence)
		assert.Equal(t, QuestJournalEventKnowledgeLearned, event.Kind)
		assert.Equal(t, QuestJournalSubject("journal."+ids[index]), event.Subject)
		assert.Equal(t, QuestJournalEventSource("observation"), event.Source)
		assert.Equal(t, "Look (Lycaeum)", event.SourceName)
		assert.Equal(t, byte(3), event.OccurredOn.Day)
	}
	assert.Len(t, state.CurrentQuestJournalDossier().Chronicle, 7)
}

func TestJournalVirtueNotificationsOnlyActualValueChanges(t *testing.T) {
	state := newJournalTestState()
	var changes [][2]int
	state.SystemCallbacks.UI.JournalChanged = func(tab, entry int) { changes = append(changes, [2]int{tab, entry}) }
	state.SetShrineQuestActive(2, false)
	state.SetShrineQuestActive(2, true)
	state.SetShrineQuestActive(2, true)
	state.SetShrineQuestCompleted(2, true)
	state.SetShrineQuestCompleted(2, true)
	state.SetShrineRuined(2, true)
	state.SetShrineRuined(2, true)
	state.SetShrineRuined(-1, true)
	assert.Equal(t, [][2]int{{journal.VirtuesTab, 2}, {journal.VirtuesTab, 2}, {journal.VirtuesTab, 2}}, changes)
	assert.Empty(t, state.QuestJournalEvents(), "flag setters do not invent Codex revelations")
}

func TestJournalFiniteChronicleRetainsAll40Discoveries(t *testing.T) {
	state := newJournalTestState()
	state.MapState = map_state.MapState{PlayerLocation: references.PlayerLocation{Location: references.CodexLocation}}
	state.ShrineScene = shrine.NewShrineScene(-1, state.MapState.PlayerLocation)
	for _, id := range []string{"domain.falsehood", "domain.hatred", "domain.cowardice", "flame.truth", "flame.love", "flame.courage", "banishment.method"} {
		require.True(t, state.DiscoverJournalKnowledge(id, DialogueKnowledgeSource("Speaker")))
	}
	for lord := references.Faulinei; lord < references.ShadowlordCount; lord++ {
		require.True(t, state.DiscoverShadowlordKnowledge(lord, ShadowlordKnowledgeName, DialogueKnowledgeSource("Speaker")))
		require.True(t, state.DiscoverShadowlordKnowledge(lord, ShadowlordKnowledgeAssociation, DialogueKnowledgeSource("Speaker")))
		require.True(t, state.RecordShadowlordDefeatEvent(lord))
	}
	for index := range TotalShrines {
		require.True(t, state.DiscoverShrineMantraKnowledge(index, DialogueKnowledgeSource("Speaker")))
		state.SetShrineQuestActive(index, true)
		review, err := state.ReviewCodexQuest()
		require.NoError(t, err)
		before := len(state.QuestJournalEvents())
		_, err = state.CommitCodexQuestAnswer(review)
		require.NoError(t, err)
		require.Len(t, state.QuestJournalEvents(), before+1)
		event := state.QuestJournalEvents()[before]
		key, _ := ShrineMantraKnowledgeKey(index)
		assert.Equal(t, QuestJournalEventCodexAnswerLearned, event.Kind)
		assert.Equal(t, QuestJournalSubject(strings.TrimSuffix(key.ID, ".mantra")), event.Subject)
		assert.Equal(t, "Codex", event.SourceName)
		assert.Equal(t, QuestJournalEventSource("revelation"), event.Source)
		_, err = state.CommitCodexQuestAnswer(review)
		require.Error(t, err)
		assert.Len(t, state.QuestJournalEvents(), before+1)
		state.SetShrineQuestActive(index, false)
	}
	for _, dungeonName := range []string{"deceit", "despise", "destard", "wrong", "covetous", "shame", "hythloth", "doom"} {
		require.True(t, state.DiscoverWordOfPowerKnowledge(dungeonName, RevelationKnowledgeSource("Revelation")))
	}
	entries := state.CurrentQuestJournalDossier().Chronicle
	require.Len(t, entries, 40)
	for index, entry := range entries {
		assert.Equal(t, uint64(index+1), entry.Sequence)
		assert.NotEmpty(t, entry.Title)
		assert.NotEmpty(t, entry.Summary)
	}
	state.DateTime.Advance(100000)
	assert.Equal(t, entries, state.CurrentQuestJournalDossier().Chronicle)
}

func TestJournalSnapshotPreservesProvenanceWithoutHydrationChanges(t *testing.T) {
	state := newJournalTestState()
	require.True(t, state.DiscoverJournalKnowledge("flame.truth", ObservationKnowledgeSource("Look (Lycaeum)")))
	require.True(t, state.DiscoverShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeName, RevelationKnowledgeSource("Banishment")))
	var snapshot save_snapshot.SaveSnapshot
	state.captureRedux(&snapshot)
	restored := newJournalTestState()
	restored.SystemCallbacks.UI.JournalChanged = func(int, int) { t.Fatal("hydration published a live change") }
	restored.restorePlayerKnowledge(snapshot.Redux.PlayerKnowledge)
	restored.restoreQuestJournal(snapshot.Redux.QuestJournal)
	var recaptured save_snapshot.SaveSnapshot
	restored.captureRedux(&recaptured)
	assert.Equal(t, snapshot.Redux, recaptured.Redux)
	assert.Equal(t, state.CurrentQuestJournalDossier(), restored.CurrentQuestJournalDossier())
	assert.False(t, restored.DiscoverJournalKnowledge("flame.truth", DialogueKnowledgeSource("Sir Sean")))
	assert.False(t, restored.DiscoverShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeName, DialogueKnowledgeSource("Sin'Vraal")))

	// Current checkpoints may intentionally lack historical notes. Hydration must
	// not backfill them and silently alter full-scene checkpoint equality.
	restored.restoreQuestJournal(nil)
	assert.Empty(t, restored.QuestJournalEvents())
}

func TestJournalNativeCaptureHydrationKeepsAuthorityAndDefaultsPresentation(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	require.True(t, state.DiscoverJournalKnowledge("domain.hatred", DialogueKnowledgeSource("Michael")))
	require.True(t, state.DiscoverJournalKnowledge("flame.truth", ObservationKnowledgeSource("Look (Lycaeum)")))
	require.True(t, state.DiscoverShadowlordKnowledge(references.Astaroth, ShadowlordKnowledgeName, RevelationKnowledgeSource("Banishment")))
	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Equal(t, journal.EmptyPresentation(), snapshot.Presentation)
	snapshot.Presentation.History = "A saved message."
	snapshot.Presentation.Visits.Record(journal.ChronicleTab, 1)
	restored, err := NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences,
		xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
	require.NoError(t, err)
	restored.SystemCallbacks = journalTestCallbacks()
	restored.SystemCallbacks.UI.JournalChanged = func(int, int) { t.Fatal("recapture published a live change") }
	recaptured, err := restored.CaptureSaveSnapshot()
	require.NoError(t, err)
	snapshot.Presentation = journal.EmptyPresentation()
	assert.Equal(t, snapshot, recaptured, "core-only capture does not own scene history or visits")
	assert.Equal(t, state.CurrentQuestJournalDossier(), restored.CurrentQuestJournalDossier())
}
