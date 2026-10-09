package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameState) castCommittedCombatCone(
	spell references.Spell,
	casterIdx int,
	direction references.Direction,
) []references.Position {
	spec, ok := CombatConeSpecForSpell(spell)
	if !ok {
		panic("combat cone configuration missing")
	}
	origin := g.combatPartyAttackerPosition(casterIdx)
	candidates := g.collectCommittedCombatCone(origin, direction, spec)

	return g.resolveCommittedCombatCone(spell, casterIdx, candidates)
}

// ResolveCommittedCombatCone applies the candidate collection retained at paid
// confirmation without rerunning or rerandomizing the source ray traversal.
func (g *GameState) ResolveCommittedCombatCone(result CombatConeTargetResult) []references.Position {
	if result.Outcome != CombatConeTargetResolved || !result.committed {
		panic("combat cone resolution requires a committed target result")
	}

	return g.resolveCommittedCombatCone(result.Request.Spell, result.Request.CasterIdx, result.candidates)
}

func (g *GameState) resolveCommittedCombatCone(
	spell references.Spell,
	casterIdx int,
	candidates []references.Position,
) []references.Position {
	//nolint:exhaustive // Only configured combat cone spells can reach resolution.
	switch spell {
	case references.InZu:
		return g.resolveCombatInZu(casterIdx, candidates)
	case references.InNoxHur:
		return g.resolveCombatInNoxHur(casterIdx, candidates)
	case references.InFlamHur:
		return g.resolveCombatInFlamHur(casterIdx, candidates)
	case references.InVasGravCorp:
		return g.resolveCombatInVasGravCorp(casterIdx, candidates)
	default:
		panic("unsupported committed combat cone spell")
	}
}

// CastCombatInNoxHur applies the original poison-wind cone to every combatant
// in its terrain-clipped footprint. Toxify applies when effective DEX is no
// greater than the D30 roll: poison a Good PC, otherwise deal direct damage.
func (g *GameState) CastCombatInNoxHur(casterIdx int, direction references.Direction) []references.Position {
	return g.castCommittedCombatCone(references.InNoxHur, casterIdx, direction)
}

func (g *GameState) resolveCombatInNoxHur(
	casterIdx int,
	candidates []references.Position,
) []references.Position {
	affected := make([]references.Position, 0)
	poisonOnset := false
	processed := make(map[CombatTurnIdentity]struct{})
	for _, position := range candidates {
		target, found := g.CombatScene.combatConeVictimAt(position, processed)
		if !found {
			continue
		}
		applied, partyMemberPoisoned := g.applyCombatPoisonWindToxify(
			casterIdx, target, g.rollCombatDexterityThreshold(g.effectiveCombatDexterity(target)),
		)
		poisonOnset = poisonOnset || partyMemberPoisoned
		if applied {
			affected = append(affected, position)
		}
	}
	if poisonOnset {
		g.SystemCallbacks.UI.ShowTransientIcon(TransientIconPoisonOnset)
	}

	return affected
}

// CastCombatInFlamHur applies the original flame-wind cone. Each occupied tile
// takes direct random damage from 0 through 30 without an armor reduction.
// original spell projection rules and original command rules establish the inclusive source contract;
// the external RNG implementation is unavailable for binary confirmation.
// See ref:SPELL-0003.
func (g *GameState) CastCombatInFlamHur(casterIdx int, direction references.Direction) []references.Position {
	return g.castCommittedCombatCone(references.InFlamHur, casterIdx, direction)
}

func (g *GameState) resolveCombatInFlamHur(
	casterIdx int,
	candidates []references.Position,
) []references.Position {
	affected := make([]references.Position, 0)
	processed := make(map[CombatTurnIdentity]struct{})
	for _, position := range candidates {
		target, found := g.CombatScene.combatConeVictimAt(position, processed)
		if found && g.applyCombatInFlamHurToTarget(casterIdx, target) {
			affected = append(affected, position)
		}
	}

	return affected
}

// CastCombatInVasGravCorp applies the original energy-vortex cone. Targets
// first receive an intelligence save; spell-immune and immortal monsters
// cannot be killed. A failed save deals the original instant-death value.
func (g *GameState) CastCombatInVasGravCorp(casterIdx int, direction references.Direction) []references.Position {
	return g.castCommittedCombatCone(references.InVasGravCorp, casterIdx, direction)
}

