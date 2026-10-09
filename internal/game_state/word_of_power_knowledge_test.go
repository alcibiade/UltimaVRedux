package game_state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

func TestWordOfPowerKnowledgeCatalogAndChronicle(t *testing.T) {
	state := newJournalTestState()
	before := state.CurrentQuestJournalDossier()
	var changes [][2]int
	state.SystemCallbacks.UI.JournalChanged = func(tab, entry int) {
		changes = append(changes, [2]int{tab, entry})
	}
	for index, test := range []struct {
		dungeon string
		word    string
		speaker string
		title   string
		summary string
	}{
		{"Deceit", "FALLAX", "Malifora", "Deceit", "The Word of Power for Deceit is FALLAX."},
		{"Despise", "VILIS", "Annon", "Despise", "The Word of Power for Despise is VILIS."},
		{"Destard", "AIPONI", "Goeth", "Destard", "Goeth spoke the runes AIPONI for Destard."},
		{"Wrong", "MALUM", "Felespar", "Wrong", "The Word of Power for Wrong is MALUM."},
		{"Covetous", "AVIDUS", "Fiona", "Covetous", "The Word of Power for Covetous is AVIDUS."},
		{"Shame", "INFAMA", "Sindar", "INFAMA", "Sindar spoke the word INFAMA."},
		{"Hythloth", "IGNAVUS", "Hassad", "Hythloth", "The Word of Power for Hythloth is IGNAVUS."},
		{"Doom", "VERAMOCOR", "Codex", "VERAMOCOR", "The Codex revealed VERAMOCOR to unlock the gate."},
	} {
		t.Run(test.dungeon, func(t *testing.T) {
			name := strings.ToLower(test.dungeon)
			key, supported := WordOfPowerKnowledgeKey(name)
			require.True(t, supported)
			assert.Equal(t, KnowledgeKey{Category: KnowledgeCategoryDungeon, ID: name + ".word_of_power"}, key)
			_, known := state.WordOfPowerKnowledge(name)
			assert.False(t, known)
			source := DialogueKnowledgeSource(test.speaker)
			if name == "doom" {
				source = RevelationKnowledgeSource(test.speaker)
			}
			firstDate := state.DateTime.DateOnly()
			require.True(t, state.DiscoverWordOfPowerKnowledge(name, source))
			entry, known := state.WordOfPowerKnowledge(name)
			require.True(t, known)
			assert.Equal(t, source, entry.Source)
			assert.Equal(t, firstDate, *entry.DiscoveredOn)
			assert.False(t, entry.Unread)
			assert.Nil(t, entry.SpellFormula)
			entry.DiscoveredOn.Day++
			state.DateTime.Advance(1440)
			assert.False(t, state.DiscoverWordOfPowerKnowledge(name, ObservationKnowledgeSource("Later observation")))
			retained, known := state.WordOfPowerKnowledge(name)
			require.True(t, known)
			assert.Equal(t, source, retained.Source)
			assert.Equal(t, firstDate, *retained.DiscoveredOn)
			event := state.QuestJournalEvents()[index]
			assert.Equal(t, uint64(index+1), event.Sequence)
			assert.Equal(t, QuestJournalEventKnowledgeLearned, event.Kind)
			assert.Equal(t, QuestJournalSubject("dungeon."+name+".word_of_power"), event.Subject)
			assert.Equal(t, QuestJournalEventSource(source.Kind), event.Source)
			assert.Equal(t, source.DisplayName, event.SourceName)
			note := state.CurrentQuestJournalDossier().Chronicle[index]
			assert.Equal(t, "Word of Power: "+test.title, note.Title)
			assert.Equal(t, test.summary, note.Summary)
			assert.Contains(t, note.Summary, test.word)
			assert.Equal(t, source.DisplayName, note.Source)
			assert.Equal(t, firstDate, *note.OccurredOn)
			assert.Equal(t, [2]int{journal.ChronicleTab, index + 1}, changes[index])
		})
	}
	assert.Len(t, changes, 8, "duplicates never notify and no fixed rows change")
	dossier := state.CurrentQuestJournalDossier()
	assert.Equal(t, before.Virtues, dossier.Virtues)
	assert.Equal(t, before.Shadowlords, dossier.Shadowlords)
	assert.Equal(t, [8]bool{}, state.DungeonStates, "learning does not unseal dungeons")
}

