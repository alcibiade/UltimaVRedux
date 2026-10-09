package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// CastCombatAnYlem removes the original set of removable objects adjacent to
// the caster, leaving the original brick-floor tile behind.
func (g *GameState) CastCombatAnYlem(casterIdx int, direction references.Direction) bool {
	origin := g.combatPartyAttackerPosition(casterIdx)
	target := direction.GetNewPositionInDirection(&origin)
	if target == nil || !isAnYlemObject(g.CombatScene.CombatMapRef.GetTileNumber(*target)) {
		return false
	}
	g.CombatScene.CombatMapRef.SetTileNumber(*target, indexes.BrickFloor)

	return true
}

func isAnYlemObject(tile indexes.SpriteIndex) bool {
	// The original An Ylem switch intentionally excludes other pushable tiles.
	//nolint:exhaustive // Only the original removable object tiles are valid.
	switch tile {
	case indexes.Plant,
		indexes.ChairBackForward, indexes.ChairBackLeft, indexes.ChairBackBack, indexes.ChairBackRight,
		indexes.TableMiddle,
		indexes.Barrel,
		indexes.Vanity,
		indexes.WaterJugTable,
		indexes.Dresser,
		indexes.EndTable,
		indexes.Box,
		indexes.Mirror:
		return true
	default:
		return false
	}
}

// CastCombatWisQuas reveals every active invisible monster in the encounter
// and returns their positions for presentation effects.
func (g *GameState) CastCombatWisQuas() []references.Position {
	revealed := make([]references.Position, 0)
	for monsterIdx := range g.CombatScene.MonsterManager.Count() {
		monster := g.CombatScene.MonsterManager.GetMonster(monsterIdx)
		if monster == nil || !monster.IsActive() || monster.IsDead() || !monster.IsInvisible() {
			continue
		}
		monster.SetInvisible(false)
		revealed = append(revealed, monster.Position())
	}

	return revealed
}

// CastCombatInVasPorYlem applies the original earthquake to hostile combatants
// whose DEX does not exceed their individual D30 roll.
func (g *GameState) CastCombatInVasPorYlem(casterIdx int) []references.Position {
	affected := make([]references.Position, 0)
	// Original prepared-spell rules scan combat queue upward, including possessed PC entries.
	// See ref:SPELL-0002.
	for sourceSlot := range sourceCombatQueueSlots {
		entry := g.CombatScene.TurnQueue.getActiveEntryAtSourceSlot(sourceSlot)
		if entry == nil {
			continue
		}
		target := turnQueueIdentity(entry)
		if g.combatIdentityLoyalty(target) != CombatMonsterLoyaltyMonster ||
			g.effectiveCombatDexterity(target) > g.RollOriginalD30() {
			continue
		}
		damage := g.RollDice(20)
		position := g.CombatScene.GetCombatantPosition(target)
		if target.IsPartyMember {
			character := g.PartyState.GetCharacter(target.EntityIndex)
			hpBefore := character.GetCurrentHp()
			character.SubtractHp(uint16(damage))
			hpLost := int(hpBefore - character.GetCurrentHp())
			g.publishCombatEarthquakeResult(casterIdx, target, position, damage, damage, hpLost, character.IsDead())
			if character.IsDead() {
				g.markCombatPCDead(target.EntityIndex)
			}
			affected = append(affected, position)

			continue
		}
		monsterIdx := target.EntityIndex
		monster := g.CombatScene.MonsterManager.GetMonster(monsterIdx)
		finalDamage := CombatApplyMagicDamageModifiers(damage, monster.MonsterType().IsUndead(), monster.HasAbility(references.Immortal))
		hpBefore := monster.CurrentHP()
		monster.TakeDamage(finalDamage)
		hpLost := hpBefore - monster.CurrentHP()
		g.publishCombatEarthquakeResult(casterIdx, target, position, damage, finalDamage, hpLost, monster.IsDead())
		if hpBefore > 0 && monster.IsDead() {
			g.finalizeCombatMonsterDeath(monster, monsterIdx, g.PartyState.GetCharacter(casterIdx), casterIdx)
		} else if hpLost > 0 {
			g.tryDivideCombatMonster(monster, monsterIdx)
		}
		affected = append(affected, position)
	}

	return affected
}

func (g *GameState) publishCombatEarthquakeResult(casterIdx int, identity CombatTurnIdentity, position references.Position, damage, finalDamage, hpLost int, fatal bool) {
	casterIdentity := g.combatPartyAttackerIdentity(casterIdx)
	attacker := combatFeedbackIdentity(casterIdentity, g.CombatScene.GetCombatantPosition(casterIdentity))
	target := combatFeedbackIdentity(identity, position)
	result := CombatFeedbackDamage
	if hpLost == 0 {
		result = CombatFeedbackGraze
	}
	if hpLost > 0 {
		g.SystemCallbacks.Visual.ShowCombatImpact(CombatAttackFeedback{
			Attacker: attacker, Target: target, Result: result, HPLost: hpLost, Fatal: fatal,
			Presentation: CombatAttackPresentation{
				Delivery: CombatAttackDeliverySpell,
				Source:   references.NoEquipment,
			},
		})
	}
	g.SystemCallbacks.Combat.PublishEvent(CombatEvent{ //nolint:exhaustruct // Earthquake has no physical attack diagnostics.
		Kind: CombatEventSpell, Attacker: attacker, Target: target, Result: result, HPLost: hpLost, Fatal: fatal, From: attacker.Position, To: position,
		Diagnostics: CombatDiagnostics{Spell: references.InVasPorYlem, FinalDamage: finalDamage, MagicDamage: true,
			Damage: CombatDamageResult{Damage: damage, AttackMax: 20, AttackRoll: damage, AttackRolled: true}},
	})
}
