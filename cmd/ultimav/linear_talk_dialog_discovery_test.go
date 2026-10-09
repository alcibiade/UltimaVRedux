package main

import (
	"strings"
	"testing"
	"time"

	conversationpatchdata "github.com/bradhannah/UltimaVRedux/data/conversation-patches"
	"github.com/bradhannah/UltimaVRedux/internal/conversation"
	conversationcatalog "github.com/bradhannah/UltimaVRedux/internal/conversation/catalog"
	"github.com/bradhannah/UltimaVRedux/internal/conversation/semantic"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func newBoundaryTalkDialog(t *testing.T, definition *semantic.Conversation) *LinearTalkDialog {
	t.Helper()
	state := newGameActionCallbacksKnowledgeState(t)
	scene := &GameScene{gameState: state, keyboard: newPresentationTestKeyboard(), now: func() time.Time { return time.Unix(100, 0) }}
	_, scene.output = newMainOutput(1280)
	state.SystemCallbacks.UI.JournalChanged = scene.journalChanged
	callbacks := NewGameActionCallbacks(scene, references.NPCReference{Location: references.Britain}, nil)
	engine, err := conversation.NewDeferredSemanticConversationEngine(definition, callbacks, false)
	require.NoError(t, err)
	callbacks.SetNPCName(engine.ExtractNPCName())
	dialog := &LinearTalkDialog{gameScene: scene, callbacks: callbacks, engine: engine}
	dialog.initializeResizeableVisualElements()
	dialog.startConversation()
	return dialog
}

func boundaryTalkDefinition(body semantic.Body) *semantic.Conversation {
	return &semantic.Conversation{
		Schema: semantic.SchemaVersion, Protocol: semantic.ProtocolUltima5, ID: "redux.boundary_test", Origin: &semantic.ReduxOrigin{},
		Name:        semantic.Body{&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "Revealing speaker"}}}},
		Description: body,
		Greeting:    semantic.Body{}, Job: semantic.Body{}, Farewell: semantic.Body{},
		Topics: []semantic.Topic{}, Questions: map[string]semantic.Question{},
	}
}

func TestTalkDiscoveryStagesInitialOutputUntilMountAndCoalescesResponse(t *testing.T) {
	definition := boundaryTalkDefinition(semantic.Body{
		&semantic.NarrateAction{Content: semantic.RichText{&semantic.TextPart{Text: "Before"}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShadowlordFaulineiName},
		&semantic.PauseAction{},
		&semantic.SayAction{Content: semantic.RichText{&semantic.RunicPart{Text: "SUMM"}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShrineHonorMantra},
		&semantic.PauseAction{},
		&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "Faulinei embodies Falsehood."}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShadowlordFaulineiAssociation},
		&semantic.EndAction{},
	})
	dialog := newBoundaryTalkDialog(t, definition)
	scene, state := dialog.gameScene, dialog.gameScene.gameState
	require.Empty(t, state.KnowledgeSnapshot(), "construction has not mounted visible output")
	require.False(t, dialog.waitingForDismissal, "terminal hint cannot precede the timed tail")
	scene.dialogStack.PushModalDialog(dialog)
	dialog.mount()
	require.Len(t, state.KnowledgeSnapshot(), 1)
	require.True(t, scene.journalNotice.pending)
	scene.updateJournalNotice()
	require.Len(t, scene.journalNotice.queue.pending, 1)
	dialog.callbacks.SetNPCName("Changed after execution")
	dialog.Refresh()
	require.Len(t, state.KnowledgeSnapshot(), 1)
	dialog.finishCurrentPause()
	entry, learned := state.ShrineMantraKnowledge(5)
	require.True(t, learned)
	require.Equal(t, "Revealing speaker", entry.Source.DisplayName, "saved clues retain their original attribution label")
	require.False(t, scene.journalNotice.pending, "another fact in the same response does not restart its notice")
	require.False(t, dialog.waitingForDismissal)
	dialog.finishCurrentPause()
	require.True(t, dialog.waitingForDismissal)
	require.Empty(t, dialog.callbacks.pendingKnowledge)
	require.Equal(t, len(dialog.currentResponse.Output), dialog.presentedSourceBytes)
	require.Len(t, scene.journalNotice.queue.pending, 1)
	revision := scene.journalVisits.Revision
	dialog.commitPresentedKnowledge()
	require.Equal(t, revision, scene.journalVisits.Revision)
	dialog.Close()
	scene.updateJournalNotice()
	require.Equal(t, "Journal updated.\n[Shift-J] View journal", scene.output.JournalHistory())
	scene.updateJournalNotice()
	require.Equal(t, 1, strings.Count(scene.output.JournalHistory(), "Journal updated."))
}

