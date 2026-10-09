package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/require"
)

func TestOppressionPasswordChroniclePreservesFirstDisclosureAndHydratesQuietly(t *testing.T) {
	state := newJournalTestState()
	require.Empty(t, state.CurrentQuestJournalDossier().Chronicle)
	_, known := state.JournalKnowledge("oppression.password")
	require.False(t, known)
	changes := 0
	state.SystemCallbacks.UI.JournalChanged = func(tab, entry int) {
		require.Equal(t, journal.ChronicleTab, tab)
		require.Equal(t, 1, entry)
		changes++
	}
	source := DialogueKnowledgeSource("Flain")
	require.True(t, state.DiscoverJournalKnowledge("oppression.password", source))
	notes := state.CurrentQuestJournalDossier().Chronicle
	require.Len(t, notes, 1)
	require.Equal(t, "Oppression password", notes[0].Title)
	require.Equal(t, "The Oppression password is IMPERA.", notes[0].Summary)
	require.Equal(t, "Flain", notes[0].Source)
	first, known := state.JournalKnowledge("oppression.password")
	require.True(t, known)
	require.NotNil(t, first.DiscoveredOn)
	state.DateTime.Day++
	require.False(t, state.DiscoverJournalKnowledge("oppression.password", DialogueKnowledgeSource("Later speaker")))
	again, known := state.JournalKnowledge("oppression.password")
	require.True(t, known)
	require.Equal(t, first, again)
	require.Equal(t, 1, changes)

	// Exercise gameplay capture/hydration independently of the versioned codec.
	var saved save_snapshot.SaveSnapshot
	state.captureRedux(&saved)
	restored := newJournalTestState()
	restored.SystemCallbacks.UI.JournalChanged = func(int, int) { t.Fatal("hydration must be quiet") }
	restored.restorePlayerKnowledge(saved.Redux.PlayerKnowledge)
	restored.restoreQuestJournal(saved.Redux.QuestJournal)
	require.Equal(t, notes, restored.CurrentQuestJournalDossier().Chronicle)
	require.False(t, restored.DiscoverJournalKnowledge("oppression.password", source))
	var recaptured save_snapshot.SaveSnapshot
	restored.captureRedux(&recaptured)
	require.Equal(t, saved.Redux, recaptured.Redux)
}
