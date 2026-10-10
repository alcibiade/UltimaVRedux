package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// The synchronous fixture adapter completes both production missile phases.
func (g *GameState) monsterAttack(monster *CombatMonster, monsterIndex int, nearest nearestPCResult) bool {
	return g.monsterAttackWithRNG(monster, monsterIndex, nearest, g.AsRandomProvider())
}

// Field unit fixtures explicitly perform both phases; the production turn
// resolver owns their context-specific ordering.
func (g *GameState) resolveCombatPCFieldEffects(identity CombatTurnIdentity) {
	g.resolveCombatFieldExpiration()
	g.resolveCombatPCStandingEffects(identity)
}

func (g *GameState) applyCombatInZuAtPosition(position references.Position, casterIntelligence uint8) bool {
	processed := make(map[CombatTurnIdentity]struct{}, 1)
	target, found := g.CombatScene.combatConeVictimAt(position, processed)

	return found && g.applyCombatInZuToTarget(target, casterIntelligence)
}

// countActivePartyMembers measures fixture occupancy, not effective allegiance.
// Production combat completion uses countActiveCombatLoyalties instead.
func (combatState *CombatState) countActivePartyMembers() int {
	count := 0
	for _, active := range combatState.PartyActive {
		if active {
			count++
		}
	}

	return count
}

// findNearestPC selects a PC for isolated attack/field fixtures, including
// fixtures that have not built a turn queue. This is not the production AI
// selector: findNearestOpponent owns loyalty, source-slot order and visibility
// exceptions, and has separate production-path tests.
func findNearestPC(monsterPos references.Position, combatState *CombatState) nearestPCResult {
	result := nearestPCResult{pcIndex: -1}
	bestDist := -1
	invisibilityProtects := combatState.invisibilityProtectsTargets()

	for i := range party_state.MaxActiveParty {
		identity := CombatTurnIdentity{IsPartyMember: true, EntityIndex: i}
		if !combatState.PartyActive[i] || combatState.isCombatPCUnderground(identity) ||
			(invisibilityProtects && combatState.IsPartyMemberInvisible(i)) {
			continue
		}

		pcPos := combatState.PartyPositions[i]
		dist := integerDistance(monsterPos, pcPos)
		if bestDist == -1 || dist < bestDist {
			bestDist = dist
			result.pcIndex = i
			result.distance = dist
			result.identity = identity
			result.found = true
			result.hasDirection = true
			result.dirX = signum(int(pcPos.X) - int(monsterPos.X))
			result.dirY = signum(int(pcPos.Y) - int(monsterPos.Y))
		}
	}
	for _, clone := range combatState.partyClones {
		identity := CombatTurnIdentity{IsPartyMember: true, EntityIndex: clone.partyIndex, Slot: clone.slot}
		if combatState.isCombatPCUnderground(identity) ||
			(invisibilityProtects && combatState.isCombatPCInvisible(identity)) {
			continue
		}
		dist := integerDistance(monsterPos, clone.position)
		if bestDist == -1 || dist < bestDist {
			bestDist = dist
			result.pcIndex = clone.partyIndex
			result.distance = dist
			result.identity = identity
			result.found = true
			result.hasDirection = true
			result.dirX = signum(int(clone.position.X) - int(monsterPos.X))
			result.dirY = signum(int(clone.position.Y) - int(monsterPos.Y))
		}
	}

	return result
}
