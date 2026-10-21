package game_state

// ActionEscapeCombatMap leaves combat when the exit is allowed and ends it as victory, evacuation or defeat.
func (g *GameState) ActionEscapeCombatMap() bool {
	if !g.CanEscapeCombatMap() {
		return false
	}
	reason := CombatVictory
	if !g.CombatScene.Victory {
		monsterCount := g.CombatScene.MonsterManager.CountActiveHostile()
		reason = CombatEvacuated
		if monsterCount > 0 {
			reason = CombatDefeat
		}
	}
	g.EndCombat(reason)

	return true
}

// CanEscapeCombatMap validates an explicit combat exit without mutating combat state.
func (g *GameState) CanEscapeCombatMap() bool {
	if g.CombatScene == nil {
		g.SystemCallbacks.Message.AddRowStr("Not here!")

		return false
	}

	pcCount := g.CombatScene.countActiveCombatPCs()
	if pcCount == 0 {
		return true
	}

	if g.CombatScene.ExitContext == CombatExitDungeonRoom {
		if !g.CombatScene.Victory {
			g.SystemCallbacks.Message.AddRowStr("Not here!")

			return false
		}
		if g.CombatScene.ExitRoute == CombatExitNone {
			g.SystemCallbacks.Message.AddRowStr("Choose an exit first.")

			return false
		}
	}

	if !g.CombatScene.Victory {
		g.SystemCallbacks.Message.AddRowStr("Not yet!")

		return false
	}

	return true
}
