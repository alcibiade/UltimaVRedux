package widgets

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/stretchr/testify/require"
)

func TestQuestJournalStonesSearchAndStableIdentity(t *testing.T) {
	dossier := testQuestJournalDossier()
	dossier.Chronicle = []game_state.QuestJournalChronicleEntry{{
		Sequence: 7, Title: "Stones: harpsichord melody", Source: "Lord Kenneth",
		Summary: "Stand immediately north of the harpsichord. Press the top-row number keys or numpad keys one at a time: 6 7 8 9 8 7 8 7 6 7 6 5 3.",
	}}
	dialog := NewQuestJournalDialog(dossier, testQuestJournalConfig(newWidgetTestKeyboard(0)))
	for _, query := range []string{"stones", "Kenneth", "harpsichord north", "numpad", "6 7 8 9"} {
		dialog.beginSearch()
		dialog.setSearchQuery(query)
		require.Len(t, dialog.searchResults, 1, query)
		dialog.openSearchResult()
		require.Equal(t, QuestJournalChronicleTab, dialog.GetActiveTab())
		require.Equal(t, 0, dialog.GetSelectedIndex())
	}
	dialog.beginSearch()
	dialog.setSearchQuery("journal.stones.melody")
	require.Empty(t, dialog.searchResults, "wire identity is not searchable hidden text")
}
