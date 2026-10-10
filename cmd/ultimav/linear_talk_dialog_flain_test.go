package main

import (
	"strings"
	"testing"

	conversationpatchdata "github.com/bradhannah/UltimaVRedux/data/conversation-patches"
	"github.com/bradhannah/UltimaVRedux/internal/conversation"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func TestOriginalFlainPasswordCommitsOnlyAfterFinalKeyContinuation(t *testing.T) {
	definition := originalFlainTalkDefinition(t)
	for _, name := range []string{"MALI", "ANNO", "GOET", "FELE", "FION", "SIND"} {
		t.Run(name, func(t *testing.T) {
			dialog := newBoundaryTalkDialog(t, definition)
			scene := dialog.gameScene
			state := scene.gameState
			for attempt := range 2 {
				if attempt != 0 {
					callbacks := NewGameActionCallbacks(scene, references.NPCReference{Location: references.SkaraBrae}, nil)
					engine, err := conversation.NewDeferredSemanticConversationEngine(definition, callbacks, false)
					require.NoError(t, err)
					callbacks.SetNPCName(engine.ExtractNPCName())
					dialog = &LinearTalkDialog{gameScene: scene, callbacks: callbacks, engine: engine}
					dialog.initializeResizeableVisualElements()
					dialog.startConversation()
				}
				scene.dialogStack.PushModalDialog(dialog)
				dialog.mount()
				for _, input := range []string{"OPPR", "", "JUDG", "Y", name} {
					submitFlainTalkInput(t, dialog, input)
				}
				require.True(t, dialog.isPausing)
				require.Len(t, state.CurrentQuestJournalDossier().Chronicle, attempt)
				dialog.finishCurrentPause()
				require.True(t, dialog.waitingForKeypressContinue)
				require.Contains(t, strings.Join(strings.Fields(dialog.Output.GetOutputStr(true)), " "), "That one will soon perish!")
				require.NotContains(t, dialog.Output.GetOutputStr(true), "Impera")
				require.Len(t, state.CurrentQuestJournalDossier().Chronicle, attempt, "naming a council member is not the disclosure")
				revision := scene.journalVisits.Revision
				dialog.waitingForKeypressContinue = false
				dialog.currentResponse = dialog.engine.ProcessInput("")
				require.NoError(t, dialog.currentResponse.Error)
				require.Len(t, dialog.callbacks.pendingKnowledge, 1)
				require.Len(t, state.CurrentQuestJournalDossier().Chronicle, attempt, "executed but unpresented speech is still unknown")
				dialog.processCurrentResponse()
				require.True(t, dialog.waitingForDismissal)
				require.Contains(t, strings.Join(strings.Fields(dialog.Output.GetOutputStr(true)), " "), "Say to her the password, 'Impera'.")
				notes := state.CurrentQuestJournalDossier().Chronicle
				require.Len(t, notes, 1, "automatic, without excerpt capture")
				require.Equal(t, "Oppression password", notes[0].Title)
				require.Equal(t, "The Oppression password is IMPERA.", notes[0].Summary)
				require.Equal(t, "Flain", notes[0].Source, "attribution does not require NAME")
				require.Empty(t, dialog.callbacks.pendingKnowledge)
				if attempt == 0 {
					require.Greater(t, scene.journalVisits.Revision, revision)
				} else {
					require.Equal(t, revision, scene.journalVisits.Revision, "repeat disclosure is quiet")
				}
				dialog.Close()
			}
		})
	}
}

func TestOriginalFlainPasswordUnrevealedBranchesAndCancellation(t *testing.T) {
	definition := originalFlainTalkDefinition(t)
	for _, test := range []struct {
		name            string
		inputs          []string
		finishPause     bool
		queueDisclosure bool
	}{
		{name: "conversation start"},
		{name: "guessed password", inputs: []string{"IMPERA"}},
		{name: "wrong referral", inputs: []string{"OPPR", "", "Nobody"}, finishPause: true},
		{name: "refusal", inputs: []string{"OPPR", "", "DRYD", "N"}},
		{name: "Hassad", inputs: []string{"OPPR", "", "JUDG", "Y", "HASS"}, finishPause: true},
		{name: "wrong name", inputs: []string{"OPPR", "", "JUDG", "Y", "Nobody"}, finishPause: true},
		{name: "cancel before acceptance speech", inputs: []string{"OPPR", "", "JUDG", "Y", "MALI"}},
		{name: "cancel before password key", inputs: []string{"OPPR", "", "JUDG", "Y", "MALI"}, finishPause: true},
		{name: "cancel unpresented password", inputs: []string{"OPPR", "", "JUDG", "Y", "MALI"}, finishPause: true, queueDisclosure: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dialog := newBoundaryTalkDialog(t, definition)
			scene := dialog.gameScene
			scene.dialogStack.PushModalDialog(dialog)
			dialog.mount()
			for _, input := range test.inputs {
				submitFlainTalkInput(t, dialog, input)
			}
			if test.finishPause {
				dialog.finishCurrentPause()
			}
			if test.queueDisclosure {
				require.True(t, dialog.waitingForKeypressContinue)
				dialog.waitingForKeypressContinue = false
				dialog.currentResponse = dialog.engine.ProcessInput("")
				require.NoError(t, dialog.currentResponse.Error)
				require.Len(t, dialog.callbacks.pendingKnowledge, 1)
			}
			scene.dialogStack.MustRemoveWidgetInstance(dialog)
			dialog.Update()
			require.True(t, dialog.closed)
			require.Empty(t, dialog.callbacks.pendingKnowledge)
			require.Empty(t, scene.gameState.CurrentQuestJournalDossier().Chronicle)
			require.False(t, scene.journalNotice.pending)
			_, known := scene.gameState.JournalKnowledge("oppression.password")
			require.False(t, known)
		})
	}
}