func (g *GameState) resolveCombatInVasGravCorp(
	casterIdx int,
	candidates []references.Position,
) []references.Position {
	affected := make([]references.Position, 0)
	processed := make(map[CombatTurnIdentity]struct{})
	for _, position := range candidates {
		target, found := g.CombatScene.combatConeVictimAt(position, processed)
		if found && g.applyCombatInVasGravCorpToTarget(casterIdx, target) {
			affected = append(affected, position)
		}
	}

	return affected
}

func (g *GameState) applyCombatInFlamHurToTarget(casterIdx int, target CombatTurnIdentity) bool {
	position := g.CombatScene.GetCombatantPosition(target)
	if !target.IsPartyMember {
		monsterIdx := target.EntityIndex
		monster := g.CombatScene.MonsterManager.GetMonster(monsterIdx)
		damage := g.GetRNG().Intn(31)
		hpBefore := monster.CurrentHP()
		finalDamage := CombatApplyMagicDamageModifiers(
			damage,
			monster.MonsterType().IsUndead(),
			monster.HasAbility(references.Immortal),
		)
		monster.TakeDamage(finalDamage)
		hpLost := hpBefore - monster.CurrentHP()
		g.publishCombatWindDamageResult(
			references.InFlamHur, CombatHitResult{}, casterIdx, false, monsterIdx, 0, position, damage, finalDamage, hpLost, monster.IsDead(),
		)
		if hpBefore > 0 && monster.IsDead() {
			g.finalizeCombatMonsterDeath(monster, monsterIdx, g.PartyState.GetCharacter(casterIdx), casterIdx)
		} else if hpLost > 0 {
			g.tryDivideCombatMonster(monster, monsterIdx)
		}

		return true
	}
	if target.IsPartyMember {
		pcIdx := target.EntityIndex
		character := g.PartyState.GetCharacter(pcIdx)
		damage := g.GetRNG().Intn(31)
		hpBefore := character.GetCurrentHp()
		character.SubtractHp(uint16(damage))
		hpLost := int(hpBefore - character.GetCurrentHp())
		if character.IsDead() {
			g.markCombatPCDead(pcIdx)
		}
		g.publishCombatWindDamageResult(
			references.InFlamHur, CombatHitResult{}, casterIdx, true, pcIdx, target.Slot, position, damage, damage, hpLost, character.IsDead(),
		)

		return true
	}

	return false
}

func (g *GameState) applyCombatInVasGravCorpToTarget(casterIdx int, target CombatTurnIdentity) bool {
	caster := g.PartyState.GetCharacter(casterIdx)
	position := g.CombatScene.GetCombatantPosition(target)
	if !target.IsPartyMember {
		monsterIdx := target.EntityIndex
		monster := g.CombatScene.MonsterManager.GetMonster(monsterIdx)
		hit := g.rollCombatIntelligenceSave(int(caster.Intelligence), monster.EnemyRef().Intelligence)
		if isCombatSpellImmuneMonster(monster) || monster.HasAbility(references.Immortal) {
			hit.Outcome = CombatHitMiss
		}
		if hit.Outcome == CombatHitMiss {
			g.publishCombatEnergyVortexResult(casterIdx, false, monsterIdx, 0, position, hit, 0, false)

			return false
		}
		hpBefore := monster.CurrentHP()
		monster.SetCurrentHP(0)
		g.publishCombatEnergyVortexResult(casterIdx, false, monsterIdx, 0, position, hit, hpBefore, true)
		g.finalizeCombatMonsterDeath(monster, monsterIdx, caster, casterIdx)

		return true
	}
	if target.IsPartyMember {
		pcIdx := target.EntityIndex
		character := g.PartyState.GetCharacter(pcIdx)
		hit := g.rollCombatIntelligenceSave(int(caster.Intelligence), int(character.Intelligence))
		if hit.Outcome == CombatHitMiss {
			g.publishCombatEnergyVortexResult(casterIdx, true, pcIdx, target.Slot, position, hit, 0, false)

			return false
		}
		hpBefore := character.GetCurrentHp()
		character.SetHp(0)
		character.MarkAsDead()
		g.markCombatPCDead(pcIdx)
		g.publishCombatEnergyVortexResult(
			casterIdx, true, pcIdx, target.Slot, position, hit, int(hpBefore), true,
		)

		return true
	}

	return false
}

