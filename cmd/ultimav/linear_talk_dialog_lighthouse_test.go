package main

import (
	"strings"
	"testing"

	conversationpatchdata "github.com/bradhannah/UltimaVRedux/data/conversation-patches"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func lighthouseTalkDefinition(t *testing.T, number int) *semantic.Conversation {
	t.Helper()
	catalogue, err := catalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	program, found := catalogue.Original(semantic.OriginalSourceDwelling, number)
	require.True(t, found)
	definition, err := program.Definition()
	require.NoError(t, err)
	return definition
}

func submitLighthouseInput(t *testing.T, dialog *LinearTalkDialog, value string) {
	t.Helper()
	require.True(t, dialog.waitingForInput)
	dialog.TextInput.SetText(value)
	dialog.onEnter()
	require.NoError(t, dialog.currentResponse.Error)
}

func TestOriginalKennethMountedDiscovery(t *testing.T) {
	definition := lighthouseTalkDefinition(t, 14)
	for _, answer := range []string{"Y", "N"} {
		t.Run(answer, func(t *testing.T) {
			dialog := newBoundaryTalkDialog(t, definition)
			scene, state := dialog.gameScene, dialog.gameScene.gameState
			require.Empty(t, state.KnowledgeSnapshot())
			scene.dialogStack.PushModalDialog(dialog)
			dialog.mount()
			for _, value := range []string{"JOB", "STONES", "Y", answer} {
				submitLighthouseInput(t, dialog, value)
			}
			require.True(t, dialog.waitingForKeypressContinue)
			require.Empty(t, state.KnowledgeSnapshot())
			// The interpreter may have produced the clue, but only mounted output grants it.
			dialog.currentResponse = dialog.engine.ProcessInput("")
			require.NotEmpty(t, dialog.callbacks.pendingKnowledge)
			require.Empty(t, state.KnowledgeSnapshot())
			dialog.processCurrentResponse()
			notes := state.CurrentQuestJournalDossier().Chronicle
			require.Len(t, notes, 1)
			require.Equal(t, "Lord Kenneth", notes[0].Source)
			require.Contains(t, notes[0].Summary, "6 7 8 9 8 7 8 7 6 7 6 5 3")
			require.Contains(t, strings.Join(strings.Fields(dialog.Output.GetOutputStr(false)), " "), "6 7 8 9 8 7 8 7 6 7 6 5 3")
			revision := scene.journalVisits.Revision
			for _, value := range []string{"MUSIC", "Y", answer} {
				submitLighthouseInput(t, dialog, value)
			}
			dialog.currentResponse = dialog.engine.ProcessInput("")
			dialog.processCurrentResponse()
			require.Equal(t, notes, state.CurrentQuestJournalDossier().Chronicle)
			require.Equal(t, revision, scene.journalVisits.Revision)
		})
	}
}

func TestOriginalKennethUnseenTeachingNeverDiscovered(t *testing.T) {
	definition := lighthouseTalkDefinition(t, 14)
	for _, inputs := range [][]string{{}, {"NAME"}, {"JOB"}, {"MUSIC", "N"}, {"STONES", "Y", "N"}} {
		waitsForContinuation := false
		for _, queueUnpresented := range []bool{false, true} {
			if queueUnpresented && !waitsForContinuation {
				continue // Without a key continuation this is the same cancellation path.
			}
			dialog := newBoundaryTalkDialog(t, definition)
			dialog.gameScene.dialogStack.PushModalDialog(dialog)
			dialog.mount()
			for _, value := range inputs {
				submitLighthouseInput(t, dialog, value)
			}
			waitsForContinuation = dialog.waitingForKeypressContinue
			if queueUnpresented && dialog.waitingForKeypressContinue {
				dialog.currentResponse = dialog.engine.ProcessInput("")
			}
			dialog.Close()
			require.Empty(t, dialog.gameScene.gameState.KnowledgeSnapshot())
			require.Empty(t, dialog.gameScene.gameState.CurrentQuestJournalDossier().Chronicle)
		}
	}
}

func TestOriginalArbuthnotMountedStory(t *testing.T) {
	dialog := newBoundaryTalkDialog(t, lighthouseTalkDefinition(t, 15))
	dialog.gameScene.dialogStack.PushModalDialog(dialog)
	dialog.mount()
	for _, value := range []string{"AVAT", "YES", "Y", dialog.callbacks.GetAvatarName()} {
		submitLighthouseInput(t, dialog, value)
	}
	for range 2 {
		require.True(t, dialog.waitingForKeypressContinue)
		dialog.currentResponse = dialog.engine.ProcessInput("")
		dialog.processCurrentResponse()
	}
	require.True(t, dialog.waitingForInput)
	speech := strings.Join(strings.Fields(dialog.Output.GetOutputStr(false)), " ")
	for _, fragment := range []string{"I minted a magical coin!", "It was in the symbol of the Codex!", "commisioned secretly by persons unknown even to me", "call back the one true Avatar!"} {
		require.Equal(t, 1, strings.Count(speech, fragment))
	}
	require.Empty(t, dialog.gameScene.gameState.KnowledgeSnapshot())
	require.Empty(t, dialog.gameScene.gameState.CurrentQuestJournalDossier().Chronicle)
	submitLighthouseInput(t, dialog, "BYE")
	require.True(t, dialog.waitingForDismissal)
}
