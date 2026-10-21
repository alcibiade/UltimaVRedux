package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	monsterSpecialEffectRollRange = 4
	// Original combat damage rules poison-on-hit check uses bounded random draw: 21 outcomes under original command rules's inclusive source contract.
	// See ref:COMBAT-0005.
	monsterPoisonDamageRange = 21
	gremlinFoodTheftAmount   = 5
)

type monsterSuccessfulHitResult struct {
	feedback       CombatFeedbackResult
	damage         int
	damageSelected bool
	replacedDamage bool
}

func (g *GameState) resolveSuccessfulMonsterHit(
	attacker *CombatMonster,
	target CombatTurnIdentity,
	adjacent bool,
	rng references.RandomProvider,
	diagnostics *CombatDiagnostics,
) monsterSuccessfulHitResult {
	food := &g.PartyState.Inventory.Provisions.Food
	if adjacent && attacker.HasAbility(references.StealsFood) && food.HasSome() {
		diagnostics.TheftRolled = true
		diagnostics.TheftRollMaximum = monsterSpecialEffectRollRange - 1
		diagnostics.TheftRoll = rng.Intn(monsterSpecialEffectRollRange)
		if diagnostics.TheftRoll != 0 {
			stolen := min(gremlinFoodTheftAmount, int(food.Get()))
			food.DecrementBy(uint16(stolen)) //nolint:gosec // Stolen is bounded by the current uint16 quantity.
			diagnostics.FoodStolen = stolen

			return monsterSuccessfulHitResult{feedback: CombatFeedbackTheft, replacedDamage: true}
		}
	}

	// Original game reference definitions POISONS is the compound 0x0204 mask: Poison or PoisonAtRange.
	// See ref:COMBAT-0005.
	if attacker.HasAbility(references.Poison) || attacker.HasAbility(references.PoisonAtRange) {
		diagnostics.PoisonRolled = true
		diagnostics.PoisonRollMaximum = monsterSpecialEffectRollRange - 1
		diagnostics.PoisonRoll = rng.Intn(monsterSpecialEffectRollRange)
		if diagnostics.PoisonRoll != 0 {
			if target.IsPartyMember {
				character := g.PartyState.GetCharacter(target.EntityIndex)
				if character.GetStatus() == party_state.Good {
					statusBefore := character.GetStatus()
					character.SetStatus(party_state.Poisoned)
					changes := transientStatusChanges{}
					changes.observePoison(statusBefore, character)
					g.publishTransientStatusChanges(changes)

					return monsterSuccessfulHitResult{feedback: CombatFeedbackPoison, replacedDamage: true}
				}
			}

			diagnostics.PoisonFallbackRolled = true
			diagnostics.PoisonFallbackDamage = rng.Intn(monsterPoisonDamageRange)
			finalDamage := diagnostics.PoisonFallbackDamage
			if !target.IsPartyMember {
				monster := g.CombatScene.MonsterManager.GetMonster(target.EntityIndex)
				if monster != nil {
					diagnostics.TargetUndead = monster.MonsterType().IsUndead()
					diagnostics.TargetImmortal = monster.HasAbility(references.Immortal)
					finalDamage = CombatApplyDamageModifiers(
						finalDamage,
						diagnostics.TargetUndead,
						diagnostics.TargetImmortal,
						diagnostics.MagicDamage,
					)
				}
			}
			diagnostics.FinalDamage = finalDamage

			return monsterSuccessfulHitResult{
				feedback:       combatFeedbackForSelectedDamage(finalDamage),
				damage:         finalDamage,
				damageSelected: true,
				replacedDamage: true,
			}
		}
	}

	if attacker.MonsterType() == references.MonsterGazer && !g.combatTargetSleeping(target) {
		g.setCombatTargetSleeping(target)

		return monsterSuccessfulHitResult{feedback: CombatFeedbackSleep, replacedDamage: true}
	}

	return monsterSuccessfulHitResult{}
}

func (g *GameState) combatTargetSleeping(target CombatTurnIdentity) bool {
	if target.IsPartyMember {
		return g.CombatScene.isCombatPCSleeping(target)
	}

	monster := g.CombatScene.MonsterManager.GetMonster(target.EntityIndex)

	return monster != nil && monster.IsAsleep()
}

func (g *GameState) setCombatTargetSleeping(target CombatTurnIdentity) {
	if target.IsPartyMember {
		g.setCombatPCSleeping(target, true)

		return
	}

	g.CombatScene.MonsterManager.GetMonster(target.EntityIndex).SetAsleep(true)
}

func combatFeedbackForSelectedDamage(damage int) CombatFeedbackResult {
	if damage == 0 {
		return CombatFeedbackGraze
	}

	return CombatFeedbackDamage
}