func TestTalkUnseenTailDiscardedAfterForcedRemoval(t *testing.T) {
	dialog := newBoundaryTalkDialog(t, boundaryTalkDefinition(semantic.Body{
		&semantic.NarrateAction{Content: semantic.RichText{&semantic.TextPart{Text: "Before"}}},
		&semantic.PauseAction{},
		&semantic.SayAction{Content: semantic.RichText{&semantic.RunicPart{Text: "SUMM"}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShrineHonorMantra},
		&semantic.WaitForKeyAction{},
	}))
	scene := dialog.gameScene
	scene.dialogStack.PushModalDialog(dialog)
	dialog.mount()
	require.False(t, dialog.waitingForKeypressContinue)
	scene.dialogStack.MustRemoveWidgetInstance(dialog)
	dialog.Update()
	require.Empty(t, dialog.callbacks.pendingKnowledge)
	require.True(t, dialog.closed)
	require.Empty(t, scene.gameState.KnowledgeSnapshot())
	require.False(t, scene.journalNotice.pending)
	require.Panics(t, dialog.mount, "cancelled response cannot be remounted to reveal its tail")
}

func TestTalkPauseBeforeKeyWaitFinishesVisibleTailFirst(t *testing.T) {
	dialog := newBoundaryTalkDialog(t, boundaryTalkDefinition(semantic.Body{
		&semantic.PauseAction{},
		&semantic.SayAction{Content: semantic.RichText{&semantic.RunicPart{Text: "SUMM"}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShrineHonorMantra},
		&semantic.WaitForKeyAction{},
	}))
	scene := dialog.gameScene
	scene.dialogStack.PushModalDialog(dialog)
	dialog.mount()
	require.False(t, dialog.waitingForKeypressContinue)
	scene.gameState.ElapsedMs += 1500
	dialog.Update()
	require.False(t, dialog.isPausing)
	require.True(t, dialog.waitingForKeypressContinue)
	_, learned := scene.gameState.ShrineMantraKnowledge(5)
	require.True(t, learned)
	require.Contains(t, dialog.Output.GetOutputStr(true), "SUMM")
}

func TestTalkDiscoveryUnmountedUpdateCannotPresentTimedTail(t *testing.T) {
	dialog := newBoundaryTalkDialog(t, boundaryTalkDefinition(semantic.Body{
		&semantic.PauseAction{},
		&semantic.SayAction{Content: semantic.RichText{&semantic.RunicPart{Text: "SUMM"}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShrineHonorMantra},
	}))
	presented := dialog.presentedSourceBytes
	dialog.gameScene.gameState.ElapsedMs += 10_000
	dialog.Update()
	require.Equal(t, presented, dialog.presentedSourceBytes)
	require.NotEmpty(t, dialog.callbacks.pendingKnowledge)
	require.Panics(t, dialog.mount)
	dialog.Close()
	dialog.finishCurrentPause()
	require.Empty(t, dialog.pendingSegments)
	require.Empty(t, dialog.callbacks.pendingKnowledge)
	require.Empty(t, dialog.gameScene.gameState.KnowledgeSnapshot())
}

func TestTalkDiscoveryHistoryResizeAndDisposalCannotRevealTail(t *testing.T) {
	dialog := newBoundaryTalkDialog(t, boundaryTalkDefinition(semantic.Body{
		&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "A clue follows."}}},
		&semantic.PauseAction{},
		&semantic.SayAction{Content: semantic.RichText{&semantic.RunicPart{Text: "SUMM"}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShrineHonorMantra},
	}))
	scene := dialog.gameScene
	scene.dialogStack.PushModalDialog(dialog)
	dialog.mount()
	presented := dialog.presentedSourceBytes
	scene.keyboard.SimulateKeyPress(ebiten.KeyPageUp)
	dialog.Update()
	require.NotNil(t, dialog.historyPanel)
	scene.gameState.ElapsedMs += 10_000
	dialog.Refresh()
	dialog.Update()
	require.Equal(t, presented, dialog.presentedSourceBytes)
	require.Empty(t, scene.gameState.KnowledgeSnapshot())
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	dialog.Update()
	require.Nil(t, dialog.historyPanel)
	dialog.Update()
	require.Empty(t, scene.gameState.KnowledgeSnapshot(), "history time cannot complete the deferred pause")
	scene.Dispose()
	dialog.Update()
	dialog.finishCurrentPause()
	require.True(t, dialog.closed)
	require.Empty(t, dialog.callbacks.pendingKnowledge)
	require.Empty(t, scene.gameState.KnowledgeSnapshot())
	require.False(t, scene.journalNotice.pending)
}

func TestTalkDiscoveryFirstChangedFactNotFirstActionControlsResponseCue(t *testing.T) {
	body := semantic.Body{
		&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "Faulinei"}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShadowlordFaulineiName},
		&semantic.PauseAction{},
		&semantic.SayAction{Content: semantic.RichText{&semantic.RunicPart{Text: "SUMM"}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShrineHonorMantra},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShadowlordFaulineiAssociation},
	}
	definition := boundaryTalkDefinition(semantic.Body{})
	definition.Job = body
	dialog := newBoundaryTalkDialog(t, definition)
	scene := dialog.gameScene
	scene.dialogStack.PushModalDialog(dialog)
	dialog.mount()
	require.NoError(t, dialog.callbacks.DiscoverKnowledge(semantic.KnowledgeShadowlordFaulineiName))
	scene.updateJournalNotice()
	for iteration := range 2 {
		revision := scene.journalVisits.Revision
		dialog.currentResponse = dialog.engine.ProcessInput("JOB")
		dialog.processCurrentResponse()
		require.False(t, scene.journalNotice.pending, "repeated first fact is not a cue")
		dialog.finishCurrentPause()
		if iteration == 0 {
			require.Greater(t, scene.journalVisits.Revision, revision)
			require.True(t, scene.journalNotice.pending)
		} else {
			require.Equal(t, revision, scene.journalVisits.Revision)
			require.False(t, scene.journalNotice.pending, "repeated response is not a new discovery")
		}
		scene.updateJournalNotice()
	}
}

