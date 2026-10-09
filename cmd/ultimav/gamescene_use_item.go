package main

import "github.com/bradhannah/UltimaVRedux/internal/event_queue"

// ShowPocketWatchTime displays the current time in a dismissible message dialog.
// Called from GameState when player uses pocket watch from Use menu.
func (g *GameScene) ShowPocketWatchTime(timeString string, complete func()) {
	g.gameState.EventQueueManager.Workflow("pocket_watch").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			g.output.AddText("Use-")
		}).ThenSequence(

		g.thenShowConfirmationDialog("The time is " + timeString)).Finally(

		func() {
			g.keyboard.ResetReleaseDebouncing()
			if complete != nil {
				complete()
			}
		}).
		Start()
}
