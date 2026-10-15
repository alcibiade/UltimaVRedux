package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

// ShowPocketWatchTime displays the current time in a dismissible message dialog.
// Called from GameState when player uses pocket watch from Use menu.
func (g *GameScene) ShowPocketWatchTime(timeString string, complete func()) {
	g.gameState.EventQueueManager.Workflow("pocket_watch").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			g.output.AddText("Use-")
		}).ThenSequence(g.thenShowPocketWatchTime(timeString, func() bool { return true })).
		Finally(func() {
			g.clearPocketWatchDialog()
			g.keyboard.ResetReleaseDebouncing()
			if complete != nil {
				complete()
			}
		}).
		Start()
}

// thenShowPocketWatchTime keeps the acknowledgement inside its caller's workflow.
// The caller's Finally owns cleanup, including shutdown before dismissal.
func (g *GameScene) thenShowPocketWatchTime(timeString string, isCurrent func() bool) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		var dialog *widgets.OverlayDialog

		return q.Then(func() {
			if !isCurrent() {
				return
			}
			dialog = widgets.NewOverlayDialog("The time is " + timeString).
				AsConfirmation().
				WithUserDismissed(g.keyboard).
				AutoPopFromStack(&g.dialogStack).
				Build()
			g.pocketWatchDialog = dialog
			dialog.ShowWithoutAnimation()
			g.dialogStack.PushModalDialog(dialog)
		}).While(
			func() bool { return dialog != nil && !dialog.IsDismissed() && isCurrent() },
			func() {},
			DialogPollingRateMs,
		).Then(func() {
			g.clearPocketWatchDialog()
			g.keyboard.ResetReleaseDebouncing()
		})
	}
}

func (g *GameScene) clearPocketWatchDialog() {
	if g.pocketWatchDialog == nil {
		return
	}
	g.pocketWatchDialog.Dismiss()
	g.dialogStack.RemoveWidgetInstance(g.pocketWatchDialog)
	g.pocketWatchDialog = nil
}

func (g *GameScene) thenUseCombatPocketWatch(canCommit func() bool, didCommit *bool) event_queue.Sequence {
	owner := g.gameState.CaptureCombatUseOwner()
	commit := g.gameState.BeginCombatPocketWatchUse(owner)
	var timeString string
	committed := false

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			if !canCommit() {
				return
			}
			timeString, committed = g.gameState.CommitCombatPocketWatchUse(commit)
			*didCommit = committed
		}).If(func() bool { return committed }, func(displayQ *event_queue.SequentialEventBuilder) {
			displayQ.ThenSequence(g.thenShowPocketWatchTime(timeString, func() bool {
				return g.gameState.IsCurrentCombatUseOwner(owner)
			}))
		})
	}
}