// Original combat damage rules poisons a Good PC; every other target takes damage.
// See ref:SPELL-0003.
func (g *GameState) applyCombatPoisonWindToxify(casterIdx int, target CombatTurnIdentity, hit CombatHitResult) (bool, bool) {
	position := g.CombatScene.GetCombatantPosition(target)
	if hit.Outcome == CombatHitMiss {
		g.publishCombatPoisonWindResult(casterIdx, target.IsPartyMember, target.EntityIndex, target.Slot, position, hit)

		return false, false
	}
	if target.IsPartyMember && g.PartyState.GetCharacter(target.EntityIndex).IsGood() {
		g.PartyState.GetCharacter(target.EntityIndex).SetStatus(party_state.Poisoned)
		g.publishCombatPoisonWindResult(casterIdx, true, target.EntityIndex, target.Slot, position, hit)

		return true, true
	}
	damage := g.GetRNG().Intn(monsterPoisonDamageRange)
	finalDamage := damage
	var hpLost int
	var fatal bool
	if target.IsPartyMember {
		character := g.PartyState.GetCharacter(target.EntityIndex)
		hpBefore := character.GetCurrentHp()
		character.SubtractHp(uint16(damage))
		hpLost, fatal = int(hpBefore-character.GetCurrentHp()), character.IsDead()
		if fatal {
			g.markCombatPCDead(target.EntityIndex)
		}
	} else {
		monster := g.CombatScene.MonsterManager.GetMonster(target.EntityIndex)
		finalDamage = CombatApplyMagicDamageModifiers(damage, monster.MonsterType().IsUndead(), monster.HasAbility(references.Immortal))
		hpBefore := monster.CurrentHP()
		monster.TakeDamage(finalDamage)
		hpLost, fatal = hpBefore-monster.CurrentHP(), monster.IsDead()
	}
	g.publishCombatWindDamageResult(references.InNoxHur, hit, casterIdx, target.IsPartyMember, target.EntityIndex, target.Slot, position, damage, finalDamage, hpLost, fatal)
	if !target.IsPartyMember {
		monster := g.CombatScene.MonsterManager.GetMonster(target.EntityIndex)
		if fatal {
			g.finalizeCombatMonsterDeath(monster, target.EntityIndex, g.PartyState.GetCharacter(casterIdx), casterIdx)
		} else if hpLost > 0 {
			g.tryDivideCombatMonster(monster, target.EntityIndex)
		}
	}

	return true, false
}

func (g *GameState) rollCombatDexterityThreshold(threshold int) CombatHitResult {
	// Original: cone-ray effect calls transformed d30 for poison wind.
	// See ref:RNG-0001.
	roll := g.RollOriginalD30()

	return combatDexterityThresholdResult(roll, threshold)
}

func combatDexterityThresholdResult(roll, threshold int) CombatHitResult {
	result := CombatHitResult{
		Outcome:              CombatHitMiss,
		DefenderEffectiveDex: threshold,
		Threshold:            threshold,
		D30Roll:              roll,
		Rolled:               true,
	}
	if roll >= threshold {
		result.Outcome = CombatHitNormal
	}

	return result
}

func (g *GameState) rollCombatIntelligenceSave(casterIntelligence, targetIntelligence int) CombatHitResult {
	// Original: Intelligence saving throw calls transformed d30.
	// See ref:RNG-0001.
	return rollCombatIntelligenceSaveWithRNG(casterIntelligence, targetIntelligence, g.AsRandomProvider())
}

func rollCombatIntelligenceSaveWithRNG(
	casterIntelligence, targetIntelligence int,
	rng references.RandomProvider,
) CombatHitResult {
	roll := rollOriginalD30(rng)
	threshold := (30 + targetIntelligence - casterIntelligence) / 2
	result := CombatHitResult{
		Outcome:      CombatHitNormal,
		AttackerStat: casterIntelligence,
		Threshold:    threshold,
		D30Roll:      roll,
		Rolled:       true,
	}
	if roll < threshold {
		result.Outcome = CombatHitMiss
	}

	return result
}

