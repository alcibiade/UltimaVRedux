package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameScene) startSceptreWorkflow(complete func()) {
	origin := g.gameState.MapState.PlayerLocation
	quantity := g.gameState.PartyState.Inventory.QuestItems.Get(references.Sceptre)
	committed := false
	g.gameState.EventQueueManager.Workflow("sceptre_workflow").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			g.gameState.SystemCallbacks.Message.AddRowStr("Sceptre")
			g.gameState.SystemCallbacks.Message.AddRowStr("Wielding the Sceptre of Lord British...")
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		}).Then(func() {
		if g.gameState.CombatScene != nil || g.gameState.MapState.PlayerLocation != origin ||
			quantity == 0 || g.gameState.PartyState.Inventory.QuestItems.Get(references.Sceptre) != quantity {
			g.gameState.SystemCallbacks.UI.ShowActionFeedback(staleUseSelectionMessage, true)

			return
		}
		g.gameState.UseExplorationSceptre()
		committed = true
	}).Then(func() {
		if complete == nil && committed {
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

func (g *GameScene) thenUseCombatSceptre(canCommit func() bool, didCommit *bool) event_queue.Sequence {
	var cleared []references.Position
	var timing combatPresentationTiming
	valid := false

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			if !canCommit() {
				return
			}
			timing = g.captureCombatTiming()
			defer g.combatTimingScope(timing)()
			g.showCombatSceptreActorEffect()
			g.gameState.SystemCallbacks.Message.AddRowStr("Sceptre")
			g.gameState.SystemCallbacks.Message.AddRowStr("Wielding the Sceptre of Lord British...")
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
		}).Then(func() {
			defer g.combatTimingScope(timing)()
			valid = canCommit()
			if !valid {
				return
			}
			cleared = g.gameState.ClearNearbySceptreBarriers()
			*didCommit = len(cleared) > 0
		}).If(
			func() bool { return valid && len(cleared) == 0 },
			func(fieldQ *event_queue.SequentialEventBuilder) {
				fieldQ.ThenSequence(g.thenShowCastUseDirectionPrompt(
					"Direction-",
					func(direction references.Direction) {
						defer g.combatTimingScope(timing)()
						if canCommit() {
							sequence := g.gameState.CastUseCommitSequence()
							g.gameState.UseCombatSceptre(direction)
							// A confirmed direction with no field is still a paid attempt.
							*didCommit = g.gameState.CastUseCommitSequence() != sequence
						}
					},
					func() {},
				))
			},
		)
	}
}
