package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameState) resolveControlledMonsterAttackAgainstPC(
	attackerIndex int,
	targetIdentity CombatTurnIdentity,
	targetPosition references.Position,
) {
	attacker := g.CombatScene.MonsterManager.GetMonster(attackerIndex)
	if attacker == nil || g.combatIdentityLoyalty(targetIdentity) != CombatMonsterLoyaltyMonster ||
		g.CombatScene.isCombatPCUnderground(targetIdentity) ||
		integerDistance(attacker.Position(), targetPosition) > max(attacker.EnemyRef().AttackRange, monsterAdjacentDistance) {
		g.SystemCallbacks.UI.ShowActionFeedback("Nothing!", true)

		return
	}
	if attacker.EnemyRef().LargeMapMissile != references.MissileNone {
		impact, reached := combatMissileImpactPosition(
			g.combatMissileTerrainAt, attacker.Position(), targetPosition, attacker.EnemyRef().LargeMapMissile,
		)
		if !reached || impact != targetPosition {
			g.SystemCallbacks.UI.ShowActionFeedback("Nothing!", true)

			return
		}
	}
	adjacent := integerDistance(attacker.Position(), targetPosition) == monsterAdjacentDistance
	defer g.finishDirectedCombatAttack(targetIdentity, g.CombatScene.isCombatPCSleeping(targetIdentity))
	if adjacent {
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundCombatSwing)
		attacker.SetCorpserExposed(true)
	}
	rng := g.AsRandomProvider()
	defense := g.combatTargetDefense(targetIdentity)
	hit := CombatHitCheck(
		g.combatPhysicalAttackAttribute(CombatTurnIdentity{EntityIndex: attackerIndex}, references.NoEquipment),
		defense.EffectiveDex,
		false,
		rng,
	)
	diagnostics := CombatDiagnostics{ //nolint:exhaustruct // Damage follows only after a hit.
		Hit:           hit,
		TargetBaseDex: defense.BaseDex,
		Missile:       attacker.EnemyRef().LargeMapMissile,
		UsedStrength:  attacker.HasAbility(references.Bludgeons),
	}
	result := CombatFeedbackMiss
	hpLost := 0
	fatal := false
	if hit.Outcome != CombatHitMiss {
		result, hpLost, fatal, diagnostics = g.applyControlledMonsterHitToPC(
			attacker, targetIdentity, adjacent, diagnostics,
		)
	}
	g.publishPartyLoyalAttackResult(
		CombatEventAttack,
		combatMonsterFeedbackEntity(attackerIndex, attacker.Position()),
		combatFeedbackIdentity(targetIdentity, targetPosition),
		result, hpLost, fatal, diagnostics,
	)
	g.tryDragCombatPCUnder(attacker, targetIdentity, adjacent, hpLost, fatal)
}

func (g *GameState) resolveCombatPCAttackSourceAgainstPC(
	attackerIdentity CombatTurnIdentity,
	source CombatAttackSource,
	targetIdentity CombatTurnIdentity,
) {
	defer g.finishDirectedCombatAttack(targetIdentity, g.CombatScene.isCombatPCSleeping(targetIdentity))
	g.SystemCallbacks.Audio.PlaySoundEffect(SoundCombatSwing)
	attackerIndex := attackerIdentity.EntityIndex
	targetIndex := targetIdentity.EntityIndex
	attackerPosition := g.CombatScene.GetCombatantPosition(attackerIdentity)
	targetPosition := g.CombatScene.GetCombatantPosition(targetIdentity)
	rng := g.AsRandomProvider()
	defense := g.combatTargetDefense(targetIdentity)
	attackerStat := g.combatPhysicalAttackAttribute(attackerIdentity, source.Equipment)
	hit := CombatHitCheck(
		attackerStat,
		defense.EffectiveDex,
		source.Equipment.HasAbility(references.AbilityShatters) ||
			source.Equipment.HasAbility(references.AbilityNeverMisses) ||
			source.Equipment.HasAbility(references.AbilityCharmsWielder),
		rng,
	)
	diagnostics := CombatDiagnostics{ //nolint:exhaustruct // Damage follows only after a hit.
		Hit: hit, Source: source.Equipment, TargetBaseDex: defense.BaseDex,
		UsedStrength: source.Equipment.IsBludgeoning(),
	}
	result := CombatFeedbackMiss
	hpLost := 0
	fatal := false
	if hit.Outcome != CombatHitMiss {
		result, hpLost, fatal, diagnostics = g.applyPlayerWeaponHitToPC(attackerIndex, source, targetIndex, diagnostics)
	}
	g.publishPartyLoyalAttackResult(
		CombatEventAttack,
		combatFeedbackIdentity(attackerIdentity, attackerPosition),
		combatFeedbackIdentity(targetIdentity, targetPosition),
		result, hpLost, fatal, diagnostics,
	)
}

