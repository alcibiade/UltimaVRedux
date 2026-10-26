package game_state

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActionZtatsShowsDetachedDossierWithoutMutatingState(t *testing.T) {
	state := newZStatsTestState(newZStatsTestCharacter("Avatar", 'A', 'G', 3, 250))
	beforeDate := state.DateTime
	beforeInventory := state.PartyState.Inventory.LockerSnapshot()
	beforeKnowledge := state.KnowledgeSnapshot()
	var shown []ZStatsDossier
	state.SystemCallbacks = &SystemCallbacks{
		UI: UICallbacks{
			ShowZStatsDossier: func(dossier ZStatsDossier) {
				shown = append(shown, dossier)
			},
		},
	}

	state.ActionZtats()

	require.Len(t, shown, 1)
	require.Len(t, shown[0].Characters, 1)
	require.Equal(t, "Avatar", shown[0].Characters[0].Name)
	require.Equal(t, beforeDate, state.DateTime)
	require.Equal(t, beforeInventory, state.PartyState.Inventory.LockerSnapshot())
	require.Equal(t, beforeKnowledge, state.KnowledgeSnapshot())
}

func TestNewSystemCallbacksRejectsNilDossierCallbacks(t *testing.T) {
	for _, test := range []struct {
		clear func(*UICallbacks)
		want  error
	}{
		{clear: func(ui *UICallbacks) { ui.ShowZStatsDossier = nil }, want: errZStatsCallbackRequired},
		{clear: func(ui *UICallbacks) { ui.ShowQuestJournalDossier = nil }, want: errQuestJournalCallbackRequired},
	} {
		valid := NewMockSystemCallbacks(t).ToSystemCallbacks()
		uiCallbacks := valid.UI
		test.clear(&uiCallbacks)

		callbacks, err := NewSystemCallbacks(
			valid.Message, valid.Visual, valid.Audio, valid.Screen, valid.Flow, valid.Talk, uiCallbacks, valid.Combat,
		)

		require.Nil(t, callbacks)
		require.ErrorIs(t, err, test.want)
	}
}
