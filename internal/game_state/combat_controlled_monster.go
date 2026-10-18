package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameState) moveControlledCombatMonster(
	monster *CombatMonster,
	monsterIndex int,
	direction references.Direction,
) CombatMoveResult {
	from := monster.Position()
	to := direction.GetNewPositionInDirection(&from)
	if to == nil {
		return CombatMoveBlocked
	}
	if to.X < 0 || to.X >= references.XCombatMapTiles ||
		to.Y < 0 || to.Y >= references.YCombatMapTiles {
		if !g.canFleeCombatIndividually() {
			return CombatMoveBlocked
		}
		monster.SetActive(false)
		g.CombatScene.TurnQueue.MarkFled(monsterIndex, false)
		g.releaseCombatController(monster.CombatantID(), CombatPossessionReleaseControllerEscaped)
		g.SystemCallbacks.Message.AddRowStr(monster.MonsterType().Name() + " - Escape!")
		g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Flee has no target or roll data.
			Kind: CombatEventFled,
			Attacker: CombatFeedbackEntity{
				IsPartyMember: false,
				Index:         monsterIndex,
				Position:      from,
			},
		})
		g.resolveCombatControlledEndAction(CombatTurnIdentity{EntityIndex: monsterIndex})

		return CombatMoveFled
	}
	if !isCombatPositionValidForMonster(g, *to, monster, monsterIndex) {
		return CombatMoveBlocked
	}

	monster.SetPosition(*to)
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Movement has no target or roll result.
		Kind: CombatEventMove,
		Attacker: CombatFeedbackEntity{
			IsPartyMember: false,
			Index:         monsterIndex,
			Position:      *to,
		},
		From: from,
		To:   *to,
	})
	g.CheckTrap(to)
	g.resolveCombatMonsterActionEffects(monster, monsterIndex)
	logging.Debug(logging.CategoryGameState,
		"Controlled monster %s (idx=%d) moved %s to (%d,%d)",
		monster.MonsterType().Name(), monsterIndex, direction.GetDirectionCompassName(), to.X, to.Y)

	return CombatMoveMoved
}

func (g *GameState) resolveControlledMonsterAttack(
	attackerIndex int,
	target *CombatMonster,
	targetIndex int,
	targetPosition references.Position,
) {
	attacker := g.CombatScene.MonsterManager.GetMonster(attackerIndex)
	if attacker == nil || target == nil || !target.IsHostile() {
		g.SystemCallbacks.UI.ShowActionFeedback("Nothing!", true)

		return
	}
	if integerDistance(attacker.Position(), targetPosition) > max(attacker.EnemyRef().AttackRange, monsterAdjacentDistance) {
		g.SystemCallbacks.UI.ShowActionFeedback("Nothing!", true)

		return
	}
	if attacker.EnemyRef().LargeMapMissile != references.MissileNone {
		impact, reachedTarget := combatMissileImpactPosition(
			g.combatMissileTerrainAt,
			attacker.Position(),
			targetPosition,
			attacker.EnemyRef().LargeMapMissile,
		)
		if !reachedTarget || impact != targetPosition {
			g.SystemCallbacks.UI.ShowActionFeedback("Nothing!", true)

			return
		}
	}
	if integerDistance(attacker.Position(), targetPosition) == monsterAdjacentDistance {
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundCombatSwing)
		attacker.SetCorpserExposed(true)
	}
	attackerStat := g.combatPhysicalAttackAttribute(CombatTurnIdentity{EntityIndex: attackerIndex}, references.NoEquipment)
	defense := g.combatTargetDefense(CombatTurnIdentity{IsPartyMember: false, EntityIndex: targetIndex})
	hitResult := CombatHitCheck(attackerStat, defense.EffectiveDex, false, g.AsRandomProvider())
	diagnostics := CombatDiagnostics{ //nolint:exhaustruct // Damage is populated only after a hit.
		Hit:           hitResult,
		TargetBaseDex: defense.BaseDex,
		UsedStrength:  attacker.HasAbility(references.Bludgeons),
	}
	if hitResult.Outcome == CombatHitMiss {
		g.showControlledMonsterCombatFeedback(
			CombatEventAttack, attackerIndex, targetIndex, CombatFeedbackMiss, 0, false, diagnostics,
		)

		return
	}
	g.resolveControlledMonsterHitAgainstMonster(
		attackerIndex, target, targetIndex, diagnostics, CombatEventAttack,
	)
}