func TestWordOfPowerKnowledgeRejectsNonCanonicalNames(t *testing.T) {
	state := newJournalTestState()
	state.SystemCallbacks.UI.JournalChanged = func(int, int) { t.Fatal("invalid discovery notified") }
	for _, name := range []string{"", "Destard", " destard", "destard ", "dest", "destard.word_of_power", "dungeon.destard.word_of_power", "AIPONI", "INOPIA", "britain"} {
		_, supported := WordOfPowerKnowledgeKey(name)
		assert.False(t, supported, name)
		assert.False(t, state.DiscoverWordOfPowerKnowledge(name, DialogueKnowledgeSource("Goeth")), name)
		_, known := state.WordOfPowerKnowledge(name)
		assert.False(t, known, name)
	}
	assert.Empty(t, state.QuestJournalEvents())
	assert.Empty(t, state.KnowledgeSnapshot())
	assert.Panics(t, func() { state.DiscoverWordOfPowerKnowledge("destard", ExperimentationKnowledgeSource()) })
}

func TestWordOfPowerProjectionOnlyExposesLearnedSpokenClue(t *testing.T) {
	for _, test := range []struct {
		dungeon string
		word    string
		source  KnowledgeSource
	}{
		{"destard", "AIPONI", DialogueKnowledgeSource("Goeth")},
		{"shame", "INFAMA", DialogueKnowledgeSource("Sindar")},
		{"doom", "VERAMOCOR", RevelationKnowledgeSource("Codex")},
	} {
		t.Run(test.dungeon, func(t *testing.T) {
			state := newJournalTestState()
			assert.Empty(t, state.CurrentQuestJournalDossier().Chronicle)
			firstDate := state.DateTime.DateOnly()
			require.True(t, state.DiscoverWordOfPowerKnowledge(test.dungeon, test.source))
			notes := state.CurrentQuestJournalDossier().Chronicle
			require.Len(t, notes, 1)
			if test.dungeon == "destard" {
				assert.Equal(t, "Word of Power: Destard", notes[0].Title)
			} else {
				assert.Equal(t, "Word of Power: "+test.word, notes[0].Title)
			}
			assert.Equal(t, test.source.DisplayName, notes[0].Source)
			assert.Equal(t, firstDate, *notes[0].OccurredOn)
			assert.Equal(t, QuestJournalSubject("dungeon."+test.dungeon+".word_of_power"), state.QuestJournalEvents()[0].Subject)
			searchText := strings.ToLower(notes[0].Title + " " + notes[0].Summary + " " + notes[0].Source + " " + notes[0].DisplayName)
			assert.Contains(t, searchText, strings.ToLower(test.word))
			assert.Contains(t, searchText, strings.ToLower(test.source.DisplayName))
			for _, hidden := range []string{"destard", "shame", "doom", "inopia", "deceit", "despise", "wrong", "covetous", "hythloth", "coordinates", "yell", ".word_of_power"} {
				if hidden == "destard" && test.dungeon == "destard" {
					continue
				}
				assert.NotContains(t, searchText, hidden)
			}
			var saved save_snapshot.SaveSnapshot
			state.captureRedux(&saved)
			restored := newJournalTestState()
			restored.SystemCallbacks.UI.JournalChanged = func(int, int) { t.Fatal("restored knowledge notified") }
			restored.restorePlayerKnowledge(saved.Redux.PlayerKnowledge)
			restored.restoreQuestJournal(saved.Redux.QuestJournal)
			assert.Equal(t, notes, restored.CurrentQuestJournalDossier().Chronicle)
			assert.False(t, restored.DiscoverWordOfPowerKnowledge(test.dungeon, test.source))
			var recaptured save_snapshot.SaveSnapshot
			restored.captureRedux(&recaptured)
			assert.Equal(t, saved.Redux, recaptured.Redux)
		})
	}
}