func TestOriginalElistariaPasswordAcceptanceDoesNotDiscoverGuessedPassword(t *testing.T) {
	catalogue, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	program, found := catalogue.Original(semantic.OriginalSourceKeep, 9)
	require.True(t, found)
	definition, err := program.Definition()
	require.NoError(t, err)
	for _, inputs := range [][]string{{"IMPERA"}, {"OPPR", "IMPERA"}} {
		dialog := newBoundaryTalkDialog(t, definition)
		scene := dialog.gameScene
		scene.dialogStack.PushModalDialog(dialog)
		dialog.mount()
		for _, input := range inputs {
			submitFlainTalkInput(t, dialog, input)
		}
		require.True(t, dialog.waitingForKeypressContinue, "correct password still reaches Elistaria's acceptance")
		require.Contains(t, strings.Join(strings.Fields(dialog.Output.GetOutputStr(true)), " "), "infiltrating the Resistance")
		require.Empty(t, scene.gameState.CurrentQuestJournalDossier().Chronicle)
		_, known := scene.gameState.JournalKnowledge("oppression.password")
		require.False(t, known)
		dialog.Close()
	}
}

func originalFlainTalkDefinition(t *testing.T) *semantic.Conversation {
	t.Helper()
	catalogue, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	program, found := catalogue.Original(semantic.OriginalSourceTowne, 38)
	require.True(t, found)
	definition, err := program.Definition()
	require.NoError(t, err)

	return definition
}

func submitFlainTalkInput(t *testing.T, dialog *LinearTalkDialog, value string) {
	t.Helper()
	if value == "" {
		require.True(t, dialog.waitingForKeypressContinue)
		dialog.waitingForKeypressContinue = false
		dialog.currentResponse = dialog.engine.ProcessInput("")
		dialog.processCurrentResponse()
	} else {
		require.True(t, dialog.waitingForInput)
		dialog.TextInput.SetText(value)
		dialog.onEnter()
	}
	require.NoError(t, dialog.currentResponse.Error)
}
