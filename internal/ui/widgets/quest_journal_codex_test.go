package widgets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestQuestJournalCompleteCodexSearchAndPaging(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).WithPlayerAt(10, 20).
		WithSystemCallbacks().Build()
	state.DiscoverWordOfPowerKnowledge("doom", game_state.RevelationKnowledgeSource("Codex of Ultimate Wisdom"))
	partial := NewQuestJournalDialog(state.CurrentQuestJournalDossier(), testQuestJournalConfig(newWidgetTestKeyboard(0)))
	partial.beginSearch()
	partial.setSearchQuery("Shame egress")
	require.Empty(t, partial.searchResults, "word knowledge must not disclose the complete clue")
	require.True(t, state.RecordCompleteCodexRevelation())
	for _, size := range [][2]int{{1280, 720}, {1920, 1080}, {2560, 1440}, {3840, 2160}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			setQuestJournalTestResolution(t, size[0], size[1])
			dialog := NewQuestJournalDialog(state.CurrentQuestJournalDossier(), testQuestJournalConfig(newWidgetTestKeyboard(0)))
			for _, query := range []string{"Shame", "Underworld", "darkness", "gate", "VERAMOCOR", "stealers souls", "awaits coming"} {
				dialog.beginSearch()
				dialog.setSearchQuery(query)
				require.Len(t, dialog.searchResults, 1, query)
				dialog.openSearchResult()
				require.Equal(t, QuestJournalChronicleTab, dialog.GetActiveTab())
				require.Zero(t, dialog.GetSelectedIndex())
			}
			var rendered strings.Builder
			for _, page := range dialog.detailPages() {
				for _, line := range page {
					rendered.WriteString(line.label + line.value)
					rendered.WriteByte(' ')
				}
			}
			require.Contains(t, rendered.String(), "awaits thy coming.")
			for _, hidden := range []string{"Doom", "dungeon.doom.word_of_power"} {
				dialog.beginSearch()
				dialog.setSearchQuery(hidden)
				require.Empty(t, dialog.searchResults, hidden)
			}
		})
	}
}
