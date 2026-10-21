package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestTimedOverlayReplacementRemovesExactPreviousInstance(t *testing.T) {
	scene := newTimedOverlayTestScene()
	scene.ShowTimedOverlayMessage("first", 1000, true, false)
	first := requireTimedOverlay(t, scene)
	cover := newTimedOverlayTestCover()
	cover.ShowWithoutAnimation()
	scene.dialogStack.PushModalDialog(cover)

	scene.gameState.ElapsedMs = 250
	scene.ShowTimedOverlayMessage("second", 2000, true, false)
	second := requireTimedOverlay(t, scene)

	assert.NotSame(t, first, second)
	assert.False(t, first.IsVisible())
	assert.NotContains(t, scene.dialogStack.Dialogs(), first)
	assert.Contains(t, scene.dialogStack.Dialogs(), cover)
	assert.Contains(t, scene.dialogStack.Dialogs(), second)
	assert.Equal(t, int64(2250), scene.timedOverlayDismissMs)
}

func TestTimedOverlayExpiresWhileCoveredWithoutRemovingCover(t *testing.T) {
	scene := newTimedOverlayTestScene()
	scene.ShowTimedOverlayMessage("timed", 1000, true, false)
	overlay := requireTimedOverlay(t, scene)
	cover := newTimedOverlayTestCover()
	cover.ShowWithoutAnimation()
	scene.dialogStack.PushModalDialog(cover)

	for range 100 {
		overlay.Update()
	}
	assert.True(t, overlay.IsVisible(), "externally managed duration must not advance through widget updates")

	scene.gameState.ElapsedMs = scene.timedOverlayDismissMs
	scene.updateTimedOverlay()

	assert.Nil(t, scene.timedOverlayDialog)
	assert.Zero(t, scene.timedOverlayDismissMs)
	assert.False(t, overlay.IsVisible())
	assert.Equal(t, 1, scene.dialogStack.Count())
	assert.Same(t, cover, scene.dialogStack.PeekTopModalDialog())
}

func TestTimedOverlayExpiresOnlyAtSceneDeadline(t *testing.T) {
	scene := newTimedOverlayTestScene()
	scene.ShowTimedOverlayMessage("timed", 1000, true, false)
	overlay := requireTimedOverlay(t, scene)

	scene.gameState.ElapsedMs = scene.timedOverlayDismissMs - 1
	scene.updateTimedOverlay()
	assert.Same(t, overlay, scene.timedOverlayDialog)
	assert.True(t, scene.dialogStack.HasModalDialog())

	scene.gameState.ElapsedMs++
	scene.updateTimedOverlay()
	assert.Nil(t, scene.timedOverlayDialog)
	assert.False(t, scene.dialogStack.HasModalDialog())
}

func TestTimedOverlaySceneReplacementRemovesExactOwnedInstance(t *testing.T) {
	scene := newTimedOverlayTestScene()
	scene.ShowTimedOverlayMessage("timed", 1000, true, false)
	overlay := requireTimedOverlay(t, scene)
	cover := newTimedOverlayTestCover()
	cover.ShowWithoutAnimation()
	scene.dialogStack.PushModalDialog(cover)

	scene.PrepareForSceneReplacement()

	assert.Nil(t, scene.timedOverlayDialog)
	assert.False(t, overlay.IsVisible())
	assert.Equal(t, []widgets.Widget{cover}, scene.dialogStack.Dialogs())
}

func TestTimedOverlayDisposalRemovesExactOwnedInstance(t *testing.T) {
	scene := newTimedOverlayTestScene()
	scene.gameState.EventQueueManager = event_queue.NewEventQueueManager(func() int64 {
		return scene.gameState.ElapsedMs
	}, nil)
	scene.ShowTimedOverlayMessage("timed", 1000, true, false)
	overlay := requireTimedOverlay(t, scene)
	cover := newTimedOverlayTestCover()
	cover.ShowWithoutAnimation()
	scene.dialogStack.PushModalDialog(cover)

	scene.Dispose()

	assert.Nil(t, scene.timedOverlayDialog)
	assert.False(t, overlay.IsVisible())
	assert.Equal(t, []widgets.Widget{cover}, scene.dialogStack.Dialogs())
	assert.True(t, scene.disposed)
}

func newTimedOverlayTestScene() *GameScene {
	return &GameScene{
		gameState: &game_state.GameState{ElapsedMs: 100},
	}
}

func newTimedOverlayTestCover() *widgets.OverlayDialog {
	return widgets.NewOverlayDialog("cover").
		AsConfirmation().
		WithUserDismissed(input.NewKeyboardWithKeyStateSource(0, noSceneTestKeyHeld)).
		Build()
}

func requireTimedOverlay(t *testing.T, scene *GameScene) *widgets.OverlayDialog {
	t.Helper()
	overlay, ok := scene.timedOverlayDialog.(*widgets.OverlayDialog)
	require.True(t, ok)
	require.True(t, overlay.IsDurationExternallyManaged())

	return overlay
}