func TestTalkDiscoveryByteCursorTracksUnicodeAndCompactedRunicSegments(t *testing.T) {
	dialog := newBoundaryTalkDialog(t, boundaryTalkDefinition(semantic.Body{
		&semantic.NarrateAction{Content: semantic.RichText{&semantic.TextPart{Text: "\u00e9 clue"}}},
		&semantic.PauseAction{},
		&semantic.SayAction{Content: semantic.RichText{&semantic.RunicPart{Text: "SU"}, &semantic.RunicPart{Text: "MM"}}},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShrineHonorMantra},
		&semantic.PauseAction{},
		&semantic.DiscoverKnowledgeAction{Knowledge: semantic.KnowledgeShadowlordFaulineiName},
		&semantic.PauseAction{},
		&semantic.SayAction{Content: semantic.RichText{&semantic.TextPart{Text: "end"}}},
	}))
	scene := dialog.gameScene
	scene.dialogStack.PushModalDialog(dialog)
	dialog.mount()
	require.Empty(t, scene.gameState.KnowledgeSnapshot())
	dialog.finishCurrentPause()
	_, learned := scene.gameState.ShrineMantraKnowledge(5)
	require.True(t, learned)
	_, learned = scene.gameState.ShadowlordKnowledge(references.Faulinei, game_state.ShadowlordKnowledgeName)
	require.False(t, learned)
	dialog.finishCurrentPause()
	_, learned = scene.gameState.ShadowlordKnowledge(references.Faulinei, game_state.ShadowlordKnowledgeName)
	require.True(t, learned, "a boundary between adjacent controls commits without waiting for later text")
	dialog.finishCurrentPause()
	require.Equal(t, len(dialog.currentResponse.Output), dialog.presentedSourceBytes)
	require.Empty(t, dialog.callbacks.pendingKnowledge)
}

