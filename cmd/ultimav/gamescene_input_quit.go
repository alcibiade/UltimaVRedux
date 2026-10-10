package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func (g *GameScene) startSurfaceQuitInput() {
	g.showSaveQuestion()
	g.secondaryKeyState = SurfaceQuitInput
	g.keyboard.SetAllowKeyPressImmediately()
}

func (g *GameScene) showSaveQuestion() {
	if g.saveQuestionDialog != nil {
		panic("save question already active")
	}
	dialog := widgets.NewOverlayDialog("Save game?").AsConfirmation().
		WithPosition(widgets.PositionMapCenterLower).WithSizing(widgets.SizingCompact).
		WithExternallyManagedDuration(0).Build()
	dialog.AppendLineBuilder().Text("[Y]", widgets.DialogControlKeyColor).
		Text(" Yes   ", widgets.DialogControlHintColor).Text("[N/Esc]", widgets.DialogControlKeyColor).
		Text(" No", widgets.DialogControlHintColor).Done()
	dialog.ShowWithoutAnimation()
	g.saveQuestionDialog = dialog
	// The retained secondary command owns input; this widget only presents it.
	g.dialogStack.PushNonBlockingDialog(dialog)
}

func (g *GameScene) clearSaveQuestion() {
	if g.saveQuestionDialog == nil {
		return
	}
	g.dialogStack.MustRemoveWidgetInstance(g.saveQuestionDialog)
	g.saveQuestionDialog = nil
	g.keyboard.PrepareForTransition(0)
}

func (g *GameScene) surfaceQuitSecondaryInput(game *Game) {
	key := g.keyboard.GetBoundKeyPressed(&boundKeysGame)
	if key == nil || !g.keyboard.TryToRegisterKeyPress(*key) {
		return
	}
	if *key != ebiten.KeyY && *key != ebiten.KeyN && *key != ebiten.KeyEscape {
		return
	}

	g.secondaryKeyState = PrimaryInput
	g.clearSaveQuestion()
	if *key != ebiten.KeyY {
		g.gameState.SystemCallbacks.Message.AddRowStr("Save game? No")

		return
	}

	g.gameState.SystemCallbacks.Message.AddRowStr("Save game? Yes")
	g.saveCurrentPlayer(game, "Surface")
}

func (g *GameScene) saveCurrentPlayer(game *Game, context string) {
	g.gameState.SystemCallbacks.Message.AddRowStr("Saving...")
	if game == nil {
		g.showQuitSaveResult("Save failed: " + ErrPlayerSaveUnavailable.Error())

		return
	}
	record, err := game.quickSavePlayerSave()
	if err != nil {
		logging.Warn(logging.CategoryGameState, "%s save failed: %v", context, err)
		g.showQuitSaveResult("Save failed: " + playerSaveErrorDetail(err))

		return
	}
	g.showQuitSaveResult("Game saved to " + playerSaveSlotName(record.Slot) + ".")
}

func (g *GameScene) showQuitSaveResult(message string) {
	g.gameState.SystemCallbacks.Message.AddRowStr(message)
	g.keyboard.PrepareForTransition(0)
	dialog := widgets.NewOverlayDialog(message).
		AsConfirmation().
		WithSizing(widgets.SizingAutoFitDefault).
		WithUserDismissed(g.keyboard).
		AutoPopFromStack(&g.dialogStack).
		OnDismiss(func() { g.keyboard.PrepareForTransition(0) }).
		Build()
	dialog.ShowWithoutAnimation()
	g.dialogStack.PushModalDialog(dialog)
}
