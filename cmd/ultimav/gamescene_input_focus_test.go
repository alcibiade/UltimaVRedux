package main

import (
	"testing"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestUnfocusedUpdateSynchronizesSkyAndAdvancesQueuesWithoutPass(t *testing.T) {
	scene, held, _, _ := newRunicReferenceTestScene(t, references.BritanniaUnderworld)
	state := scene.gameState
	scene.transientCommandInput.focused = func() bool { return false }
	// Retain the fixture's stabilized astronomical state: changing the hour here
	// would enqueue a higher-priority moongate transition ahead of this workflow.
	beforeDate, beforeRNG := state.DateTime, state.GetRNG().Calls()
	presented := 0
	state.EventQueueManager.Workflow("focus_presentation").Priority(event_queue.PriorityNormal).
		Sequential().First(func() {}).Wait(50).Then(func() { presented++ }).Start()
	t.Cleanup(state.EventQueueManager.Shutdown)
	held[ebiten.KeySpace] = true
	scene.keyboard.SimulateKeyPress(ebiten.KeyG)
	for range 3 {
		state.ElapsedMs += 55
		require.NoError(t, scene.Update(nil))
	}
	require.Equal(t, projectSky(beforeDate), scene.sky.strip, "sky must synchronize even before any Draw")
	require.Equal(t, 1, presented)
	require.Greater(t, state.GetRNG().Calls(), beforeRNG, "the normal shared-wind timeline continues")
	require.Equal(t, beforeDate, state.DateTime, "unfocused Space cannot Pass")
	require.Equal(t, PrimaryInput, scene.secondaryKeyState)
	require.False(t, scene.transientCommandInput.pass.active)
}

func TestUnfocusedModalFadeContinuesWithoutHiddenAcknowledgement(t *testing.T) {
	scene, held, _, _ := newRunicReferenceTestScene(t, references.BritanniaUnderworld)
	focused := false
	scene.transientCommandInput.focused = func() bool { return focused }
	dismissed := 0
	dialog := widgets.NewMessageDialog("Focus-safe presentation").WithEffects(true).
		WithFadeDurations(100, 100).WithInputDismissal(scene.keyboard).
		WithDismissCallback(func() { dismissed++ }).Build()
	dialog.UseAnimationClock(scene.getElapsedMs)
	scene.dialogStack.PushModalDialog(dialog)
	dialog.Show()
	startedAt := scene.getElapsedMs()
	held[ebiten.KeyNumpadEnter] = true
	scene.gameState.ElapsedMs = startedAt + 50
	require.NoError(t, scene.Update(nil))
	require.InDelta(t, 0.5, dialog.DissolveLevel(), 0.000001)
	require.Zero(t, dismissed)
	scene.gameState.ElapsedMs = startedAt + 100
	require.NoError(t, scene.Update(nil))
	require.Zero(t, dialog.DissolveLevel())
	focused = true
	require.NoError(t, scene.Update(nil)) // Focus-return frame still rejects input.
	require.NoError(t, scene.Update(nil))
	require.Zero(t, dismissed, "held keypad Enter cannot dismiss on focus return")
	clear(held)
	require.NoError(t, scene.Update(nil))
	held[ebiten.KeyNumpadEnter] = true
	require.NoError(t, scene.Update(nil))
	require.Equal(t, 1, dismissed)
	scene.dialogStack.RemoveWidgetInstance(dialog)
}

func TestUnfocusedUpdateDoesNotCallInputOwningModal(t *testing.T) {
	scene, _, _, _ := newRunicReferenceTestScene(t, references.BritanniaUnderworld)
	focused := false
	scene.transientCommandInput.focused = func() bool { return focused }
	dialog := &countingDialog{}
	scene.dialogStack.PushModalDialog(dialog)
	require.NoError(t, scene.Update(nil))
	require.Zero(t, dialog.updates)
	focused = true
	require.NoError(t, scene.Update(nil))
	require.Zero(t, dialog.updates, "focus-return input suppression also covers private modal adapters")
	require.NoError(t, scene.Update(nil))
	require.Equal(t, 1, dialog.updates)
}

type nativeSuspensionTestDialog struct {
	countingDialog
	suspensions int
}

func (d *nativeSuspensionTestDialog) SuspendTextInput() { d.suspensions++ }

func TestUnfocusedAtlasSuspendsNativeChildBeforeReaderReturn(t *testing.T) {
	scene, _, _, _ := newRunicReferenceTestScene(t, references.BritanniaUnderworld)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.openAtlas(nil)
	t.Cleanup(func() { scene.closeAtlasSession(true) })
	dialog := &nativeSuspensionTestDialog{}
	scene.dialogStack.PushModalDialog(dialog)
	scene.transientCommandInput.focused = func() bool { return false }
	elapsed, date := scene.gameState.ElapsedMs, scene.gameState.DateTime
	require.NoError(t, scene.Update(nil))
	require.Equal(t, 1, dialog.suspensions)
	require.Zero(t, dialog.updates, "native teardown must not poll hidden input")
	require.Equal(t, elapsed, scene.gameState.ElapsedMs)
	require.Equal(t, date, scene.gameState.DateTime)
	scene.dialogStack.RemoveWidgetInstance(dialog)
}

func TestUnfocusedRawWaitRequiresReleaseAfterFocusReturns(t *testing.T) {
	scene, held, update, _ := newRunicReferenceTestScene(t, references.BritanniaUnderworld)
	state := scene.gameState
	focused := true
	scene.transientCommandInput.focused = func() bool { return focused }
	state.SystemCallbacks.Visual.IsAnyKeyPressed = scene.IsAnyKeyPressed
	acknowledged := 0
	state.EventQueueManager.Workflow("focus_acknowledgement").Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().First(func() {}).WaitForAnyKey().Then(func() { acknowledged++ }).Start()
	t.Cleanup(state.EventQueueManager.Shutdown)
	update()
	update() // Arm the ordinary wait before losing focus.
	focused = false
	held[ebiten.KeySpace] = true
	update()
	update()
	require.Zero(t, acknowledged)
	focused = true
	update()
	update()
	require.Zero(t, acknowledged)
	clear(held)
	update()
	held[ebiten.KeyEnter] = true
	update()
	require.Equal(t, 1, acknowledged)
	clear(held)
	update()
	require.False(t, scene.inputBlocked)
}

func TestUnfocusedUpdatePreservesAtlasAndJournalPause(t *testing.T) {
	for _, reader := range []string{"atlas", "journal"} {
		t.Run(reader, func(t *testing.T) {
			scene, held, _, _ := newRunicReferenceTestScene(t, references.BritanniaUnderworld)
			scene.spriteSheet = sharedPresentationTestSpriteSheet()
			state := scene.gameState
			now := time.Unix(100, 0)
			scene.now, scene.startTime = func() time.Time { return now }, now
			if reader == "atlas" {
				scene.openAtlas(nil)
				require.NotNil(t, scene.atlasSession)
				t.Cleanup(func() { scene.closeAtlasSession(true) })
			} else {
				scene.ShowQuestJournalDossier(state.CurrentQuestJournalDossier())
				require.NotNil(t, scene.journalSession)
				t.Cleanup(func() { scene.closeJournalSession(scene.journalSession, false) })
			}
			date, elapsed, calls := state.DateTime, state.ElapsedMs, state.GetRNG().Calls()
			modal := scene.dialogStack.PeekTopModalDialog()
			fired := 0
			state.EventQueueManager.Workflow("focus_reader_pause").Priority(event_queue.PriorityNormal).
				Sequential().First(func() { fired++ }).Start()
			t.Cleanup(state.EventQueueManager.Shutdown)
			scene.transientCommandInput.focused = func() bool { return false }
			held[ebiten.KeyEscape] = true
			now = now.Add(time.Hour)
			require.NoError(t, scene.Update(nil))
			require.Same(t, modal, scene.dialogStack.PeekTopModalDialog())
			require.Zero(t, fired)
			require.Equal(t, date, state.DateTime)
			require.Equal(t, elapsed, state.ElapsedMs)
			require.Equal(t, elapsed, scene.getElapsedMs())
			require.Equal(t, calls, state.GetRNG().Calls())
		})
	}
}
