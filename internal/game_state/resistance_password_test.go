package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/require"
)

func TestResistancePasswordChronicleRoundTrip(t *testing.T) {
	state := newJournalTestState()
	require.Empty(t, state.CurrentQuestJournalDossier().Chronicle)
	changes := 0
	state.SystemCallbacks.UI.JournalChanged = func(tab, entry int) {
		require.Equal(t, journal.ChronicleTab, tab)
		require.Equal(t, 1, entry)
		changes++
	}
	source := DialogueKnowledgeSource("Chamfort")
	require.True(t, state.DiscoverJournalKnowledge("resistance.password", source))
	notes := state.CurrentQuestJournalDossier().Chronicle
	require.Len(t, notes, 1)
	require.Equal(t, "The Resistance password is DAWN.", notes[0].Summary)
	require.Equal(t, "Chamfort", notes[0].Source)
	require.Equal(t, 1, changes)
	require.False(t, state.DiscoverJournalKnowledge("resistance.password", DialogueKnowledgeSource("Later speaker")))
	var saved save_snapshot.SaveSnapshot
	state.captureRedux(&saved)
	restored := newJournalTestState()
	restored.SystemCallbacks.UI.JournalChanged = func(int, int) { t.Fatal("hydration must be quiet") }
	restored.restorePlayerKnowledge(saved.Redux.PlayerKnowledge)
	restored.restoreQuestJournal(saved.Redux.QuestJournal)
	require.Equal(t, notes, restored.CurrentQuestJournalDossier().Chronicle)
	require.False(t, restored.DiscoverJournalKnowledge("resistance.password", source))
	var recaptured save_snapshot.SaveSnapshot
	restored.captureRedux(&recaptured)
	require.Equal(t, saved.Redux, recaptured.Redux)
}