// Approved Redux policy: controlled physical attacks use the hostile monster
// damage classification, including poison fallback and surviving-hit condition.
// REDUX DEVIATION: player attack passes NOTHING (255), whose >= MAGIC_WEAPON
// comparison sets magic. Its later attack-range table/attack attribute selection accesses with that sentinel
// are invalid; the recovered source does not define a reliable binary outcome.
// RangedMagic controls monster missile abilities, not this physical damage class.
// See ref:COMBAT-0004.
func (g *GameState) resolveControlledMonsterHitAgainstMonster(
	attackerIndex int,
	target *CombatMonster,
	targetIndex int,
	diagnostics CombatDiagnostics,
	eventKind CombatEventKind,
) {
	attacker := g.CombatScene.MonsterManager.GetMonster(attackerIndex)
	targetIdentity := CombatTurnIdentity{IsPartyMember: false, EntityIndex: targetIndex}
	adjacent := eventKind == CombatEventAttack &&
		integerDistance(attacker.Position(), target.Position()) == monsterAdjacentDistance
	resolution := g.resolveSuccessfulMonsterHit(
		attacker, targetIdentity, adjacent, g.AsRandomProvider(), &diagnostics,
	)
	if resolution.replacedDamage {
		hpBefore := target.CurrentHP()
		if resolution.damageSelected {
			target.TakeDamage(resolution.damage)
		}
		hpLost := hpBefore - target.CurrentHP()
		g.showControlledMonsterCombatFeedback(
			eventKind, attackerIndex, targetIndex, resolution.feedback, hpLost, target.IsDead(), diagnostics,
		)
		if hpBefore > 0 && target.IsDead() {
			g.finalizeCombatMonsterDeathWithPolicy(target, targetIndex, nil, NoTargetPC, false, true)
		} else if resolution.damageSelected && !target.IsDead() {
			g.resolveSurvivingCombatMonsterHit(target, targetIndex, true)
		}

		return
	}
	damageResult := CombatCalcDamageRaw(attacker.EnemyRef().Damage, target.EnemyRef().Armour, g.AsRandomProvider())
	diagnostics.Damage = damageResult
	diagnostics.TargetUndead = target.MonsterType().IsUndead()
	diagnostics.TargetImmortal = target.HasAbility(references.Immortal)
	diagnostics.MagicWeapon = false
	diagnostics.FinalDamage = CombatApplyDamageModifiers(
		damageResult.Damage,
		diagnostics.TargetUndead,
		diagnostics.TargetImmortal,
		diagnostics.MagicWeapon,
	)
	if damageResult.Deflected {
		g.showControlledMonsterCombatFeedback(
			eventKind, attackerIndex, targetIndex, CombatFeedbackGraze, 0, false, diagnostics,
		)

		return
	}

	hpBefore := target.CurrentHP()
	target.TakeDamage(combatMonsterDamageToApply(
		hpBefore, diagnostics.FinalDamage, damageResult.InstantKill,
	))
	hpLost := hpBefore - target.CurrentHP()
	g.showControlledMonsterCombatFeedback(
		eventKind, attackerIndex, targetIndex, CombatFeedbackDamage, hpLost, target.IsDead(), diagnostics,
	)
	if hpBefore > 0 && target.IsDead() {
		g.finalizeCombatMonsterDeathWithPolicy(target, targetIndex, nil, NoTargetPC, false, true)
	} else if !target.IsDead() {
		g.resolveSurvivingCombatMonsterHit(target, targetIndex, true)
	}
}

func (g *GameState) showControlledMonsterCombatFeedback(
	eventKind CombatEventKind,
	attackerIndex, targetIndex int,
	result CombatFeedbackResult,
	hpLost int,
	fatal bool,
	diagnostics CombatDiagnostics,
) {
	attacker := g.CombatScene.MonsterManager.GetMonster(attackerIndex)
	target := g.CombatScene.MonsterManager.GetMonster(targetIndex)
	injury := g.captureCombatMonsterInjury(CombatTurnIdentity{EntityIndex: targetIndex}, hpLost)
	feedback := CombatAttackFeedback{
		Attacker: CombatFeedbackEntity{
			IsPartyMember: false,
			Index:         attackerIndex,
			Position:      attacker.Position(),
		},
		Target: CombatFeedbackEntity{
			IsPartyMember: false,
			Index:         targetIndex,
			Position:      target.Position(),
		},
		Result: result,
		HPLost: hpLost,
		Fatal:  fatal,
		Presentation: combatAttackPresentationForEvent(
			eventKind,
			diagnostics,
			false,
		),
	}
	g.SystemCallbacks.Visual.ShowCombatImpact(feedback)
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind:          eventKind,
		Attacker:      feedback.Attacker,
		Target:        feedback.Target,
		Result:        result,
		HPLost:        hpLost,
		Fatal:         fatal,
		From:          feedback.Attacker.Position,
		To:            feedback.Target.Position,
		Diagnostics:   diagnostics,
		MonsterInjury: injury,
	})
}
