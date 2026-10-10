package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// StartSceptreWorkflow initiates surface/town Sceptre use.
// Sceptre context and field-removal rules are documented in ref:QUEST-0003.
func (g *GameScene) StartSceptreWorkflow() {
	g.startSceptreWorkflow(nil)
}

func (g *GameScene) startSceptreWorkflow(complete func()) {
	g.gameState.EventQueueManager.Workflow("sceptre_workflow").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			g.gameState.SystemCallbacks.Message.AddRowStr("Sceptre")
			g.gameState.SystemCallbacks.Message.AddRowStr("Wielding the Sceptre of Lord British...")
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		}).Then(func() {
		g.gameState.UseExplorationSceptre()
	}).Then(func() {
		if complete == nil {
			g.gameState.FinishTurn()
		}
	}).Finally(func() {
		g.keyboard.ResetReleaseDebouncing()
		if complete != nil {
			complete()
		}
	}).
		Start()
}

func (g *GameScene) thenUseCombatSceptre() event_queue.Sequence {
	var cleared []references.Position

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			g.showCombatSceptreActorEffect()
			g.gameState.SystemCallbacks.Message.AddRowStr("Sceptre")
			g.gameState.SystemCallbacks.Message.AddRowStr("Wielding the Sceptre of Lord British...")
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		}).Then(func() {
			cleared = g.gameState.ClearNearbySceptreBarriers()
		}).If(
			func() bool { return len(cleared) == 0 },
			func(fieldQ *event_queue.SequentialEventBuilder) {
				fieldQ.ThenSequence(g.thenShowCastUseDirectionPrompt(
					"Direction-",
					func(direction references.Direction) {
						g.gameState.UseCombatSceptre(direction)
					},
					func() {},
				))
			},
		)
	}
}
