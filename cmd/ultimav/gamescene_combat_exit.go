package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

const combatVictoryExitDelayMs int64 = 180

func (g *GameScene) startCombatVictoryExitWorkflow() {
	timing := g.captureCombatTiming()
	combat := g.gameState.CombatScene
	members := g.gameState.ActiveCombatPartyMembers()
	if len(members) == 0 {
		g.gameState.ActionEscapeCombatMap()

		return
	}
	queue := g.gameState.EventQueueManager.Workflow("victory_exit_workflow").
		Priority(event_queue.PriorityHigh).
		Sequential().BlockInput().
		First(func() {})
	for _, memberIndex := range members {
		index := memberIndex
		queue.Then(func() {
			if g.gameState.CombatScene == combat {
				g.gameState.DepartCombatPartyMemberAfterVictory(index)
			}
		}).Wait(timing.durationMs(combatVictoryExitDelayMs))
	}
	queue.Finally(func() {
		if g.gameState.CombatScene != nil && g.gameState.CombatScene == combat {
			g.gameState.EndCombat(game_state.CombatVictory)
		}
	}).
		Start()
}
