package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
)

// showSpellExpiredPopup displays a dismissible popup when a duration effect expires.
// Shows message like "Light wore off!" with any-key dismissal.
// Based on original U5 behavior where effects silently expired, but enhanced for UX.
func (g *GameScene) showSpellExpiredPopup(spellName string) {
	message := spellName + " wore off!"
	workflowName := "spell_expired_" + spellName
	if g.gameState.EventQueueManager.HasWorkflow(workflowName) {
		return
	}

	g.gameState.EventQueueManager.Workflow(workflowName).
		Priority(event_queue.PriorityNormal).
		Sequential().
		First(func() {

		}).ThenSequence(

		g.thenShowConfirmationDialog(message)).
		Start()
}
