package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestTaughtSpellKnowledgeCatalogChronicleAndNativeRoundTrip(t *testing.T) {
	for _, lesson := range []struct {
		spell   references.Spell
		teacher string
	}{
		{references.RelXenBet, "Grendel"}, {references.InXenMani, "Lord Stuart"},
	} {
		t.Run(lesson.teacher, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
			state.playerKnowledge = newPlayerKnowledge()
			state.questJournal = newQuestJournal()
			for _, spell := range []references.Spell{references.RelXenBet, references.InXenMani} {
				state.PartyState.Inventory.Spells.Set(spell, 1)
				for _, entry := range state.CurrentSpellCatalog() {
					require.NotEqual(t, spell.ID(), entry.ID)
				}
			}
			require.Empty(t, state.CurrentQuestJournalDossier().Chronicle)
			require.True(t, state.DiscoverSpellKnowledge(lesson.spell, DialogueKnowledgeSource(lesson.teacher)))
			require.False(t, state.DiscoverSpellKnowledge(lesson.spell, DialogueKnowledgeSource(lesson.teacher)))
			chronicle := state.CurrentQuestJournalDossier().Chronicle
			require.Len(t, chronicle, 1)
			assert.Equal(t, lesson.teacher, chronicle[0].Source)
			require.NotNil(t, chronicle[0].OccurredOn)
			seen := 0
			for _, entry := range state.CurrentSpellCatalog() {
				if entry.ID == references.RelXenBet.ID() || entry.ID == references.InXenMani.ID() {
					assert.Equal(t, lesson.spell.ID(), entry.ID)
					seen++
				}
			}
			assert.Equal(t, 1, seen)
			snapshot, err := state.CaptureSaveSnapshot()
			require.NoError(t, err)
			reloaded, err := NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
			require.NoError(t, err)
			assert.Equal(t, chronicle, reloaded.CurrentQuestJournalDossier().Chronicle)
			assert.Equal(t, state.CurrentSpellCatalog(), reloaded.CurrentSpellCatalog())
		})
	}
}
