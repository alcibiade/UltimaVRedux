package main

import (
	"testing"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestTransientInputInitializationPreservesIndependentKeyboardClock(t *testing.T) {
	inputMs := int64(1000)
	keyboard := input.NewKeyboardWithKeyStateSource(keyPressDelay, func(ebiten.Key) bool { return false })
	keyboard.SetElapsedMillisecondsSource(func() int64 { return inputMs })
	scene := &GameScene{gameState: &game_state.GameState{ElapsedMs: 321}, keyboard: keyboard}
	scene.initializeTransientCommandInput()
	keyboard.SetLastKeyPressedNowPlusMs(0)
	inputMs += keyPressDelay + 1
	require.Equal(t, keyPressDelay+1, keyboard.GetMsSinceLastKeyPress(),
		"production initialization must not replace the adapter clock with paused game elapsed time")
	require.EqualValues(t, 321, scene.getElapsedMs())
}

func TestJournalFreshTabPressesUseInputClockWhileGameplayRemainsPaused(t *testing.T) {
	scene, held, inputMs, update := newReaderInputClockScene(t)
	state := scene.gameState
	held[ebiten.KeyShiftLeft], held[ebiten.KeyJ] = true, true
	update()
	require.NotNil(t, scene.journalSession)
	dialog := scene.journalSession.dialog
	require.Equal(t, widgets.QuestJournalChronicleTab, dialog.GetActiveTab())
	clear(held)
	update()
	date, elapsed, calls := state.DateTime, state.ElapsedMs, state.GetRNG().Calls()
	fired := 0
	state.EventQueueManager.Workflow("reader_clock_pause").Priority(event_queue.PriorityNormal).
		Sequential().First(func() { fired++ }).Start()
	for _, expected := range []widgets.QuestJournalTab{widgets.QuestJournalVirtuesTab, widgets.QuestJournalShadowlordsTab} {
		*inputMs += keyPressDelay + 1
		held[ebiten.KeyTab] = true
		update()
		require.Equal(t, expected, dialog.GetActiveTab(), "each fresh Tab advances even though game elapsed time is frozen")
		*inputMs += keyPressDelay + 1
		update()
		require.Equal(t, expected, dialog.GetActiveTab(), "holding Tab still does not repeat")
		delete(held, ebiten.KeyTab)
		update()
		require.Equal(t, elapsed, scene.getElapsedMs())
		require.Equal(t, elapsed, state.ElapsedMs)
		require.Equal(t, date, state.DateTime)
		require.Equal(t, calls, state.GetRNG().Calls())
		require.Zero(t, fired, "independent input timing must not unpause gameplay queues")
	}
}

func TestReaderFreshOpeningChordClosesWhileGameplayClockIsFrozen(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyJ, ebiten.KeyM} {
		t.Run(key.String(), func(t *testing.T) {
			scene, held, inputMs, update := newReaderInputClockScene(t)
			state := scene.gameState
			held[ebiten.KeyShiftLeft], held[key] = true, true
			update()
			if key == ebiten.KeyJ {
				require.NotNil(t, scene.journalSession)
			} else {
				require.NotNil(t, scene.atlasSession)
			}
			date, elapsed, calls := state.DateTime, state.ElapsedMs, state.GetRNG().Calls()
			clear(held)
			update()
			// No intervening key changes the keyboard's last accepted key. The
			// same closing chord must use elapsed input time, not frozen game time.
			*inputMs += keyPressDelay + 1
			held[ebiten.KeyShiftLeft], held[key] = true, true
			update()
			if key == ebiten.KeyJ {
				require.True(t, scene.journalSession.closing)
			} else {
				require.True(t, scene.atlasSession.closing)
			}
			require.Zero(t, scene.dialogStack.Count())
			*inputMs += 1000
			update()
			require.Equal(t, elapsed, scene.getElapsedMs(), "held closing chord still owns the reader pause")
			clear(held)
			update()
			require.Nil(t, scene.journalSession)
			require.Nil(t, scene.atlasSession)
			require.Equal(t, elapsed, scene.getElapsedMs(), "reader release rebases without catching up paused time")
			require.Equal(t, elapsed, state.ElapsedMs)
			require.Equal(t, date, state.DateTime)
			require.Equal(t, calls, state.GetRNG().Calls())
			*inputMs += 5
			require.Equal(t, elapsed+5, scene.getElapsedMs())
		})
	}
}

func newReaderInputClockScene(t *testing.T) (*GameScene, map[ebiten.Key]bool, *int64, func()) {
	t.Helper()
	scene, held, _, _ := newRunicReferenceTestScene(t, references.BritanniaUnderworld)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	inputMs := int64(1000)
	scene.keyboard = input.NewKeyboardWithKeyStateSource(keyPressDelay, func(key ebiten.Key) bool { return held[key] })
	scene.keyboard.SetElapsedMillisecondsSource(func() int64 { return inputMs })
	// Exercise production initialization after injection: an accidental override
	// would silently bind these real reader controls back to the paused clock.
	scene.initializeTransientCommandInput()
	scene.transientCommandInput.focused = func() bool { return true }
	scene.gameState.ElapsedMs = 321
	scene.elapsedBaseMs = scene.gameState.ElapsedMs
	origin := time.Unix(100, 0)
	scene.now = func() time.Time { return origin.Add(time.Duration(inputMs) * time.Millisecond) }
	scene.startTime = scene.now()
	scene.gameState.SystemCallbacks.UI.ShowQuestJournalDossier = scene.ShowQuestJournalDossier
	update := func() {
		t.Helper()
		require.NoError(t, scene.Update(nil))
	}
	t.Cleanup(func() {
		clear(held)
		scene.closeJournalSession(scene.journalSession, false)
		scene.closeAtlasSession(true)
		scene.gameState.EventQueueManager.Shutdown()
	})

	return scene, held, &inputMs, update
}
