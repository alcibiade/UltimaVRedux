package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// CastCombatAnXenCorp applies the original repel-undead spell to every active
// monster turn-queue entry. PCs and non-undead monsters are not candidates.
func (g *GameState) CastCombatAnXenCorp(casterIdx int) CombatAnXenCorpResult {
	if g.CombatScene == nil {
		panic("CastCombatAnXenCorp requires active combat")
	}
	caster := g.PartyState.GetCharacter(casterIdx)
	if caster == nil {
		panic("CastCombatAnXenCorp requires a valid caster")
	}

	result := CombatAnXenCorpResult{
		Affected: make([]CombatSpellMonster, 0),
		Resisted: make([]CombatSpellMonster, 0),
		Immune:   make([]CombatSpellMonster, 0),
	}
	for _, entry := range g.CombatScene.TurnQueue.entries {
		if entry.status != turnQueueActive || entry.Combatant.IsPC() {
			continue
		}
		monster := g.CombatScene.MonsterManager.GetMonster(entry.EntityIndex)
		if monster == nil || !monster.IsActive() || monster.IsDead() {
			continue
		}
		target := CombatSpellMonster{
			CombatantID: monster.CombatantID(),
			Index:       entry.EntityIndex,
			MonsterType: monster.MonsterType(),
			Position:    monster.Position(),
		}
		if !monster.HasAbility(references.Undead) {
			continue
		}
		if isCombatSpellImmuneMonster(monster) {
			result.Immune = append(result.Immune, target)

			continue
		}

		target.Save = g.rollCombatIntelligenceSave(int(caster.Intelligence), monster.EnemyRef().Intelligence)
		if target.Save.Outcome == CombatHitMiss {
			result.Resisted = append(result.Resisted, target)

			continue
		}
		monster.SetCurrentHP(1)
		monster.SetFleeing(true)
		result.Affected = append(result.Affected, target)
	}

	casterIdentity := g.combatPartyAttackerIdentity(casterIdx)
	casterEntity := combatFeedbackIdentity(casterIdentity, g.CombatScene.GetCombatantPosition(casterIdentity))
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind:      CombatEventRepelUndead,
		Attacker:  casterEntity,
		From:      casterEntity.Position,
		AnXenCorp: CombatAnXenCorpEvent{Caster: casterEntity, Result: result},
		Diagnostics: CombatDiagnostics{
			Spell: references.AnXenCorp,
		},
	})

	return result
}

// CastCombatInQuasCorp applies fear to every active monster queue entry,
// including party-loyal creatures. Unlike An Xen Corp it has no undead filter.
func (g *GameState) CastCombatInQuasCorp(casterIdx int) CombatAnXenCorpResult {
	if g.CombatScene == nil {
		panic("CastCombatInQuasCorp requires active combat")
	}
	caster := g.PartyState.GetCharacter(casterIdx)
	if caster == nil {
		panic("CastCombatInQuasCorp requires a valid caster")
	}
	result := CombatAnXenCorpResult{Affected: make([]CombatSpellMonster, 0), Resisted: make([]CombatSpellMonster, 0), Immune: make([]CombatSpellMonster, 0)}
	for _, entry := range g.CombatScene.TurnQueue.entries {
		if entry.status != turnQueueActive || entry.Combatant.IsPC() {
			continue
		}
		monster := g.CombatScene.MonsterManager.GetMonster(entry.EntityIndex)
		if monster == nil || !monster.IsActive() || monster.IsDead() {
			continue
		}
		target := CombatSpellMonster{CombatantID: monster.CombatantID(), Index: entry.EntityIndex, MonsterType: monster.MonsterType(), Position: monster.Position()}
		if isCombatSpellImmuneMonster(monster) {
			result.Immune = append(result.Immune, target)

			continue
		}
		target.Save = g.rollCombatIntelligenceSave(int(caster.Intelligence), monster.EnemyRef().Intelligence)
		if target.Save.Outcome == CombatHitMiss {
			result.Resisted = append(result.Resisted, target)

			continue
		}
		monster.SetCurrentHP(1)
		monster.SetFleeing(true)
		result.Affected = append(result.Affected, target)
	}
	casterIdentity := g.combatPartyAttackerIdentity(casterIdx)
	casterEntity := combatFeedbackIdentity(casterIdentity, g.CombatScene.GetCombatantPosition(casterIdentity))
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind: CombatEventRepelUndead, Attacker: casterEntity, From: casterEntity.Position,
		AnXenCorp: CombatAnXenCorpEvent{Caster: casterEntity, Result: result}, Diagnostics: CombatDiagnostics{Spell: references.InQuasCorp},
	})

	return result
}
