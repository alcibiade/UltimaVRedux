package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

const (
	relTymDurationTurns    = 30
	inSanctDurationTurns   = 20
	quasAnWisDurationTurns = 20
	inAnDurationTurns      = 10
	anTymDurationTurns     = 10
)

// CastCombatRelTym activates the original single-slot quickness effect.
func (g *GameState) CastCombatRelTym() bool {
	return g.ActivateQuicknessEffect(relTymDurationTurns)
}

// CastCombatSanctLor makes the caster invisible for the active encounter.
func (g *GameState) CastCombatSanctLor(casterIdx int) bool {
	if g.CombatScene == nil {
		return false
	}
	identity := g.CombatScene.GetCurrentTurnIdentity()
	if !identity.IsPartyMember || identity.EntityIndex != casterIdx ||
		!g.CombatScene.isCombatPCActive(identity) {
		return false
	}
	g.CombatScene.setCombatPCInvisible(identity, true)

	return true
}

// CastInSanct grants the original global +3 defense duration effect.
func (g *GameState) CastInSanct() bool {
	return g.ActivateProtectionEffect(inSanctDurationTurns)
}

// CastCombatQuasAnWis enables the original per-action monster confusion check.
func (g *GameState) CastCombatQuasAnWis() bool {
	return g.ActivateConfusionEffect(quasAnWisDurationTurns)
}

// CastCombatInAn blocks combat magic without removing existing fields.
func (g *GameState) CastCombatInAn() bool {
	return g.ActivateNegateMagicEffect(inAnDurationTurns)
}

// CastAnTym suppresses monster-loyal actions unless a present Shadowlord absorbs it.
func (g *GameState) CastAnTym() bool {
	if g.isAnTymAbsorbed() {
		return false
	}

	return g.ActivateTimeStopEffect(anTymDurationTurns)
}

func (g *GameState) isAnTymAbsorbed() bool {
	// REDUX DEVIATION: SPELL-0004: original absorption tests one exact base
	// Shadowlord tile. Redux uses living Shadowlord identity/presence so
	// presentation variants cannot bypass absorption.
	if g.IsShadowlordNPCPresent() {
		return true
	}
	if g.CombatScene != nil {
		for index := range g.CombatScene.MonsterManager.Count() {
			monster := g.CombatScene.MonsterManager.GetMonster(index)
			if monster != nil && monster.IsActive() && !monster.IsDead() &&
				monster.MonsterType() == references.MonsterShadowlord {
				return true
			}
		}
	}

	return false
}