func TestOriginalGrumanDeferredRunicDiscovery(t *testing.T) {
	catalog, err := conversationcatalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	program, found := catalog.Original(semantic.OriginalSourceTowne, 34)
	require.True(t, found)
	definition, err := program.Definition()
	require.NoError(t, err)
	dialog := newBoundaryTalkDialog(t, definition)
	scene := dialog.gameScene
	scene.gameState.PartyState.Karma.Value = 50
	scene.dialogStack.PushModalDialog(dialog)
	dialog.mount()
	for _, value := range []string{"n", "mant", "y"} {
		dialog.currentResponse = dialog.engine.ProcessInput(value)
		dialog.processCurrentResponse()
	}
	require.NoError(t, dialog.currentResponse.Error)
	require.Contains(t, dialog.currentResponse.Output, "SUMM")
	require.True(t, dialog.isPausing)
	_, learned := scene.gameState.ShrineMantraKnowledge(5)
	require.False(t, learned)
	require.False(t, scene.journalNotice.pending)
	for steps := 0; dialog.isPausing && steps < 10; steps++ {
		dialog.finishCurrentPause()
	}
	_, learned = scene.gameState.ShrineMantraKnowledge(5)
	require.True(t, learned)
	require.True(t, scene.journalNotice.pending)
}

func TestOriginalSutekDiscoveryRequiresCompleteMethodParagraph(t *testing.T) {
	catalog, err := conversationcatalog.LoadOriginal(originaldata.Directory(t), conversationpatchdata.Files)
	require.NoError(t, err)
	program, found := catalog.Original(semantic.OriginalSourceDwelling, 10)
	require.True(t, found)
	definition, err := program.Definition()
	require.NoError(t, err)
	dialog := newBoundaryTalkDialog(t, definition)
	scene := dialog.gameScene
	scene.dialogStack.PushModalDialog(dialog)
	dialog.mount()
	for _, value := range []string{scene.gameState.PartyState.AvatarName(), "y"} {
		dialog.currentResponse = dialog.engine.ProcessInput(value)
		dialog.processCurrentResponse()
	}
	for steps := 0; !strings.Contains(dialog.currentResponse.Output, "Principle it opposes") && steps < 12; steps++ {
		require.Empty(t, scene.gameState.QuestJournalEvents(), "partial instructions do not reveal the complete method")
		require.False(t, scene.journalNotice.pending)
		require.True(t, dialog.waitingForKeypressContinue)
		dialog.waitingForKeypressContinue = false
		dialog.currentResponse = dialog.engine.ProcessInput("")
		dialog.processCurrentResponse()
	}
	require.Contains(t, strings.Join(strings.Fields(dialog.Output.GetOutputStr(true)), " "), "Principle it opposes")
	require.Len(t, scene.gameState.QuestJournalEvents(), 1)
	require.True(t, scene.journalNotice.pending)
}
