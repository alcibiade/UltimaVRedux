package main

import (
	"testing"

	conversationpatchdata "github.com/bradhannah/UltimaVRedux/data/conversation-patches"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func TestOriginalChamfortPasswordPresentation(t *testing.T) {
	catalogue, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	program, found := catalogue.Original(semantic.OriginalSourceTowne, 17)
	require.True(t, found)
	definition, err := program.Definition()
	require.NoError(t, err)
	for _, named := range []bool{false, true} {
		dialog := newBoundaryTalkDialog(t, definition)
		scene := dialog.gameScene
		state := scene.gameState
		scene.dialogStack.PushModalDialog(dialog)
		dialog.mount()
		submit := func(input string) {
			t.Helper()
			require.True(t, dialog.waitingForInput)
			dialog.TextInput.SetText(input)
			dialog.onEnter()
			require.NoError(t, dialog.currentResponse.Error)
		}
		if named {
			submit("NAME")
		}
		submit("RESI")
		submit("TERR")
		dialog.finishCurrentPause()
		submit("Y")
		dialog.finishCurrentPause()
		submit(dialog.callbacks.GetAvatarName())
		require.True(t, dialog.isPausing)
		require.NotContains(t, dialog.exchange.Passages, journal.Passage{Text: " DAWN", Runic: true})
		require.Empty(t, state.CurrentQuestJournalDossier().Chronicle)
		dialog.finishCurrentPause()
		require.Contains(t, dialog.exchange.Passages, journal.Passage{Text: " DAWN", Runic: true})
		notes := state.CurrentQuestJournalDossier().Chronicle
		require.Len(t, notes, 1, "automatic, without Tab capture")
		require.Equal(t, "Resistance password", notes[0].Title)
		require.Equal(t, "The Resistance password is DAWN.", notes[0].Summary)
		require.Equal(t, "Chamfort", notes[0].Source, "saved clues identify their speaker even without NAME")
		dialog.finishCurrentPause()
		for _, topic := range []string{"RESI", "TERR"} {
			submit(topic)
		}
		dialog.finishCurrentPause()
		submit("Y")
		dialog.finishCurrentPause()
		submit(dialog.callbacks.GetAvatarName())
		dialog.finishCurrentPause()
		dialog.finishCurrentPause()
		require.Equal(t, notes, state.CurrentQuestJournalDossier().Chronicle)
		submit("MANT")
		submit("JERE")
		_, learned := state.ShrineMantraKnowledge(3)
		require.True(t, learned, "the existing independent mantra discovery survives")
	}
}

func TestOriginalChamfortPasswordNotRevealed(t *testing.T) {
	catalogue, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	program, found := catalogue.Original(semantic.OriginalSourceTowne, 17)
	require.True(t, found)
	definition, err := program.Definition()
	require.NoError(t, err)
	for _, inputs := range [][]string{
		{}, {"RESI", "N"}, {"RESI", "TERR", "N"},
		{"RESI", "TERR", "Y", "Nobody"},
		{"RESI", "TERR", "Y", "@party"},
	} {
		dialog := newBoundaryTalkDialog(t, definition)
		dialog.gameScene.dialogStack.PushModalDialog(dialog)
		dialog.mount()
		for _, input := range inputs {
			for dialog.isPausing {
				dialog.finishCurrentPause()
			}
			if input == "@party" {
				input = dialog.callbacks.GetAvatarName()
			}
			dialog.TextInput.SetText(input)
			dialog.onEnter()
		}
		// The successful path is cancelled at the pause BEFORE the password.
		dialog.Close()
		require.Empty(t, dialog.gameScene.gameState.CurrentQuestJournalDossier().Chronicle)
	}
}
