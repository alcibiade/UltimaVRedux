package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionQuestJournalShowsDetachedDossierWithoutAdvancingGameplay(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	beforeDate := state.DateTime
	beforeInventory := state.PartyState.Inventory.LockerSnapshot()
	var shown []QuestJournalDossier
	state.SystemCallbacks.UI.ShowQuestJournalDossier = func(dossier QuestJournalDossier) {
		shown = append(shown, dossier)
	}
	callbacks.Reset()

	state.ActionQuestJournal()

	require.Len(t, shown, 1)
	require.Len(t, shown[0].Virtues, TotalShrines)
	require.Len(t, shown[0].Shadowlords, 3)
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Equal(t, beforeInventory, state.PartyState.Inventory.LockerSnapshot())
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestNewSystemCallbacksRejectsNilShowQuestJournalDossier(t *testing.T) {
	valid := NewMockSystemCallbacks(t).ToSystemCallbacks()
	uiCallbacks := valid.UI
	uiCallbacks.ShowQuestJournalDossier = nil

	callbacks, err := NewSystemCallbacks(
		valid.Message,
		valid.Visual,
		valid.Audio,
		valid.Screen,
		valid.Flow,
		valid.Talk,
		uiCallbacks,
		valid.Combat,
	)

	require.Nil(t, callbacks)
	require.ErrorIs(t, err, errQuestJournalCallbackRequired)
}
