package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/mainscreen"
)

func combatConditions(combat *game_state.CombatState, identity game_state.CombatTurnIdentity) mainscreen.CombatConditions {
	if combat == nil || !identity.IsPartyMember || !combat.IsCombatPCActive(identity) {
		return mainscreen.CombatConditions{}
	}

	_, turns, timed := combat.CombatPossessionProgress(identity)

	return mainscreen.CombatConditions{
		Possessed: combat.IsCombatPCPossessed(identity), Underground: combat.IsCombatPCUnderground(identity),
		Sleeping:        combat.IsCombatPCSleeping(identity),
		PossessionTurns: turns, TimedPossession: timed,
	}
}

func (g *GameScene) appendCombatTurnConditions(identity game_state.CombatTurnIdentity) {
	conditions := combatConditions(g.gameState.CombatScene, identity)
	if conditions == (mainscreen.CombatConditions{}) {
		return
	}
	mainscreen.AppendCombatConditions(g.combatHUD.turnOutput.AppendLine(), conditions).Done()
}
