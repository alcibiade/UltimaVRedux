package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// rememberMeleeEngager records the latest adjacent attacker before
// resolving the hit, so misses replace the remembered attacker too. ref:EQUIP-0016
func (g *GameState) rememberMeleeEngager(target, attacker CombatTurnIdentity) {
	if g.CombatScene == nil {
		return
	}
	if g.CombatScene.meleeEngagers == nil {
		g.CombatScene.meleeEngagers = make(map[CombatTurnIdentity]CombatTurnIdentity)
	}
	g.CombatScene.meleeEngagers[target] = attacker
}

func (g *GameState) rememberedMeleeEngager(target CombatTurnIdentity) (CombatTurnIdentity, bool) {
	if g.CombatScene == nil {
		return CombatTurnIdentity{}, false
	}
	attacker, found := g.CombatScene.meleeEngagers[target]

	return attacker, found
}

// validRememberedMeleeEngager returns the one remembered attacker only when it
// still satisfies original attack rules's interference conditions.
// See ref:EQUIP-0016.
func (g *GameState) validRememberedMeleeEngager(target CombatTurnIdentity) (CombatTurnIdentity, bool) {
	attacker, found := g.rememberedMeleeEngager(target)
	if !found || g.HasActiveDurationEffect(DurationEffectTimeStop) ||
		!g.isActiveCombatIdentity(attacker) || g.isCombatIdentityAsleep(attacker) ||
		(attacker.IsPartyMember && g.CombatScene.isCombatPCUnderground(attacker)) ||
		g.combatIdentityLoyalty(attacker) == g.combatIdentityLoyalty(target) ||
		integerDistance(g.CombatScene.GetCombatantPosition(attacker), g.CombatScene.GetCombatantPosition(target)) != monsterAdjacentDistance {
		return CombatTurnIdentity{}, false
	}

	return attacker, true
}

// CheckCombatCastInterference applies melee casting interference before a PC enters
// the Cast workflow. Interference preserves the exact actor and action.
// See ref:EQUIP-0016.
func (g *GameState) CheckCombatCastInterference() bool {
	if g.CombatScene == nil {
		return false
	}
	actor := g.CombatScene.GetCurrentTurnIdentity()
	if !actor.IsPartyMember {
		return false
	}
	engager, found := g.validRememberedMeleeEngager(actor)
	if !found {
		return false
	}
	g.publishCombatInterference(actor, engager, CombatInterruptedCast, references.NoEquipment)

	return true
}

func (g *GameState) publishCombatInterference(
	actor, engager CombatTurnIdentity,
	action CombatInterruptedAction,
	source references.Equipment,
) {
	g.SystemCallbacks.Message.AddRowStr(g.combatIdentityName(engager) + " interferes!")
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{
		Kind:              CombatEventInterruption,
		Attacker:          combatFeedbackIdentity(engager, g.combatTurnPosition(engager)),
		Target:            combatFeedbackIdentity(actor, g.combatTurnPosition(actor)),
		InterruptedAction: action,
		Diagnostics:       CombatDiagnostics{Source: source},
	})
}

// clearRememberedMeleeEngager clears the actor’s remembered attacker
// after that actor’s scheduled action. ref:EQUIP-0016
func (g *GameState) clearRememberedMeleeEngager(target CombatTurnIdentity) {
	if g.CombatScene != nil {
		delete(g.CombatScene.meleeEngagers, target)
	}
}

func (g *GameState) isActiveCombatIdentity(identity CombatTurnIdentity) bool {
	if g.CombatScene.TurnQueue == nil || identity.EntityIndex < 0 {
		return false
	}
	entry := g.CombatScene.TurnQueue.GetEntryByIdentity(identity)
	if entry == nil || entry.status != turnQueueActive || !entry.Combatant.IsAlive() {
		return false
	}
	if identity.IsPartyMember {
		if identity.EntityIndex >= len(g.CombatScene.PartyActive) {
			return false
		}

		return identity.Slot != 0 || g.CombatScene.PartyActive[identity.EntityIndex]
	}
	monster := g.CombatScene.MonsterManager.GetMonster(identity.EntityIndex)

	return monster != nil && monster.IsActive() && !monster.IsDead()
}

func (g *GameState) isCombatIdentityAsleep(identity CombatTurnIdentity) bool {
	if identity.IsPartyMember {
		return g.CombatScene.isCombatPCSleeping(identity)
	}
	monster := g.CombatScene.MonsterManager.GetMonster(identity.EntityIndex)

	return monster == nil || monster.IsAsleep()
}

func (g *GameState) combatIdentityLoyalty(identity CombatTurnIdentity) CombatMonsterLoyalty {
	if identity.IsPartyMember {
		if g.CombatScene.isCombatPCPossessed(identity) || g.isIntrinsicHostilePC(identity) {
			return CombatMonsterLoyaltyMonster
		}

		return CombatMonsterLoyaltyParty
	}
	monster := g.CombatScene.MonsterManager.GetMonster(identity.EntityIndex)
	if monster == nil {
		return CombatMonsterLoyaltyMonster
	}

	return monster.Loyalty()
}

func (g *GameState) isIntrinsicHostilePC(identity CombatTurnIdentity) bool {
	if !identity.IsPartyMember || identity.EntityIndex == party_state.AvatarCharacterIndex {
		return false
	}
	character := g.PartyState.GetCharacter(identity.EntityIndex)

	// REDUX DEVIATION: allegiance handling identifies Saduj by testing
	// name[4] == 'j'. Match the canonical full name so unrelated names are not hostile.
	// See ref:EQUIP-0016.
	return character != nil && character.GetNameAsString() == "Saduj"
}