func (g *GameState) publishCombatPoisonWindResult(
	casterIdx int,
	partyMember bool,
	targetIdx, targetSlot int,
	position references.Position,
	hit CombatHitResult,
) {
	result := CombatFeedbackMiss
	if hit.Outcome != CombatHitMiss {
		result = CombatFeedbackPoison
	}
	casterIdentity := g.combatPartyAttackerIdentity(casterIdx)
	attacker := combatFeedbackIdentity(casterIdentity, g.CombatScene.GetCombatantPosition(casterIdentity))
	target := CombatFeedbackEntity{
		IsPartyMember: partyMember, Index: targetIdx, Slot: targetSlot, Position: position,
	}
	g.SystemCallbacks.Visual.ShowCombatImpact(CombatAttackFeedback{
		Attacker: attacker,
		Target:   target,
		Result:   result,
		Presentation: CombatAttackPresentation{
			Delivery: CombatAttackDeliverySpell,
			Source:   references.NoEquipment,
		},
	})
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Poison wind has no HP damage or physical attack data.
		Kind:     CombatEventSpell,
		Attacker: attacker,
		Target:   target,
		Result:   result,
		From:     attacker.Position,
		To:       position,
		Diagnostics: CombatDiagnostics{
			Hit:         hit,
			Spell:       references.InNoxHur,
			MagicDamage: true,
		},
	})
}

func (g *GameState) publishCombatWindDamageResult(
	spell references.Spell,
	hit CombatHitResult,
	casterIdx int,
	partyMember bool,
	targetIdx, targetSlot int,
	position references.Position,
	damage, finalDamage, hpLost int,
	fatal bool,
) {
	attackMax := 30
	if spell == references.InNoxHur {
		attackMax = monsterPoisonDamageRange - 1
	}
	result := CombatFeedbackDamage
	if hpLost == 0 {
		result = CombatFeedbackGraze
	}
	casterIdentity := g.combatPartyAttackerIdentity(casterIdx)
	attacker := combatFeedbackIdentity(casterIdentity, g.CombatScene.GetCombatantPosition(casterIdentity))
	target := CombatFeedbackEntity{
		IsPartyMember: partyMember, Index: targetIdx, Slot: targetSlot, Position: position,
	}
	if hpLost > 0 {
		g.SystemCallbacks.Visual.ShowCombatImpact(CombatAttackFeedback{
			Attacker: attacker,
			Target:   target,
			Result:   result,
			HPLost:   hpLost,
			Fatal:    fatal,
			Presentation: CombatAttackPresentation{
				Delivery: CombatAttackDeliverySpell,
				Source:   references.NoEquipment,
			},
		})
	}
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Wind damage has no physical attack data.
		Kind:     CombatEventSpell,
		Attacker: attacker,
		Target:   target,
		Result:   result,
		HPLost:   hpLost,
		Fatal:    fatal,
		From:     attacker.Position,
		To:       position,
		Diagnostics: CombatDiagnostics{
			Hit: hit,
			Damage: CombatDamageResult{ //nolint:exhaustruct // Wind damage has no weapon or armor roll.
				Damage:       damage,
				AttackMax:    attackMax,
				AttackRoll:   damage,
				AttackRolled: true,
			},
			FinalDamage: finalDamage,
			Spell:       spell,
			MagicDamage: true,
		},
	})
}

func (g *GameState) publishCombatEnergyVortexResult(
	casterIdx int,
	partyMember bool,
	targetIdx, targetSlot int,
	position references.Position,
	hit CombatHitResult,
	hpLost int,
	fatal bool,
) {
	result := CombatFeedbackMiss
	if hpLost > 0 {
		result = CombatFeedbackDamage
	}
	casterIdentity := g.combatPartyAttackerIdentity(casterIdx)
	attacker := combatFeedbackIdentity(casterIdentity, g.CombatScene.GetCombatantPosition(casterIdentity))
	target := CombatFeedbackEntity{
		IsPartyMember: partyMember, Index: targetIdx, Slot: targetSlot, Position: position,
	}
	if hpLost > 0 {
		g.SystemCallbacks.Visual.ShowCombatImpact(CombatAttackFeedback{
			Attacker: attacker,
			Target:   target,
			Result:   result,
			HPLost:   hpLost,
			Fatal:    fatal,
			Presentation: CombatAttackPresentation{
				Delivery: CombatAttackDeliverySpell,
				Source:   references.NoEquipment,
			},
		})
	}
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Energy vortex has no physical attack data.
		Kind:     CombatEventSpell,
		Attacker: attacker,
		Target:   target,
		Result:   result,
		HPLost:   hpLost,
		Fatal:    fatal,
		From:     attacker.Position,
		To:       position,
		Diagnostics: CombatDiagnostics{
			Hit:         hit,
			FinalDamage: combatInstantKillDamage,
			Spell:       references.InVasGravCorp,
			MagicDamage: true,
		},
	})
}