func (g *GameState) publishPartyLoyalAttackResult(
	kind CombatEventKind,
	attacker CombatFeedbackEntity,
	target CombatFeedbackEntity,
	result CombatFeedbackResult,
	hpLost int,
	fatal bool,
	diagnostics CombatDiagnostics,
) {
	feedback := CombatAttackFeedback{
		Attacker: attacker,
		Target:   target,
		Result:   result,
		HPLost:   hpLost,
		Fatal:    fatal,
		Presentation: combatAttackPresentationForEvent(
			kind,
			diagnostics,
			attacker.IsPartyMember,
		),
	}
	g.SystemCallbacks.Visual.ShowCombatImpact(feedback)
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind: kind, Attacker: attacker, Target: target,
		Result: result, HPLost: hpLost, Fatal: fatal,
		From: attacker.Position, To: target.Position, Diagnostics: diagnostics,
	})
}

func (g *GameState) applyPlayerWeaponHitToPC(
	attackerIndex int,
	source CombatAttackSource,
	targetIndex int,
	diagnostics CombatDiagnostics,
) (CombatFeedbackResult, int, bool, CombatDiagnostics) {
	attacker := g.PartyState.GetCharacter(attackerIndex)
	target := g.PartyState.GetCharacter(targetIndex)
	damage := CombatCalcDamage(source.Equipment, g.combatPartyDefense(target), g.AsRandomProvider())
	diagnostics.Damage = damage
	diagnostics.FinalDamage = damage.Damage
	if damage.WeaponShattered {
		attacker.SetEquippedItem(source.Slot, references.NoEquipment)
	}
	if damage.Deflected && !damage.InstantKill {
		return CombatFeedbackGraze, 0, false, diagnostics
	}
	hpBefore := target.GetCurrentHp()
	if damage.InstantKill {
		target.SetHp(0)
		target.MarkAsDead()
	} else {
		target.SubtractHp(uint16(damage.Damage)) //nolint:gosec // Weapon damage is bounded by equipment data.
	}
	if target.IsDead() {
		g.markCombatPCDead(targetIndex)
	}

	return CombatFeedbackDamage, int(hpBefore - target.GetCurrentHp()), target.IsDead(), diagnostics
}

func (g *GameState) applyControlledMonsterHitToPC(
	attacker *CombatMonster,
	targetIdentity CombatTurnIdentity,
	adjacent bool,
	diagnostics CombatDiagnostics,
) (CombatFeedbackResult, int, bool, CombatDiagnostics) {
	targetIndex := targetIdentity.EntityIndex
	target := g.PartyState.GetCharacter(targetIndex)
	resolution := g.resolveSuccessfulMonsterHit(
		attacker, targetIdentity, adjacent, g.AsRandomProvider(), &diagnostics,
	)
	if resolution.replacedDamage {
		hpBefore := target.GetCurrentHp()
		if resolution.damageSelected {
			target.SubtractHp(uint16(resolution.damage)) //nolint:gosec // Poison fallback is bounded to 0..20.
		}
		if target.IsDead() {
			g.markCombatPCDead(targetIndex)
		}

		return resolution.feedback, int(hpBefore - target.GetCurrentHp()), target.IsDead(), diagnostics
	}
	damage := CombatCalcDamageRaw(attacker.EnemyRef().Damage, g.combatPartyDefense(target), g.AsRandomProvider())
	diagnostics.Damage = damage
	diagnostics.FinalDamage = damage.Damage
	if damage.Deflected {
		return CombatFeedbackGraze, 0, false, diagnostics
	}
	hpBefore := target.GetCurrentHp()
	target.SubtractHp(uint16(combatMonsterDamageToApply(
		int(hpBefore), damage.Damage, damage.InstantKill,
	)))
	if target.IsDead() {
		g.markCombatPCDead(targetIndex)
	}

	return CombatFeedbackDamage, int(hpBefore - target.GetCurrentHp()), target.IsDead(), diagnostics
}
